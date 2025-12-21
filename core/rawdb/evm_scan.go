// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rawdb

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/rlp"
)

// PreShanghaiHeader represents a header before the Shanghai fork.
// EVM uses this format (no WithdrawalsHash field).
type PreShanghaiHeader struct {
	ParentHash  common.Hash
	UncleHash   common.Hash
	Coinbase    common.Address
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
	Bloom       types.Bloom
	Difficulty  *big.Int
	Number      *big.Int
	GasLimit    uint64
	GasUsed     uint64
	Time        uint64
	Extra       []byte
	MixDigest   common.Hash
	Nonce       types.BlockNonce
	BaseFee     *big.Int `rlp:"optional"` // EIP-1559
	ExtDataHash []byte   `rlp:"optional"` // EVM specific
}

// ToPostShanghai converts pre-Shanghai header to post-Shanghai types.Header
// CRITICAL: This changes the header hash because RLP encoding changes!
func (h *PreShanghaiHeader) ToPostShanghai() *types.Header {
	return &types.Header{
		ParentHash:      h.ParentHash,
		UncleHash:       h.UncleHash,
		Coinbase:        h.Coinbase,
		Root:            h.Root,
		TxHash:          h.TxHash,
		ReceiptHash:     h.ReceiptHash,
		Bloom:           h.Bloom,
		Difficulty:      h.Difficulty,
		Number:          h.Number,
		GasLimit:        h.GasLimit,
		GasUsed:         h.GasUsed,
		Time:            h.Time,
		Extra:           h.Extra,
		MixDigest:       h.MixDigest,
		Nonce:           h.Nonce,
		BaseFee:         h.BaseFee,
		WithdrawalsHash: nil, // Pre-Shanghai has no withdrawals
	}
}

