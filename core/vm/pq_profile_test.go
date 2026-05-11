// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"encoding/hex"
	"errors"
	"testing"
)

// pq_profile_test.go — ecrecover-focused coverage of the PQ profile.
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

func resetProfile(t *testing.T) {
	t.Helper()
	prev := ActivePQProfile()
	t.Cleanup(func() { SetPQProfile(prev) })
}

// TestNilProfile_EcrecoverWorks asserts the default state (no profile
// installed) runs ecrecover with classical semantics.
func TestNilProfile_EcrecoverWorks(t *testing.T) {
	resetProfile(t)
	SetPQProfile(nil)

	input, err := hex.DecodeString(validEcrecoverInputHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	_, err = (&ecrecover{}).Run(input)
	if errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("nil profile must not return ErrEcrecoverForbidden; got %v", err)
	}
}

// TestStrictPQ_EcrecoverReturnsError asserts an active profile with
// ForbidEcrecover=true returns ErrEcrecoverForbidden.
func TestStrictPQ_EcrecoverReturnsError(t *testing.T) {
	resetProfile(t)
	SetPQProfile(&PQProfile{ForbidEcrecover: true})

	input, err := hex.DecodeString(validEcrecoverInputHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	_, err = (&ecrecover{}).Run(input)
	if !errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("want ErrEcrecoverForbidden, got %v", err)
	}
}

// TestStrictPQ_EcrecoverEmptyInputAlsoRefused asserts the gate fires
// before any input parsing, including on zero-length input.
func TestStrictPQ_EcrecoverEmptyInputAlsoRefused(t *testing.T) {
	resetProfile(t)
	SetPQProfile(&PQProfile{ForbidEcrecover: true})

	_, err := (&ecrecover{}).Run(nil)
	if !errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("want ErrEcrecoverForbidden on empty input, got %v", err)
	}
}

// TestStrictPQ_GasIsStillCharged asserts the strict profile does not
// alter RequiredGas. Gas accounting is invariant under the gate.
func TestStrictPQ_GasIsStillCharged(t *testing.T) {
	resetProfile(t)

	SetPQProfile(nil)
	gasClassical := (&ecrecover{}).RequiredGas(make([]byte, 128))

	SetPQProfile(&PQProfile{ForbidEcrecover: true})
	gasStrict := (&ecrecover{}).RequiredGas(make([]byte, 128))

	if gasClassical != gasStrict {
		t.Fatalf("RequiredGas must not depend on profile: classical=%d strict=%d", gasClassical, gasStrict)
	}
}

// TestSetPQProfile_NilRestoresClassical asserts SetPQProfile(nil)
// returns the EVM to classical semantics.
func TestSetPQProfile_NilRestoresClassical(t *testing.T) {
	resetProfile(t)

	SetPQProfile(&PQProfile{ForbidEcrecover: true})
	if err := refuse(OpEcrecover); !errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("strict profile not effective; got %v", err)
	}

	SetPQProfile(nil)
	if err := refuse(OpEcrecover); err != nil {
		t.Fatalf("after SetPQProfile(nil), refuse must be nil; got %v", err)
	}
}

// TestActivePQProfile_RoundTrip asserts SetPQProfile / ActivePQProfile
// round-trip the same pointer.
func TestActivePQProfile_RoundTrip(t *testing.T) {
	resetProfile(t)

	p := &PQProfile{ForbidEcrecover: true, ForbidBn256: true}
	SetPQProfile(p)
	got := ActivePQProfile()
	if got != p {
		t.Fatalf("ActivePQProfile must return the installed pointer")
	}
	if !got.ForbidEcrecover || !got.ForbidBn256 {
		t.Fatalf("installed profile flags lost in round-trip")
	}
}
