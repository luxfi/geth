package badgerdb

import (
	"fmt"
	"sync/atomic"
	"time"
	
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/rlp"
)

// FreezerConfig holds configuration for the freezer
type FreezerConfig struct {
	AncientPath    string // Path to ancient store
	FreezeThreshold uint64 // Number of blocks to keep in main DB
	BatchSize      uint64 // Number of blocks to freeze at once
}

// Freezer manages the migration of finalized blocks to ancient store
type Freezer struct {
	mainDB   ethdb.Database
	ancientDB *BadgerDatabaseWithAncient
	
	config FreezerConfig
	
	// Metrics
	frozenBlocks atomic.Uint64
	lastFrozen   atomic.Uint64
	
	// Control
	quitCh chan struct{}
}

// NewFreezer creates a new freezer instance
func NewFreezer(mainDB ethdb.Database, config FreezerConfig) (*Freezer, error) {
	// Open the ancient database
	ancientDB, err := NewBadgerDatabaseWithAncient(config.AncientPath, "", false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open ancient database: %w", err)
	}
	
	freezer := &Freezer{
		mainDB:    mainDB,
		ancientDB: ancientDB,
		config:    config,
		quitCh:    make(chan struct{}),
	}
	
	// Get the current ancient head
	ancients, err := ancientDB.Ancients()
	if err == nil && ancients > 0 {
		freezer.lastFrozen.Store(ancients - 1)
	}
	
	return freezer, nil
}

// FreezeBlocks freezes blocks from the main database to the ancient store
func (f *Freezer) FreezeBlocks(head uint64) error {
	// Calculate what needs to be frozen
	if head <= f.config.FreezeThreshold {
		return nil // Nothing to freeze
	}
	
	freezeTarget := head - f.config.FreezeThreshold
	lastFrozen := f.lastFrozen.Load()
	
	if freezeTarget <= lastFrozen {
		return nil // Already frozen
	}
	
	log.Info("Freezing blocks", 
		"from", lastFrozen+1, 
		"to", freezeTarget,
		"head", head)
	
	// Freeze in batches
	for start := lastFrozen + 1; start <= freezeTarget; {
		end := start + f.config.BatchSize - 1
		if end > freezeTarget {
			end = freezeTarget
		}
		
		if err := f.freezeBatch(start, end); err != nil {
			return fmt.Errorf("failed to freeze batch %d-%d: %w", start, end, err)
		}
		
		start = end + 1
		f.lastFrozen.Store(end)
		
		// Allow interruption
		select {
		case <-f.quitCh:
			return nil
		default:
		}
	}
	
	log.Info("Freezing complete", "frozen", f.frozenBlocks.Load())
	return nil
}

// freezeBatch freezes a batch of blocks
func (f *Freezer) freezeBatch(start, end uint64) error {
	var blocks []*types.Block
	var receipts []types.Receipts
	
	// Read blocks from main database
	for num := start; num <= end; num++ {
		hash := rawdb.ReadCanonicalHash(f.mainDB, num)
		if hash == (common.Hash{}) {
			return fmt.Errorf("canonical hash not found for block %d", num)
		}
		
		block := rawdb.ReadBlock(f.mainDB, hash, num)
		if block == nil {
			return fmt.Errorf("block %d not found", num)
		}
		
		// Read receipts with proper parameters
		blockReceipts := rawdb.ReadReceipts(f.mainDB, hash, num, block.Time(), params.AllDevChainProtocolChanges)
		
		blocks = append(blocks, block)
		receipts = append(receipts, blockReceipts)
	}
	
	// Encode receipts
	encodedReceipts := make([]rlp.RawValue, len(receipts))
	for i, blockReceipts := range receipts {
		encoded, err := rlp.EncodeToBytes(blockReceipts)
		if err != nil {
			return fmt.Errorf("failed to encode receipts for block %d: %w", start+uint64(i), err)
		}
		encodedReceipts[i] = encoded
	}
	
	// Write to ancient store
	written, err := rawdb.WriteAncientBlocks(f.ancientDB, blocks, encodedReceipts)
	if err != nil {
		return fmt.Errorf("failed to write ancient blocks: %w", err)
	}
	
	f.frozenBlocks.Add(uint64(len(blocks)))
	log.Info("Froze blocks", "from", start, "to", end, "written", written)
	
	// Delete from main database (optional - can be done separately)
	// This is typically done after verifying the ancient write succeeded
	// for num := start; num <= end; num++ {
	//     hash := rawdb.ReadCanonicalHash(f.mainDB, num)
	//     rawdb.DeleteBlock(f.mainDB, hash, num)
	// }
	
	return nil
}

// StartAutoFreeze starts automatic freezing in the background
func (f *Freezer) StartAutoFreeze(getHead func() uint64, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				head := getHead()
				if err := f.FreezeBlocks(head); err != nil {
					log.Error("Auto-freeze failed", "err", err)
				}
			case <-f.quitCh:
				return
			}
		}
	}()
}

// Stop stops the freezer
func (f *Freezer) Stop() error {
	close(f.quitCh)
	return f.ancientDB.Close()
}

// CreateReadOnlyMount creates a read-only mount of the ancient store
func CreateReadOnlyMount(ancientPath string) (*BadgerDatabaseWithAncient, error) {
	return NewBadgerDatabaseWithAncient(ancientPath, "", true, true)
}

// ExportAncientSnapshot exports the ancient store to a compressed archive
func ExportAncientSnapshot(ancientPath, outputPath string) error {
	// This would create a tar.gz of the ancient store for easy distribution
	// Implementation would use archive/tar and compress/gzip
	// For now, just a placeholder
	return fmt.Errorf("not implemented yet")
}