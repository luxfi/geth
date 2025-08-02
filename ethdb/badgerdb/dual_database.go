package badgerdb

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/badger/v3"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
)

// DualBadgerDatabase implements a dual-database architecture using BadgerDB
// - Archive DB: Read-only shared database for finalized blocks (optional)
// - Current DB: Read-write database for recent/unfinalized blocks
type DualBadgerDatabase struct {
	archiveDB *BadgerDatabase // Can be nil if no archive specified
	currentDB *BadgerDatabase
	
	// Configuration
	hasArchive      bool
	finalityHeight  uint64 // Blocks below this are considered finalized
	
	// Metrics
	archiveReads  atomic.Uint64
	currentReads  atomic.Uint64
	currentWrites atomic.Uint64
	
	mu sync.RWMutex
}

// DualDatabaseConfig holds configuration for dual database
type DualDatabaseConfig struct {
	ArchivePath    string // Empty means no archive (C-Chain starts fresh)
	CurrentPath    string // Required - current chaindata
	FinalityHeight uint64 // Height below which blocks are finalized
	ArchiveShared  bool   // If true, archive uses BypassLockGuard for shared access
}

// NewDualBadgerDatabase creates a new dual database instance
func NewDualBadgerDatabase(config DualDatabaseConfig) (*DualBadgerDatabase, error) {
	if config.CurrentPath == "" {
		return nil, errors.New("current database path is required")
	}
	
	ddb := &DualBadgerDatabase{
		hasArchive:     config.ArchivePath != "",
		finalityHeight: config.FinalityHeight,
	}
	
	// Open archive database if specified (read-only, possibly shared)
	if config.ArchivePath != "" {
		archiveDB, err := NewBadgerDatabase(config.ArchivePath, true, config.ArchiveShared)
		if err != nil {
			return nil, fmt.Errorf("failed to open archive database: %w", err)
		}
		ddb.archiveDB = archiveDB
		
		log.Info("Dual database: archive enabled",
			"path", config.ArchivePath,
			"shared", config.ArchiveShared,
			"finality", config.FinalityHeight)
	} else {
		log.Info("Dual database: no archive, starting fresh")
	}
	
	// Open current database (read-write)
	currentDB, err := NewBadgerDatabase(config.CurrentPath, false, false)
	if err != nil {
		if ddb.archiveDB != nil {
			ddb.archiveDB.Close()
		}
		return nil, fmt.Errorf("failed to open current database: %w", err)
	}
	ddb.currentDB = currentDB
	
	log.Info("Dual BadgerDB initialized",
		"archive", config.ArchivePath,
		"current", config.CurrentPath,
		"has_history", ddb.hasArchive)
	
	return ddb, nil
}

// Has checks if a key exists in either database
func (ddb *DualBadgerDatabase) Has(key []byte) (bool, error) {
	// Check current first (more likely for recent data)
	if has, err := ddb.currentDB.Has(key); err != nil || has {
		return has, err
	}
	
	// If we have archive, check there
	if ddb.hasArchive && ddb.archiveDB != nil {
		return ddb.archiveDB.Has(key)
	}
	
	return false, nil
}

// Get retrieves from appropriate database
func (ddb *DualBadgerDatabase) Get(key []byte) ([]byte, error) {
	// Try current first
	if val, err := ddb.currentDB.Get(key); err == nil {
		ddb.currentReads.Add(1)
		return val, nil
	} else if !errors.Is(err, badger.ErrKeyNotFound) {
		return nil, err
	}
	
	// Fall back to archive if available
	if ddb.hasArchive && ddb.archiveDB != nil {
		ddb.archiveReads.Add(1)
		return ddb.archiveDB.Get(key)
	}
	
	return nil, badger.ErrKeyNotFound
}

// Put writes to current database only
func (ddb *DualBadgerDatabase) Put(key []byte, value []byte) error {
	ddb.currentWrites.Add(1)
	return ddb.currentDB.Put(key, value)
}

