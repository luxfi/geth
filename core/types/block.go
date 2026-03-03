// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package types contains data types related to Ethereum consensus.
package types

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"slices"
	"sync/atomic"
	"time"

	"github.com/luxfi/crypto/verkle"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/common/hexutil"
	"github.com/luxfi/geth/rlp"
)

// A BlockNonce is a 64-bit hash which proves (combined with the
// mix-hash) that a sufficient amount of computation has been carried
// out on a block.
type BlockNonce [8]byte

// EncodeNonce converts the given integer to a block nonce.
func EncodeNonce(i uint64) BlockNonce {
	var n BlockNonce
	binary.BigEndian.PutUint64(n[:], i)
	return n
}

// Uint64 returns the integer value of a block nonce.
func (n BlockNonce) Uint64() uint64 {
	return binary.BigEndian.Uint64(n[:])
}

// MarshalText encodes n as a hex string with 0x prefix.
func (n BlockNonce) MarshalText() ([]byte, error) {
	return hexutil.Bytes(n[:]).MarshalText()
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (n *BlockNonce) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("BlockNonce", input, n[:])
}

// ExecutionWitness represents the witness + proof used in a verkle context,
// to provide the ability to execute a block statelessly.
type ExecutionWitness struct {
	StateDiff   verkle.StateDiff    `json:"stateDiff"`
	VerkleProof *verkle.VerkleProof `json:"verkleProof"`
}

//go:generate go run github.com/fjl/gencodec -type Header -field-override headerMarshaling -out gen_header_json.go
//go:generate go run ../../rlp/rlpgen -type Header -out gen_header_rlp.go

// Header represents a block header in the Lux C-chain (EVM-compatible).
//
// IMPORTANT: Lux C-Chain Header Format Requirements
//
// The Lux C-chain uses different header formats depending on the block:
//
//   - Genesis (Block 0): 16 fields (post-London, pre-ExtDataHash)
//   - Post-Genesis:      19 fields (with ExtDataHash, ExtDataGasUsed, BlockGasCost)
//   - Future:            20-24 fields (adding Ethereum 2.0 fields)
//
// Critical Network Parameters:
//
//	Chain ID:     96369
//	Genesis Hash: 0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
//	State Root:   0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80
//
// RLP Field Order (for hash computation):
//
//	Pos 0-14:  Core Ethereum fields (ParentHash through Nonce)
//	Pos 15:    BaseFee (EIP-1559, added in London)
//	Pos 16-18: Lux-specific fields (ExtDataHash, ExtDataGasUsed, BlockGasCost)
//	Pos 19+:   Ethereum 2.0 fields (BlobGasUsed, ExcessBlobGas, etc.)
//
// CRITICAL: ExtDataHash Type Difference from Original Coreth
//
// The original coreth implementation used common.Hash (value type) for ExtDataHash:
//
//	ExtDataHash common.Hash `rlp:"optional"`  // WRONG - original coreth
//
// Lux geth uses *common.Hash (pointer type) to allow proper nil encoding:
//
//	ExtDataHash *common.Hash `rlp:"optional"` // CORRECT - Lux geth
//
// This is intentional: pointer types encode as absent when nil, while value types
// encode as zero hash. The genesis block must NOT include ExtDataHash to produce
// the correct 16-field hash.
//
// Hash Methods:
//   - Hash():     Uses rawRLP if set, otherwise re-encodes (for decoded blocks)
//   - Hash16():   Computes 16-field hash for genesis verification
//
// See LLM.md for comprehensive documentation on header formats.
type Header struct {
	// Positions 0-14: Core Ethereum fields (unchanged since frontier)
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"` // Pos 0
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"` // Pos 1
	Coinbase    common.Address `json:"miner"`                                // Pos 2
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"` // Pos 3
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"` // Pos 4
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"` // Pos 5
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"` // Pos 6 (256 bytes)
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"` // Pos 7
	Number      *big.Int       `json:"number"           gencodec:"required"` // Pos 8
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"` // Pos 9
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"` // Pos 10
	Time        uint64         `json:"timestamp"        gencodec:"required"` // Pos 11
	Extra       []byte         `json:"extraData"        gencodec:"required"` // Pos 12
	MixDigest   common.Hash    `json:"mixHash"`                              // Pos 13
	Nonce       BlockNonce     `json:"nonce"`                                // Pos 14 (8 bytes)

	// Position 15: EIP-1559 (London fork)
	// Present in 16-field genesis AND all post-genesis blocks.
	BaseFee *big.Int `json:"baseFeePerGas" rlp:"optional"` // Pos 15

	// Positions 16-18: Lux/coreth-specific fields
	// NOT present in genesis (16-field format), present in post-genesis (19+ fields).
	// These MUST come before Ethereum 2.0 fields to maintain hash compatibility.
	//
	// IMPORTANT: ExtDataHash uses *common.Hash (pointer), NOT common.Hash (value).
	// Original coreth used value type, but Lux geth uses pointer for proper nil encoding.
	ExtDataHash    *common.Hash `json:"extDataHash" rlp:"optional"`    // Pos 16 (when present)
	ExtDataGasUsed *big.Int     `json:"extDataGasUsed" rlp:"optional"` // Pos 17 (when present)
	BlockGasCost   *big.Int     `json:"blockGasCost" rlp:"optional"`   // Pos 18 (when present)

	// Positions 19+: Ethereum 2.0 fields (Shanghai/Cancun/Prague)
	// Only present when chain upgrades to support these features.
	// Note: These come AFTER Lux fields, unlike standard Ethereum order.
	BlobGasUsed      *uint64      `json:"blobGasUsed" rlp:"optional"`           // Pos 19 (when present)
	ExcessBlobGas    *uint64      `json:"excessBlobGas" rlp:"optional"`         // Pos 20 (when present)
	ParentBeaconRoot *common.Hash `json:"parentBeaconBlockRoot" rlp:"optional"` // Pos 21 (when present)
	WithdrawalsHash  *common.Hash `json:"withdrawalsRoot" rlp:"optional"`       // Pos 22 (when present)
	RequestsHash     *common.Hash `json:"requestsHash" rlp:"optional"`          // Pos 23 (when present)

	// rawRLP stores the original RLP bytes for hash computation.
	// This ensures hash compatibility with blocks from different chain formats
	// (e.g., Lux C-chain 19-field format vs Ethereum 20+ field format).
	// When set, Hash() uses these bytes directly instead of re-encoding.
	rawRLP []byte `json:"-" rlp:"-"`

	// rlpFormat tracks the original RLP encoding format for this header.
	// Used to ensure re-encoding produces the same bytes as the original.
	rlpFormat RLPFormat `json:"-" rlp:"-"`
}

