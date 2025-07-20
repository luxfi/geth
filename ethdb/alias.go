// (c) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package ethdb

import (
	"github.com/ethereum/go-ethereum/ethdb"
)

// Type aliases for go-ethereum database interfaces
// These aliases allow us to use ethereum types transparently while maintaining our own package structure
type (
	// Core interfaces - use ethereum types directly as aliases
	Batch         = ethdb.Batch
	Database      = ethdb.Database
	Iterator      = ethdb.Iterator
	
	// Other interfaces
	Batcher                = ethdb.Batcher
	KeyValueStore          = ethdb.KeyValueStore
	KeyValueReader         = ethdb.KeyValueReader
	KeyValueWriter         = ethdb.KeyValueWriter
	KeyValueStater         = ethdb.KeyValueStater
	Iteratee               = ethdb.Iteratee
	AncientReader          = ethdb.AncientReader
	AncientWriter          = ethdb.AncientWriter
	AncientStater          = ethdb.AncientStater
	AncientWriteOp         = ethdb.AncientWriteOp
	AncientReaderOp        = ethdb.AncientReaderOp
	Reader                 = ethdb.Reader
	// These interfaces might be embedded in Database, not separate
	// Writer                 = ethdb.Writer
	// Stater                 = ethdb.Stater
	Compacter              = ethdb.Compacter
	// Closer                 = ethdb.Closer
	// Snapshotter            = ethdb.Snapshotter
	AncientStore           = ethdb.AncientStore
	ResettableAncientStore = ethdb.ResettableAncientStore
)

// Constants
const (
	IdealBatchSize = ethdb.IdealBatchSize
)

// Error values - these might not exist in newer versions
// var (
// 	ErrKeyNotFound = ethdb.ErrKeyNotFound
// 	ErrNotFound    = ethdb.ErrNotFound
// )

// Re-export utility functions - these might be in rawdb package now
// var (
// 	HasCode          = ethdb.HasCode
// 	HasBody          = ethdb.HasBody
// 	HasReceipts      = ethdb.HasReceipts
// 	HasHeader        = ethdb.HasHeader
// 	IsCodeKey        = ethdb.IsCodeKey
// 	IsLegacyTrieNode = ethdb.IsLegacyTrieNode
// )
