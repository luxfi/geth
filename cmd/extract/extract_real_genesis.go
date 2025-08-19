//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/node"
)

func main() {
	// Open the BadgerDB database
	dbPath := "/home/z/work/lux/test-migration/dest-badgerdb"
	
	// Create database config
	dbConfig := &node.Config{
		DataDir: dbPath,
		DBEngine: "badgerdb",
	}
	
	// Open database
	db, err := node.OpenDatabase(dbConfig, "chaindata", "", false)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Read the actual genesis from the database
	genesis, err := core.ReadGenesis(db)
	if err != nil {
		log.Fatal("Failed to read genesis from database:", err)
	}

	fmt.Printf("Successfully read genesis from database\n")
	fmt.Printf("Chain ID: %v\n", genesis.Config.ChainID)
	fmt.Printf("Timestamp: 0x%x\n", genesis.Timestamp)
	fmt.Printf("Difficulty: %v\n", genesis.Difficulty)
	fmt.Printf("Gas Limit: 0x%x\n", genesis.GasLimit)
	fmt.Printf("Number of allocations: %d\n", len(genesis.Alloc))

	// Convert to C-Chain genesis format
	cChainGenesis := map[string]interface{}{
		"config": genesis.Config,
		"nonce": fmt.Sprintf("0x%x", genesis.Nonce),
		"timestamp": fmt.Sprintf("0x%x", genesis.Timestamp),
		"extraData": fmt.Sprintf("0x%x", genesis.ExtraData),
		"gasLimit": fmt.Sprintf("0x%x", genesis.GasLimit),
		"difficulty": fmt.Sprintf("0x%x", genesis.Difficulty),
		"mixHash": genesis.Mixhash.Hex(),
		"coinbase": genesis.Coinbase.Hex(),
		"number": "0x0",
		"gasUsed": "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"alloc": genesis.Alloc,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(cChainGenesis, "", "  ")
	if err != nil {
		log.Fatal("Failed to marshal genesis:", err)
	}

	// Save the extracted genesis
	outputFile := "/home/z/work/lux/extracted_real_genesis.json"
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		log.Fatal("Failed to write genesis file:", err)
	}

	fmt.Printf("\nExtracted genesis saved to %s\n", outputFile)
	fmt.Printf("Genesis hash: %s\n", genesis.ToBlock().Hash().Hex())
}