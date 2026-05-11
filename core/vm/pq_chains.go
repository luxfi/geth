// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

// pq_chains.go — canonical chain-to-profile mapping.
//
// One function. One source of truth. The node binary reads chain
// config at bootstrap and calls SetPQProfile(ProfileFor(chain)).
//
// Mapping (one-and-only-one):
//
//   liquidity   →  AllForbidden() — strict PQ only
//   lux         →  nil            — classical EVM
//   zoo         →  nil            — classical EVM
//   hanzo       →  nil            — classical EVM
//   <other>     →  nil            — classical EVM (default)
//
// A chain that wants something between strict and classical builds its
// own *PQProfile literal at the call site; ProfileFor only encodes the
// canonical defaults.

// Chain is the canonical brand identifier passed to ProfileFor.
type Chain string

// Canonical chain identifiers. String values are stable and match the
// genesis-config "chain" field; renaming is a backward-incompatible
// change to chain config files.
const (
	ChainLiquidity Chain = "liquidity"
	ChainLux       Chain = "lux"
	ChainZoo       Chain = "zoo"
	ChainHanzo     Chain = "hanzo"
)

// ProfileFor returns the canonical PQ profile for a chain. nil means
// classical EVM semantics. The node binary calls
// SetPQProfile(ProfileFor(chain)) once at bootstrap.
func ProfileFor(chain Chain) *PQProfile {
	switch chain {
	case ChainLiquidity:
		return AllForbidden()
	case ChainLux, ChainZoo, ChainHanzo:
		return nil
	}
	return nil
}
