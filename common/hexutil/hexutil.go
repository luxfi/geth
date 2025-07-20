// Package hexutil provides wrapper types for go-ethereum's hexutil implementation
package hexutil

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Re-export types
type (
	Big = hexutil.Big
	Uint64 = hexutil.Uint64
	Uint = hexutil.Uint
	Bytes = hexutil.Bytes
)

// Re-export errors
var (
	ErrEmptyString = hexutil.ErrEmptyString
	ErrSyntax = hexutil.ErrSyntax
	ErrMissingPrefix = hexutil.ErrMissingPrefix
	ErrOddLength = hexutil.ErrOddLength
	ErrEmptyNumber = hexutil.ErrEmptyNumber
	ErrLeadingZero = hexutil.ErrLeadingZero
	ErrUint64Range = hexutil.ErrUint64Range
	ErrUintRange = hexutil.ErrUintRange
	ErrBig256Range = hexutil.ErrBig256Range
)

// Re-export functions
var (
	Decode = hexutil.Decode
	MustDecode = hexutil.MustDecode
	Encode = hexutil.Encode
	EncodeBig = hexutil.EncodeBig
	DecodeBig = hexutil.DecodeBig
	MustDecodeBig = hexutil.MustDecodeBig
	DecodeUint64 = hexutil.DecodeUint64
	MustDecodeUint64 = hexutil.MustDecodeUint64
	EncodeUint64 = hexutil.EncodeUint64
	UnmarshalFixedJSON = hexutil.UnmarshalFixedJSON
	UnmarshalFixedText = hexutil.UnmarshalFixedText
	UnmarshalFixedUnprefixedText = hexutil.UnmarshalFixedUnprefixedText
)