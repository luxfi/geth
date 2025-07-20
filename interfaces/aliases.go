// Package interfaces provides type aliases for ethereum interfaces
package interfaces

import (
	eth "github.com/ethereum/go-ethereum"
)

// Type aliases for ethereum interfaces
type (
	ChainStateReader  = eth.ChainStateReader
	ChainReader       = eth.ChainReader
	TransactionReader = eth.TransactionReader
	CallMsg           = eth.CallMsg
	FilterQuery       = eth.FilterQuery
	GasEstimator      = eth.GasEstimator
	GasPricer         = eth.GasPricer
	GasPricer1559     = eth.GasPricer1559
	FeeHistory        = eth.FeeHistory
	PendingStateReader = eth.PendingStateReader
	PendingStateEventer = eth.PendingStateEventer
	PendingContractCaller = eth.PendingContractCaller
	LogFilterer       = eth.LogFilterer
	
	// Core types that might be referenced
	ContractCaller    = eth.ContractCaller
	TransactionSender = eth.TransactionSender
)