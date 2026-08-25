// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rawdb

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/luxfi/geth/ethdb"
)

// sharedTestTables exercises both an uncompressed and a snappy table, since the
// index file name and the read path differ between them.
var sharedTestTables = map[string]freezerTableConfig{
	"raw": {noSnappy: true, prunable: true},
	"zip": {noSnappy: false, prunable: true},
}

// sharedTestItem is the payload for item n. Long enough that a small max table
// size rolls the data file over every few items, which is what puts a reader in
// the position of having to open a file that did not exist when it started.
func sharedTestItem(n uint64) []byte {
	return bytes.Repeat([]byte(fmt.Sprintf("<%d>", n)), 64)
}

// newWriterFreezer opens a writable freezer with a deliberately tiny max table
// size so the tests cross data file boundaries.
func newWriterFreezer(t *testing.T, dir string) *Freezer {
	t.Helper()
	f, err := NewFreezer(dir, "", false, 2049, sharedTestTables)
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}
	return f
}

// appendItems writes items [from, to) to every table in one batch.
func appendItems(t *testing.T, f *Freezer, from, to uint64) {
	t.Helper()
	_, err := f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for n := from; n < to; n++ {
			for kind := range sharedTestTables {
				if err := op.AppendRaw(kind, n, sharedTestItem(n)); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("append [%d,%d): %v", from, to, err)
	}
	if err := f.SyncAncient(); err != nil {
		t.Fatalf("sync: %v", err)
	}
}

// readAll checks that the reader returns the expected payload for every item in
// [from, to) on every table.
func readAll(t *testing.T, r *Freezer, from, to uint64) {
	t.Helper()
	for n := from; n < to; n++ {
		for kind := range sharedTestTables {
			got, err := r.Ancient(kind, n)
			if err != nil {
				t.Fatalf("read %s[%d]: %v", kind, n, err)
			}
			if want := sharedTestItem(n); !bytes.Equal(got, want) {
				t.Fatalf("read %s[%d]: got %q want %q", kind, n, got, want)
			}
		}
	}
}

// TestSharedFreezerReadsWhatTheWriterWrote is the base case: one writer, one
// reader, same directory, at the same time.
func TestSharedFreezerReadsWhatTheWriterWrote(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	defer w.Close()
	appendItems(t, w, 0, 128)

	r, err := NewSharedFreezer(dir, sharedTestTables)
	if err != nil {
		t.Fatalf("open shared reader while the writer holds the store: %v", err)
	}
	defer r.Close()

	if head, _ := r.Ancients(); head != 128 {
		t.Fatalf("reader head is %d, want 128", head)
	}
	readAll(t, r, 0, 128)

	// Past the end is out of bounds, not a stale hit.
	if _, err := r.Ancient("raw", 128); !errors.Is(err, errOutOfBounds) {
		t.Fatalf("read past head: got %v, want out of bounds", err)
	}
}

// TestSharedFreezerFollowsWriter proves the reader picks up what the writer
// appends after the reader opened, including items in data files that did not
// exist at open time.
func TestSharedFreezerFollowsWriter(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	defer w.Close()
	appendItems(t, w, 0, 16)

	r, err := NewSharedFreezer(dir, sharedTestTables)
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer r.Close()
	readAll(t, r, 0, 16)

	// Enough rounds to roll the data file over several times.
	for round := uint64(1); round <= 8; round++ {
		from, to := round*16, round*16+16
		appendItems(t, w, from, to)

		// The reader rate-limits how often it goes back to disk, so give it the
		// interval rather than racing it.
		time.Sleep(sharedRefreshInterval + 50*time.Millisecond)

		readAll(t, r, from, to)
		if head, _ := r.Ancients(); head != to {
			t.Fatalf("round %d: reader head is %d, want %d", round, head, to)
		}
	}
	// Everything written is still readable from the start.
	readAll(t, r, 0, 144)
}

// TestSharedFreezerManyConcurrentReaders runs the shape this is for: one writer
// and many readers over one copy of the data, all live at once.
func TestSharedFreezerManyConcurrentReaders(t *testing.T) {
	const (
		readers = 16
		rounds  = 12
		batch   = 8
	)
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	defer w.Close()
	appendItems(t, w, 0, batch)

	var (
		wg      sync.WaitGroup
		stop    = make(chan struct{})
		failed  atomic.Bool
		reasons = make(chan string, readers)
	)
	for i := 0; i < readers; i++ {
		r, err := NewSharedFreezer(dir, sharedTestTables)
		if err != nil {
			t.Fatalf("reader %d: %v", i, err)
		}
		defer r.Close()

		wg.Add(1)
		go func(id int, r *Freezer) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				head, _ := r.Ancients()
				for n := uint64(0); n < head; n++ {
					for kind := range sharedTestTables {
						got, err := r.Ancient(kind, n)
						if err != nil {
							failed.Store(true)
							reasons <- fmt.Sprintf("reader %d: %s[%d]: %v", id, kind, n, err)
							return
						}
						if want := sharedTestItem(n); !bytes.Equal(got, want) {
							failed.Store(true)
							reasons <- fmt.Sprintf("reader %d: %s[%d]: got %q want %q", id, kind, n, got, want)
							return
						}
					}
				}
			}
		}(i, r)
	}
	for round := uint64(1); round <= rounds && !failed.Load(); round++ {
		appendItems(t, w, round*batch, round*batch+batch)
		time.Sleep(10 * time.Millisecond)
	}
	close(stop)
	wg.Wait()
	close(reasons)
	for reason := range reasons {
		t.Error(reason)
	}
}

