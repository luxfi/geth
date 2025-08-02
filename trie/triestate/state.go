// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package triestate provides types for trie state management
package triestate

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/crypto"
	"github.com/luxfi/geth/triedb"
)

// Set is an alias for triedb.StateSet
type Set = triedb.StateSet

// New creates a new state set
func New(accounts map[common.Address][]byte, storages map[common.Address]map[common.Hash][]byte, incomplete map[common.Address]struct{}) *Set {
	set := triedb.NewStateSet()
	// Convert to the expected format
	for addr, data := range accounts {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		set.Accounts[addrHash] = data
		set.AddressesToHashes[addr] = addrHash
		set.HashesToAddresses[addrHash] = addr
		if _, ok := incomplete[addr]; ok {
			// Mark as incomplete in origin (requires further handling)
			set.AccountsOrigin[addr] = data
		}
	}
	for addr, storage := range storages {
		addrHash := crypto.Keccak256Hash(addr.Bytes())
		set.Storages[addrHash] = storage
		// Make sure address mappings exist
		if _, exists := set.AddressesToHashes[addr]; !exists {
			set.AddressesToHashes[addr] = addrHash
			set.HashesToAddresses[addrHash] = addr
		}
	}
	// Copy incomplete map
	for addr := range incomplete {
		set.Incomplete[addr] = struct{}{}
	}
	return set
}

// TrieLoader is an interface for loading trie data
// TODO: Implement proper trie loading functionality
type TrieLoader interface {
	// Load loads trie data for the given root
	Load(root common.Hash) error
}

// Trie represents a trie structure
// TODO: Implement proper trie functionality
type Trie struct {
	root common.Hash
}

// Apply applies state changes to another state set
// TODO: Implement proper state application
func Apply(target *Set, source *Set) {
	if source == nil || target == nil {
		return
	}
	// Copy accounts
	for hash, data := range source.Accounts {
		target.Accounts[hash] = data
	}
	// Copy storages
	for addrHash, storage := range source.Storages {
		if _, exists := target.Storages[addrHash]; !exists {
			target.Storages[addrHash] = make(map[common.Hash][]byte)
		}
		for slotHash, data := range storage {
			target.Storages[addrHash][slotHash] = data
		}
	}
	// Copy mappings
	for addr, hash := range source.AddressesToHashes {
		target.AddressesToHashes[addr] = hash
	}
	for hash, addr := range source.HashesToAddresses {
		target.HashesToAddresses[hash] = addr
	}
	// Copy incomplete
	for addr := range source.Incomplete {
		target.Incomplete[addr] = struct{}{}
	}
}