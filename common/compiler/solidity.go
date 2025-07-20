// Package compiler provides compatibility layer for removed compiler functionality
package compiler

// Note: Solidity compilation was removed from go-ethereum
// Use the solc compiler directly or the compat package for stubs

import (
	"github.com/luxfi/geth/compat"
)

// Re-export types from compat
type (
	Contract     = compat.Contract
	ContractInfo = compat.ContractInfo
)

// Re-export functions from compat
var (
	CompileSolidityString = compat.CompileSolidityString
	CompileSolidity       = compat.CompileSolidity
	SolidityVersion       = compat.SolidityVersion
)