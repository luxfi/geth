// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// One ancient store, one writer, any number of readers.
//
// The freezer's on-disk form is append-only: the payload goes into a .cdat or
// .rdat file first, then a fixed-width entry describing it is appended to the
// .cidx or .ridx index. Bytes that have been written are never rewritten. That
// ordering is the whole reason a live writer and live readers can share one
// directory with nothing arbitrating between them — an index entry that exists
// always names payload bytes that are already on disk, and a reader that
// re-reads the index picks up exactly what the writer has appended since.
//
// The writer holds the directory's FLOCK, as it always has. A shared reader
// takes no lock at all. Asking for the shared half of that lock would be worse
// than useless: an exclusive lock and a shared lock exclude each other, so the
// reader would be refused while the writer runs, which is the only time it has
// anything to read. It opens every file O_RDONLY and issues no write, no
// truncate and no unlink, so having no lock costs nothing.
//
// One machine therefore holds one copy of the finalized chain no matter how
// many nodes run on it, and each node's own database carries only the blocks
// above the freeze threshold.
package rawdb

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/geth/metrics"
	log "github.com/luxfi/log"
)

// sharedRefreshInterval bounds how often a reader goes back to disk to see what
// the writer has added. Blocks arrive seconds apart, so a quarter second of
// staleness is invisible, and the cost of a pass is one stat and two short
// reads per table.
const sharedRefreshInterval = 250 * time.Millisecond

// NewSharedFreezer opens an existing ancient store for reading alongside its
// writer and alongside other readers. Nothing is created: a reader that found
// no store would be describing an empty chain rather than sharing one, so a
// missing or unreadable directory is an error.
func NewSharedFreezer(datadir string, tables map[string]freezerTableConfig) (*Freezer, error) {
	info, err := os.Lstat(datadir)
	if err != nil {
		return nil, fmt.Errorf("shared ancient store %q: %w", datadir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errSymlinkDatadir
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("shared ancient store %q is not a directory", datadir)
	}
	freezer := &Freezer{
		datadir:  datadir,
		readonly: true,
		shared:   true,
		tables:   make(map[string]*freezerTable),
	}
	for name, config := range tables {
		table, err := openSharedTable(datadir, name, config)
		if err != nil {
			freezer.closeTables()
			return nil, fmt.Errorf("shared ancient table %q: %w", name, err)
		}
		freezer.tables[name] = table
	}
	if err := freezer.Refresh(); err != nil {
		freezer.closeTables()
		return nil, err
	}
	log.Info("Opened shared ancient store", "database", datadir,
		"items", freezer.frozen.Load(), "tail", freezer.tail.Load())
	return freezer, nil
}

// Refresh brings the reader up to whatever the writer has appended since the
// last pass. Only a shared reader has anything to refresh; for every other
// freezer the store cannot change behind its back.
func (f *Freezer) Refresh() error {
	if !f.shared {
		return nil
	}
	f.refreshLock.Lock()
	defer f.refreshLock.Unlock()
	return f.refreshLocked()
}

func (f *Freezer) refreshLocked() error {
	for kind, table := range f.tables {
		if err := table.refresh(); err != nil {
			return fmt.Errorf("shared ancient table %q: %w", kind, err)
		}
	}
	// A writer commits its tables one after another, so a reader can catch it
	// with the header of a block written and the body not. The shortest table
	// is the last block every table has; the deepest tail is the first block
	// they all still hold. Reading between those two is reading whole blocks.
	var (
		head = uint64(math.MaxUint64)
		tail uint64
	)
	for _, table := range f.tables {
		head = min(head, table.items.Load())
		tail = max(tail, table.itemHidden.Load())
	}
	if head == math.MaxUint64 {
		head = 0
	}
	f.frozen.Store(head)
	f.tail.Store(tail)
	f.refreshed.Store(time.Now().UnixNano())
	return nil
}

// track refreshes a shared reader that is being asked for something at or past
// the end of what it last saw, which is how a reader notices the writer has
// moved. Rate-limited, because the ordinary reason a read lands past the end is
// that the block is still hot and belongs to the caller's own database.
func (f *Freezer) track() {
	if !f.shared || f.recentlyRefreshed() {
		return
	}
	// A read spanning several tables holds this lock so its view cannot move
	// between them. Waiting for it would be worse than skipping: the caller
	// would get an extent from after the read it is in the middle of. The next
	// read past the end asks again, so nothing is lost by giving up here.
	if !f.refreshLock.TryLock() {
		return
	}
	defer f.refreshLock.Unlock()
	if f.recentlyRefreshed() {
		return
	}
	if err := f.refreshLocked(); err != nil {
		log.Debug("Shared ancient store refresh failed", "database", f.datadir, "err", err)
	}
}

func (f *Freezer) recentlyRefreshed() bool {
	return time.Since(time.Unix(0, f.refreshed.Load())) < sharedRefreshInterval
}

func (f *Freezer) closeTables() {
	for _, table := range f.tables {
		table.Close()
	}
}

// openSharedTable opens one freezer table for shared reading. The table has to
// exist already: bringing one into being is the writer's job, and a reader that
// created an empty one would report a chain with no history in it.
func openSharedTable(path, name string, config freezerTableConfig) (*freezerTable, error) {
	idxName := name + ".cidx"
	if config.noSnappy {
		idxName = name + ".ridx"
	}
	index, err := openFreezerFileForReadOnly(filepath.Join(path, idxName))
	if err != nil {
		return nil, err
	}
	meta, err := openFreezerFileForReadOnly(filepath.Join(path, name+".meta"))
	if err != nil {
		index.Close()
		return nil, err
	}
	metadata, err := loadMetadata(meta)
	if err != nil {
		index.Close()
		meta.Close()
		return nil, err
	}
	tab := &freezerTable{
		index:       index,
		metadata:    metadata,
		lastSync:    time.Now(),
		files:       make(map[uint32]*os.File),
		readMeter:   metrics.NewInactiveMeter(),
		writeMeter:  metrics.NewInactiveMeter(),
		sizeGauge:   metrics.NewGauge(),
		name:        name,
		path:        path,
		logger:      log.New("database", path, "table", name),
		config:      config,
		readonly:    true,
		maxFileSize: freezerTableSize,
	}
	if err := tab.refresh(); err != nil {
		tab.Close()
		return nil, err
	}
	return tab, nil
}

// refresh re-reads the index and metadata so a reader follows a writer that is
// still appending to this table.
//
// The index is the authority on what exists. Payload bytes are written before
// the entry that describes them, so an entry the reader can see always points
// at bytes already on disk, and a torn tail can only make the index look
// shorter than the payload, never longer. Nothing here needs to agree with the
// writer about anything; it only needs to read the index the way the writer
// wrote it.
// indexPath names the index file this table reads, the way the writer names it.
func (t *freezerTable) indexPath() string {
	idxName := fmt.Sprintf("%s.cidx", t.name)
	if t.config.noSnappy {
		idxName = fmt.Sprintf("%s.ridx", t.name)
	}
	return filepath.Join(t.path, idxName)
}

// reopenIndexIfReplaced re-opens the index when the path no longer names the
// file we hold. That is how a tail prune becomes visible to a reader: the
// writer renames a rebuilt index over the old one, giving a new inode.
func (t *freezerTable) reopenIndexIfReplaced() error {
	path := t.indexPath()
	onDisk, err := os.Stat(path)
	if err != nil {
		return err
	}
	held, err := t.index.Stat()
	if err != nil {
		return err
	}
	if os.SameFile(onDisk, held) {
		return nil
	}
	index, err := openFreezerFileForReadOnly(path)
	if err != nil {
		return err
	}
	t.index.Close()
	t.index = index
	return nil
}

func (t *freezerTable) refresh() error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Pruning the tail deletes a data file, and it can land between reading the
	// index and opening what the index names. Re-reading resolves it; three
	// passes is far more than one prune needs.
	var err error
	for range 3 {
		if err = t.refreshLocked(); !os.IsNotExist(err) {
			return err
		}
	}
	return err
}

