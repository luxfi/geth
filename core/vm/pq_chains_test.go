// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"testing"
)

// TestProfileFor_LiquidityIsStrict asserts Liquidity gets AllForbidden.
func TestProfileFor_LiquidityIsStrict(t *testing.T) {
	resetProfile(t)
	p := ProfileFor(ChainLiquidity)
	if p == nil {
		t.Fatal("ChainLiquidity must return AllForbidden(), got nil")
	}
	want := AllForbidden()
	if *p != *want {
		t.Fatalf("ChainLiquidity profile mismatch\n got %+v\nwant %+v", *p, *want)
	}
}

// TestProfileFor_LuxZooHanzoAreClassical asserts the three Lux-family
// chains get classical EVM (nil profile).
func TestProfileFor_LuxZooHanzoAreClassical(t *testing.T) {
	for _, c := range []Chain{ChainLux, ChainZoo, ChainHanzo} {
		t.Run(string(c), func(t *testing.T) {
			if p := ProfileFor(c); p != nil {
				t.Fatalf("Chain %s must return nil; got %+v", c, *p)
			}
		})
	}
}

// TestProfileFor_UnknownIsClassical asserts unknown chain names get
// the safe classical default.
func TestProfileFor_UnknownIsClassical(t *testing.T) {
	if p := ProfileFor(Chain("xchain")); p != nil {
		t.Fatalf("unknown chain must return nil; got %+v", *p)
	}
}

// TestProfileFor_EndToEnd_LiquidityRefusesEcrecover asserts the
// liquidity chain projection, when installed, refuses ecrecover.
func TestProfileFor_EndToEnd_LiquidityRefusesEcrecover(t *testing.T) {
	resetProfile(t)
	SetPQProfile(ProfileFor(ChainLiquidity))

	_, err := (&ecrecover{}).Run(make([]byte, 128))
	if !errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("liquidity chain must refuse ecrecover; got %v", err)
	}
}

// TestProfileFor_EndToEnd_LuxAdmitsEcrecover asserts the lux chain
// projection, when installed (nil), admits ecrecover.
func TestProfileFor_EndToEnd_LuxAdmitsEcrecover(t *testing.T) {
	resetProfile(t)
	SetPQProfile(ProfileFor(ChainLux))

	_, err := (&ecrecover{}).Run(make([]byte, 128))
	if errors.Is(err, ErrEcrecoverForbidden) {
		t.Fatalf("lux chain must not refuse ecrecover; got %v", err)
	}
}
