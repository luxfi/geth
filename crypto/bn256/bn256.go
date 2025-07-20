// Package bn256 provides wrapper types for go-ethereum's bn256 implementation
package bn256

import (
	"github.com/ethereum/go-ethereum/crypto/bn256"
)

// Re-export types
type (
	G1 = bn256.G1
	G2 = bn256.G2
)

// GT is missing from newer versions, define it here
type GT struct{}

// Re-export functions
var (
	PairingCheck = bn256.PairingCheck
