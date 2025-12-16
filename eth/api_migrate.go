// Copyright 2025 Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package eth

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/holiman/uint256"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/log"
	"github.com/luxfi/geth/rlp"
	"github.com/luxfi/geth/trie"
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
// Supports two formats:
// - Zoo format: height, header, body, receipts
// - Lux format: number, header_rlp, body_rlp, receipts_rlp
type BlockEntry struct {
	// Zoo format
	Height   uint64 `json:"height"`
	Header   string `json:"header"`   // hex-encoded RLP
	Body     string `json:"body"`     // hex-encoded RLP
	Receipts string `json:"receipts"` // hex-encoded RLP

	// Lux format (alternative field names)
	Number      uint64 `json:"number"`
	HeaderRLP   string `json:"header_rlp"`
	BodyRLP     string `json:"body_rlp"`
	ReceiptsRLP string `json:"receipts_rlp"`

	// Common
	Hash string `json:"hash"`
}

// GetHeight returns the block height from either format
func (e *BlockEntry) GetHeight() uint64 {
	if e.Height > 0 {
		return e.Height
	}
	return e.Number
}

// GetHeader returns the header RLP from either format
func (e *BlockEntry) GetHeader() string {
	if e.Header != "" {
		return e.Header
	}
	return e.HeaderRLP
}

// GetBody returns the body RLP from either format
func (e *BlockEntry) GetBody() string {
	if e.Body != "" {
		return e.Body
	}
	return e.BodyRLP
}

