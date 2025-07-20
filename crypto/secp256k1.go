// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.

package crypto

import (
	"crypto/elliptic"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return secp256k1.S256()
