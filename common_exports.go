// (c) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package geth

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
)

// Re-export common types and constants
const (
	HashLength    = common.HashLength
	AddressLength = common.AddressLength
)

// Common types
type (
	Hash              = common.Hash
	Address           = common.Address
	UnprefixedHash    = common.UnprefixedHash
	UnprefixedAddress = common.UnprefixedAddress
	StorageSize       = common.StorageSize
	MixedcaseAddress  = common.MixedcaseAddress
	PrettyDuration    = common.PrettyDuration
	PrettyAge         = common.PrettyAge
	PrettyBytes       = common.PrettyBytes
)

// Big integers
var (
	Big0   = common.Big0
	Big1   = common.Big1
	Big2   = common.Big2
	Big3   = common.Big3
	Big32  = common.Big32
	Big256 = common.Big256
	Big257 = common.Big257
)

// Common functions
var (
	BytesToHash         = common.BytesToHash
	BigToHash           = common.BigToHash
	HexToHash           = common.HexToHash
	BytesToAddress      = common.BytesToAddress
	BigToAddress        = common.BigToAddress
	HexToAddress        = common.HexToAddress
	IsHexAddress        = common.IsHexAddress
	Hex2Bytes           = common.Hex2Bytes
	FromHex             = common.FromHex
	CopyBytes           = common.CopyBytes
	LeftPadBytes        = common.LeftPadBytes
	RightPadBytes       = common.RightPadBytes
	TrimLeftZeroes      = common.TrimLeftZeroes
	TrimRightZeroes     = common.TrimRightZeroes
	Bytes2Hex           = common.Bytes2Hex
	NewMixedcaseAddress = common.NewMixedcaseAddress
	LoadJSON            = common.LoadJSON
	FileExist           = common.FileExist
	AbsolutePath        = common.AbsolutePath
)

// Hexutil exports
var (
	Encode           = hexutil.Encode
	EncodeBig        = hexutil.EncodeBig
	EncodeUint64     = hexutil.EncodeUint64
	Decode           = hexutil.Decode
	DecodeBig        = hexutil.DecodeBig
	DecodeUint64     = hexutil.DecodeUint64
	MustDecode       = hexutil.MustDecode
	MustDecodeBig    = hexutil.MustDecodeBig
	MustDecodeUint64 = hexutil.MustDecodeUint64
)

// Hexutil types
type (
	HexBytes  = hexutil.Bytes
	HexBig    = hexutil.Big
	HexUint   = hexutil.Uint
	HexUint64 = hexutil.Uint64
)

// Math exports
var (
	BigPow         = math.BigPow
	PaddedBigBytes = math.PaddedBigBytes
	ReadBits       = math.ReadBits
)

// Math types
type HexOrDecimal64 = math.HexOrDecimal64

// Compiler types - removed, use compat package for compiler functionality

// Bitutil exports
var (
	CompressBytes   = bitutil.CompressBytes
	DecompressBytes = bitutil.DecompressBytes
	XORBytes        = bitutil.XORBytes
	ANDBytes        = bitutil.ANDBytes
	ORBytes         = bitutil.ORBytes
	TestBytes       = bitutil.TestBytes
)

// LRU cache types - use github.com/ethereum/go-ethereum/common/lru directly

// Prque exports - use github.com/ethereum/go-ethereum/common/prque directly

// IsHex validates whether each byte is valid hexadecimal string.
func IsHex(str string) bool {
	if len(str)%2 != 0 {
		return false
	}
	for _, c := range []byte(str) {
		if !isHexCharacter(c) {
			return false
		}
	}
	return true
}

// isHexCharacter returns bool of c being a valid hexadecimal.
func isHexCharacter(c byte) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

