// Copyright 2025 Lux Industries, Inc.
// Standalone replay tool for importing blockchain data

package main

import (
	"context"
	"flag"
	"fmt"
	"math/big"
	"os"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/replay"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/leveldb"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/geth/params"
)

var (
	badgerDBPath   = flag.String("badgerdb", "/home/z/work/lux/state/chaindata/lux-mainnet-96369/db", "Path to BadgerDB source")
	targetDBPath   = flag.String("targetdb", "/home/z/.luxd/db/C", "Path to target database")
	startBlock     = flag.Uint64("start", 0, "Starting block number")
	endBlock       = flag.Uint64("end", 1000000, "Ending block number")
	batchSize      = flag.Int("batch", 1000, "Batch size for processing")
	networkID      = flag.Uint64("network", 96369, "Network ID")
	verbosity      = flag.Int("verbosity", 3, "Logging verbosity (0=silent, 1=error, 2=warn, 3=info, 4=debug)")
	validateBlocks = flag.Bool("validate", true, "Validate blocks during replay")
	dryRun         = flag.Bool("dry-run", false, "Perform a dry run without writing to database")
)

func main() {
	flag.Parse()

	// Setup logging
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.FromLegacyLevel(*verbosity), true)))

	log.Info("LUX Blockchain Replay Tool",
		"version", "1.0.0",
		"badgerdb", *badgerDBPath,
		"targetdb", *targetDBPath,
		"network", *networkID,
	)

	// Validate paths
	if _, err := os.Stat(*badgerDBPath); err != nil {
		log.Error("BadgerDB path does not exist", "path", *badgerDBPath, "error", err)
		os.Exit(1)
	}

	// Open or create target database
	targetDB, err := openDatabase(*targetDBPath)
	if err != nil {
		log.Error("Failed to open target database", "error", err)
		os.Exit(1)
	}
	defer targetDB.Close()

	// Get chain config
	chainConfig := getChainConfig(*networkID)

	// Create replay configuration
	replayConfig := &replay.ReplayConfig{
		BadgerDBPath:     *badgerDBPath,
		TargetDB:         targetDB,
		ChainConfig:      chainConfig,
		NetworkID:        *networkID,
		Layer:            0, // C-Chain
		ConsensusType:    0, // POA
		StartBlock:       *startBlock,
		EndBlock:         *endBlock,
		BatchSize:        *batchSize,
		UpgradeToQuantum: false,
	}

	// Add progress callback
	replayConfig.OnProgress = func(current, total uint64) {
		if current%100 == 0 || current == total {
			progress := float64(current) / float64(total) * 100
			log.Info("Replay progress",
				"current", current,
				"total", total,
				"progress", fmt.Sprintf("%.2f%%", progress),
			)
		}
	}

	// Perform replay
	ctx := context.Background()

	if *dryRun {
		log.Warn("DRY RUN MODE - No data will be written to database")
		// In dry run, use memory database
		replayConfig.TargetDB = rawdb.NewMemoryDatabase()
	}

	log.Info("Starting replay operation",
		"start_block", *startBlock,
		"end_block", *endBlock,
		"batch_size", *batchSize,
		"validate", *validateBlocks,
	)

	result, err := replay.FullReplay(ctx, replayConfig)
	if err != nil {
		log.Error("Replay failed", "error", err)
		os.Exit(1)
	}

	// Print results
	log.Info("Replay completed successfully",
		"total_blocks", result.TotalBlocks,
		"processed", result.ProcessedBlocks,
		"failed", result.FailedBlocks,
		"duration", result.Duration,
		"blocks_per_second", float64(result.ProcessedBlocks)/result.Duration.Seconds(),
	)

	if len(result.Errors) > 0 {
		log.Warn("Replay completed with errors", "error_count", len(result.Errors))
		for i, err := range result.Errors {
			if i < 10 { // Show first 10 errors
				log.Error("Replay error", "index", i, "error", err)
			}
		}
	}

	// Verify final state if not dry run
	if !*dryRun && *validateBlocks {
		if err := verifyDatabase(targetDB, result.ProcessedBlocks); err != nil {
			log.Error("Database verification failed", "error", err)
			os.Exit(1)
		}
		log.Info("Database verification completed successfully")
	}

	log.Info("Replay tool completed successfully")
}

// openDatabase opens or creates the target database
func openDatabase(path string) (ethdb.Database, error) {
	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open the LevelDB instance directly
	kvdb, err := leveldb.New(path, 256, 256, "replay/db/", false)
	if err != nil {
		return nil, fmt.Errorf("failed to open LevelDB: %w", err)
	}

	// Create database options for freezer support
	opts := rawdb.OpenOptions{
		ReadOnly: false,
	}

	// Open the database with freezer support
	return rawdb.Open(kvdb, opts)
}

// getChainConfig returns the chain configuration for the given network ID
func getChainConfig(networkID uint64) *params.ChainConfig {
	switch networkID {
	case 96369:
		// LUX Mainnet configuration
		return &params.ChainConfig{
			ChainID:             big.NewInt(96369),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			CancunTime:          nil,
			// POA configuration
			Clique: &params.CliqueConfig{
				Period: 1,
				Epoch:  30000,
			},
		}
	case 96368:
		// LUX Testnet configuration
		config := getChainConfig(96369)
		config.ChainID = big.NewInt(96368)
		return config
	default:
		// Default configuration
		return &params.ChainConfig{
			ChainID:             big.NewInt(int64(networkID)),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
		}
	}
}

// verifyDatabase performs basic verification of the replayed data
func verifyDatabase(db ethdb.Database, expectedBlocks uint64) error {
	// Check head block
	headHash := rawdb.ReadHeadBlockHash(db)
	if headHash == (common.Hash{}) {
		return fmt.Errorf("no head block found in database")
	}

	// Check head number
	headNumber, found := rawdb.ReadHeaderNumber(db, headHash)
	if !found {
		return fmt.Errorf("no head block number found")
	}

	log.Info("Database verification",
		"head_hash", headHash.Hex(),
		"head_number", headNumber,
		"expected_blocks", expectedBlocks,
	)

	// Check if we have at least some blocks
	if headNumber == 0 && expectedBlocks > 0 {
		return fmt.Errorf("no blocks were imported (head at genesis)")
	}

	// Sample check: verify some random blocks exist
	for i := uint64(0); i < min(10, headNumber); i++ {
		blockNum := i * (headNumber / 10)
		hash := rawdb.ReadCanonicalHash(db, blockNum)
		if hash == (common.Hash{}) {
			log.Warn("Missing canonical hash", "block", blockNum)
		} else {
			header := rawdb.ReadHeader(db, hash, blockNum)
			if header == nil {
				log.Warn("Missing header", "block", blockNum, "hash", hash.Hex())
			}
		}
	}

	return nil
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}