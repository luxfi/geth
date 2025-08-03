// Copyright 2025 The go-ethereum Authors
// Copyright (C) 2019-2025, Lux Partners Limited. All rights reserved.
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package badgerdb

import (
	"bytes"
	"errors"

	"github.com/dgraph-io/badger/v4"
	"github.com/luxfi/geth/ethdb"
)

var errNotSupported = errors.New("not supported")

// New returns a wrapped badgerdb database that implements ethdb.Database
func New(file string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	// Create badgerdb instance
	opts := badger.DefaultOptions(file)
	opts.ReadOnly = readonly
	opts.Logger = nil // Disable badger's own logging
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &database{db: db}, nil
}

// database wraps a badger.DB to implement ethdb.Database
type database struct {
	db *badger.DB
}

// Close implements ethdb.Database
func (d *database) Close() error {
	return d.db.Close()
}

// Has implements ethdb.KeyValueReader
func (d *database) Has(key []byte) (bool, error) {
	err := d.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Get implements ethdb.KeyValueReader
func (d *database) Get(key []byte) ([]byte, error) {
	var val []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		val, err = item.ValueCopy(nil)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return nil, nil
	}
	return val, err
}

// Put implements ethdb.KeyValueWriter
func (d *database) Put(key []byte, value []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// Delete implements ethdb.KeyValueDeleter
func (d *database) Delete(key []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// DeleteRange deletes all keys with prefix (not fully supported)
func (d *database) DeleteRange(start, end []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(start); it.Valid(); it.Next() {
			key := it.Item().Key()
			if end != nil && bytes.Compare(key, end) >= 0 {
				break
			}
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// NewBatch implements ethdb.Batcher
func (d *database) NewBatch() ethdb.Batch {
	return &batch{db: d.db, ops: make([]batchOp, 0)}
}

// NewBatchWithSize implements ethdb.Batcher
func (d *database) NewBatchWithSize(size int) ethdb.Batch {
	return &batch{db: d.db, ops: make([]batchOp, 0, size)}
}

// NewIterator implements ethdb.Iteratee
func (d *database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	txn := d.db.NewTransaction(false)
	opts := badger.DefaultIteratorOptions
	opts.PrefetchSize = 10
	it := txn.NewIterator(opts)
	
	if start != nil {
		it.Seek(start)
	} else if prefix != nil {
		it.Seek(prefix)
	}
	
	return &iterator{txn: txn, it: it, prefix: prefix}
}

// Stat implements ethdb.Stater
func (d *database) Stat() (string, error) {
	return "badgerdb", nil
}

// Compact implements ethdb.Compacter
func (d *database) Compact(start []byte, limit []byte) error {
	// BadgerDB handles compaction automatically via GC
	return d.db.RunValueLogGC(0.5)
}

// SyncKeyValue implements ethdb.Database
func (d *database) SyncKeyValue() error {
	// BadgerDB automatically syncs data, this is a no-op
	return nil
}

// Ancient store methods (not supported by badgerdb)

// AncientDatadir returns the path of the ancient store
func (d *database) AncientDatadir() (string, error) {
	return "", errNotSupported
}

// HasAncient returns an indicator whether the specified ancient data exists
func (d *database) HasAncient(kind string, number uint64) (bool, error) {
	return false, nil
}

// Ancient retrieves an ancient binary blob
func (d *database) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

// AncientRange retrieves multiple ancient binary blobs
func (d *database) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errNotSupported
}

// Ancients returns the number of ancient items
func (d *database) Ancients() (uint64, error) {
	return 0, nil
}

// Tail returns the number of the first item
func (d *database) Tail() (uint64, error) {
	return 0, nil
}

// AncientSize returns the size of the ancient store
func (d *database) AncientSize(kind string) (uint64, error) {
	return 0, nil
}

// ReadAncients runs a read operation on the ancient store
func (d *database) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	return fn(d)
}

// ModifyAncients runs a write operation on the ancient store
func (d *database) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, errNotSupported
}

// SyncAncient flushes ancient store data to disk
func (d *database) SyncAncient() error {
	return nil
}

// TruncateHead discards recent ancient data
func (d *database) TruncateHead(n uint64) (uint64, error) {
	return 0, errNotSupported
}

// TruncateTail discards oldest ancient data
func (d *database) TruncateTail(n uint64) (uint64, error) {
	return 0, errNotSupported
}

// MigrateTable migrates a table to new format (no-op for badgerdb)
func (d *database) MigrateTable(kind string, convert func([]byte) ([]byte, error)) error {
	return nil
}

// batchOp represents a single batch operation
type batchOp struct {
	isDelete bool
	key      []byte
	value    []byte
}

// batch wraps badger batch operations to implement ethdb.Batch
type batch struct {
	db   *badger.DB
	ops  []batchOp
	size int
}

// Put implements ethdb.Batch
func (b *batch) Put(key []byte, value []byte) error {
	b.ops = append(b.ops, batchOp{
		isDelete: false,
		key:      append([]byte(nil), key...),
		value:    append([]byte(nil), value...),
	})
	b.size += len(key) + len(value)
	return nil
}

// Delete implements ethdb.Batch
func (b *batch) Delete(key []byte) error {
	b.ops = append(b.ops, batchOp{
		isDelete: true,
		key:      append([]byte(nil), key...),
	})
	b.size += len(key)
	return nil
}

// DeleteRange deletes all keys with prefix (not supported)
func (b *batch) DeleteRange(start, end []byte) error {
	// Not supported by badgerdb batch
	return errNotSupported
}

// ValueSize implements ethdb.Batch
func (b *batch) ValueSize() int {
	return b.size
}

// Write implements ethdb.Batch
func (b *batch) Write() error {
	return b.db.Update(func(txn *badger.Txn) error {
		for _, op := range b.ops {
			if op.isDelete {
				if err := txn.Delete(op.key); err != nil {
					return err
				}
			} else {
				if err := txn.Set(op.key, op.value); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Reset implements ethdb.Batch
func (b *batch) Reset() {
	b.ops = b.ops[:0]
	b.size = 0
}

// Replay implements ethdb.Batch
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	for _, op := range b.ops {
		if op.isDelete {
			if deleter, ok := w.(interface{ Delete([]byte) error }); ok {
				if err := deleter.Delete(op.key); err != nil {
					return err
				}
			}
		} else {
			if err := w.Put(op.key, op.value); err != nil {
				return err
			}
		}
	}
	return nil
}


// iterator wraps a badger iterator to implement ethdb.Iterator
type iterator struct {
	txn    *badger.Txn
	it     *badger.Iterator
	prefix []byte
}

// Next implements ethdb.Iterator
func (i *iterator) Next() bool {
	i.it.Next()
	if !i.it.Valid() {
		return false
	}
	// Check prefix if set
	if i.prefix != nil {
		key := i.it.Item().Key()
		if !bytes.HasPrefix(key, i.prefix) {
			return false
		}
	}
	return true
}

// Error implements ethdb.Iterator
func (i *iterator) Error() error {
	return i.it.Error()
}

// Key implements ethdb.Iterator
func (i *iterator) Key() []byte {
	if !i.it.Valid() {
		return nil
	}
	return i.it.Item().KeyCopy(nil)
}

// Value implements ethdb.Iterator
func (i *iterator) Value() []byte {
	if !i.it.Valid() {
		return nil
	}
	val, err := i.it.Item().ValueCopy(nil)
	if err != nil {
		return nil
	}
	return val
}

// Release implements ethdb.Iterator
func (i *iterator) Release() {
	i.it.Close()
	i.txn.Discard()
}