// GetReceipts returns the receipts RLP from either format
func (e *BlockEntry) GetReceipts() string {
	if e.Receipts != "" {
		return e.Receipts
	}
	return e.ReceiptsRLP
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

// ProcessBlocksRequest holds the request for ProcessBlocks
type ProcessBlocksRequest struct {
	Blocks []BlockEntry `json:"blocks"`
}

// ReplayBlocksRequest holds the request for ReplayBlocks
// This mode executes transactions but allows state root mismatches
// for cross-VM migration where trie encoding differs
type ReplayBlocksRequest struct {
	Blocks  []BlockEntry `json:"blocks"`
	SetHead bool         `json:"setHead"` // If true, update blockchain head after processing
}

// ReplayBlocksResult is the response from ReplayBlocks
type ReplayBlocksResult struct {
	Processed        int               `json:"processed"`
	Failed           int               `json:"failed"`
	FirstHeight      uint64            `json:"firstHeight"`
	LastHeight       uint64            `json:"lastHeight"`
	CurrentHead      uint64            `json:"currentHead"`
	GethStateRoot    string            `json:"gethStateRoot"`    // State root computed by geth
	OriginalStateRoot string           `json:"originalStateRoot"` // State root from original block header
	ComputedRoots    map[uint64]string `json:"computedRoots"`    // Per-block computed roots
	Errors           []string          `json:"errors,omitempty"`
}

// ProcessBlocks imports and processes a batch of blocks with full state execution
func (api *MigrateAPI) ProcessBlocks(req ProcessBlocksRequest) (*ProcessBlocksResult, error) {
	if len(req.Blocks) == 0 {
		return &ProcessBlocksResult{}, nil
	}

	result := &ProcessBlocksResult{
		FirstHeight: req.Blocks[0].Height,
		LastHeight:  req.Blocks[len(req.Blocks)-1].Height,
	}

	blockchain := api.eth.BlockChain()
	decodedBlocks := make([]*types.Block, 0, len(req.Blocks))

	for i, entry := range req.Blocks {
		// Skip genesis
		if entry.GetHeight() == 0 {
			continue
		}

		block, err := decodeBlockEntry(&entry)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: decode error: %v", entry.GetHeight(), err))
			log.Warn("Failed to decode block", "height", entry.GetHeight(), "idx", i, "err", err)
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

// ReplayBlocks imports blocks from another VM by re-executing transactions and
// creating new blocks with geth's computed state roots. This enables cross-VM
// migration where the source VM (e.g., SubnetEVM) uses different trie encodings.
//
// The process:
// 1. Decode original block to get transactions
// 2. Execute transactions using geth's EVM
// 3. Create new block with geth's computed state root, receipt hash, and bloom
// 4. Insert block through normal blockchain APIs
//
// IMPORTANT: Blocks must be processed sequentially starting from block 1.
func (api *MigrateAPI) ReplayBlocks(req ReplayBlocksRequest) (*ReplayBlocksResult, error) {
	if len(req.Blocks) == 0 {
		return &ReplayBlocksResult{ComputedRoots: make(map[uint64]string)}, nil
	}

	result := &ReplayBlocksResult{
		FirstHeight:   req.Blocks[0].Height,
		LastHeight:    req.Blocks[len(req.Blocks)-1].Height,
		ComputedRoots: make(map[uint64]string),
	}

	blockchain := api.eth.BlockChain()
	processor := blockchain.Processor()
	chainDb := api.eth.ChainDb()

	// Track migration state for continuation across RPC calls
	migrateStateKey := []byte("migrate-state-root")
	migrateHeightKey := []byte("migrate-height")
	migrateParentKey := []byte("migrate-parent-hash")

	var lastComputedRoot common.Hash
	var lastHeight uint64
	var lastParentHash common.Hash

	// Check if we have prior migration state
	if storedRoot, _ := chainDb.Get(migrateStateKey); len(storedRoot) == 32 {
		copy(lastComputedRoot[:], storedRoot)
		if storedHeight, _ := chainDb.Get(migrateHeightKey); len(storedHeight) == 8 {
			lastHeight = binary.BigEndian.Uint64(storedHeight)
		}
		if storedParent, _ := chainDb.Get(migrateParentKey); len(storedParent) == 32 {
			copy(lastParentHash[:], storedParent)
		}
	}

	// If no stored state, start from genesis
	if lastComputedRoot == (common.Hash{}) {
		genesis := blockchain.Genesis()
		lastComputedRoot = genesis.Root()
		lastParentHash = genesis.Hash()
		lastHeight = 0
	}

	log.Info("Starting ReplayBlocks",
		"startHeight", req.Blocks[0].Height,
		"endHeight", req.Blocks[len(req.Blocks)-1].Height,
		"lastProcessedHeight", lastHeight,
		"lastStateRoot", lastComputedRoot.Hex())

	for _, entry := range req.Blocks {
		// Skip genesis
		if entry.GetHeight() == 0 {
			continue
		}

		// Check for sequential blocks
		if entry.GetHeight() != lastHeight+1 {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: expected height %d (blocks must be sequential)", entry.GetHeight(), lastHeight+1))
			continue
		}

		origBlock, err := decodeBlockEntry(&entry)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: decode error: %v", entry.GetHeight(), err))
			log.Warn("Failed to decode block for replay", "height", entry.GetHeight(), "err", err)
			continue
		}

		// Create state from geth's computed parent state root
		statedb, err := blockchain.StateAt(lastComputedRoot)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: failed to open state at root %s: %v", entry.GetHeight(), lastComputedRoot.Hex(), err))
			continue
		}

		// Process block (execute transactions)
		processResult, err := processor.Process(origBlock, statedb, *blockchain.GetVMConfig())
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: process error: %v", entry.GetHeight(), err))
			continue
		}
		receipts := processResult.Receipts
		usedGas := processResult.GasUsed

		// Compute geth's values
		computedRoot := statedb.IntermediateRoot(blockchain.Config().IsEIP158(origBlock.Number()))
		receiptHash := types.DeriveSha(receipts, trie.NewStackTrie(nil))
		bloom := types.MergeBloom(receipts)

		result.ComputedRoots[entry.GetHeight()] = computedRoot.Hex()

		// Create new header with geth's computed values
		newHeader := &types.Header{
			ParentHash:       lastParentHash, // Point to our previous geth block
			UncleHash:        origBlock.UncleHash(),
			Coinbase:         origBlock.Coinbase(),
			Root:             computedRoot, // Geth's state root
			TxHash:           origBlock.TxHash(),
			ReceiptHash:      receiptHash, // Geth's receipt hash
			Bloom:            bloom,       // Geth's bloom
			Difficulty:       origBlock.Difficulty(),
			Number:           origBlock.Number(),
			GasLimit:         origBlock.GasLimit(),
			GasUsed:          usedGas,
			Time:             origBlock.Time(),
			Extra:            origBlock.Extra(),
			MixDigest:        origBlock.MixDigest(),
			Nonce:            origBlock.Header().Nonce,
			BaseFee:          origBlock.BaseFee(),
			WithdrawalsHash:  origBlock.Header().WithdrawalsHash,
			BlobGasUsed:      origBlock.Header().BlobGasUsed,
			ExcessBlobGas:    origBlock.Header().ExcessBlobGas,
			ParentBeaconRoot: origBlock.Header().ParentBeaconRoot,
			RequestsHash:     origBlock.Header().RequestsHash,
		}

		// Create new block with geth header and original body
		newBlock := types.NewBlockWithHeader(newHeader).WithBody(*origBlock.Body())

		// Commit state
		root, err := statedb.Commit(newBlock.NumberU64(), blockchain.Config().IsEIP158(newBlock.Number()), false)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("block %d: commit error: %v", entry.GetHeight(), err))
			continue
		}

		// Flush trie periodically
		if entry.GetHeight()%50 == 0 || entry.GetHeight() == req.Blocks[len(req.Blocks)-1].GetHeight() {
			if err := blockchain.TrieDB().Commit(root, false); err != nil {
				log.Warn("Failed to commit trie", "height", entry.GetHeight(), "root", root.Hex(), "err", err)
			}
		}

		// Write block to database
		rawdb.WriteBlock(chainDb, newBlock)
		rawdb.WriteReceipts(chainDb, newBlock.Hash(), newBlock.NumberU64(), receipts)
		rawdb.WriteCanonicalHash(chainDb, newBlock.Hash(), newBlock.NumberU64())
		rawdb.WriteHeadBlockHash(chainDb, newBlock.Hash())
		rawdb.WriteHeadHeaderHash(chainDb, newBlock.Hash())
		rawdb.WriteTxLookupEntriesByBlock(chainDb, newBlock)

		// Update tracking for next block
		lastComputedRoot = root
		lastParentHash = newBlock.Hash()
		lastHeight = entry.GetHeight()

		// Persist migration state
		chainDb.Put(migrateStateKey, root.Bytes())
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, entry.GetHeight())
		chainDb.Put(migrateHeightKey, heightBytes)
		chainDb.Put(migrateParentKey, newBlock.Hash().Bytes())

		result.Processed++

		log.Info("Replayed block",
			"height", entry.GetHeight(),
			"hash", newBlock.Hash().Hex()[:10],
			"txs", len(newBlock.Transactions()),
			"gas", usedGas,
			"root", root.Hex())
	}

	// Get current state
	currentBlock := blockchain.CurrentBlock()
	if currentBlock != nil {
		result.CurrentHead = currentBlock.Number.Uint64()
		result.GethStateRoot = currentBlock.Root.Hex()
	}

	if len(req.Blocks) > 0 {
		lastBlock, _ := decodeBlockEntry(&req.Blocks[len(req.Blocks)-1])
		if lastBlock != nil {
			result.OriginalStateRoot = lastBlock.Root().Hex()
		}
	}

	log.Info("Replay complete",
		"processed", result.Processed,
		"failed", result.Failed,
		"first", result.FirstHeight,
		"last", result.LastHeight)

	// Reload blockchain head from database
	if req.SetHead && result.Processed > 0 && lastHeight > 0 {
		if err := blockchain.SetHead(lastHeight); err != nil {
			log.Warn("Failed to set blockchain head after replay", "height", lastHeight, "err", err)
		} else {
			currentBlock = blockchain.CurrentBlock()
			if currentBlock != nil {
				result.CurrentHead = currentBlock.Number.Uint64()
				result.GethStateRoot = currentBlock.Root.Hex()
			}
			log.Info("Updated blockchain head", "height", lastHeight)
		}
	}

	return result, nil
}

