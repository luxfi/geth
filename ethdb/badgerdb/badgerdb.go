package badgerdb

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
)

// BadgerDatabase wraps a BadgerDB instance for Ethereum state storage
type BadgerDatabase struct {
	db     *badger.DB
	path   string
	logger log.Logger
	quitCh chan struct{}
	wg     sync.WaitGroup
}

// NewBadgerDatabase creates a new BadgerDB instance
func NewBadgerDatabase(path string, readOnly bool, bypassLock bool) (*BadgerDatabase, error) {
	opts := badger.DefaultOptions(path)
	
	// Configure for read-only access if needed
	if readOnly {
		opts.ReadOnly = true
		opts.BypassLockGuard = bypassLock // Allow multiple RO instances
		opts.SyncWrites = false
		opts.DetectConflicts = false
		
		// Optimize for read-only access
		opts.MemTableSize = 64 << 20 // 64MB
		opts.ValueLogMaxEntries = 1000000
		
		log.Info("Opening BadgerDB in read-only mode", 
			"path", path, 
			"bypass_lock", bypassLock)
	} else {
		// Optimize for read-write access
		opts.SyncWrites = true
		opts.DetectConflicts = true
		opts.NumVersionsToKeep = 1
		opts.NumGoroutines = 8
		opts.NumCompactors = 4
		
		// Enable compression
		opts.Compression = options.Snappy
		
		log.Info("Opening BadgerDB in read-write mode", "path", path)
	}
	
	// Common optimizations
	opts.Logger = nil // Use ethereum logger instead
	opts.ValueLogFileSize = 1 << 28 // 256MB
	opts.ValueThreshold = 32
	opts.BlockCacheSize = 256 << 20 // 256MB
	opts.IndexCacheSize = 256 << 20 // 256MB
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}
	
	bdb := &BadgerDatabase{
		db:     db,
		path:   path,
		logger: log.New("database", "badger", "path", path),
		quitCh: make(chan struct{}),
	}
	
	// Start garbage collection for read-write instances
	if !readOnly {
		bdb.wg.Add(1)
		go bdb.gcLoop()
	}
	
	return bdb, nil
}

// gcLoop runs periodic garbage collection
func (db *BadgerDatabase) gcLoop() {
	defer db.wg.Done()
	
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			err := db.db.RunValueLogGC(0.5)
			if err != nil && !errors.Is(err, badger.ErrNoRewrite) {
				log.Warn("BadgerDB GC error", "err", err)
			}
		case <-db.quitCh:
			return
		}
	}
}

// Has checks if a key exists
func (db *BadgerDatabase) Has(key []byte) (bool, error) {
	err := db.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get retrieves a value by key
func (db *BadgerDatabase) Get(key []byte) ([]byte, error) {
	var value []byte
	
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		
		value, err = item.ValueCopy(nil)
		return err
	})
	
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, badger.ErrKeyNotFound
	}
	
	return value, err
}

