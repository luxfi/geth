package badgerdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	
	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/badger/v3/options"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
)

const (
	// Ancient store kinds - these are the standard Ethereum ancient data types
	chainFreezerHeaderTable     = "headers"
	chainFreezerHashTable       = "hashes"
	chainFreezerBodiesTable     = "bodies"
	chainFreezerReceiptTable    = "receipts"
	chainFreezerDifficultyTable = "diffs"
)

// BadgerAncientStore implements the ancient store interface using BadgerDB
type BadgerAncientStore struct {
	db     *badger.DB
	path   string
	
	// Metadata tracking
	ancients   atomic.Uint64  // Total number of ancient items
	tail       atomic.Uint64  // First stored item (for pruning)
	
	// Table metadata
	tables map[string]*tableMetadata
	mu     sync.RWMutex
	
	// Write batch for atomic operations
	writeLock sync.Mutex
}

type tableMetadata struct {
	items atomic.Uint64
	size  atomic.Uint64
}

// NewBadgerAncientStore creates a new ancient store using BadgerDB
func NewBadgerAncientStore(path string) (*BadgerAncientStore, error) {
	ancientPath := filepath.Join(path, "ancient")
	
	opts := badger.DefaultOptions(ancientPath)
	opts.SyncWrites = true
	opts.DetectConflicts = false // Ancient data is append-only
	opts.NumVersionsToKeep = 1
	opts.Compression = options.Snappy
	opts.Logger = nil
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open ancient BadgerDB: %w", err)
	}
	
	store := &BadgerAncientStore{
		db:     db,
		path:   ancientPath,
		tables: make(map[string]*tableMetadata),
	}
	
	// Initialize table metadata
	for _, table := range []string{
		chainFreezerHeaderTable,
		chainFreezerHashTable,
		chainFreezerBodiesTable,
		chainFreezerReceiptTable,
		chainFreezerDifficultyTable,
	} {
		store.tables[table] = &tableMetadata{}
	}
	
	// Load metadata from database
	if err := store.loadMetadata(); err != nil {
		db.Close()
		return nil, err
	}
	
	log.Info("BadgerDB ancient store opened", "path", ancientPath)
	return store, nil
}

// loadMetadata loads the ancient store metadata from the database
func (s *BadgerAncientStore) loadMetadata() error {
	return s.db.View(func(txn *badger.Txn) error {
		// Load ancients count
		if item, err := txn.Get([]byte("ancient-count")); err == nil {
			val, err := item.ValueCopy(nil)
			if err == nil && len(val) == 8 {
				s.ancients.Store(binary.BigEndian.Uint64(val))
			}
		}
		
		// Load tail
		if item, err := txn.Get([]byte("ancient-tail")); err == nil {
			val, err := item.ValueCopy(nil)
			if err == nil && len(val) == 8 {
				s.tail.Store(binary.BigEndian.Uint64(val))
			}
		}
		
		// Load table metadata
		for table, meta := range s.tables {
			// Load item count
			if item, err := txn.Get([]byte(fmt.Sprintf("table-%s-count", table))); err == nil {
				val, err := item.ValueCopy(nil)
				if err == nil && len(val) == 8 {
					meta.items.Store(binary.BigEndian.Uint64(val))
				}
			}
			
			// Load size
			if item, err := txn.Get([]byte(fmt.Sprintf("table-%s-size", table))); err == nil {
				val, err := item.ValueCopy(nil)
				if err == nil && len(val) == 8 {
					meta.size.Store(binary.BigEndian.Uint64(val))
				}
			}
		}
		
		return nil
	})
}

// makeKey creates a key for ancient data
func makeAncientKey(kind string, number uint64) []byte {
	key := make([]byte, len(kind)+1+8)
	copy(key, kind)
	key[len(kind)] = ':'
	binary.BigEndian.PutUint64(key[len(kind)+1:], number)
	return key
}

