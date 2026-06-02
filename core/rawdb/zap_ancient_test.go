// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// ZAP-native ancient store tests.
//
// We exercise the store against the in-memory KV backend so tests are
// hermetic and race-clean. The ZAP-backed integration path is the same
// thanks to the zapAncientBackend interface — anything that ethdb/zapdb
// New() satisfies, the store works against.
package rawdb

import (
	"bytes"
	"testing"

	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/memorydb"
)

// memBackend wraps memorydb so ZapAncientStore can take ownership of
// Close. The standard memorydb.Database is a struct, not an interface;
// this small wrapper hands the right methods to the store.
type memBackend struct{ *memorydb.Database }

func (m memBackend) NewBatch() ethdb.Batch { return m.Database.NewBatch() }

func newTestStore(t *testing.T) (*ZapAncientStore, func()) {
	t.Helper()
	tables := map[string]freezerTableConfig{
		"headers":  {noSnappy: false, prunable: false},
		"hashes":   {noSnappy: true, prunable: false},
		"bodies":   {noSnappy: false, prunable: true},
		"receipts": {noSnappy: false, prunable: true},
	}
	store, err := NewZapAncientStore(t.TempDir(), tables, memBackend{memorydb.New()})
	if err != nil {
		t.Fatalf("NewZapAncientStore: %v", err)
	}
	return store, func() { _ = store.Close() }
}

// TestZapAncientStore_AppendRoundtrip writes blocks 0..N and reads them
// back. Asserts the (kind, number) → value mapping survives compression
// and the head pointers advance in lockstep.
func TestZapAncientStore_AppendRoundtrip(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	const n = 16
	type want struct {
		kind  string
		bytes []byte
	}
	wants := make([]want, 0, n*4)
	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < n; i++ {
			headerVal := bytes.Repeat([]byte{byte(i)}, 100)
			hashVal := bytes.Repeat([]byte{byte(0xff - i)}, 32)
			bodyVal := bytes.Repeat([]byte{byte(i + 1)}, 200)
			receiptVal := bytes.Repeat([]byte{byte(i + 2)}, 64)
			if err := op.AppendRaw("headers", i, headerVal); err != nil {
				return err
			}
			if err := op.AppendRaw("hashes", i, hashVal); err != nil {
				return err
			}
			if err := op.AppendRaw("bodies", i, bodyVal); err != nil {
				return err
			}
			if err := op.AppendRaw("receipts", i, receiptVal); err != nil {
				return err
			}
			wants = append(wants,
				want{"headers", headerVal},
				want{"hashes", hashVal},
				want{"bodies", bodyVal},
				want{"receipts", receiptVal},
			)
		}
		return nil
	}); err != nil {
		t.Fatalf("ModifyAncients: %v", err)
	}

	if h, err := store.Ancients(); err != nil || h != n {
		t.Fatalf("Ancients head = %d (err=%v), want %d", h, err, n)
	}

	for i := uint64(0); i < n; i++ {
		for _, kind := range []string{"headers", "hashes", "bodies", "receipts"} {
			got, err := store.Ancient(kind, i)
			if err != nil {
				t.Fatalf("Ancient(%s, %d): %v", kind, i, err)
			}
			// Find expected.
			var w want
			for _, e := range wants {
				if e.kind == kind {
					w = e
					wants = wants[1:]
					break
				}
			}
			if !bytes.Equal(got, w.bytes) {
				t.Errorf("Ancient(%s, %d) length=%d, want %d", kind, i, len(got), len(w.bytes))
			}
		}
	}
}

// TestZapAncientStore_OutOfOrderAppendRejected asserts the writeOp
// refuses to skip a number — the freezer's append-only no-gap rule.
func TestZapAncientStore_OutOfOrderAppendRejected(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		// Number 0 is fine.
		if err := op.AppendRaw("headers", 0, []byte("a")); err != nil {
			return err
		}
		// Number 2 is the gap — must error.
		return op.AppendRaw("headers", 2, []byte("c"))
	}); err == nil {
		t.Fatal("expected out-of-order error, got nil")
	}
}

// TestZapAncientStore_RangeReadsRespectMaxBytes asserts AncientRange
// returns at least one item even if maxBytes is below the first item's
// size — the contract callers depend on for progress.
func TestZapAncientStore_RangeReadsRespectMaxBytes(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	val := bytes.Repeat([]byte{0xab}, 500)
	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 4; i++ {
			if err := op.AppendRaw("bodies", i, val); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("ModifyAncients: %v", err)
	}

	out, err := store.AncientRange("bodies", 0, 10, 100)
	if err != nil {
		t.Fatalf("AncientRange: %v", err)
	}
	// With maxBytes=100 < 500 we must still get exactly one item.
	if len(out) != 1 {
		t.Errorf("AncientRange should return 1 item with maxBytes < itemSize, got %d", len(out))
	}

	out, err = store.AncientRange("bodies", 0, 10, 1100)
	if err != nil {
		t.Fatalf("AncientRange: %v", err)
	}
	// 1100 fits 2 items @ 500 bytes each.
	if len(out) != 2 {
		t.Errorf("AncientRange maxBytes=1100 should return 2 items, got %d", len(out))
	}
}

