// export_subnetevm_blocks exports blocks from SubnetEVM PebbleDB to RLP format
package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/rlp"
)

// RLPBlockEntry is the RLP output format for each block
type RLPBlockEntry struct {
	Height   uint64
	Hash     []byte // 32 bytes
	Header   []byte // raw RLP
	Body     []byte // raw RLP
	Receipts []byte // raw RLP
}

func main() {
	dbPath := flag.String("db", "", "Path to SubnetEVM PebbleDB")
	namespaceHex := flag.String("namespace", "", "32-byte namespace in hex")
	output := flag.String("output", "", "Output RLP file (or - for stdout)")
	start := flag.Uint64("start", 0, "Start block number")
	end := flag.Uint64("end", 0, "End block number (0 = to tip)")
	flag.Parse()

	if *dbPath == "" || *namespaceHex == "" {
		fmt.Fprintln(os.Stderr, "Usage: export_subnetevm_blocks -db <path> -namespace <hex> -output <file>")
		flag.PrintDefaults()
		os.Exit(1)
	}

	namespace, err := hex.DecodeString(*namespaceHex)
	if err != nil || len(namespace) != 32 {
		fmt.Fprintf(os.Stderr, "Invalid namespace: %v\n", err)
		os.Exit(1)
	}

	// Open database
	db, err := pebble.Open(*dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get tip height - try AcceptorTipHeightKey, fall back to user-specified end
	tipHeight := uint64(1082780) // Default for Lux mainnet
	tipHeightKey := append(namespace, []byte("AcceptorTipHeightKey")...)
	tipHeightBytes, closer, err := db.Get(tipHeightKey)
	if err == nil {
		tipHeight = binary.BigEndian.Uint64(tipHeightBytes)
		closer.Close()
	} else {
		fmt.Fprintf(os.Stderr, "Note: AcceptorTipHeightKey not found, using default tip %d\n", tipHeight)
	}

	if *end == 0 {
		*end = tipHeight
	}

	fmt.Fprintf(os.Stderr, "Exporting blocks %d to %d (tip: %d)\n", *start, *end, tipHeight)

	// Open output file
	var out *os.File
	if *output == "" || *output == "-" {
		out = os.Stdout
	} else {
		out, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()
	}

	exported := 0
	errors := 0

	for height := *start; height <= *end; height++ {
		entry, err := exportBlock(db, namespace, height)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting block %d: %v\n", height, err)
			errors++
			continue
		}

		// Write RLP-encoded block entry
		encoded, err := rlp.EncodeToBytes(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding block %d: %v\n", height, err)
			errors++
			continue
		}

		if _, err := out.Write(encoded); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing block %d: %v\n", height, err)
			errors++
			continue
		}

		exported++
		if exported%10000 == 0 {
			fmt.Fprintf(os.Stderr, "Exported %d blocks (current: %d, errors: %d)\n", exported, height, errors)
		}
	}

	fmt.Fprintf(os.Stderr, "Done. Exported %d blocks, %d errors.\n", exported, errors)
}

func exportBlock(db *pebble.DB, namespace []byte, height uint64) (*RLPBlockEntry, error) {
	// SubnetEVM stores headers with key: namespace || 'h' || be8(number) || hash
	// We need to scan for headers at this height
	prefix := makeKey(namespace, 'h', height, nil)

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: makeKey(namespace, 'h', height+1, nil),
	})
	if err != nil {
		return nil, fmt.Errorf("create iterator: %w", err)
	}
	defer iter.Close()

	if !iter.First() {
		return nil, fmt.Errorf("no header found at height %d", height)
	}

	// Extract hash from key
	// Key format: namespace(32) || 'h'(1) || number(8) || hash(32) = 73 bytes total
	key := iter.Key()
	expectedLen := 32 + 1 + 8 + 32 // 73 bytes
	if len(key) != expectedLen {
		return nil, fmt.Errorf("unexpected key length: got %d, want %d", len(key), expectedLen)
	}

	hashOffset := 32 + 1 + 8 // namespace + prefix + number
	blockHash := make([]byte, 32)
	copy(blockHash, key[hashOffset:hashOffset+32])

	// Get header RLP (raw, no decoding/verification)
	headerRLP := make([]byte, len(iter.Value()))
	copy(headerRLP, iter.Value())

	// Get body
	bodyKey := makeKey(namespace, 'b', height, blockHash)
	bodyRLP, closer, err := db.Get(bodyKey)
	if err != nil {
		// Empty body for genesis or blocks without transactions
		bodyRLP = []byte{0xc2, 0xc0, 0xc0} // RLP for empty body [[], []]
	} else {
		bodyCopy := make([]byte, len(bodyRLP))
		copy(bodyCopy, bodyRLP)
		closer.Close()
		bodyRLP = bodyCopy
	}

	// Get receipts
	receiptsKey := makeKey(namespace, 'r', height, blockHash)
	receiptsRLP, closer, err := db.Get(receiptsKey)
	if err != nil {
		// Empty receipts
		receiptsRLP = []byte{0xc0} // RLP for empty list []
	} else {
		receiptsCopy := make([]byte, len(receiptsRLP))
		copy(receiptsCopy, receiptsRLP)
		closer.Close()
		receiptsRLP = receiptsCopy
	}

	return &RLPBlockEntry{
		Height:   height,
		Hash:     blockHash,
		Header:   headerRLP,
		Body:     bodyRLP,
		Receipts: receiptsRLP,
	}, nil
}

func makeKey(namespace []byte, prefix byte, number uint64, hash []byte) []byte {
	size := 32 + 1 + 8 // namespace + prefix + number
	if hash != nil {
		size += 32
	}
	key := make([]byte, 0, size)
	key = append(key, namespace...)
	key = append(key, prefix)

	var numBytes [8]byte
	binary.BigEndian.PutUint64(numBytes[:], number)
	key = append(key, numBytes[:]...)

	if hash != nil {
		key = append(key, hash...)
	}
	return key
}
