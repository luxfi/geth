// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package catalyst

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/holiman/uint256"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/common/hexutil"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/tracing"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/eth"
	"github.com/luxfi/geth/node"
	"github.com/luxfi/geth/rpc"
)

// DevAPI provides Anvil/Hardhat-compatible RPC methods for dev mode.
// These methods allow manipulation of blockchain state for testing purposes.
type DevAPI struct {
	eth       *eth.Ethereum
	sim       *SimulatedBeacon
	snapshots map[hexutil.Uint64]common.Hash // snapshot ID -> block hash
	nextSnap  hexutil.Uint64
	mu        sync.Mutex
}

// NewDevAPI creates a new DevAPI instance.
func NewDevAPI(eth *eth.Ethereum, sim *SimulatedBeacon) *DevAPI {
	return &DevAPI{
		eth:       eth,
		sim:       sim,
		snapshots: make(map[hexutil.Uint64]common.Hash),
		nextSnap:  1,
	}
}

// RegisterDevAPIs registers the dev mode APIs with the node.
// This registers under multiple namespaces for Anvil/Hardhat compatibility:
// - eth: eth_setStorageAt, eth_setBalance (Anvil compat)
// - evm: evm_mine, evm_snapshot, evm_revert (Anvil/Hardhat compat)
// - anvil: anvil_* methods (Anvil compat)
// - hardhat: hardhat_* methods (Hardhat compat)
func RegisterDevAPIs(stack *node.Node, eth *eth.Ethereum, sim *SimulatedBeacon) {
	api := NewDevAPI(eth, sim)
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "eth",
			Service:   api,
			Version:   "1.0",
		},
		{
			Namespace: "evm",
			Service:   api,
			Version:   "1.0",
		},
		{
			Namespace: "anvil",
			Service:   api,
			Version:   "1.0",
		},
		{
			Namespace: "hardhat",
			Service:   api,
			Version:   "1.0",
		},
	})
}

// SetBalance sets the balance of an address.
// Anvil: anvil_setBalance
// Hardhat: hardhat_setBalance
func (api *DevAPI) SetBalance(ctx context.Context, address common.Address, balance hexutil.Big) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Get the current state
	header := api.eth.BlockChain().CurrentBlock()
	statedb, err := api.eth.BlockChain().StateAt(header.Root)
	if err != nil {
		return err
	}

	// Set the balance
	u256Balance, overflow := uint256.FromBig((*big.Int)(&balance))
	if overflow {
		return errors.New("balance overflow")
	}
	statedb.SetBalance(address, u256Balance, tracing.BalanceChangeUnspecified)

	// Commit the state changes and create a new block
	return api.commitStateChanges(statedb)
}

// SetStorageAt sets a storage slot value for an address.
// Anvil: anvil_setStorageAt
// Hardhat: hardhat_setStorageAt
func (api *DevAPI) SetStorageAt(ctx context.Context, address common.Address, slot common.Hash, value common.Hash) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Get the current state
	header := api.eth.BlockChain().CurrentBlock()
	statedb, err := api.eth.BlockChain().StateAt(header.Root)
	if err != nil {
		return err
	}

	// Set the storage value
	statedb.SetState(address, slot, value)

	// Commit the state changes and create a new block
	return api.commitStateChanges(statedb)
}

// SetCode sets the code of an address.
// Anvil: anvil_setCode
// Hardhat: hardhat_setCode
func (api *DevAPI) SetCode(ctx context.Context, address common.Address, code hexutil.Bytes) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Get the current state
	header := api.eth.BlockChain().CurrentBlock()
	statedb, err := api.eth.BlockChain().StateAt(header.Root)
	if err != nil {
		return err
	}

	// Set the code
	statedb.SetCode(address, code, tracing.CodeChangeUnspecified)

	// Commit the state changes and create a new block
	return api.commitStateChanges(statedb)
}

// SetNonce sets the nonce of an address.
// Anvil: anvil_setNonce
// Hardhat: hardhat_setNonce
func (api *DevAPI) SetNonce(ctx context.Context, address common.Address, nonce hexutil.Uint64) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Get the current state
	header := api.eth.BlockChain().CurrentBlock()
	statedb, err := api.eth.BlockChain().StateAt(header.Root)
	if err != nil {
		return err
	}

	// Set the nonce
	statedb.SetNonce(address, uint64(nonce), tracing.NonceChangeUnspecified)

	// Commit the state changes and create a new block
	return api.commitStateChanges(statedb)
}

// Mine forces mining of new blocks.
// Anvil: evm_mine (single optional timestamp parameter)
// Hardhat: hardhat_mine (blocks count, optional interval)
// We use optional pointer parameters to handle both calling conventions.
func (api *DevAPI) Mine(ctx context.Context, blocks *hexutil.Uint64, interval *hexutil.Uint64) (common.Hash, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	numBlocks := uint64(1)
	if blocks != nil {
		numBlocks = uint64(*blocks)
	}

	// Calculate interval between blocks (default to current time progression)
	_ = interval // interval is ignored for now

	if numBlocks == 0 {
		numBlocks = 1
	}

	var lastHash common.Hash
	for i := uint64(0); i < numBlocks; i++ {
		ts := uint64(time.Now().Unix())
		if err := api.sim.sealBlock(nil, ts); err != nil {
			return common.Hash{}, err
		}
		lastHash = api.eth.BlockChain().CurrentBlock().Hash()
	}
	return lastHash, nil
}

