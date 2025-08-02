package dualdb

import (
	"time"
)

// Config defines the dual database configuration
type Config struct {
	// Enable dual database mode
	Enabled bool `json:"enabled"`

	// Finality delay - blocks older than this are considered finalized
	FinalityDelay uint64 `json:"finality_delay"`

	// Archive database config (read-only, finalized blocks)
	Archive ArchiveConfig `json:"archive"`

	// Current database config (read-write, recent blocks)
	Current CurrentConfig `json:"current"`

	// Sync configuration
	Sync SyncConfig `json:"sync"`

	// Maintenance configuration
	Maintenance MaintenanceConfig `json:"maintenance"`
}

// ArchiveConfig for the read-only finalized blocks database
type ArchiveConfig struct {
	// Database type (pebble, leveldb, or custom like parquet/clickhouse)
	Type string `json:"type"`

	// Path to archive database
	Path string `json:"path"`

	// Enable compression
	Compression bool `json:"compression"`

	// Compression algorithm (zstd, lz4, snappy)
	CompressionAlgo string `json:"compression_algo"`

	// Cache size in MB
	CacheSize int `json:"cache_size"`

	// Custom backend options (for specialized databases)
	BackendOptions map[string]interface{} `json:"backend_options"`
}

// CurrentConfig for the read-write recent blocks database
type CurrentConfig struct {
	// Database type (usually pebble for performance)
	Type string `json:"type"`

	// Path to current database
	Path string `json:"path"`

	// Retention period (blocks to keep before archiving)
	RetentionBlocks uint64 `json:"retention_blocks"`

	// Cache size in MB
	CacheSize int `json:"cache_size"`
}

// SyncConfig for fast sync using snapshots
type SyncConfig struct {
	// Enable snapshot sync
	SnapshotSync bool `json:"snapshot_sync"`

	// Snapshot source URLs
	SnapshotSources []string `json:"snapshot_sources"`

	// Verify snapshots
	VerifySnapshots bool `json:"verify_snapshots"`

	// Parallel download threads
	DownloadThreads int `json:"download_threads"`
}

// MaintenanceConfig for periodic optimization
type MaintenanceConfig struct {
	// Enable automatic archiving
	AutoArchive bool `json:"auto_archive"`

	// Archive interval
	ArchiveInterval time.Duration `json:"archive_interval"`

	// Compression interval
	CompressionInterval time.Duration `json:"compression_interval"`

	// Optimization interval
	OptimizationInterval time.Duration `json:"optimization_interval"`

	// Keep archive snapshots
	KeepSnapshots int `json:"keep_snapshots"`
}

// DefaultConfig returns a default dual database configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:       false,
		FinalityDelay: 32, // ~6.4 seconds at 200ms blocks

		Archive: ArchiveConfig{
			Type:            "pebble",
			Path:            "./chaindata/archive",
			Compression:     true,
			CompressionAlgo: "zstd",
			CacheSize:       1024, // 1GB
		},

		Current: CurrentConfig{
			Type:            "pebble",
			Path:            "./chaindata/current",
			RetentionBlocks: 128, // Keep ~25 seconds of blocks
			CacheSize:       2048, // 2GB
		},

		Sync: SyncConfig{
			SnapshotSync:    true,
			VerifySnapshots: true,
			DownloadThreads: 4,
		},

		Maintenance: MaintenanceConfig{
			AutoArchive:          true,
			ArchiveInterval:      1 * time.Hour,
			CompressionInterval:  24 * time.Hour,
			OptimizationInterval: 7 * 24 * time.Hour,
			KeepSnapshots:        3,
		},
	}
}