// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"sync/atomic"
)

// lux_security_profile.go — strict-PQ profile gate inside the EVM
// precompile layer.
//
// The canonical chain-wide ChainSecurityProfile lives in
// github.com/luxfi/consensus/config and is loaded at chain bootstrap by
// the genesis loader (see luxfi/genesis/pkg/genesis/security_profile.go).
// The node binary copies the two flags that the EVM precompile layer
// needs into a LuxSecurityProfile and installs it here via
// SetActiveSecurityProfile.
//
// We deliberately do NOT import the consensus package from core/vm:
//   - core/vm must stay free of higher-level dependencies.
//   - Only two booleans are needed at this layer; the wider profile is
//     irrelevant to the EVM gas/auth gate.
//
// Default (no profile set) preserves classical EVM semantics. Strict-PQ
// chains MUST install a profile with ForbidECDSAContractAuth=true at
// boot, before any EVM execution; otherwise ecrecover at 0x01 will keep
// returning recovered addresses.
//
// One way only: install once at boot, read with an atomic load on every
// precompile call. No per-tx mutation, no per-call configuration.

// LuxSecurityProfile is the subset of consensusconfig.ChainSecurityProfile
// that the EVM precompile layer enforces. The node binary populates it
// from the genesis-resolved profile and installs it with
// SetActiveSecurityProfile during chain bootstrap.
type LuxSecurityProfile struct {
	// ForbidECDSAContractAuth refuses any contract-authorisation primitive
	// that names classical ECDSA. When true, the ecrecover precompile at
	// 0x01 returns ErrClassicalAuthForbidden instead of a recovered
	// address. Strict-PQ profiles set this true.
	ForbidECDSAContractAuth bool
}

// ErrClassicalAuthForbidden is returned by classical contract-auth
// primitives (currently ecrecover at 0x01) when the active
// LuxSecurityProfile has ForbidECDSAContractAuth=true. This is a hard
// error: the precompile reverts with this message rather than producing
// a value the contract could be tricked into trusting.
var ErrClassicalAuthForbidden = errors.New("classical contract-auth forbidden by chain security profile (strict-PQ)")

// activeSecurityProfile holds the chain-wide LuxSecurityProfile installed
// at bootstrap. nil means classical-compat semantics; in that mode all
// classical precompiles run as in upstream go-ethereum.
var activeSecurityProfile atomic.Pointer[LuxSecurityProfile]

// SetActiveSecurityProfile installs the chain-wide LuxSecurityProfile.
// The node binary calls this once at chain bootstrap, after the genesis
// loader has resolved the canonical ChainSecurityProfile and verified
// its hash. Subsequent EVM execution reads the profile via an atomic
// load. Passing nil restores classical-compat semantics (test-only).
func SetActiveSecurityProfile(p *LuxSecurityProfile) {
	activeSecurityProfile.Store(p)
}

// ActiveSecurityProfile returns the chain-wide LuxSecurityProfile, or
// nil if none has been installed. Callers must handle nil as
// classical-compat semantics.
func ActiveSecurityProfile() *LuxSecurityProfile {
	return activeSecurityProfile.Load()
}

// forbidClassicalContractAuth reports whether the chain currently
// forbids classical contract-auth primitives. Equivalent to
// ActiveSecurityProfile().ForbidECDSAContractAuth with nil-safety. Hot
// path on every ecrecover call; one atomic.Pointer load.
func forbidClassicalContractAuth() bool {
	p := activeSecurityProfile.Load()
	if p == nil {
		return false
	}
	return p.ForbidECDSAContractAuth
}
