// Package rlp provides wrapper types for go-ethereum's rlp implementation
package rlp

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

// Re-export types
type (
	Stream   = rlp.Stream
	Encoder  = rlp.Encoder
	Decoder  = rlp.Decoder
	RawValue = rlp.RawValue
)

// Re-export functions
var (
	Encode           = rlp.Encode
	EncodeToBytes    = rlp.EncodeToBytes
	EncodeToReader   = rlp.EncodeToReader
	Decode           = rlp.Decode
	DecodeBytes      = rlp.DecodeBytes
	NewStream        = rlp.NewStream
	NewListStream    = rlp.NewListStream
	NewEncoderBuffer = rlp.NewEncoderBuffer
	ListSize         = rlp.ListSize
	AppendUint64     = rlp.AppendUint64
	List             = rlp.List
	Byte             = rlp.Byte
	BytesSize        = rlp.BytesSize
	SplitList        = rlp.SplitList
	Split            = rlp.Split
)

// Re-export constants
var (
	EmptyString = rlp.EmptyString
	EmptyList   = rlp.EmptyList
)

// Re-export errors
var (
	ErrNegativeBigInt = rlp.ErrNegativeBigInt
	ErrExpectedString = rlp.ErrExpectedString
	ErrExpectedList   = rlp.ErrExpectedList
	ErrCanonInt       = rlp.ErrCanonInt
	ErrCanonSize      = rlp.ErrCanonSize
	ErrElemTooLarge   = rlp.ErrElemTooLarge
	ErrValueTooLarge  = rlp.ErrValueTooLarge
)

// ByteReader wraps io.Reader
type ByteReader interface {
	io.Reader
	io.ByteReader
}
