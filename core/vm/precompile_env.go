// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/params"
)

// PrecompileEnvironment provides the execution environment for precompiled contracts
type PrecompileEnvironment interface {
	BlockHeader() (*types.Header, error)
	BlockNumber() *big.Int
	BlockTime() uint64
	Addresses() *Addresses
	ReadOnly() bool
	ReadOnlyState() StateReader
	StateDB() StateDB
	ChainConfig() *params.ChainConfig
}

// Addresses contains common addresses used in precompiles
type Addresses struct {
	Coinbase common.Address
	Caller   common.Address
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

// StatefulPrecompiledContract represents a precompiled contract with state access
type StatefulPrecompiledContract interface {
	PrecompiledContract
	RunStateful(env PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error)
}