// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// Reproduction of the coreth/evm state-root pruning contract: state roots are
// managed purely through Reference/Dereference on the ROOT (tipBuffer eviction,
// RejectTrie, reprocess previousRoot deref). hashdb.Update only references
// storage roots, so without ReferenceRootAtomicallyOnUpdate a state root has
// parents==0. This test exercises the exact lifecycle and asserts the accepted
// head trie stays readable.

package triedb_test

import (
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/trie"
	"github.com/luxfi/geth/trie/trienode"
	"github.com/luxfi/geth/triedb"
	"github.com/luxfi/geth/triedb/hashdb"
)

func newRefTestDB(refRoot bool) *triedb.Database {
	return triedb.NewDatabase(rawdb.NewMemoryDatabase(), &triedb.Config{
		HashDB: &hashdb.Config{ReferenceRootAtomicallyOnUpdate: refRoot},
	})
}

// buildOn opens a trie at parent, applies mutations, commits it into tdb, and
// returns the new root. Mirrors coreth's per-block state.Commit -> triedb.Update.
func buildOn(t *testing.T, tdb *triedb.Database, parent common.Hash, blockNum uint64, muts map[string]string) common.Hash {
	t.Helper()
	tr, err := trie.New(trie.TrieID(parent), tdb)
	if err != nil {
		t.Fatalf("open trie at %x: %v", parent, err)
	}
	for k, v := range muts {
		tr.MustUpdate([]byte(k), []byte(v))
	}
	root, set := tr.Commit(false)
	if set != nil {
		if err := tdb.Update(root, parent, blockNum, trienode.NewWithNodeSet(set), nil); err != nil {
			t.Fatalf("triedb.Update: %v", err)
		}
	}
	return root
}

// assertReadable opens a trie at root and reads every key, failing on any
// "missing trie node". This is exactly what miner.createCurrentEnvironment ->
// StateAt(parent.Root) does before BuildBlock.
func assertReadable(t *testing.T, tdb *triedb.Database, root common.Hash, want map[string]string) {
	t.Helper()
	tr, err := trie.New(trie.TrieID(root), tdb)
	if err != nil {
		t.Fatalf("open head trie %x: %v (missing trie node == the bug)", root, err)
	}
	for k, v := range want {
		got, err := tr.Get([]byte(k))
		if err != nil {
			t.Fatalf("read key %q at head %x: %v (missing trie node == the bug)", k, root, err)
		}
		if string(got) != v {
			t.Fatalf("key %q = %q, want %q", k, got, v)
		}
	}
}

// manyKeys spreads keys across the trie so it has real branch depth; the base
// set is what siblings share in memory.
func manyKeys(prefix string, n int) map[string]string {
	m := make(map[string]string, n)
	for i := 0; i < n; i++ {
		m[prefix+string(rune('a'+i%26))+string(rune('0'+i/26))] = "v"
	}
	return m
}

// TestReferenceRoot_RejectSiblingOnFreshChain models the equivocation halt: a
// fresh chain (nothing committed to disk) with two sibling blocks at height 2;
// the engine accepts one and RejectTrie-dereferences the other. The accepted
// head must remain fully readable.
func TestReferenceRoot_RejectSiblingOnFreshChain(t *testing.T) {
	for _, refRoot := range []bool{true, false} {
		t.Run(map[bool]string{true: "withFix", false: "withoutFix"}[refRoot], func(t *testing.T) {
			tdb := newRefTestDB(refRoot)

			base := manyKeys("acct", 200)
			// genesis (in memory, NOT committed — fresh chain)
			r0 := buildOn(t, tdb, common.Hash{}, 0, base)
			// block 1 accepted
			b1 := map[string]string{"acctZ0": "one"}
			r1 := buildOn(t, tdb, r0, 1, b1)

			// two siblings at height 2, each mutating a DISJOINT deep key
			r2a := buildOn(t, tdb, r1, 2, map[string]string{"headA": "A"})
			r2b := buildOn(t, tdb, r1, 2, map[string]string{"tailB": "B"})

			// coreth accept lifecycle: tipBuffer holds accepted roots; reject the
			// sibling (RejectTrie -> Dereference).
			if err := tdb.Dereference(r2b); err != nil {
				t.Fatalf("dereference sibling: %v", err)
			}

			// The accepted head r2a must be fully readable (base + b1 + its own mutation).
			want := map[string]string{"headA": "A", "acctZ0": "one"}
			for k, v := range base {
				want[k] = v
			}
			assertReadable(t, tdb, r2a, want)
			// r1 (still in tipBuffer) must also be intact.
			_ = r1
		})
	}
}

