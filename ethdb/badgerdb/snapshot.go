package badgerdb

import (
	"errors"
	
	"github.com/dgraph-io/badger/v3"
)

// BadgerSnapshot implements ethdb.Snapshot using BadgerDB's native snapshot support
type BadgerSnapshot struct {
	db  *badger.DB
	txn *badger.Txn
}

// NewBadgerSnapshot creates a new snapshot using BadgerDB's read-only transaction
func NewBadgerSnapshot(db *badger.DB) (*BadgerSnapshot, error) {
	// BadgerDB uses read-only transactions as snapshots
	// They provide a consistent view of the database at a point in time
	txn := db.NewTransaction(false) // false = read-only
	
	return &BadgerSnapshot{
		db:  db,
		txn: txn,
	}, nil
}

// Has checks if a key exists in the snapshot
func (s *BadgerSnapshot) Has(key []byte) (bool, error) {
	_, err := s.txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get retrieves a value from the snapshot
func (s *BadgerSnapshot) Get(key []byte) ([]byte, error) {
	item, err := s.txn.Get(key)
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, badger.ErrKeyNotFound
		}
		return nil, err
	}
	
	value, err := item.ValueCopy(nil)
	if err != nil {
		return nil, err
	}
	
	return value, nil
}

// Release releases the snapshot
func (s *BadgerSnapshot) Release() {
	if s.txn != nil {
		s.txn.Discard()
		s.txn = nil
	}
}