// field type overrides for gencodec
type headerMarshaling struct {
	Difficulty     *hexutil.Big
	Number         *hexutil.Big
	GasLimit       hexutil.Uint64
	GasUsed        hexutil.Uint64
	Time           hexutil.Uint64
	Extra          hexutil.Bytes
	BaseFee        *hexutil.Big
	ExtDataGasUsed *hexutil.Big
	BlockGasCost   *hexutil.Big
	Hash           common.Hash `json:"hash"` // adds call to Hash() in MarshalJSON
	BlobGasUsed    *hexutil.Uint64
	ExcessBlobGas  *hexutil.Uint64
}

// Hash returns the block hash of the header, which is simply the keccak256 hash of its
// RLP encoding.
//
// Special handling (in order of priority):
//  1. Genesis blocks (Number == 0): ALWAYS uses 16-field format to match original Lux
//     mainnet genesis hash. This takes priority over rawRLP because genesis blocks may
//     be stored with extra fork fields (WithdrawalsHash, etc.) but must hash correctly.
//  2. rawRLP set (from decoding): Uses stored bytes to preserve hash compatibility with
//     the original chain format for imported/historic blocks.
//  3. Lux blocks (ExtDataGasUsed or BlockGasCost set): Uses 19-field format with
//     ExtDataHash as value type to match original coreth encoding.
//  4. Default: Uses standard RLP encoding.
func (h *Header) Hash() common.Hash {
	// For blocks with rawRLP set (from decoding), use stored bytes to preserve hash
	if len(h.rawRLP) > 0 {
		return rlpHashBytes(h.rawRLP)
	}
	// Genesis block handling: detect format based on fields present
	if h.Number != nil && h.Number.Sign() == 0 {
		// Check if this is a standard Ethereum genesis (has Eth2 fields)
		if h.isEthFormat() {
			return h.HashEth()
		}
		// Lux mainnet genesis: use 16-field format
		return h.Hash16()
	}
	// Lux block detection: has ExtDataGasUsed or BlockGasCost set, but no Eth2 fields
	// For newly constructed blocks (no rawRLP), use Hash19() to match coreth format
	// This must match the logic in isLux19Format() and EncodeRLP()
	if h.isLux19Format() {
		return h.Hash19()
	}
	// Use Ethereum format for blocks with Eth2 fields but no Lux fields
	if h.isEthFormat() {
		return h.HashEth()
	}
	return rlpHash(h)
}

