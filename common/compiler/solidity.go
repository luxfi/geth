// Package compiler provides wrapper types for go-ethereum's compiler implementation
package compiler

import (
	"github.com/luxfi/geth/common/compiler"
)

// Re-export types
type (
	Contract = compiler.Contract
	ContractInfo = compiler.ContractInfo
)

// Re-export functions
var (
	CompileSolidityString = compiler.CompileSolidityString
	CompileSolidity = compiler.CompileSolidity
	ParseCombinedJSON = compiler.ParseCombinedJSON
)