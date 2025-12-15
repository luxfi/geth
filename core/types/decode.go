// Copyright 2025 Lux Industries Inc
// Header and block decoding.

package types

import (
	"fmt"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

// DecodeHeader decodes a header from RLP bytes.
// Detects format by field count: 15 (pre-London), 16 (post-London), 17+ (extended).
func DecodeHeader(data []byte) (*Header, error) {
	fieldCount, err := countFields(data)
	if err != nil {
		return nil, fmt.Errorf("count fields: %w", err)
	}

	switch {
	case fieldCount == 15:
		return decode15(data)
	case fieldCount == 16:
		return decode16(data)
	case fieldCount >= 17:
		return decodeExt(data, fieldCount)
	default:
		return nil, fmt.Errorf("unsupported field count: %d", fieldCount)
	}
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
	ExtDataHash common.Hash
}

type hdr18 struct {
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
	ExtDataHash    common.Hash
	ExtDataGasUsed *big.Int
	BlockGasCost   *big.Int
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
		return decode20eth(data)
	case 21:
		return decode21eth(data)
	default:
		// Try standard Ethereum format first
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
		ExtDataHash: &h.ExtDataHash,
	}, nil
}

func decode18(data []byte) (*Header, error) {
	var h hdr18
	if err := rlp.DecodeBytes(data, &h); err != nil {
		return nil, err
	}
	return &Header{
		ParentHash:   h.ParentHash,
		UncleHash:    h.UncleHash,
		Coinbase:     h.Coinbase,
		Root:         h.Root,
		TxHash:       h.TxHash,
		ReceiptHash:  h.ReceiptHash,
		Bloom:        h.Bloom,
		Difficulty:   h.Difficulty,
		Number:       h.Number,
		GasLimit:     h.GasLimit,
		GasUsed:      h.GasUsed,
		Time:         h.Time,
		Extra:        h.Extra,
		MixDigest:    h.MixDigest,
		Nonce:        h.Nonce,
		BaseFee:      h.BaseFee,
		ExtDataHash:  &h.ExtDataHash,
		BlockGasCost: h.BlockGasCost,
	}, nil
}

func decode19(data []byte) (*Header, error) {
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
		ExtDataHash:    &h.ExtDataHash,
		ExtDataGasUsed: h.ExtDataGasUsed,
		BlockGasCost:   h.BlockGasCost,
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
