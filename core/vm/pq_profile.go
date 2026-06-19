// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/pq"
)

// pq_profile.go — EVM-side bindings for github.com/luxfi/pq.
//
// luxfi/pq is the single source of truth for the strict-PQ profile.
// This file re-exports its types and constants so call sites inside
// core/vm read naturally (vm.PQProfile, vm.OpEcrecover, etc.) without
// the package qualifier.
//
// One way only: the host VM installs a *PQProfile on ChainConfig.PQ at
// chain bootstrap; (*EVM).runPrecompile reads chainConfig.PQ and calls
// (*PQProfile).RefuseUnder(op) before dispatching the precompile. The
// per-precompile Run() methods stay clean — no global, no atomic load.
// Multi-chain hosts (strict-PQ and permissive chains in one process)
// get the right gate per EVM instance.

// PQProfile aliases [pq.Profile]: the strict-PQ profile value.
type PQProfile = pq.Profile

// Op aliases [pq.Op]: the precompile-family enum.
type Op = pq.Op

// Op constants — direct re-export of [pq.Op*].
const (
	OpUnknown         = pq.OpUnknown
	OpEcrecover       = pq.OpEcrecover
	OpP256Verify      = pq.OpP256Verify
	OpSHA256          = pq.OpSHA256
	OpRIPEMD160       = pq.OpRIPEMD160
	OpBlake2F         = pq.OpBlake2F
	OpBn256Add        = pq.OpBn256Add
	OpBn256ScalarMul  = pq.OpBn256ScalarMul
	OpBn256Pairing    = pq.OpBn256Pairing
	OpBLS12381G1Add   = pq.OpBLS12381G1Add
	OpBLS12381G1MSM   = pq.OpBLS12381G1MSM
	OpBLS12381G2Add   = pq.OpBLS12381G2Add
	OpBLS12381G2MSM   = pq.OpBLS12381G2MSM
	OpBLS12381Pairing = pq.OpBLS12381Pairing
	OpBLS12381MapG1   = pq.OpBLS12381MapG1
	OpBLS12381MapG2   = pq.OpBLS12381MapG2
	OpKZGPointEval    = pq.OpKZGPointEval
)

// Family errors — direct re-export of [pq.Err*Forbidden].
var (
	ErrEcrecoverForbidden  = pq.ErrEcrecoverForbidden
	ErrP256VerifyForbidden = pq.ErrP256VerifyForbidden
	ErrSHA256Forbidden     = pq.ErrSHA256Forbidden
	ErrRIPEMD160Forbidden  = pq.ErrRIPEMD160Forbidden
	ErrBlake2FForbidden    = pq.ErrBlake2FForbidden
	ErrBn256Forbidden      = pq.ErrBn256Forbidden
	ErrBLS12381Forbidden   = pq.ErrBLS12381Forbidden
	ErrKZGForbidden        = pq.ErrKZGForbidden
)

// AllForbidden returns the maximal strict-PQ profile (alias for
// [pq.Strict]): every classical precompile family refused, including the
// standard alt_bn128 (BN254) precompiles at 0x06–0x08.
//
// NOTE: a Lux-derived EVM chain does NOT install this maximal profile. It
// installs its own projection (evm/plugin/evm.LuxStrictPQ) which keeps the
// standard bn256 precompiles (0x06–0x08) available for Ethereum-compat
// dapps — Lux's security-critical pairing/DLOG usage lives in CUSTOM
// precompiles gated separately, not in 0x06–0x08. The per-chain profile is
// installed on ChainConfig.PQ by the EVM plugin; (*EVM).runPrecompile reads
// it. AllForbidden remains here only as the library maximal primitive and
// as a test baseline. core/vm/pq_lux_profile_test.go proves the gate admits
// bn256 under the bn256-carve-out profile through the real dispatch.
func AllForbidden() *PQProfile { return pq.Strict() }

// SetPQProfile installs the deprecated package-global PQ projection.
//
// Deprecated: process-global PQ state has last-writer-wins semantics
// across chains hosted in one binary. Set [params.ChainConfig.PQ]
// instead and let (*EVM).runPrecompile gate per chain. Retained as a
// shim that delegates to [pq.SetActive].
func SetPQProfile(p *PQProfile) { pq.SetActive(p) }

// ActivePQProfile returns the deprecated package-global PQ projection.
//
// Deprecated: see [SetPQProfile].
func ActivePQProfile() *PQProfile { return pq.Active() }

// opForPrecompile maps a precompile implementation to its [pq.Op]
// classification. Returns OpUnknown for precompiles that are not gated
// by the strict-PQ profile (dataCopy, bigModExp, stateful custom
// precompiles installed by overriders). OpUnknown is always admitted
// by [(*PQProfile).RefuseUnder], so the dispatch is fail-open for
// unrecognized types.
//
// Type-switch dispatch (not address dispatch) so the mapping is robust
// to precompile remapping via [PrecompileOverrider] and to ad-hoc test
// setups that install precompile types at non-standard addresses.
func opForPrecompile(p PrecompiledContract) Op {
	switch p.(type) {
	case *ecrecover:
		return OpEcrecover
	case *sha256hash:
		return OpSHA256
	case *ripemd160hash:
		return OpRIPEMD160
	case *blake2F:
		return OpBlake2F
	case *bn256AddIstanbul, *bn256AddByzantium:
		return OpBn256Add
	case *bn256ScalarMulIstanbul, *bn256ScalarMulByzantium:
		return OpBn256ScalarMul
	case *bn256PairingIstanbul, *bn256PairingByzantium:
		return OpBn256Pairing
	case *bls12381G1Add:
		return OpBLS12381G1Add
	case *bls12381G1MultiExp:
		return OpBLS12381G1MSM
	case *bls12381G2Add:
		return OpBLS12381G2Add
	case *bls12381G2MultiExp:
		return OpBLS12381G2MSM
	case *bls12381Pairing:
		return OpBLS12381Pairing
	case *bls12381MapG1:
		return OpBLS12381MapG1
	case *bls12381MapG2:
		return OpBLS12381MapG2
	case *kzgPointEvaluation:
		return OpKZGPointEval
	case *p256Verify:
		return OpP256Verify
	}
	return OpUnknown
}
