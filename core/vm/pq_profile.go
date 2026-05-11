// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"sync/atomic"
)

// pq_profile.go — PQ profile gate inside the EVM precompile layer.
//
// One concept, one way:
//
//   1. SetPQProfile(p)   — install the chain-wide projection
//   2. refuse(op)        — the gate every classical precompile calls
//
// No second flag. No "required but missing" middle state. A nil
// profile means classical EVM semantics; an installed profile carries
// every decision. The node binary is responsible for calling
// SetPQProfile during chain bootstrap when the chain config declares
// PQ; if bootstrap forgets, the chain runs classical — that is a
// bootstrap bug, not a runtime concern.
//
// We deliberately do NOT import the consensus or pq packages from
// core/vm:
//   - core/vm must stay free of higher-level dependencies.
//   - Only a handful of booleans are needed at this layer; the wider
//     profile is irrelevant to the EVM gas / auth gate.
//
// One name per concept; no aliases.

// PQProfile is the EVM-side projection of the chain-wide post-quantum
// profile. Each Forbid* flag names a precompile family by its
// cryptographic primitive — not by an abstract category. A chain that
// wants to keep secp256r1 for passkey bridging but forbid every other
// classical primitive sets each flag independently.
//
// AllForbidden returns the canonical strict-PQ projection with every
// flag true.
type PQProfile struct {
	// Asymmetric signatures.
	ForbidEcrecover  bool // ecrecover at 0x01 (secp256k1 recovery)
	ForbidP256Verify bool // p256Verify at 0x100 (RIP-7212, secp256r1)

	// Non-SHA3 hash / compression precompiles.
	ForbidSHA256    bool // 0x02
	ForbidRIPEMD160 bool // 0x03
	ForbidBlake2F   bool // 0x09 (Blake2b compression — not BLAKE3)

	// Pairing-friendly curves, per family.
	ForbidBn256    bool // alt_bn128 at 0x06–0x08 (BN254)
	ForbidBLS12381 bool // EIP-2537 at 0x0b–0x11

	// Polynomial commitment.
	ForbidKZG bool // kzgPointEvaluation at 0x0a (Cancun)
}

// AllForbidden returns a profile that forbids every classical
// precompile family. The canonical strict-PQ projection.
func AllForbidden() *PQProfile {
	return &PQProfile{
		ForbidEcrecover:  true,
		ForbidP256Verify: true,
		ForbidSHA256:     true,
		ForbidRIPEMD160:  true,
		ForbidBlake2F:    true,
		ForbidBn256:      true,
		ForbidBLS12381:   true,
		ForbidKZG:        true,
	}
}

// Op names the classical primitive a precompile invokes. Byzantium and
// Istanbul variants of the same operation report the same Op (e.g.
// both bn256AddByzantium and bn256AddIstanbul → OpBn256Add).
// Append-only; renumbering is a backward-incompatible change to log
// lines and dashboards.
type Op uint8

const (
	OpUnknown Op = iota

	// Asymmetric signatures.
	OpEcrecover  // ecrecover (0x01)
	OpP256Verify // p256Verify (0x100)

	// Hash / compression.
	OpSHA256    // 0x02
	OpRIPEMD160 // 0x03
	OpBlake2F   // 0x09

	// alt_bn128 (BN254).
	OpBn256Add       // 0x06
	OpBn256ScalarMul // 0x07
	OpBn256Pairing   // 0x08

	// BLS12-381 (EIP-2537).
	OpBLS12381G1Add   // 0x0b
	OpBLS12381G1MSM   // 0x0c
	OpBLS12381G2Add   // 0x0d
	OpBLS12381G2MSM   // 0x0e
	OpBLS12381Pairing // 0x0f
	OpBLS12381MapG1   // 0x10
	OpBLS12381MapG2   // 0x11

	// Polynomial commitment.
	OpKZGPointEval // 0x0a (Cancun)
)

// Family errors. Each names the specific primitive family it refuses.
var (
	ErrEcrecoverForbidden  = errors.New("ecrecover forbidden by chain security profile (PQ)")
	ErrP256VerifyForbidden = errors.New("p256Verify forbidden by chain security profile (PQ)")
	ErrSHA256Forbidden     = errors.New("sha256 forbidden by chain security profile (PQ)")
	ErrRIPEMD160Forbidden  = errors.New("ripemd160 forbidden by chain security profile (PQ)")
	ErrBlake2FForbidden    = errors.New("blake2F forbidden by chain security profile (PQ)")
	ErrBn256Forbidden      = errors.New("alt_bn128 (BN254) family forbidden by chain security profile (PQ)")
	ErrBLS12381Forbidden   = errors.New("BLS12-381 family forbidden by chain security profile (PQ)")
	ErrKZGForbidden        = errors.New("KZG point evaluation forbidden by chain security profile (PQ)")
)

// activePQProfile holds the chain-wide projection installed at
// bootstrap. nil means "classical EVM semantics", the default for
// non-PQ chains and the initial state at process start.
var activePQProfile atomic.Pointer[PQProfile]

// SetPQProfile installs the chain-wide projection. Called once at
// chain bootstrap by the node binary from chain config. Passing nil
// restores classical semantics (test-only in production).
func SetPQProfile(p *PQProfile) {
	activePQProfile.Store(p)
}

// ActivePQProfile returns the chain-wide projection, or nil if none
// has been installed. nil means classical EVM semantics.
func ActivePQProfile() *PQProfile {
	return activePQProfile.Load()
}

// refuse is the single profile gate. Every classical precompile calls
// it once at the top of its Run() and otherwise proceeds in its own
// lane. Hot path: one atomic load plus one switch dispatch.
//
// Returns nil when the op is admissible (classical EVM, or a PQ
// profile installed that does not forbid this op); returns the
// family-specific error when the active profile forbids the op.
//
// refuse(OpUnknown) returns nil; callers MUST use a recognized Op or
// risk silently bypassing the gate.
func refuse(op Op) error {
	p := activePQProfile.Load()
	if p == nil {
		return nil
	}
	switch op {
	case OpEcrecover:
		if p.ForbidEcrecover {
			return ErrEcrecoverForbidden
		}
	case OpP256Verify:
		if p.ForbidP256Verify {
			return ErrP256VerifyForbidden
		}
	case OpSHA256:
		if p.ForbidSHA256 {
			return ErrSHA256Forbidden
		}
	case OpRIPEMD160:
		if p.ForbidRIPEMD160 {
			return ErrRIPEMD160Forbidden
		}
	case OpBlake2F:
		if p.ForbidBlake2F {
			return ErrBlake2FForbidden
		}
	case OpBn256Add, OpBn256ScalarMul, OpBn256Pairing:
		if p.ForbidBn256 {
			return ErrBn256Forbidden
		}
	case OpBLS12381G1Add, OpBLS12381G1MSM,
		OpBLS12381G2Add, OpBLS12381G2MSM,
		OpBLS12381Pairing,
		OpBLS12381MapG1, OpBLS12381MapG2:
		if p.ForbidBLS12381 {
			return ErrBLS12381Forbidden
		}
	case OpKZGPointEval:
		if p.ForbidKZG {
			return ErrKZGForbidden
		}
	}
	return nil
}
