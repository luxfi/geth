// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"sync/atomic"
)

// pq_profile.go — PQ profile gate inside the EVM precompile layer.
//
// PQ mode is binary: a chain is PQ or it isn't. The canonical chain-wide
// ChainSecurityProfile lives in github.com/luxfi/pq. The node binary
// projects the EVM-relevant subset into a PQProfile and installs it here
// via SetPQProfile at chain bootstrap.
//
// We deliberately do NOT import the consensus or pq packages from
// core/vm:
//   - core/vm must stay free of higher-level dependencies.
//   - Only a handful of booleans are needed at this layer; the wider
//     profile is irrelevant to the EVM gas / auth gate.
//
// Default (no profile, no PQ required) preserves classical EVM
// semantics. PQ bootstrap is a two-call protocol:
//
//	1. RequirePQ()                              // record "this chain MUST be PQ"
//	2. SetPQProfile(&PQProfile{All: true})      // install the projection
//
// Step 1 fails the gate closed: until step 2 installs the projection,
// any classical precompile call returns ErrMissingPQProfile. This
// catches the case where chain config declared PQ but the bootstrap
// path forgot to (or could not) install the projection.
//
// One way only: install once at boot, read with atomic loads on every
// precompile call. No per-tx mutation, no per-call configuration. The
// type is PQProfile, the setter is SetPQProfile, the require flag is
// RequirePQ, the gate is refuse(Op). One name per concept.

// PQProfile is the EVM-side projection of the chain-wide post-quantum
// profile. Each Forbid* field maps to one category of classical
// primitive. A chain operator typically sets every field at once via
// the All shortcut.
type PQProfile struct {
	// ForbidECDSAContractAuth refuses ECDSA-family contract-auth
	// precompiles. Covers ecrecover at 0x01 and secp256r1 verify at
	// 0x100 (RIP-7212).
	ForbidECDSAContractAuth bool

	// ForbidNonSHA3Hashes refuses non-SHA3 hash and compression
	// precompiles. Covers sha256 at 0x02, ripemd160 at 0x03, and
	// blake2F at 0x09. SHA-256 is not quantum-broken but a strict-PQ
	// profile demands a single NIST-recommended hash family (SHA3)
	// for hash-of-everything composability.
	ForbidNonSHA3Hashes bool

	// ForbidPairingPrecompiles refuses pairing-friendly curve
	// operations. Covers alt_bn128 at 0x06/0x07/0x08 and the BLS12-381
	// family at EIP-2537 addresses 0x0a–0x12 (mapping depends on the
	// fork).
	ForbidPairingPrecompiles bool

	// ForbidKZGPrecompiles refuses EIP-4844 point evaluation at 0x0a
	// (Cancun-numbered). KZG point evaluation is pairing-based and
	// shares the security assumption of BN254 / BLS12-381.
	ForbidKZGPrecompiles bool
}

// AllForbidden returns a profile that forbids every classical category.
// This is the canonical projection a strict-PQ chain installs.
func AllForbidden() *PQProfile {
	return &PQProfile{
		ForbidECDSAContractAuth:  true,
		ForbidNonSHA3Hashes:      true,
		ForbidPairingPrecompiles: true,
		ForbidKZGPrecompiles:     true,
	}
}

// Op enumerates the classical operations a precompile call invokes.
// Op values are categorical: a given precompile maps to exactly one Op
// regardless of how many fork-specific implementations the codebase
// carries (e.g. bn256AddByzantium and bn256AddIstanbul both report
// OpBn256Add).
type Op uint8

// The complete set of classical operations gated by the PQ profile.
// Append-only; renumbering is a backward-incompatible change to log
// lines and dashboards.
const (
	OpUnknown Op = iota

	// ECDSA family — CatECDSAContractAuth.
	OpECDSARecover // 0x01 — ecrecover
	OpP256Verify   // 0x100 — RIP-7212 secp256r1 verify

	// Non-SHA3 hash — CatNonSHA3Hash.
	OpSHA256    // 0x02
	OpRIPEMD160 // 0x03
	OpBlake2F   // 0x09

	// Pairing-friendly curves — CatPairingPrecompile.
	OpBn256Add         // 0x06
	OpBn256ScalarMul   // 0x07
	OpBn256Pairing     // 0x08
	OpBLS12381G1Add    // 0x0b
	OpBLS12381G1MSM    // 0x0c
	OpBLS12381G2Add    // 0x0d
	OpBLS12381G2MSM    // 0x0e
	OpBLS12381Pairing  // 0x0f
	OpBLS12381MapG1    // 0x10
	OpBLS12381MapG2    // 0x11

	// Polynomial commitment — CatKZGPrecompile.
	OpKZGPointEval // 0x0a (Cancun)
)

