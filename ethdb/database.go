// (c) 2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.

package ethdb

import (
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