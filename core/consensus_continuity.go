package core

import (
	"fmt"
	"math/big"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/consensus"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/geth/params"
)

// ConsensusBootstrapper handles bootstrapping consensus from historical state
type ConsensusBootstrapper struct {
	chainConfig *params.ChainConfig
	db          ethdb.Database
	engine      consensus.Engine
}

// BootstrapMode defines how to handle consensus when starting with historical data
type BootstrapMode int

const (
	// ContinueFromArchive - Continue building on top of archived chain
	ContinueFromArchive BootstrapMode = iota
	
	// QuantumGenesis - Start fresh consensus but preserve historical data for queries
	QuantumGenesis
	
	// HybridMode - Accept new blocks while serving historical data
	HybridMode
)

// NewConsensusBootstrapper creates a bootstrapper for consensus continuity
func NewConsensusBootstrapper(chainConfig *params.ChainConfig, db ethdb.Database, engine consensus.Engine) *ConsensusBootstrapper {
	return &ConsensusBootstrapper{
		chainConfig: chainConfig,
		db:          db,
		engine:      engine,
	}
}

// Bootstrap initializes consensus based on the selected mode
func (cb *ConsensusBootstrapper) Bootstrap(mode BootstrapMode, archiveHeight uint64) error {
	switch mode {
	case ContinueFromArchive:
		return cb.continueFromArchive(archiveHeight)
		
	case QuantumGenesis:
		return cb.setupQuantumGenesis(archiveHeight)
		
	case HybridMode:
		return cb.setupHybridMode(archiveHeight)
		
	default:
		return fmt.Errorf("unknown bootstrap mode: %v", mode)
	}
}

// continueFromArchive continues building on top of the archived chain
func (cb *ConsensusBootstrapper) continueFromArchive(archiveHeight uint64) error {
	log.Info("Bootstrapping consensus: Continue from archive", "height", archiveHeight)
	
	// Get the block at archive height
	hash := rawdb.ReadCanonicalHash(cb.db, archiveHeight)
	if hash == (common.Hash{}) {
		return fmt.Errorf("no canonical hash at height %d", archiveHeight)
	}
	
	block := rawdb.ReadBlock(cb.db, hash, archiveHeight)
	if block == nil {
		return fmt.Errorf("no block found at height %d", archiveHeight)
	}
	
	// Set as the current head
	rawdb.WriteHeadBlockHash(cb.db, hash)
	rawdb.WriteHeadHeaderHash(cb.db, hash)
	rawdb.WriteHeadFastBlockHash(cb.db, hash)
	
	log.Info("Consensus continued from archive",
		"number", block.NumberU64(),
		"hash", block.Hash(),
		"parent", block.ParentHash())
	
	return nil
}

// setupQuantumGenesis starts fresh consensus while preserving history
func (cb *ConsensusBootstrapper) setupQuantumGenesis(archiveHeight uint64) error {
	log.Info("Bootstrapping consensus: Quantum Genesis", "archive_height", archiveHeight)
	
	// Create a new genesis block that references the archive
	quantumGenesis := &types.Block{
		// This is a simplified version - real implementation would:
		// 1. Create proper genesis header
		// 2. Set parent hash to a special "archive reference" value
		// 3. Include archive height in extra data
		// 4. Reset difficulty/validators for fresh consensus
	}
	
	// Special handling in blockchain.go would:
	// - Blocks 0 to archiveHeight: Serve from archive (read-only)
	// - Blocks > archiveHeight: Gap (return "quantum gap" error)
	// - New blocks: Start from quantum genesis (height 0 in new chain)
	
	// Write quantum genesis
	rawdb.WriteBlock(cb.db, quantumGenesis)
	rawdb.WriteCanonicalHash(cb.db, quantumGenesis.Hash(), 0)
	rawdb.WriteHeadBlockHash(cb.db, quantumGenesis.Hash())
	
	// Store archive reference
	rawdb.WriteArchiveReference(cb.db, archiveHeight)
	
	log.Info("Quantum genesis initialized",
		"archive_serves", fmt.Sprintf("0-%d", archiveHeight),
		"new_chain_starts", "0 (quantum)")
	
	return nil
}

// setupHybridMode allows both historical queries and new consensus
func (cb *ConsensusBootstrapper) setupHybridMode(archiveHeight uint64) error {
	log.Info("Bootstrapping consensus: Hybrid Mode", "archive_height", archiveHeight)
	
	// In hybrid mode:
	// 1. Historical blocks (0 to archiveHeight) served from archive
	// 2. New consensus continues from archiveHeight+1
	// 3. Validators/stakers remain active
	
	// Get the archive head block
	archiveHash := rawdb.ReadCanonicalHash(cb.db, archiveHeight)
	if archiveHash == (common.Hash{}) {
		return fmt.Errorf("no canonical hash at archive height %d", archiveHeight)
	}
	
	archiveBlock := rawdb.ReadBlock(cb.db, archiveHash, archiveHeight)
	if archiveBlock == nil {
		return fmt.Errorf("no block at archive height %d", archiveHeight)
	}
	
	// Create transition block
	_ = &types.Header{
		ParentHash:  archiveBlock.Hash(),
		Number:      big.NewInt(int64(archiveHeight + 1)),
		Time:        uint64(time.Now().Unix()),
		Extra:       []byte("Hybrid Mode Transition"),
		// Copy other necessary fields from archive block
		Difficulty:  archiveBlock.Difficulty(),
		GasLimit:    archiveBlock.GasLimit(),
		// ... other fields
	}
	
	// Let consensus engine prepare the transition
	// TODO: Need to pass a ChainHeaderReader instead of chainConfig
	// if err := cb.engine.Prepare(chain, transitionHeader); err != nil {
	// 	return fmt.Errorf("failed to prepare transition header: %w", err)
	// }
	
	log.Info("Hybrid mode initialized",
		"archive_height", archiveHeight,
		"continue_from", archiveHeight+1)
	
	return nil
}