// TestSharedFreezerDoesNotExcludeTheWriter is the reason a shared reader takes
// no file lock: a reader that took even the shared half of the lock would be
// refused while the writer runs, and would refuse a writer that came later.
func TestSharedFreezerDoesNotExcludeTheWriter(t *testing.T) {
	dir := t.TempDir()

	// Writer first, then readers.
	w := newWriterFreezer(t, dir)
	appendItems(t, w, 0, 8)
	readers := make([]*Freezer, 0, 4)
	for i := 0; i < 4; i++ {
		r, err := NewSharedFreezer(dir, sharedTestTables)
		if err != nil {
			t.Fatalf("reader %d could not open alongside the writer: %v", i, err)
		}
		readers = append(readers, r)
	}
	// The writer still works with readers attached.
	appendItems(t, w, 8, 16)
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	// A new writer can take the store while the readers hold it open.
	w2, err := NewFreezer(dir, "", false, 2049, sharedTestTables)
	if err != nil {
		t.Fatalf("writer could not open alongside %d readers: %v", len(readers), err)
	}
	appendItems(t, w2, 16, 24)
	if err := w2.Close(); err != nil {
		t.Fatalf("close second writer: %v", err)
	}
	for i, r := range readers {
		if err := r.Refresh(); err != nil {
			t.Fatalf("reader %d refresh: %v", i, err)
		}
		readAll(t, r, 0, 24)
		if err := r.Close(); err != nil {
			t.Fatalf("close reader %d: %v", i, err)
		}
	}
}

// TestSharedFreezerRefusesMutation checks a reader cannot write through the
// ancient store interface no matter what it is asked to do.
func TestSharedFreezerRefusesMutation(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	appendItems(t, w, 0, 32)
	w.Close()

	r, err := NewSharedFreezer(dir, sharedTestTables)
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer r.Close()

	if _, err := r.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		return op.AppendRaw("raw", 32, sharedTestItem(32))
	}); !errors.Is(err, errReadOnly) {
		t.Fatalf("ModifyAncients: got %v, want read only", err)
	}
	if _, err := r.TruncateHead(4); !errors.Is(err, errReadOnly) {
		t.Fatalf("TruncateHead: got %v, want read only", err)
	}
	if _, err := r.TruncateTail(4); !errors.Is(err, errReadOnly) {
		t.Fatalf("TruncateTail: got %v, want read only", err)
	}
	if head, _ := r.Ancients(); head != 32 {
		t.Fatalf("head moved to %d after refused writes, want 32", head)
	}
	readAll(t, r, 0, 32)
}