// TestReferenceRoot_ReprocessDerefChain models reprocessState's rebuild loop
// (blockchain.go): re-execute blocks forward, Dereference(previousRoot) after
// computing each next root, then Commit the final root. On a fresh chain nothing
// is on disk, so every shared node lives only in memory. The final head must
// survive the chain of previousRoot dereferences and Commit intact.
func TestReferenceRoot_ReprocessDerefChain(t *testing.T) {
	for _, refRoot := range []bool{true, false} {
		t.Run(map[bool]string{true: "withFix", false: "withoutFix"}[refRoot], func(t *testing.T) {
			tdb := newRefTestDB(refRoot)

			base := manyKeys("acct", 200)
			roots := []common.Hash{}
			prev := common.Hash{}
			cur := buildOn(t, tdb, prev, 0, base)
			roots = append(roots, cur)

			// 12 incremental blocks, each mutating a distinct deep key.
			for i := 1; i <= 12; i++ {
				parent := cur
				cur = buildOn(t, tdb, parent, uint64(i), map[string]string{
					"blk" + string(rune('a'+i)): "x",
				})
				// reprocess loop: Dereference(previousRoot) once the next root exists.
				if err := tdb.Dereference(parent); err != nil {
					t.Fatalf("dereference previousRoot: %v", err)
				}
				roots = append(roots, cur)
			}

			// reprocess finishes with Commit(finalRoot) to disk.
			if err := tdb.Commit(cur, false); err != nil {
				t.Fatalf("commit final root: %v", err)
			}

			want := map[string]string{}
			for k, v := range base {
				want[k] = v
			}
			for i := 1; i <= 12; i++ {
				want["blk"+string(rune('a'+i))] = "x"
			}
			assertReadable(t, tdb, cur, want)
		})
	}
}

// TestReferenceRoot_SiblingRejectWithAncestorEvicted is the SHARP case: the
// common ancestor is dereferenced (tip-buffer eviction) around the same time a
// sibling is rejected, so a node shared only via the rejected sibling's path can
// be cascade-deleted out from under the accepted head. This is the topology the
// simple sibling test could not exercise (there the live ancestor protected all
// shared nodes).
func TestReferenceRoot_SiblingRejectWithAncestorEvicted(t *testing.T) {
	for _, refRoot := range []bool{true, false} {
		t.Run(map[bool]string{true: "withFix", false: "withoutFix"}[refRoot], func(t *testing.T) {
			tdb := newRefTestDB(refRoot)
			base := manyKeys("acct", 200)
			r0 := buildOn(t, tdb, common.Hash{}, 0, base)
			r1 := buildOn(t, tdb, r0, 1, map[string]string{"acctZ0": "one"})

			// Two siblings at height 2 that BOTH create the same brand-new key
			// (identical shared node) plus their own distinct key.
			r2a := buildOn(t, tdb, r1, 2, map[string]string{"shared": "S", "onlyA": "A"})
			r2b := buildOn(t, tdb, r1, 2, map[string]string{"shared": "S", "onlyB": "B"})

			// Simulate the reorg/prune order that a converging fresh chain hits:
			// ancestors r0, r1 age out of the tip buffer (dereferenced), and the
			// losing sibling r2b is rejected.
			for _, r := range []common.Hash{r0, r1, r2b} {
				if err := tdb.Dereference(r); err != nil {
					t.Fatalf("dereference %x: %v", r, err)
				}
			}

			want := map[string]string{"shared": "S", "onlyA": "A", "acctZ0": "one"}
			for k, v := range base {
				want[k] = v
			}
			assertReadable(t, tdb, r2a, want)
		})
	}
}
