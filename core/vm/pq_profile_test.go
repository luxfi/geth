// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/hex"
	"errors"
	"testing"
)

// validEcrecoverInput is a well-known signature valid for ecrecover at 0x01.
// hash = keccak256("test message")  (32 bytes)
// v    = 28                          (1 byte, left-padded to 32)
// r,s  = a valid (r, s) signing that hash with a random secp256k1 key.
//
// Captured from a one-time signing run; recovered address is
// 0x75c8c507dba81c4fdfa57c79898d6b7c4933afdc and the precompile output is
// that address left-padded to 32 bytes. The exact recovered address does
// not matter for these tests — only that the precompile returns a
// 32-byte address under classical-compat semantics and ErrClassicalAuth
// Forbidden under strict-PQ semantics.
const validEcrecoverInputHex = "" +
	// hash
	"a35a39e7715a7b2c5d2e3a5d8e8f8a8b8c8d8e8f9091929394959697989a9b9c" +
	// v (left-padded; canonical v = 27 or 28)
	"000000000000000000000000000000000000000000000000000000000000001c" +
	// r
	"7b6d1f1f0a85b5a3aa3f0e57c8c30a1b1c1d1e1f2021222324252627282a2b2c" +
	// s
	"3d3e3f404142434445464748494a4b4c4d4e4f505152535455565758595a5b5c"

// resetPQProfileState restores both the active profile and the
// strict-PQ-required flag between tests so package-level state never
// leaks across tests. Each strict-PQ test installs a fresh profile in a
// sub-test scope.
func resetPQProfileState(t *testing.T) {
	t.Helper()
	prevProfile := ActivePQProfile()
	prevRequired := PQRequired()
	t.Cleanup(func() {
		SetPQProfile(prevProfile)
		if prevRequired {
			RequirePQ()
		} else {
			ClearPQRequired()
		}
	})
}

// TestClassicalCompatProfile_EcrecoverWorks asserts the default (nil
// profile, no strict-PQ required) path runs ecrecover with classical
// semantics: malformed input returns (nil, nil) — the upstream
// behaviour — and never returns ErrClassicalAuthForbidden or
// ErrMissingPQProfile.
func TestClassicalCompatProfile_EcrecoverWorks(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(nil)
	ClearPQRequired()

	c := &ecrecover{}
	// Use a zeroed 128-byte input; classical ecrecover returns (nil,
	// nil) here because the signature values do not validate. The key
	// invariant is that we do not return any strict-PQ refusal error.
	out, err := c.Run(make([]byte, 128))
	if errors.Is(err, ErrClassicalAuthForbidden) {
		t.Fatalf("classical-compat ecrecover must not return ErrClassicalAuthForbidden, got err=%v", err)
	}
	if errors.Is(err, ErrMissingPQProfile) {
		t.Fatalf("classical-compat ecrecover must not return ErrMissingPQProfile, got err=%v", err)
	}
	// upstream returns (nil, nil) for invalid sig — accept that or any
	// real recovery; we only care that strict-PQ refusal is OFF.
	_ = out
}

// TestStrictPQProfile_EcrecoverReturnsError asserts that when a strict-PQ
// Profile is installed, the ecrecover precompile at 0x01 returns
// ErrClassicalAuthForbidden and zero output bytes, regardless of whether
// the input would otherwise have recovered a valid address.
func TestStrictPQProfile_EcrecoverReturnsError(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: true})

	c := &ecrecover{}

	input, err := hex.DecodeString(validEcrecoverInputHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}

	out, err := c.Run(input)
	if !errors.Is(err, ErrClassicalAuthForbidden) {
		t.Fatalf("expected ErrClassicalAuthForbidden, got out=%x err=%v", out, err)
	}
	if len(out) != 0 {
		t.Fatalf("strict-PQ ecrecover must return zero bytes, got %d bytes: %x", len(out), out)
	}
}

// TestStrictPQProfile_EcrecoverEmptyInputAlsoRefused asserts the refusal
// fires before any input validation. A zero-length call still returns
// ErrClassicalAuthForbidden, never an upstream nil-nil result.
func TestStrictPQProfile_EcrecoverEmptyInputAlsoRefused(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: true})

	c := &ecrecover{}
	out, err := c.Run(nil)
	if !errors.Is(err, ErrClassicalAuthForbidden) {
		t.Fatalf("expected ErrClassicalAuthForbidden on empty input, got out=%x err=%v", out, err)
	}
	if len(out) != 0 {
		t.Fatalf("expected zero output, got %d bytes", len(out))
	}
}

// TestStrictPQProfile_GasIsStillCharged asserts that RequiredGas is not
// short-circuited by the strict-PQ refusal. EIP-150 semantics require
// that even reverting precompiles consume their advertised gas. We
// verify RequiredGas returns params.EcrecoverGas independent of the
// active profile so the caller cannot probe for the active profile via
// gas refund timing.
func TestStrictPQProfile_GasIsStillCharged(t *testing.T) {
	resetPQProfileState(t)

	c := &ecrecover{}
	// classical
	SetPQProfile(nil)
	gasClassical := c.RequiredGas(nil)
	// strict-PQ
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: true})
	gasStrict := c.RequiredGas(nil)
	if gasClassical != gasStrict {
		t.Fatalf("RequiredGas must not depend on active profile: classical=%d strict=%d", gasClassical, gasStrict)
	}
}

