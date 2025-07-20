// Package common provides wrapper types for go-ethereum's common types
package common

import (
	"math/big"
	"github.com/ethereum/go-ethereum/common"
)

// Re-export constants
const (
	HashLength = common.HashLength
	AddressLength = common.AddressLength
)

// Re-export types
type (
	Hash = common.Hash
	Address = common.Address
	UnprefixedHash = common.UnprefixedHash
	UnprefixedAddress = common.UnprefixedAddress
	StorageSize = common.StorageSize
	MixedcaseAddress = common.MixedcaseAddress
)

// Re-export common big integers
var (
	Big0 = common.Big0
	Big1 = common.Big1
	Big2 = common.Big2
	Big3 = common.Big3
	Big32 = common.Big32
	Big256 = common.Big256
	Big257 = common.Big257
)

// Re-export functions
var (
	BytesToHash = common.BytesToHash
	BigToHash = common.BigToHash
	HexToHash = common.HexToHash
	BytesToAddress = common.BytesToAddress
	BigToAddress = common.BigToAddress
	HexToAddress = common.HexToAddress
	IsHexAddress = common.IsHexAddress
	Hex2Bytes = common.Hex2Bytes
	FromHex = common.FromHex
	CopyBytes = common.CopyBytes
	LeftPadBytes = common.LeftPadBytes
	RightPadBytes = common.RightPadBytes
	TrimLeftZeroes = common.TrimLeftZeroes
	TrimRightZeroes = common.TrimRightZeroes
	Bytes2Hex = common.Bytes2Hex
	NewMixedcaseAddress = common.NewMixedcaseAddress
)

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

// BigMax returns the larger of x or y.
func BigMax(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return x
	}
	return y
}

// BigMin returns the smaller of x or y.
func BigMin(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return x
	}
	return y
