// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"

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
	// OpNone is the declaration a precompile makes when its soundness
	// rests on no problem a quantum computer solves: lattice, code and
	// hash-based primitives, and precompiles that compute rather than
	// verify (identity, modexp). It is a claim, not an absence — the
	// absence of a claim is [classify] returning false, which is
	// refused rather than admitted.
	OpNone            = pq.OpUnknown
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

// AllForbidden returns the canonical strict-PQ profile (alias for
// [pq.Strict]).
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

// ErrUnclassifiedForbidden refuses a precompile that has not said what
// its soundness rests on. It fires only on a chain whose profile
// constrains something; see [constrains].
//
// Lives here rather than in [pq] because it is a property of this
// dispatcher, not of the profile vocabulary: pq describes which
// primitive families a chain refuses, and cannot know that a host
// failed to classify one of its own precompiles.
var ErrUnclassifiedForbidden = errors.New("unclassified precompile forbidden by chain security profile (PQ)")

// Classified is implemented by a precompile that names the classical
// primitive family its soundness reduces to.
//
// The declaration lives with the implementation because the answer is
// a property of the verifier's mathematics, not of its address: a
// module that swaps its proof system changes its own answer and
// nothing central needs editing. That is the difference from a central
// type switch, which answered "admit" for every module it had never
// heard of — which is how a strict-PQ chain came to refuse the native
// bn256Pairing while admitting a stateful precompile doing the same
// BN254 pairing internally.
//
// The test is not "does it touch an elliptic curve" but "does soundness
// reduce to a problem Shor solves". Halo2 uses no pairing and is still
// classical, because inner-product-argument soundness rests on
// elliptic-curve discrete log. X-Wing performs an X25519 exchange and
// is still post-quantum, because it is a combiner whose security holds
// if either half holds.
type Classified interface {
	// PQOp names the primitive family. [OpNone] is a valid answer and
	// declares post-quantum soundness.
	PQOp() Op
}

// classify resolves a precompile to the op the profile judges it by.
//
// The second result is the whole point: false means "this precompile
// never said", which is a different fact from [OpNone] ("it said, and
// the answer is nothing classical"). Conflating those two is what left
// stateful precompiles ungated — the old central switch returned one
// value for both and [(*PQProfile).RefuseUnder] admits it.
func classify(p PrecompiledContract) (Op, bool) {
	if d, ok := p.(Classified); ok {
		return d.PQOp(), true
	}
	return builtinOp(p)
}

// constrains reports whether a profile expresses any refusal at all.
//
// A nil profile and the zero value both constrain nothing, so they
// admit an unclassified precompile exactly as they always have. This
// is what keeps the deny-by-default rule inside the profile: a chain
// that never opted in sees no change in what it accepts.
func constrains(p *PQProfile) bool {
	return p != nil && *p != (PQProfile{})
}

// builtinOp classifies the precompiles geth itself defines.
//
// These are classified centrally rather than by a PQOp method on each
// type because they are upstream go-ethereum types: a method per type
// would be a merge conflict per type, forever. The set is closed —
// it changes only when upstream adds a precompile — and
// TestEveryPrecompileIsClassified fails when it drifts.
//
// Type-switch dispatch (not address dispatch) so the mapping is robust
// to precompile remapping via [PrecompileOverrider] and to ad-hoc test
// setups that install precompile types at non-standard addresses.
func builtinOp(p PrecompiledContract) (Op, bool) {
	switch p.(type) {
	// Compute, not verification: no soundness property to reduce.
	case *dataCopy, *bigModExp:
		return OpNone, true

	case *ecrecover:
		return OpEcrecover, true
	case *sha256hash:
		return OpSHA256, true
	case *ripemd160hash:
		return OpRIPEMD160, true
	case *blake2F:
		return OpBlake2F, true
	case *bn256AddIstanbul, *bn256AddByzantium:
		return OpBn256Add, true
	case *bn256ScalarMulIstanbul, *bn256ScalarMulByzantium:
		return OpBn256ScalarMul, true
	case *bn256PairingIstanbul, *bn256PairingByzantium:
		return OpBn256Pairing, true
	case *bls12381G1Add:
		return OpBLS12381G1Add, true
	case *bls12381G1MultiExp:
		return OpBLS12381G1MSM, true
	case *bls12381G2Add:
		return OpBLS12381G2Add, true
	case *bls12381G2MultiExp:
		return OpBLS12381G2MSM, true
	case *bls12381Pairing:
		return OpBLS12381Pairing, true
	case *bls12381MapG1:
		return OpBLS12381MapG1, true
	case *bls12381MapG2:
		return OpBLS12381MapG2, true
	case *kzgPointEvaluation:
		return OpKZGPointEval, true
	case *p256Verify:
		return OpP256Verify, true
	}
	return OpNone, false
}
