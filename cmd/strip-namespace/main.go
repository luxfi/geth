package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/pebble"
	badger "github.com/dgraph-io/badger/v4"
	"github.com/golang/snappy"
)

const (
	// SubnetEVM namespace (32 bytes)
	namespaceSize = 32
)

var (
	sourceDir   = flag.String("source", "", "Source PebbleDB directory (required)")
	targetDir   = flag.String("target", "", "Target BadgerDB directory (required)")
	maxBlocks   = flag.Int("blocks", 1000, "Maximum number of blocks to process (0 for all)")
	dryRun      = flag.Bool("dry-run", false, "Dry run - don't write to target")
	verbose     = flag.Bool("verbose", false, "Verbose output")
	verifyOnly  = flag.Bool("verify", false, "Only verify namespace presence, don't convert")
)

// Key prefixes used in the database
var (
	headerPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	headerNumberPrefix = []byte("H") // headerNumberPrefix + hash -> num (uint64 big endian)
	blockBodyPrefix    = []byte("b") // blockBodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	headerHashSuffix   = []byte("n") // headerPrefix + num (uint64 big endian) + headerHashSuffix -> hash
	lastHeaderKey      = []byte("LastHeader")
	lastBlockKey       = []byte("LastBlock")
	lastFinalizedKey   = []byte("LastFinalized")
	trieNodePrefix     = []byte("t") // trieNodePrefix + hash -> trie node
)

type Stats struct {
	TotalKeys        int64
	ProcessedKeys    int64
	SkippedKeys      int64
	HeaderKeys       int64
	BodyKeys         int64
	ReceiptKeys      int64
	TrieKeys         int64
	CanonicalKeys    int64
	OtherKeys        int64
	BytesProcessed   int64
	StartTime        time.Time
}

func (s *Stats) Print() {
	elapsed := time.Since(s.StartTime)
	fmt.Printf("\n=== Conversion Statistics ===\n")
	fmt.Printf("Total keys examined: %d\n", s.TotalKeys)
	fmt.Printf("Keys processed: %d\n", s.ProcessedKeys)
	fmt.Printf("Keys skipped: %d\n", s.SkippedKeys)
	fmt.Printf("  Headers: %d\n", s.HeaderKeys)
	fmt.Printf("  Bodies: %d\n", s.BodyKeys)
	fmt.Printf("  Receipts: %d\n", s.ReceiptKeys)
	fmt.Printf("  Canonical: %d\n", s.CanonicalKeys)
	fmt.Printf("  Trie nodes: %d\n", s.TrieKeys)
	fmt.Printf("  Other: %d\n", s.OtherKeys)
	fmt.Printf("Bytes processed: %.2f MB\n", float64(s.BytesProcessed)/(1024*1024))
	fmt.Printf("Time elapsed: %v\n", elapsed)
	if elapsed.Seconds() > 0 {
		fmt.Printf("Keys per second: %.0f\n", float64(s.TotalKeys)/elapsed.Seconds())
	}
}

func main() {
	flag.Parse()

	if *sourceDir == "" {
		flag.Usage()
		log.Fatal("-source directory is required")
	}

	// Ensure source exists
	if _, err := os.Stat(*sourceDir); err != nil {
		log.Fatalf("Source directory does not exist: %v", err)
	}

	if *verifyOnly {
		verifyNamespace()
		return
	}

	// For conversion, target is required
	if *targetDir == "" {
		flag.Usage()
		log.Fatal("-target directory is required for conversion")
	}

	// Ensure target directory exists (create parent if needed)
	if err := os.MkdirAll(filepath.Dir(*targetDir), 0755); err != nil {
		log.Fatalf("Failed to create target parent directory: %v", err)
	}

	convertDatabase()
}

func verifyNamespace() {
	fmt.Printf("Verifying namespace in source database: %s\n", *sourceDir)

	// Open source PebbleDB
	srcDB, err := pebble.Open(*sourceDir, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source PebbleDB: %v", err)
	}
	defer srcDB.Close()

	// Check first few keys
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	hasNamespace := 0
	noNamespace := 0

	fmt.Println("\nFirst 10 keys (showing hex):")
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		keyHex := hex.EncodeToString(key)

		if len(key) > namespaceSize {
			// Check if first 32 bytes look like a namespace
			namespace := key[:namespaceSize]
			nsHex := hex.EncodeToString(namespace)

			// The expected namespace based on your analysis
			expectedNS := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"

			if nsHex == expectedNS {
				hasNamespace++
				fmt.Printf("  [NS] Key: %s... (len=%d)\n", keyHex[:80], len(key))
				if *verbose {
					fmt.Printf("       Namespace: %s\n", nsHex)
					fmt.Printf("       After NS: %s\n", hex.EncodeToString(key[namespaceSize:]))
				}
			} else {
				noNamespace++
				fmt.Printf("  [??] Key: %s... (len=%d)\n", keyHex[:min(80, len(keyHex))], len(key))
			}
		} else {
			noNamespace++
			fmt.Printf("  [NO] Key: %s (len=%d)\n", keyHex, len(key))
		}
		count++
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	fmt.Printf("\nResults: %d keys with expected namespace, %d without\n", hasNamespace, noNamespace)
}

