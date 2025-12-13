// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"context"

	"github.com/luxfi/consensus/engine/chain/block"
	"github.com/luxfi/ids"
)

// VM implements the Snowman ChainVM interface for the C-Chain
type VM struct {
	IsPlugin bool
}

// Ensure VM implements ChainVM
var _ block.ChainVM = (*VM)(nil)

// Initialize implements the snowman.ChainVM interface
func (vm *VM) Initialize(
	ctx context.Context,
	chainCtx interface{},
	db interface{},
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	msgChan interface{},
	fxs []interface{},
	appSender interface{},
) error {
	// TODO: Initialize the unified EVM
	return nil
}

// SetState sets the VM state
func (vm *VM) SetState(ctx context.Context, state uint32) error {
	return nil
}

// Shutdown shuts down the VM
func (vm *VM) Shutdown(ctx context.Context) error {
	return nil
}

// Version returns the VM version
func (vm *VM) Version(ctx context.Context) (string, error) {
	return "1.0.0", nil
}

// NewHTTPHandler returns the HTTP handlers for the VM
func (vm *VM) NewHTTPHandler(ctx context.Context) (interface{}, error) {
	return nil, nil
}

// HealthCheck returns health status of the VM
func (vm *VM) HealthCheck(ctx context.Context) (interface{}, error) {
	return nil, nil
}

// Connected is called when a new node is connected
func (vm *VM) Connected(ctx context.Context, nodeID ids.NodeID, nodeVersion interface{}) error {
	return nil
}

// Disconnected is called when a node is disconnected
func (vm *VM) Disconnected(ctx context.Context, nodeID ids.NodeID) error {
	return nil
}

// GetBlock returns a block by its ID
func (vm *VM) GetBlock(ctx context.Context, blkID ids.ID) (block.Block, error) {
	return nil, nil
}

// ParseBlock parses a block from bytes
func (vm *VM) ParseBlock(ctx context.Context, b []byte) (block.Block, error) {
	return nil, nil
}

// BuildBlock builds a new block
func (vm *VM) BuildBlock(ctx context.Context) (block.Block, error) {
	return nil, nil
}

// SetPreference sets the preferred block
func (vm *VM) SetPreference(ctx context.Context, blkID ids.ID) error {
	return nil
}

// LastAccepted returns the ID of the last accepted block
func (vm *VM) LastAccepted(ctx context.Context) (ids.ID, error) {
	return ids.Empty, nil
}

// GetBlockIDAtHeight returns the block ID at a specific height
func (vm *VM) GetBlockIDAtHeight(ctx context.Context, height uint64) (ids.ID, error) {
	return ids.Empty, nil
}

// WaitForEvent blocks until an event occurs that should trigger block building
func (vm *VM) WaitForEvent(ctx context.Context) (interface{}, error) {
	return nil, nil
}
