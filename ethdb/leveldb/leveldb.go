// Package leveldb provides a wrapper for go-ethereum's leveldb implementation
package leveldb

import (
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/luxfi/geth/ethdb"
)

// New returns a wrapped LevelDB database
func New(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := leveldb.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	// leveldb.New returns ethdb.Database, not *leveldb.Database
	return ethdb.NewDatabase(db), nil
}