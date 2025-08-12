// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/core/vm"
	"github.com/luxfi/geth/params"
)

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
	BlockHeader() (*vm.BlockHeader, error)
	BlockNumber() *big.Int
	BlockTime() uint64
	Addresses() *vm.Addresses
	ReadOnly() bool
	ReadOnlyState() StateReader
	StateDB() StateReader
	ChainConfig() *vm.ChainConfig
}

// Validator is an interface which defines the standard for block validation
type Validator interface {
	// ValidateBody validates the given block's content.
	ValidateBody(block *types.Block) error

	// ValidateState validates the given statedb and optionally the receipts and
	// gas used.
	ValidateState(block *types.Block, state *state.StateDB, receipts types.Receipts, usedGas uint64) error

	// ValidateWitness validates the given block's witness.
	ValidateWitness(witness *types.ExecutionWitness, receiptRoot common.Hash, stateRoot common.Hash) error
}

// Prefetcher is an interface for pre-caching transaction signatures and state.
type Prefetcher interface {
	// Prefetch processes the state changes according to the block and converts
	// the result into a data structure for concurrent access.
	Prefetch(block *types.Block, statedb *state.StateDB, cfg *params.ChainConfig, interrupt *uint32)
}

// Processor is an interface for processing blocks
type Processor interface {
	// Process processes the state changes according to the Ethereum rules by running
	// the transaction messages using the statedb and applying any rewards to both
	// the processor (coinbase) and any included uncles.
	Process(block *types.Block, statedb *state.StateDB, cfg *params.ChainConfig) (*ProcessResult, error)
}

// ProcessResult contains the result of processing a block
type ProcessResult struct {
	Receipts types.Receipts
	Requests [][]byte
	Logs     []*types.Log
	GasUsed  uint64
}