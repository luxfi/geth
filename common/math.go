// (c) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import "math/big"

// BigMax returns the larger of x or y.
func BigMax(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return x
	}
	return y
}

// BigMin returns the smaller of x or y.
func BigMin(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return x
	}
	return y
}