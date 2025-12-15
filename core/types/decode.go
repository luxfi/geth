// Copyright 2025 Lux Industries Inc
//
// Header and block decoding with multi-format support for Lux C-chain.
//
// Lux C-Chain Header Format Detection
//
// The Lux C-chain uses different RLP header formats depending on the block
// and chain configuration. This file provides automatic format detection
// and decoding for all supported formats.
//
// Supported Formats:
//
//	15 fields: Pre-London (legacy Ethereum, no BaseFee)
//	16 fields: Post-London (EIP-1559) - USED BY LUX MAINNET GENESIS
//	17 fields: Lux + ExtDataHash only
//	18 fields: Lux + ExtDataHash + ExtDataGasUsed
//	19 fields: Lux Full (+ BlockGasCost) - USED BY LUX POST-GENESIS BLOCKS
//	20 fields: Lux Extended OR Ethereum Shanghai
//	21 fields: Lux Extended OR Ethereum Cancun
//	22-24 fields: Lux Extended with full Ethereum 2.0 support
//
// Critical Network Parameters:
//
//	Chain ID:     96369
//	Genesis Hash: 0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
//	State Root:   0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80
//
// Format Detection Strategy:
//
// For ambiguous field counts (20, 21), the decoder tries Lux format first
// (ExtDataHash, ExtDataGasUsed, BlockGasCost before Ethereum 2.0 fields),
// then falls back to standard Ethereum format if decoding fails.
//
// Lux vs Ethereum Field Order (20 fields):
//
//	Standard Ethereum: [core] + BaseFee + WithdrawalsHash + BlobGasUsed + ExcessBlobGas + ParentBeaconRoot
//	Lux Extended:      [core] + BaseFee + ExtDataHash + ExtDataGasUsed + BlockGasCost + BlobGasUsed
//
// IMPORTANT: ExtDataHash Type Difference
//
// Original coreth used common.Hash (value type) for ExtDataHash, which encodes
// as a zero hash when "empty". Lux geth uses *common.Hash (pointer type) which
// encodes as absent (empty string) when nil. The decoder handles both cases.
//
// See LLM.md for comprehensive documentation on header formats.

package types

