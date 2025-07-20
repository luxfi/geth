// Package tracing provides wrapper types for go-ethereum's tracing implementation
package tracing

import (
	"github.com/ethereum/go-ethereum/core/tracing"
)

// Re-export types
type (
	Hooks = tracing.Hooks
	OpContext = tracing.OpContext
	StateDB = tracing.StateDB
	BalanceChangeReason = tracing.BalanceChangeReason
)

// Re-export constants for balance change reasons
const (
	BalanceChangeUnspecified = tracing.BalanceChangeUnspecified
	BalanceIncreaseRewardMineUncle = tracing.BalanceIncreaseRewardMineUncle
	BalanceIncreaseRewardMineBlock = tracing.BalanceIncreaseRewardMineBlock
	BalanceIncreaseWithdrawal = tracing.BalanceIncreaseWithdrawal
	BalanceIncreaseGenesisBalance = tracing.BalanceIncreaseGenesisBalance
	BalanceIncreaseRewardTransactionFee = tracing.BalanceIncreaseRewardTransactionFee
	BalanceDecreaseGasBuy = tracing.BalanceDecreaseGasBuy
	BalanceDecreaseSelfdestructBurn = tracing.BalanceDecreaseSelfdestructBurn