// hdr19val is the 19-field header format for Lux blocks with ExtDataHash as VALUE type.
// Original coreth used common.Hash (value), not *common.Hash (pointer).
// This affects RLP encoding: pointer nil -> empty string, value zero -> 32 zero bytes.
type hdr19val struct {
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
	ExtDataHash    common.Hash // VALUE type, not pointer - matches original coreth
	ExtDataGasUsed *big.Int
	BlockGasCost   *big.Int
}

// Hash19 returns the hash using 19-field Lux format with ExtDataHash as value type.
// This matches original coreth encoding where ExtDataHash was common.Hash (not pointer).
// When h.ExtDataHash is nil, encodes as zero hash (32 zero bytes).
func (h *Header) Hash19() common.Hash {
	// Convert pointer to value, defaulting nil to zero hash
	var extDataHash common.Hash
	if h.ExtDataHash != nil {
		extDataHash = *h.ExtDataHash
	}

	return rlpHash(&hdr19val{
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
		ExtDataHash:    extDataHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
	})
}

// hdrEth is the standard Ethereum header format (without Lux-specific fields).
// Used for computing hashes compatible with Ethereum mainnet.
type hdrEth struct {
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
	BaseFee          *big.Int     `rlp:"optional"`
	WithdrawalsHash  *common.Hash `rlp:"optional"`
	BlobGasUsed      *uint64      `rlp:"optional"`
	ExcessBlobGas    *uint64      `rlp:"optional"`
	ParentBeaconRoot *common.Hash `rlp:"optional"`
	RequestsHash     *common.Hash `rlp:"optional"`
}

// HashEth returns the hash using standard Ethereum header field order.
// This excludes Lux-specific fields (ExtDataHash, ExtDataGasUsed, BlockGasCost)
// and uses the Ethereum field order: BaseFee, WithdrawalsHash, BlobGasUsed,
// ExcessBlobGas, ParentBeaconRoot, RequestsHash.
func (h *Header) HashEth() common.Hash {
	return rlpHash(&hdrEth{
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
	})
}

// SetRawRLP sets the raw RLP bytes for hash computation.
// This is used when decoding blocks to preserve the original hash.
func (h *Header) SetRawRLP(raw []byte) {
	h.rawRLP = raw
}

// RawRLP returns the raw RLP bytes if set.
func (h *Header) RawRLP() []byte {
	return h.rawRLP
}

// EncodeRLP encodes the header to RLP using the appropriate format.
// The format is determined by rlpFormat (set during decode) or detected from fields.
//
// Format handling (in priority order):
//  1. rlpFormat set (from decode): Uses the format detected during decode
//  2. Genesis blocks (Number == 0): Uses format based on fields present
//  3. Lux 19-field format: If ExtDataGasUsed or BlockGasCost set (but no Eth2 fields)
//  4. Ethereum format: If Eth2 fields present but no Lux fields
//  5. Default: Standard encoding with all non-nil fields
func (h *Header) EncodeRLP(w io.Writer) error {
	switch h.rlpFormat {
	case RLPFormat17:
		return h.encodeRLP17(w)
	case RLPFormat17Eth:
		return h.encodeRLP17Eth(w)
	case RLPFormat18:
		return h.encodeRLP18(w)
	case RLPFormat19Lux:
		return h.encodeRLP19Lux(w)
	case RLPFormat20Eth:
		return h.encodeRLPEth(w)
	case RLPFormat21Eth:
		return h.encodeRLPEth(w)
	default:
		// Auto-detect format for new headers
		if h.isLux19Format() {
			return h.encodeRLP19Lux(w)
		}
		// Use Ethereum format for blocks with Eth2 fields but no Lux fields
		if h.isEthFormat() {
			return h.encodeRLPEth(w)
		}
		// Genesis blocks without Eth2 fields: use 16-field Lux format
		if h.Number != nil && h.Number.Sign() == 0 {
			return h.encodeRLP16(w)
		}
		return h.encodeRLPDefault(w)
	}
}

