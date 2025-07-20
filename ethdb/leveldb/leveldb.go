// Package leveldb provides a wrapper for go-ethereum's leveldb implementation
package leveldb

import (
	goleveldb "github.com/ethereum/go-ethereum/ethdb/leveldb"
)

// Database is an alias for go-ethereum's leveldb.Database
type Database = goleveldb.Database

// New returns a LevelDB database backend
func New(file string, cache int, handles int, namespace string, readonly bool) (*Database, error) {
	return goleveldb.New(file, cache, handles, namespace, readonly)
}