// HasAncient returns whether an ancient binary blob is available
func (s *BadgerAncientStore) HasAncient(kind string, number uint64) (bool, error) {
	var exists bool
	err := s.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(makeAncientKey(kind, number))
		if err == nil {
			exists = true
			return nil
		}
		if errors.Is(err, badger.ErrKeyNotFound) {
			exists = false
			return nil
		}
		return err
	})
	return exists, err
}

// Ancient retrieves an ancient binary blob
func (s *BadgerAncientStore) Ancient(kind string, number uint64) ([]byte, error) {
	var data []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(makeAncientKey(kind, number))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return fmt.Errorf("ancient %s #%d not found", kind, number)
			}
			return err
		}
		
		data, err = item.ValueCopy(nil)
		return err
	})
	return data, err
}

// AncientRange retrieves multiple items in sequence
func (s *BadgerAncientStore) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	var items [][]byte
	var totalBytes uint64
	
	err := s.db.View(func(txn *badger.Txn) error {
		for i := uint64(0); i < count; i++ {
			item, err := txn.Get(makeAncientKey(kind, start+i))
			if err != nil {
				if errors.Is(err, badger.ErrKeyNotFound) {
					break // No more items
				}
				return err
			}
			
			data, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			
			// Check size limit
			if maxBytes > 0 && totalBytes+uint64(len(data)) > maxBytes {
				// Return at least one item
				if len(items) == 0 {
					items = append(items, data)
				}
				break
			}
			
			items = append(items, data)
			totalBytes += uint64(len(data))
		}
		return nil
	})
	
	return items, err
}

// Ancients returns the ancient item numbers
func (s *BadgerAncientStore) Ancients() (uint64, error) {
	return s.ancients.Load(), nil
}

// Tail returns the number of first stored item
func (s *BadgerAncientStore) Tail() (uint64, error) {
	return s.tail.Load(), nil
}

// AncientSize returns the ancient size of the specified category
func (s *BadgerAncientStore) AncientSize(kind string) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if meta, ok := s.tables[kind]; ok {
		return meta.size.Load(), nil
	}
	return 0, fmt.Errorf("unknown ancient table: %s", kind)
}

// ReadAncients runs a read operation while ensuring no writes take place
func (s *BadgerAncientStore) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	// BadgerDB handles read isolation via MVCC transactions
	return fn(s)
}

// ModifyAncients runs a write operation on the ancient store
func (s *BadgerAncientStore) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	
	// Create a write batch wrapper
	batch := &ancientBatch{
		store: s,
		ops:   make([]ancientOp, 0),
	}
	
	// Run the modification function
	if err := fn(batch); err != nil {
		return 0, err
	}
	
	// Apply all operations atomically
	var written int64
	err := s.db.Update(func(txn *badger.Txn) error {
		for _, op := range batch.ops {
			key := makeAncientKey(op.kind, op.number)
			
			if op.delete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			} else {
				if err := txn.Set(key, op.data); err != nil {
					return err
				}
				written += int64(len(op.data))
				
				// Update table metadata
				if meta, ok := s.tables[op.kind]; ok {
					meta.size.Add(uint64(len(op.data)))
				}
			}
		}
		
		// Update metadata
		if batch.newAncients > 0 {
			s.ancients.Store(batch.newAncients)
			val := make([]byte, 8)
			binary.BigEndian.PutUint64(val, batch.newAncients)
			if err := txn.Set([]byte("ancient-count"), val); err != nil {
				return err
			}
		}
		
		if batch.newTail > 0 {
			s.tail.Store(batch.newTail)
			val := make([]byte, 8)
			binary.BigEndian.PutUint64(val, batch.newTail)
			if err := txn.Set([]byte("ancient-tail"), val); err != nil {
				return err
			}
		}
		
		return nil
	})
	
	return written, err
}