import (
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

// RLPFormat represents the RLP encoding format for headers.
// This is tracked to ensure hash consistency when re-encoding.
type RLPFormat uint8

const (
	RLPFormatUnknown RLPFormat = 0
	RLPFormat15      RLPFormat = 15 // Pre-London
	RLPFormat16      RLPFormat = 16 // Post-London (genesis)
	RLPFormat17      RLPFormat = 17 // With ExtDataHash (pointer)
	RLPFormat18      RLPFormat = 18 // With ExtDataGasUsed
	RLPFormat19Lux   RLPFormat = 19 // Lux 19-field (ExtDataHash as VALUE)
	RLPFormat19Ptr   RLPFormat = 29 // 19-field with ExtDataHash as pointer (internal)
	RLPFormat20      RLPFormat = 20 // Lux extended
	RLPFormat20Eth   RLPFormat = 30 // Ethereum Cancun
	RLPFormat21      RLPFormat = 21 // Lux extended
	RLPFormat21Eth   RLPFormat = 31 // Ethereum with RequestsHash
	RLPFormat22      RLPFormat = 22 // Lux extended
	RLPFormat23      RLPFormat = 23 // Lux extended
	RLPFormat24      RLPFormat = 24 // Lux extended
)

// GetRLPFormat returns the RLP encoding format used when this header was decoded.
// Returns RLPFormatUnknown if the header was not decoded from RLP.
func (h *Header) GetRLPFormat() RLPFormat {
	return h.rlpFormat
}

// SetRLPFormat sets the RLP encoding format for this header.
// This affects how the header is re-encoded.
func (h *Header) SetRLPFormat(f RLPFormat) {
	h.rlpFormat = f
}

// EncodeRLP19Lux encodes the header using Lux 19-field format.
// This format uses ExtDataHash as a VALUE type (common.Hash), not pointer.
// Returns the RLP-encoded bytes.
func (h *Header) EncodeRLP19Lux() ([]byte, error) {
	var extHash common.Hash
	if h.ExtDataHash != nil {
		extHash = *h.ExtDataHash
	}
	return rlp.EncodeToBytes(&hdr19lux{
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Coinbase:       h.Coinbase,
		Root:           h.Root,
		TxHash:         h.TxHash,
		ReceiptHash:    h.ReceiptHash,
		Bloom:          h.Bloom,
		Difficulty:     h.Difficulty,
		Number:         h.Number,
		GasLimit:       h.GasLimit,
		GasUsed:        h.GasUsed,
		Time:           h.Time,
		Extra:          h.Extra,
		MixDigest:      h.MixDigest,
		Nonce:          h.Nonce,
		BaseFee:        h.BaseFee,
		ExtDataHash:    extHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
	})
}

// Hash19Lux returns the hash using Lux 19-field format.
// This is used for Lux post-genesis blocks where ExtDataHash is a VALUE type.
func (h *Header) Hash19Lux() common.Hash {
	var extHash common.Hash
	if h.ExtDataHash != nil {
		extHash = *h.ExtDataHash
	}
	return rlpHash(&hdr19lux{
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Coinbase:       h.Coinbase,
		Root:           h.Root,
		TxHash:         h.TxHash,
		ReceiptHash:    h.ReceiptHash,
		Bloom:          h.Bloom,
		Difficulty:     h.Difficulty,
		Number:         h.Number,
		GasLimit:       h.GasLimit,
		GasUsed:        h.GasUsed,
		Time:           h.Time,
		Extra:          h.Extra,
		MixDigest:      h.MixDigest,
		Nonce:          h.Nonce,
		BaseFee:        h.BaseFee,
		ExtDataHash:    extHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
	})
}

// DecodeHeader decodes a header from RLP bytes.
// Detects format by field count: 15 (pre-London), 16 (post-London), 17+ (extended).
// The raw RLP bytes are stored for hash computation compatibility.
func DecodeHeader(data []byte) (*Header, error) {
	fieldCount, err := countFields(data)
	if err != nil {
		return nil, fmt.Errorf("count fields: %w", err)
	}

	var header *Header
	switch {
	case fieldCount == 15:
		header, err = decode15(data)
	case fieldCount == 16:
		header, err = decode16(data)
	case fieldCount >= 17:
		header, err = decodeExt(data, fieldCount)
	default:
		return nil, fmt.Errorf("unsupported field count: %d", fieldCount)
	}

	if err != nil {
		return nil, err
	}

	// Store raw RLP bytes for hash computation compatibility.
	// This ensures Hash() returns the same value as the original chain format.
	header.rawRLP = make([]byte, len(data))
	copy(header.rawRLP, data)

	return header, nil
}

// DecodeBlock decodes a block from RLP bytes.
func DecodeBlock(data []byte) (*Block, error) {
	var items []rlp.RawValue
	if err := rlp.DecodeBytes(data, &items); err != nil {
		return nil, fmt.Errorf("decode block list: %w", err)
	}

	if len(items) < 3 {
		return nil, fmt.Errorf("block: need 3+ items, got %d", len(items))
	}

	header, err := DecodeHeader(items[0])
	if err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	// Preserve raw header RLP for hash computation (ensures chain compatibility)
	header.SetRawRLP(items[0])

	var txs []*Transaction
	if err := rlp.DecodeBytes(items[1], &txs); err != nil {
		return nil, fmt.Errorf("transactions: %w", err)
	}

	var uncleRaws []rlp.RawValue
	if err := rlp.DecodeBytes(items[2], &uncleRaws); err != nil {
		return nil, fmt.Errorf("uncles: %w", err)
	}
	uncles := make([]*Header, len(uncleRaws))
	for i, raw := range uncleRaws {
		uncles[i], err = DecodeHeader(raw)
		if err != nil {
			return nil, fmt.Errorf("uncle %d: %w", i, err)
		}
		uncles[i].SetRawRLP(raw) // Preserve for hash computation
	}

	var withdrawals []*Withdrawal
	if len(items) > 3 {
		rlp.DecodeBytes(items[3], &withdrawals) // optional
	}

	block := NewBlockWithHeader(header).WithBody(Body{
		Transactions: txs,
		Uncles:       uncles,
		Withdrawals:  withdrawals,
	})
	block.size.Store(uint64(len(data)))
	return block, nil
}

func countFields(data []byte) (int, error) {
	content, _, err := rlp.SplitList(data)
	if err != nil {
		return 0, err
	}
	count := 0
	for len(content) > 0 {
		_, rest, err := rlp.SplitString(content)
		if err != nil {
			_, rest, err = rlp.SplitList(content) // bloom is a list
			if err != nil {
				return 0, err
			}
		}
		content = rest
		count++
	}
	return count, nil
}

type hdr15 struct {
	ParentHash  common.Hash
	UncleHash   common.Hash
	Coinbase    common.Address
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
	Bloom       Bloom
	Difficulty  *big.Int
	Number      *big.Int
	GasLimit    uint64
	GasUsed     uint64
	Time        uint64
	Extra       []byte
	MixDigest   common.Hash
	Nonce       BlockNonce
}

type hdr16 struct {
	ParentHash  common.Hash
	UncleHash   common.Hash
	Coinbase    common.Address
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
	Bloom       Bloom
	Difficulty  *big.Int
	Number      *big.Int
	GasLimit    uint64
	GasUsed     uint64
	Time        uint64
	Extra       []byte
	MixDigest   common.Hash
	Nonce       BlockNonce
	BaseFee     *big.Int
}

type hdr17 struct {
	ParentHash  common.Hash
	UncleHash   common.Hash
	Coinbase    common.Address
	Root        common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
	Bloom       Bloom
	Difficulty  *big.Int
	Number      *big.Int
	GasLimit    uint64
	GasUsed     uint64
	Time        uint64
	Extra       []byte
	MixDigest   common.Hash
	Nonce       BlockNonce
	BaseFee     *big.Int
	ExtDataHash *common.Hash `rlp:"nil"`
}

type hdr18 struct {
	ParentHash     common.Hash
	UncleHash      common.Hash
	Coinbase       common.Address
	Root           common.Hash
	TxHash         common.Hash
	ReceiptHash    common.Hash
	Bloom          Bloom
	Difficulty     *big.Int
	Number         *big.Int
	GasLimit       uint64
	GasUsed        uint64
	Time           uint64
	Extra          []byte
	MixDigest      common.Hash
	Nonce          BlockNonce
	BaseFee        *big.Int
	ExtDataHash    *common.Hash `rlp:"nil"`
	ExtDataGasUsed *big.Int
}

type hdr19 struct {
	ParentHash     common.Hash
	UncleHash      common.Hash
	Coinbase       common.Address
	Root           common.Hash
	TxHash         common.Hash
	ReceiptHash    common.Hash
	Bloom          Bloom
	Difficulty     *big.Int
	Number         *big.Int
	GasLimit       uint64
	GasUsed        uint64
	Time           uint64
	Extra          []byte
	MixDigest      common.Hash
	Nonce          BlockNonce
	BaseFee        *big.Int
	ExtDataHash    *common.Hash `rlp:"nil"`
	ExtDataGasUsed *big.Int
	BlockGasCost   *big.Int
}

// hdr19lux is an alias for hdr19val (defined in block.go).
// The Lux 19-field format uses ExtDataHash as VALUE type (common.Hash),
// not a pointer. This is the actual format used in Lux block production.
type hdr19lux = hdr19val

// Lux 20-field header (19 + BlobGasUsed)
type hdr20 struct {
	ParentHash     common.Hash
	UncleHash      common.Hash
	Coinbase       common.Address
	Root           common.Hash
	TxHash         common.Hash
	ReceiptHash    common.Hash
	Bloom          Bloom
	Difficulty     *big.Int
	Number         *big.Int
	GasLimit       uint64
	GasUsed        uint64
	Time           uint64
	Extra          []byte
	MixDigest      common.Hash
	Nonce          BlockNonce
	BaseFee        *big.Int
	ExtDataHash    *common.Hash `rlp:"nil"`
	ExtDataGasUsed *big.Int
	BlockGasCost   *big.Int
	BlobGasUsed    *uint64 `rlp:"nil"`
}

// Lux 21-field header (20 + ExcessBlobGas)
type hdr21 struct {
	ParentHash     common.Hash
	UncleHash      common.Hash
	Coinbase       common.Address
	Root           common.Hash
	TxHash         common.Hash
	ReceiptHash    common.Hash
	Bloom          Bloom
	Difficulty     *big.Int
	Number         *big.Int
	GasLimit       uint64
	GasUsed        uint64
	Time           uint64
	Extra          []byte
	MixDigest      common.Hash
	Nonce          BlockNonce
	BaseFee        *big.Int
	ExtDataHash    *common.Hash `rlp:"nil"`
	ExtDataGasUsed *big.Int
	BlockGasCost   *big.Int
	BlobGasUsed    *uint64 `rlp:"nil"`
	ExcessBlobGas  *uint64 `rlp:"nil"`
}

// Lux 22-field header (21 + ParentBeaconRoot)
type hdr22 struct {
	ParentHash       common.Hash
	UncleHash        common.Hash
	Coinbase         common.Address
	Root             common.Hash
	TxHash           common.Hash
	ReceiptHash      common.Hash
	Bloom            Bloom
	Difficulty       *big.Int
	Number           *big.Int
	GasLimit         uint64
	GasUsed          uint64
	Time             uint64
	Extra            []byte
	MixDigest        common.Hash
	Nonce            BlockNonce
	BaseFee          *big.Int
	ExtDataHash      *common.Hash `rlp:"nil"`
	ExtDataGasUsed   *big.Int
	BlockGasCost     *big.Int
	BlobGasUsed      *uint64 `rlp:"nil"`
	ExcessBlobGas    *uint64 `rlp:"nil"`
	ParentBeaconRoot *common.Hash `rlp:"nil"`
}

// Lux 23-field header (22 + WithdrawalsHash)
type hdr23 struct {
	ParentHash       common.Hash
	UncleHash        common.Hash
	Coinbase         common.Address
	Root             common.Hash
	TxHash           common.Hash
	ReceiptHash      common.Hash
	Bloom            Bloom
	Difficulty       *big.Int
	Number           *big.Int
	GasLimit         uint64
	GasUsed          uint64
	Time             uint64
	Extra            []byte
	MixDigest        common.Hash
	Nonce            BlockNonce
	BaseFee          *big.Int
	ExtDataHash      *common.Hash `rlp:"nil"`
	ExtDataGasUsed   *big.Int
	BlockGasCost     *big.Int
	BlobGasUsed      *uint64 `rlp:"nil"`
	ExcessBlobGas    *uint64 `rlp:"nil"`
	ParentBeaconRoot *common.Hash `rlp:"nil"`
	WithdrawalsHash  *common.Hash `rlp:"nil"`
}

// Lux 24-field header (23 + RequestsHash)
type hdr24 struct {
	ParentHash       common.Hash
	UncleHash        common.Hash
	Coinbase         common.Address
	Root             common.Hash
	TxHash           common.Hash
	ReceiptHash      common.Hash
	Bloom            Bloom
	Difficulty       *big.Int
	Number           *big.Int
	GasLimit         uint64
	GasUsed          uint64
	Time             uint64
	Extra            []byte
	MixDigest        common.Hash
	Nonce            BlockNonce
	BaseFee          *big.Int
	ExtDataHash      *common.Hash `rlp:"nil"`
	ExtDataGasUsed   *big.Int
	BlockGasCost     *big.Int
	BlobGasUsed      *uint64 `rlp:"nil"`
	ExcessBlobGas    *uint64 `rlp:"nil"`
	ParentBeaconRoot *common.Hash `rlp:"nil"`
	WithdrawalsHash  *common.Hash `rlp:"nil"`
	RequestsHash     *common.Hash `rlp:"nil"`
}

// Standard Ethereum 20-field header (EIP-4844: Shanghai + Cancun)
type hdr20eth struct {
	ParentHash       common.Hash
	UncleHash        common.Hash
	Coinbase         common.Address
	Root             common.Hash
	TxHash           common.Hash
	ReceiptHash      common.Hash
	Bloom            Bloom
	Difficulty       *big.Int
	Number           *big.Int
	GasLimit         uint64
	GasUsed          uint64
	Time             uint64
	Extra            []byte
	MixDigest        common.Hash
	Nonce            BlockNonce
	BaseFee          *big.Int
	WithdrawalsHash  common.Hash
	BlobGasUsed      uint64
	ExcessBlobGas    uint64
	ParentBeaconRoot common.Hash
}

// Standard Ethereum 21-field header (EIP-7685: requests)
type hdr21eth struct {
	ParentHash       common.Hash
	UncleHash        common.Hash
	Coinbase         common.Address
	Root             common.Hash
	TxHash           common.Hash
	ReceiptHash      common.Hash
	Bloom            Bloom
	Difficulty       *big.Int
	Number           *big.Int
	GasLimit         uint64
	GasUsed          uint64
	Time             uint64
	Extra            []byte
	MixDigest        common.Hash
	Nonce            BlockNonce
	BaseFee          *big.Int
	WithdrawalsHash  common.Hash
	BlobGasUsed      uint64
	ExcessBlobGas    uint64
	ParentBeaconRoot common.Hash
	RequestsHash     common.Hash
}

func decode15(data []byte) (*Header, error) {
	var h hdr15
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:  h.ParentHash,
		UncleHash:   h.UncleHash,
		Coinbase:    h.Coinbase,
		Root:        h.Root,
		TxHash:      h.TxHash,
		ReceiptHash: h.ReceiptHash,
		Bloom:       h.Bloom,
		Difficulty:  h.Difficulty,
		Number:      h.Number,
		GasLimit:    h.GasLimit,
		GasUsed:     h.GasUsed,
		Time:        h.Time,
		Extra:       h.Extra,
		MixDigest:   h.MixDigest,
		Nonce:       h.Nonce,
	}, nil
}

