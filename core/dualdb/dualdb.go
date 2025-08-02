package dualdb

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNotInitialized = errors.New("dual database not initialized")
	ErrInvalidConfig  = errors.New("invalid dual database configuration")
)

// DualDatabase implements a two-tier database system:
// - Archive DB: Read-only, finalized blocks (can be shared across nodes)
// - Current DB: Read-write, recent blocks
type DualDatabase struct {
	config *Config

	// Archive database (read-only, finalized blocks)
	archiveDB ethdb.Database
	
	// Current database (read-write, recent blocks)
	currentDB ethdb.Database

	// Latest finalized block
	finalizedHeight atomic.Uint64
	finalizedHash   atomic.Value // common.Hash

	// Synchronization
	mu sync.RWMutex

	// Metrics
	archiveReads  atomic.Uint64
	currentReads  atomic.Uint64
	currentWrites atomic.Uint64

	// Maintenance goroutines
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewDualDatabase creates a new dual database instance
func NewDualDatabase(config *Config) (*DualDatabase, error) {
	if !config.Enabled {
		return nil, ErrInvalidConfig
	}

	db := &DualDatabase{
		config: config,
		stopCh: make(chan struct{}),
	}

	// Open archive database (read-only, can bypass lock for shared access)
	archiveDB, err := db.openArchiveDB()
	if err != nil {
		return nil, fmt.Errorf("failed to open archive database: %w", err)
	}
	db.archiveDB = archiveDB

	// Open current database (read-write)
	currentDB, err := db.openCurrentDB()
	if err != nil {
		archiveDB.Close()
		return nil, fmt.Errorf("failed to open current database: %w", err)
	}
	db.currentDB = currentDB

	// Initialize finalized height from archive
	if err := db.initializeFinalizedState(); err != nil {
		archiveDB.Close()
		currentDB.Close()
		return nil, fmt.Errorf("failed to initialize finalized state: %w", err)
	}

	// Start maintenance routines
	if config.Maintenance.AutoArchive {
		db.wg.Add(1)
		go db.archiveLoop()
	}

	log.Info("Dual database initialized",
		"archive_path", config.Archive.Path,
		"current_path", config.Current.Path,
		"finalized_height", db.finalizedHeight.Load())

	return db, nil
}

// openArchiveDB opens the archive database with appropriate settings
func (db *DualDatabase) openArchiveDB() (ethdb.Database, error) {
	switch db.config.Archive.Type {
	case "pebble":
		// Open with read-only mode and bypass lock for shared access
		opts := rawdb.DefaultPebbleOptions()
		opts.ReadOnly = true
		// Note: PebbleDB doesn't need explicit lock bypass in read-only mode
		
		return rawdb.NewPebbleDBDatabase(db.config.Archive.Path, 
			db.config.Archive.CacheSize, 0, "", opts)
			
	case "leveldb":
		// LevelDB with read-only settings
		return rawdb.NewLevelDBDatabaseWithFreezer(db.config.Archive.Path,
			db.config.Archive.CacheSize, 0, "", "", true)
			
	default:
		return nil, fmt.Errorf("unsupported archive database type: %s", db.config.Archive.Type)
	}
}

// openCurrentDB opens the current database for read-write
func (db *DualDatabase) openCurrentDB() (ethdb.Database, error) {
	switch db.config.Current.Type {
	case "pebble":
		return rawdb.NewPebbleDBDatabase(db.config.Current.Path,
			db.config.Current.CacheSize, 0, "", nil)
			
	case "leveldb":
		return rawdb.NewLevelDBDatabase(db.config.Current.Path,
			db.config.Current.CacheSize, 0, "", false)
			
	default:
		return nil, fmt.Errorf("unsupported current database type: %s", db.config.Current.Type)
	}
}

// initializeFinalizedState loads the latest finalized block info
func (db *DualDatabase) initializeFinalizedState() error {
	// Try to get finalized info from current DB first
	if height := rawdb.ReadFinalizedBlockHeight(db.currentDB); height != nil {
		hash := rawdb.ReadCanonicalHash(db.archiveDB, *height)
		if hash != (common.Hash{}) {
			db.finalizedHeight.Store(*height)
			db.finalizedHash.Store(hash)
			return nil
		}
	}

	// Otherwise, find the latest block in archive
	// This would need to iterate or maintain an index
	return nil
}

// ReadBlock reads a block, routing to appropriate database
func (db *DualDatabase) ReadBlock(hash common.Hash, number uint64) *types.Block {
	finalizedHeight := db.finalizedHeight.Load()
	
	// Route based on finalized height
	if number <= finalizedHeight {
		db.archiveReads.Add(1)
		return rawdb.ReadBlock(db.archiveDB, hash, number)
	}
	
	db.currentReads.Add(1)
	return rawdb.ReadBlock(db.currentDB, hash, number)
}

// WriteBlock writes a block to the current database
func (db *DualDatabase) WriteBlock(block *types.Block) error {
	db.currentWrites.Add(1)
	rawdb.WriteBlock(db.currentDB, block)
	return nil
}

// Has checks if a key exists in either database
func (db *DualDatabase) Has(key []byte) (bool, error) {
	// Check current first (more likely for recent data)
	if has, err := db.currentDB.Has(key); err != nil || has {
		return has, err
	}
	
	// Then check archive
	return db.archiveDB.Has(key)
}

// Get retrieves from appropriate database
func (db *DualDatabase) Get(key []byte) ([]byte, error) {
	// Try current first
	if val, err := db.currentDB.Get(key); err == nil {
		db.currentReads.Add(1)
		return val, nil
	}
	
	// Fall back to archive
	db.archiveReads.Add(1)
	return db.archiveDB.Get(key)
}

// Put writes to current database only
func (db *DualDatabase) Put(key []byte, value []byte) error {
	db.currentWrites.Add(1)
	return db.currentDB.Put(key, value)
}

// Delete removes from current database only
func (db *DualDatabase) Delete(key []byte) error {
	return db.currentDB.Delete(key)
}

// archiveLoop periodically moves finalized blocks to archive
func (db *DualDatabase) archiveLoop() {
	defer db.wg.Done()
	
	ticker := time.NewTicker(db.config.Maintenance.ArchiveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := db.archiveFinalized(); err != nil {
				log.Error("Failed to archive finalized blocks", "err", err)
			}
		case <-db.stopCh:
			return
		}
	}
}