// TestZapAncientStore_TruncateHead asserts a truncate drops the head
// pointer + records, and the next append number resumes at the new
// head.
func TestZapAncientStore_TruncateHead(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 10; i++ {
			val := []byte{byte(i)}
			for _, kind := range []string{"headers", "hashes", "bodies", "receipts"} {
				if err := op.AppendRaw(kind, i, val); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("ModifyAncients: %v", err)
	}

	prev, err := store.TruncateHead(5)
	if err != nil {
		t.Fatalf("TruncateHead: %v", err)
	}
	if prev != 10 {
		t.Errorf("TruncateHead prev head = %d, want 10", prev)
	}

	if _, err := store.Ancient("headers", 5); err == nil {
		t.Error("Ancient(5) should be gone after TruncateHead(5)")
	}
	if _, err := store.Ancient("headers", 4); err != nil {
		t.Errorf("Ancient(4) should remain: %v", err)
	}

	// Append must resume at 5.
	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for _, kind := range []string{"headers", "hashes", "bodies", "receipts"} {
			if err := op.AppendRaw(kind, 5, []byte{0x55}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("resumed ModifyAncients: %v", err)
	}
	got, err := store.Ancient("headers", 5)
	if err != nil || len(got) != 1 || got[0] != 0x55 {
		t.Errorf("Ancient(5) after resumed append = %v err=%v", got, err)
	}
}

// TestZapAncientStore_TruncateTailSkipsNonPrunable asserts the tail
// truncation honors the per-table prunable flag. Headers + hashes are
// non-prunable; bodies + receipts are prunable.
func TestZapAncientStore_TruncateTailSkipsNonPrunable(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 8; i++ {
			val := []byte{byte(i)}
			for _, kind := range []string{"headers", "hashes", "bodies", "receipts"} {
				if err := op.AppendRaw(kind, i, val); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("ModifyAncients: %v", err)
	}

	if _, err := store.TruncateTail(4); err != nil {
		t.Fatalf("TruncateTail: %v", err)
	}

	// bodies / receipts are prunable — tail records gone.
	if _, err := store.Ancient("bodies", 2); err == nil {
		t.Error("bodies @ 2 should be pruned (prunable=true)")
	}
	// headers / hashes are non-prunable — record stays.
	if _, err := store.Ancient("headers", 2); err != nil {
		t.Errorf("headers @ 2 should remain (prunable=false): %v", err)
	}
}

// TestZapAncientStore_CompressionEnabledForCompressibleTables asserts
// a snappy-encoded record on disk is smaller than the raw payload for
// a compressible value, while a no-compression table stores the raw
// bytes verbatim.
func TestZapAncientStore_CompressionEnabledForCompressibleTables(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	// A highly-compressible body (all zero bytes).
	compressible := make([]byte, 10_000)
	// A non-compressible hash (random-looking content).
	uncompressible := bytes.Repeat([]byte{0x12, 0x34, 0x56, 0x78}, 8)

	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		if err := op.AppendRaw("bodies", 0, compressible); err != nil {
			return err
		}
		return op.AppendRaw("hashes", 0, uncompressible)
	}); err != nil {
		t.Fatalf("ModifyAncients: %v", err)
	}

	// AncientSize on the bodies table must be far less than the raw
	// payload — snappy compresses zero bytes hard.
	sz, err := store.AncientSize("bodies")
	if err != nil {
		t.Fatalf("AncientSize bodies: %v", err)
	}
	if sz >= uint64(len(compressible)) {
		t.Errorf("expected snappy-compressed bodies (got %d bytes) to be < %d", sz, len(compressible))
	}

	// AncientSize on hashes must match raw size — noSnappy=true.
	sz, err = store.AncientSize("hashes")
	if err != nil {
		t.Fatalf("AncientSize hashes: %v", err)
	}
	if sz != uint64(len(uncompressible)) {
		t.Errorf("non-snappy table size: got %d want %d", sz, len(uncompressible))
	}

	// Roundtrip the decompressed value is identical.
	got, err := store.Ancient("bodies", 0)
	if err != nil {
		t.Fatalf("Ancient bodies: %v", err)
	}
	if !bytes.Equal(got, compressible) {
		t.Errorf("compressed body roundtrip mismatch (len=%d, want=%d)", len(got), len(compressible))
	}
}

// TestZapAncientStore_PersistAcrossOpen asserts the on-disk metadata
// (head, tail) survives a Close + re-open against the same backend.
func TestZapAncientStore_PersistAcrossOpen(t *testing.T) {
	tables := map[string]freezerTableConfig{
		"headers": {noSnappy: false, prunable: false},
	}
	mem := memBackend{memorydb.New()}
	store, err := NewZapAncientStore(t.TempDir(), tables, mem)
	if err != nil {
		t.Fatalf("NewZapAncientStore: %v", err)
	}
	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 5; i++ {
			if err := op.AppendRaw("headers", i, []byte{byte(i)}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("ModifyAncients: %v", err)
	}
	// NOTE: Close would shut the backend; reuse the backend via the
	// re-open path instead. Production paths share an existing *DB
	// across processes via the FLOCK file.
	store2, err := NewZapAncientStore(t.TempDir(), tables, mem)
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer store2.Close()
	h, err := store2.Ancients()
	if err != nil {
		t.Fatalf("Ancients: %v", err)
	}
	if h != 5 {
		t.Errorf("re-open head = %d, want 5", h)
	}
}
