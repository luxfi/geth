// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// Migration helpers — exported wrappers around the internal
// freezerTableConfig + NewFreezer / NewZapAncientStore plumbing so the
// `cmd/migrate-ancient` tool can drive both stores without re-importing
// the unexported config struct.
//
// We deliberately keep these wrappers minimal: only the table
// configuration and the open-for-migration entry points are exposed.
// The full ResettableAncientStore surface stays on the existing
// `ethdb.AncientStore` interface.
package rawdb

import (
	"github.com/luxfi/geth/ethdb"
)

// FreezerTableConfig is the exported mirror of the unexported
// freezerTableConfig. Only the migration tool consumes this — the
// production path keeps the unexported struct.
type FreezerTableConfig struct {
	NoSnappy bool
	Prunable bool
}

// internalTables converts the exported config into the unexported
// shape the freezer + ZAP store consume.
func internalTables(in map[string]FreezerTableConfig) map[string]freezerTableConfig {
	out := make(map[string]freezerTableConfig, len(in))
	for k, v := range in {
		out[k] = freezerTableConfig{noSnappy: v.NoSnappy, prunable: v.Prunable}
	}
	return out
}

// OpenFreezerForMigration opens the upstream-format ancient store in
// read-only mode. Used by `cmd/migrate-ancient` as the source of a
// migration.
func OpenFreezerForMigration(datadir string, tables map[string]FreezerTableConfig) (ethdb.AncientStore, error) {
	return NewFreezer(datadir, "migrate", true, freezerTableSize, internalTables(tables))
}

// NewZapAncientStoreForMigration opens a fresh ZAP-native ancient
// store at the given datadir. The caller supplies the underlying KV
// store — typically `ethdb/zapdb.New(datadir, ...)` in production;
// `memorydb.New()` in tests.
func NewZapAncientStoreForMigration(datadir string, tables map[string]FreezerTableConfig, store zapAncientBackend) (*ZapAncientStore, error) {
	return NewZapAncientStore(datadir, internalTables(tables), store)
}
