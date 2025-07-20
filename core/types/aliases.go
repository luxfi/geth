// Package types provides type aliases to ethereum types
// This ensures compatibility with ethereum interfaces while allowing our own extensions
package types

import (
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// Core types - direct aliases to ethereum types
type (
	// Transaction types
	Transaction  = ethtypes.Transaction
	Transactions = ethtypes.Transactions
	TxData       = ethtypes.TxData
	TxByNonce    = ethtypes.TxByNonce
	
	// Block types
	Block      = ethtypes.Block
	Header     = ethtypes.Header
	// Body is defined in block_ext.go as ExtendedBody
	BlockNonce = ethtypes.BlockNonce
	
	// Receipt types
	Receipt  = ethtypes.Receipt
	Receipts = ethtypes.Receipts
	Log      = ethtypes.Log
	Bloom    = ethtypes.Bloom
	
	// Transaction implementations
	LegacyTx     = ethtypes.LegacyTx
	AccessListTx = ethtypes.AccessListTx
	DynamicFeeTx = ethtypes.DynamicFeeTx
	BlobTx       = ethtypes.BlobTx
	
	// Other types
	AccessList    = ethtypes.AccessList
	AccessTuple   = ethtypes.AccessTuple
	BlobTxSidecar = ethtypes.BlobTxSidecar
	Withdrawal    = ethtypes.Withdrawal
	Withdrawals   = ethtypes.Withdrawals
	
	// Signer types
	Signer       = ethtypes.Signer
	EIP155Signer = ethtypes.EIP155Signer
)

// Constants
const (
	// Transaction types
	LegacyTxType     = ethtypes.LegacyTxType
	AccessListTxType = ethtypes.AccessListTxType
	DynamicFeeTxType = ethtypes.DynamicFeeTxType
	BlobTxType       = ethtypes.BlobTxType
	
	// Receipt status
	ReceiptStatusFailed     = ethtypes.ReceiptStatusFailed
	ReceiptStatusSuccessful = ethtypes.ReceiptStatusSuccessful
)

// Variables
var (
	// Hashes
	EmptyRootHash        = ethtypes.EmptyRootHash
	EmptyCodeHash        = ethtypes.EmptyCodeHash
	EmptyTxsHash         = ethtypes.EmptyTxsHash
	EmptyReceiptsHash    = ethtypes.EmptyReceiptsHash
	EmptyWithdrawalsHash = ethtypes.EmptyWithdrawalsHash
	
	// Errors
	ErrInvalidSig           = ethtypes.ErrInvalidSig
	ErrUnexpectedProtection = ethtypes.ErrUnexpectedProtection
	ErrInvalidTxType        = ethtypes.ErrInvalidTxType
	ErrTxTypeNotSupported   = ethtypes.ErrTxTypeNotSupported
	ErrGasFeeCapTooLow      = ethtypes.ErrGasFeeCapTooLow
	ErrInvalidChainId       = ethtypes.ErrInvalidChainId
)

// Functions
var (
	// Transaction creation
	NewTx               = ethtypes.NewTx
	NewTransaction      = ethtypes.NewTransaction
	NewContractCreation = ethtypes.NewContractCreation
	
	// Signing
	SignTx                 = ethtypes.SignTx
	SignNewTx              = ethtypes.SignNewTx
	MustSignNewTx          = ethtypes.MustSignNewTx
	Sender                 = ethtypes.Sender
	LatestSigner           = ethtypes.LatestSigner
	LatestSignerForChainID = ethtypes.LatestSignerForChainID
	NewEIP155Signer        = ethtypes.NewEIP155Signer
	NewLondonSigner        = ethtypes.NewLondonSigner
	NewCancunSigner        = ethtypes.NewCancunSigner
	MakeSigner             = ethtypes.MakeSigner
	
	// Block functions
	CalcUncleHash      = ethtypes.CalcUncleHash
	DeriveSha          = ethtypes.DeriveSha
	NewBlockWithHeader = ethtypes.NewBlockWithHeader
)

// Interfaces
type (
	DerivableList = ethtypes.DerivableList
)

// Additional aliases for missing types
type (
	HomesteadSigner    = ethtypes.HomesteadSigner
	FrontierSigner     = ethtypes.FrontierSigner
	StateAccount       = ethtypes.StateAccount
	ReceiptForStorage  = ethtypes.ReceiptForStorage
)

// Constants for bloom filters
const (
	BloomBitLength  = ethtypes.BloomBitLength
	BloomByteLength = ethtypes.BloomByteLength
)

// Functions for accounts
var (
	NewEmptyStateAccount = ethtypes.NewEmptyStateAccount
)

// TrieRootHash is the hash of a trie root
type TrieRootHash = common.Hash

// FullAccount is StateAccount in newer versions
type FullAccount = StateAccount