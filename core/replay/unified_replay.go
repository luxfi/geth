// Copyright 2025 Lux Industries, Inc.
// Unified Block Replay System for migrated data

package replay

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/rlp"
)

// ReplayConfig configures the unified replay system
type ReplayConfig struct {
	// BadgerDB path for migrated blocks
	BadgerDBPath string
	
	// Target database for replayed blocks
	TargetDB ethdb.Database
	
	// Chain configuration
	ChainConfig *params.ChainConfig
	
	// Network parameters
	NetworkID     uint64
	Layer         uint8  // 0=C-Chain, 1=L1, 2=L2, 3=L3
	ConsensusType uint8  // 0=POA, 1=POS, 2=POW
	
	// Replay parameters
	StartBlock      uint64
	EndBlock        uint64
	BatchSize       int
	UpgradeToQuantum bool
	
	// Progress callback
	OnProgress func(current, total uint64)
}

// UnifiedReplayer handles block replay with format conversion
type UnifiedReplayer struct {
	config    *ReplayConfig
	badgerDB  *badger.DB
	blockchain *core.BlockChain
	stateDB   state.Database
	
	mu              sync.RWMutex
	currentBlock    uint64
	totalBlocks     uint64
	lastCheckpoint  time.Time
	errors          []error
}

// NewUnifiedReplayer creates a new replay instance
func NewUnifiedReplayer(config *ReplayConfig) (*UnifiedReplayer, error) {
	// Open BadgerDB
	opts := badger.DefaultOptions(config.BadgerDBPath)
	opts.ReadOnly = true
	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open BadgerDB: %w", err)
	}
	
	return &UnifiedReplayer{
		config:         config,
		badgerDB:       badgerDB,
		lastCheckpoint: time.Now(),
	}, nil
}

// Close releases resources
func (r *UnifiedReplayer) Close() error {
	if r.badgerDB != nil {
		return r.badgerDB.Close()
	}
	return nil
}

// ReplayBlocks replays blocks from BadgerDB to the unified format
func (r *UnifiedReplayer) ReplayBlocks(ctx context.Context) error {
	log.Info("Starting unified block replay",
		"start", r.config.StartBlock,
		"end", r.config.EndBlock,
		"network", r.config.NetworkID,
		"layer", r.config.Layer,
	)
	
	// Calculate total blocks
	r.totalBlocks = r.config.EndBlock - r.config.StartBlock + 1
	
	// Process blocks in batches
	batch := make([]*types.UnifiedBlock, 0, r.config.BatchSize)
	
	for blockNum := r.config.StartBlock; blockNum <= r.config.EndBlock; blockNum++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Read block from BadgerDB
		block, err := r.readBlockFromBadger(blockNum)
		if err != nil {
			r.recordError(fmt.Errorf("failed to read block %d: %w", blockNum, err))
			continue
		}
		
		// Convert to unified format
		unifiedBlock := r.convertToUnified(block)
		
		// Upgrade to quantum if requested
		if r.config.UpgradeToQuantum {
			r.upgradeToQuantum(unifiedBlock)
		}
		
		batch = append(batch, unifiedBlock)
		
		// Process batch when full
		if len(batch) >= r.config.BatchSize {
			if err := r.processBatch(batch); err != nil {
				return fmt.Errorf("failed to process batch: %w", err)
			}
			batch = batch[:0]
		}
		
		// Update progress
		r.updateProgress(blockNum)
	}
	
	// Process remaining blocks
	if len(batch) > 0 {
		if err := r.processBatch(batch); err != nil {
			return fmt.Errorf("failed to process final batch: %w", err)
		}
	}
	
	log.Info("Unified block replay completed",
		"total", r.totalBlocks,
		"errors", len(r.errors),
	)
	
	return nil
}

// readBlockFromBadger reads a block from BadgerDB
func (r *UnifiedReplayer) readBlockFromBadger(blockNum uint64) (*types.Block, error) {
	var blockData []byte
	
	err := r.badgerDB.View(func(txn *badger.Txn) error {
		// Create key for block number
		key := makeBlockKey(blockNum)
		
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		
		blockData, err = item.ValueCopy(nil)
		return err
	})
	
	if err != nil {
		return nil, err
	}
	
	// Decode RLP block
	var block types.Block
	if err := rlp.DecodeBytes(blockData, &block); err != nil {
		// Try legacy format
		var legacyBlock LegacyBlock
		if err2 := rlp.DecodeBytes(blockData, &legacyBlock); err2 == nil {
			return r.convertLegacyBlock(&legacyBlock), nil
		}
		return nil, err
	}
	
	return &block, nil
}

