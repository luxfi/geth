package types

import (
	"fmt"
	"io"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

// LegacyHeader represents a SubnetEVM header (pre-Shanghai format)
// Has 18 fields total: 16 standard pre-Shanghai + 2 SubnetEVM-specific
// Missing only the 4 Shanghai fields: WithdrawalsHash, BlobGasUsed, ExcessBlobGas, ParentBeaconRoot
type LegacyHeader struct {
	ParentHash   common.Hash    `json:"parentHash"`
	UncleHash    common.Hash    `json:"sha3Uncles"`
	Coinbase     common.Address `json:"miner"`
	Root         common.Hash    `json:"stateRoot"`
	TxHash       common.Hash    `json:"transactionsRoot"`
	ReceiptHash  common.Hash    `json:"receiptsRoot"`
	Bloom        Bloom          `json:"logsBloom"`
	Difficulty   *big.Int       `json:"difficulty"`
	Number       *big.Int       `json:"number"`
	GasLimit     uint64         `json:"gasLimit"`
	GasUsed      uint64         `json:"gasUsed"`
	Time         uint64         `json:"timestamp"`
	Extra        []byte         `json:"extraData"`
	MixDigest    common.Hash    `json:"mixHash"`
	Nonce        BlockNonce     `json:"nonce"`
	BaseFee      *big.Int       `json:"baseFeePerGas"`
	ExtDataHash  common.Hash    `json:"extDataHash"`  // SubnetEVM specific
	BlockGasCost *big.Int       `json:"blockGasCost"` // SubnetEVM specific
}

// LegacyHeader17 represents actual SubnetEVM header format (17 fields)
// Pre-Shanghai + ExtDataHash (but no BlockGasCost)
// Note: ExtDataHash is often empty (bytes[0]), so we use []byte instead of common.Hash
type LegacyHeader17 struct {
	ParentHash  common.Hash    `json:"parentHash"`
	UncleHash   common.Hash    `json:"sha3Uncles"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"`
	TxHash      common.Hash    `json:"transactionsRoot"`
	ReceiptHash common.Hash    `json:"receiptsRoot"`
	Bloom       Bloom          `json:"logsBloom"`
	Difficulty  *big.Int       `json:"difficulty"`
	Number      *big.Int       `json:"number"`
	GasLimit    uint64         `json:"gasLimit"`
	GasUsed     uint64         `json:"gasUsed"`
	Time        uint64         `json:"timestamp"`
	Extra       []byte         `json:"extraData"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`
	BaseFee     *big.Int       `json:"baseFeePerGas"`
	ExtDataHash []byte         `json:"extDataHash"` // SubnetEVM specific (often empty)
}

// LegacyHeader16 represents a standard pre-Shanghai header (16 fields)
// No SubnetEVM extensions, just the standard Ethereum pre-Shanghai format
type LegacyHeader16 struct {
	ParentHash  common.Hash    `json:"parentHash"`
	UncleHash   common.Hash    `json:"sha3Uncles"`
	Coinbase    common.Address `json:"miner"`
	Root        common.Hash    `json:"stateRoot"`
	TxHash      common.Hash    `json:"transactionsRoot"`
	ReceiptHash common.Hash    `json:"receiptsRoot"`
	Bloom       Bloom          `json:"logsBloom"`
	Difficulty  *big.Int       `json:"difficulty"`
	Number      *big.Int       `json:"number"`
	GasLimit    uint64         `json:"gasLimit"`
	GasUsed     uint64         `json:"gasUsed"`
	Time        uint64         `json:"timestamp"`
	Extra       []byte         `json:"extraData"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"`
	BaseFee     *big.Int       `json:"baseFeePerGas"`
}

// ToHeader converts a LegacyHeader to a modern Header
func (lh *LegacyHeader) ToHeader() *Header {
	return &Header{
		ParentHash:  lh.ParentHash,
		UncleHash:   lh.UncleHash,
		Coinbase:    lh.Coinbase,
		Root:        lh.Root,
		TxHash:      lh.TxHash,
		ReceiptHash: lh.ReceiptHash,
		Bloom:       lh.Bloom,
		Difficulty:  lh.Difficulty,
		Number:      lh.Number,
		GasLimit:    lh.GasLimit,
		GasUsed:     lh.GasUsed,
		Time:        lh.Time,
		Extra:       lh.Extra,
		MixDigest:   lh.MixDigest,
		Nonce:       lh.Nonce,
		BaseFee:     lh.BaseFee,
		// New fields default to zero/nil
		WithdrawalsHash: nil,
		BlobGasUsed:     nil,
		ExcessBlobGas:   nil,
		ParentBeaconRoot: nil,
	}
}

// ToHeader converts a LegacyHeader17 to a modern Header
func (lh *LegacyHeader17) ToHeader() *Header {
	return &Header{
		ParentHash:  lh.ParentHash,
		UncleHash:   lh.UncleHash,
		Coinbase:    lh.Coinbase,
		Root:        lh.Root,
		TxHash:      lh.TxHash,
		ReceiptHash: lh.ReceiptHash,
		Bloom:       lh.Bloom,
		Difficulty:  lh.Difficulty,
		Number:      lh.Number,
		GasLimit:    lh.GasLimit,
		GasUsed:     lh.GasUsed,
		Time:        lh.Time,
		Extra:       lh.Extra,
		MixDigest:   lh.MixDigest,
		Nonce:       lh.Nonce,
		BaseFee:     lh.BaseFee,
		// New fields default to zero/nil
		WithdrawalsHash: nil,
		BlobGasUsed:     nil,
		ExcessBlobGas:   nil,
		ParentBeaconRoot: nil,
	}
}

// ToHeader converts a LegacyHeader16 to a modern Header
func (lh *LegacyHeader16) ToHeader() *Header {
	return &Header{
		ParentHash:  lh.ParentHash,
		UncleHash:   lh.UncleHash,
		Coinbase:    lh.Coinbase,
		Root:        lh.Root,
		TxHash:      lh.TxHash,
		ReceiptHash: lh.ReceiptHash,
		Bloom:       lh.Bloom,
		Difficulty:  lh.Difficulty,
		Number:      lh.Number,
		GasLimit:    lh.GasLimit,
		GasUsed:     lh.GasUsed,
		Time:        lh.Time,
		Extra:       lh.Extra,
		MixDigest:   lh.MixDigest,
		Nonce:       lh.Nonce,
		BaseFee:     lh.BaseFee,
		// New fields default to zero/nil
		WithdrawalsHash: nil,
		BlobGasUsed:     nil,
		ExcessBlobGas:   nil,
		ParentBeaconRoot: nil,
	}
}

// DecodeRLPWithLegacySupport tries to decode as modern Header first,
// falls back to LegacyHeader if that fails
func DecodeRLPWithLegacySupport(s *rlp.Stream) (*Header, error) {
	// Try modern format first
	var h Header
	err := s.Decode(&h)
	if err == nil {
		return &h, nil
	}

	// Check if error is about missing fields
	// This indicates it might be a legacy header
	if err.Error() != "rlp: input string too short for common.Hash, decoding into (types.Header).WithdrawalsHash" &&
		err.Error() != "rlp: input string too short" {
		// Different error, not a legacy header issue
		return nil, err
	}

	// Reset stream and try legacy format
	// Note: We can't actually reset the stream, so we need a different approach
	// We'll need to use raw bytes
	return nil, err
}

// DecodeRLPBytesWithLegacySupport decodes header bytes with backward compatibility
// Supports four formats:
// 1. Modern (22 fields): Full post-Shanghai header
// 2. Legacy 18-field: Pre-Shanghai + ExtDataHash + BlockGasCost
// 3. Legacy 17-field: Pre-Shanghai + ExtDataHash only (actual SubnetEVM format!)
// 4. Legacy 16-field: Pre-Shanghai only (standard Ethereum)
func DecodeRLPBytesWithLegacySupport(data []byte) (*Header, error) {
	// Try modern format first
	var h Header
	err := rlp.DecodeBytes(data, &h)
	if err == nil {
		fmt.Printf("🔍 Legacy decoder: SUCCESS with modern 22-field header\n")
		return &h, nil
	}
	fmt.Printf("🔍 Legacy decoder: 22-field failed: %v\n", err)

	// Try 18-field legacy format (with both SubnetEVM fields)
	var lh18 LegacyHeader
	err = rlp.DecodeBytes(data, &lh18)
	if err == nil {
		fmt.Printf("🔍 Legacy decoder: SUCCESS with 18-field header\n")
		return lh18.ToHeader(), nil
	}
	fmt.Printf("🔍 Legacy decoder: 18-field failed: %v\n", err)

	// Try 17-field legacy format (with ExtDataHash only - actual SubnetEVM!)
	var lh17 LegacyHeader17
	err = rlp.DecodeBytes(data, &lh17)
	if err == nil {
		fmt.Printf("🔍 Legacy decoder: SUCCESS with 17-field header\n")
		return lh17.ToHeader(), nil
	}
	fmt.Printf("🔍 Legacy decoder: 17-field failed: %v\n", err)

	// Try 16-field legacy format (standard pre-Shanghai)
	var lh16 LegacyHeader16
	err = rlp.DecodeBytes(data, &lh16)
	if err != nil {
		fmt.Printf("🔍 Legacy decoder: 16-field failed: %v\n", err)
		return nil, err
	}

	fmt.Printf("🔍 Legacy decoder: SUCCESS with 16-field header\n")
	// Convert to modern header
	return lh16.ToHeader(), nil
}

// EncodeRLP implements rlp.Encoder for LegacyHeader
func (lh *LegacyHeader) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, []interface{}{
		lh.ParentHash,
		lh.UncleHash,
		lh.Coinbase,
		lh.Root,
		lh.TxHash,
		lh.ReceiptHash,
		lh.Bloom,
		lh.Difficulty,
		lh.Number,
		lh.GasLimit,
		lh.GasUsed,
		lh.Time,
		lh.Extra,
		lh.MixDigest,
		lh.Nonce,
		lh.BaseFee,
		lh.ExtDataHash,
		lh.BlockGasCost,
	})
}

