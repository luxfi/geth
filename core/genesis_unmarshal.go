// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package core

import (
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/common/hexutil"
	"github.com/luxfi/geth/common/math"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/params"
)

// UnmarshalJSON implements json.Unmarshaler to handle SubnetEVM genesis compatibility
func (g *Genesis) UnmarshalJSON(input []byte) error {
	// Define intermediate structure with flexible coinbase field
	type genesisJSON struct {
		Config        *params.ChainConfig                        `json:"config"`
		Nonce         *math.HexOrDecimal64                       `json:"nonce"`
		Timestamp     *math.HexOrDecimal64                       `json:"timestamp"`
		ExtraData     *hexutil.Bytes                             `json:"extraData"`
		GasLimit      *math.HexOrDecimal64                       `json:"gasLimit"   gencodec:"required"`
		Difficulty    *math.HexOrDecimal256                      `json:"difficulty" gencodec:"required"`
		Mixhash       *common.Hash                               `json:"mixHash"`
		Coinbase      json.RawMessage                            `json:"coinbase"` // Can be address or hash
		Alloc         map[common.UnprefixedAddress]types.Account `json:"alloc"      gencodec:"required"`
		AirdropHash   *common.Hash                               `json:"airdropHash,omitempty"`
		AirdropAmount *math.HexOrDecimal256                      `json:"airdropAmount,omitempty"`
		Number        *math.HexOrDecimal64                       `json:"number"`
		GasUsed       *math.HexOrDecimal64                       `json:"gasUsed"`
		ParentHash    *common.Hash                               `json:"parentHash"`
		BaseFee       *math.HexOrDecimal256                      `json:"baseFeePerGas"`
		ExcessBlobGas *math.HexOrDecimal64                       `json:"excessBlobGas"`
		BlobGasUsed   *math.HexOrDecimal64                       `json:"blobGasUsed"`
	}

	var dec genesisJSON
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	// Handle each field
	if dec.Config != nil {
		g.Config = dec.Config
	}
	if dec.Nonce != nil {
		g.Nonce = uint64(*dec.Nonce)
	}
	if dec.Timestamp != nil {
		g.Timestamp = uint64(*dec.Timestamp)
	}
	if dec.ExtraData != nil {
		g.ExtraData = *dec.ExtraData
	}
	if dec.GasLimit == nil {
		return errors.New("missing required field 'gasLimit' for Genesis")
	}
	g.GasLimit = uint64(*dec.GasLimit)
	if dec.Difficulty == nil {
		return errors.New("missing required field 'difficulty' for Genesis")
	}
	g.Difficulty = (*big.Int)(dec.Difficulty)
	if dec.Mixhash != nil {
		g.Mixhash = *dec.Mixhash
	}

	// Handle coinbase - can be either address (40 chars) or hash (64 chars)
	if dec.Coinbase != nil && len(dec.Coinbase) > 0 {
		var coinbaseStr string
		if err := json.Unmarshal(dec.Coinbase, &coinbaseStr); err == nil {
			// Remove 0x prefix if present
			coinbaseStr = strings.TrimPrefix(strings.TrimPrefix(coinbaseStr, "0x"), "0X")

			// If it's a 64-character string (hash), use zero address
			if len(coinbaseStr) == 64 {
				// SubnetEVM uses hash in coinbase, we convert to zero address
				g.Coinbase = common.Address{}
			} else if len(coinbaseStr) == 40 {
				// Normal 20-byte address
				g.Coinbase = common.HexToAddress(coinbaseStr)
			} else {
				// Try to parse as-is
				g.Coinbase = common.HexToAddress(coinbaseStr)
			}
		} else {
			// Try to unmarshal as address directly
			var addr common.Address
			if err := json.Unmarshal(dec.Coinbase, &addr); err == nil {
				g.Coinbase = addr
			} else {
				// Default to zero address if parsing fails
				g.Coinbase = common.Address{}
			}
		}
	}

	// Handle allocations
	if dec.Alloc == nil {
		return errors.New("missing required field 'alloc' for Genesis")
	}
	g.Alloc = make(types.GenesisAlloc, len(dec.Alloc))
	for k, v := range dec.Alloc {
		g.Alloc[common.Address(k)] = v
	}

	// Handle optional SubnetEVM fields
	if dec.AirdropHash != nil {
		g.AirdropHash = *dec.AirdropHash
	}
	if dec.AirdropAmount != nil {
		g.AirdropAmount = (*big.Int)(dec.AirdropAmount)
	}
	if dec.Number != nil {
		g.Number = uint64(*dec.Number)
	}
	if dec.GasUsed != nil {
		g.GasUsed = uint64(*dec.GasUsed)
	}
	if dec.ParentHash != nil {
		g.ParentHash = *dec.ParentHash
	}
	if dec.BaseFee != nil {
		g.BaseFee = (*big.Int)(dec.BaseFee)
	}
	if dec.ExcessBlobGas != nil {
		g.ExcessBlobGas = (*uint64)(dec.ExcessBlobGas)
	}
	if dec.BlobGasUsed != nil {
		g.BlobGasUsed = (*uint64)(dec.BlobGasUsed)
	}

	return nil
}