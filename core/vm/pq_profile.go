// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"github.com/luxfi/pq"
)

// pq_profile.go — EVM-side bindings for github.com/luxfi/pq.
//
// luxfi/pq is the single source of truth for the strict-PQ profile.
// This file re-exports its types and constants so call sites inside
// core/vm read naturally (vm.PQProfile, vm.OpEcrecover, etc.) without
// the package qualifier.
//
// Naming symmetry: every name in luxfi/pq has a vm-local alias with
// the "PQ" prefix where needed to disambiguate from existing vm
// identifiers (Op collides with the opcode type, so we keep the full
// pq.Op constants like OpEcrecover unprefixed only because vm has no
// OpEcrecover of its own).
//
// One way only: chain bootstrap calls vm.SetPQProfile (= pq.SetActive),
// each gated precompile calls refuse() (= pq.Refuse).

// PQProfile aliases [pq.Profile]: the strict-PQ profile value.
type PQProfile = pq.Profile

// Op aliases [pq.Op]: the precompile-family enum.
type Op = pq.Op

// Op constants — direct re-export of [pq.Op*].
const (
	OpUnknown         = pq.OpUnknown
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

// SetPQProfile installs the chain-wide projection (alias for
// [pq.SetActive]). Called once at chain bootstrap.
func SetPQProfile(p *PQProfile) { pq.SetActive(p) }

// ActivePQProfile returns the chain-wide projection (alias for
// [pq.Active]).
func ActivePQProfile() *PQProfile { return pq.Active() }

// refuse is the single profile gate every classical precompile calls
// at the top of its Run(). Hot path: one atomic load plus one switch
// dispatch inside [pq.Refuse].
func refuse(op Op) error { return pq.Refuse(op) }
