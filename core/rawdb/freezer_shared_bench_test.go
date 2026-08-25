// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rawdb

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	log "github.com/luxfi/log"

	"github.com/luxfi/geth/ethdb"
)

// dirBytes is what the store actually costs on disk.
func dirBytes(t testing.TB, dir string) int64 {
	t.Helper()
	var total int64
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// quiet stops the freezer's own logging from landing in the middle of a
// benchmark result line, which is the only thing separating a number from noise.
func quiet(t testing.TB) {
	prior := log.Root()
	log.SetDefault(log.NewWriter(io.Discard).Level(log.Disabled))
	t.Cleanup(func() { log.SetDefault(prior) })
}

// nextFD reports the number the kernel would hand out for the next descriptor.
// Descriptors are allocated lowest-free-first, so the difference across a block
// of work is how many that work is holding — which is what decides how many
// readers fit on one box. Counting /dev/fd is unreliable here; this is not.
func nextFD(t testing.TB) int {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("probe descriptor: %v", err)
	}
	defer f.Close()
	return int(f.Fd())
}

// readerCost is what one more node costs on a box that already holds the blocks.
type readerCost struct {
	storeBytes int64
	heapPer    float64 // bytes
	fdsPer     float64
}

// costOfReaders fills a store, opens `readers` of them against it, and measures
// what they cost. Every reader is made to serve real bytes first — an unread
// reader has not opened the data files, so measuring one measures nothing.
func costOfReaders(t *testing.T, items uint64, maxFileSize uint32, readers int) readerCost {
	t.Helper()
	dir := t.TempDir()
	w, err := NewFreezer(dir, "", false, maxFileSize, sharedTestTables)
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	if _, err := w.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for n := uint64(0); n < items; n++ {
			for kind := range sharedTestTables {
				if err := op.AppendRaw(kind, n, sharedTestItem(n)); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := w.SyncAncient(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	defer w.Close()
	storeBytes := dirBytes(t, dir)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	fdBefore := nextFD(t)

	open := make([]*Freezer, 0, readers)
	for i := 0; i < readers; i++ {
		r, err := NewSharedFreezer(dir, sharedTestTables)
		if err != nil {
			t.Fatalf("reader %d of %d failed to open: %v", i, readers, err)
		}
		open = append(open, r)
	}
	for i, r := range open {
		if got, err := r.Ancient("raw", items-1); err != nil {
			t.Fatalf("reader %d cannot read: %v", i, err)
		} else if want := sharedTestItem(items - 1); string(got) != string(want) {
			t.Fatalf("reader %d served wrong bytes", i)
		}
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	fdAfter := nextFD(t)

	for _, r := range open {
		r.Close()
	}
	return readerCost{
		storeBytes: storeBytes,
		heapPer:    float64(int64(after.HeapAlloc)-int64(before.HeapAlloc)) / float64(readers),
		fdsPer:     float64(fdAfter-fdBefore) / float64(readers),
	}
}

// TestSharedStoreCostOfManyReaders measures what the design is for: what one
// more node costs once the blocks are already on disk once.
//
// The number that decides whether ~100 nodes fit on one box is not any single
// measurement below — it is whether the per-reader cost stays flat as the store
// grows. A reader holds the index, the metadata and the data files the index
// spans; the index is read at fixed offsets, not slurped, and the file count is
// set by the table size rather than the chain length. So a longer chain should
// cost a reader nothing. That is asserted, not just printed.
func TestSharedStoreCostOfManyReaders(t *testing.T) {
	quiet(t)
	if testing.Short() {
		t.Skip("measures resource cost; skipped in short mode")
	}
	const readers = 100
	// Production rolls a new data file every freezerTableSize bytes, so the
	// small-file case exists only to show that file count, not chain length, is
	// what a reader pays for.
	for _, c := range []struct {
		name        string
		items       uint64
		maxFileSize uint32
	}{
		{"20k items, 2MiB files", 20_000, 2 * 1024 * 1024},
		{"20k items, production file size", 20_000, freezerTableSize},
		{"200k items, production file size", 200_000, freezerTableSize},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := costOfReaders(t, c.items, c.maxFileSize, readers)
			t.Logf("store on disk:            %7.1f MiB for %d items", float64(got.storeBytes)/(1<<20), c.items)
			t.Logf("heap per reader:          %7.0f KiB", got.heapPer/1024)
			t.Logf("fds per reader:           %7.1f", got.fdsPer)
			t.Logf("disk, %d private copies: %7.1f MiB", readers, float64(got.storeBytes)*readers/(1<<20))
			t.Logf("disk, one shared store:   %7.1f MiB  (%dx less)", float64(got.storeBytes)/(1<<20), readers)

			// A per-reader cost that grew with the store would put a ceiling on
			// how many nodes fit, which is the whole claim. These bounds are
			// loose on purpose: they catch a reader that started holding the
			// chain, not a few KiB of drift.
			if got.heapPer > 256*1024 {
				t.Errorf("heap per reader is %.0f KiB; a reader is holding chain-sized state", got.heapPer/1024)
			}
			if got.fdsPer > 32 {
				t.Errorf("fds per reader is %.1f; %d readers would need %.0f descriptors", got.fdsPer, readers, got.fdsPer*readers)
			}
		})
	}
}

// benchItems is small enough to stay in page cache, so the benchmarks measure
// the code path rather than the disk underneath it.
const benchItems = 5000

// storeOf writes a filled ancient store and returns its directory.
func storeOf(b *testing.B) string {
	b.Helper()
	dir := b.TempDir()
	w, err := NewFreezer(dir, "", false, 2*1024*1024, sharedTestTables)
	if err != nil {
		b.Fatalf("open writer: %v", err)
	}
	if _, err := w.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for n := uint64(0); n < benchItems; n++ {
			for kind := range sharedTestTables {
				if err := op.AppendRaw(kind, n, sharedTestItem(n)); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		b.Fatalf("append: %v", err)
	}
	if err := w.SyncAncient(); err != nil {
		b.Fatalf("sync: %v", err)
	}
	w.Close()
	return dir
}

func readLoop(b *testing.B, f *Freezer) {
	b.Helper()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := f.Ancient("raw", uint64(i%benchItems)); err != nil {
			b.Fatalf("read: %v", err)
		}
	}
}

// BenchmarkSharedReaderRead measures the per-read cost through a shared reader,
// which is what a node pays for every historical block it no longer stores.
// Read next to BenchmarkPrivateReaderRead: alone it is a number with nothing to
// compare it against.
func BenchmarkSharedReaderRead(b *testing.B) {
	quiet(b)
	r, err := NewSharedFreezer(storeOf(b), sharedTestTables)
	if err != nil {
		b.Fatalf("open shared reader: %v", err)
	}
	defer r.Close()
	readLoop(b, r)
}

// BenchmarkPrivateReaderRead is the control: the same read against a store the
// node owns outright. The difference between the two is the price of sharing.
func BenchmarkPrivateReaderRead(b *testing.B) {
	quiet(b)
	r, err := NewFreezer(storeOf(b), "", true, 2*1024*1024, sharedTestTables)
	if err != nil {
		b.Fatalf("open private reader: %v", err)
	}
	defer r.Close()
	readLoop(b, r)
}
