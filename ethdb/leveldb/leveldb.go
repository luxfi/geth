// Package leveldb provides a wrapper for go-ethereum's leveldb implementation
package leveldb

import (
	goleveldb "github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Database wraps go-ethereum's leveldb.Database
type Database = goleveldb.Database

// New returns a wrapped LevelDB database
func New(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := goleveldb.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	// Wrap the go-ethereum database with our wrapper
	return ethdb.NewDatabase(db), nil
}