// isLux19Format returns true if this header appears to be a Lux 19-field format.
// Detection: has ExtDataGasUsed or BlockGasCost set, but no Ethereum 2.0 fields.
func (h *Header) isLux19Format() bool {
	hasLuxFields := h.ExtDataGasUsed != nil || h.BlockGasCost != nil
	hasEth2Fields := h.BlobGasUsed != nil || h.ExcessBlobGas != nil ||
		h.ParentBeaconRoot != nil || h.WithdrawalsHash != nil || h.RequestsHash != nil
	return hasLuxFields && !hasEth2Fields
}

// isEthFormat returns true if this header uses standard Ethereum format.
// Detection: has Eth2 fields but no Lux-specific fields.
func (h *Header) isEthFormat() bool {
	hasLuxFields := h.ExtDataHash != nil || h.ExtDataGasUsed != nil || h.BlockGasCost != nil
	hasEth2Fields := h.BlobGasUsed != nil || h.ExcessBlobGas != nil ||
		h.ParentBeaconRoot != nil || h.WithdrawalsHash != nil || h.RequestsHash != nil
	return hasEth2Fields && !hasLuxFields
}

// ClearInternalFields clears internal fields used for decode/encode operations.
// This is useful for testing when comparing headers after roundtrip encoding.
func (h *Header) ClearInternalFields() {
	h.rawRLP = nil
	h.rlpFormat = 0
}

// encodeRLP16 encodes using 16-field format (post-London, pre-ExtDataHash).
// This is used for Lux mainnet genesis to produce the correct hash.
func (h *Header) encodeRLP16(w io.Writer) error {
	return rlp.Encode(w, &hdr16{
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
	})
}

// encodeRLP17 encodes using 17-field format (BaseFee + ExtDataHash).
func (h *Header) encodeRLP17(w io.Writer) error {
	return rlp.Encode(w, &hdr17{
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
	})
}

// encodeRLP17Eth encodes using Ethereum Shanghai 17-field format.
// Field order: Core(15) + BaseFee(pos 15) + WithdrawalsHash(pos 16)
func (h *Header) encodeRLP17Eth(w io.Writer) error {
	return rlp.Encode(w, &hdrEth{
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
		WithdrawalsHash: h.WithdrawalsHash,
	})
}

// encodeRLP18 encodes using 18-field format (+ ExtDataGasUsed).
func (h *Header) encodeRLP18(w io.Writer) error {
	return rlp.Encode(w, &hdr18{
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
	})
}

