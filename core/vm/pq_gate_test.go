// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"testing"
)

// pq_gate_test.go — coverage for the cross-category PQ gate.
//
// Each test case installs the strict-PQ projection (AllForbidden) and
// invokes a precompile's Run() with a benign input. The assertion is
// not on the cryptographic output (the precompile never reaches its
// math under strict-PQ) but on the error type returned by refuse().

// resetGate restores both the active profile and the PQ-required flag
// between tests so package state never leaks across cases.
func resetGate(t *testing.T) {
	t.Helper()
	prev := ActivePQProfile()
	req := PQRequired()
	t.Cleanup(func() {
		SetPQProfile(prev)
		if req {
			RequirePQ()
		} else {
			ClearPQRequired()
		}
	})
}

// gateCase is one row of the gate matrix: a precompile, the Op it
// reports, the wantErr it must return when the strict profile is
// installed.
type gateCase struct {
	name    string
	run     func() error
	op      Op
	wantErr error
}

// gateCases enumerates every gated precompile and the error refuse()
// must return for that op under the strict-PQ profile.
func gateCases() []gateCase {
	// Inputs are size-conforming but otherwise benign; the gate must
	// fire before the precompile interprets them.
	zero128 := make([]byte, 128)
	zero64 := make([]byte, 64)
	zero96 := make([]byte, 96)
	zero213 := make([]byte, 213)
	zero192 := make([]byte, 192)
	zero256 := make([]byte, 256)
	zero160 := make([]byte, 160)
	zero288 := make([]byte, 288)
	zero384 := make([]byte, 384)
	// EIP-2537 G1/G2 multi-exp pair size is 160 / 288 per pair.

	return []gateCase{
		{
			name:    "ecrecover",
			op:      OpECDSARecover,
			run:     func() error { _, err := (&ecrecover{}).Run(zero128); return err },
			wantErr: ErrClassicalAuthForbidden,
		},
		{
			name:    "sha256",
			op:      OpSHA256,
			run:     func() error { _, err := (&sha256hash{}).Run(zero64); return err },
			wantErr: ErrNonSHA3HashForbidden,
		},
		{
			name:    "ripemd160",
			op:      OpRIPEMD160,
			run:     func() error { _, err := (&ripemd160hash{}).Run(zero64); return err },
			wantErr: ErrNonSHA3HashForbidden,
		},
		{
			name:    "blake2F",
			op:      OpBlake2F,
			run:     func() error { _, err := (&blake2F{}).Run(zero213); return err },
			wantErr: ErrNonSHA3HashForbidden,
		},
		{
			name:    "bn256AddIstanbul",
			op:      OpBn256Add,
			run:     func() error { _, err := (&bn256AddIstanbul{}).Run(zero128); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bn256AddByzantium",
			op:      OpBn256Add,
			run:     func() error { _, err := (&bn256AddByzantium{}).Run(zero128); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bn256ScalarMulIstanbul",
			op:      OpBn256ScalarMul,
			run:     func() error { _, err := (&bn256ScalarMulIstanbul{}).Run(zero96); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bn256ScalarMulByzantium",
			op:      OpBn256ScalarMul,
			run:     func() error { _, err := (&bn256ScalarMulByzantium{}).Run(zero96); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bn256PairingIstanbul",
			op:      OpBn256Pairing,
			run:     func() error { _, err := (&bn256PairingIstanbul{}).Run(zero192); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bn256PairingByzantium",
			op:      OpBn256Pairing,
			run:     func() error { _, err := (&bn256PairingByzantium{}).Run(zero192); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381G1Add",
			op:      OpBLS12381G1Add,
			run:     func() error { _, err := (&bls12381G1Add{}).Run(zero256); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381G1MultiExp",
			op:      OpBLS12381G1MSM,
			run:     func() error { _, err := (&bls12381G1MultiExp{}).Run(zero160); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381G2Add",
			op:      OpBLS12381G2Add,
			run:     func() error { _, err := (&bls12381G2Add{}).Run(zero384); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381G2MultiExp",
			op:      OpBLS12381G2MSM,
			run:     func() error { _, err := (&bls12381G2MultiExp{}).Run(zero288); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381Pairing",
			op:      OpBLS12381Pairing,
			run:     func() error { _, err := (&bls12381Pairing{}).Run(zero384); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381MapG1",
			op:      OpBLS12381MapG1,
			run:     func() error { _, err := (&bls12381MapG1{}).Run(zero64); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "bls12381MapG2",
			op:      OpBLS12381MapG2,
			run:     func() error { _, err := (&bls12381MapG2{}).Run(zero128); return err },
			wantErr: ErrPairingForbidden,
		},
		{
			name:    "p256Verify",
			op:      OpP256Verify,
			run:     func() error { _, err := (&p256Verify{}).Run(zero160); return err },
			wantErr: ErrClassicalAuthForbidden,
		},
		{
			name:    "kzgPointEvaluation",
			op:      OpKZGPointEval,
			run:     func() error { _, err := (&kzgPointEvaluation{}).Run(zero192); return err },
			wantErr: ErrKZGForbidden,
		},
	}
}

// TestStrictPQRefusesEveryClassicalPrecompile asserts the strict-PQ
// projection refuses every gated precompile with the expected error
// from refuse().
func TestStrictPQRefusesEveryClassicalPrecompile(t *testing.T) {
	resetGate(t)
	SetPQProfile(AllForbidden())

	for _, tc := range gateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatalf("%s: strict-PQ profile must refuse Run, got nil error", tc.name)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("%s: want err=%v, got err=%v", tc.name, tc.wantErr, err)
			}
		})
	}
}

// TestRefuseReportsMissingProfileWhenPQRequired asserts every gated
// precompile fails closed with ErrMissingPQProfile when RequirePQ has
// been called but no projection is installed.
func TestRefuseReportsMissingProfileWhenPQRequired(t *testing.T) {
	resetGate(t)
	SetPQProfile(nil)
	RequirePQ()

	for _, tc := range gateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if !errors.Is(err, ErrMissingPQProfile) {
				t.Fatalf("%s: want ErrMissingPQProfile, got %v", tc.name, err)
			}
		})
	}
}

// TestPermissiveProfileAdmitsEveryPrecompile asserts the zero-value
// profile (Permissive) lets every precompile reach its math layer.
// "Reach the math" means the precompile returns an error that is not
// one of the gate errors, or returns success — anything except the
// gate errors signals the gate did not fire.
func TestPermissiveProfileAdmitsEveryPrecompile(t *testing.T) {
	resetGate(t)
	SetPQProfile(&PQProfile{})

	gateErrors := []error{
		ErrClassicalAuthForbidden,
		ErrNonSHA3HashForbidden,
		ErrPairingForbidden,
		ErrKZGForbidden,
		ErrMissingPQProfile,
	}

	for _, tc := range gateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			for _, gateErr := range gateErrors {
				if errors.Is(err, gateErr) {
					t.Fatalf("%s: permissive profile must not refuse; got %v", tc.name, err)
				}
			}
		})
	}
}

// TestAllForbiddenSetsAllFlags asserts the AllForbidden constructor
// produces a profile with every Forbid* set true. This is the
// canonical projection a strict-PQ chain installs.
func TestAllForbiddenSetsAllFlags(t *testing.T) {
	p := AllForbidden()
	if !p.ForbidECDSAContractAuth {
		t.Error("ForbidECDSAContractAuth must be true in AllForbidden()")
	}
	if !p.ForbidNonSHA3Hashes {
		t.Error("ForbidNonSHA3Hashes must be true in AllForbidden()")
	}
	if !p.ForbidPairingPrecompiles {
		t.Error("ForbidPairingPrecompiles must be true in AllForbidden()")
	}
	if !p.ForbidKZGPrecompiles {
		t.Error("ForbidKZGPrecompiles must be true in AllForbidden()")
	}
}

// TestRefuseUnknownOpIsPermissive documents that refuse(OpUnknown)
// returns nil — callers must use a recognised Op or the gate silently
// bypasses. Refuse cannot fail-closed on OpUnknown without breaking
// callers who haven't been ported yet.
func TestRefuseUnknownOpIsPermissive(t *testing.T) {
	resetGate(t)
	SetPQProfile(AllForbidden())
	if err := refuse(OpUnknown); err != nil {
		t.Errorf("refuse(OpUnknown) under strict profile must be nil; got %v", err)
	}
}