// Delete removes from current database only
func (ddb *DualBadgerDatabase) Delete(key []byte) error {
	return ddb.currentDB.Delete(key)
}

// DeleteRange removes all keys in the given range from the current database
func (ddb *DualBadgerDatabase) DeleteRange(start []byte, end []byte) error {
	return ddb.currentDB.DeleteRange(start, end)
}

// ReadBlock reads a block, routing based on height
func (ddb *DualBadgerDatabase) ReadBlock(hash common.Hash, number uint64) *types.Block {
	// If we have archive and block is finalized, try archive first
	if ddb.hasArchive && ddb.archiveDB != nil && number < ddb.finalityHeight {
		ddb.archiveReads.Add(1)
		if block := ddb.readBlockFromDB(ddb.archiveDB, hash, number); block != nil {
			return block
		}
	}
	
	// Otherwise read from current
	ddb.currentReads.Add(1)
	return ddb.readBlockFromDB(ddb.currentDB, hash, number)
}

// readBlockFromDB is a helper to read block from specific database
func (ddb *DualBadgerDatabase) readBlockFromDB(db *BadgerDatabase, hash common.Hash, number uint64) *types.Block {
	// This would use rawdb functions, simplified here
	headerKey := append([]byte("h"), hash.Bytes()...)
	headerKey = append(headerKey, encodeBlockNumber(number)...)
	
	headerData, err := db.Get(headerKey)
	if err != nil {
		return nil
	}
	
	// Decode header and body (simplified)
	// In real implementation, use rawdb.ReadBlock
	_ = headerData
	return nil
}

// NewBatch creates a new batch for current database
func (ddb *DualBadgerDatabase) NewBatch() ethdb.Batch {
	return ddb.currentDB.NewBatch()
}

// NewBatchWithSize creates a new batch with size hint
func (ddb *DualBadgerDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return ddb.currentDB.NewBatchWithSize(size)
}

// NewIterator creates an iterator that spans both databases
func (ddb *DualBadgerDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	if ddb.hasArchive && ddb.archiveDB != nil {
		// Create a combined iterator
		return &dualIterator{
			archiveIter: ddb.archiveDB.NewIterator(prefix, start),
			currentIter: ddb.currentDB.NewIterator(prefix, start),
			useArchive:  true,
		}
	}
	
	// No archive, just use current
	return ddb.currentDB.NewIterator(prefix, start)
}

// Stat returns combined statistics
func (ddb *DualBadgerDatabase) Stat() (string, error) {
	currentStat, _ := ddb.currentDB.Stat()
	
	stats := "DualBadgerDatabase Statistics:\n"
	
	if ddb.hasArchive && ddb.archiveDB != nil {
		archiveStat, _ := ddb.archiveDB.Stat()
		stats += "\n=== Archive Database ===\n"
		stats += archiveStat
		stats += fmt.Sprintf("\nArchive Reads: %d\n", ddb.archiveReads.Load())
	}
	
	stats += "\n=== Current Database ===\n"
	stats += currentStat
	stats += fmt.Sprintf("\nCurrent Reads: %d\n", ddb.currentReads.Load())
	stats += fmt.Sprintf("Current Writes: %d\n", ddb.currentWrites.Load())
	stats += fmt.Sprintf("Finality Height: %d\n", ddb.finalityHeight)
	
	return stats, nil
}

// Compact compacts both databases
func (ddb *DualBadgerDatabase) Compact(start []byte, limit []byte) error {
	if err := ddb.currentDB.Compact(start, limit); err != nil {
		return err
	}
	
	if ddb.hasArchive && ddb.archiveDB != nil {
		return ddb.archiveDB.Compact(start, limit)
	}
	
	return nil
}

// NewSnapshot creates a snapshot from the current database
func (ddb *DualBadgerDatabase) NewSnapshot() (ethdb.Snapshot, error) {
	// Snapshots only apply to the current database since that's where writes go
	return ddb.currentDB.NewSnapshot()
}