// encodeRLP19Lux encodes using Lux 19-field format with value-type ExtDataHash.
func (h *Header) encodeRLP19Lux(w io.Writer) error {
	var extHash common.Hash
	if h.ExtDataHash != nil {
		extHash = *h.ExtDataHash
	}
	return rlp.Encode(w, &hdr19val{
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

// encodeRLPEth encodes using standard Ethereum format (no Lux-specific fields).
// Field order: BaseFee, WithdrawalsHash, BlobGasUsed, ExcessBlobGas, ParentBeaconRoot, RequestsHash
func (h *Header) encodeRLPEth(w io.Writer) error {
	return rlp.Encode(w, &hdrEth{
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
	})
}

// DecodeRLP decodes a header from RLP with format detection.
// Supports 15-21 field formats for legacy, London, Shanghai, and Lux chains.
// Stores the original RLP bytes for hash computation to ensure cryptographic
// continuity regardless of format differences during re-encoding.
func (h *Header) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	decoded, err := DecodeHeader(raw)
	if err != nil {
		return err
	}
	*h = *decoded
	// Store original RLP bytes for hash computation.
	// This ensures Hash() returns the correct hash even if re-encoding
	// would produce different bytes (e.g., different field order or format).
	h.rawRLP = raw
	return nil
}

// Hash16 returns the hash using 16-field format (post-London, pre-ExtDataHash).
// Used for Lux mainnet genesis compatibility.
func (h *Header) Hash16() common.Hash {
	return rlpHash(&hdr16{
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
	})
}

var headerSize = common.StorageSize(reflect.TypeFor[Header]().Size())

// Size returns the approximate memory used by all internal contents. It is used
// to approximate and limit the memory consumption of various caches.
func (h *Header) Size() common.StorageSize {
	var baseFeeBits, blockGasCostBits int
	if h.BaseFee != nil {
		baseFeeBits = h.BaseFee.BitLen()
	}
	if h.BlockGasCost != nil {
		blockGasCostBits = h.BlockGasCost.BitLen()
	}
	return headerSize + common.StorageSize(len(h.Extra)+(h.Difficulty.BitLen()+h.Number.BitLen()+baseFeeBits+blockGasCostBits)/8)
}

// SanityCheck checks a few basic things -- these checks are way beyond what
// any 'sane' production values should hold, and can mainly be used to prevent
// that the unbounded fields are stuffed with junk data to add processing
// overhead
func (h *Header) SanityCheck() error {
	if h.Number != nil && !h.Number.IsUint64() {
		return fmt.Errorf("too large block number: bitlen %d", h.Number.BitLen())
	}
	if h.Difficulty != nil {
		if diffLen := h.Difficulty.BitLen(); diffLen > 80 {
			return fmt.Errorf("too large block difficulty: bitlen %d", diffLen)
		}
	}
	if eLen := len(h.Extra); eLen > 100*1024 {
		return fmt.Errorf("too large block extradata: size %d", eLen)
	}
	if h.BaseFee != nil {
		if bfLen := h.BaseFee.BitLen(); bfLen > 256 {
			return fmt.Errorf("too large base fee: bitlen %d", bfLen)
		}
	}
	if h.BlockGasCost != nil {
		if bgcLen := h.BlockGasCost.BitLen(); bgcLen > 256 {
			return fmt.Errorf("too large block gas cost: bitlen %d", bgcLen)
		}
	}
	return nil
}

// EmptyBody returns true if there is no additional 'body' to complete the header
// that is: no transactions, no uncles and no withdrawals.
func (h *Header) EmptyBody() bool {
	var (
		emptyWithdrawals = h.WithdrawalsHash == nil || *h.WithdrawalsHash == EmptyWithdrawalsHash
	)
	return h.TxHash == EmptyTxsHash && h.UncleHash == EmptyUncleHash && emptyWithdrawals
}

// EmptyReceipts returns true if there are no receipts for this header/block.
func (h *Header) EmptyReceipts() bool {
	return h.ReceiptHash == EmptyReceiptsHash
}

// NumberU64 returns the block number as a uint64.
func (h *Header) NumberU64() uint64 {
	if h.Number == nil {
		return 0
	}
	return h.Number.Uint64()
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions and uncles) together.
type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
	Withdrawals  []*Withdrawal `rlp:"optional"`
}

// Block represents an Ethereum block.
//
// Note the Block type tries to be 'immutable', and contains certain caches that rely
// on that. The rules around block immutability are as follows:
//
//   - We copy all data when the block is constructed. This makes references held inside
//     the block independent of whatever value was passed in.
//
//   - We copy all header data on access. This is because any change to the header would mess
//     up the cached hash and size values in the block. Calling code is expected to take
//     advantage of this to avoid over-allocating!
//
//   - When new body data is attached to the block, a shallow copy of the block is returned.
//     This ensures block modifications are race-free.
//
//   - We do not copy body data on access because it does not affect the caches, and also
//     because it would be too expensive.
type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions
	withdrawals  Withdrawals

	// witness is not an encoded part of the block body.
	// It is held in Block in order for easy relaying to the places
	// that process it.
	witness *ExecutionWitness

	// caches
	hash atomic.Pointer[common.Hash]
	size atomic.Uint64

	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

// "external" block encoding. used for eth protocol, etc.
type extblock struct {
	Header      *Header
	Txs         []*Transaction
	Uncles      []*Header
	Withdrawals []*Withdrawal `rlp:"optional"`
}

// NewBlock creates a new block. The input data is copied, changes to header and to the
// field values will not affect the block.
//
// The body elements and the receipts are used to recompute and overwrite the
// relevant portions of the header.
//
// The receipt's bloom must already calculated for the block's bloom to be
// correctly calculated.
func NewBlock(header *Header, body *Body, receipts []*Receipt, hasher ListHasher) *Block {
	if body == nil {
		body = &Body{}
	}
	var (
		b           = NewBlockWithHeader(header)
		txs         = body.Transactions
		uncles      = body.Uncles
		withdrawals = body.Withdrawals
	)

	if len(txs) == 0 {
		b.header.TxHash = EmptyTxsHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs), hasher)
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	if len(receipts) == 0 {
		b.header.ReceiptHash = EmptyReceiptsHash
	} else {
		b.header.ReceiptHash = DeriveSha(Receipts(receipts), hasher)
		// Receipts must go through MakeReceipt to calculate the receipt's bloom
		// already. Merge the receipt's bloom together instead of recalculating
		// everything.
		b.header.Bloom = MergeBloom(receipts)
	}

	if len(uncles) == 0 {
		b.header.UncleHash = EmptyUncleHash
	} else {
		b.header.UncleHash = CalcUncleHash(uncles)
		b.uncles = make([]*Header, len(uncles))
		for i := range uncles {
			b.uncles[i] = CopyHeader(uncles[i])
		}
	}

	if withdrawals == nil {
		b.header.WithdrawalsHash = nil
	} else if len(withdrawals) == 0 {
		b.header.WithdrawalsHash = &EmptyWithdrawalsHash
		b.withdrawals = Withdrawals{}
	} else {
		hash := DeriveSha(Withdrawals(withdrawals), hasher)
		b.header.WithdrawalsHash = &hash
		b.withdrawals = slices.Clone(withdrawals)
	}

	return b
}

