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