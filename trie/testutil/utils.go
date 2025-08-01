// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package testutil provides testing utilities for trie operations
package testutil

import (
	"math/rand"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/trie"
	"github.com/luxfi/geth/triedb"
)

// RandomHash generates a random hash for testing
func RandomHash() common.Hash {
	var hash common.Hash
	rand.Read(hash[:])
	return hash
}

// RandomAddress generates a random address for testing
func RandomAddress() common.Address {
	var addr common.Address
	rand.Read(addr[:])
	return addr
}

// NewTestTrie creates a new trie for testing
func NewTestTrie(t *testing.T) (*trie.Trie, triedb.Database) {
	db := rawdb.NewMemoryDatabase()
	triedb := triedb.NewDatabase(db, &triedb.Config{})
	tr, err := trie.New(trie.TrieID(common.Hash{}), triedb)
	if err != nil {
		t.Fatalf("Failed to create trie: %v", err)
	}
	return tr, triedb
}

// RandomData generates random data for testing
func RandomData(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

// GenerateTestKeys generates a set of test keys
func GenerateTestKeys(count int) [][]byte {
	keys := make([][]byte, count)
	for i := 0; i < count; i++ {
		keys[i] = RandomData(32)
	}
	return keys
}

// GenerateTestValues generates a set of test values
func GenerateTestValues(count int) [][]byte {
	values := make([][]byte, count)
	for i := 0; i < count; i++ {
		values[i] = RandomData(rand.Intn(100) + 1)
	}
	return values
}

// InsertTestData inserts test data into a trie
func InsertTestData(t *testing.T, tr *trie.Trie, keys, values [][]byte) {
	if len(keys) != len(values) {
		t.Fatalf("Keys and values length mismatch: %d vs %d", len(keys), len(values))
	}
	for i := range keys {
		if err := tr.Update(keys[i], values[i]); err != nil {
			t.Fatalf("Failed to insert key %x: %v", keys[i], err)
		}
	}
}

// VerifyTestData verifies test data in a trie
func VerifyTestData(t *testing.T, tr *trie.Trie, keys, values [][]byte) {
	if len(keys) != len(values) {
		t.Fatalf("Keys and values length mismatch: %d vs %d", len(keys), len(values))
	}
	for i := range keys {
		val, err := tr.Get(keys[i])
		if err != nil {
			t.Fatalf("Failed to get key %x: %v", keys[i], err)
		}
		if string(val) != string(values[i]) {
			t.Fatalf("Value mismatch for key %x: got %x, want %x", keys[i], val, values[i])
		}
	}
}