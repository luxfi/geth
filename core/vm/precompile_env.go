// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/params"
)

// PrecompileEnvironment is the interface for precompile execution environment.
// This is a Lux extension for stateful precompiles that need access to
// the execution context.
type PrecompileEnvironment interface {
	// BlockHeader returns the block header
	BlockHeader() (*types.Header, error)
	// Rules returns the chain rules
	Rules() params.Rules
	// BlockNumber returns the current block number
	BlockNumber() *big.Int
	// BlockTime returns the current block timestamp
	BlockTime() uint64
	// Addresses returns the caller and contract addresses
	Addresses() PrecompileAddresses
	// ReadOnly returns whether the environment is read-only
	ReadOnly() bool
	// ChainConfig returns the chain config
	ChainConfig() *params.ChainConfig
	// StateDB returns the state database
	StateDB() StateDB
	// ReadOnlyState returns a read-only state view
	ReadOnlyState() StateDB
	// UseGas deducts gas from the available gas
	UseGas(gas uint64) bool
	// Gas returns the available gas
	Gas() uint64
	// Call executes a call to another contract
	Call(addr common.Address, input []byte, gas uint64, value *big.Int, opts ...CallOption) ([]byte, uint64, error)
}

// CallOption is an option for Call
type CallOption func(*CallConfig)

// CallConfig holds call configuration
type CallConfig struct {
	ProxyCaller bool
}

// WithUNSAFECallerAddressProxying returns an option to proxy caller address
func WithUNSAFECallerAddressProxying() CallOption {
	return func(c *CallConfig) {
		c.ProxyCaller = true
	}
}

// PrecompileAddresses holds the caller and self addresses for a precompile call
type PrecompileAddresses struct {
	Caller common.Address
	Self   common.Address
}