// ScanEVMHeaders scans a EVM database (with 32-byte namespace prefix)
// and returns all headers found WITH POST-SHANGHAI HASHES.
//
// CRITICAL: EVM headers are in pre-Shanghai format (no WithdrawalsHash).
// We convert them to post-Shanghai format and return the NEW hashes.
// This ensures canonical mappings use post-Shanghai hashes that match what
// the blockchain will compute when loading headers.
//
// EVM format: [32-byte namespace][h][8-byte num][32-byte hash]
// We scan for keys starting with namespace+'h', extract the number and hash.
func ScanEVMHeaders(rawDB ethdb.Database, namespace []byte) (map[uint64]common.Hash, error) {
	if len(namespace) != 32 {
		return nil, fmt.Errorf("namespace must be exactly 32 bytes, got %d", len(namespace))
	}

	headers := make(map[uint64]common.Hash)

	// Build prefix for header keys: namespace + 'h'
	headerKeyPrefix := append(namespace, headerPrefix[0])

	iter := rawDB.NewIterator(headerKeyPrefix, nil)
	defer iter.Release()

	scanned := 0
	converted := 0
	written := 0
	bodiesWritten := 0

	batch := rawDB.NewBatch()

	for iter.Next() {
		key := iter.Key()

		// Expected format: [32-byte namespace]['h'][8-byte num][32-byte hash] = 73 bytes
		if len(key) != 73 {
			continue
		}

		// Extract block number (bytes 33-41) and pre-Shanghai hash (bytes 41-73)
		num := binary.BigEndian.Uint64(key[33:41])
		preShanghaiHash := common.BytesToHash(key[41:73])

		// Read the header data to convert to post-Shanghai format
		headerData := iter.Value()
		if len(headerData) > 0 {
			// Try to decode as pre-Shanghai header
			var preHeader PreShanghaiHeader
			if err := rlp.DecodeBytes(headerData, &preHeader); err == nil {
				// Convert to post-Shanghai format (adds WithdrawalsHash field)
				postHeader := preHeader.ToPostShanghai()

				// The post-Shanghai hash is DIFFERENT from pre-Shanghai hash
				// because the RLP encoding changes when we add WithdrawalsHash
				postShanghaiHash := postHeader.Hash()

				// Store the POST-SHANGHAI hash for canonical mappings
				headers[num] = postShanghaiHash
				converted++

				// CRITICAL: Write the header under BOTH hashes
				// 1. Post-Shanghai hash (for canonical lookups)
				// 2. Pre-Shanghai hash (for parent lookups during chain walking)
				// This ensures blockchain can find headers both ways

				// Encode the post-Shanghai header
				postHeaderData, err := rlp.EncodeToBytes(postHeader)
				if err != nil {
					fmt.Printf("⚠️  Failed to encode post-Shanghai header for block %d: %v\n", num, err)
					continue
				}

				// Write under POST-SHANGHAI hash (for canonical lookups)
				postHeaderKey := make([]byte, 41)
				postHeaderKey[0] = headerPrefix[0] // 'h'
				binary.BigEndian.PutUint64(postHeaderKey[1:9], num)
				copy(postHeaderKey[9:41], postShanghaiHash.Bytes())

				if err := batch.Put(postHeaderKey, postHeaderData); err != nil {
					fmt.Printf("⚠️  Failed to write post-Shanghai header for block %d: %v\n", num, err)
					continue
				}

				// ALSO write under PRE-SHANGHAI hash (for parent lookups)
				// When blockchain walks parent chain, it uses ParentHash from headers
				// which are pre-Shanghai hashes
				preHeaderKey := make([]byte, 41)
				preHeaderKey[0] = headerPrefix[0] // 'h'
				binary.BigEndian.PutUint64(preHeaderKey[1:9], num)
				copy(preHeaderKey[9:41], preShanghaiHash.Bytes())

				if err := batch.Put(preHeaderKey, postHeaderData); err != nil {
					fmt.Printf("⚠️  Failed to write pre-Shanghai header for block %d: %v\n", num, err)
					continue
				}

				// ALSO read and write the body under BOTH hashes
				// Body key format: [namespace]['b'][8-byte num][32-byte hash]
				preBodyKey := make([]byte, 73)
				copy(preBodyKey[0:32], namespace)
				preBodyKey[32] = 'b'
				binary.BigEndian.PutUint64(preBodyKey[33:41], num)
				copy(preBodyKey[41:73], preShanghaiHash.Bytes())

				if bodyData, err := rawDB.Get(preBodyKey); err == nil && len(bodyData) > 0 {
					// Write body under POST-SHANGHAI hash
					postBodyKey := make([]byte, 41)
					postBodyKey[0] = 'b'
					binary.BigEndian.PutUint64(postBodyKey[1:9], num)
					copy(postBodyKey[9:41], postShanghaiHash.Bytes())

					if err := batch.Put(postBodyKey, bodyData); err != nil {
						fmt.Printf("⚠️  Failed to write post-Shanghai body for block %d: %v\n", num, err)
					} else {
						// Also write under PRE-SHANGHAI hash
						preBodyKeyNoNS := make([]byte, 41)
						preBodyKeyNoNS[0] = 'b'
						binary.BigEndian.PutUint64(preBodyKeyNoNS[1:9], num)
						copy(preBodyKeyNoNS[9:41], preShanghaiHash.Bytes())

						if err := batch.Put(preBodyKeyNoNS, bodyData); err != nil {
							fmt.Printf("⚠️  Failed to write pre-Shanghai body for block %d: %v\n", num, err)
						} else {
							bodiesWritten++
						}
					}
				}

				written++

				// Commit batch every 10000 headers
				if written%10000 == 0 {
					if err := batch.Write(); err != nil {
						return nil, fmt.Errorf("failed to write batch: %w", err)
					}
					batch.Reset()
					fmt.Printf("📝 Written %d headers and %d bodies to database\n", written, bodiesWritten)
				}

				if converted%100000 == 0 {
					fmt.Printf("📊 Converted %d headers to post-Shanghai format, highest: %d\n", converted, num)
				}
			} else {
				// If decode fails, use the original pre-Shanghai hash
				// This shouldn't happen but provides fallback
				headers[num] = preShanghaiHash
			}
		} else {
			// No header data, use original hash
			headers[num] = preShanghaiHash
		}

		scanned++
	}

	// Write any remaining headers in batch
	if err := batch.Write(); err != nil {
		return nil, fmt.Errorf("failed to write final header batch: %w", err)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	fmt.Printf("✅ Found %d headers, converted %d to post-Shanghai format\n", len(headers), converted)
	fmt.Printf("✅ Written %d post-Shanghai headers and %d bodies to database\n", written, bodiesWritten)
	return headers, nil
}

// WriteMappings writes canonical mappings to database
// Simple, focused: just write forward and reverse mappings
func WriteMappings(db ethdb.Database, headers map[uint64]common.Hash) error {
	batch := db.NewBatch()
	written := 0

	for num, hash := range headers {
		WriteCanonicalHash(batch, hash, num)
		WriteHeaderNumber(batch, hash, num)
		written++

		if written%10000 == 0 {
			if err := batch.Write(); err != nil {
				return fmt.Errorf("batch write failed: %w", err)
			}
			batch.Reset()
			fmt.Printf("📝 Written %d/%d mappings\n", written, len(headers))
		}
	}

	return batch.Write()
}

// WriteMappingsWithNamespace writes canonical mappings with EVM namespace prefix
// For EVM databases: adds 32-byte namespace prefix to all keys
func WriteMappingsWithNamespace(db ethdb.Database, headers map[uint64]common.Hash, namespace []byte) error {
	if len(namespace) != 32 {
		return fmt.Errorf("namespace must be exactly 32 bytes, got %d", len(namespace))
	}

	batch := db.NewBatch()
	written := 0

	for num, hash := range headers {
		// Write canonical hash with namespace: namespace + 'h' + be8(num) → hash
		canonKey := make([]byte, 32+1+8)
		copy(canonKey[0:32], namespace)
		canonKey[32] = headerPrefix[0] // 'h'
		binary.BigEndian.PutUint64(canonKey[33:41], num)
		if err := batch.Put(canonKey, hash.Bytes()); err != nil {
			return fmt.Errorf("failed to write canonical hash: %w", err)
		}

		// Write header number with namespace: namespace + 'H' + hash → be8(num)
		numKey := make([]byte, 32+1+32)
		copy(numKey[0:32], namespace)
		numKey[32] = headerNumberPrefix[0] // 'H'
		copy(numKey[33:65], hash.Bytes())
		numBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(numBytes, num)
		if err := batch.Put(numKey, numBytes); err != nil {
			return fmt.Errorf("failed to write header number: %w", err)
		}

		written++

		if written%10000 == 0 {
			if err := batch.Write(); err != nil {
				return fmt.Errorf("batch write failed: %w", err)
			}
			batch.Reset()
			fmt.Printf("📝 Written %d/%d namespaced mappings\n", written, len(headers))
		}
	}

	return batch.Write()
}

// VerifyMappings checks that mappings are readable
func VerifyMappings(db ethdb.Database, headers map[uint64]common.Hash, checkBlocks []uint64) error {
	failed := 0

	for _, num := range checkBlocks {
		expected, ok := headers[num]
		if !ok {
			continue
		}

		actual := ReadCanonicalHash(db, num)
		if actual != expected {
			failed++
			fmt.Printf("  ✗ Block %d mismatch\n", num)
		}
	}

	if failed > 0 {
		return fmt.Errorf("%d blocks failed verification", failed)
	}

	return nil
}
