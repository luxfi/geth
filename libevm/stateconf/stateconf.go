// Copyright (C) 2020-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package stateconf provides configuration types for state database operations.
// This is a Lux extension for configurable state operations.
package stateconf

import "github.com/luxfi/geth/common"

// StateDBStateOption is an option for state database operations
type StateDBStateOption func(*StateDBStateConfig)

// StateDBStateConfig holds configuration for state DB operations
type StateDBStateConfig struct {
	SkipKeyTransformation bool
}

// SkipStateKeyTransformation returns an option to skip state key transformation
func SkipStateKeyTransformation() StateDBStateOption {
	return func(c *StateDBStateConfig) {
		c.SkipKeyTransformation = true
	}
}

// ApplyStateDBOptions applies the given options to a config
func ApplyStateDBOptions(opts ...StateDBStateOption) *StateDBStateConfig {
	config := &StateDBStateConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// TrieDBUpdateOption is an option for trie database updates
type TrieDBUpdateOption func(*TrieDBUpdateConfig)

// TrieDBUpdateConfig holds configuration for trie DB updates
type TrieDBUpdateConfig struct {
	ParentHash common.Hash
	BlockHash  common.Hash
}

// WithTrieDBUpdatePayload returns an option with parent and block hashes
func WithTrieDBUpdatePayload(parentHash, blockHash common.Hash) TrieDBUpdateOption {
	return func(c *TrieDBUpdateConfig) {
		c.ParentHash = parentHash
		c.BlockHash = blockHash
	}
}

// ExtractTrieDBUpdatePayload extracts the payload from options
func ExtractTrieDBUpdatePayload(opts ...TrieDBUpdateOption) (common.Hash, common.Hash) {
	config := &TrieDBUpdateConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config.ParentHash, config.BlockHash
}

// SnapshotUpdateOption is an option for snapshot updates
type SnapshotUpdateOption func(*SnapshotUpdateConfig)

// SnapshotUpdateConfig holds configuration for snapshot updates
type SnapshotUpdateConfig struct {
	Payload interface{}
}

// WithSnapshotUpdatePayload returns an option with the given payload
func WithSnapshotUpdatePayload(payload interface{}) SnapshotUpdateOption {
	return func(c *SnapshotUpdateConfig) {
		c.Payload = payload
	}
}

// ExtractSnapshotUpdatePayload extracts the payload from an option
func ExtractSnapshotUpdatePayload(opt SnapshotUpdateOption) interface{} {
	config := &SnapshotUpdateConfig{}
	opt(config)
	return config.Payload
}

// CommitOption is an option for state commit operations
type CommitOption func(*CommitConfig)

// CommitConfig holds configuration for commit operations
type CommitConfig struct {
	TrieDBUpdateOpts   []TrieDBUpdateOption
	SnapshotUpdateOpts []SnapshotUpdateOption
}

// WithTrieDBUpdateOpts returns a commit option with trie DB update options
func WithTrieDBUpdateOpts(opts ...TrieDBUpdateOption) CommitOption {
	return func(c *CommitConfig) {
		c.TrieDBUpdateOpts = append(c.TrieDBUpdateOpts, opts...)
	}
}

// WithSnapshotUpdateOpts returns a commit option with snapshot update options
func WithSnapshotUpdateOpts(opts ...SnapshotUpdateOption) CommitOption {
	return func(c *CommitConfig) {
		c.SnapshotUpdateOpts = append(c.SnapshotUpdateOpts, opts...)
	}
}

// ApplyCommitOptions applies commit options and returns config
func ApplyCommitOptions(opts ...CommitOption) *CommitConfig {
	config := &CommitConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}
