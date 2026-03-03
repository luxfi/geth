// Copyright (C) 2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapdb

import (
	"errors"
	"fmt"

	"github.com/luxfi/geth/ethdb"
	badger "github.com/luxfi/zapdb/v4"
)

var (
	// errNotSupported is returned if the database doesn't support a requested operation.
	errNotSupported = errors.New("this operation is not supported")
)

// Database is a zapdb (badgerdb fork) implementation of ethdb.Database
type Database struct {
	db *badger.DB
}

// New creates a new zapdb-backed database
func New(path string, cache int, handles int, namespace string, readonly bool) (ethdb.Database, error) {
	opts := badger.DefaultOptions(path)
	opts.ReadOnly = readonly
	opts.Logger = nil // Disable logging

	// Performance tuning for EVM workloads
	opts.SyncWrites = false
	opts.NumCompactors = 4
	opts.NumMemtables = 5
	opts.MemTableSize = 64 << 20    // 64 MB
	opts.BlockCacheSize = 256 << 20 // 256 MB
	opts.IndexCacheSize = 100 << 20 // 100 MB
	opts.BloomFalsePositive = 0.01
	opts.DetectConflicts = false
	opts.NumVersionsToKeep = 1
	opts.CompactL0OnClose = true

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

// Has checks if key exists
func (d *Database) Has(key []byte) (bool, error) {
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

// Get retrieves value for key
func (d *Database) Get(key []byte) ([]byte, error) {
	var value []byte
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		value, err = item.ValueCopy(nil)
		return err
	})
	return value, err
}

// Put stores value for key
func (d *Database) Put(key []byte, value []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, value)
	})
}

// Delete removes key
func (d *Database) Delete(key []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
}

// NewBatch creates a new batch
func (d *Database) NewBatch() ethdb.Batch {
	return &batch{
		db:   d.db,
		wb:   d.db.NewWriteBatch(),
		size: 0,
	}
}

// NewBatchWithSize creates a new batch with size hint
func (d *Database) NewBatchWithSize(size int) ethdb.Batch {
	return d.NewBatch()
}

// NewIterator creates a new iterator
func (d *Database) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	if d.db == nil {
		return &iterator{iter: nil, txn: nil}
	}

	txn := d.db.NewTransaction(false)
	if txn == nil {
		return &iterator{iter: nil, txn: nil}
	}

	opts := badger.DefaultIteratorOptions
	opts.Prefix = prefix
	iter := txn.NewIterator(opts)

	if iter != nil {
		if start != nil {
			iter.Seek(start)
		} else {
			iter.Rewind()
		}
	}

	return &iterator{
		iter:  iter,
		txn:   txn,
		first: true,
	}
}

// Stat returns database statistics
func (d *Database) Stat() (string, error) {
	lsm, vlog := d.db.Size()
	return fmt.Sprintf("LSM: %d bytes, VLog: %d bytes", lsm, vlog), nil
}

// Compact compacts the database
func (d *Database) Compact(start []byte, limit []byte) error {
	return d.db.Flatten(4)
}

// Close closes the database
func (d *Database) Close() error {
	return d.db.Close()
}

// batch implements ethdb.Batch
type batch struct {
	db   *badger.DB
	wb   *badger.WriteBatch
	size int
}

func (b *batch) Put(key []byte, value []byte) error {
	b.size += len(key) + len(value)
	return b.wb.Set(key, value)
}

func (b *batch) Delete(key []byte) error {
	b.size += len(key)
	return b.wb.Delete(key)
}

func (b *batch) ValueSize() int {
	return b.size
}

func (b *batch) Write() error {
	if err := b.wb.Flush(); err != nil {
		return err
	}
	return b.db.Sync()
}

func (b *batch) Reset() {
	b.wb.Cancel()
	b.wb = b.db.NewWriteBatch()
	b.size = 0
}

func (b *batch) DeleteRange(start, end []byte) error {
	return nil
}

func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	return nil
}

// iterator implements ethdb.Iterator
type iterator struct {
	iter  *badger.Iterator
	txn   *badger.Txn
	first bool
}

func (i *iterator) Next() bool {
	if i.iter == nil {
		return false
	}
	if i.first {
		i.first = false
		return i.iter.Valid()
	}
	if !i.iter.Valid() {
		return false
	}
	i.iter.Next()
	return i.iter.Valid()
}

func (i *iterator) Error() error {
	return nil
}

func (i *iterator) Key() []byte {
	if i.iter == nil || !i.iter.Valid() {
		return nil
	}
	return i.iter.Item().KeyCopy(nil)
}

func (i *iterator) Value() []byte {
	if i.iter == nil || !i.iter.Valid() {
		return nil
	}
	val, _ := i.iter.Item().ValueCopy(nil)
	return val
}

func (i *iterator) Release() {
	if i.iter != nil {
		i.iter.Close()
	}
	if i.txn != nil {
		i.txn.Discard()
	}
}

// Snapshot support
type Snapshot interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
	Release()
}

type snapshot struct {
	db  *badger.DB
	txn *badger.Txn
}

func (s *snapshot) Has(key []byte) (bool, error) {
	_, err := s.txn.Get(key)
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	return err == nil, err
}

func (s *snapshot) Get(key []byte) ([]byte, error) {
	item, err := s.txn.Get(key)
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

func (s *snapshot) Release() {
	s.txn.Discard()
}

func (d *Database) NewSnapshot() (Snapshot, error) {
	return &snapshot{
		db:  d.db,
		txn: d.db.NewTransaction(false),
	}, nil
}

func (d *Database) DeleteRange(start, end []byte) error {
	return d.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		iter := txn.NewIterator(opts)
		defer iter.Close()

		for iter.Seek(start); iter.Valid(); iter.Next() {
			key := iter.Item().KeyCopy(nil)
			if end != nil && string(key) >= string(end) {
				break
			}
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// Sync flushes all pending writes to disk
func (d *Database) Sync() error {
	return d.db.Sync()
}

func (d *Database) SyncKeyValue() error {
	return d.db.Sync()
}

// Ancient/Freezer stubs (zapdb doesn't use freezer)

func (d *Database) AncientDatadir() (string, error) { return "", nil }

func (d *Database) ReadAncients(fn func(ethdb.AncientReaderOp) error) error { return nil }

func (d *Database) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, nil
}

func (d *Database) TruncateHead(n uint64) (uint64, error) { return 0, nil }
func (d *Database) TruncateTail(n uint64) (uint64, error) { return 0, nil }
func (d *Database) SyncAncient() error                    { return nil }

func (d *Database) MigrateTable(s string, f func([]byte) ([]byte, error)) error { return nil }

func (d *Database) AncientOffSet() uint64                   { return 0 }
func (d *Database) Ancients() (uint64, error)               { return 0, nil }
func (d *Database) Tail() (uint64, error)                   { return 0, nil }
func (d *Database) AncientSize(kind string) (uint64, error) { return 0, nil }

func (d *Database) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errNotSupported
}

func (d *Database) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	return nil, errNotSupported
}

func (d *Database) HasAncient(kind string, number uint64) (bool, error) { return false, nil }
func (d *Database) Ancient(kind string, number uint64) ([]byte, error)  { return nil, errNotSupported }
func (d *Database) AncientBatch() ethdb.AncientWriteOp                  { return nil }
