// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rawdb

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/memorydb"
)

// buildChain writes blocks [0, count) into db with a parent link, canonical
// hashes, receipts and a head pointer, which is everything the freezer needs to
// move a block into the ancient store.
func buildChain(t *testing.T, db ethdb.KeyValueStore, count uint64) []*types.Block {
	t.Helper()
	return buildChainSeeded(t, db, count, 0)
}

// buildChainSeeded builds a chain whose blocks differ from another seed's at
// every height, for testing what a node does when a store holds a chain that is
// not its own.
func buildChainSeeded(t *testing.T, db ethdb.KeyValueStore, count uint64, seed byte) []*types.Block {
	t.Helper()
	var (
		blocks []*types.Block
		parent common.Hash
	)
	for n := uint64(0); n < count; n++ {
		extra := []byte{byte(n), byte(n >> 8)}
		// Genesis is shared: a store whose genesis differs is already refused at
		// open, so divergence only means anything from block 1 on.
		if n > 0 {
			extra = append(extra, seed)
		}
		header := &types.Header{
			Number:     big.NewInt(int64(n)),
			ParentHash: parent,
			Extra:      extra,
		}
		block := types.NewBlock(header, nil, nil, newTestHasher())
		WriteBlock(db, block)
		WriteCanonicalHash(db, block.Hash(), n)
		WriteReceipts(db, block.Hash(), n, nil)
		parent = block.Hash()
		blocks = append(blocks, block)
	}
	WriteHeadBlockHash(db, parent)
	WriteHeadHeaderHash(db, parent)
	return blocks
}

// keyCount counts what a node is actually storing.
func keyCount(t *testing.T, db ethdb.KeyValueStore) int {
	t.Helper()
	it := db.NewIterator(nil, nil)
	defer it.Release()
	n := 0
	for it.Next() {
		n++
	}
	return n
}

// TestSharedAncientStoreLeavesReadersWithOnlyTheHotWindow is the whole point of
// the shared store, end to end: one node freezes history into a directory, and
// other nodes reading that directory drop their own copies of the same blocks
// while still serving them.
func TestSharedAncientStoreLeavesReadersWithOnlyTheHotWindow(t *testing.T) {
	const (
		blocks    = 96
		threshold = 8
	)
	ancient := t.TempDir()

	// The node that owns the store: it freezes and then drops what it froze.
	writerKV := memorydb.New()
	blocksWritten := buildChain(t, writerKV, blocks)
	writer, err := Open(writerKV, OpenOptions{Ancient: ancient, FreezeThreshold: threshold})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	if err := writer.(*freezerdb).Freeze(); err != nil {
		t.Fatalf("freeze: %v", err)
	}
	frozen, err := writer.Ancients()
	if err != nil {
		t.Fatalf("ancients: %v", err)
	}
	// max(finality, head-threshold) with no finality reported is head-threshold,
	// and the limit is inclusive, so the store holds one more than that.
	if want := uint64(blocks - 1 - threshold + 1); frozen != want {
		t.Fatalf("store holds %d blocks, want %d", frozen, want)
	}

	// A second node with a full copy of the same chain, reading that store.
	readerKV := memorydb.New()
	buildChain(t, readerKV, blocks)
	before := keyCount(t, readerKV)

	reader, err := Open(readerKV, OpenOptions{
		Ancient:         ancient,
		AncientShared:   true,
		FreezeThreshold: threshold,
	})
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer reader.Close()

	// Prune until it settles: each pass is capped at a batch.
	for i := 0; i < 4; i++ {
		if err := reader.(*freezerdb).Freeze(); err != nil {
			t.Fatalf("prune pass %d: %v", i, err)
		}
	}
	after := keyCount(t, readerKV)
	if after >= before {
		t.Fatalf("reader kept %d keys, was %d: nothing was shared", after, before)
	}
	t.Logf("reader database went from %d keys to %d holding %d blocks", before, after, uint64(blocks))

	// The pruned blocks are gone from the reader's own store...
	nfdb := &nofreezedb{KeyValueStore: readerKV}
	for n := uint64(1); n < frozen; n++ {
		if hash := ReadCanonicalHash(nfdb, n); hash != (common.Hash{}) {
			t.Fatalf("block %d is still in the reader's own database", n)
		}
	}
	// ...and still readable, because the shared store has them.
	for n := uint64(0); n < frozen; n++ {
		want := blocksWritten[n]
		if hash := ReadCanonicalHash(reader, n); hash != want.Hash() {
			t.Fatalf("block %d: canonical hash %x, want %x", n, hash, want.Hash())
		}
		header := ReadHeader(reader, want.Hash(), n)
		if header == nil {
			t.Fatalf("block %d: header not readable through the shared store", n)
		}
		if !bytes.Equal(header.Extra, want.Header().Extra) {
			t.Fatalf("block %d: header extra %x, want %x", n, header.Extra, want.Header().Extra)
		}
		if body := ReadBody(reader, want.Hash(), n); body == nil {
			t.Fatalf("block %d: body not readable through the shared store", n)
		}
	}
	// The hot window stays in the reader's own database, which is what it needs
	// to keep building on the chain.
	for n := frozen; n < blocks; n++ {
		if hash := ReadCanonicalHash(nfdb, n); hash != blocksWritten[n].Hash() {
			t.Fatalf("block %d fell out of the hot window", n)
		}
	}
}

