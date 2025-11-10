// Copyright (C) 2019-2025, Lux Partners Limited All rights reserved.
// See the file LICENSE for licensing terms.

package atomic

import (
	"errors"

	"github.com/luxfi/ids"
	"github.com/luxfi/geth/common"
)

var (
	ErrNotImplemented = errors.New("atomic operations not implemented")
)

// Tx represents an atomic transaction
type Tx struct {
	ID         ids.ID
	SourceChain ids.ID
	Inputs     []common.Address
	Outputs    []common.Address
}

// SharedMemory provides access to shared memory for atomic operations
type SharedMemory interface {
	GetAtomicTxs(chainID ids.ID) ([]*Tx, error)
}