func decode16(data []byte) (*Header, error) {
	var h hdr16
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:  h.ParentHash,
		UncleHash:   h.UncleHash,
		Coinbase:    h.Coinbase,
		Root:        h.Root,
		TxHash:      h.TxHash,
		ReceiptHash: h.ReceiptHash,
		Bloom:       h.Bloom,
		Difficulty:  h.Difficulty,
		Number:      h.Number,
		GasLimit:    h.GasLimit,
		GasUsed:     h.GasUsed,
		Time:        h.Time,
		Extra:       h.Extra,
		MixDigest:   h.MixDigest,
		Nonce:       h.Nonce,
		BaseFee:     h.BaseFee,
	}, nil
}

func decodeExt(data []byte, count int) (*Header, error) {
	switch count {
	case 17:
		return decode17(data)
	case 18:
		return decode18(data)
	case 19:
		return decode19(data)
	case 20:
		// Try Lux format first, then standard Ethereum
		if h, err := decode20(data); err == nil {
			return h, nil
		}
		return decode20eth(data)
	case 21:
		// Try Lux format first, then standard Ethereum
		if h, err := decode21(data); err == nil {
			return h, nil
		}
		return decode21eth(data)
	case 22:
		return decode22(data)
	case 23:
		return decode23(data)
	case 24:
		return decode24(data)
	default:
		// Try Lux extended formats first
		if h, err := decode24(data); err == nil {
			return h, nil
		}
		if h, err := decode23(data); err == nil {
			return h, nil
		}
		// Then try standard Ethereum format
		if h, err := decode21eth(data); err == nil {
			return h, nil
		}
		if h, err := decode20eth(data); err == nil {
			return h, nil
		}
		return nil, fmt.Errorf("unsupported field count: %d", count)
	}
}

