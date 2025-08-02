package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/luxfi/geth/ethdb/badgerdb"
	"github.com/luxfi/geth/ethdb/genesis"
)

func main() {
	// Command line flags
	var (
		genesisPath    = flag.String("genesis", "", "Path to genesis PebbleDB")
		archivePath    = flag.String("archive", "", "Path to archive BadgerDB")
		currentPath    = flag.String("current", "", "Path to current BadgerDB")
		batchSize      = flag.Int("batch-size", 1000, "Blocks per batch")
		verify         = flag.Bool("verify", false, "Verify blocks during replay")
		continueOnErr  = flag.Bool("continue-on-error", false, "Continue on non-fatal errors")
		targetHeight   = flag.Uint64("target-height", 0, "Target height to replay (0 = all)")
		finalityDelay  = flag.Uint64("finality-delay", 100, "Blocks before finality")
		skipArchive    = flag.Bool("skip-archive", false, "Skip archive creation")
	)
	flag.Parse()

	// Setup logging
	log.Root().SetHandler(log.StreamHandler(os.Stdout, log.TerminalFormat(true)))

	// Validate arguments
	if *genesisPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --genesis is required")
		flag.Usage()
		os.Exit(1)
	}
	if *currentPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --current is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Info("Quantum Genesis Replayer",
		"genesis", *genesisPath,
		"archive", *archivePath,
		"current", *currentPath,
		"target", *targetHeight,
		"finality", *finalityDelay)

	// Phase 1: Replay genesis to current BadgerDB
	replayConfig := genesis.ReplayConfig{
		GenesisPath:     *genesisPath,
		TargetPath:      *currentPath,
		BatchSize:       *batchSize,
		VerifyBlocks:    *verify,
		ContinueOnError: *continueOnErr,
		MaxHeight:       *targetHeight,
	}

	replayer, err := genesis.NewGenesisReplayer(replayConfig)
	if err != nil {
		log.Error("Failed to create replayer", "err", err)
		os.Exit(1)
	}

	start := time.Now()
	if err := replayer.Replay(); err != nil {
		log.Error("Replay failed", "err", err)
		os.Exit(1)
	}
	log.Info("Genesis replay completed", "duration", time.Since(start))
	
	// Close the replayer to ensure BadgerDB is properly closed
	if err := replayer.Close(); err != nil {
		log.Warn("Failed to close replayer", "err", err)
	}

	// Phase 2: Initialize dual database with archiving
	if !*skipArchive && *archivePath != "" {
		log.Info("Setting up dual database architecture...")
		
		// Create dual database config
		dualConfig := badgerdb.DualDatabaseConfig{
			ArchivePath:    *archivePath,
			CurrentPath:    *currentPath,
			FinalityHeight: *finalityDelay,
			ArchiveShared:  true, // Enable shared access for load balancing
		}

		dualDB, err := badgerdb.NewDualBadgerDatabase(dualConfig)
		if err != nil {
			log.Error("Failed to create dual database", "err", err)
			os.Exit(1)
		}
		defer dualDB.Close()

		// Run initial archiving to move finalized blocks
		log.Info("Running initial archiving...")
		archiver := badgerdb.NewArchiver(dualDB, *finalityDelay, 1*time.Minute)
		
		// Run archiving once
		if err := archiver.ArchiveFinalized(); err != nil {
			log.Error("Initial archiving failed", "err", err)
			os.Exit(1)
		}

		stats, _ := dualDB.Stat("")
		log.Info("Dual database initialized", "stats", stats)
	}

	log.Info("Quantum genesis import completed successfully!",
		"archive_enabled", !*skipArchive && *archivePath != "",
		"finality_delay", *finalityDelay)
}