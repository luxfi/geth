// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"sync/atomic"
)

// pq_profile.go — PQ profile gate inside the EVM precompile layer.
//
// PQ mode is binary: a chain is PQ or it isn't. The canonical chain-
// wide ChainSecurityProfile lives in github.com/luxfi/consensus/config.
// The node binary projects the subset that the EVM precompile layer
// needs into a PQProfile and installs it here via SetPQProfile at
// chain bootstrap.
//
// We deliberately do NOT import the consensus package from core/vm:
//   - core/vm must stay free of higher-level dependencies.
//   - Only a couple of booleans are needed at this layer; the wider
//     profile is irrelevant to the EVM gas/auth gate.
//
// Default (no profile set, no PQ required) preserves classical EVM
// semantics.
//
// PQ bootstrap is a two-call protocol:
//
//	1. RequirePQ()             // record "this chain MUST be PQ"
//	2. SetPQProfile(&PQProfile{
//	       ForbidECDSAContractAuth: true,
//	   })
//
// Step 1 fails the gate closed: until step 2 installs the projection,
// any classical contract-auth precompile call returns ErrMissingPQProfile.
// This catches the case where chain config declared PQ but the
// bootstrap path forgot to (or could not) install the projection.
//
// One way only: install once at boot, read with atomic loads on every
// precompile call. No per-tx mutation, no per-call configuration. No
// alias names — the type is PQProfile, the setter is SetPQProfile,
// the require flag is RequirePQ. One name per concept.

// PQProfile is the subset of consensusconfig.ChainSecurityProfile that
// the EVM precompile layer enforces. The node binary populates it from
// the genesis-resolved profile and installs it with SetPQProfile during
// chain bootstrap.
type PQProfile struct {
	// ForbidECDSAContractAuth refuses any contract-authorisation
	// primitive that names classical ECDSA. When true, the ecrecover
	// precompile at 0x01 returns ErrClassicalAuthForbidden instead of
	// a recovered address. PQ profiles set this true.
	ForbidECDSAContractAuth bool
}

// ErrClassicalAuthForbidden is returned by classical contract-auth
// primitives (currently ecrecover at 0x01) when the active PQProfile
// has ForbidECDSAContractAuth=true. This is a hard error: the
// precompile reverts with this message rather than producing a value
// the contract could be tricked into trusting.
var ErrClassicalAuthForbidden = errors.New("classical contract-auth forbidden by chain security profile (PQ)")

// ErrMissingPQProfile is returned by classical contract-auth primitives
// when RequirePQ() has been called but the chain-wide PQProfile
// projection has not yet been installed via SetPQProfile. PQ chains
// MUST install the projection before any EVM execution; if they don't,
// every classical contract-auth call fails closed with this error.
var ErrMissingPQProfile = errors.New("PQ chain expected but profile projection not installed in EVM")

// activePQProfile holds the chain-wide PQProfile installed at
// bootstrap. nil means "no projection installed"; combined with
// pqRequired this determines fail-open vs fail-closed semantics.
var activePQProfile atomic.Pointer[PQProfile]

// pqRequired records whether the chain declared PQ in its
// configuration. When true and activePQProfile is nil, classical
// contract-auth precompiles return ErrMissingPQProfile (fail closed).
// Set once at bootstrap by RequirePQ(); never cleared in production.
var pqRequired atomic.Bool

// SetPQProfile installs the chain-wide PQProfile. The node binary
// calls this once at chain bootstrap, after the genesis loader has
// resolved the canonical ChainSecurityProfile and verified its hash.
// Subsequent EVM execution reads the profile via an atomic load.
// Passing nil restores "no projection installed" (test-only).
func SetPQProfile(p *PQProfile) {
	activePQProfile.Store(p)
}

// ActivePQProfile returns the chain-wide PQProfile, or nil if none has
// been installed. Callers must combine the result with PQRequired() to
// interpret nil correctly.
func ActivePQProfile() *PQProfile {
	return activePQProfile.Load()
}

// RequirePQ records that the chain has been configured as PQ. Any
// classical contract-auth call that finds the projection not installed
// will return ErrMissingPQProfile. Called once at bootstrap before
// SetPQProfile (so any race window or out-of-order boot fails closed).
func RequirePQ() {
	pqRequired.Store(true)
}

// ClearPQRequired un-sets the PQ expectation. Test-only.
func ClearPQRequired() {
	pqRequired.Store(false)
}

// PQRequired reports whether RequirePQ() has been called.
// Test/diagnostic accessor.
func PQRequired() bool {
	return pqRequired.Load()
}

// classicalContractAuthCheck is the single decision function for
// classical contract-auth precompiles (currently just ecrecover at
// 0x01). Hot path: at most two atomic loads.
//
// Returns:
//   - nil                           proceed with classical semantics
//   - ErrClassicalAuthForbidden     profile installed, forbids ECDSA
//   - ErrMissingPQProfile           PQ required, profile missing
func classicalContractAuthCheck() error {
	if p := activePQProfile.Load(); p != nil {
		if p.ForbidECDSAContractAuth {
			return ErrClassicalAuthForbidden
		}
		return nil
	}
	if pqRequired.Load() {
		return ErrMissingPQProfile
	}
	return nil
}

// forbidClassicalContractAuth reports whether the chain currently
// refuses classical contract-auth, for either reason. Kept as a thin
// boolean wrapper around classicalContractAuthCheck for callers that
// only need yes/no.
func forbidClassicalContractAuth() bool {
	return classicalContractAuthCheck() != nil
}