// dirFingerprint records the name, size and modification time of every file in
// the store, so a test can prove a reader left all of them alone.
func dirFingerprint(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", e.Name(), err)
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		out[e.Name()] = fmt.Sprintf("size=%d mtime=%d body=%x", info.Size(), info.ModTime().UnixNano(), body)
	}
	return out
}

// TestSharedFreezerLeavesTheStoreUntouched is the safety property N readers
// depend on: opening and reading the shared store changes nothing on disk, so
// no reader can damage what every other reader is using.
func TestSharedFreezerLeavesTheStoreUntouched(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	appendItems(t, w, 0, 64)
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	before := dirFingerprint(t, dir)

	for i := 0; i < 4; i++ {
		r, err := NewSharedFreezer(dir, sharedTestTables)
		if err != nil {
			t.Fatalf("reader %d: %v", i, err)
		}
		readAll(t, r, 0, 64)
		if _, err := r.AncientRange("raw", 0, 64, 0); err != nil {
			t.Fatalf("reader %d range: %v", i, err)
		}
		if err := r.SyncAncient(); err != nil {
			t.Fatalf("reader %d sync: %v", i, err)
		}
		if _, err := r.Ancient("raw", 999); err == nil {
			t.Fatal("read past the head should fail")
		}
		if err := r.Refresh(); err != nil {
			t.Fatalf("reader %d refresh: %v", i, err)
		}
		if err := r.Close(); err != nil {
			t.Fatalf("close reader %d: %v", i, err)
		}
	}
	after := dirFingerprint(t, dir)
	if len(before) != len(after) {
		t.Fatalf("reader changed the file set: %d files before, %d after", len(before), len(after))
	}
	for name, want := range before {
		got, ok := after[name]
		if !ok {
			t.Fatalf("reader removed %s", name)
		}
		if got != want {
			t.Fatalf("reader changed %s:\n before %s\n after  %s", name, want, got)
		}
	}
	// A reader must not have left a lock file behind either.
	if _, err := os.Stat(filepath.Join(dir, "FLOCK")); err == nil {
		if _, ok := before["FLOCK"]; !ok {
			t.Fatal("reader created a FLOCK file")
		}
	}
}

// TestSharedFreezerFollowsTailPruning checks a reader tracks the writer taking
// history away from the front of the store, and stops serving what is gone.
func TestSharedFreezerFollowsTailPruning(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	defer w.Close()
	appendItems(t, w, 0, 128)

	r, err := NewSharedFreezer(dir, sharedTestTables)
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer r.Close()
	readAll(t, r, 0, 128)

	if _, err := w.TruncateTail(64); err != nil {
		t.Fatalf("truncate tail: %v", err)
	}
	if err := r.Refresh(); err != nil {
		t.Fatalf("refresh after tail prune: %v", err)
	}
	if tail, _ := r.Tail(); tail != 64 {
		t.Fatalf("reader tail is %d, want 64", tail)
	}
	if _, err := r.Ancient("raw", 0); !errors.Is(err, errOutOfBounds) {
		t.Fatalf("read below the tail: got %v, want out of bounds", err)
	}
	readAll(t, r, 64, 128)

	// Reading pruned-away history proves nothing on its own: a reader holding a
	// replaced index still answers those from the stale index and the unlinked
	// data files it keeps open. The prune is only really followed if the reader
	// also sees what the writer appends AFTER it, so assert that.
	appendItems(t, w, 128, 160)
	if err := r.Refresh(); err != nil {
		t.Fatalf("refresh after append following a tail prune: %v", err)
	}
	if n, _ := r.Ancients(); n != 160 {
		t.Fatalf("reader stopped following the writer after a tail prune: sees %d items, want 160", n)
	}
	readAll(t, r, 128, 160)
}