// VerifyBalance checks if an account has the expected balance at the current migration state
// Use this after ReplayBlocks to verify final state correctness
func (api *MigrateAPI) VerifyBalance(address string, expectedBalanceHex string) (map[string]interface{}, error) {
	blockchain := api.eth.BlockChain()
	chainDb := api.eth.ChainDb()

	// Read migration state (computed state root from ReplayBlocks)
	migrateStateKey := []byte("migrate-state-root")
	migrateHeightKey := []byte("migrate-height")

	var stateRoot common.Hash
	var height uint64

	if storedRoot, _ := chainDb.Get(migrateStateKey); len(storedRoot) == 32 {
		copy(stateRoot[:], storedRoot)
		if storedHeight, _ := chainDb.Get(migrateHeightKey); len(storedHeight) == 8 {
			height = binary.BigEndian.Uint64(storedHeight)
		}
	}

	// If no migration state, fall back to blockchain's current block
	if stateRoot == (common.Hash{}) {
		currentBlock := blockchain.CurrentBlock()
		stateRoot = currentBlock.Root
		height = currentBlock.Number.Uint64()
	}

	state, err := blockchain.StateAt(stateRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open state at root %s: %w", stateRoot.Hex(), err)
	}

	addr := common.HexToAddress(address)
	actualBalance := state.GetBalance(addr)

	// Parse expected balance as uint256
	expectedStr := strings.TrimPrefix(expectedBalanceHex, "0x")
	expectedBalance, err := uint256.FromHex("0x" + expectedStr)
	if err != nil {
		return nil, fmt.Errorf("invalid expected balance hex: %s: %w", expectedBalanceHex, err)
	}

	match := actualBalance.Cmp(expectedBalance) == 0

	return map[string]interface{}{
		"address":         address,
		"expectedBalance": expectedBalanceHex,
		"actualBalance":   fmt.Sprintf("0x%x", actualBalance),
		"match":           match,
		"blockNumber":     height,
		"stateRoot":       stateRoot.Hex(),
	}, nil
}

