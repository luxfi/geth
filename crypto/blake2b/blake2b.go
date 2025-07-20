// Package blake2b provides wrapper types for go-ethereum's blake2b implementation
package blake2b

import (
	"github.com/ethereum/go-ethereum/crypto/blake2b"
)

// Re-export functions
var (
	F = blake2b.F
)
