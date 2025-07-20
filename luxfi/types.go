// Copyright (C) 2019-2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package geth

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/geth/accounts/abi"
	"github.com/luxfi/geth/accounts/abi/bind"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/crypto"
)

// Re-export ABI types
type (
	ABI    = abi.ABI
	Type   = abi.Type
	Method = abi.Method
	Event  = abi.Event
)

// Re-export bind types
type (
	TransactOpts  = bind.TransactOpts
	CallOpts      = bind.CallOpts
	BoundContract = bind.BoundContract
	WatchOpts     = bind.WatchOpts
	FilterOpts    = bind.FilterOpts
	MetaData      = bind.MetaData
)

// Re-export transaction types
type (
	Transaction  = types.Transaction
	Receipt      = types.Receipt
	Log          = types.Log
	Block        = types.Block
	Header       = types.Header
	AccessList   = types.AccessList
	AccessTuple  = types.AccessTuple
	TxData       = types.TxData
	LegacyTx     = types.LegacyTx
	AccessListTx = types.AccessListTx
	DynamicFeeTx = types.DynamicFeeTx
)

// Re-export signer types
type (
	Signer          = types.Signer
	EIP155Signer    = types.EIP155Signer
	HomesteadSigner = types.HomesteadSigner
	FrontierSigner  = types.FrontierSigner
)

// Re-export crypto types
type (
	PrivateKey = ecdsa.PrivateKey
	PublicKey  = ecdsa.PublicKey
)

// Re-export bind functions
var (
	NewTransactor                    = bind.NewTransactor
	NewTransactorWithChainID         = bind.NewTransactorWithChainID
	NewKeyedTransactor               = bind.NewKeyedTransactor
	NewKeyedTransactorWithChainID    = bind.NewKeyedTransactorWithChainID
	NewClefTransactor                = bind.NewClefTransactor
	NewKeyStoreTransactor            = bind.NewKeyStoreTransactor
	NewKeyStoreTransactorWithChainID = bind.NewKeyStoreTransactorWithChainID
	WaitMined                        = bind.WaitMined
	WaitDeployed                     = bind.WaitDeployed
	DeployContract                   = bind.DeployContract
)

// Re-export transaction functions
var (
	NewTransaction         = types.NewTransaction
	NewContractCreation    = types.NewContractCreation
	SignTx                 = types.SignTx
	SignNewTx              = types.SignNewTx
	MustSignNewTx          = types.MustSignNewTx
	LatestSigner           = types.LatestSigner
	LatestSignerForChainID = types.LatestSignerForChainID
	NewEIP155Signer        = types.NewEIP155Signer
	NewLondonSigner        = types.NewLondonSigner
)

// Re-export common types
type (
	Hash    = common.Hash
	Address = common.Address
)

// Re-export common variables
var (
	Big0   = common.Big0
	Big1   = common.Big1
	Big32  = common.Big32
	Big256 = common.Big256
	Big257 = common.Big257
)

// Re-export common functions
var (
	BytesToHash    = common.BytesToHash
	BigToHash      = common.BigToHash
	HexToHash      = common.HexToHash
	BytesToAddress = common.BytesToAddress
	BigToAddress   = common.BigToAddress
	HexToAddress   = common.HexToAddress
	IsHexAddress   = common.IsHexAddress
	Hex2Bytes      = common.Hex2Bytes
	FromHex        = common.FromHex
	CopyBytes      = common.CopyBytes
	LeftPadBytes   = common.LeftPadBytes
	RightPadBytes  = common.RightPadBytes
	BigMax         = common.BigMax
	BigMin         = common.BigMin
)

// Re-export crypto functions
var (
	GenerateKey     = crypto.GenerateKey
	FromECDSA       = crypto.FromECDSA
	FromECDSAPub    = crypto.FromECDSAPub
	ToECDSA         = crypto.ToECDSA
	ToECDSAUnsafe   = crypto.ToECDSAUnsafe
	HexToECDSA      = crypto.HexToECDSA
	LoadECDSA       = crypto.LoadECDSA
	SaveECDSA       = crypto.SaveECDSA
	SigToPub        = crypto.SigToPub
	Sign            = crypto.Sign
	VerifySignature = crypto.VerifySignature
	Keccak256       = crypto.Keccak256
	Keccak256Hash   = crypto.Keccak256Hash
	CreateAddress   = crypto.CreateAddress
	CreateAddress2  = crypto.CreateAddress2
	PubkeyToAddress = crypto.PubkeyToAddress
)