func decode17(data []byte) (*Header, error) {
	var h hdr17
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:  h.ParentHash,
		UncleHash:   h.UncleHash,
		Coinbase:    h.Coinbase,
		Root:        h.Root,
		TxHash:      h.TxHash,
		ReceiptHash: h.ReceiptHash,
		Bloom:       h.Bloom,
		Difficulty:  h.Difficulty,
		Number:      h.Number,
		GasLimit:    h.GasLimit,
		GasUsed:     h.GasUsed,
		Time:        h.Time,
		Extra:       h.Extra,
		MixDigest:   h.MixDigest,
		Nonce:       h.Nonce,
		BaseFee:     h.BaseFee,
		ExtDataHash: h.ExtDataHash,
		rlpFormat:   RLPFormat17, // Track 17-field format for re-encoding
	}, nil
}

func decode18(data []byte) (*Header, error) {
	var h hdr18
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Coinbase:       h.Coinbase,
		Root:           h.Root,
		TxHash:         h.TxHash,
		ReceiptHash:    h.ReceiptHash,
		Bloom:          h.Bloom,
		Difficulty:     h.Difficulty,
		Number:         h.Number,
		GasLimit:       h.GasLimit,
		GasUsed:        h.GasUsed,
		Time:           h.Time,
		Extra:          h.Extra,
		MixDigest:      h.MixDigest,
		Nonce:          h.Nonce,
		BaseFee:        h.BaseFee,
		ExtDataHash:    h.ExtDataHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		rlpFormat:      RLPFormat18, // Track 18-field format for re-encoding
	}, nil
}

