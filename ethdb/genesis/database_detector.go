package genesis

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/log"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/badgerdb"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	LevelDB   DatabaseType = "leveldb"
	PebbleDB  DatabaseType = "pebbledb"
	BadgerDB  DatabaseType = "badgerdb"
	UnknownDB DatabaseType = "unknown"
)

// DetectDatabaseType detects the type of database at the given path
func DetectDatabaseType(dbPath string) DatabaseType {
	// Check for LevelDB (has CURRENT file)
	if _, err := os.Stat(filepath.Join(dbPath, "CURRENT")); err == nil {
		log.Info("Detected LevelDB", "path", dbPath)
		return LevelDB
	}
	
	// Check for PebbleDB (has MANIFEST-000000 or MANIFEST-000001)
	if _, err := os.Stat(filepath.Join(dbPath, "MANIFEST-000000")); err == nil {
		log.Info("Detected PebbleDB", "path", dbPath)
		return PebbleDB
	}
	if _, err := os.Stat(filepath.Join(dbPath, "MANIFEST-000001")); err == nil {
		log.Info("Detected PebbleDB", "path", dbPath)
		return PebbleDB
	}
	
	// Check for BadgerDB (has MANIFEST file)
	if _, err := os.Stat(filepath.Join(dbPath, "MANIFEST")); err == nil {
		log.Info("Detected BadgerDB", "path", dbPath)
		return BadgerDB
	}
	
	log.Warn("Unknown database type", "path", dbPath)
	return UnknownDB
}

// OpenDatabase opens a database of the detected type in read-only mode
func OpenDatabase(dbPath string, dbType DatabaseType) (ethdb.Database, error) {
	switch dbType {
	case LevelDB:
		return openLevelDB(dbPath)
	case PebbleDB:
		return openPebbleDB(dbPath)
	case BadgerDB:
		return openBadgerDB(dbPath)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// openLevelDB opens a LevelDB database in read-only mode
func openLevelDB(path string) (ethdb.Database, error) {
	// Import the actual LevelDB implementation
	// This would use go-ethereum's leveldb wrapper
	log.Info("Opening LevelDB", "path", path)
	
	// For now, return error as we need to import the actual implementation
	return nil, fmt.Errorf("LevelDB support not yet implemented")
}

// The actual openPebbleDB is now in pebble_wrapper.go

// openBadgerDB opens a BadgerDB database in read-only mode
func openBadgerDB(path string) (ethdb.Database, error) {
	log.Info("Opening BadgerDB", "path", path)
	return badgerdb.NewBadgerDatabase(path, true, true)
}

// DatabaseInfo contains information about a database
type DatabaseInfo struct {
	Type       DatabaseType
	Path       string
	Size       int64
	FileCount  int
	IsReadOnly bool
}

// GetDatabaseInfo returns information about a database
func GetDatabaseInfo(dbPath string) (*DatabaseInfo, error) {
	info := &DatabaseInfo{
		Path: dbPath,
		Type: DetectDatabaseType(dbPath),
	}
	
	// Calculate size and file count
	var totalSize int64
	var fileCount int
	
	err := filepath.Walk(dbPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			totalSize += fi.Size()
			fileCount++
		}
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}
	
	info.Size = totalSize
	info.FileCount = fileCount
	
	return info, nil
}