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
}

// NewStateSet initializes an empty state set.
func NewStateSet() *StateSet {
	return &StateSet{
		Accounts:       make(map[common.Hash][]byte),
		AccountsOrigin: make(map[common.Address][]byte),
		Storages:       make(map[common.Hash]map[common.Hash][]byte),
		StoragesOrigin: make(map[common.Address]map[common.Hash][]byte),
	}
}

// internal returns a state set for path database internal usage.
func (set *StateSet) internal() *pathdb.StateSetWithOrigin {
	// the nil state set is possible in tests.
	if set == nil {
		return nil
	}
	return pathdb.NewStateSetWithOrigin(set.Accounts, set.Storages, set.AccountsOrigin, set.StoragesOrigin, set.RawStorageKey)
}

// Size returns the approximate memory size of the state set.
func (set *StateSet) Size() int {
	if set == nil {
		return 0
	}
	size := 0
	for _, account := range set.Accounts {
		size += len(account)
	}
	for _, accountOrigin := range set.AccountsOrigin {
		size += len(accountOrigin)
	}
	for _, storage := range set.Storages {
		for _, data := range storage {
			size += len(data)
		}
	}
	for _, storageOrigin := range set.StoragesOrigin {
		for _, data := range storageOrigin {
			size += len(data)
		}
	}
	return size
}

// Incomplete returns a flag whether the state set contains incomplete data.
func (set *StateSet) Incomplete() bool {
	// For now, always return false as we don't track incomplete states
	return false
}