// LegacyBlock represents old format blocks
type LegacyBlock struct {
	Header       *LegacyHeader
	Transactions types.Transactions
	Uncles       []*LegacyHeader
}

// LegacyHeader represents old format header
type LegacyHeader struct {
	ParentHash  common.Hash
	UncleHash   common.Hash
	Coinbase    common.Address
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
	Bloom       types.Bloom
	Difficulty  *big.Int
	Number      *big.Int
	GasLimit    uint64
	GasUsed     uint64
	Time        uint64
	Extra       []byte
	MixDigest   common.Hash
	Nonce       types.BlockNonce
	
	// SubnetEVM specific
	BlockGasCost *big.Int `rlp:"optional"`
}

// convertLegacyBlock converts legacy format to standard block
func (r *UnifiedReplayer) convertLegacyBlock(legacy *LegacyBlock) *types.Block {
	header := &types.Header{
		ParentHash:  legacy.Header.ParentHash,
		UncleHash:   legacy.Header.UncleHash,
		Coinbase:    legacy.Header.Coinbase,
		Root:        legacy.Header.Root,
		TxHash:      legacy.Header.TxHash,
		ReceiptHash: legacy.Header.ReceiptHash,
		Bloom:       legacy.Header.Bloom,
		Difficulty:  legacy.Header.Difficulty,
		Number:      legacy.Header.Number,
		GasLimit:    legacy.Header.GasLimit,
		GasUsed:     legacy.Header.GasUsed,
		Time:        legacy.Header.Time,
		Extra:       legacy.Header.Extra,
		MixDigest:   legacy.Header.MixDigest,
		Nonce:       legacy.Header.Nonce,
	}
	
	// Convert uncles
	uncles := make([]*types.Header, len(legacy.Uncles))
	for i, uncle := range legacy.Uncles {
		uncles[i] = &types.Header{
			ParentHash:  uncle.ParentHash,
			UncleHash:   uncle.UncleHash,
			Coinbase:    uncle.Coinbase,
			Root:        uncle.Root,
			TxHash:      uncle.TxHash,
			ReceiptHash: uncle.ReceiptHash,
			Bloom:       uncle.Bloom,
			Difficulty:  uncle.Difficulty,
			Number:      uncle.Number,
			GasLimit:    uncle.GasLimit,
			GasUsed:     uncle.GasUsed,
			Time:        uncle.Time,
			Extra:       uncle.Extra,
			MixDigest:   uncle.MixDigest,
			Nonce:       uncle.Nonce,
		}
	}
	
	body := &types.Body{
		Transactions: legacy.Transactions,
		Uncles:       uncles,
	}
	return types.NewBlock(header, body, nil, nil)
}

// convertToUnified converts standard block to unified format
func (r *UnifiedReplayer) convertToUnified(block *types.Block) *types.UnifiedBlock {
	return types.FromLegacyBlock(block)
}

// upgradeToQuantum upgrades block to quantum format
func (r *UnifiedReplayer) upgradeToQuantum(block *types.UnifiedBlock) {
	block.Header().UpgradeToQuantum(
		r.config.NetworkID,
		r.config.Layer,
		r.config.ConsensusType,
	)
	
	// Set additional quantum metadata
	if block.Header().Quantum != nil {
		// Add validator set hash for POA/POS
		if r.config.ConsensusType <= 1 {
			validatorHash := r.calculateValidatorSetHash(block.NumberU64())
			block.Header().Quantum.ValidatorSetHash = &validatorHash
		}
		
		// Add state accumulator
		stateAcc := r.calculateStateAccumulator(block)
		block.Header().Quantum.StateAccumulator = &stateAcc
	}
}

// processBatch writes a batch of blocks to the target database
func (r *UnifiedReplayer) processBatch(batch []*types.UnifiedBlock) error {
	// Begin database transaction
	dbBatch := r.config.TargetDB.NewBatch()
	
	for _, block := range batch {
		// Convert back to legacy for storage (until full migration)
		legacyBlock := block.ToLegacyBlock()
		
		// Write block to database
		rawdb.WriteBlock(dbBatch, legacyBlock)
		rawdb.WriteCanonicalHash(dbBatch, legacyBlock.Hash(), legacyBlock.NumberU64())
		rawdb.WriteHeadBlockHash(dbBatch, legacyBlock.Hash())
		
		// Store quantum metadata separately
		if block.Header().IsQuantum() {
			r.storeQuantumMetadata(dbBatch, block)
		}
	}
	
	// Commit batch
	if err := dbBatch.Write(); err != nil {
		return fmt.Errorf("failed to write batch: %w", err)
	}
	
	return nil
}

