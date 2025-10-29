// Copyright 2025 Lux Industries, Inc.
// Unified Quantum Block Implementation for LUX Ecosystem

package types

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/luxfi/geth/common"
)

// BlockVersion defines the version of the block format
type BlockVersion uint8

const (
	// LegacyBlockVersion for historical blocks (pre-quantum)
	LegacyBlockVersion BlockVersion = 0
	// QuantumBlockVersion for new enhanced blocks
	QuantumBlockVersion BlockVersion = 1
)

// QuantumMetadata contains enhanced metadata for quantum blocks
type QuantumMetadata struct {
	// Layer information (C-Chain=0, L1=1, L2=2, L3=3)
	Layer uint8 `json:"layer"`
	
	// Network ID for multi-network support
	NetworkID uint64 `json:"networkId"`
	
	// Consensus type (POA=0, POS=1, POW=2)
	ConsensusType uint8 `json:"consensusType"`
	
	// Cross-layer references
	ParentNetworkBlockHash *common.Hash `json:"parentNetworkBlockHash,omitempty" rlp:"optional"`
	
	// Validator set hash for POS/POA
	ValidatorSetHash *common.Hash `json:"validatorSetHash,omitempty" rlp:"optional"`
	
	// Accumulator for state proof
	StateAccumulator *common.Hash `json:"stateAccumulator,omitempty" rlp:"optional"`
	
	// Warp message root for cross-chain communication
	WarpMessageRoot *common.Hash `json:"warpMessageRoot,omitempty" rlp:"optional"`
	
	// Block gas cost (from subnet EVM)
	BlockGasCost *big.Int `json:"blockGasCost,omitempty" rlp:"optional"`
	
	// Extended timestamp with nanosecond precision
	TimestampNano uint64 `json:"timestampNano"`
	
	// Version for future upgrades
	MetadataVersion uint8 `json:"metadataVersion"`
}

// UnifiedHeader represents the unified block header for all LUX layers
// It maintains backward compatibility while supporting quantum features
type UnifiedHeader struct {
	// Standard Ethereum fields (legacy compatible)
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`
	Number      *big.Int       `json:"number"           gencodec:"required"`
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`
	Time        uint64         `json:"timestamp"        gencodec:"required"`
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`
	
	// EIP-1559 and beyond
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"`
	
	// EIP-4895 Withdrawals
	WithdrawalsHash *common.Hash `json:"withdrawalsRoot" rlp:"optional"`
	
	// EIP-4844 Blob gas
	BlobGasUsed   *uint64 `json:"blobGasUsed" rlp:"optional"`
	ExcessBlobGas *uint64 `json:"excessBlobGas" rlp:"optional"`
	
	// EIP-4788 Beacon root
	ParentBeaconRoot *common.Hash `json:"parentBeaconBlockRoot" rlp:"optional"`
	
	// EIP-7685 Requests
	RequestsHash *common.Hash `json:"requestsHash" rlp:"optional"`
	
	// Quantum extensions
	Version  BlockVersion     `json:"version" rlp:"optional"`
	Quantum  *QuantumMetadata `json:"quantum,omitempty" rlp:"optional"`
}

// Cache for quantum metadata
var (
	quantumCache      = make(map[common.Hash]*QuantumMetadata)
	quantumCacheMutex sync.RWMutex
)

// ToLegacyHeader converts UnifiedHeader to standard Header for compatibility
func (h *UnifiedHeader) ToLegacyHeader() *Header {
	return &Header{
		ParentHash:       h.ParentHash,
		UncleHash:        h.UncleHash,
		Coinbase:         h.Coinbase,
		Root:             h.Root,
		TxHash:           h.TxHash,
		ReceiptHash:      h.ReceiptHash,
		Bloom:            h.Bloom,
		Difficulty:       h.Difficulty,
		Number:           h.Number,
		GasLimit:         h.GasLimit,
		GasUsed:          h.GasUsed,
		Time:             h.Time,
		Extra:            h.Extra,
		MixDigest:        h.MixDigest,
		Nonce:            h.Nonce,
		BaseFee:          h.BaseFee,
		WithdrawalsHash:  h.WithdrawalsHash,
		BlobGasUsed:      h.BlobGasUsed,
		ExcessBlobGas:    h.ExcessBlobGas,
		ParentBeaconRoot: h.ParentBeaconRoot,
		RequestsHash:     h.RequestsHash,
	}
}

