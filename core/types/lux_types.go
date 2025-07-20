package types

import (
	"math/big"
	
	"github.com/luxfi/geth/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// Body represents a block body with Lux extensions
type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
	Version      uint32
	ExtData      []byte
}

// NewBlockWithExtData creates a new block with extended data
// This is a Lux-specific function that maintains compatibility
func NewBlockWithExtData(header *Header, txs []*Transaction, uncles []*Header, 
	receipts []*Receipt, version uint32, extData []byte, commit bool) *Block {
	// For now, just create a standard block
	// The extended data would be handled separately in the actual implementation
	// Create ethereum body and use standard method
	ethBody := ethtypes.Body{
		Transactions: txs,
		Uncles:       uncles,
	}
	return NewBlockWithHeader(header).WithBody(ethBody)
}

// BlockWithExtData adds extended data to a block
// This is a helper function since we can't add methods to the aliased Block type
func BlockWithExtData(b *Block, version uint32, extData []byte) *Block {
	// In the actual implementation, this would create a new block with extended data
	// For now, just return the block unchanged
	return b
}

// ExtendedStateAccount is a Lux-specific extension of StateAccount
type ExtendedStateAccount struct {
	Nonce       uint64
	Balance     *big.Int
	Root        common.Hash
	CodeHash    []byte
	IsMultiCoin bool
}

// StateAccount is our extended state account type
type StateAccount = ExtendedStateAccount

// ExtendedSlimAccount is a Lux-specific extension of SlimAccount  
type ExtendedSlimAccount struct {
	Nonce       uint64
	Balance     *big.Int
	Root        []byte
	CodeHash    []byte
	IsMultiCoin bool
}

// SlimAccount is our extended slim account type
type SlimAccount = ExtendedSlimAccount

// SlimAccountRLP converts an account to its RLP representation for snapshot storage
func SlimAccountRLP(acc StateAccount) []byte {
	// Convert our extended account to ethereum StateAccount
	// Need to convert big.Int to uint256
	var balance uint256.Int
	if acc.Balance != nil {
		balance.SetFromBig(acc.Balance)
	}
	
	ethAcc := ethtypes.StateAccount{
		Nonce:    acc.Nonce,
		Balance:  &balance,
		Root:     acc.Root,
		CodeHash: acc.CodeHash,
	}
	return ethtypes.SlimAccountRLP(ethAcc)
}