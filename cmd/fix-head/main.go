package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
	"github.com/luxfi/geth/common"
)

var (
	dbPath = flag.String("db", "", "BadgerDB path (required)")
)

func main() {
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("--db flag is required")
	}

	fmt.Printf("Opening database: %s\n", *dbPath)
	
	opts := badger.DefaultOptions(*dbPath)
	opts.ReadOnly = false
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf("Failed to open BadgerDB: %v", err)
	}
	defer db.Close()

	// Read the tip block number and hash
	var tipNumber uint64 = 1082780
	var tipHash common.Hash

	// Read canonical hash for tip block
	err = db.View(func(txn *badger.Txn) error {
		// Canonical hash key: 'h' + number + 'n'
		key := make([]byte, 1+8+1)
		key[0] = 'h'
		binary.BigEndian.PutUint64(key[1:9], tipNumber)
		key[9] = 'n'

		item, err := txn.Get(key)
		if err != nil {
			return fmt.Errorf("failed to get canonical hash for block %d: %v", tipNumber, err)
		}

		return item.Value(func(val []byte) error {
			copy(tipHash[:], val)
			return nil
		})
	})

	if err != nil {
		log.Fatalf("Error reading tip hash: %v", err)
	}

	fmt.Printf("Tip block: %d\n", tipNumber)
	fmt.Printf("Tip hash: %s\n", tipHash.Hex())

	// Write head pointers
	err = db.Update(func(txn *badger.Txn) error {
		// Write LastHeader
		lastHeaderKey := []byte("LastHeader")
		fmt.Printf("Writing LastHeader key\n")
		if err := txn.Set(lastHeaderKey, tipHash.Bytes()); err != nil {
			return fmt.Errorf("failed to write LastHeader: %v", err)
		}

		// Write LastBlock
		lastBlockKey := []byte("LastBlock")
		fmt.Printf("Writing LastBlock key\n")
		if err := txn.Set(lastBlockKey, tipHash.Bytes()); err != nil {
			return fmt.Errorf("failed to write LastBlock: %v", err)
		}

		// Write headHeaderKey (format: 'H' for header)
		headHeaderKey := []byte("LastHeader")
		fmt.Printf("Writing headHeaderKey\n")
		if err := txn.Set(headHeaderKey, tipHash.Bytes()); err != nil {
			return fmt.Errorf("failed to write headHeaderKey: %v", err)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Error writing head pointers: %v", err)
	}

	fmt.Println("\n✅ Successfully wrote all head pointers!")
	fmt.Printf("   Tip block: %d\n", tipNumber)
	fmt.Printf("   Tip hash: %s\n", tipHash.Hex())
}
