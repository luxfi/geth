// Package vm provides wrapper types for go-ethereum's vm errors
package vm

import (
	"github.com/ethereum/go-ethereum/core/vm"
)

// Re-export error variables
var (
	ErrOutOfGas = vm.ErrOutOfGas
	ErrCodeStoreOutOfGas = vm.ErrCodeStoreOutOfGas
	ErrDepth = vm.ErrDepth
	ErrInsufficientBalance = vm.ErrInsufficientBalance
	ErrContractAddressCollision = vm.ErrContractAddressCollision
	ErrExecutionReverted = vm.ErrExecutionReverted
	ErrMaxCodeSizeExceeded = vm.ErrMaxCodeSizeExceeded
	ErrInvalidJump = vm.ErrInvalidJump
	ErrWriteProtection = vm.ErrWriteProtection
	ErrReturnDataOutOfBounds = vm.ErrReturnDataOutOfBounds
	ErrGasUintOverflow = vm.ErrGasUintOverflow
	ErrInvalidCode = vm.ErrInvalidCode
	ErrNonceUintOverflow = vm.ErrNonceUintOverflow
