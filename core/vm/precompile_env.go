// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"math/big"

	"github.com/holiman/uint256"
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
	// ConsensusContext returns the consensus context (use runtime.FromContext to get runtime)
	ConsensusContext() context.Context
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

// evmPrecompileEnv implements PrecompileEnvironment using the EVM
type evmPrecompileEnv struct {
	evm      *EVM
	caller   common.Address
	self     common.Address
	gas      uint64
	readOnly bool
}

// NewPrecompileEnvironment creates a PrecompileEnvironment from EVM context
func NewPrecompileEnvironment(evm *EVM, caller, self common.Address, gas uint64, readOnly bool) PrecompileEnvironment {
	return &evmPrecompileEnv{
		evm:      evm,
		caller:   caller,
		self:     self,
		gas:      gas,
		readOnly: readOnly,
	}
}

func (e *evmPrecompileEnv) BlockHeader() (*types.Header, error) {
	return &types.Header{
		Number:   e.evm.Context.BlockNumber,
		Time:     e.evm.Context.Time,
		Coinbase: e.evm.Context.Coinbase,
		GasLimit: e.evm.Context.GasLimit,
		BaseFee:  e.evm.Context.BaseFee,
	}, nil
}

func (e *evmPrecompileEnv) Rules() params.Rules {
	return e.evm.chainRules
}

func (e *evmPrecompileEnv) BlockNumber() *big.Int {
	return new(big.Int).Set(e.evm.Context.BlockNumber)
}

func (e *evmPrecompileEnv) BlockTime() uint64 {
	return e.evm.Context.Time
}

func (e *evmPrecompileEnv) Addresses() PrecompileAddresses {
	return PrecompileAddresses{
		Caller: e.caller,
		Self:   e.self,
	}
}

func (e *evmPrecompileEnv) ReadOnly() bool {
	return e.readOnly
}

func (e *evmPrecompileEnv) ChainConfig() *params.ChainConfig {
	return e.evm.chainConfig
}

func (e *evmPrecompileEnv) StateDB() StateDB {
	return e.evm.StateDB
}

func (e *evmPrecompileEnv) ReadOnlyState() StateDB {
	return e.evm.StateDB
}

func (e *evmPrecompileEnv) UseGas(gas uint64) bool {
	if e.gas < gas {
		return false
	}
	e.gas -= gas
	return true
}

func (e *evmPrecompileEnv) Gas() uint64 {
	return e.gas
}

func (e *evmPrecompileEnv) Call(addr common.Address, input []byte, gas uint64, value *big.Int, opts ...CallOption) ([]byte, uint64, error) {
	// Handle call options
	cfg := &CallConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	caller := e.self
	if cfg.ProxyCaller {
		caller = e.caller
	}

	// Convert *big.Int to *uint256.Int
	var val *uint256.Int
	if value != nil {
		val = new(uint256.Int)
		val.SetFromBig(value)
	} else {
		val = new(uint256.Int)
	}

	ret, remainingGas, err := e.evm.Call(caller, addr, input, gas, val)
	return ret, remainingGas, err
}

func (e *evmPrecompileEnv) ConsensusContext() context.Context {
	if e.evm.Context.ConsensusContext != nil {
		return e.evm.Context.ConsensusContext
	}
	return context.Background()
}