// CopyHeader creates a deep copy of a block header.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if h.BaseFee != nil {
		cpy.BaseFee = new(big.Int).Set(h.BaseFee)
	}
	if h.BlockGasCost != nil {
		cpy.BlockGasCost = new(big.Int).Set(h.BlockGasCost)
	}
	if h.ExtDataGasUsed != nil {
		cpy.ExtDataGasUsed = new(big.Int).Set(h.ExtDataGasUsed)
	}
	if h.ExtDataHash != nil {
		cpy.ExtDataHash = new(common.Hash)
		*cpy.ExtDataHash = *h.ExtDataHash
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	if h.WithdrawalsHash != nil {
		cpy.WithdrawalsHash = new(common.Hash)
		*cpy.WithdrawalsHash = *h.WithdrawalsHash
	}
	if h.ExcessBlobGas != nil {
		cpy.ExcessBlobGas = new(uint64)
		*cpy.ExcessBlobGas = *h.ExcessBlobGas
	}
	if h.BlobGasUsed != nil {
		cpy.BlobGasUsed = new(uint64)
		*cpy.BlobGasUsed = *h.BlobGasUsed
	}
	if h.ParentBeaconRoot != nil {
		cpy.ParentBeaconRoot = new(common.Hash)
		*cpy.ParentBeaconRoot = *h.ParentBeaconRoot
	}
	if h.RequestsHash != nil {
		cpy.RequestsHash = new(common.Hash)
		*cpy.RequestsHash = *h.RequestsHash
	}
	// Copy rawRLP to preserve hash computation capability
	if len(h.rawRLP) > 0 {
		cpy.rawRLP = make([]byte, len(h.rawRLP))
		copy(cpy.rawRLP, h.rawRLP)
	}
	// rlpFormat is already copied by value assignment (cpy := *h)
	return &cpy
}

// DecodeRLP decodes a block from RLP with format detection.
func (b *Block) DecodeRLP(s *rlp.Stream) error {
	raw, err := s.Raw()
	if err != nil {
		return err
	}
	block, err := DecodeBlock(raw)
	if err != nil {
		return err
	}
	b.header = block.header
	b.uncles = block.uncles
	b.transactions = block.transactions
	b.withdrawals = block.withdrawals
	b.size.Store(uint64(len(raw)))
	return nil
}

// legacyExtblock is used for decoding blocks with legacy (pre-Shanghai) headers
type legacyExtblock struct {
	Header      []byte // Raw header bytes for flexible decoding
	Txs         []*Transaction
	Uncles      []*Header
	Withdrawals []*Withdrawal `rlp:"optional"`
}