// TruncateAncients discards all but the first n ancient data
func (s *BadgerAncientStore) TruncateAncients(n uint64) error {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	
	current := s.ancients.Load()
	if n >= current {
		return nil // Nothing to truncate
	}
	
	return s.db.Update(func(txn *badger.Txn) error {
		// Delete items from n to current
		for _, kind := range []string{
			chainFreezerHeaderTable,
			chainFreezerHashTable,
			chainFreezerBodiesTable,
			chainFreezerReceiptTable,
			chainFreezerDifficultyTable,
		} {
			for i := n; i < current; i++ {
				key := makeAncientKey(kind, i)
				if err := txn.Delete(key); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
					return err
				}
			}
		}
		
		// Update ancients count
		s.ancients.Store(n)
		val := make([]byte, 8)
		binary.BigEndian.PutUint64(val, n)
		return txn.Set([]byte("ancient-count"), val)
	})
}

// TruncateHead discards all but the first n ancient data from the ancient store
func (s *BadgerAncientStore) TruncateHead(n uint64) (uint64, error) {
	oldHead := s.ancients.Load()
	if err := s.TruncateAncients(n); err != nil {
		return 0, err
	}
	return oldHead, nil
}

// TruncateTail discards the first n ancient data from the ancient store
func (s *BadgerAncientStore) TruncateTail(n uint64) (uint64, error) {
	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	
	oldTail := s.tail.Load()
	newTail := oldTail + n
	ancients := s.ancients.Load()
	
	if newTail > ancients {
		return 0, errors.New("truncate tail beyond ancients")
	}
	
	return oldTail, s.db.Update(func(txn *badger.Txn) error {
		// Delete items from oldTail to newTail
		for _, kind := range []string{
			chainFreezerHeaderTable,
			chainFreezerHashTable,
			chainFreezerBodiesTable,
			chainFreezerReceiptTable,
			chainFreezerDifficultyTable,
		} {
			for i := oldTail; i < newTail; i++ {
				key := makeAncientKey(kind, i)
				if err := txn.Delete(key); err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
					return err
				}
			}
		}
		
		// Update tail
		s.tail.Store(newTail)
		val := make([]byte, 8)
		binary.BigEndian.PutUint64(val, newTail)
		return txn.Set([]byte("ancient-tail"), val)
	})
}

// SyncAncient flushes all in-memory ancient store data to disk
func (s *BadgerAncientStore) SyncAncient() error {
	return s.db.Sync()
}

// AncientDatadir returns the ancient datadir
func (s *BadgerAncientStore) AncientDatadir() (string, error) {
	return s.path, nil
}

// Close closes the ancient store
func (s *BadgerAncientStore) Close() error {
	return s.db.Close()
}

// ancientBatch implements ethdb.AncientWriteOp
type ancientBatch struct {
	store       *BadgerAncientStore
	ops         []ancientOp
	newAncients uint64
	newTail     uint64
}

type ancientOp struct {
	kind   string
	number uint64
	data   []byte
	delete bool
}

// Append adds an RLP-encoded item
func (b *ancientBatch) Append(kind string, number uint64, item interface{}) error {
	// For now, we'll just store the raw bytes
	// In a full implementation, this would RLP-encode the item
	return errors.New("RLP encoding not implemented - use AppendRaw")
}

// AppendRaw adds an item without RLP-encoding it
func (b *ancientBatch) AppendRaw(kind string, number uint64, item []byte) error {
	b.ops = append(b.ops, ancientOp{
		kind:   kind,
		number: number,
		data:   item,
	})
	
	// Update ancients count if this is a new item
	if number >= b.store.ancients.Load() {
		b.newAncients = number + 1
	}
	
	return nil
}

// TruncateAncients truncates ancients
func (b *ancientBatch) TruncateAncients(n uint64) error {
	// This is handled differently in ModifyAncients
	b.newAncients = n
	return nil
}

// TruncateTail truncates tail
func (b *ancientBatch) TruncateTail(n uint64) error {
	b.newTail = b.store.tail.Load() + n
	return nil
}