// storeQuantumMetadata stores quantum metadata in database
func (r *UnifiedReplayer) storeQuantumMetadata(batch ethdb.Batch, block *types.UnifiedBlock) {
	if block.Header().Quantum == nil {
		return
	}
	
	key := append([]byte("quantum-"), block.Hash().Bytes()...)
	data, err := rlp.EncodeToBytes(block.Header().Quantum)
	if err != nil {
		log.Error("Failed to encode quantum metadata", "err", err)
		return
	}
	
	batch.Put(key, data)
}

// calculateValidatorSetHash calculates validator set hash for a block
func (r *UnifiedReplayer) calculateValidatorSetHash(blockNum uint64) common.Hash {
	// For POA with single validator, use a deterministic hash
	if r.config.ConsensusType == 0 {
		// Single validator address (from config)
		validator := common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC")
		return common.BytesToHash(validator.Bytes())
	}
	
	// For POS, would calculate from actual validator set
	// This is a placeholder
	return common.BytesToHash(binary.BigEndian.AppendUint64(nil, blockNum))
}

// calculateStateAccumulator calculates state accumulator for a block
func (r *UnifiedReplayer) calculateStateAccumulator(block *types.UnifiedBlock) common.Hash {
	// Simplified accumulator: hash of state root + parent accumulator
	hasher := common.BytesToHash(block.Header().Root.Bytes())
	
	if block.NumberU64() > 0 && block.Header().Quantum != nil {
		// Get parent's accumulator
		parentKey := append([]byte("quantum-"), block.Header().ParentHash.Bytes()...)
		if data, err := r.config.TargetDB.Get(parentKey); err == nil {
			var parentQuantum types.QuantumMetadata
			if err := rlp.DecodeBytes(data, &parentQuantum); err == nil && parentQuantum.StateAccumulator != nil {
				// Combine with parent
				combined := append(hasher.Bytes(), parentQuantum.StateAccumulator.Bytes()...)
				hasher = common.BytesToHash(common.BytesToHash(combined).Bytes())
			}
		}
	}
	
	return hasher
}

// updateProgress updates replay progress
func (r *UnifiedReplayer) updateProgress(current uint64) {
	r.mu.Lock()
	r.currentBlock = current
	r.mu.Unlock()
	
	if r.config.OnProgress != nil {
		r.config.OnProgress(current-r.config.StartBlock+1, r.totalBlocks)
	}
	
	// Log progress every 1000 blocks or 10 seconds
	if current%1000 == 0 || time.Since(r.lastCheckpoint) > 10*time.Second {
		progress := float64(current-r.config.StartBlock+1) / float64(r.totalBlocks) * 100
		log.Info("Replay progress",
			"block", current,
			"progress", fmt.Sprintf("%.2f%%", progress),
			"errors", len(r.errors),
		)
		r.lastCheckpoint = time.Now()
	}
}

// recordError records a non-fatal error
func (r *UnifiedReplayer) recordError(err error) {
	r.mu.Lock()
	r.errors = append(r.errors, err)
	r.mu.Unlock()
	
	log.Warn("Replay error", "err", err)
}

// GetErrors returns all recorded errors
func (r *UnifiedReplayer) GetErrors() []error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	errors := make([]error, len(r.errors))
	copy(errors, r.errors)
	return errors
}

// makeBlockKey creates BadgerDB key for block number
func makeBlockKey(blockNum uint64) []byte {
	key := make([]byte, 9)
	key[0] = 'b' // prefix for blocks
	binary.BigEndian.PutUint64(key[1:], blockNum)
	return key
}

// ReplayResult contains replay statistics
type ReplayResult struct {
	TotalBlocks     uint64
	ProcessedBlocks uint64
	FailedBlocks    uint64
	UpgradedBlocks  uint64
	Duration        time.Duration
	Errors          []error
}

// FullReplay performs complete replay with statistics
func FullReplay(ctx context.Context, config *ReplayConfig) (*ReplayResult, error) {
	startTime := time.Now()
	
	replayer, err := NewUnifiedReplayer(config)
	if err != nil {
		return nil, err
	}
	defer replayer.Close()
	
	// Track statistics
	result := &ReplayResult{
		TotalBlocks: config.EndBlock - config.StartBlock + 1,
	}
	
	// Set progress callback
	config.OnProgress = func(current, total uint64) {
		result.ProcessedBlocks = current
		if config.UpgradeToQuantum {
			result.UpgradedBlocks = current
		}
	}
	
	// Run replay
	if err := replayer.ReplayBlocks(ctx); err != nil {
		return nil, err
	}
	
	// Collect results
	result.Errors = replayer.GetErrors()
	result.FailedBlocks = uint64(len(result.Errors))
	result.Duration = time.Since(startTime)
	
	return result, nil
}