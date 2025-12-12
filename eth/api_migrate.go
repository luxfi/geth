// Copyright 2025 Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package eth

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/geth/rlp"
)

// MigrateAPI provides RPC methods for migration operations
type MigrateAPI struct {
	eth *Ethereum
}

// NewMigrateAPI creates a new instance of MigrateAPI
func NewMigrateAPI(eth *Ethereum) *MigrateAPI {
	return &MigrateAPI{eth: eth}
}

// BlockEntry represents a block entry for migration
type BlockEntry struct {
	Height   uint64 `json:"height"`
	Hash     string `json:"hash"`
	Header   string `json:"header"`   // hex-encoded RLP
	Body     string `json:"body"`     // hex-encoded RLP
	Receipts string `json:"receipts"` // hex-encoded RLP
}

// ProcessBlocksResult is the response from ProcessBlocks
type ProcessBlocksResult struct {
	Processed   int      `json:"processed"`
	Failed      int      `json:"failed"`
	FirstHeight uint64   `json:"firstHeight"`
	LastHeight  uint64   `json:"lastHeight"`
	CurrentHead uint64   `json:"currentHead"`
	StateRoot   string   `json:"stateRoot"`
	Errors      []string `json:"errors,omitempty"`
}

// ProcessBlocks imports and processes a batch of blocks with state execution
func (api *MigrateAPI) ProcessBlocks(blocks []BlockEntry) (*ProcessBlocksResult, error) {
	if len(blocks) == 0 {
		return &ProcessBlocksResult{}, nil
	}

	result := &ProcessBlocksResult{
		FirstHeight: blocks[0].Height,
		LastHeight:  blocks[len(blocks)-1].Height,
	}

	blockchain := api.eth.BlockChain()
	decodedBlocks := make([]*types.Block, 0, len(blocks))

	for i, entry := range blocks {
		// Skip genesis
		if entry.Height == 0 {
			continue
		}

		block, err := decodeBlockEntry(&entry)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: decode error: %v", entry.Height, err))
			log.Warn("Failed to decode block", "height", entry.Height, "idx", i, "err", err)
			continue
		}

		// Skip if already exists
		if blockchain.HasBlock(block.Hash(), block.NumberU64()) {
			continue
		}

		decodedBlocks = append(decodedBlocks, block)
	}

	if len(decodedBlocks) == 0 {
		// All blocks already exist
		currentBlock := blockchain.CurrentBlock()
		result.CurrentHead = currentBlock.Number.Uint64()
		result.StateRoot = currentBlock.Root.Hex()
		return result, nil
	}

	// Insert the chain - this executes transactions and builds state
	if _, err := blockchain.InsertChain(decodedBlocks); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("chain insert failed: %v", err))
		return result, fmt.Errorf("failed to insert blocks: %w", err)
	}

	result.Processed = len(decodedBlocks)

	// Get current state
	currentBlock := blockchain.CurrentBlock()
	result.CurrentHead = currentBlock.Number.Uint64()
	result.StateRoot = currentBlock.Root.Hex()

	log.Info("Processed blocks",
		"count", result.Processed,
		"first", result.FirstHeight,
		"last", result.LastHeight,
		"head", result.CurrentHead,
		"stateRoot", result.StateRoot)

	return result, nil
}

// Status returns the current migration status
func (api *MigrateAPI) Status() map[string]interface{} {
	blockchain := api.eth.BlockChain()
	current := blockchain.CurrentBlock()

	return map[string]interface{}{
		"currentBlock": current.Number.Uint64(),
		"currentHash":  current.Hash().Hex(),
		"stateRoot":    current.Root.Hex(),
		"chainId":      blockchain.Config().ChainID.Uint64(),
	}
}

// decodeBlockEntry decodes a BlockEntry into a types.Block
func decodeBlockEntry(entry *BlockEntry) (*types.Block, error) {
	// Decode header
	headerBytes, err := hexToBytes(entry.Header)
	if err != nil {
		return nil, fmt.Errorf("decode header hex: %w", err)
	}

	var header types.Header
	if err := rlp.DecodeBytes(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("decode header RLP: %w", err)
	}

	// Decode body
	bodyBytes, err := hexToBytes(entry.Body)
	if err != nil {
		return nil, fmt.Errorf("decode body hex: %w", err)
	}

	var body types.Body
	if err := rlp.DecodeBytes(bodyBytes, &body); err != nil {
		return nil, fmt.Errorf("decode body RLP: %w", err)
	}

	// Create block
	block := types.NewBlockWithHeader(&header).WithBody(body)

	return block, nil
}

// hexToBytes decodes a hex string (with or without 0x prefix) to bytes
func hexToBytes(s string) ([]byte, error) {
	s = strings.TrimPrefix(s, "0x")
	return hex.DecodeString(s)
}

// hasAllBlocks checks if all blocks already exist
func hasAllBlocks(chain *core.BlockChain, blocks []*types.Block) bool {
	for _, b := range blocks {
		if !chain.HasBlock(b.Hash(), b.NumberU64()) {
			return false
		}
	}
	return true
}
