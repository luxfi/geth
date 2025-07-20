// Package pebble provides a wrapper for go-ethereum's pebble implementation
package pebble

import (
	gopebble "github.com/ethereum/go-ethereum/ethdb/pebble"
)

// Database is an alias for go-ethereum's pebble.Database
type Database = gopebble.Database

// New returns a Pebble database backend
func New(file string, cache int, handles int, namespace string, readonly bool) (*Database, error) {
	return gopebble.New(file, cache, handles, namespace, readonly)
}
