// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package bn256 implements the Optimal Ate pairing over a 256-bit Barreto-Naehrig curve.
package bn256

import (
	bn256cf "github.com/luxfi/geth/crypto/bn256/cloudflare"
)

// Re-export the cloudflare implementation as the default
type G1 = bn256cf.G1
type G2 = bn256cf.G2
type GT = bn256cf.GT

// PairingCheck calculates the Optimal Ate pairing for a set of points.
func PairingCheck(a []*G1, b []*G2) bool {
	return bn256cf.PairingCheck(a, b)
}