package dualdb

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// SharedArchiveManager manages a shared read-only archive database
// that can be accessed by multiple luxd instances simultaneously
type SharedArchiveManager struct {
	mu sync.RWMutex
	
	// Path to shared archive
	archivePath string
	
	// Reference count for active readers
	readers int32
	
	// Shared database options for read-only access
	readOnlyOpts *pebble.Options
}

// NewSharedArchiveManager creates a manager for shared archive access
func NewSharedArchiveManager(archivePath string) *SharedArchiveManager {
	opts := rawdb.DefaultPebbleOptions()
	opts.ReadOnly = true
	opts.ErrorIfNotExists = false
	
	// Disable file locking for shared read-only access
	opts.Lock = nil
	
	// Optimize for read-only access
	opts.DisableWAL = true
	opts.MemTableStopWritesThreshold = 1
	
	return &SharedArchiveManager{
		archivePath:  archivePath,
		readOnlyOpts: opts,
	}
}

// OpenSharedArchive opens a read-only connection to the shared archive
func (m *SharedArchiveManager) OpenSharedArchive() (ethdb.Database, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if archive exists
	if _, err := os.Stat(m.archivePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("shared archive not found at %s", m.archivePath)
	}
	
	// Open database in read-only mode without locks
	db, err := rawdb.NewPebbleDBDatabase(m.archivePath, 1024, 0, "", m.readOnlyOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to open shared archive: %w", err)
	}
	
	m.readers++
	log.Info("Opened shared archive connection", "path", m.archivePath, "readers", m.readers)
	
	return db, nil
}

// SharedDualDatabase extends DualDatabase for multi-node deployment
type SharedDualDatabase struct {
	*DualDatabase
	
	// Node-specific identifier
	nodeID string
	
	// Shared archive path (network mount, shared storage, etc.)
	sharedArchivePath string
	
	// Local current database path (node-specific)
	localCurrentPath string
}

// NewSharedDualDatabase creates a dual database with shared archive
func NewSharedDualDatabase(nodeID string, sharedArchivePath string, localDataDir string, config *Config) (*SharedDualDatabase, error) {
	// Override paths for shared deployment
	config.Archive.Path = sharedArchivePath
	config.Current.Path = filepath.Join(localDataDir, nodeID, "current")
	
	// Ensure archive is read-only
	config.Archive.BackendOptions = map[string]interface{}{
		"read_only":        true,
		"bypass_lock":      true,  // For backends that support it
		"shared_mode":      true,
		"cache_shared":     true,  // Share cache across readers if possible
	}
	
	log.Info("Initializing shared dual database",
		"node_id", nodeID,
		"shared_archive", sharedArchivePath,
		"local_current", config.Current.Path)
	
	// Create base dual database
	dualDB, err := NewDualDatabase(config)
	if err != nil {
		return nil, err
	}
	
	return &SharedDualDatabase{
		DualDatabase:      dualDB,
		nodeID:           nodeID,
		sharedArchivePath: sharedArchivePath,
		localCurrentPath:  config.Current.Path,
	}, nil
}

// Example configuration for multi-node deployment:
/*

Node Architecture:

┌─────────────────────────────────────────────────────────┐
│                    Shared Storage (NFS/S3/etc)          │
│  ┌─────────────────────────────────────────────────┐   │
│  │          Shared Archive Database (RO)            │   │
│  │   - Finalized blocks (height 0 to N-finality)   │   │
│  │   - Compressed & optimized                      │   │
│  │   - Periodic snapshots for fast sync            │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
┌───────▼────────┐ ┌───────▼────────┐ ┌───────▼────────┐
│   Node 1       │ │   Node 2       │ │   Node 3       │
│                │ │                │ │                │
│ Local Current  │ │ Local Current  │ │ Local Current  │
│   Database     │ │   Database     │ │   Database     │
│ - Recent blocks│ │ - Recent blocks│ │ - Recent blocks│
│ - Pending state│ │ - Pending state│ │ - Pending state│
└────────────────┘ └────────────────┘ └────────────────┘

Benefits:
1. Shared archive reduces storage requirements
2. Fast node bootstrap from archive snapshots  
3. Each node maintains only recent state
4. Archive can be optimized/compressed periodically
5. Archive can use specialized read-optimized storage

*/

// CreateSharedArchiveSnapshot creates a snapshot of finalized blocks
// This can be run by a single maintenance node periodically
func CreateSharedArchiveSnapshot(sourceDB ethdb.Database, targetPath string, finalizedHeight uint64) error {
	log.Info("Creating shared archive snapshot",
		"target", targetPath,
		"finalized_height", finalizedHeight)
	
	// Create snapshot directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to create snapshot directory: %w", err)
	}
	
	// Open target database with compression
	opts := rawdb.DefaultPebbleOptions()
	opts.Compression = pebble.ZstdCompression
	opts.FormatMajorVersion = pebble.FormatNewest
	
	targetDB, err := pebble.Open(targetPath, opts)
	if err != nil {
		return fmt.Errorf("failed to create snapshot database: %w", err)
	}
	defer targetDB.Close()
	
	// Copy all finalized blocks and state
	batch := targetDB.NewBatch()
	count := 0
	
	// Use iterator to copy all relevant data
	iter := sourceDB.NewIterator(nil, nil)
	defer iter.Release()
	
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Filter to only include finalized data
		// (This is simplified - real implementation would parse keys properly)
		if shouldIncludeInArchive(key, finalizedHeight) {
			if err := batch.Set(key, value, nil); err != nil {
				return fmt.Errorf("failed to write to batch: %w", err)
			}
			count++
			
			// Commit batch periodically
			if count%10000 == 0 {
				if err := batch.Commit(nil); err != nil {
					return fmt.Errorf("failed to commit batch: %w", err)
				}
				batch = targetDB.NewBatch()
				log.Info("Archive snapshot progress", "keys", count)
			}
		}
	}
	
	// Final batch commit
	if err := batch.Commit(nil); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}
	
	log.Info("Archive snapshot created successfully",
		"path", targetPath,
		"keys", count)
	
	return nil
}

// shouldIncludeInArchive determines if a key should be included in archive
func shouldIncludeInArchive(key []byte, finalizedHeight uint64) bool {
	// This is a simplified version - real implementation would:
	// 1. Parse the key to determine data type
	// 2. Check if it's related to finalized blocks
	// 3. Exclude recent/pending state
	// 4. Include all headers, bodies, receipts up to finalizedHeight
	// 5. Include finalized state trie nodes
	
	// For now, just return true for demonstration
	return true
}