// FromLegacyHeader creates UnifiedHeader from standard Header
func FromLegacyHeader(h *Header) *UnifiedHeader {
	return &UnifiedHeader{
		ParentHash:       h.ParentHash,
		UncleHash:        h.UncleHash,
		Coinbase:         h.Coinbase,
		Root:             h.Root,
		TxHash:           h.TxHash,
		ReceiptHash:      h.ReceiptHash,
		Bloom:            h.Bloom,
		Difficulty:       h.Difficulty,
		Number:           h.Number,
		GasLimit:         h.GasLimit,
		GasUsed:          h.GasUsed,
		Time:             h.Time,
		Extra:            h.Extra,
		MixDigest:        h.MixDigest,
		Nonce:            h.Nonce,
		BaseFee:          h.BaseFee,
		WithdrawalsHash:  h.WithdrawalsHash,
		BlobGasUsed:      h.BlobGasUsed,
		ExcessBlobGas:    h.ExcessBlobGas,
		ParentBeaconRoot: h.ParentBeaconRoot,
		RequestsHash:     h.RequestsHash,
		Version:          LegacyBlockVersion,
	}
}

// Hash returns the block hash of the header
func (h *UnifiedHeader) Hash() common.Hash {
	if h.Version == QuantumBlockVersion {
		// Use enhanced hash for quantum blocks
		return h.quantumHash()
	}
	// Use standard hash for legacy blocks
	return rlpHash(h.ToLegacyHeader())
}

// quantumHash computes enhanced hash including quantum metadata
func (h *UnifiedHeader) quantumHash() common.Hash {
	// Create hasher with all fields
	hasher := sha256.New()
	
	// Write version
	hasher.Write([]byte{byte(h.Version)})
	
	// Write standard fields hash
	legacyHash := rlpHash(h.ToLegacyHeader())
	hasher.Write(legacyHash[:])
	
	// Write quantum metadata if present
	if h.Quantum != nil {
		binary.Write(hasher, binary.BigEndian, h.Quantum.Layer)
		binary.Write(hasher, binary.BigEndian, h.Quantum.NetworkID)
		binary.Write(hasher, binary.BigEndian, h.Quantum.ConsensusType)
		
		if h.Quantum.ParentNetworkBlockHash != nil {
			hasher.Write(h.Quantum.ParentNetworkBlockHash[:])
		}
		if h.Quantum.ValidatorSetHash != nil {
			hasher.Write(h.Quantum.ValidatorSetHash[:])
		}
		if h.Quantum.StateAccumulator != nil {
			hasher.Write(h.Quantum.StateAccumulator[:])
		}
		if h.Quantum.WarpMessageRoot != nil {
			hasher.Write(h.Quantum.WarpMessageRoot[:])
		}
		if h.Quantum.BlockGasCost != nil {
			hasher.Write(h.Quantum.BlockGasCost.Bytes())
		}
		binary.Write(hasher, binary.BigEndian, h.Quantum.TimestampNano)
		hasher.Write([]byte{h.Quantum.MetadataVersion})
	}
	
	sum := hasher.Sum(nil)
	return common.BytesToHash(sum)
}

// IsQuantum returns true if this is a quantum block
func (h *UnifiedHeader) IsQuantum() bool {
	return h.Version == QuantumBlockVersion
}

// GetBlockGasCost returns the block gas cost from quantum metadata
func (h *UnifiedHeader) GetBlockGasCost() *big.Int {
	if h.Quantum != nil && h.Quantum.BlockGasCost != nil {
		return new(big.Int).Set(h.Quantum.BlockGasCost)
	}
	return nil
}

// SetBlockGasCost sets the block gas cost in quantum metadata
func (h *UnifiedHeader) SetBlockGasCost(cost *big.Int) {
	if h.Quantum == nil {
		h.Quantum = &QuantumMetadata{}
	}
	if cost != nil {
		h.Quantum.BlockGasCost = new(big.Int).Set(cost)
	}
}

// UpgradeToQuantum upgrades a legacy header to quantum format
func (h *UnifiedHeader) UpgradeToQuantum(networkID uint64, layer uint8, consensusType uint8) {
	if h.Version == QuantumBlockVersion {
		return // Already quantum
	}
	
	h.Version = QuantumBlockVersion
	h.Quantum = &QuantumMetadata{
		Layer:           layer,
		NetworkID:       networkID,
		ConsensusType:   consensusType,
		TimestampNano:   h.Time * 1e9, // Convert to nanoseconds
		MetadataVersion: 1,
	}
	
	// Cache the quantum metadata
	hash := h.Hash()
	quantumCacheMutex.Lock()
	quantumCache[hash] = h.Quantum
	quantumCacheMutex.Unlock()
}

// GetQuantumMetadata returns the quantum metadata for a block hash
func GetQuantumMetadata(hash common.Hash) *QuantumMetadata {
	quantumCacheMutex.RLock()
	defer quantumCacheMutex.RUnlock()
	return quantumCache[hash]
}

