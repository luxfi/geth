// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"context"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/precompile/contract"
	coronathreshold "github.com/luxfi/precompile/corona"
	"github.com/luxfi/precompile/mldsa"
	"github.com/luxfi/precompile/mlkem"
	"github.com/luxfi/precompile/p3q"
	"github.com/luxfi/precompile/precompileconfig"
	"github.com/luxfi/precompile/pulsar"
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

// NewPrecompileAdapter wraps a luxfi/precompile StatefulPrecompiledContract
// into a geth-compatible StatefulPrecompiledContract. Exported so that L1
// chains can assemble their own precompile sets via PrecompileOverrider
// without requiring changes to geth itself.
func NewPrecompileAdapter(name string, address common.Address, inner contract.StatefulPrecompiledContract, gasFunc func([]byte) uint64) StatefulPrecompiledContract {
	return &precompileAdapter{
		name:    name,
		address: address,
		inner:   inner,
		gasFunc: gasFunc,
	}
}

// LuxPrecompiles returns the baseline Lux precompiles that every Lux chain
// should support. This is the LP-4200 unified PQCrypto block: ML-KEM,
// ML-DSA, SLH-DSA, Pulsar (Module-LWE threshold ML-DSA), P3Q (strict-PQ
// STARK), and Corona (Ring-LWE threshold).
// L1 chains can add additional precompiles (DEX, sr25519, FROST, etc.)
// by implementing PrecompileOverrider in their Rules.Payload.
//
// Addresses (LP-4200 unified block):
//
//	0x012201 = ML-KEM   (FIPS 203 — Module-LWE KEM)
//	0x012202 = ML-DSA   (FIPS 204 — Module-LWE single-party signatures)
//	0x012203 = SLH-DSA  (FIPS 205 — stateless hash-based signatures)
//	0x012204 = Pulsar   (Module-LWE threshold FIPS 204, byte-equal to ML-DSA)
//	0x012205 = P3Q      (strict-PQ STARK / FRI / cSHAKE256 verifier)
//	0x012206 = Corona   (Ring-LWE threshold)
//
// All six are always available regardless of PQ profile; the profile
// gate only constrains the *classical* precompiles (ecrecover, sha256,
// alt_bn128, BLS12-381, KZG) — it never disables PQ primitives.
func LuxPrecompiles() PrecompiledContracts {
	return PrecompiledContracts{
		// ML-KEM (FIPS 203 — post-quantum key encapsulation)
		mlkem.ContractAddress: &precompileAdapter{
			name:    "mlkem",
			address: mlkem.ContractAddress,
			inner:   mlkem.MLKEMPrecompile,
			gasFunc: mlkem.MLKEMPrecompile.RequiredGas,
		},

		// ML-DSA (FIPS 204 — single-party post-quantum signatures)
		mldsa.ContractMLDSAVerifyAddress: &precompileAdapter{
			name:    "mldsa",
			address: mldsa.ContractMLDSAVerifyAddress,
			inner:   mldsa.MLDSAVerifyPrecompile,
			gasFunc: mldsa.MLDSAVerifyPrecompile.RequiredGas,
		},

		// SLH-DSA (FIPS 205 — stateless hash-based signatures)
		slhdsa.ContractSLHDSAVerifyAddress: &precompileAdapter{
			name:    "slhdsa",
			address: slhdsa.ContractSLHDSAVerifyAddress,
			inner:   slhdsa.SLHDSAVerifyPrecompile,
			gasFunc: slhdsa.SLHDSAVerifyPrecompile.RequiredGas,
		},

		// Pulsar (threshold FIPS 204 — Module-LWE threshold ML-DSA).
		// Byte-equal output to single-party ML-DSA per the NIST MPTC
		// Class N1 manifesto; the underlying verifier dispatches to
		// luxfi/crypto/mldsa.
		pulsar.ContractPulsarVerifyAddress: &precompileAdapter{
			name:    "pulsar",
			address: pulsar.ContractPulsarVerifyAddress,
			inner:   pulsar.PulsarVerifyPrecompile,
			gasFunc: pulsar.PulsarVerifyPrecompile.RequiredGas,
		},

		// P3Q (strict-PQ STARK / FRI / cSHAKE256 / Goldilocks).
		// Verifier callback wired at node init via p3q.RegisterVerifier.
		p3q.ContractP3QVerifyAddress: &precompileAdapter{
			name:    "p3q",
			address: p3q.ContractP3QVerifyAddress,
			inner:   p3q.P3QVerifyPrecompile,
			gasFunc: p3q.P3QVerifyPrecompile.RequiredGas,
		},

		// Corona (Ring-LWE threshold). Distinct algebra from Pulsar:
		// Pulsar is Module-LWE (FIPS 204 byte-equal); Corona is Ring-LWE
		// over a single ring. Both are NIST MPTC Class N1 candidates.
		coronathreshold.ContractCoronaThresholdAddress: &precompileAdapter{
			name:    "corona",
			address: coronathreshold.ContractCoronaThresholdAddress,
			inner:   coronathreshold.CoronaThresholdPrecompile,
			gasFunc: coronathreshold.CoronaThresholdPrecompile.RequiredGas,
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

// init registers a safe-refuse stub for the P3Q strict-PQ STARK
// verifier at boot.
//
// The real verifier is a CGO bridge into the Rust crate at
// ~/work/lux/p3q (Plonky3 fork, FRI over Goldilocks, cSHAKE256).
// That crate doesn't yet ship a `cdylib` build target + Go FFI; until
// it does, every node boots without a registered verifier and the
// precompile returns p3q.ErrVerifierNotRegistered — safe-refuse, no
// forgery oracle.
//
// To wire the real verifier when the Rust FFI lands, replace the
// stub closure with the cgo entry point and rebuild geth. The
// registration call here is the single seam.
func init() {
	p3q.RegisterVerifier(func(version byte, proof, pubInputs []byte) (bool, error) {
		// Refuse-by-default. Real verifier slots in here.
		return false, p3q.ErrVerifierNotRegistered
	})
}
