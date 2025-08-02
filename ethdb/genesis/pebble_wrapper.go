package genesis

import (
	"errors"
	
	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/ethdb"
)

// openPebbleDB opens a PebbleDB database in read-only mode
func openPebbleDB(path string) (ethdb.Database, error) {
	opts := &pebble.Options{
		ReadOnly: true,
	}
	
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}
	
	// Return a simple wrapper that implements ethdb.Database
	return &pebbleWrapper{db: db}, nil
}

// pebbleWrapper is a minimal wrapper around pebble.DB
type pebbleWrapper struct {
	db *pebble.DB
}

// Get retrieves the given key if it's present in the key-value data store.
func (p *pebbleWrapper) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	
	// Copy the value since it's only valid until closer.Close()
	result := make([]byte, len(val))
	copy(result, val)
	return result, nil
}

// Has retrieves if a key is present in the key-value data store.
func (p *pebbleWrapper) Has(key []byte) (bool, error) {
	_, closer, err := p.db.Get(key)
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	closer.Close()
	return true, nil
}

// Put inserts the given value into the key-value data store.
func (p *pebbleWrapper) Put(key []byte, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

// Delete removes the key from the key-value data store.
func (p *pebbleWrapper) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

// DeleteRange deletes all of the keys (and values) in the range [start,end)
func (p *pebbleWrapper) DeleteRange(start, end []byte) error {
	return p.db.DeleteRange(start, end, pebble.Sync)
}

// NewBatch creates a write-only database that buffers changes to its host db
func (p *pebbleWrapper) NewBatch() ethdb.Batch {
	return &pebbleBatch{db: p.db, b: p.db.NewBatch()}
}

// NewBatchWithSize creates a write-only database batch with pre-allocated buffer.
func (p *pebbleWrapper) NewBatchWithSize(size int) ethdb.Batch {
	return p.NewBatch()
}

// NewIterator creates a binary-alphabetical iterator over a subset
func (p *pebbleWrapper) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	opts := &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	}
	if start != nil {
		opts.LowerBound = start
	}
	iter, err := p.db.NewIter(opts)
	if err != nil {
		// Return a dummy iterator that immediately returns false on Next()
		return &pebbleIterator{iter: nil, err: err}
	}
	return &pebbleIterator{iter: iter}
}

// Stat returns the statistic data of the database.
func (p *pebbleWrapper) Stat(property string) (string, error) {
	metrics := p.db.Metrics()
	return metrics.String(), nil
}

// SyncKeyValue ensures that all pending writes are flushed to disk
func (p *pebbleWrapper) SyncKeyValue() error {
	return p.db.Flush()
}

// Compact flattens the underlying data store for the given key range.
func (p *pebbleWrapper) Compact(start []byte, limit []byte) error {
	return p.db.Compact(start, limit, true)
}

// NewSnapshot creates a database snapshot based on the current state.
func (p *pebbleWrapper) NewSnapshot() (ethdb.Snapshot, error) {
	// Pebble snapshots don't implement ethdb.Snapshot interface directly
	// Return an error for now
	return nil, errors.New("snapshots not supported in pebble wrapper")
}

// Sync flushes all pending writes to disk
func (p *pebbleWrapper) Sync() error {
	return p.db.Flush()
}

// Close closes the database
func (p *pebbleWrapper) Close() error {
	return p.db.Close()
}

// Ancient store methods (not supported)
func (p *pebbleWrapper) HasAncient(kind string, number uint64) (bool, error) {
	return false, nil
}

func (p *pebbleWrapper) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, errNotSupported
}

func (p *pebbleWrapper) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, errNotSupported
}

func (p *pebbleWrapper) Ancients() (uint64, error) {
	return 0, errNotSupported
}

func (p *pebbleWrapper) Tail() (uint64, error) {
	return 0, errNotSupported
}

func (p *pebbleWrapper) AncientSize(kind string) (uint64, error) {
	return 0, errNotSupported
}

func (p *pebbleWrapper) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	return errNotSupported
}

func (p *pebbleWrapper) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, errNotSupported
}

func (p *pebbleWrapper) SyncAncient() error {
	return errNotSupported
}

func (p *pebbleWrapper) TruncateHead(n uint64) error {
	return errNotSupported
}

func (p *pebbleWrapper) TruncateTail(n uint64) error {
	return errNotSupported
}

func (p *pebbleWrapper) AncientDatadir() (string, error) {
	return "", errNotSupported
}

func (p *pebbleWrapper) MigrateTable(kind string, convert func([]byte) ([]byte, error)) error {
	return errNotSupported
}

// pebbleBatch is a write-only batch
type pebbleBatch struct {
	db *pebble.DB
	b  *pebble.Batch
}

func (b *pebbleBatch) Put(key, value []byte) error {
	return b.b.Set(key, value, nil)
}

func (b *pebbleBatch) Delete(key []byte) error {
	return b.b.Delete(key, nil)
}

func (b *pebbleBatch) ValueSize() int {
	return int(b.b.Len())
}

func (b *pebbleBatch) Write() error {
	return b.b.Commit(pebble.Sync)
}

func (b *pebbleBatch) Reset() {
	b.b.Close()
	b.b = b.db.NewBatch()
}

func (b *pebbleBatch) Replay(w ethdb.KeyValueWriter) error {
	return nil // Not implemented
}

// pebbleIterator is an iterator
type pebbleIterator struct {
	iter *pebble.Iterator
	err  error
}

func (it *pebbleIterator) Next() bool {
	if it.iter == nil {
		return false
	}
	return it.iter.Next()
}

func (it *pebbleIterator) Error() error {
	if it.err != nil {
		return it.err
	}
	if it.iter == nil {
		return nil
	}
	return it.iter.Error()
}

func (it *pebbleIterator) Key() []byte {
	if it.iter == nil {
		return nil
	}
	return it.iter.Key()
}

func (it *pebbleIterator) Value() []byte {
	if it.iter == nil {
		return nil
	}
	return it.iter.Value()
}

func (it *pebbleIterator) Release() {
	if it.iter != nil {
		it.iter.Close()
	}
}

var errNotSupported = errors.New("not supported")