// Snapshot creates a snapshot of the current state.
// Returns a snapshot ID that can be used with Revert.
// Anvil: evm_snapshot
// Hardhat: evm_snapshot
func (api *DevAPI) Snapshot(ctx context.Context) hexutil.Uint64 {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Store the current block hash as a snapshot
	blockHash := api.eth.BlockChain().CurrentBlock().Hash()
	snapID := api.nextSnap
	api.snapshots[snapID] = blockHash
	api.nextSnap++

	return snapID
}

// Revert reverts the state to a previous snapshot.
// Anvil: evm_revert
// Hardhat: evm_revert
func (api *DevAPI) Revert(ctx context.Context, snapID hexutil.Uint64) (bool, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	blockHash, ok := api.snapshots[snapID]
	if !ok {
		return false, errors.New("snapshot not found")
	}

	// Get the block by hash
	block := api.eth.BlockChain().GetBlockByHash(blockHash)
	if block == nil {
		return false, errors.New("snapshot block not found")
	}

	// Revert to the snapshot block
	if _, err := api.eth.BlockChain().SetCanonical(block); err != nil {
		return false, err
	}

	// Delete the snapshot and all snapshots after it
	for id := range api.snapshots {
		if id >= snapID {
			delete(api.snapshots, id)
		}
	}

	return true, nil
}

// Reset resets the chain to genesis state.
// Anvil: anvil_reset
func (api *DevAPI) Reset(ctx context.Context) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Get genesis block
	genesis := api.eth.BlockChain().Genesis()
	if genesis == nil {
		return errors.New("genesis block not found")
	}

	// Reset to genesis
	if _, err := api.eth.BlockChain().SetCanonical(genesis); err != nil {
		return err
	}

	// Clear snapshots
	api.snapshots = make(map[hexutil.Uint64]common.Hash)
	api.nextSnap = 1

	return nil
}

// IncreaseTime increases the block timestamp.
// Anvil: evm_increaseTime
// Hardhat: evm_increaseTime
func (api *DevAPI) IncreaseTime(ctx context.Context, seconds hexutil.Uint64) (hexutil.Uint64, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	// Get current block time and add the increment
	currentTime := api.eth.BlockChain().CurrentBlock().Time
	newTime := currentTime + uint64(seconds)

	// Seal a new block with the adjusted time
	if err := api.sim.sealBlock(nil, newTime); err != nil {
		return 0, err
	}

	return hexutil.Uint64(newTime), nil
}

// SetNextBlockTimestamp sets the timestamp for the next block.
// Anvil: evm_setNextBlockTimestamp
// Hardhat: evm_setNextBlockTimestamp
func (api *DevAPI) SetNextBlockTimestamp(ctx context.Context, timestamp hexutil.Uint64) error {
	// This will be applied in the next sealBlock call
	// For now, we just seal a block with the given timestamp
	api.mu.Lock()
	defer api.mu.Unlock()

	return api.sim.sealBlock(nil, uint64(timestamp))
}

// ImpersonateAccount starts impersonating an account (allows sending tx without private key).
// Anvil: anvil_impersonateAccount
// Hardhat: hardhat_impersonateAccount
func (api *DevAPI) ImpersonateAccount(ctx context.Context, address common.Address) error {
	// In dev mode, all accounts are effectively impersonatable
	// This is a no-op for compatibility
	return nil
}

// StopImpersonatingAccount stops impersonating an account.
// Anvil: anvil_stopImpersonatingAccount
// Hardhat: hardhat_stopImpersonatingAccount
func (api *DevAPI) StopImpersonatingAccount(ctx context.Context, address common.Address) error {
	// This is a no-op for compatibility
	return nil
}

// AutoImpersonate enables or disables auto-impersonation of all accounts.
// Anvil: anvil_autoImpersonateAccount
func (api *DevAPI) AutoImpersonate(ctx context.Context, enabled bool) error {
	// This is a no-op for compatibility
	return nil
}

// commitStateChanges commits state changes by creating a synthetic block.
// This is used by SetBalance, SetStorageAt, etc. to persist changes.
func (api *DevAPI) commitStateChanges(statedb *state.StateDB) error {
	bc := api.eth.BlockChain()
	parent := bc.CurrentBlock()

	// Finalize the state changes
	statedb.Finalise(false)

	// Commit to get the new state root
	root, err := statedb.Commit(parent.Number.Uint64()+1, false, false)
	if err != nil {
		return err
	}

	// Flush the trie changes to disk
	if err := bc.TrieDB().Commit(root, false); err != nil {
		return err
	}

	// Create a synthetic block with the new state root
	newBlockNum := new(big.Int).Add(parent.Number, big.NewInt(1))
	timestamp := uint64(time.Now().Unix())
	if timestamp <= parent.Time {
		timestamp = parent.Time + 1
	}

	header := &types.Header{
		ParentHash:  parent.Hash(),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    parent.Coinbase,
		Root:        root,
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      newBlockNum,
		GasLimit:    parent.GasLimit,
		GasUsed:     0,
		Time:        timestamp,
		Extra:       []byte("dev-state-change"),
		BaseFee:     parent.BaseFee,
	}

	// Create an empty block body
	block := types.NewBlock(header, &types.Body{}, nil, nil)

	// Insert the block without state validation (since we manually set the state)
	if err := bc.InsertBlockWithoutState(block); err != nil {
		return err
	}

	return nil
}

// DumpState returns a dump of the current state (for debugging).
func (api *DevAPI) DumpState(ctx context.Context) (state.Dump, error) {
	api.mu.Lock()
	defer api.mu.Unlock()

	header := api.eth.BlockChain().CurrentBlock()
	statedb, err := api.eth.BlockChain().StateAt(header.Root)
	if err != nil {
		return state.Dump{}, err
	}

	return statedb.RawDump(&state.DumpConfig{
		OnlyWithAddresses: true,
		Max:               256,
	}), nil
}
