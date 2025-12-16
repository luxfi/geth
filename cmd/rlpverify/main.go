// Copyright 2025 Lux Industries Inc
// RLP encoding verification tool for header round-trip testing.
// Compares raw header RLP bytes with re-encoded bytes to detect encoding differences.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/luxfi/crypto"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: rlpverify <blocks.rlp>")
		fmt.Println("Verifies RLP encoding round-trip for block headers")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Read %d bytes from %s\n", len(data), os.Args[1])

	// Process blocks from the concatenated RLP stream
	remaining := data
	blockNum := 0
	mismatches := 0

	for len(remaining) > 0 {
		// Get next RLP item (block) - returns content and rest
		_, content, rest, err := rlp.Split(remaining)
		if err != nil {
			fmt.Fprintf(os.Stderr, "split at block %d: %v\n", blockNum, err)
			break
		}

		// Block data is the original bytes minus the rest
		blockData := remaining[:len(remaining)-len(rest)]
		remaining = rest

		// Extract header from block
		headerRaw, err := extractHeaderRLP(blockData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "block %d: extract header: %v\n", blockNum, err)
			blockNum++
			continue
		}

		// Compute hash of raw bytes
		rawHash := keccak256(headerRaw)

		// Decode header
		header, err := types.DecodeHeader(headerRaw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "block %d: decode header: %v\n", blockNum, err)
			blockNum++
			continue
		}

		// Re-encode header
		var buf bytes.Buffer
		if err := header.EncodeRLP(&buf); err != nil {
			fmt.Fprintf(os.Stderr, "block %d: encode header: %v\n", blockNum, err)
			blockNum++
			continue
		}
		reencoded := buf.Bytes()

		// Compute hash of re-encoded bytes
		reencodedHash := keccak256(reencoded)

		// Compare
		if rawHash != reencodedHash {
			mismatches++
			fmt.Printf("\n=== MISMATCH block %d (height %s) ===\n", blockNum, header.Number)
			fmt.Printf("Raw hash:       %s\n", rawHash.Hex())
			fmt.Printf("Re-encoded hash: %s\n", reencodedHash.Hex())
			fmt.Printf("Raw length:      %d\n", len(headerRaw))
			fmt.Printf("Re-encoded len:  %d\n", len(reencoded))

			// Count fields
			rawFields, _ := countRLPFields(headerRaw)
			reFields, _ := countRLPFields(reencoded)
			fmt.Printf("Raw fields:      %d\n", rawFields)
			fmt.Printf("Re-encoded fields: %d\n", reFields)

			// Show byte-level diff
			showDiff(headerRaw, reencoded)

			// Show header details
			fmt.Printf("\nHeader details:\n")
			fmt.Printf("  Number:         %s\n", header.Number)
			fmt.Printf("  ParentHash:     %s\n", header.ParentHash.Hex())
			fmt.Printf("  BaseFee:        %v\n", header.BaseFee)
			fmt.Printf("  ExtDataHash:    %v\n", header.ExtDataHash)
			fmt.Printf("  ExtDataGasUsed: %v\n", header.ExtDataGasUsed)
			fmt.Printf("  BlockGasCost:   %v\n", header.BlockGasCost)
			fmt.Printf("  BlobGasUsed:    %v\n", header.BlobGasUsed)
			fmt.Printf("  ExcessBlobGas:  %v\n", header.ExcessBlobGas)
			fmt.Printf("  ParentBeacon:   %v\n", header.ParentBeaconRoot)
			fmt.Printf("  Withdrawals:    %v\n", header.WithdrawalsHash)
			fmt.Printf("  RequestsHash:   %v\n", header.RequestsHash)
		} else {
			fmt.Printf("block %d (height %s): OK\n", blockNum, header.Number)
		}

		blockNum++

		// Use content to avoid unused variable warning
		_ = content
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Blocks processed: %d\n", blockNum)
	fmt.Printf("Mismatches:       %d\n", mismatches)

	if mismatches > 0 {
		os.Exit(1)
	}
}

// extractHeaderRLP extracts raw header bytes from a block RLP
func extractHeaderRLP(blockData []byte) ([]byte, error) {
	// Block is [header, txs, uncles, ...]
	content, _, err := rlp.SplitList(blockData)
	if err != nil {
		return nil, fmt.Errorf("split block list: %w", err)
	}

	// First item is header - get its content and rest to calculate size
	_, headerContent, rest, err := rlp.Split(content)
	if err != nil {
		return nil, fmt.Errorf("split header: %w", err)
	}

	// Header bytes = content minus rest
	headerSize := len(content) - len(rest)
	_ = headerContent // Use to avoid warning
	return content[:headerSize], nil
}

