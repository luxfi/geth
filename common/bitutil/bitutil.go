// Package bitutil provides wrapper types for go-ethereum's bitutil implementation
package bitutil

import (
	"github.com/luxfi/geth/common/bitutil"
)

// Re-export functions
var (
	XORBytes = bitutil.XORBytes
	ANDBytes = bitutil.ANDBytes
	ORBytes = bitutil.ORBytes
	TestBytes = bitutil.TestBytes
	CompressBytes = bitutil.CompressBytes
	DecompressBytes = bitutil.DecompressBytes
)