// SetQuantumMetadata sets the quantum metadata for a block hash
func SetQuantumMetadata(hash common.Hash, metadata *QuantumMetadata) {
	quantumCacheMutex.Lock()
	defer quantumCacheMutex.Unlock()
	quantumCache[hash] = metadata
}

// UnifiedBlock represents a block with unified header
type UnifiedBlock struct {
	header       *UnifiedHeader
	uncles       []*UnifiedHeader
	transactions Transactions
	withdrawals  Withdrawals
	
	// caches
	hash atomic.Pointer[common.Hash]
	size atomic.Uint64
	
	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// NewUnifiedBlock creates a new unified block
func NewUnifiedBlock(header *UnifiedHeader, txs Transactions, uncles []*UnifiedHeader, receipts Receipts, withdrawals Withdrawals) *UnifiedBlock {
	b := &UnifiedBlock{
		header:       CopyUnifiedHeader(header),
		transactions: make(Transactions, len(txs)),
		uncles:       make([]*UnifiedHeader, len(uncles)),
		withdrawals:  make(Withdrawals, len(withdrawals)),
	}
	copy(b.transactions, txs)
	copy(b.withdrawals, withdrawals)
	
	for i := range uncles {
		b.uncles[i] = CopyUnifiedHeader(uncles[i])
	}
	
	return b
}

// CopyUnifiedHeader creates a deep copy of a unified header
func CopyUnifiedHeader(h *UnifiedHeader) *UnifiedHeader {
	if h == nil {
		return nil
	}
	
	cpy := *h
	if h.Difficulty != nil {
		cpy.Difficulty = new(big.Int).Set(h.Difficulty)
	}
	if h.Number != nil {
		cpy.Number = new(big.Int).Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if h.Quantum != nil {
		qCpy := *h.Quantum
		if h.Quantum.BlockGasCost != nil {
			qCpy.BlockGasCost = new(big.Int).Set(h.Quantum.BlockGasCost)
		}
		cpy.Quantum = &qCpy
	}
	cpy.Extra = make([]byte, len(h.Extra))
	copy(cpy.Extra, h.Extra)
	
	return &cpy
}

// Header returns the block header (as UnifiedHeader)
func (b *UnifiedBlock) Header() *UnifiedHeader { return CopyUnifiedHeader(b.header) }

// Transactions returns the block transactions
func (b *UnifiedBlock) Transactions() Transactions { return b.transactions }

// Withdrawals returns the block withdrawals
func (b *UnifiedBlock) Withdrawals() Withdrawals { return b.withdrawals }

// Number returns the block number
func (b *UnifiedBlock) Number() *big.Int { return new(big.Int).Set(b.header.Number) }

// NumberU64 returns the block number as uint64
func (b *UnifiedBlock) NumberU64() uint64 { return b.header.Number.Uint64() }

// Time returns the block timestamp
func (b *UnifiedBlock) Time() uint64 { return b.header.Time }

// Hash returns the block hash
func (b *UnifiedBlock) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return *hash
	}
	h := b.header.Hash()
	b.hash.Store(&h)
	return h
}

// ParentHash returns the parent block hash
func (b *UnifiedBlock) ParentHash() common.Hash { return b.header.ParentHash }

// ToLegacyBlock converts UnifiedBlock to standard Block for compatibility
func (b *UnifiedBlock) ToLegacyBlock() *Block {
	legacyHeader := b.header.ToLegacyHeader()
	legacyUncles := make([]*Header, len(b.uncles))
	for i, uncle := range b.uncles {
		legacyUncles[i] = uncle.ToLegacyHeader()
	}
	
	body := &Body{
		Transactions: b.transactions,
		Uncles:       legacyUncles,
		Withdrawals:  b.withdrawals,
	}
	return NewBlock(legacyHeader, body, nil, nil)
}

// FromLegacyBlock creates UnifiedBlock from standard Block
func FromLegacyBlock(b *Block) *UnifiedBlock {
	if b == nil {
		return nil
	}
	
	unifiedHeader := FromLegacyHeader(b.header)
	unifiedUncles := make([]*UnifiedHeader, len(b.uncles))
	for i, uncle := range b.uncles {
		unifiedUncles[i] = FromLegacyHeader(uncle)
	}
	
	return &UnifiedBlock{
		header:       unifiedHeader,
		transactions: b.transactions,
		uncles:       unifiedUncles,
		withdrawals:  b.withdrawals,
		ReceivedAt:   b.ReceivedAt,
		ReceivedFrom: b.ReceivedFrom,
	}
}