// keccak256 computes the keccak256 hash
func keccak256(data []byte) common.Hash {
	h := crypto.NewKeccakState()
	h.Write(data)
	var hash common.Hash
	h.Read(hash[:])
	return hash
}

// countRLPFields counts fields in an RLP list
func countRLPFields(data []byte) (int, error) {
	content, _, err := rlp.SplitList(data)
	if err != nil {
		return 0, err
	}
	count := 0
	for len(content) > 0 {
		_, rest, err := rlp.SplitString(content)
		if err != nil {
			_, rest, err = rlp.SplitList(content)
			if err != nil {
				return count, err
			}
		}
		content = rest
		count++
	}
	return count, nil
}

// showDiff shows byte-level differences between two byte slices
func showDiff(raw, reencoded []byte) {
	fmt.Printf("\nByte comparison:\n")

	// Find first difference
	minLen := len(raw)
	if len(reencoded) < minLen {
		minLen = len(reencoded)
	}

	firstDiff := -1
	for i := 0; i < minLen; i++ {
		if raw[i] != reencoded[i] {
			firstDiff = i
			break
		}
	}

	if firstDiff == -1 && len(raw) != len(reencoded) {
		firstDiff = minLen
	}

	if firstDiff == -1 {
		fmt.Printf("  No byte differences found (unexpected)\n")
		return
	}

	fmt.Printf("  First difference at byte %d\n", firstDiff)

	// Show context around first difference
	start := firstDiff - 16
	if start < 0 {
		start = 0
	}
	end := firstDiff + 32
	if end > len(raw) {
		end = len(raw)
	}

	fmt.Printf("\n  Raw bytes [%d:%d]:\n", start, end)
	fmt.Printf("  %s\n", hex.EncodeToString(raw[start:end]))

	end = firstDiff + 32
	if end > len(reencoded) {
		end = len(reencoded)
	}
	fmt.Printf("\n  Re-encoded bytes [%d:%d]:\n", start, end)
	if start < len(reencoded) {
		fmt.Printf("  %s\n", hex.EncodeToString(reencoded[start:end]))
	}

	// Decode and show field-by-field comparison
	fmt.Printf("\nField-by-field analysis:\n")
	analyzeFieldDiff(raw, reencoded)
}

// analyzeFieldDiff compares fields between raw and re-encoded RLP
func analyzeFieldDiff(raw, reencoded []byte) {
	rawFields := splitFields(raw)
	reFields := splitFields(reencoded)

	maxFields := len(rawFields)
	if len(reFields) > maxFields {
		maxFields = len(reFields)
	}

	fieldNames := []string{
		"ParentHash", "UncleHash", "Coinbase", "Root", "TxHash",
		"ReceiptHash", "Bloom", "Difficulty", "Number", "GasLimit",
		"GasUsed", "Time", "Extra", "MixDigest", "Nonce",
		"BaseFee", "ExtDataHash", "ExtDataGasUsed", "BlockGasCost",
		"BlobGasUsed", "ExcessBlobGas", "ParentBeaconRoot", "WithdrawalsHash", "RequestsHash",
	}

	for i := 0; i < maxFields; i++ {
		name := fmt.Sprintf("Field[%d]", i)
		if i < len(fieldNames) {
			name = fieldNames[i]
		}

		var rawField, reField []byte
		if i < len(rawFields) {
			rawField = rawFields[i]
		}
		if i < len(reFields) {
			reField = reFields[i]
		}

		if !bytes.Equal(rawField, reField) {
			fmt.Printf("  DIFF %s:\n", name)
			fmt.Printf("    raw:       %s\n", hex.EncodeToString(rawField))
			fmt.Printf("    reencoded: %s\n", hex.EncodeToString(reField))
		}
	}
}

// splitFields splits an RLP list into individual field bytes
func splitFields(data []byte) [][]byte {
	content, _, err := rlp.SplitList(data)
	if err != nil {
		return nil
	}

	var fields [][]byte
	for len(content) > 0 {
		_, _, rest, err := rlp.Split(content)
		if err != nil {
			break
		}
		fieldSize := len(content) - len(rest)
		fields = append(fields, content[:fieldSize])
		content = rest
	}
	return fields
}
