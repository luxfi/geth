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

package triedb

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/triedb/pathdb"
)

// StateSet represents a collection of mutated states during a state transition.
type StateSet struct {
	Accounts       map[common.Hash][]byte                    // Mutated accounts in 'slim RLP' encoding
	AccountsOrigin map[common.Address][]byte                 // Original values of mutated accounts in 'slim RLP' encoding
	Storages       map[common.Hash]map[common.Hash][]byte    // Mutated storage slots in 'prefix-zero-trimmed' RLP format
	StoragesOrigin map[common.Address]map[common.Hash][]byte // Original values of mutated storage slots in 'prefix-zero-trimmed' RLP format
	RawStorageKey  bool                                      // Flag whether the storage set uses the raw slot key or the hash
	
	// Additional mappings to track addresses for journal
	AddressesToHashes map[common.Address]common.Hash         // Maps addresses to their hashes
	HashesToAddresses map[common.Hash]common.Address         // Maps hashes back to addresses
	Incomplete        map[common.Address]struct{}            // Tracks incomplete accounts
}

// NewStateSet initializes an empty state set.
func NewStateSet() *StateSet {
	return &StateSet{
		Accounts:          make(map[common.Hash][]byte),
		AccountsOrigin:    make(map[common.Address][]byte),
		Storages:          make(map[common.Hash]map[common.Hash][]byte),
		StoragesOrigin:    make(map[common.Address]map[common.Hash][]byte),
		AddressesToHashes: make(map[common.Address]common.Hash),
		HashesToAddresses: make(map[common.Hash]common.Address),
		Incomplete:        make(map[common.Address]struct{}),
	}
}

// Size returns the approximate memory size of the state set.
func (set *StateSet) Size() int {
	if set == nil {
		return 0
	}
	size := 0
	for _, account := range set.Accounts {
		size += len(account) + 32 // 32 bytes for hash key
	}
	for _, accountOrig := range set.AccountsOrigin {
		size += len(accountOrig) + 20 // 20 bytes for address key
	}
	for _, storage := range set.Storages {
		size += 32 // outer hash key
		for _, slot := range storage {
			size += len(slot) + 32 // 32 bytes for inner hash key
		}
	}
	for _, storageOrig := range set.StoragesOrigin {
		size += 20 // address key
		for _, slot := range storageOrig {
			size += len(slot) + 32 // 32 bytes for hash key
		}
	}
	return size
}

// internal returns a state set for path database internal usage.
func (set *StateSet) internal() *pathdb.StateSetWithOrigin {
	// the nil state set is possible in tests.
	if set == nil {
		return nil
	}
	return pathdb.NewStateSetWithOrigin(set.Accounts, set.Storages, set.AccountsOrigin, set.StoragesOrigin, set.RawStorageKey)
}
