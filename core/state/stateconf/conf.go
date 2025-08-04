// Copyright 2020-2025 The go-ethereum Authors
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

// Package stateconf configures state management.
package stateconf

import (
	"github.com/luxfi/geth/common"
)

// SnapshotUpdateOption is a placeholder for snapshot update options
// This is implemented as an empty interface for now, but can be expanded
// to carry payloads as needed.
type SnapshotUpdateOption interface{}

// TrieDBUpdateOption is a placeholder for trie database update options
// This is implemented as an empty interface for now, but can be expanded
// to carry payloads as needed.
type TrieDBUpdateOption interface{}

// snapshotUpdatePayload represents a snapshot update with payload
type snapshotUpdatePayload struct {
	payload interface{}
}

// WithSnapshotUpdatePayload returns a SnapshotUpdateOption carrying an arbitrary payload
func WithSnapshotUpdatePayload(p interface{}) SnapshotUpdateOption {
	return &snapshotUpdatePayload{payload: p}
}

// ExtractSnapshotUpdatePayload extracts the payload from snapshot update options
func ExtractSnapshotUpdatePayload(opts ...SnapshotUpdateOption) interface{} {
	for _, opt := range opts {
		if p, ok := opt.(*snapshotUpdatePayload); ok {
			return p.payload
		}
	}
	return nil
}

// trieDBUpdatePayload represents a trie DB update with block hashes
type trieDBUpdatePayload struct {
	parentBlockHash  common.Hash
	currentBlockHash common.Hash
}

// WithTrieDBUpdatePayload returns a TrieDBUpdateOption carrying two block hashes
func WithTrieDBUpdatePayload(parent common.Hash, current common.Hash) TrieDBUpdateOption {
	return &trieDBUpdatePayload{
		parentBlockHash:  parent,
		currentBlockHash: current,
	}
}

// ExtractTrieDBUpdatePayload extracts the payload from trie DB update options
func ExtractTrieDBUpdatePayload(opts ...TrieDBUpdateOption) (common.Hash, common.Hash, bool) {
	for _, opt := range opts {
		if p, ok := opt.(*trieDBUpdatePayload); ok {
			return p.parentBlockHash, p.currentBlockHash, true
		}
	}
	return common.Hash{}, common.Hash{}, false
}