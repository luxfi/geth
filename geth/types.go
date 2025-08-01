// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package geth

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/params"
)

// Type aliases for convenience
type Header = types.Header
type ChainConfig = params.ChainConfig

// Addresses contains common addresses used in precompiles
type Addresses struct {
	Coinbase common.Address
	Caller   common.Address
}

// AddressContext contains address information for contract creation
type AddressContext struct {
	Origin common.Address
	Self   common.Address
}

// StateReader interface for reading state
type StateReader interface {
	GetState(common.Address, common.Hash) common.Hash
	GetBalance(common.Address) *big.Int
	GetNonce(common.Address) uint64
	GetCode(common.Address) []byte
	GetCodeHash(common.Address) common.Hash
	GetCodeSize(common.Address) int
	Exist(common.Address) bool
	Empty(common.Address) bool
}

// PrecompiledContract represents a precompiled contract
type PrecompiledContract interface {
	Run(input []byte) ([]byte, error)
	RequiredGas(input []byte) uint64
}

// RulesHooks interface for chain rules extensions
type RulesHooks interface {
	CanCreateContract(ac *AddressContext, gas uint64, state StateReader) (uint64, error)
	CanExecuteTransaction(from common.Address, to *common.Address, state StateReader) error
	ActivePrecompiles(existing []common.Address) []common.Address
	MinimumGasConsumption(x uint64) uint64
	PrecompileOverride(addr common.Address) (PrecompiledContract, bool)
}

// PrecompileEnvironment provides the execution environment for precompiled contracts
type PrecompileEnvironment interface {
	BlockHeader() (*Header, error)
	BlockNumber() *big.Int
	BlockTime() uint64
	Addresses() *Addresses
	ReadOnly() bool
	ReadOnlyState() StateReader
	StateDB() StateReader
	ChainConfig() *ChainConfig
}