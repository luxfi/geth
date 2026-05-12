// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/luxfi/geth/params"
)

// pq_profile_test.go — ecrecover-focused coverage of the per-chain PQ
// profile. The gate lives in (*EVM).runPrecompile; tests drive the
// precompile through a test EVM whose ChainConfig.PQ is set per case.
//
// The cross-family matrix lives in pq_gate_test.go.

// validEcrecoverInput is a well-known signature valid for ecrecover.
// The exact recovered address does not matter — only that the
// precompile returns a 32-byte address under classical semantics and
// ErrEcrecoverForbidden under strict-PQ.
const validEcrecoverInputHex = "" +
	"a35a39e7715a7b2c5d2e3a5d8e8f8a8b8c8d8e8f9091929394959697989a9b9c" +
	"000000000000000000000000000000000000000000000000000000000000001c" +
	"7b6d1f1f0a85b5a3aa3f0e57c8c30a1b1c1d1e1f2021222324252627282a2b2c" +
	"3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c"

// newTestEVMWithProfile constructs a minimal EVM whose chain config
// has its PQ profile set as supplied. State is nil; the test only
// exercises (*EVM).runPrecompile, which does not touch state.
func newTestEVMWithProfile(t *testing.T, p *PQProfile) *EVM {
	t.Helper()
	cc := *params.TestChainConfig
	cc.PQ = p
	return NewEVM(BlockContext{BlockNumber: big.NewInt(0)}, nil, &cc, Config{})
}

// TestNilProfile_EcrecoverWorks asserts the default state (no profile
// installed on the chain config) runs ecrecover with classical
// semantics — runPrecompile does not return the family error.
func TestNilProfile_EcrecoverWorks(t *testing.T) {
	evm := newTestEVMWithProfile(t, nil)
	p := &ecrecover{}
	input, err := hex.DecodeString(validEcrecoverInputHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	_, _, err = evm.runPrecompile(p, input, p.RequiredGas(input))
	if errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("nil profile must not return ErrEcrecoverForbidden; got %v", err)
	}
}

// TestStrictPQ_EcrecoverReturnsError asserts a chain config with
// ForbidEcrecover=true causes runPrecompile to return
// ErrEcrecoverForbidden.
func TestStrictPQ_EcrecoverReturnsError(t *testing.T) {
	evm := newTestEVMWithProfile(t, &PQProfile{ForbidEcrecover: true})
	p := &ecrecover{}
	input, err := hex.DecodeString(validEcrecoverInputHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	_, _, err = evm.runPrecompile(p, input, p.RequiredGas(input))
	if !errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("want ErrEcrecoverForbidden, got %v", err)
	}
}

// TestStrictPQ_EcrecoverEmptyInputAlsoRefused asserts the gate fires
// before any input parsing, including on zero-length input. Note
// RequiredGas is invariant under input here so we supply the same
// fixed gas budget either way.
func TestStrictPQ_EcrecoverEmptyInputAlsoRefused(t *testing.T) {
	evm := newTestEVMWithProfile(t, &PQProfile{ForbidEcrecover: true})
	p := &ecrecover{}
	_, _, err := evm.runPrecompile(p, nil, p.RequiredGas(nil))
	if !errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("want ErrEcrecoverForbidden on empty input, got %v", err)
	}
}

// TestStrictPQ_GasIsStillCharged asserts that under strict-PQ the
// runPrecompile path charges RequiredGas: refusal must not be
// detectable via gas timing.
func TestStrictPQ_GasIsStillCharged(t *testing.T) {
	p := &ecrecover{}
	input := make([]byte, 128)
	cost := p.RequiredGas(input)
	// Classical (nil profile).
	evmClassical := newTestEVMWithProfile(t, nil)
	_, remainingClassical, _ := evmClassical.runPrecompile(p, input, cost*2)
	// Strict-PQ.
	evmStrict := newTestEVMWithProfile(t, &PQProfile{ForbidEcrecover: true})
	_, remainingStrict, _ := evmStrict.runPrecompile(p, input, cost*2)
	if remainingClassical != remainingStrict {
		t.Fatalf("runPrecompile gas accounting must not depend on profile: classical_remaining=%d strict_remaining=%d (cost=%d)",
			remainingClassical, remainingStrict, cost)
	}
}

// TestChainConfigPQ_RoundTrip asserts the profile installed on
// ChainConfig.PQ is the one observed by runPrecompile.
func TestChainConfigPQ_RoundTrip(t *testing.T) {
	profile := &PQProfile{ForbidEcrecover: true, ForbidBn256: true}
	evm := newTestEVMWithProfile(t, profile)
	if got := evm.ChainConfig().PQ; got != profile {
		t.Fatalf("ChainConfig.PQ must return the installed pointer")
	}
	if !evm.ChainConfig().PQ.ForbidEcrecover || !evm.ChainConfig().PQ.ForbidBn256 {
		t.Fatalf("installed profile flags lost in round-trip")
	}
}
