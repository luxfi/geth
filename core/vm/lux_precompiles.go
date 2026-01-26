// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/precompile/contract"
	"github.com/luxfi/precompile/mldsa"
	"github.com/luxfi/precompile/pqcrypto"
	"github.com/luxfi/precompile/precompileconfig"
	"github.com/luxfi/precompile/slhdsa"
)

// precompileAdapter wraps a precompiles.StatefulPrecompiledContract to implement
// geth's StatefulPrecompiledContract interface.
type precompileAdapter struct {
	name    string
	address common.Address
	inner   contract.StatefulPrecompiledContract
	gasFunc func([]byte) uint64
}

// Name returns the precompile name
func (p *precompileAdapter) Name() string {
	return p.name
}

// RequiredGas returns the gas required for this precompile
func (p *precompileAdapter) RequiredGas(input []byte) uint64 {
	return p.gasFunc(input)
}

// Run implements PrecompiledContract.Run (stateless - not used for stateful precompiles)
func (p *precompileAdapter) Run(input []byte) ([]byte, error) {
	// Stateful precompiles use RunStateful instead
	return nil, ErrExecutionReverted
}

// RunStateful implements StatefulPrecompiledContract.RunStateful
func (p *precompileAdapter) RunStateful(env PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	// Create an AccessibleState adapter
	accessibleState := &accessibleStateAdapter{env: env}

	// Get caller and self addresses from environment
	addresses := env.Addresses()

	// Call the inner precompile
	ret, remainingGas, err := p.inner.Run(
		accessibleState,
		addresses.Caller,
		addresses.Self,
		input,
		suppliedGas,
		env.ReadOnly(),
	)

	return ret, remainingGas, err
}

// accessibleStateAdapter adapts PrecompileEnvironment to contract.AccessibleState
type accessibleStateAdapter struct {
	env PrecompileEnvironment
}

func (a *accessibleStateAdapter) GetStateDB() contract.StateDB {
	return nil // PQ crypto precompiles don't use state
}

func (a *accessibleStateAdapter) GetBlockContext() contract.BlockContext {
	return &blockContextAdapter{env: a.env}
}

func (a *accessibleStateAdapter) GetConsensusContext() context.Context {
	return a.env.ConsensusContext()
}

func (a *accessibleStateAdapter) GetChainConfig() precompileconfig.ChainConfig {
	return nil // PQ crypto precompiles don't use chain config
}

func (a *accessibleStateAdapter) GetPrecompileEnv() contract.PrecompileEnvironment {
	return &precompileEnvAdapter{env: a.env}
}

// precompileEnvAdapter adapts vm.PrecompileEnvironment to contract.PrecompileEnvironment
type precompileEnvAdapter struct {
	env PrecompileEnvironment
}

func (p *precompileEnvAdapter) BlockNumber() *big.Int {
	return p.env.BlockNumber()
}

func (p *precompileEnvAdapter) BlockTime() uint64 {
	return p.env.BlockTime()
}

func (p *precompileEnvAdapter) ReadOnly() bool {
	return p.env.ReadOnly()
}

func (p *precompileEnvAdapter) ChainConfig() *params.ChainConfig {
	return p.env.ChainConfig()
}

func (p *precompileEnvAdapter) StateDB() interface{} {
	return p.env.StateDB()
}

func (p *precompileEnvAdapter) Gas() uint64 {
	return p.env.Gas()
}

func (p *precompileEnvAdapter) UseGas(gas uint64) bool {
	return p.env.UseGas(gas)
}

// blockContextAdapter adapts PrecompileEnvironment to contract.BlockContext
type blockContextAdapter struct {
	env PrecompileEnvironment
}

func (b *blockContextAdapter) Number() *big.Int {
	return b.env.BlockNumber()
}

func (b *blockContextAdapter) Timestamp() uint64 {
	return b.env.BlockTime()
}

func (b *blockContextAdapter) GetPredicateResults(txHash common.Hash, precompileAddress common.Address) []byte {
	return nil // Not needed for PQ crypto precompiles
}

// LuxPrecompiles returns a map of Lux-specific precompiles (PQ crypto, etc.)
// These should be merged with the standard precompiles at chain initialization.
func LuxPrecompiles() PrecompiledContracts {
	return PrecompiledContracts{
		// ML-DSA (Post-Quantum Signatures - FIPS 204)
		mldsa.ContractMLDSAVerifyAddress: &precompileAdapter{
			name:    "mldsa",
			address: mldsa.ContractMLDSAVerifyAddress,
			inner:   mldsa.MLDSAVerifyPrecompile,
			gasFunc: mldsa.MLDSAVerifyPrecompile.RequiredGas,
		},

		// SLH-DSA (Hash-based Signatures - FIPS 205)
		slhdsa.ContractSLHDSAVerifyAddress: &precompileAdapter{
			name:    "slhdsa",
			address: slhdsa.ContractSLHDSAVerifyAddress,
			inner:   slhdsa.SLHDSAVerifyPrecompile,
			gasFunc: slhdsa.SLHDSAVerifyPrecompile.RequiredGas,
		},

		// PQCrypto (General Post-Quantum Operations including ML-KEM)
		pqcrypto.ContractAddress: &precompileAdapter{
			name:    "pqcrypto",
			address: pqcrypto.ContractAddress,
			inner:   pqcrypto.PQCryptoPrecompile,
			gasFunc: pqcrypto.PQCryptoPrecompile.RequiredGas,
		},
	}
}

// MergeLuxPrecompiles merges Lux precompiles into the given precompile map.
// Returns a new map with all precompiles.
func MergeLuxPrecompiles(base PrecompiledContracts) PrecompiledContracts {
	result := make(PrecompiledContracts)

	// Copy base precompiles
	for addr, p := range base {
		result[addr] = p
	}

	// Add Lux precompiles
	for addr, p := range LuxPrecompiles() {
		result[addr] = p
	}

	return result
}

// PrecompiledContractsLux contains all precompiles for Lux chains.
// This includes standard Ethereum precompiles plus Lux-specific ones.
var PrecompiledContractsLux = MergeLuxPrecompiles(PrecompiledContractsCancun)
