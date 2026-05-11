// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"testing"
)

// pq_gate_test.go — coverage for the cross-family PQ gate.
//
// Each gate case names a precompile, the Op it reports, and the
// family-specific error refuse() must return when AllForbidden() is
// the active profile. resetGate restores the prior profile after each
// test so package state never leaks.

func resetGate(t *testing.T) {
	t.Helper()
	prev := ActivePQProfile()
	t.Cleanup(func() { SetPQProfile(prev) })
}

// gateCase is one row of the gate matrix.
type gateCase struct {
	name    string
	run     func() error
	op      Op
	wantErr error
}

// gateCases enumerates every gated precompile and the error refuse()
// must return for that op under AllForbidden().
func gateCases() []gateCase {
	// Inputs are size-conforming but otherwise zero. The gate fires
	// before the precompile interprets them; cryptographic output is
	// irrelevant.
	zero64 := make([]byte, 64)
	zero96 := make([]byte, 96)
	zero128 := make([]byte, 128)
	zero160 := make([]byte, 160)
	zero192 := make([]byte, 192)
	zero213 := make([]byte, 213)
	zero256 := make([]byte, 256)
	zero288 := make([]byte, 288)
	zero384 := make([]byte, 384)

	return []gateCase{
		{"ecrecover", func() error { _, e := (&ecrecover{}).Run(zero128); return e }, OpEcrecover, ErrEcrecoverForbidden},
		{"sha256", func() error { _, e := (&sha256hash{}).Run(zero64); return e }, OpSHA256, ErrSHA256Forbidden},
		{"ripemd160", func() error { _, e := (&ripemd160hash{}).Run(zero64); return e }, OpRIPEMD160, ErrRIPEMD160Forbidden},
		{"blake2F", func() error { _, e := (&blake2F{}).Run(zero213); return e }, OpBlake2F, ErrBlake2FForbidden},
		{"bn256AddIstanbul", func() error { _, e := (&bn256AddIstanbul{}).Run(zero128); return e }, OpBn256Add, ErrBn256Forbidden},
		{"bn256AddByzantium", func() error { _, e := (&bn256AddByzantium{}).Run(zero128); return e }, OpBn256Add, ErrBn256Forbidden},
		{"bn256ScalarMulIstanbul", func() error { _, e := (&bn256ScalarMulIstanbul{}).Run(zero96); return e }, OpBn256ScalarMul, ErrBn256Forbidden},
		{"bn256ScalarMulByzantium", func() error { _, e := (&bn256ScalarMulByzantium{}).Run(zero96); return e }, OpBn256ScalarMul, ErrBn256Forbidden},
		{"bn256PairingIstanbul", func() error { _, e := (&bn256PairingIstanbul{}).Run(zero192); return e }, OpBn256Pairing, ErrBn256Forbidden},
		{"bn256PairingByzantium", func() error { _, e := (&bn256PairingByzantium{}).Run(zero192); return e }, OpBn256Pairing, ErrBn256Forbidden},
		{"bls12381G1Add", func() error { _, e := (&bls12381G1Add{}).Run(zero256); return e }, OpBLS12381G1Add, ErrBLS12381Forbidden},
		{"bls12381G1MultiExp", func() error { _, e := (&bls12381G1MultiExp{}).Run(zero160); return e }, OpBLS12381G1MSM, ErrBLS12381Forbidden},
		{"bls12381G2Add", func() error { _, e := (&bls12381G2Add{}).Run(zero384); return e }, OpBLS12381G2Add, ErrBLS12381Forbidden},
		{"bls12381G2MultiExp", func() error { _, e := (&bls12381G2MultiExp{}).Run(zero288); return e }, OpBLS12381G2MSM, ErrBLS12381Forbidden},
		{"bls12381Pairing", func() error { _, e := (&bls12381Pairing{}).Run(zero384); return e }, OpBLS12381Pairing, ErrBLS12381Forbidden},
		{"bls12381MapG1", func() error { _, e := (&bls12381MapG1{}).Run(zero64); return e }, OpBLS12381MapG1, ErrBLS12381Forbidden},
		{"bls12381MapG2", func() error { _, e := (&bls12381MapG2{}).Run(zero128); return e }, OpBLS12381MapG2, ErrBLS12381Forbidden},
		{"p256Verify", func() error { _, e := (&p256Verify{}).Run(zero160); return e }, OpP256Verify, ErrP256VerifyForbidden},
		{"kzgPointEvaluation", func() error { _, e := (&kzgPointEvaluation{}).Run(zero192); return e }, OpKZGPointEval, ErrKZGForbidden},
	}
}

// TestStrictPQRefusesEveryClassicalPrecompile asserts AllForbidden()
// refuses every gated precompile with the family-specific error.
func TestStrictPQRefusesEveryClassicalPrecompile(t *testing.T) {
	resetGate(t)
	SetPQProfile(AllForbidden())

	for _, tc := range gateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatalf("strict-PQ profile must refuse Run, got nil")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestNilProfileAdmitsEveryPrecompile asserts the default state
// (no profile installed) lets every precompile reach its math.
func TestNilProfileAdmitsEveryPrecompile(t *testing.T) {
	resetGate(t)
	SetPQProfile(nil)

	gateErrors := []error{
		ErrEcrecoverForbidden, ErrP256VerifyForbidden,
		ErrSHA256Forbidden, ErrRIPEMD160Forbidden, ErrBlake2FForbidden,
		ErrBn256Forbidden, ErrBLS12381Forbidden, ErrKZGForbidden,
	}

	for _, tc := range gateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			for _, ge := range gateErrors {
				if errors.Is(err, ge) {
					t.Fatalf("nil profile must not refuse; got %v", err)
				}
			}
		})
	}
}