// DecodeBlockRLPWithLegacySupport decodes a block from RLP with support for legacy headers.
// This is used for importing blocks from older chains that use pre-Shanghai header format.
func DecodeBlockRLPWithLegacySupport(data []byte) (*Block, error) {
	// First try standard decoding
	var block Block
	if err := rlp.DecodeBytes(data, &block); err == nil {
		return &block, nil
	}

	// Try decoding with legacy header support
	// We need to decode the list items manually
	var items []rlp.RawValue
	if err := rlp.DecodeBytes(data, &items); err != nil {
		return nil, err
	}

	if len(items) < 3 {
		return nil, fmt.Errorf("invalid block: expected at least 3 items, got %d", len(items))
	}

	header, err := DecodeHeader(items[0])
	if err != nil {
		return nil, fmt.Errorf("header: %v", err)
	}

	var txs []*Transaction
	if err := rlp.DecodeBytes(items[1], &txs); err != nil {
		return nil, fmt.Errorf("transactions: %v", err)
	}

	var uncleRaws []rlp.RawValue
	if err := rlp.DecodeBytes(items[2], &uncleRaws); err != nil {
		return nil, fmt.Errorf("uncles: %v", err)
	}
	uncles := make([]*Header, len(uncleRaws))
	for i, raw := range uncleRaws {
		uncles[i], err = DecodeHeader(raw)
		if err != nil {
			return nil, fmt.Errorf("uncle %d: %v", i, err)
		}
	}

	// Decode withdrawals if present
	var withdrawals []*Withdrawal
	if len(items) > 3 {
		if err := rlp.DecodeBytes(items[3], &withdrawals); err != nil {
			// Withdrawals are optional, ignore decode errors
			withdrawals = nil
		}
	}

	return NewBlockWithHeader(header).WithBody(Body{
		Transactions: txs,
		Uncles:       uncles,
		Withdrawals:  withdrawals,
	}), nil
}

// EncodeRLP serializes a block as RLP.
func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &extblock{
		Header:      b.header,
		Txs:         b.transactions,
		Uncles:      b.uncles,
		Withdrawals: b.withdrawals,
	})
}

// Body returns the non-header content of the block.
// Note the returned data is not an independent copy.
func (b *Block) Body() *Body {
	return &Body{b.transactions, b.uncles, b.withdrawals}
}

// Accessors for body data. These do not return a copy because the content
// of the body slices does not affect the cached hash/size in block.

func (b *Block) Uncles() []*Header          { return b.uncles }
func (b *Block) Transactions() Transactions { return b.transactions }
func (b *Block) Withdrawals() Withdrawals   { return b.withdrawals }

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

// Header returns the block header (as a copy).
func (b *Block) Header() *Header {
	return CopyHeader(b.header)
}

// Header value accessors. These do copy!

func (b *Block) Number() *big.Int     { return new(big.Int).Set(b.header.Number) }
func (b *Block) GasLimit() uint64     { return b.header.GasLimit }
func (b *Block) GasUsed() uint64      { return b.header.GasUsed }
func (b *Block) Difficulty() *big.Int { return new(big.Int).Set(b.header.Difficulty) }
func (b *Block) Time() uint64         { return b.header.Time }

