// (c) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package ethdb

import (
	"github.com/luxfi/geth/ethdb"
)

// Type aliases for go-ethereum database interfaces
// We use our own Database and Batch interfaces (see database.go) which extend the base ones
type (
	Batcher        = ethdb.Batcher
	KeyValueStore  = ethdb.KeyValueStore
	KeyValueReader = ethdb.KeyValueReader
	KeyValueWriter = ethdb.KeyValueWriter
	KeyValueStater = ethdb.KeyValueStater
	Iterator       = ethdb.Iterator
	Iteratee       = ethdb.Iteratee
	AncientReader  = ethdb.AncientReader
	AncientWriter  = ethdb.AncientWriter
	AncientStater  = ethdb.AncientStater
	AncientWriteOp = ethdb.AncientWriteOp
	AncientReaderOp = ethdb.AncientReaderOp
	Reader         = ethdb.Reader
	Writer         = ethdb.Writer
	Stater         = ethdb.Stater
	Compacter      = ethdb.Compacter
	Closer         = ethdb.Closer
	Snapshotter    = ethdb.Snapshotter
	AncientStore   = ethdb.AncientStore
	ResettableAncientStore = ethdb.ResettableAncientStore
)

// Constants
const (
	IdealBatchSize = ethdb.IdealBatchSize
)

// Error values
var (
	ErrKeyNotFound = ethdb.ErrKeyNotFound
	ErrNotFound    = ethdb.ErrNotFound
)

// Re-export utility functions
var (
	HasCode     = ethdb.HasCode
	HasBody     = ethdb.HasBody
	HasReceipts = ethdb.HasReceipts
	HasHeader   = ethdb.HasHeader
	IsCodeKey   = ethdb.IsCodeKey
	IsLegacyTrieNode = ethdb.IsLegacyTrieNode
)