func decode19(data []byte) (*Header, error) {
	// Try Lux value-type format first (ExtDataHash as common.Hash, not pointer)
	var hlux hdr19lux
	if err := rlp.DecodeBytes(data, &hlux); err == nil {
		extHash := hlux.ExtDataHash // Copy to get pointer
		return &Header{
			ParentHash:     hlux.ParentHash,
			UncleHash:      hlux.UncleHash,
			Coinbase:       hlux.Coinbase,
			Root:           hlux.Root,
			TxHash:         hlux.TxHash,
			ReceiptHash:    hlux.ReceiptHash,
			Bloom:          hlux.Bloom,
			Difficulty:     hlux.Difficulty,
			Number:         hlux.Number,
			GasLimit:       hlux.GasLimit,
			GasUsed:        hlux.GasUsed,
			Time:           hlux.Time,
			Extra:          hlux.Extra,
			MixDigest:      hlux.MixDigest,
			Nonce:          hlux.Nonce,
			BaseFee:        hlux.BaseFee,
			ExtDataHash:    &extHash,
			ExtDataGasUsed: hlux.ExtDataGasUsed,
			BlockGasCost:   hlux.BlockGasCost,
			rlpFormat:      RLPFormat19Lux, // Track original format
		}, nil
	}

	// Fall back to pointer format
	var h hdr19
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Coinbase:       h.Coinbase,
		Root:           h.Root,
		TxHash:         h.TxHash,
		ReceiptHash:    h.ReceiptHash,
		Bloom:          h.Bloom,
		Difficulty:     h.Difficulty,
		Number:         h.Number,
		GasLimit:       h.GasLimit,
		GasUsed:        h.GasUsed,
		Time:           h.Time,
		Extra:          h.Extra,
		MixDigest:      h.MixDigest,
		Nonce:          h.Nonce,
		BaseFee:        h.BaseFee,
		ExtDataHash:    h.ExtDataHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
		rlpFormat:      RLPFormat19Ptr, // Track original format
	}, nil
}

