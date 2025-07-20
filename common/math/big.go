// Package math provides wrapper types for go-ethereum's math implementation
package math

import (
	"github.com/ethereum/go-ethereum/common/math"
)

// Re-export types
type (
	HexOrDecimal256 = math.HexOrDecimal256
	HexOrDecimal64  = math.HexOrDecimal64
)

// Re-export functions
var (
	BigPow          = math.BigPow
	SafeSub         = math.SafeSub
	SafeAdd         = math.SafeAdd
	SafeMul         = math.SafeMul
	ParseBig256     = math.ParseBig256
	MustParseBig256 = math.MustParseBig256
	PaddedBigBytes  = math.PaddedBigBytes
	ReadBits        = math.ReadBits
	U256Bytes       = math.U256Bytes
)