func convertDatabase() {
	stats := &Stats{StartTime: time.Now()}

	fmt.Printf("Converting database from SubnetEVM to C-Chain format\n")
	fmt.Printf("Source (PebbleDB): %s\n", *sourceDir)
	fmt.Printf("Target (BadgerDB): %s\n", *targetDir)
	if *dryRun {
		fmt.Println("DRY RUN - no changes will be made")
	}
	fmt.Println()

	// Open source PebbleDB
	srcDB, err := pebble.Open(*sourceDir, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source PebbleDB: %v", err)
	}
	defer srcDB.Close()

	var tgtDB *badger.DB
	if !*dryRun {
		// Open target BadgerDB
		tgtDB, err = badger.Open(badger.DefaultOptions(*targetDir))
		if err != nil {
			log.Fatalf("Failed to open target BadgerDB: %v", err)
		}
		defer tgtDB.Close()
	}

	// Process all keys
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	batch := make(map[string][]byte)
	batchSize := 0
	maxBatchSize := 10000 // Write in batches

	blocksProcessed := make(map[uint64]bool) // Track unique block numbers

	for iter.First(); iter.Valid(); iter.Next() {
		stats.TotalKeys++

		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())

		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())

		stats.BytesProcessed += int64(len(key) + len(value))

		// Skip keys that are too short to have namespace
		if len(key) < namespaceSize {
			stats.SkippedKeys++
			if *verbose {
				fmt.Printf("Skipping short key: %x\n", key)
			}
			continue
		}

		// Extract and verify namespace
		namespace := key[:namespaceSize]
		expectedNS, _ := hex.DecodeString("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")

		if !bytes.Equal(namespace, expectedNS) {
			stats.SkippedKeys++
			if *verbose {
				fmt.Printf("Skipping key with unexpected namespace: %x\n", namespace)
			}
			continue
		}

		// Strip namespace
		strippedKey := key[namespaceSize:]

		// Categorize and potentially transform the key
		transformedKey := transformKey(strippedKey, stats)

		if transformedKey != nil {
			stats.ProcessedKeys++

			if *verbose && stats.ProcessedKeys <= 10 {
				fmt.Printf("Processing: %x -> %x\n", key[:min(64, len(key))], transformedKey[:min(32, len(transformedKey))])
			}

			if !*dryRun {
				batch[string(transformedKey)] = value
				batchSize++

				// Write batch if it's full
				if batchSize >= maxBatchSize {
					writeBatch(tgtDB, batch)
					batch = make(map[string][]byte)
					batchSize = 0
				}
			}

			// Check if we've processed enough blocks
			if blockNum := getBlockNumber(strippedKey); blockNum != nil {
				blocksProcessed[*blockNum] = true
				if *maxBlocks > 0 && len(blocksProcessed) >= *maxBlocks {
					fmt.Printf("Reached block limit (%d blocks)\n", *maxBlocks)
					break
				}
			}
		}

		// Show progress
		if stats.TotalKeys%10000 == 0 {
			fmt.Printf("Progress: %d keys examined, %d processed\n", stats.TotalKeys, stats.ProcessedKeys)
		}
	}

	// Write remaining batch
	if !*dryRun && batchSize > 0 {
		writeBatch(tgtDB, batch)
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	stats.Print()
}

func transformKey(key []byte, stats *Stats) []byte {
	if len(key) == 0 {
		return nil
	}

	// Identify key type and transform accordingly
	switch key[0] {
	case 'h':
		// Header-related keys
		if len(key) > 1 && key[len(key)-1] == 'n' {
			// h[num]n -> h[num]n (header hash by number)
			stats.HeaderKeys++
			return key
		} else if len(key) == 1+8+32 {
			// h[num][hash] -> h[num][hash] (header by number and hash)
			stats.HeaderKeys++
			return key
		}

	case 'H':
		// H[hash] -> H[hash] (canonical header number)
		stats.CanonicalKeys++
		return key

	case 'b':
		// b[num][hash] -> b[num][hash] (block body)
		stats.BodyKeys++
		return key

	case 'r':
		// r[num][hash] -> r[num][hash] (receipts)
		stats.ReceiptKeys++
		return key

	case 't':
		// t[hash] -> t[hash] (trie node)
		stats.TrieKeys++
		return key

	case 'L':
		// Last* keys
		if bytes.Equal(key, lastHeaderKey) || bytes.Equal(key, lastBlockKey) || bytes.Equal(key, lastFinalizedKey) {
			stats.OtherKeys++
			return key
		}
	}

	// Handle other keys
	stats.OtherKeys++
	return key
}

func getBlockNumber(key []byte) *uint64 {
	if len(key) == 0 {
		return nil
	}
	// Check if this is a header key with block number (h + 8 bytes + hash/suffix)
	if key[0] == 'h' && len(key) >= 9 {
		num := binary.BigEndian.Uint64(key[1:9])
		return &num
	}
	// Check if this is a body key with block number (b + 8 bytes + hash)
	if key[0] == 'b' && len(key) >= 9 {
		num := binary.BigEndian.Uint64(key[1:9])
		return &num
	}
	return nil
}

func writeBatch(db *badger.DB, batch map[string][]byte) {
	txn := db.NewTransaction(true)
	defer txn.Discard()

	for k, v := range batch {
		// Decompress value if needed
		decompressed, err := snappy.Decode(nil, v)
		if err == nil {
			v = decompressed
		}

		if err := txn.Set([]byte(k), v); err != nil {
			if err == badger.ErrTxnTooBig {
				// Commit current transaction and start new one
				if err := txn.Commit(); err != nil {
					log.Printf("Failed to commit transaction: %v", err)
				}
				txn = db.NewTransaction(true)
				// Retry the failed key
				if err := txn.Set([]byte(k), v); err != nil {
					log.Printf("Failed to set key %x: %v", k, err)
				}
			} else {
				log.Printf("Failed to set key %x: %v", k, err)
			}
		}
	}

	if err := txn.Commit(); err != nil {
		log.Printf("Failed to commit final transaction: %v", err)
	}
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}