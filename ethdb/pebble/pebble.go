// Package pebble provides a wrapper for go-ethereum's pebble implementation
package pebble

import (
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/luxfi/geth/ethdb"
)

// New returns a wrapped Pebble database
func New(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	db, err := pebble.New(file, cache, handles, namespace, readonly)
	if err != nil {
		return nil, err
	}
	return ethdb.NewDatabase(db), nil
}