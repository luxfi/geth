// Package interfaces provides type definitions for Lux-specific interfaces
package interfaces

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
)

// AcceptedContractCaller defines the methods needed to perform contract calls on accepted state
type AcceptedContractCaller interface {
	// AcceptedCodeAt returns the code of the given account in the accepted state.
	AcceptedCodeAt(ctx context.Context, contract common.Address) ([]byte, error)

	// AcceptedCallContract executes an Ethereum contract call against the accepted state.
	AcceptedCallContract(ctx context.Context, call CallMsg) ([]byte, error)
}