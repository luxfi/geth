// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package stateconf

// SnapshotUpdateOption represents an option for snapshot updates
type SnapshotUpdateOption interface {
	apply(*snapshotUpdateConfig)
}

// StateDBStateOption represents an option for statedb state
type StateDBStateOption interface {
	apply(*stateDBConfig)
}

// TrieDBUpdateOption represents an option for triedb updates
type TrieDBUpdateOption interface {
	apply(*trieDBUpdateConfig)
}

type snapshotUpdateConfig struct {
	payload interface{}
}

type stateDBConfig struct {
	// Add fields as needed
}

type trieDBUpdateConfig struct {
	// Add fields as needed
}

type snapshotUpdateOptionFunc func(*snapshotUpdateConfig)

func (f snapshotUpdateOptionFunc) apply(c *snapshotUpdateConfig) {
	f(c)
}

// WithSnapshotUpdatePayload creates an option with a payload
func WithSnapshotUpdatePayload(payload interface{}) SnapshotUpdateOption {
	return snapshotUpdateOptionFunc(func(c *snapshotUpdateConfig) {
		c.payload = payload
	})
}

// ExtractSnapshotUpdatePayload extracts the payload from a snapshot update option
func ExtractSnapshotUpdatePayload(opt SnapshotUpdateOption) interface{} {
	c := &snapshotUpdateConfig{}
	opt.apply(c)
	return c.payload
}