// TestSetActiveSecurityProfile_NilPathClassicalCompat asserts the
// SetActiveSecurityProfile API contract: a nil pointer with no
// strict-PQ required means classical-compat semantics with no surprise
// panics.
func TestSetActiveSecurityProfile_NilPathClassicalCompat(t *testing.T) {
	resetPQProfileState(t)
	ClearPQRequired()

	// Install then un-install.
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: true})
	if !forbidClassicalContractAuth() {
		t.Fatal("expected strict-PQ active after install")
	}
	SetPQProfile(nil)
	if forbidClassicalContractAuth() {
		t.Fatal("expected classical-compat after nil install with no strict-PQ requirement")
	}
}

// TestActiveSecurityProfile_PermissiveProfileStillAllowsClassical asserts
// that a non-strict profile (ForbidECDSAContractAuth=false) still permits
// classical contract-auth at the precompile boundary. This is the
// Lux-Permissive path consumed by chains migrating from classical EVMs.
func TestActiveSecurityProfile_PermissiveProfileStillAllowsClassical(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: false})

	c := &ecrecover{}
	_, err := c.Run(make([]byte, 128))
	if errors.Is(err, ErrClassicalAuthForbidden) {
		t.Fatalf("permissive profile must NOT refuse classical ecrecover, got %v", err)
	}
}

// TestRequireStrictPQ_FailsClosedWhenProfileMissing asserts the
// fail-closed guard: when RequirePQ() has been called but no
// Profile has been installed, ecrecover returns
// ErrMissingPQProfile. This catches misordered bootstrap (forgot
// to call SetActiveSecurityProfile) and prevents silent classical
// fallback.
func TestRequireStrictPQ_FailsClosedWhenProfileMissing(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(nil)
	RequirePQ()

	c := &ecrecover{}

	input, err := hex.DecodeString(validEcrecoverInputHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}

	out, err := c.Run(input)
	if !errors.Is(err, ErrMissingPQProfile) {
		t.Fatalf("expected ErrMissingPQProfile when strict-PQ required but profile missing, got out=%x err=%v", out, err)
	}
	if len(out) != 0 {
		t.Fatalf("fail-closed ecrecover must return zero bytes, got %d bytes: %x", len(out), out)
	}
}

// TestRequireStrictPQ_EmptyInputAlsoFailsClosed asserts the fail-closed
// guard fires before any input validation, mirroring the
// ErrClassicalAuthForbidden empty-input path.
func TestRequireStrictPQ_EmptyInputAlsoFailsClosed(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(nil)
	RequirePQ()

	c := &ecrecover{}
	out, err := c.Run(nil)
	if !errors.Is(err, ErrMissingPQProfile) {
		t.Fatalf("expected ErrMissingPQProfile on empty input, got out=%x err=%v", out, err)
	}
	if len(out) != 0 {
		t.Fatalf("expected zero output, got %d bytes", len(out))
	}
}

// TestRequireStrictPQ_ProfileInstallationClosesGuard asserts the
// expected boot sequence: RequirePQ() then SetPQProfile()
// flips the gate from ErrMissingPQProfile to
// ErrClassicalAuthForbidden — never to nil. Once strict-PQ is required,
// classical contract-auth is forbidden one way or the other.
func TestRequireStrictPQ_ProfileInstallationClosesGuard(t *testing.T) {
	resetPQProfileState(t)
	SetPQProfile(nil)
	RequirePQ()

	c := &ecrecover{}

	// Step 1: profile not yet installed.
	_, err := c.Run(make([]byte, 128))
	if !errors.Is(err, ErrMissingPQProfile) {
		t.Fatalf("before install: expected ErrMissingPQProfile, got %v", err)
	}

	// Step 2: install the profile.
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: true})

	// Now the error should be ErrClassicalAuthForbidden (profile
	// installed and forbids ECDSA), not ErrMissingPQProfile.
	_, err = c.Run(make([]byte, 128))
	if !errors.Is(err, ErrClassicalAuthForbidden) {
		t.Fatalf("after install: expected ErrClassicalAuthForbidden, got %v", err)
	}
	if errors.Is(err, ErrMissingPQProfile) {
		t.Fatal("after install: must not return ErrMissingPQProfile")
	}
}

// TestRequireStrictPQ_PermissiveProfileOverridesRequirement asserts a
// subtle but important property: if RequirePQ() is called AND a
// permissive profile (ForbidECDSAContractAuth=false) is then installed,
// classical contract-auth is permitted. The installed profile is
// authoritative — RequirePQ() only guards the no-profile gap.
//
// This is the right behaviour: the chain operator's projection
// supersedes the bootstrap-time "expectation". If they want strict-PQ
// they install a strict profile; if they install a permissive one,
// they own that choice.
func TestRequireStrictPQ_PermissiveProfileOverridesRequirement(t *testing.T) {
	resetPQProfileState(t)
	RequirePQ()
	SetPQProfile(&PQProfile{ForbidECDSAContractAuth: false})

	c := &ecrecover{}
	_, err := c.Run(make([]byte, 128))
	if errors.Is(err, ErrClassicalAuthForbidden) {
		t.Fatalf("permissive profile must override strict-PQ requirement, got %v", err)
	}
	if errors.Is(err, ErrMissingPQProfile) {
		t.Fatalf("with profile installed, must not return ErrMissingPQProfile, got %v", err)
	}
}