func (b *Block) NumberU64() uint64        { return b.header.Number.Uint64() }
func (b *Block) MixDigest() common.Hash   { return b.header.MixDigest }
func (b *Block) Nonce() uint64            { return binary.BigEndian.Uint64(b.header.Nonce[:]) }
func (b *Block) Bloom() Bloom             { return b.header.Bloom }
func (b *Block) Coinbase() common.Address { return b.header.Coinbase }
func (b *Block) Root() common.Hash        { return b.header.Root }
func (b *Block) ParentHash() common.Hash  { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash      { return b.header.TxHash }
func (b *Block) ReceiptHash() common.Hash { return b.header.ReceiptHash }
func (b *Block) UncleHash() common.Hash   { return b.header.UncleHash }
func (b *Block) Extra() []byte            { return common.CopyBytes(b.header.Extra) }

func (b *Block) BaseFee() *big.Int {
	if b.header.BaseFee == nil {
		return nil
	}
	return new(big.Int).Set(b.header.BaseFee)
}

func (b *Block) BeaconRoot() *common.Hash   { return b.header.ParentBeaconRoot }
func (b *Block) RequestsHash() *common.Hash { return b.header.RequestsHash }

func (b *Block) ExcessBlobGas() *uint64 {
	var excessBlobGas *uint64
	if b.header.ExcessBlobGas != nil {
		excessBlobGas = new(uint64)
		*excessBlobGas = *b.header.ExcessBlobGas
	}
	return excessBlobGas
}

func (b *Block) BlobGasUsed() *uint64 {
	var blobGasUsed *uint64
	if b.header.BlobGasUsed != nil {
		blobGasUsed = new(uint64)
		*blobGasUsed = *b.header.BlobGasUsed
	}
	return blobGasUsed
}

// ExecutionWitness returns the verkle execution witneess + proof for a block
func (b *Block) ExecutionWitness() *ExecutionWitness { return b.witness }

// Size returns the true RLP encoded storage size of the block, either by encoding
// and returning it, or returning a previously cached value.
func (b *Block) Size() uint64 {
	if size := b.size.Load(); size > 0 {
		return size
	}
	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(uint64(c))
	return uint64(c)
}

// SanityCheck can be used to prevent that unbounded fields are
// stuffed with junk data to add processing overhead
func (b *Block) SanityCheck() error {
	return b.header.SanityCheck()
}

type writeCounter uint64

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

func CalcUncleHash(uncles []*Header) common.Hash {
	if len(uncles) == 0 {
		return EmptyUncleHash
	}
	return rlpHash(uncles)
}

// CalcRequestsHash creates the block requestsHash value for a list of requests.
func CalcRequestsHash(requests [][]byte) common.Hash {
	h1, h2 := sha256.New(), sha256.New()
	var buf common.Hash
	for _, item := range requests {
		if len(item) > 1 { // skip items with only requestType and no data.
			h1.Reset()
			h1.Write(item)
			h2.Write(h1.Sum(buf[:0]))
		}
	}
	h2.Sum(buf[:0])
	return buf
}

// NewBlockWithHeader creates a block with the given header data. The
// header data is copied, changes to header and to the field values
// will not affect the block.
func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	return &Block{
		header:       CopyHeader(header),
		transactions: b.transactions,
		uncles:       b.uncles,
		withdrawals:  b.withdrawals,
		witness:      b.witness,
	}
}

// WithBody returns a new block with the original header and a deep copy of the
// provided body.
func (b *Block) WithBody(body Body) *Block {
	block := &Block{
		header:       b.header,
		transactions: slices.Clone(body.Transactions),
		uncles:       make([]*Header, len(body.Uncles)),
		withdrawals:  slices.Clone(body.Withdrawals),
		witness:      b.witness,
	}
	for i := range body.Uncles {
		block.uncles[i] = CopyHeader(body.Uncles[i])
	}
	return block
}

func (b *Block) WithWitness(witness *ExecutionWitness) *Block {
	return &Block{
		header:       b.header,
		transactions: b.transactions,
		uncles:       b.uncles,
		withdrawals:  b.withdrawals,
		witness:      witness,
	}
}

// Hash returns the keccak256 hash of b's header.
// The hash is computed on the first call and cached thereafter.
func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return *hash
	}
	h := b.header.Hash()
	b.hash.Store(&h)
	return h
}

// HashEth returns the keccak256 hash using standard Ethereum header format.
// This excludes Lux-specific fields and uses Ethereum field ordering.
// Not cached - use for compatibility checks with Ethereum mainnet only.
func (b *Block) HashEth() common.Hash {
	return b.header.HashEth()
}

type Blocks []*Block

// HeaderParentHashFromRLP returns the parentHash of an RLP-encoded
// header. If 'header' is invalid, the zero hash is returned.
func HeaderParentHashFromRLP(header []byte) common.Hash {
	// parentHash is the first list element.
	listContent, _, err := rlp.SplitList(header)
	if err != nil {
		return common.Hash{}
	}
	parentHash, _, err := rlp.SplitString(listContent)
	if err != nil {
		return common.Hash{}
	}
	if len(parentHash) != 32 {
		return common.Hash{}
	}
	return common.BytesToHash(parentHash)
}