func decode20(data []byte) (*Header, error) {
	var h hdr20
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Coinbase:       h.Coinbase,
		Root:           h.Root,
		TxHash:         h.TxHash,
		ReceiptHash:    h.ReceiptHash,
		Bloom:          h.Bloom,
		Difficulty:     h.Difficulty,
		Number:         h.Number,
		GasLimit:       h.GasLimit,
		GasUsed:        h.GasUsed,
		Time:           h.Time,
		Extra:          h.Extra,
		MixDigest:      h.MixDigest,
		Nonce:          h.Nonce,
		BaseFee:        h.BaseFee,
		ExtDataHash:    h.ExtDataHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
		BlobGasUsed:    h.BlobGasUsed,
	}, nil
}

func decode21(data []byte) (*Header, error) {
	var h hdr21
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:     h.ParentHash,
		UncleHash:      h.UncleHash,
		Coinbase:       h.Coinbase,
		Root:           h.Root,
		TxHash:         h.TxHash,
		ReceiptHash:    h.ReceiptHash,
		Bloom:          h.Bloom,
		Difficulty:     h.Difficulty,
		Number:         h.Number,
		GasLimit:       h.GasLimit,
		GasUsed:        h.GasUsed,
		Time:           h.Time,
		Extra:          h.Extra,
		MixDigest:      h.MixDigest,
		Nonce:          h.Nonce,
		BaseFee:        h.BaseFee,
		ExtDataHash:    h.ExtDataHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
		BlobGasUsed:    h.BlobGasUsed,
		ExcessBlobGas:  h.ExcessBlobGas,
	}, nil
}