// TestFreezeThresholdSetsTheHotWindow checks the retention knob does what it
// says: it is the number of recent blocks that stay out of the ancient store.
func TestFreezeThresholdSetsTheHotWindow(t *testing.T) {
	for _, threshold := range []uint64{4, 16, 64} {
		kv := memorydb.New()
		buildChain(t, kv, 128)
		db, err := Open(kv, OpenOptions{Ancient: t.TempDir(), FreezeThreshold: threshold})
		if err != nil {
			t.Fatalf("threshold %d: open: %v", threshold, err)
		}
		if err := db.(*freezerdb).Freeze(); err != nil {
			t.Fatalf("threshold %d: freeze: %v", threshold, err)
		}
		frozen, _ := db.Ancients()
		if want := uint64(128) - threshold; frozen != want {
			t.Fatalf("threshold %d: froze %d blocks, want %d", threshold, frozen, want)
		}
		hot := uint64(128) - frozen
		if hot != threshold {
			t.Fatalf("threshold %d: hot window is %d blocks", threshold, hot)
		}
		db.Close()
	}
}

// TestSharedReaderKeepsWhatTheStoreNoLongerServes checks the floor on pruning:
// below the store's tail the writer has thrown the bodies away, so a reader's
// copy is the only one left and dropping it would lose the block outright.
func TestSharedReaderKeepsWhatTheStoreNoLongerServes(t *testing.T) {
	const (
		blocks    = 96
		threshold = 8
		storeTail = 40
	)
	ancient := t.TempDir()
	writerKV := memorydb.New()
	blocksWritten := buildChain(t, writerKV, blocks)
	writer, err := Open(writerKV, OpenOptions{Ancient: ancient, FreezeThreshold: threshold})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	if err := writer.(*freezerdb).Freeze(); err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if _, err := writer.TruncateTail(storeTail); err != nil {
		t.Fatalf("truncate the store's tail: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	readerKV := memorydb.New()
	buildChain(t, readerKV, blocks)
	reader, err := Open(readerKV, OpenOptions{Ancient: ancient, AncientShared: true, FreezeThreshold: threshold})
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer reader.Close()
	for i := 0; i < 4; i++ {
		if err := reader.(*freezerdb).Freeze(); err != nil {
			t.Fatalf("prune pass %d: %v", i, err)
		}
	}
	nfdb := &nofreezedb{KeyValueStore: readerKV}
	// Everything below the store's tail is still this node's own.
	for n := uint64(0); n < storeTail; n++ {
		if hash := ReadCanonicalHash(nfdb, n); hash != blocksWritten[n].Hash() {
			t.Fatalf("block %d was dropped even though the store no longer has its body", n)
		}
		if body := ReadBody(nfdb, blocksWritten[n].Hash(), n); body == nil {
			t.Fatalf("block %d lost its body", n)
		}
	}
	// Above it, the store carries them and this node does not.
	if hash := ReadCanonicalHash(nfdb, storeTail+1); hash != (common.Hash{}) {
		t.Fatalf("block %d is still duplicated in the reader's database", storeTail+1)
	}
	if body := ReadBody(reader, blocksWritten[storeTail+1].Hash(), storeTail+1); body == nil {
		t.Fatalf("block %d is not readable through the shared store", storeTail+1)
	}
}

// TestSharedReaderNeverWritesToTheStore checks a reader node cannot damage the
// store the rest of the box depends on, whatever its own database says.
func TestSharedReaderNeverWritesToTheStore(t *testing.T) {
	ancient := t.TempDir()
	writerKV := memorydb.New()
	buildChain(t, writerKV, 64)
	writer, err := Open(writerKV, OpenOptions{Ancient: ancient, FreezeThreshold: 8})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	if err := writer.(*freezerdb).Freeze(); err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	store := resolveChainFreezerDir(ancient)
	before := dirFingerprint(t, store)

	// A reader whose own chain is longer than the store's: it still has nothing
	// to add, because adding is not something a reader does.
	readerKV := memorydb.New()
	buildChain(t, readerKV, 128)
	reader, err := Open(readerKV, OpenOptions{Ancient: ancient, AncientShared: true, FreezeThreshold: 8})
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := reader.(*freezerdb).Freeze(); err != nil {
			t.Fatalf("prune pass %d: %v", i, err)
		}
	}
	if _, err := reader.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		return op.AppendRaw(ChainFreezerHashTable, 56, common.Hash{}.Bytes())
	}); err == nil {
		t.Fatal("a reader was allowed to write to the shared store")
	}
	if _, err := reader.TruncateHead(4); err == nil {
		t.Fatal("a reader was allowed to truncate the shared store")
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}
	after := dirFingerprint(t, store)
	if len(before) != len(after) {
		t.Fatalf("the store has %d files, had %d", len(after), len(before))
	}
	for name, want := range before {
		if got := after[name]; got != want {
			t.Fatalf("a reader changed %s in the shared store", name)
		}
	}
}

