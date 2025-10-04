package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

func main() {
	pebblePath := flag.String("pebble", "/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb", "PebbleDB path")
	targetPath := flag.String("target", "/home/z/.luxd/db/C", "Target database path")
	startBlock := flag.Uint64("start", 0, "Start block")
	endBlock := flag.Uint64("end", 10, "End block")
	dryRun := flag.Bool("dry-run", false, "Dry run mode")
	scanOnly := flag.Bool("scan", false, "Only scan and show database structure")
	flag.Parse()

	// Open PebbleDB
	opts := &pebble.Options{}
	pdb, err := pebble.Open(*pebblePath, opts)
	if err != nil {
		log.Fatal("Failed to open PebbleDB:", err)
	}
	defer pdb.Close()

	if *scanOnly {
		scanDatabase(pdb)
		return
	}

	// Open target database if not dry run
	var targetDB *pebble.DB
	if !*dryRun {
		targetOpts := &pebble.Options{}
		targetDB, err = pebble.Open(*targetPath, targetOpts)
		if err != nil {
			log.Fatal("Failed to open target database:", err)
		}
		defer targetDB.Close()
	}

	fmt.Printf("Starting PebbleDB extraction\n")
	fmt.Printf("Source: %s\n", *pebblePath)
	fmt.Printf("Target: %s\n", *targetPath)
	fmt.Printf("Blocks: %d to %d\n", *startBlock, *endBlock)
	fmt.Printf("Dry run: %v\n\n", *dryRun)

	// Extract and migrate blocks
	fmt.Println("Starting block extraction...")

	// For now, just scan and understand the key structure
	// We'll implement actual extraction once we understand the format
}

func scanDatabase(pdb *pebble.DB) {
	fmt.Println("Scanning PebbleDB structure...")
	fmt.Println("================================")

	iter, _ := pdb.NewIter(nil)
	defer iter.Close()

	count := 0
	patterns := make(map[string]int)

	for iter.First(); iter.Valid() && count < 1000; iter.Next() {
		key := iter.Key()
		value, _ := iter.ValueAndErr()

		// Analyze key patterns
		if len(key) > 0 {
			prefix := hex.EncodeToString(key[:min(4, len(key))])
			patterns[prefix]++
		}

		// Show first few entries
		if count < 20 {
			fmt.Printf("Key %d:\n", count)
			fmt.Printf("  Hex: %s\n", hex.EncodeToString(key))
			fmt.Printf("  Len: %d\n", len(key))
			if len(value) > 0 {
				fmt.Printf("  Val len: %d\n", len(value))
				// Try to decode as header
				if len(key) > 8 && key[0] == 'h' {
					var header types.Header
					if err := rlp.DecodeBytes(value, &header); err == nil {
						fmt.Printf("  Decoded as header: Block #%s\n", header.Number.String())
					}
				}
				// Try to decode as body
				if len(key) > 8 && key[0] == 'b' {
					var body types.Body
					if err := rlp.DecodeBytes(value, &body); err == nil {
						fmt.Printf("  Decoded as body: %d txs\n", len(body.Transactions))
					}
				}
			}
			fmt.Println()
		}
		count++
	}

	fmt.Printf("\nKey patterns found (first 4 bytes):\n")
	for pattern, c := range patterns {
		if c > 10 {
			fmt.Printf("  %s: %d occurrences\n", pattern, c)
		}
	}
	fmt.Printf("\nTotal keys scanned: %d\n", count)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}