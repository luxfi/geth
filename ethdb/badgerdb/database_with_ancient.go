package badgerdb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/dgraph-io/badger/v3"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/log"
)

// BadgerDatabaseWithAncient implements ethdb.Database with integrated ancient store
type BadgerDatabaseWithAncient struct {
	*BadgerDatabase         // Embed the regular database for current data
	ancient *BadgerAncientStore  // Ancient store for historical data
}

// NewBadgerDatabaseWithAncient creates a database with integrated ancient store
func NewBadgerDatabaseWithAncient(path string, namespace string, readOnly bool, sharedAncient bool) (*BadgerDatabaseWithAncient, error) {
	// Create the main database
	mainDB, err := NewBadgerDatabase(filepath.Join(path, "chaindata"), readOnly, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open main database: %w", err)
	}
	
	// Create or open the ancient store
	ancientPath := filepath.Join(path, "chaindata", "ancient")
	if namespace != "" {
		ancientPath = filepath.Join(path, "chaindata", fmt.Sprintf("ancient-%s", namespace))
	}
	
	// For shared ancient store, open in read-only mode with bypass lock
	var ancient *BadgerAncientStore
	if sharedAncient && readOnly {
		ancient, err = NewSharedBadgerAncientStore(ancientPath)
	} else {
		ancient, err = NewBadgerAncientStore(ancientPath)
	}
	
	if err != nil {
		mainDB.Close()
		return nil, fmt.Errorf("failed to open ancient store: %w", err)
	}
	
	db := &BadgerDatabaseWithAncient{
		BadgerDatabase: mainDB,
		ancient:       ancient,
	}
	
	log.Info("BadgerDB with ancient store initialized",
		"path", path,
		"namespace", namespace,
		"readOnly", readOnly,
		"sharedAncient", sharedAncient)
	
	return db, nil
}

// NewSharedBadgerAncientStore creates a read-only ancient store that can be shared
func NewSharedBadgerAncientStore(path string) (*BadgerAncientStore, error) {
	// First check if the database exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("ancient store path does not exist: %s", path)
	}
	
	// Check if it's a valid BadgerDB directory
	manifestPath := filepath.Join(path, "MANIFEST")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a valid BadgerDB directory (no MANIFEST found): %s", path)
	}
	
	opts := badger.DefaultOptions(path)
	opts.ReadOnly = true
	opts.BypassLockGuard = true // Allow multiple read-only instances
	opts.SyncWrites = false
	opts.DetectConflicts = false
	opts.Logger = nil
	
	// Optimize for read-only access
	opts.MemTableSize = 64 << 20 // 64MB
	opts.ValueLogMaxEntries = 1000000
	opts.BlockCacheSize = 256 << 20 // 256MB
	opts.IndexCacheSize = 256 << 20 // 256MB
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open shared ancient BadgerDB: %w", err)
	}
	
	store := &BadgerAncientStore{
		db:     db,
		path:   path,
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
	
	log.Info("Shared BadgerDB ancient store opened", "path", path)
	return store, nil
}

// Ancient store methods - delegate to the ancient store

func (db *BadgerDatabaseWithAncient) HasAncient(kind string, number uint64) (bool, error) {
	if db.ancient == nil {
		return false, nil
	}
	return db.ancient.HasAncient(kind, number)
}

func (db *BadgerDatabaseWithAncient) Ancient(kind string, number uint64) ([]byte, error) {
	if db.ancient == nil {
		return nil, errors.New("ancient store not available")
	}
	return db.ancient.Ancient(kind, number)
}

func (db *BadgerDatabaseWithAncient) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	if db.ancient == nil {
		return nil, errors.New("ancient store not available")
	}
	return db.ancient.AncientRange(kind, start, count, maxBytes)
}

func (db *BadgerDatabaseWithAncient) Ancients() (uint64, error) {
	if db.ancient == nil {
		return 0, nil
	}
	return db.ancient.Ancients()
}

func (db *BadgerDatabaseWithAncient) Tail() (uint64, error) {
	if db.ancient == nil {
		return 0, nil
	}
	return db.ancient.Tail()
}

func (db *BadgerDatabaseWithAncient) AncientSize(kind string) (uint64, error) {
	if db.ancient == nil {
		return 0, nil
	}
	return db.ancient.AncientSize(kind)
}

func (db *BadgerDatabaseWithAncient) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	if db.ancient == nil {
		return errors.New("ancient store not available")
	}
	return db.ancient.ReadAncients(fn)
}

func (db *BadgerDatabaseWithAncient) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	if db.ancient == nil {
		return 0, errors.New("ancient store not available")
	}
	return db.ancient.ModifyAncients(fn)
}

func (db *BadgerDatabaseWithAncient) TruncateAncients(n uint64) error {
	if db.ancient == nil {
		return errors.New("ancient store not available")
	}
	return db.ancient.TruncateAncients(n)
}

func (db *BadgerDatabaseWithAncient) TruncateHead(n uint64) (uint64, error) {
	if db.ancient == nil {
		return 0, errors.New("ancient store not available")
	}
	return db.ancient.TruncateHead(n)
}

func (db *BadgerDatabaseWithAncient) TruncateTail(n uint64) (uint64, error) {
	if db.ancient == nil {
		return 0, errors.New("ancient store not available")
	}
	return db.ancient.TruncateTail(n)
}

func (db *BadgerDatabaseWithAncient) SyncAncient() error {
	if db.ancient == nil {
		return nil
	}
	return db.ancient.SyncAncient()
}

func (db *BadgerDatabaseWithAncient) AncientDatadir() (string, error) {
	if db.ancient == nil {
		return "", errors.New("ancient store not available")
	}
	return db.ancient.AncientDatadir()
}

func (db *BadgerDatabaseWithAncient) MigrateTable(s string, f func([]byte) ([]byte, error)) error {
	// Migration from current to ancient store could be implemented here
	return errors.New("table migration not implemented")
}

// Close closes both the main database and ancient store
func (db *BadgerDatabaseWithAncient) Close() error {
	var errs []error
	
	// Close main database
	if err := db.BadgerDatabase.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close main database: %w", err))
	}
	
	// Close ancient store
	if db.ancient != nil {
		if err := db.ancient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close ancient store: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}