// archiveFinalized moves finalized blocks from current to archive
func (db *DualDatabase) archiveFinalized() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Get current head
	currentHead := rawdb.ReadHeadBlockHash(db.currentDB)
	if currentHead == (common.Hash{}) {
		return nil
	}
	
	currentNumber := rawdb.ReadHeaderNumber(db.currentDB, currentHead)
	if currentNumber == nil {
		return nil
	}

	// Calculate new finalized height
	if *currentNumber <= db.config.FinalityDelay {
		return nil // Not enough blocks yet
	}
	
	newFinalizedHeight := *currentNumber - db.config.FinalityDelay
	oldFinalizedHeight := db.finalizedHeight.Load()
	
	if newFinalizedHeight <= oldFinalizedHeight {
		return nil // Nothing new to finalize
	}

	log.Info("Archiving finalized blocks",
		"from", oldFinalizedHeight+1,
		"to", newFinalizedHeight)

	// Move blocks to archive
	batch := db.archiveDB.NewBatch()
	for num := oldFinalizedHeight + 1; num <= newFinalizedHeight; num++ {
		hash := rawdb.ReadCanonicalHash(db.currentDB, num)
		if hash == (common.Hash{}) {
			continue
		}

		// Copy block data
		block := rawdb.ReadBlock(db.currentDB, hash, num)
		if block == nil {
			continue
		}

		rawdb.WriteBlock(batch, block)
		
		// Copy receipts
		receipts := rawdb.ReadReceipts(db.currentDB, hash, num, block.Time(), nil)
		rawdb.WriteReceipts(batch, hash, num, receipts)
		
		// TODO: Copy state data if needed
	}

	if err := batch.Write(); err != nil {
		return fmt.Errorf("failed to write archive batch: %w", err)
	}

	// Update finalized markers
	db.finalizedHeight.Store(newFinalizedHeight)
	finalizedHash := rawdb.ReadCanonicalHash(db.currentDB, newFinalizedHeight)
	db.finalizedHash.Store(finalizedHash)
	
	// Save finalized height to current DB
	rawdb.WriteFinalizedBlockHeight(db.currentDB, newFinalizedHeight)

	// Optionally prune old blocks from current DB
	if db.config.Current.RetentionBlocks > 0 {
		pruneBelow := newFinalizedHeight
		if pruneBelow > db.config.Current.RetentionBlocks {
			pruneBelow = newFinalizedHeight - db.config.Current.RetentionBlocks
		}
		
		// TODO: Implement pruning of old blocks from current DB
		log.Debug("Would prune current DB", "below", pruneBelow)
	}

	return nil
}

// Close shuts down the dual database
func (db *DualDatabase) Close() error {
	close(db.stopCh)
	db.wg.Wait()

	var errs []error
	if err := db.currentDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close current DB: %w", err))
	}
	if err := db.archiveDB.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close archive DB: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// Implement remaining ethdb.Database interface methods...
func (db *DualDatabase) NewBatch() ethdb.Batch {
	// Batches only write to current DB
	return db.currentDB.NewBatch()
}

func (db *DualDatabase) NewBatchWithSize(size int) ethdb.Batch {
	return db.currentDB.NewBatchWithSize(size)
}

func (db *DualDatabase) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	// Create a combined iterator that reads from both databases
	return NewDualIterator(db.archiveDB.NewIterator(prefix, start),
		db.currentDB.NewIterator(prefix, start))
}

func (db *DualDatabase) Stat(property string) (string, error) {
	// Combine stats from both databases
	archiveStat, _ := db.archiveDB.Stat(property)
	currentStat, _ := db.currentDB.Stat(property)
	
	return fmt.Sprintf("archive: %s, current: %s, reads: %d/%d, writes: %d",
		archiveStat, currentStat,
		db.archiveReads.Load(), db.currentReads.Load(),
		db.currentWrites.Load()), nil
}

func (db *DualDatabase) Compact(start []byte, limit []byte) error {
	// Compact both databases
	if err := db.archiveDB.Compact(start, limit); err != nil {
		return err
	}
	return db.currentDB.Compact(start, limit)
}

func (db *DualDatabase) NewSnapshot() (ethdb.Snapshot, error) {
	// Snapshots from current DB only
	return db.currentDB.NewSnapshot()
}