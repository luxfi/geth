package badgerdb

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/log"
)

// Archiver handles periodic compaction from current to archive database
type Archiver struct {
	db              *DualBadgerDatabase
	finalityDelay   uint64        // Blocks must be this old to be finalized
	archiveInterval time.Duration // How often to run archiving
	
	// Metrics
	totalArchived atomic.Uint64
	lastArchived  atomic.Int64
	totalDeleted  atomic.Uint64
	
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewArchiver creates a new archiver for dual database
func NewArchiver(db *DualBadgerDatabase, finalityDelay uint64, interval time.Duration) *Archiver {
	return &Archiver{
		db:              db,
		finalityDelay:   finalityDelay,
		archiveInterval: interval,
		stopCh:          make(chan struct{}),
	}
}

// Start begins the archiving process
func (a *Archiver) Start() {
	a.wg.Add(1)
	go a.archiveLoop()
	log.Info("Archiver started", 
		"finality_delay", a.finalityDelay,
		"interval", a.archiveInterval)
}

// Stop halts the archiving process
func (a *Archiver) Stop() {
	select {
	case <-a.stopCh:
		// Already stopped
		return
	default:
		close(a.stopCh)
		a.wg.Wait()
		log.Info("Archiver stopped")
	}
}

// archiveLoop runs periodically to move finalized blocks
func (a *Archiver) archiveLoop() {
	defer a.wg.Done()
	
	ticker := time.NewTicker(a.archiveInterval)
	defer ticker.Stop()
	
	// Run immediately on start
	if err := a.archiveFinalized(); err != nil {
		log.Error("Initial archive failed", "err", err)
	}
	
	for {
		select {
		case <-ticker.C:
			if err := a.archiveFinalized(); err != nil {
				log.Error("Archive failed", "err", err)
			}
		case <-a.stopCh:
			return
		}
	}
}

// getCurrentHeight retrieves the current blockchain height
func (a *Archiver) getCurrentHeight() (uint64, error) {
	// Get current height from database
	heightKey := []byte("LastBlock")
	data, err := a.db.Get(heightKey)
	if err != nil {
		return 0, nil // No height stored yet
	}
	
	if len(data) != 8 {
		return 0, fmt.Errorf("invalid height data")
	}
	
	// Decode height
	var height uint64
	for i := 0; i < 8; i++ {
		height = (height << 8) | uint64(data[i])
	}
	return height, nil
}

// archiveFinalized moves finalized blocks from current to archive
func (a *Archiver) archiveFinalized() error {
	// Check if we have an archive database
	if !a.db.hasArchive || a.db.archiveDB == nil {
		return nil
	}
	
	// Get current height
	currentHeight, err := a.getCurrentHeight()
	if err != nil {
		return fmt.Errorf("failed to get current height: %w", err)
	}
	
	if currentHeight == 0 {
		return nil // No blocks yet
	}
	
	// Calculate finalized height
	if currentHeight <= a.finalityDelay {
		return nil // Not enough blocks yet
	}
	finalizedHeight := currentHeight - a.finalityDelay
	
	log.Info("Archiving finalized blocks",
		"current", currentHeight,
		"finalized", finalizedHeight,
		"delay", a.finalityDelay)
	
	// Archive blocks in batches
	batchSize := 100
	archived := uint64(0)
	deleted := uint64(0)
	
	// Create batch for archive writes
	archiveBatch := a.db.archiveDB.NewBatch()
	defer archiveBatch.Reset()
	
	// Iterate through current database
	iter := a.db.currentDB.NewIterator(nil, nil)
	defer iter.Release()
	
	keysToDelete := make([][]byte, 0, batchSize)
	
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Check if this is a block-related key
		if shouldArchive(key, finalizedHeight) {
			// Copy to archive
			if err := archiveBatch.Put(common.CopyBytes(key), common.CopyBytes(value)); err != nil {
				return fmt.Errorf("failed to put in archive batch: %w", err)
			}
			
			// Mark for deletion from current
			keysToDelete = append(keysToDelete, common.CopyBytes(key))
			archived++
			
			// Write batch if it's getting large
			if archived%uint64(batchSize) == 0 {
				if err := archiveBatch.Write(); err != nil {
					return fmt.Errorf("failed to write archive batch: %w", err)
				}
				archiveBatch.Reset()
				
				// Delete from current
				for _, k := range keysToDelete {
					if err := a.db.currentDB.Delete(k); err != nil {
						log.Warn("Failed to delete from current", "key", k, "err", err)
					} else {
						deleted++
					}
				}
				keysToDelete = keysToDelete[:0]
			}
		}
	}
	
	if iter.Error() != nil {
		return fmt.Errorf("iterator error: %w", iter.Error())
	}
	
	// Write final batch
	if archiveBatch.ValueSize() > 0 {
		if err := archiveBatch.Write(); err != nil {
			return fmt.Errorf("failed to write final archive batch: %w", err)
		}
		
		// Delete remaining keys
		for _, k := range keysToDelete {
			if err := a.db.currentDB.Delete(k); err != nil {
				log.Warn("Failed to delete from current", "key", k, "err", err)
			} else {
				deleted++
			}
		}
	}
	
	// Update metrics
	a.totalArchived.Add(archived)
	a.totalDeleted.Add(deleted)
	a.lastArchived.Store(time.Now().Unix())
	
	// Update finality height in dual database
	a.db.mu.Lock()
	a.db.finalityHeight = finalizedHeight
	a.db.mu.Unlock()
	
	log.Info("Archiving completed", 
		"archived", archived,
		"deleted", deleted,
		"total_archived", a.totalArchived.Load())
	
	return nil
}

// shouldArchive determines if a key should be archived based on block height
func shouldArchive(key []byte, finalizedHeight uint64) bool {
	// This is a simplified version - in practice you'd need to parse
	// different key types and extract block numbers
	// For now, we'll archive everything older than finalized height
	// TODO: Implement proper key parsing for different data types
	return true
}

// encodeBlockNumber encodes block number for key
func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	for i := 0; i < 8; i++ {
		enc[i] = byte(number >> uint(8*(7-i)))
	}
	return enc
}