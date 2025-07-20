// Package extheader provides Lux-specific header extensions
// This wrapper allows us to add Lux-specific fields while maintaining
// compatibility with upstream ethereum types.
package extheader

import (
	"math/big"

	"github.com/luxfi/geth/common"
	ethTypes "github.com/luxfi/geth/core/types" // the alias pkg!
)

// Header carries canonical header + Lux extras.
type Header struct {
	*ethTypes.Header                     // pointer-embed → no data copy
	ExtDataHash    common.Hash  `json:"extDataHash"      gencodec:"required"`
	ExtDataGasUsed *big.Int     `json:"extDataGasUsed"   rlp:"optional"`
	BlockGasCost   *big.Int     `json:"blockGasCost"     rlp:"optional"`
}

// --- helper constructors & up/down converters ---

// New creates a new extheader from a canonical header
func New(h *ethTypes.Header) *Header { 
	return &Header{Header: h} 
}

// Upstream returns the canonical header so shared code remains happy
func (h *Header) Upstream() *ethTypes.Header { 
	return h.Header 
}

// As converts a header to extheader, avoiding double-wrapping
func As(h *ethTypes.Header) *Header {
	if lux, ok := any(h).(*Header); ok { // already extended
		return lux
	}
	return New(h)
}

// Copy creates a deep copy of the header
func (h *Header) Copy() *Header {
	if h == nil {
		return nil
	}
	cpy := &Header{
		Header: ethTypes.CopyHeader(h.Header),
	}
	cpy.ExtDataHash = h.ExtDataHash
	if h.ExtDataGasUsed != nil {
		cpy.ExtDataGasUsed = new(big.Int).Set(h.ExtDataGasUsed)
	}
	if h.BlockGasCost != nil {
		cpy.BlockGasCost = new(big.Int).Set(h.BlockGasCost)
	}
	return cpy
}