// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"testing"
)

// pq_lux_profile_test.go pins the GUARDRAIL through the REAL EVM dispatch:
// the strict-PQ profile a Lux EVM chain installs must NOT touch the
// standard alt_bn128 (BN254) precompiles at 0x06/0x07/0x08 — they are
// Ethereum-compat for dapps, not Lux settlement-security. Every other
// classical family stays forbidden, identical to AllForbidden() minus the
// bn256 carve-out.
//
// The authoritative projection lives in the EVM plugin
// (evm/plugin/evm.LuxStrictPQ); this geth-layer test builds the same
// bn256-carve-out shape inline (a *PQProfile literal) and drives it through
// (*EVM).runPrecompile so the guardrail is proven at the dispatch chokepoint
// the chain actually uses, not just at the profile struct.

// luxStrictPQProfile is the bn256-carve-out profile shape the EVM plugin
// installs: every classical family forbidden except bn256. Built inline so
// geth carries no duplicate projection symbol (the owner is the EVM plugin).
func luxStrictPQProfile() *PQProfile {
	return &PQProfile{
		ForbidEcrecover:  true,
		ForbidP256Verify: true,
		ForbidSHA256:     true,
		ForbidRIPEMD160:  true,
		ForbidBlake2F:    true,
		ForbidBn256:      false, // Ethereum-compat carve-out
		ForbidBLS12381:   true,
		ForbidKZG:        true,
	}
}

// bn256GateCases are the standard alt_bn128 precompiles at 0x06–0x08.
func bn256GateCases() []gateCase {
	zero96 := make([]byte, 96)
	zero128 := make([]byte, 128)
	zero192 := make([]byte, 192)
	return []gateCase{
		{"bn256AddIstanbul", &bn256AddIstanbul{}, zero128, OpBn256Add, ErrBn256Forbidden},
		{"bn256AddByzantium", &bn256AddByzantium{}, zero128, OpBn256Add, ErrBn256Forbidden},
		{"bn256ScalarMulIstanbul", &bn256ScalarMulIstanbul{}, zero96, OpBn256ScalarMul, ErrBn256Forbidden},
		{"bn256ScalarMulByzantium", &bn256ScalarMulByzantium{}, zero96, OpBn256ScalarMul, ErrBn256Forbidden},
		{"bn256PairingIstanbul", &bn256PairingIstanbul{}, zero192, OpBn256Pairing, ErrBn256Forbidden},
		{"bn256PairingByzantium", &bn256PairingByzantium{}, zero192, OpBn256Pairing, ErrBn256Forbidden},
	}
}

// TestLuxStrictPQ_AdmitsBn256 proves that under the Lux strict-PQ profile,
// the standard bn256 precompiles (0x06–0x08) are NOT refused by the gate.
// (They may still fail on their own input validation — zero input is not
// a valid pairing — but never with ErrBn256Forbidden.) This is deliverable
// (f) at the geth dispatch layer: standard EVM bn256 still works under
// strict-PQ.
func TestLuxStrictPQ_AdmitsBn256(t *testing.T) {
	for _, tc := range bn256GateCases() {
		t.Run(tc.name, func(t *testing.T) {
			err := runGate(t, luxStrictPQProfile(), tc)
			if errors.Is(err, ErrBn256Forbidden) {
				t.Fatalf("Lux strict-PQ must NOT forbid standard bn256 0x06-0x08; got %v", err)
			}
		})
	}
}

// TestLuxStrictPQ_StillRefusesOtherFamilies proves the carve-out is
// surgical: every NON-bn256 classical family is still refused under the
// Lux strict-PQ profile, with its family-specific error.
func TestLuxStrictPQ_StillRefusesOtherFamilies(t *testing.T) {
	for _, tc := range gateCases() {
		// Skip the bn256 family — it is the deliberate carve-out.
		if tc.op == OpBn256Add || tc.op == OpBn256ScalarMul || tc.op == OpBn256Pairing {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			err := runGate(t, luxStrictPQProfile(), tc)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Lux strict-PQ must refuse %s with %v; got %v", tc.name, tc.wantErr, err)
			}
		})
	}
}

// TestLuxStrictPQ_FlagShape documents the exact profile shape: every
// Forbid* flag true except ForbidBn256.
func TestLuxStrictPQ_FlagShape(t *testing.T) {
	p := luxStrictPQProfile()
	if p.ForbidBn256 {
		t.Error("luxStrictPQProfile().ForbidBn256 must be false (Ethereum-compat carve-out)")
	}
	trueFlags := map[string]bool{
		"Ecrecover":  p.ForbidEcrecover,
		"P256Verify": p.ForbidP256Verify,
		"SHA256":     p.ForbidSHA256,
		"RIPEMD160":  p.ForbidRIPEMD160,
		"Blake2F":    p.ForbidBlake2F,
		"BLS12381":   p.ForbidBLS12381,
		"KZG":        p.ForbidKZG,
	}
	for name, v := range trueFlags {
		if !v {
			t.Errorf("luxStrictPQProfile().Forbid%s must be true", name)
		}
	}
}
