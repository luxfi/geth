// Copyright 2025 Lux Industries, Inc.
// Replay integration for importing historical blockchain data

package eth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/replay"
	"github.com/luxfi/geth/log"
)

// ReplayConfig represents the replay configuration from file
type ReplayConfigFile struct {
	ReplayEnabled     bool   `json:"replay-enabled"`
	ReplaySourceDB    string `json:"replay-source-db"`
	ReplayDBType      string `json:"replay-db-type"`
	ReplayBatchSize   int    `json:"replay-batch-size"`
	ReplayAsync       bool   `json:"replay-async"`
	ReplayVerify      bool   `json:"replay-verify"`
	ReplayLogProgress bool   `json:"replay-log-progress"`
	ReplayLogInterval int    `json:"replay-log-interval"`
	ReplayStartBlock  uint64 `json:"replay-start-block,omitempty"`
	ReplayEndBlock    uint64 `json:"replay-end-block,omitempty"`
}

// LoadReplayConfig loads replay configuration from the C-Chain config file
func LoadReplayConfig(configPath string) (*ReplayConfigFile, error) {
	if configPath == "" {
		// Default path for C-Chain config
		configPath = filepath.Join("/home/z/.luxd/configs/chains/C/config.json")
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read replay config: %w", err)
	}

	var config ReplayConfigFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse replay config: %w", err)
	}

	return &config, nil
}

// InitiateReplay starts the replay process based on configuration
func (eth *Ethereum) InitiateReplay(replayConfigPath string) error {
	// Load replay configuration
	replayConf, err := LoadReplayConfig(replayConfigPath)
	if err != nil {
		log.Warn("Failed to load replay config, skipping replay", "error", err)
		return nil // Non-fatal, allow node to continue
	}

	if !replayConf.ReplayEnabled {
		log.Info("Replay is disabled in configuration")
		return nil
	}

	log.Info("Starting blockchain replay",
		"source", replayConf.ReplaySourceDB,
		"type", replayConf.ReplayDBType,
		"batch_size", replayConf.ReplayBatchSize,
	)

	// Create replay configuration
	config := &replay.ReplayConfig{
		BadgerDBPath:     replayConf.ReplaySourceDB,
		TargetDB:         eth.chainDb,
		ChainConfig:      eth.blockchain.Config(),
		NetworkID:        eth.networkID,
		Layer:            0, // C-Chain
		ConsensusType:    0, // POA
		StartBlock:       replayConf.ReplayStartBlock,
		EndBlock:         replayConf.ReplayEndBlock,
		BatchSize:        replayConf.ReplayBatchSize,
		UpgradeToQuantum: false,
	}

	// If end block not specified, try to determine from BadgerDB
	if config.EndBlock == 0 {
		config.EndBlock = 1000000 // Default to 1M blocks for now
	}

	// Add progress callback
	config.OnProgress = func(current, total uint64) {
		if replayConf.ReplayLogProgress && current%uint64(replayConf.ReplayLogInterval) == 0 {
			progress := float64(current) / float64(total) * 100
			log.Info("Replay progress", 
				"current", current, 
				"total", total, 
				"progress", fmt.Sprintf("%.2f%%", progress))
		}
	}

	// Run replay asynchronously if configured
	if replayConf.ReplayAsync {
		go func() {
			if err := eth.executeReplay(config); err != nil {
				log.Error("Replay failed", "error", err)
			}
		}()
	} else {
		return eth.executeReplay(config)
	}

	return nil
}

// executeReplay performs the actual replay operation
func (eth *Ethereum) executeReplay(config *replay.ReplayConfig) error {
	ctx := context.Background()

	// Perform the replay
	result, err := replay.FullReplay(ctx, config)
	if err != nil {
		return fmt.Errorf("replay execution failed: %w", err)
	}

	log.Info("Replay completed",
		"total_blocks", result.TotalBlocks,
		"processed", result.ProcessedBlocks,
		"failed", result.FailedBlocks,
		"duration", result.Duration,
	)

	if len(result.Errors) > 0 {
		log.Warn("Replay completed with errors", "error_count", len(result.Errors))
		for i, err := range result.Errors {
			if i < 10 { // Log first 10 errors
				log.Debug("Replay error", "index", i, "error", err)
			}
		}
	}

	// Force blockchain to reload after replay
	eth.blockchain.Stop()
	newBC, err := core.NewBlockChain(eth.chainDb, nil, eth.engine, &core.BlockChainConfig{
		TrieCleanLimit: eth.config.TrieCleanCache,
		TrieDirtyLimit: eth.config.TrieDirtyCache,
		ArchiveMode:    eth.config.NoPruning,
		TrieTimeLimit:  eth.config.TrieTimeout,
		SnapshotLimit:  eth.config.SnapshotCache,
		Preimages:      eth.config.Preimages,
		StateHistory:   eth.config.StateHistory,
	})
	if err != nil {
		return fmt.Errorf("failed to reload blockchain after replay: %w", err)
	}
	eth.blockchain = newBC

	return nil
}

// ReplayAPI provides RPC methods for replay control
type ReplayAPI struct {
	eth *Ethereum
}

// NewReplayAPI creates a new replay API instance
func NewReplayAPI(eth *Ethereum) *ReplayAPI {
	return &ReplayAPI{eth: eth}
}

// Status returns the current replay status
func (api *ReplayAPI) Status() map[string]interface{} {
	currentBlock := api.eth.blockchain.CurrentBlock()
	return map[string]interface{}{
		"enabled":       true,
		"currentBlock":  currentBlock.Number.Uint64(),
		"currentHash":   currentBlock.Hash().Hex(),
		"stateRoot":     currentBlock.Root.Hex(),
	}
}

// TriggerReplay manually triggers a replay operation
func (api *ReplayAPI) TriggerReplay(sourceDB string, startBlock, endBlock uint64) (string, error) {
	config := &replay.ReplayConfig{
		BadgerDBPath:     sourceDB,
		TargetDB:         api.eth.chainDb,
		ChainConfig:      api.eth.blockchain.Config(),
		NetworkID:        api.eth.networkID,
		Layer:            0, // C-Chain
		ConsensusType:    0, // POA
		StartBlock:       startBlock,
		EndBlock:         endBlock,
		BatchSize:        1000,
		UpgradeToQuantum: false,
	}

	// Run replay asynchronously
	go func() {
		if err := api.eth.executeReplay(config); err != nil {
			log.Error("Manual replay failed", "error", err)
		}
	}()

	return "Replay started", nil
}