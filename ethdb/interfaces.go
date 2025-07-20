// Package ethdb provides extended interfaces for database operations
package ethdb

import (
	gethdb "github.com/luxfi/geth/ethdb"
)

// databaseWrapper wraps a gethdb.Database to implement our extended Database interface
type databaseWrapper struct {
	gethdb.Database
}

// SyncAncient syncs the ancient store
func (db *databaseWrapper) SyncAncient() error {
	// Check if the underlying database supports SyncAncient
	type syncer interface {
		SyncAncient() error
	}
	if s, ok := db.Database.(syncer); ok {
		return s.SyncAncient()
	}
	// If not supported, it's a no-op
	return nil
}

// NewBatch creates a write-only database that buffers changes to its host db
func (db *databaseWrapper) NewBatch() gethdb.Batch {
	return NewBatch(db.Database.NewBatch())
}

// NewBatchWithSize creates a write-only database batch with pre-allocated buffer
func (db *databaseWrapper) NewBatchWithSize(size int) gethdb.Batch {
	return NewBatch(db.Database.NewBatchWithSize(size))
}

// batchWrapper wraps a gethdb.Batch to implement our extended Batch interface
type batchWrapper struct {
	gethdb.Batch
}

// DeleteRange deletes all keys in the range [start, end)
func (b *batchWrapper) DeleteRange(start, end []byte) error {
	// Check if the underlying batch supports DeleteRange
	type rangeDeleter interface {
		DeleteRange(start, end []byte) error
	}
	if rd, ok := b.Batch.(rangeDeleter); ok {
		return rd.DeleteRange(start, end)
	}
	// If not supported, it's a no-op
	return nil
}

// NewDatabase creates a new Database that wraps a gethdb.Database
func NewDatabase(db gethdb.Database) Database {
	// If it already implements our interface, return as-is
	if d, ok := db.(Database); ok {
		return d
	}
	return &databaseWrapper{Database: db}
}

// NewBatch creates a new Batch that wraps a gethdb.Batch
func NewBatch(b gethdb.Batch) Batch {
	// If it already implements our interface, return as-is
	if batch, ok := b.(Batch); ok {
		return batch
	}
	return &batchWrapper{Batch: b}
}