// TestEmptyProfileAdmitsEveryPrecompile asserts a profile with every
// Forbid* false (the zero value) admits every precompile, just like
// the nil case.
func TestEmptyProfileAdmitsEveryPrecompile(t *testing.T) {
	resetGate(t)
	SetPQProfile(&PQProfile{})

	gateErrors := []error{
		ErrEcrecoverForbidden, ErrP256VerifyForbidden,
		ErrSHA256Forbidden, ErrRIPEMD160Forbidden, ErrBlake2FForbidden,
		ErrBn256Forbidden, ErrBLS12381Forbidden, ErrKZGForbidden,
	}

	for _, tc := range gateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			for _, ge := range gateErrors {
				if errors.Is(err, ge) {
					t.Fatalf("empty profile must not refuse; got %v", err)
				}
			}
		})
	}
}

// TestPerFamilyIsolation asserts each Forbid* flag refuses only its
// own family. Setting ForbidBn256 alone must not refuse BLS12-381,
// SHA-256, or any other family.
func TestPerFamilyIsolation(t *testing.T) {
	resetGate(t)

	cases := []struct {
		name    string
		profile *PQProfile
		op      Op
		wantErr error
	}{
		{"only-ecrecover", &PQProfile{ForbidEcrecover: true}, OpEcrecover, ErrEcrecoverForbidden},
		{"only-p256", &PQProfile{ForbidP256Verify: true}, OpP256Verify, ErrP256VerifyForbidden},
		{"only-sha256", &PQProfile{ForbidSHA256: true}, OpSHA256, ErrSHA256Forbidden},
		{"only-ripemd", &PQProfile{ForbidRIPEMD160: true}, OpRIPEMD160, ErrRIPEMD160Forbidden},
		{"only-blake2", &PQProfile{ForbidBlake2F: true}, OpBlake2F, ErrBlake2FForbidden},
		{"only-bn256", &PQProfile{ForbidBn256: true}, OpBn256Pairing, ErrBn256Forbidden},
		{"only-bls", &PQProfile{ForbidBLS12381: true}, OpBLS12381Pairing, ErrBLS12381Forbidden},
		{"only-kzg", &PQProfile{ForbidKZG: true}, OpKZGPointEval, ErrKZGForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			SetPQProfile(c.profile)
			if err := refuse(c.op); !errors.Is(err, c.wantErr) {
				t.Fatalf("want %v, got %v", c.wantErr, err)
			}
			// Every other op must pass under this single-flag profile.
			otherOps := []Op{
				OpEcrecover, OpP256Verify, OpSHA256, OpRIPEMD160, OpBlake2F,
				OpBn256Add, OpBn256ScalarMul, OpBn256Pairing,
				OpBLS12381G1Add, OpBLS12381G1MSM,
				OpBLS12381G2Add, OpBLS12381G2MSM,
				OpBLS12381Pairing, OpBLS12381MapG1, OpBLS12381MapG2,
				OpKZGPointEval,
			}
			for _, op := range otherOps {
				if op == c.op {
					continue
				}
				// Skip ops in the same family as the forbidden one.
				if sameFamily(op, c.op) {
					continue
				}
				if err := refuse(op); err != nil {
					t.Errorf("single-flag profile %s also refused %v: %v", c.name, op, err)
				}
			}
		})
	}
}

// sameFamily reports whether two ops share a Forbid* flag.
func sameFamily(a, b Op) bool {
	bn := func(op Op) bool {
		return op == OpBn256Add || op == OpBn256ScalarMul || op == OpBn256Pairing
	}
	bls := func(op Op) bool {
		switch op {
		case OpBLS12381G1Add, OpBLS12381G1MSM,
			OpBLS12381G2Add, OpBLS12381G2MSM,
			OpBLS12381Pairing, OpBLS12381MapG1, OpBLS12381MapG2:
			return true
		}
		return false
	}
	switch {
	case bn(a) && bn(b):
		return true
	case bls(a) && bls(b):
		return true
	}
	return false
}

// TestAllForbiddenSetsAllFlags asserts AllForbidden() sets every
// Forbid* flag. The canonical strict-PQ projection.
func TestAllForbiddenSetsAllFlags(t *testing.T) {
	p := AllForbidden()
	cases := map[string]bool{
		"Ecrecover":  p.ForbidEcrecover,
		"P256Verify": p.ForbidP256Verify,
		"SHA256":     p.ForbidSHA256,
		"RIPEMD160":  p.ForbidRIPEMD160,
		"Blake2F":    p.ForbidBlake2F,
		"Bn256":      p.ForbidBn256,
		"BLS12381":   p.ForbidBLS12381,
		"KZG":        p.ForbidKZG,
	}
	for name, v := range cases {
		if !v {
			t.Errorf("AllForbidden().Forbid%s must be true", name)
		}
	}
}

// TestRefuseUnknownOpIsPermissive documents that refuse(OpUnknown)
// returns nil even under strict-PQ. Callers MUST use a recognized Op;
// the gate cannot fail-closed on unknown ops without breaking callers
// who have not been ported yet.
func TestRefuseUnknownOpIsPermissive(t *testing.T) {
	resetGate(t)
	SetPQProfile(AllForbidden())
	if err := refuse(OpUnknown); err != nil {
		t.Errorf("refuse(OpUnknown) must be nil; got %v", err)
	}
}