func decode22(data []byte) (*Header, error) {
	var h hdr22
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
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
		ExtDataHash:      h.ExtDataHash,
		ExtDataGasUsed:   h.ExtDataGasUsed,
		BlockGasCost:     h.BlockGasCost,
		BlobGasUsed:      h.BlobGasUsed,
		ExcessBlobGas:    h.ExcessBlobGas,
		ParentBeaconRoot: h.ParentBeaconRoot,
	}, nil
}

func decode23(data []byte) (*Header, error) {
	var h hdr23
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
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
		ExtDataHash:      h.ExtDataHash,
		ExtDataGasUsed:   h.ExtDataGasUsed,
		BlockGasCost:     h.BlockGasCost,
		BlobGasUsed:      h.BlobGasUsed,
		ExcessBlobGas:    h.ExcessBlobGas,
		ParentBeaconRoot: h.ParentBeaconRoot,
		WithdrawalsHash:  h.WithdrawalsHash,
	}, nil
}

func decode24(data []byte) (*Header, error) {
	var h hdr24
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
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
		ExtDataHash:      h.ExtDataHash,
		ExtDataGasUsed:   h.ExtDataGasUsed,
		BlockGasCost:     h.BlockGasCost,
		BlobGasUsed:      h.BlobGasUsed,
		ExcessBlobGas:    h.ExcessBlobGas,
		ParentBeaconRoot: h.ParentBeaconRoot,
		WithdrawalsHash:  h.WithdrawalsHash,
		RequestsHash:     h.RequestsHash,
	}, nil
}

func decode20eth(data []byte) (*Header, error) {
	var h hdr20eth
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	blobGasUsed := h.BlobGasUsed
	excessBlobGas := h.ExcessBlobGas
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
		WithdrawalsHash:  &h.WithdrawalsHash,
		BlobGasUsed:      &blobGasUsed,
		ExcessBlobGas:    &excessBlobGas,
		ParentBeaconRoot: &h.ParentBeaconRoot,
	}, nil
}

func decode21eth(data []byte) (*Header, error) {
	var h hdr21eth
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	blobGasUsed := h.BlobGasUsed
	excessBlobGas := h.ExcessBlobGas
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
		WithdrawalsHash:  &h.WithdrawalsHash,
		BlobGasUsed:      &blobGasUsed,
		ExcessBlobGas:    &excessBlobGas,
		ParentBeaconRoot: &h.ParentBeaconRoot,
		RequestsHash:     &h.RequestsHash,
	}, nil
}