// TestSharedFreezerNeedsAnExistingStore checks a reader never conjures an empty
// store: pointed at nothing, it says so instead of reporting a chain with no
// history in it.
func TestSharedFreezerNeedsAnExistingStore(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there")
	if _, err := NewSharedFreezer(missing, sharedTestTables); err == nil {
		t.Fatal("opening a store that does not exist should fail")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("a failed open created the directory")
	}

	// An existing but empty directory has no tables in it and is equally not a
	// store.
	empty := t.TempDir()
	if _, err := NewSharedFreezer(empty, sharedTestTables); err == nil {
		t.Fatal("opening an empty directory as a store should fail")
	}
	entries, err := os.ReadDir(empty)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("a failed open wrote %d files into the directory", len(entries))
	}
}

// TestSharedFreezerStopsAtTheFlushedLine checks a reader never serves index
// entries the writer has not flushed. Above that line an entry can name payload
// bytes a power loss left unwritten, and the writer discards exactly those when
// it next boots — a reader that came up first must not have served them.
func TestSharedFreezerStopsAtTheFlushedLine(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	defer w.Close()
	appendItems(t, w, 0, 64)
	if err := w.SyncAncient(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	r, err := NewSharedFreezer(dir, sharedTestTables)
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer r.Close()
	flushed, err := r.Ancients()
	if err != nil {
		t.Fatalf("ancients: %v", err)
	}
	if flushed != 64 {
		t.Fatalf("reader sees %d flushed items, want 64", flushed)
	}

	// Written but deliberately not synced: the index may be on disk while the
	// payload is not, which is the state a power loss leaves behind. Bypasses
	// appendItems because that helper syncs.
	if _, err := w.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for n := uint64(64); n < 96; n++ {
			for kind := range sharedTestTables {
				if err := op.AppendRaw(kind, n, sharedTestItem(n)); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("append without sync: %v", err)
	}
	if err := r.Refresh(); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if n, _ := r.Ancients(); n > 64 {
		t.Fatalf("reader exposed %d items past the flushed line of 64", n)
	}

	// Once the writer flushes, the reader follows.
	if err := w.SyncAncient(); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := r.Refresh(); err != nil {
		t.Fatalf("refresh after sync: %v", err)
	}
	if n, _ := r.Ancients(); n != 96 {
		t.Fatalf("reader sees %d items after the writer flushed, want 96", n)
	}
	readAll(t, r, 64, 96)
}

// TestSharedReaderHoldsItsViewStillDuringARead checks that a read spanning
// several tables sees one extent rather than two.
//
// Callers reach for ReadAncients to read a header, a body and a receipt as one
// block. The writer is another process, so the lock that name implies cannot
// hold it still — but a refresh of this reader's own view landing between two of
// those reads would answer the second from a later extent than the first, and
// that much is ours to prevent.
func TestSharedReaderHoldsItsViewStillDuringARead(t *testing.T) {
	dir := t.TempDir()
	w := newWriterFreezer(t, dir)
	defer w.Close()
	appendItems(t, w, 0, 8)

	r, err := NewSharedFreezer(dir, sharedTestTables)
	if err != nil {
		t.Fatalf("open shared reader: %v", err)
	}
	defer r.Close()

	// The writer moves on while the reader is mid-read, which is the ordinary
	// case: this is a live store, not a snapshot.
	appendItems(t, w, 8, 64)

	var first, second uint64
	err = r.ReadAncients(func(op ethdb.AncientReaderOp) error {
		var err error
		if first, err = op.Ancients(); err != nil {
			return err
		}
		// Make a refresh due. Without one held off, the next call goes back to
		// disk and answers from the writer's newer extent.
		r.refreshed.Store(0)
		second, err = op.Ancients()
		return err
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if first != second {
		t.Fatalf("one read saw two extents: %d then %d; a caller reading a header and a body here would pair them across a move", first, second)
	}

	// The view is held still for the callback, not forever: the next read sees
	// what the writer has since added.
	if got, err := r.Ancients(); err != nil {
		t.Fatalf("ancients after the read: %v", err)
	} else if got == first {
		t.Fatalf("the reader is stuck at %d and never picked the writer up again", got)
	}
}
