// Package compat provides compatibility shims for functions removed from go-ethereum
package compat

import (
	"crypto/elliptic"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/common/math"
)

// S256 returns the secp256k1 curve
func S256() elliptic.Curve {
	return ethcrypto.S256()
}

// SolidityVersion returns a hardcoded version since the function was removed
func SolidityVersion() string {
	return "0.8.26"
}

// CompileSolidityString is a stub for the removed function
func CompileSolidityString(source, solc string) (map[string]*Contract, error) {
	// This functionality was removed from go-ethereum
	// Projects should use the solc compiler directly
	return nil, nil
}

// CompileSolidity is a stub for the removed function  
func CompileSolidity(solc string, sourcefiles ...string) (map[string]*Contract, error) {
	// This functionality was removed from go-ethereum
	// Projects should use the solc compiler directly
	return nil, nil
}

// Contract represents a compiled contract (stub)
type Contract struct {
	Code string
	Info ContractInfo
}

// ContractInfo represents contract metadata (stub)
type ContractInfo struct {
	Source          string
	Language        string
	LanguageVersion string
	CompilerVersion string
	AbiDefinition   interface{}
	UserDoc         interface{}
	DeveloperDoc    interface{}
	Metadata        string
}

// BigMax returns the larger of x or y
func BigMax(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return new(big.Int).Set(x)
	}
	return new(big.Int).Set(y)
}

// BigMin returns the smaller of x or y
func BigMin(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return new(big.Int).Set(x)
	}
	return new(big.Int).Set(y)
}

// BigPow returns a ** b as a big integer
func BigPow(a, b int64) *big.Int {
	return math.BigPow(a, b)
}

// U256 encodes as a 256 bit two's complement number
func U256(x *big.Int) *big.Int {
	return math.U256(x)
}

// U256Bytes converts a big Int into a 256bit byte array
func U256Bytes(n *big.Int) []byte {
	return math.U256Bytes(n)
}

// S256Big interprets x as a two's complement number  
func S256Big(x *big.Int) *big.Int {
	if x.Cmp(tt255) < 0 {
		return x
	}
	return new(big.Int).Sub(x, tt256)
}

var (
	tt255 = new(big.Int).Lsh(big.NewInt(1), 255)
	tt256 = new(big.Int).Lsh(big.NewInt(1), 256)
)