// TestSharedReaderKeepsHistoryTheStoreDisagreesWith checks that pruning is
// bounded by identity and not merely by extent. The store is written by another
// process, so a store holding a different block at a height must not cost this
// node the copy it validated itself — that copy is the only thing left that
// could contradict the store.
func TestSharedReaderKeepsHistoryTheStoreDisagreesWith(t *testing.T) {
	const (
		blocks    = 96
		threshold = 8
	)
	ancient := t.TempDir()

	writerKV := memorydb.New()
	buildChain(t, writerKV, blocks)
	writer, err := Open(writerKV, OpenOptions{Ancient: ancient, FreezeThreshold: threshold})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	if err := writer.(*freezerdb).Freeze(); err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	// A node whose own chain disagrees with the store: same heights, different
	// blocks. Nothing here is malicious — a divergent store looks the same.
	readerKV := memorydb.New()
	buildChainSeeded(t, readerKV, blocks, 0xAA)
	before := keyCount(t, readerKV)

	reader, err := Open(readerKV, OpenOptions{
		Ancient:         ancient,
		AncientShared:   true,
		FreezeThreshold: threshold,
	})
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer reader.Close()

	for i := 0; i < 4; i++ {
		if err := reader.(*freezerdb).Freeze(); err != nil {
			t.Fatalf("prune pass %d: %v", i, err)
		}
	}
	if after := keyCount(t, readerKV); after != before {
		t.Fatalf("reader dropped history the store disagrees with: %d keys, was %d", after, before)
	}
	nfdb := &nofreezedb{KeyValueStore: readerKV}
	if hash := ReadCanonicalHash(nfdb, 1); hash == (common.Hash{}) {
		t.Fatal("reader deleted block 1 in favour of a store that holds a different chain")
	}
}

// TestFreshReaderSurfacesTheStoresGenesis covers the one case the cross-check in
// Open cannot: a node with an empty key-value store has no genesis of its own,
// so there is nothing there to compare the store against.
//
// That is not the hole it looks like. The node still has to learn a genesis from
// somewhere, and what it reads is the store's — core.LoadChainConfig compares
// that against the genesis the operator configured and returns
// GenesisMismatchError when they differ, which is the check upstream deliberately
// left to that layer. What matters here is that the store's genesis is what
// surfaces, because a value that never surfaced could never be refused.
func TestFreshReaderSurfacesTheStoresGenesis(t *testing.T) {
	const (
		blocks    = 96
		threshold = 8
	)
	ancient := t.TempDir()

	writerKV := memorydb.New()
	stored := buildChain(t, writerKV, blocks)
	writer, err := Open(writerKV, OpenOptions{Ancient: ancient, FreezeThreshold: threshold})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	if err := writer.(*freezerdb).Freeze(); err != nil {
		t.Fatalf("freeze: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	// A node created from scratch against a store it did not build. The control
	// first: with the store taken away this database answers with nothing, so a
	// genesis read through it afterwards can only have come from the store.
	freshKV := memorydb.New()
	if got := ReadCanonicalHash(&nofreezedb{KeyValueStore: freshKV}, 0); got != (common.Hash{}) {
		t.Fatalf("a fresh database already holds genesis %x; this test would pass without reading the store at all", got)
	}
	reader, err := Open(freshKV, OpenOptions{
		Ancient:         ancient,
		AncientShared:   true,
		FreezeThreshold: threshold,
	})
	if err != nil {
		t.Fatalf("a fresh node could not open the store: %v", err)
	}
	defer reader.Close()

	if got := ReadCanonicalHash(reader, 0); got != stored[0].Hash() {
		t.Fatalf("a fresh node reads genesis %x, want the store's %x; nothing upstream can refuse a genesis it never sees", got, stored[0].Hash())
	}
	// The rest of the store is readable too, which is what makes the node usable
	// at all once its configured genesis has been accepted.
	if got := ReadCanonicalHash(reader, blocks-1-threshold); got != stored[blocks-1-threshold].Hash() {
		t.Fatalf("a fresh node cannot read the store's history at %d", blocks-1-threshold)
	}
}
