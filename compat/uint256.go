// Package compat provides compatibility helpers for uint256 and hexutil conversions
package compat

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

// U256Hex converts *uint256.Int to the canonical 0x… hex string
func U256Hex(u *uint256.Int) string {
	if u == nil {
		return "0x0"
	}
	return hexutil.EncodeBig(u.ToBig())
}

// U256HexBig keeps old call-sites that passed *big.Int
func U256HexBig(b *big.Int) string {
	if b == nil {
		return "0x0"
	}
	return hexutil.EncodeBig(b)
}

// U256FromBig converts a big.Int to uint256.Int
func U256FromBig(b *big.Int) *uint256.Int {
	if b == nil {
		return uint256.NewInt(0)
	}
	u, _ := uint256.FromBig(b)
	return u
}