func (t *freezerTable) refreshLocked() error {
	if t.index == nil {
		return errClosed
	}
	// Tail pruning publishes a new index by rename, so the path can name an
	// inode we are not holding. A descriptor to the replaced file keeps
	// reporting the pre-prune size forever, while the metadata — rewritten in
	// place, so the same inode — keeps advancing the hidden count. The table
	// then reaches hidden > items, where every read is out of bounds and
	// nothing reports it. Follow the path, not the descriptor.
	if err := t.reopenIndexIfReplaced(); err != nil {
		return err
	}
	t.metadata.reload()

	stat, err := t.index.Stat()
	if err != nil {
		return err
	}
	// A trailing partial write is not an entry yet.
	entries := stat.Size() / indexEntrySize
	// Only trust the index as far as the writer has flushed it. Above that line
	// an entry can name payload bytes that a power loss left unwritten — the
	// writer discards exactly those on its next boot, and a reader that came up
	// first would otherwise serve them. The cost is lagging the writer by up to
	// one flush, which does not matter here: everything in this store is older
	// than the freeze threshold, and the reader keeps its own copy of the hot
	// window. It matters that pruning follows this same line, which it does —
	// prune never passes what the reader can see.
	if flushed := t.metadata.flushOffset / indexEntrySize; flushed < entries {
		entries = flushed
	}
	if entries < 1 {
		// The writer has created the index but not yet written the leading
		// sentinel. There is nothing to read here yet.
		t.items.Store(0)
		t.itemOffset.Store(0)
		t.itemHidden.Store(0)
		return nil
	}
	buf := make([]byte, indexEntrySize)

	// The leading entry carries the tail: which file history now starts in, and
	// how many items have been dropped from the front of the table.
	var first indexEntry
	if _, err := t.index.ReadAt(buf, 0); err != nil {
		return err
	}
	first.unmarshalBinary(buf)

	last := indexEntry{filenum: first.filenum}
	if entries > 1 {
		if _, err := t.index.ReadAt(buf, (entries-1)*indexEntrySize); err != nil {
			return err
		}
		last.unmarshalBinary(buf)
	}
	tailId, headId := first.filenum, last.filenum
	itemOffset := uint64(first.offset)

	// Open every data file the index now spans, and let go of the ones tail
	// pruning has moved past. Letting go is all a reader may do with them: the
	// files belong to the writer, and unlinking one would take history away
	// from every other reader on the box.
	for i := tailId; i <= headId; i++ {
		if _, err := t.openFile(i, openFreezerFileForReadOnly); err != nil {
			return err
		}
	}
	for num, f := range t.files {
		if num < tailId || num > headId {
			delete(t.files, num)
			f.Close()
		}
	}
	t.tailId, t.headId = tailId, headId
	t.head = t.files[headId]
	t.headBytes = int64(last.offset)
	t.itemOffset.Store(itemOffset)
	t.itemHidden.Store(max(t.metadata.virtualTail, itemOffset))
	t.items.Store(itemOffset + uint64(entries) - 1)
	return nil
}