// ErrClassicalAuthForbidden is returned when a classical contract-auth
// precompile is invoked under a profile that forbids ECDSA family ops.
var ErrClassicalAuthForbidden = errors.New("classical contract-auth forbidden by chain security profile (PQ)")

// ErrNonSHA3HashForbidden is returned when SHA-256, RIPEMD-160, or
// BLAKE2 is invoked under a profile that demands SHA3 exclusively.
var ErrNonSHA3HashForbidden = errors.New("non-SHA3 hash forbidden by chain security profile (PQ)")

// ErrPairingForbidden is returned when an alt_bn128 or BLS12-381
// pairing-family precompile is invoked under a profile that forbids
// pairing-based primitives.
var ErrPairingForbidden = errors.New("pairing-friendly curve precompile forbidden by chain security profile (PQ)")

// ErrKZGForbidden is returned when EIP-4844 point evaluation is invoked
// under a profile that forbids KZG.
var ErrKZGForbidden = errors.New("KZG point evaluation forbidden by chain security profile (PQ)")

// ErrMissingPQProfile is returned when RequirePQ() has been called but
// the chain-wide PQProfile projection has not yet been installed via
// SetPQProfile. Catches the misordered or incomplete bootstrap path.
var ErrMissingPQProfile = errors.New("PQ chain expected but profile projection not installed in EVM")

// activePQProfile holds the chain-wide projection installed at
// bootstrap. nil means "no projection installed"; combined with
// pqRequired this determines fail-open vs fail-closed semantics.
var activePQProfile atomic.Pointer[PQProfile]

// pqRequired records whether the chain declared PQ in its
// configuration. When true and activePQProfile is nil, every gated
// precompile fails closed with ErrMissingPQProfile.
var pqRequired atomic.Bool

// SetPQProfile installs the chain-wide projection. Called once at
// chain bootstrap by the node binary. Passing nil restores the
// "no projection installed" state and is test-only.
func SetPQProfile(p *PQProfile) {
	activePQProfile.Store(p)
}

// ActivePQProfile returns the chain-wide projection, or nil if none
// has been installed. Callers must combine the result with PQRequired
// to interpret nil correctly.
func ActivePQProfile() *PQProfile {
	return activePQProfile.Load()
}

// RequirePQ records that the chain has been configured as PQ. Called
// once at bootstrap before SetPQProfile so any race window or
// out-of-order boot fails closed.
func RequirePQ() {
	pqRequired.Store(true)
}

// ClearPQRequired un-sets the PQ expectation. Test-only.
func ClearPQRequired() {
	pqRequired.Store(false)
}

// PQRequired reports whether RequirePQ has been called.
func PQRequired() bool {
	return pqRequired.Load()
}

// refuse is the single profile gate. Every classical precompile calls
// it once at the top of its Run() and otherwise proceeds in its own
// lane. Hot path: at most two atomic loads plus one map lookup.
//
// Returns:
//   - nil                            proceed with classical semantics
//   - one of the four Err* values    profile forbids this op
//   - ErrMissingPQProfile            PQ required, profile missing
//
// refuse(OpUnknown) always returns nil; callers MUST use a recognized
// Op value or risk silently bypassing the gate.
func refuse(op Op) error {
	p := activePQProfile.Load()
	if p == nil {
		if pqRequired.Load() {
			return ErrMissingPQProfile
		}
		return nil
	}
	switch op {
	case OpECDSARecover, OpP256Verify:
		if p.ForbidECDSAContractAuth {
			return ErrClassicalAuthForbidden
		}
	case OpSHA256, OpRIPEMD160, OpBlake2F:
		if p.ForbidNonSHA3Hashes {
			return ErrNonSHA3HashForbidden
		}
	case OpBn256Add, OpBn256ScalarMul, OpBn256Pairing,
		OpBLS12381G1Add, OpBLS12381G1MSM,
		OpBLS12381G2Add, OpBLS12381G2MSM,
		OpBLS12381Pairing,
		OpBLS12381MapG1, OpBLS12381MapG2:
		if p.ForbidPairingPrecompiles {
			return ErrPairingForbidden
		}
	case OpKZGPointEval:
		if p.ForbidKZGPrecompiles {
			return ErrKZGForbidden
		}
	}
	return nil
}

// classicalContractAuthCheck is the legacy alias used by ecrecover.Run.
// It dispatches to refuse(OpECDSARecover) so the gate logic stays
// single-sourced. Kept so the call site at contracts.go:295 stays
// compact and reads cleanly.
func classicalContractAuthCheck() error {
	return refuse(OpECDSARecover)
}

// forbidClassicalContractAuth reports whether the chain currently
// refuses classical contract-auth. Diagnostic-only.
func forbidClassicalContractAuth() bool {
	return classicalContractAuthCheck() != nil
}
