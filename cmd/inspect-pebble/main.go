package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	var dbPath string
	var limit int
	flag.StringVar(&dbPath, "db", "", "Path to PebbleDB")
	flag.IntVar(&limit, "limit", 50, "Number of keys to show")
	flag.Parse()

	if dbPath == "" {
		fmt.Println("Usage: inspect-pebble -db <path>")
		os.Exit(1)
	}

	// Open PebbleDB in read-only mode
	opts := &pebble.Options{
		ReadOnly: true,
	}

	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Create iterator
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	defer iter.Close()

	fmt.Printf("First %d keys in %s:\n\n", limit, dbPath)

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		key := iter.Key()
		val, err := iter.ValueAndErr()
		if err != nil {
			fmt.Printf("Key %d: %s (error reading value: %v)\n", count, hex.EncodeToString(key), err)
		} else {
			keyStr := hex.EncodeToString(key)
			valLen := len(val)
			
			// Try to identify key type
			keyType := "unknown"
			if len(key) > 0 {
				switch key[0] {
				case 'h':
					keyType = "header/hash"
				case 'b':
					keyType = "body"
				case 'r':
					keyType = "receipts"
				case 't':
					keyType = "total difficulty"
				case 'H':
					keyType = "head"
				case 'l':
					keyType = "log"
				case 'n':
					keyType = "number"
				}
			}
			
			fmt.Printf("Key %d: %s (type: %s, val_len: %d)\n", count, keyStr, keyType, valLen)
			
			// Show ASCII representation if it looks like text
			if len(key) < 32 {
				fmt.Printf("  ASCII: %q\n", key)
			}
		}
		count++
	}

	if err := iter.Error(); err != nil {
		log.Fatal("Iterator error:", err)
	}

	// Get database stats
	metrics := db.Metrics()
	fmt.Printf("\nDatabase metrics:\n%s\n", metrics.String())
}