// Close closes both databases
func (ddb *DualBadgerDatabase) Close() error {
	var errs []error
	
	if err := ddb.currentDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close current DB: %w", err))
	}
	
	if ddb.hasArchive && ddb.archiveDB != nil {
		if err := ddb.archiveDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close archive DB: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (ddb *DualBadgerDatabase) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errors.New("ancient store not supported in BadgerDB")
}

// HasAncient returns whether an ancient binary blob is available in the ancient store.
func (ddb *DualBadgerDatabase) HasAncient(kind string, number uint64) (bool, error) {
	return false, nil
}

// AncientRange retrieves multiple items in sequence, starting from the index 'start'.
func (ddb *DualBadgerDatabase) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errors.New("ancient store not supported in BadgerDB")
}

// Ancients returns the ancient item numbers in the ancient store.
func (ddb *DualBadgerDatabase) Ancients() (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// AncientSize returns the ancient size of the specified category.
func (ddb *DualBadgerDatabase) AncientSize(kind string) (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// ModifyAncients runs a write operation on the ancient store.
func (ddb *DualBadgerDatabase) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// TruncateAncients discards all but the first n ancient data from the ancient store.
func (ddb *DualBadgerDatabase) TruncateAncients(n uint64) error {
	return errors.New("ancient store not supported in BadgerDB")
}

// Sync flushes all in-memory ancient store data to disk.
func (ddb *DualBadgerDatabase) Sync() error {
	return ddb.currentDB.Sync()
}

// SyncKeyValue flushes all pending writes to disk
func (ddb *DualBadgerDatabase) SyncKeyValue() error {
	return ddb.currentDB.SyncKeyValue()
}

// AncientDatadir returns the path to the ancient data directory.
func (ddb *DualBadgerDatabase) AncientDatadir() (string, error) {
	return "", errors.New("ancient store not supported in BadgerDB")
}

// MigrateTable migrates a table from one database to another.
func (ddb *DualBadgerDatabase) MigrateTable(s string, f func([]byte) ([]byte, error)) error {
	return errors.New("table migration not supported in BadgerDB")
}

// ReadAncients runs a read operation on the ancient store.
func (ddb *DualBadgerDatabase) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	return errors.New("ancient store not supported in BadgerDB")
}

// Tail returns the number of first stored item in the ancient store.
func (ddb *DualBadgerDatabase) Tail() (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// TruncateHead discards all but the first n ancient data from the ancient store.
// Returns the previous head position.
func (ddb *DualBadgerDatabase) TruncateHead(n uint64) (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// TruncateTail discards the first n ancient data from the ancient store.
// Returns the previous tail position.
func (ddb *DualBadgerDatabase) TruncateTail(n uint64) (uint64, error) {
	return 0, errors.New("ancient store not supported in BadgerDB")
}

// SyncAncient flushes all in-memory ancient store data to disk.
func (ddb *DualBadgerDatabase) SyncAncient() error {
	return errors.New("ancient store not supported in BadgerDB")
}

// dualIterator combines iterators from both databases
type dualIterator struct {
	archiveIter ethdb.Iterator
	currentIter ethdb.Iterator
	useArchive  bool
}

func (it *dualIterator) Next() bool {
	if it.useArchive && it.archiveIter.Next() {
		return true
	}
	it.useArchive = false
	return it.currentIter.Next()
}

func (it *dualIterator) Error() error {
	if err := it.archiveIter.Error(); err != nil {
		return err
	}
	return it.currentIter.Error()
}

func (it *dualIterator) Key() []byte {
	if it.useArchive {
		return it.archiveIter.Key()
	}
	return it.currentIter.Key()
}

func (it *dualIterator) Value() []byte {
	if it.useArchive {
		return it.archiveIter.Value()
	}
	return it.currentIter.Value()
}

func (it *dualIterator) Release() {
	it.archiveIter.Release()
	it.currentIter.Release()
}