// DecodeRLP implements rlp.Decoder for LegacyHeader
func (lh *LegacyHeader) DecodeRLP(s *rlp.Stream) error {
	type legacyHeader struct {
		ParentHash   common.Hash
		UncleHash    common.Hash
		Coinbase     common.Address
		Root         common.Hash
		TxHash       common.Hash
		ReceiptHash  common.Hash
		Bloom        Bloom
		Difficulty   *big.Int
		Number       *big.Int
		GasLimit     uint64
		GasUsed      uint64
		Time         uint64
		Extra        []byte
		MixDigest    common.Hash
		Nonce        BlockNonce
		BaseFee      *big.Int
		ExtDataHash  common.Hash
		BlockGasCost *big.Int
	}

	var dec legacyHeader
	if err := s.Decode(&dec); err != nil {
		return err
	}

	lh.ParentHash = dec.ParentHash
	lh.UncleHash = dec.UncleHash
	lh.Coinbase = dec.Coinbase
	lh.Root = dec.Root
	lh.TxHash = dec.TxHash
	lh.ReceiptHash = dec.ReceiptHash
	lh.Bloom = dec.Bloom
	lh.Difficulty = dec.Difficulty
	lh.Number = dec.Number
	lh.GasLimit = dec.GasLimit
	lh.GasUsed = dec.GasUsed
	lh.Time = dec.Time
	lh.Extra = dec.Extra
	lh.MixDigest = dec.MixDigest
	lh.Nonce = dec.Nonce
	lh.BaseFee = dec.BaseFee
	lh.ExtDataHash = dec.ExtDataHash
	lh.BlockGasCost = dec.BlockGasCost

	return nil
}
