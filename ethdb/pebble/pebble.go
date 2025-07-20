// Package pebble provides a wrapper for go-ethereum's pebble implementation
package pebble

import (
	gopebble "github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Database wraps go-ethereum's pebble.Database
type Database = gopebble.Database

// New returns a wrapped Pebble database
func New(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := gopebble.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	// Wrap the go-ethereum database with our wrapper
	return ethdb.NewDatabase(db), nil
}