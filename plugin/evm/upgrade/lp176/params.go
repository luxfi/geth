// Copyright (C) 2019-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package lp176

import "math/big"

// LP-176: Dynamic EVM Gas Limit and Price Discovery
// Based on Avalanche ACP-176, adapted for Lux Network
//
// See: ~/work/lux/lps/LP-176-dynamic-gas-pricing.md for full specification
var (
	// MinBaseFee is the minimum base fee
	MinBaseFee = big.NewInt(25_000_000_000) // 25 gwei

	// MaxBaseFee is the maximum base fee
	MaxBaseFee = new(big.Int).Mul(big.NewInt(1000), big.NewInt(1_000_000_000)) // 1000 gwei

	// BaseFeeChangeDenominator controls the rate of base fee change
	BaseFeeChangeDenominator = uint64(8)

	// ElasticityMultiplier is the multiplier for target gas
	ElasticityMultiplier = uint64(2)
)