// Put stores a key-value pair
func (db *BadgerDatabase) Put(key []byte, value []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// Delete removes a key
func (db *BadgerDatabase) Delete(key []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// DeleteRange deletes all keys in the range [start, end)
func (db *BadgerDatabase) DeleteRange(start, end []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = nil
		iter := txn.NewIterator(opts)
		defer iter.Close()
		
		// Collect keys to delete first to avoid iterator invalidation
		var keysToDelete [][]byte
		for iter.Seek(start); iter.Valid(); iter.Next() {
			key := iter.Item().KeyCopy(nil)
			// Check if we've reached the end of the range
			if end != nil && bytes.Compare(key, end) >= 0 {
				break
			}
			keysToDelete = append(keysToDelete, key)
		}
		
		// Now delete the collected keys
		for _, key := range keysToDelete {
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// NewBatch creates a new batch
func (db *BadgerDatabase) NewBatch() ethdb.Batch {
	return &badgerBatch{
		db:      db,
		entries: make(map[string]*batchEntry),
	}
}

// NewBatchWithSize creates a new batch with size hint
func (db *BadgerDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return &badgerBatch{
		db:      db,
		entries: make(map[string]*batchEntry, size/64), // Estimate entries
	}
}

// NewIterator creates a new iterator
func (db *BadgerDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return &badgerIterator{
		db:     db.db,
		prefix: prefix,
		start:  start,
	}
}

// Stat returns database statistics
func (db *BadgerDatabase) Stat() (string, error) {
	lsm, vlog := db.db.Size()
	
	// Return all statistics in a formatted string
	stats := fmt.Sprintf("BadgerDB Statistics:\n")
	stats += fmt.Sprintf("  LSM Size: %d bytes\n", lsm)
	stats += fmt.Sprintf("  ValueLog Size: %d bytes\n", vlog)
	stats += fmt.Sprintf("  Total Size: %d bytes\n", lsm+vlog)
	stats += fmt.Sprintf("  Path: %s\n", db.path)
	
	return stats, nil
}

// Compact runs manual compaction
func (db *BadgerDatabase) Compact(start []byte, limit []byte) error {
	// BadgerDB handles compaction automatically
	// But we can trigger a manual compaction
	return db.db.Flatten(1)
}

// NewSnapshot creates a new snapshot using BadgerDB's native support
func (db *BadgerDatabase) NewSnapshot() (ethdb.Snapshot, error) {
	return NewBadgerSnapshot(db.db)
}

// Sync flushes all pending writes to disk
func (db *BadgerDatabase) Sync() error {
	return db.db.Sync()
}

// SyncKeyValue flushes all pending writes to disk
func (db *BadgerDatabase) SyncKeyValue() error {
	return db.db.Sync()
}

// Close closes the database
func (db *BadgerDatabase) Close() error {
	close(db.quitCh)
	db.wg.Wait()
	return db.db.Close()
}

// Path returns the path to the database
func (db *BadgerDatabase) Path() string {
	return db.path
}

// Ancient store methods (not supported in BadgerDB)

// HasAncient returns whether an ancient binary blob is available in the ancient store.
func (db *BadgerDatabase) HasAncient(kind string, number uint64) (bool, error) {
	return false, nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (db *BadgerDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errors.New("ancient store not supported in BadgerDB")
}

// AncientRange retrieves multiple items in sequence, starting from the index 'start'.
func (db *BadgerDatabase) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errors.New("ancient store not supported in BadgerDB")
}

// Ancients returns the ancient item numbers in the ancient store.
func (db *BadgerDatabase) Ancients() (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// Tail returns the number of first stored item in the ancient store.
func (db *BadgerDatabase) Tail() (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// AncientSize returns the ancient size of the specified category.
func (db *BadgerDatabase) AncientSize(kind string) (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// ReadAncients runs the given read operation while ensuring that no writes take place.
func (db *BadgerDatabase) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	return errors.New("ancient store not supported in BadgerDB")
}

// ModifyAncients runs a write operation on the ancient store.
func (db *BadgerDatabase) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// TruncateAncients discards all but the first n ancient data from the ancient store.
func (db *BadgerDatabase) TruncateAncients(n uint64) error {
	return errors.New("ancient store not supported in BadgerDB")
}

// AncientDatadir returns the path to the ancient data directory.
func (db *BadgerDatabase) AncientDatadir() (string, error) {
	return "", errors.New("ancient store not supported in BadgerDB")
}

// MigrateTable migrates a table from one database to another.
func (db *BadgerDatabase) MigrateTable(s string, f func([]byte) ([]byte, error)) error {
	return errors.New("table migration not supported in BadgerDB")
}

// SyncAncient flushes all in-memory ancient store data to disk.
func (db *BadgerDatabase) SyncAncient() error {
	return errors.New("ancient store not supported in BadgerDB")
}

// TruncateHead discards all but the first n ancient data from the ancient store.
// Returns the previous head position.
func (db *BadgerDatabase) TruncateHead(n uint64) (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// TruncateTail discards the first n ancient data from the ancient store.
// Returns the previous tail position.
func (db *BadgerDatabase) TruncateTail(n uint64) (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}


// View runs a read-only transaction
func (db *BadgerDatabase) View(fn func(txn *badger.Txn) error) error {
	return db.db.View(fn)
}

// NewTransaction creates a new transaction
func (db *BadgerDatabase) NewTransaction(update bool) *badger.Txn {
	return db.db.NewTransaction(update)
}

// badgerBatch implements ethdb.Batch
type badgerBatch struct {
	db      *BadgerDatabase
	entries map[string]*batchEntry
	size    int
}

type batchEntry struct {
	value  []byte
	delete bool
}

func (b *badgerBatch) Put(key, value []byte) error {
	b.entries[string(key)] = &batchEntry{value: value, delete: false}
	b.size += len(key) + len(value)
	return nil
}

func (b *badgerBatch) Delete(key []byte) error {
	b.entries[string(key)] = &batchEntry{delete: true}
	b.size += len(key)
	return nil
}

func (b *badgerBatch) DeleteRange(start, end []byte) error {
	// We need to iterate through the database to find keys in range
	// This is not efficient in a batch, but we implement it for compatibility
	return b.db.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		iter := txn.NewIterator(opts)
		defer iter.Close()
		
		for iter.Seek(start); iter.Valid(); iter.Next() {
			key := iter.Item().KeyCopy(nil)
			// Check if we've reached the end of the range
			if end != nil && bytes.Compare(key, end) >= 0 {
				break
			}
			// Add to batch as delete operation
			b.entries[string(key)] = &batchEntry{delete: true}
			b.size += len(key)
		}
		return nil
	})
}

func (b *badgerBatch) ValueSize() int {
	return b.size
}

func (b *badgerBatch) Write() error {
	if len(b.entries) == 0 {
		return nil
	}
	
	return b.db.db.Update(func(txn *badger.Txn) error {
		for key, entry := range b.entries {
			if entry.delete {
				if err := txn.Delete([]byte(key)); err != nil {
					return err
				}
			} else {
				if err := txn.Set([]byte(key), entry.value); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (b *badgerBatch) Reset() {
	b.entries = make(map[string]*batchEntry)
	b.size = 0
}

func (b *badgerBatch) Replay(w ethdb.KeyValueWriter) error {
	for key, entry := range b.entries {
		if entry.delete {
			if err := w.Delete([]byte(key)); err != nil {
				return err
			}
		} else {
			if err := w.Put([]byte(key), entry.value); err != nil {
				return err
			}
		}
	}
	return nil
}

// badgerIterator implements ethdb.Iterator
type badgerIterator struct {
	db     *badger.DB
	txn    *badger.Txn
	iter   *badger.Iterator
	prefix []byte
	start  []byte
	
	key        []byte
	value      []byte
	err        error
	initialized bool
	firstNext   bool
}

func (it *badgerIterator) init() {
	if it.initialized {
		return
	}
	
	it.txn = it.db.NewTransaction(false)
	opts := badger.DefaultIteratorOptions
	opts.Prefix = it.prefix
	
	it.iter = it.txn.NewIterator(opts)
	
	// Determine the seek position
	seekKey := it.prefix
	if len(it.start) > 0 {
		// If we have both prefix and start, we need to seek to prefix+start
		// unless start already contains the prefix
		if bytes.HasPrefix(it.start, it.prefix) {
			seekKey = it.start
		} else {
			seekKey = append(it.prefix, it.start...)
		}
	}
	
	if len(seekKey) > 0 {
		it.iter.Seek(seekKey)
	} else {
		it.iter.Rewind()
	}
	
	it.initialized = true
	it.firstNext = true
}

func (it *badgerIterator) loadCurrent() {
	if !it.iter.Valid() {
		it.key = nil
		it.value = nil
		return
	}
	
	item := it.iter.Item()
	it.key = item.KeyCopy(nil)
	
	value, err := item.ValueCopy(nil)
	if err != nil {
		it.err = err
		return
	}
	it.value = value
}

func (it *badgerIterator) Next() bool {
	it.init()
	
	if it.err != nil {
		return false
	}
	
	// On first call to Next(), just load current position
	if it.firstNext {
		it.firstNext = false
		it.loadCurrent()
		return it.iter.Valid()
	}
	
	// Subsequent calls advance the iterator
	if !it.iter.Valid() {
		return false
	}
	
	it.iter.Next()
	it.loadCurrent()
	
	return it.iter.Valid()
}

func (it *badgerIterator) Error() error {
	return it.err
}

func (it *badgerIterator) Key() []byte {
	return it.key
}

func (it *badgerIterator) Value() []byte {
	return it.value
}

func (it *badgerIterator) Release() {
	if it.iter != nil {
		it.iter.Close()
	}
	if it.txn != nil {
		it.txn.Discard()
	}
}