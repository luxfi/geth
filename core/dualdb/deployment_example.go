package dualdb

// Example deployment configurations for different scenarios

// SingleNodeConfig - Traditional single node with dual DB
func SingleNodeConfig() *Config {
	return &Config{
		Enabled:       true,
		FinalityDelay: 32, // ~6.4 seconds

		Archive: ArchiveConfig{
			Type:            "pebble",
			Path:            "./chaindata/archive",
			Compression:     true,
			CompressionAlgo: "zstd",
			CacheSize:       1024,
		},

		Current: CurrentConfig{
			Type:            "pebble",
			Path:            "./chaindata/current",
			RetentionBlocks: 128,
			CacheSize:       2048,
		},

		Maintenance: MaintenanceConfig{
			AutoArchive:         true,
			ArchiveInterval:     1 * time.Hour,
			CompressionInterval: 24 * time.Hour,
		},
	}
}

// SharedArchiveConfig - Multiple nodes sharing read-only archive
func SharedArchiveConfig(nodeID string) *Config {
	return &Config{
		Enabled:       true,
		FinalityDelay: 32,

		Archive: ArchiveConfig{
			Type:            "pebble",
			Path:            "/mnt/shared/lux-archive", // Shared mount point
			Compression:     false, // Already compressed
			CacheSize:       512,   // Smaller cache per node
			BackendOptions: map[string]interface{}{
				"read_only":   true,
				"bypass_lock": true,
				"shared_mode": true,
			},
		},

		Current: CurrentConfig{
			Type:            "pebble",
			Path:            fmt.Sprintf("./chaindata/%s/current", nodeID),
			RetentionBlocks: 64, // Less retention needed
			CacheSize:       2048,
		},

		Sync: SyncConfig{
			SnapshotSync: true,
			SnapshotSources: []string{
				"https://snapshots.lux.network/archive/latest",
				"s3://lux-snapshots/archive/latest",
			},
			VerifySnapshots: true,
			DownloadThreads: 8,
		},

		Maintenance: MaintenanceConfig{
			AutoArchive: false, // Archive managed by dedicated node
		},
	}
}

// ArchiveMaintainerConfig - Dedicated node that maintains the archive
func ArchiveMaintainerConfig() *Config {
	return &Config{
		Enabled:       true,
		FinalityDelay: 32,

		Archive: ArchiveConfig{
			Type:            "pebble",
			Path:            "/mnt/shared/lux-archive",
			Compression:     true,
			CompressionAlgo: "zstd",
			CacheSize:       4096, // Large cache for maintenance
		},

		Current: CurrentConfig{
			Type:            "pebble",
			Path:            "./chaindata/maintainer/current",
			RetentionBlocks: 256, // Keep more for archiving
			CacheSize:       2048,
		},

		Maintenance: MaintenanceConfig{
			AutoArchive:          true,
			ArchiveInterval:      30 * time.Minute,     // More frequent
			CompressionInterval:  6 * time.Hour,        // Compress often
			OptimizationInterval: 24 * time.Hour,       // Daily optimization
			KeepSnapshots:        5,                    // Keep more snapshots
		},
	}
}

// CloudOptimizedConfig - For cloud deployments with object storage
func CloudOptimizedConfig(nodeID string) *Config {
	return &Config{
		Enabled:       true,
		FinalityDelay: 64, // Higher finality for cloud

		Archive: ArchiveConfig{
			Type: "custom", // Custom S3-backed implementation
			Path: "s3://lux-archive/mainnet",
			BackendOptions: map[string]interface{}{
				"backend":       "s3",
				"region":        "us-east-1",
				"read_only":     true,
				"cache_blocks":  true,
				"prefetch_size": 10, // Prefetch 10 blocks
			},
			CacheSize: 2048, // Larger cache for S3 latency
		},

		Current: CurrentConfig{
			Type:            "pebble",
			Path:            fmt.Sprintf("/ephemeral/chaindata/%s", nodeID),
			RetentionBlocks: 32,  // Minimal retention
			CacheSize:       4096, // Large cache
		},

		Sync: SyncConfig{
			SnapshotSync: true,
			SnapshotSources: []string{
				"s3://lux-snapshots/archive/latest",
			},
			VerifySnapshots: true,
			DownloadThreads: 16, // More threads for S3
		},
	}
}

/*
Deployment Examples:

1. SINGLE HIGH-PERFORMANCE NODE:
   - Use SingleNodeConfig()
   - Both DBs on fast NVMe storage
   - Full archival node capabilities

2. MULTI-NODE CLUSTER (Shared Storage):
   ```
   # Archive maintainer node
   ./luxd --dual-db-config=maintainer
   
   # Regular nodes (share archive)
   ./luxd --dual-db-config=shared --node-id=node1
   ./luxd --dual-db-config=shared --node-id=node2
   ```

3. CLOUD DEPLOYMENT (S3 Archive):
   ```
   # Nodes read from S3, write locally
   ./luxd --dual-db-config=cloud --node-id=i-abc123
   ```

4. FAST SYNC NEW NODE:
   ```
   # Download archive snapshot
   wget https://snapshots.lux.network/archive/latest.tar.zst
   tar -xf latest.tar.zst -C /mnt/shared/
   
   # Start node (syncs only recent blocks)
   ./luxd --dual-db-config=shared --node-id=new-node
   ```

5. VALIDATOR SETUP:
   ```
   # Validators can use minimal current DB
   ./luxd --dual-db-config=validator \
          --dual-db-current-retention=32 \
          --dual-db-archive-path=/mnt/shared/lux-archive
   ```
*/