// SetGenesisRLP replaces the genesis block with the provided RLP-encoded data
// This is a destructive operation used for chain migration
func (api *MigrateAPI) SetGenesisRLP(headerHex, bodyHex string) (map[string]interface{}, error) {
	// Decode header
	headerBytes, err := hexToBytes(headerHex)
	if err != nil {
		return nil, fmt.Errorf("decode header hex: %w", err)
	}

	var header types.Header
	if err := rlp.DecodeBytes(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("decode header RLP: %w", err)
	}

	// Decode body
	bodyBytes, err := hexToBytes(bodyHex)
	if err != nil {
		return nil, fmt.Errorf("decode body hex: %w", err)
	}

	var body types.Body
	if err := rlp.DecodeBytes(bodyBytes, &body); err != nil {
		return nil, fmt.Errorf("decode body RLP: %w", err)
	}

	// Verify this is a genesis block
	if header.Number.Uint64() != 0 {
		return nil, fmt.Errorf("not a genesis block: number=%d", header.Number.Uint64())
	}

	// Create block
	block := types.NewBlockWithHeader(&header).WithBody(body)

	log.Info("Setting genesis from RLP",
		"hash", block.Hash().Hex(),
		"stateRoot", header.Root.Hex(),
		"gasLimit", header.GasLimit)

	// Use ResetWithGenesisBlock to properly reset the chain
	blockchain := api.eth.BlockChain()
	if err := blockchain.ResetWithGenesisBlock(block); err != nil {
		return nil, fmt.Errorf("reset with genesis: %w", err)
	}

	log.Info("Genesis block set successfully",
		"hash", block.Hash().Hex(),
		"stateRoot", header.Root.Hex())

	return map[string]interface{}{
		"hash":      block.Hash().Hex(),
		"stateRoot": header.Root.Hex(),
		"gasLimit":  header.GasLimit,
		"timestamp": header.Time,
	}, nil
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
// Supports both Zoo format (header, body) and Lux format (header_rlp, body_rlp)
func decodeBlockEntry(entry *BlockEntry) (*types.Block, error) {
	// Decode header (use getter to support both formats)
	headerBytes, err := hexToBytes(entry.GetHeader())
	if err != nil {
		return nil, fmt.Errorf("decode header hex: %w", err)
	}

	var header types.Header
	if err := rlp.DecodeBytes(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("decode header RLP: %w", err)
	}

	// Decode body (use getter to support both formats)
	bodyBytes, err := hexToBytes(entry.GetBody())
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
