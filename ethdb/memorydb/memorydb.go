// Package memorydb provides a wrapper for go-ethereum's memorydb implementation
package memorydb

import (
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

// Re-export memorydb types
type Database = memorydb.Database

// Re-export functions
var (
	New = memorydb.New
	NewWithCap = memorydb.NewWithCap
)