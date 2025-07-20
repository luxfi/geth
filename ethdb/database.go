// Package ethdb provides wrapper types for go-ethereum's ethdb interfaces
package ethdb

import (
	"errors"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Database interface that extends ethdb.Database with SyncAncient
type Database interface {
	ethdb.Database
	SyncAncient() error
}

// Batch interface that extends ethdb.Batch with DeleteRange
type Batch interface {
	ethdb.Batch
	DeleteRange(start, end []byte) error
}

// Re-export all other ethdb types from go-ethereum
type (
	Batcher    = ethdb.Batcher
	Iterator   = ethdb.Iterator
	Iteratee   = ethdb.Iteratee
	KeyValueReader = ethdb.KeyValueReader
	KeyValueWriter = ethdb.KeyValueWriter
	KeyValueStater = ethdb.KeyValueStater
	KeyValueStore = ethdb.KeyValueStore
	Compacter  = ethdb.Compacter
	AncientReader = ethdb.AncientReader
	AncientWriter = ethdb.AncientWriter
	AncientStater = ethdb.AncientStater
	AncientWriteOp = ethdb.AncientWriteOp
	AncientReaderOp = ethdb.AncientReaderOp
	Reader     = ethdb.Reader
	AncientStore = ethdb.AncientStore
	ResettableAncientStore = ethdb.ResettableAncientStore
)

// Re-export constants
const (
	IdealBatchSize = ethdb.IdealBatchSize
)

// Re-export errors
var (
	// ErrInvalidBatch is returned when batch is invalid.
	ErrInvalidBatch = errors.New("invalid batch")
)

