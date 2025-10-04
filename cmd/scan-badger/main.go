package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v3"
)

func main() {
	dbPath := flag.String("db", "/home/z/work/lux/state/chaindata/lux-mainnet-96369/db", "BadgerDB path")
	limit := flag.Int("limit", 100, "Number of keys to scan")
	flag.Parse()

	opts := badger.DefaultOptions(*dbPath)
	opts.ReadOnly = true
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	count := 0
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		fmt.Println("Scanning BadgerDB keys...")
		fmt.Println("=========================")

		for it.Rewind(); it.Valid(); it.Next() {
			if count >= *limit {
				break
			}
			item := it.Item()
			key := item.Key()

			// Get value size
			valueSize := item.ValueSize()

			// Try to get a sample of the value
			var valueSample []byte
			err := item.Value(func(val []byte) error {
				if len(val) > 32 {
					valueSample = val[:32]
				} else {
					valueSample = val
				}
				return nil
			})
			if err != nil {
				valueSample = []byte("error reading")
			}

			fmt.Printf("Key %d:\n", count)
			fmt.Printf("  Raw: %s\n", hex.EncodeToString(key))
			fmt.Printf("  ASCII: %q\n", string(key))
			fmt.Printf("  Length: %d bytes\n", len(key))
			fmt.Printf("  Value size: %d bytes\n", valueSize)
			if len(valueSample) > 0 {
				fmt.Printf("  Value sample: %s...\n", hex.EncodeToString(valueSample))
			}
			fmt.Println()

			count++
		}

		fmt.Printf("Total keys scanned: %d\n", count)
		return nil
	})

	if err != nil {
		log.Fatal("Error scanning database:", err)
	}
}