// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package stateconf

// Config represents state configuration
type Config struct {
	// Pruning enables state pruning
	Pruning bool

	// SnapshotCache is the cache size for snapshots
	SnapshotCache int

	// OfflinePruning enables offline pruning
	OfflinePruning bool

	// StateSyncEnabled enables state sync
	StateSyncEnabled bool
}

// DefaultConfig returns the default state configuration
func DefaultConfig() *Config {
	return &Config{
		Pruning:          false,
		SnapshotCache:    256,
		OfflinePruning:   false,
		StateSyncEnabled: false,
	}
}

// SnapshotUpdateOption is a placeholder for snapshot update options.
// This is implemented as an empty interface for now, but can be expanded
// to carry payloads as needed.
type SnapshotUpdateOption interface{}

// TrieDBUpdateOption is a placeholder for trie database update options.
// This is implemented as an empty interface for now, but can be expanded
// to carry payloads as needed.
type TrieDBUpdateOption interface{}

// StateDBStateOption is a placeholder for state DB state options.
// This is used to pass options to state read/write operations.
type StateDBStateOption interface{}

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
	parentBlockHash  interface{} // Using interface{} to avoid importing common.Hash
	currentBlockHash interface{}
}

// WithTrieDBUpdatePayload returns a TrieDBUpdateOption carrying two block hashes
func WithTrieDBUpdatePayload(parent interface{}, current interface{}) TrieDBUpdateOption {
	return &trieDBUpdatePayload{
		parentBlockHash:  parent,
		currentBlockHash: current,
	}
}

// ExtractTrieDBUpdatePayload extracts the payload from trie DB update options
func ExtractTrieDBUpdatePayload(opts ...TrieDBUpdateOption) (interface{}, interface{}, bool) {
	for _, opt := range opts {
		if p, ok := opt.(*trieDBUpdatePayload); ok {
			return p.parentBlockHash, p.currentBlockHash, true
		}
	}
	return nil, nil, false
}

// skipStateKeyTransformation is a marker option to skip state key transformation
type skipStateKeyTransformation struct{}

// SkipStateKeyTransformation returns a StateDBStateOption that signals
// to skip state key transformation during state operations.
func SkipStateKeyTransformation() StateDBStateOption {
	return &skipStateKeyTransformation{}
}

// ShouldSkipStateKeyTransformation checks if any of the provided options
// indicates that state key transformation should be skipped.
func ShouldSkipStateKeyTransformation(opts ...StateDBStateOption) bool {
	for _, opt := range opts {
		if _, ok := opt.(*skipStateKeyTransformation); ok {
			return true
		}
	}
	return false
}
