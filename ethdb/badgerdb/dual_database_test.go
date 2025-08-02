package badgerdb

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewDualBadgerDatabase tests dual database creation
func TestNewDualBadgerDatabase(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	tests := []struct {
		name        string
		config      DualDatabaseConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid with archive",
			config: DualDatabaseConfig{
				ArchivePath:    t.TempDir(),
				CurrentPath:    t.TempDir(),
				FinalityHeight: 32,
				ArchiveShared:  false,
			},
			expectError: false,
		},
		{
			name: "Valid without archive",
			config: DualDatabaseConfig{
				ArchivePath:    "",
				CurrentPath:    t.TempDir(),
				FinalityHeight: 32,
			},
			expectError: false,
		},
		{
			name: "Missing current path",
			config: DualDatabaseConfig{
				ArchivePath: t.TempDir(),
				CurrentPath: "",
			},
			expectError: true,
			errorMsg:    "current database path is required",
		},
		{
			name: "Invalid archive path",
			config: DualDatabaseConfig{
				ArchivePath: "/invalid/path/that/does/not/exist",
				CurrentPath: t.TempDir(),
			},
			expectError: true,
			errorMsg:    "failed to open archive database",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewDualBadgerDatabase(tt.config)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, db)
			defer db.Close()
			
			// Verify configuration
			assert.Equal(t, tt.config.ArchivePath != "", db.hasArchive)
			assert.Equal(t, tt.config.FinalityHeight, db.finalityHeight)
		})
	}
}

// TestDualDatabaseReadWrite tests read/write routing
func TestDualDatabaseReadWrite(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	// Create dual database with archive
	config := DualDatabaseConfig{
		ArchivePath:    t.TempDir(),
		CurrentPath:    t.TempDir(),
		FinalityHeight: 100,
		ArchiveShared:  false,
	}
	
	// First, populate archive with some data
	archiveDB, err := NewBadgerDatabase(config.ArchivePath, false, false)
	require.NoError(t, err)
	
	// Add finalized data to archive
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := archiveDB.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	archiveDB.Close()
	
	// Now create dual database
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Test reading from archive (finalized data)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("block-%d", i)
		value, err := ddb.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("value-%d", i), string(value))
	}
	
	// Test writing to current (new data)
	for i := 100; i < 150; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	
	// Test reading recent data from current
	for i := 100; i < 150; i++ {
		key := fmt.Sprintf("block-%d", i)
		value, err := ddb.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("value-%d", i), string(value))
	}
	
	// Verify metrics
	assert.Greater(t, ddb.archiveReads.Load(), uint64(0))
	assert.Greater(t, ddb.currentReads.Load(), uint64(0))
	assert.Greater(t, ddb.currentWrites.Load(), uint64(0))
}

// TestDualDatabaseHas tests Has operation across databases
func TestDualDatabaseHas(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath:    t.TempDir(),
		CurrentPath:    t.TempDir(),
		FinalityHeight: 50,
	}
	
	// Setup archive with data
	archiveDB, err := NewBadgerDatabase(config.ArchivePath, false, false)
	require.NoError(t, err)
	err = archiveDB.Put([]byte("archive-key"), []byte("archive-value"))
	assert.NoError(t, err)
	archiveDB.Close()
	
	// Create dual database
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Add current data
	err = ddb.Put([]byte("current-key"), []byte("current-value"))
	assert.NoError(t, err)
	
	// Test Has
	tests := []struct {
		key      string
		expected bool
	}{
		{"archive-key", true},
		{"current-key", true},
		{"missing-key", false},
	}
	
	for _, tt := range tests {
		has, err := ddb.Has([]byte(tt.key))
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, has, "key: %s", tt.key)
	}
}

// TestDualDatabaseDelete tests delete operations
func TestDualDatabaseDelete(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Add and delete data
	key := []byte("delete-key")
	value := []byte("delete-value")
	
	err = ddb.Put(key, value)
	assert.NoError(t, err)
	
	has, err := ddb.Has(key)
	assert.NoError(t, err)
	assert.True(t, has)
	
	err = ddb.Delete(key)
	assert.NoError(t, err)
	
	has, err = ddb.Has(key)
	assert.NoError(t, err)
	assert.False(t, has)
}

// TestDualDatabaseBatch tests batch operations
func TestDualDatabaseBatch(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Create batch
	batch := ddb.NewBatch()
	
	// Add operations to batch
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("batch-key-%d", i)
		value := fmt.Sprintf("batch-value-%d", i)
		err := batch.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	
	// Write batch
	err = batch.Write()
	assert.NoError(t, err)
	
	// Verify all entries
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("batch-key-%d", i)
		value, err := ddb.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("batch-value-%d", i), string(value))
	}
}

// TestDualDatabaseIterator tests iterator functionality
func TestDualDatabaseIterator(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath:    t.TempDir(),
		CurrentPath:    t.TempDir(),
		FinalityHeight: 50,
	}
	
	// Setup archive data
	archiveDB, err := NewBadgerDatabase(config.ArchivePath, false, false)
	require.NoError(t, err)
	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("key-%03d", i)
		value := fmt.Sprintf("archive-value-%d", i)
		err := archiveDB.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	archiveDB.Close()
	
	// Create dual database
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Add current data
	for i := 50; i < 100; i++ {
		key := fmt.Sprintf("key-%03d", i)
		value := fmt.Sprintf("current-value-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	
	// Test full iteration
	iter := ddb.NewIterator(nil, nil)
	defer iter.Release()
	
	count := 0
	for iter.Next() {
		count++
		key := string(iter.Key())
		value := string(iter.Value())
		
		// Verify we get data from both databases
		if count <= 50 {
			assert.Contains(t, value, "archive-value")
		} else {
			assert.Contains(t, value, "current-value")
		}
		_ = key
	}
	assert.NoError(t, iter.Error())
	assert.Equal(t, 100, count)
}

// TestDualDatabaseStat tests statistics collection
func TestDualDatabaseStat(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath: t.TempDir(),
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Perform some operations
	for i := 0; i < 10; i++ {
		err := ddb.Put([]byte(fmt.Sprintf("key-%d", i)), []byte("value"))
		assert.NoError(t, err)
		_, err = ddb.Get([]byte(fmt.Sprintf("key-%d", i)))
		assert.NoError(t, err)
	}
	
	// Get stats
	stat, err := ddb.Stat()
	assert.NoError(t, err)
	assert.Contains(t, stat, "Archive Database")
	assert.Contains(t, stat, "Current Database")
	assert.Contains(t, stat, "Reads")
	assert.Contains(t, stat, "Writes")
}

// TestDualDatabaseCompact tests compaction
func TestDualDatabaseCompact(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath: t.TempDir(),
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Add data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("compact-key-%d", i)
		value := fmt.Sprintf("compact-value-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	
	// Compact
	err = ddb.Compact(nil, nil)
	assert.NoError(t, err)
	
	// Verify data still accessible
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("compact-key-%d", i)
		has, err := ddb.Has([]byte(key))
		assert.NoError(t, err)
		assert.True(t, has)
	}
}

// TestDualDatabaseNoArchive tests behavior without archive
func TestDualDatabaseNoArchive(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		CurrentPath:    t.TempDir(),
		FinalityHeight: 100,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Should work normally with just current database
	testData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	
	// Write data
	for k, v := range testData {
		err := ddb.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}
	
	// Read data
	for k, expectedV := range testData {
		v, err := ddb.Get([]byte(k))
		assert.NoError(t, err)
		assert.Equal(t, expectedV, string(v))
	}
	
	// Iterator should work
	iter := ddb.NewIterator(nil, nil)
	count := 0
	for iter.Next() {
		count++
	}
	iter.Release()
	assert.Equal(t, len(testData), count)
}

// TestDualDatabaseConcurrency tests concurrent access
func TestDualDatabaseConcurrency(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath: t.TempDir(),
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOps := 100
	
	// Writers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("concurrent-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				err := ddb.Put([]byte(key), []byte(value))
				assert.NoError(t, err)
			}
		}(i)
	}
	
	// Readers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Let writers get ahead
			time.Sleep(10 * time.Millisecond)
			
			for j := 0; j < numOps; j++ {
				key := fmt.Sprintf("concurrent-%d-%d", id, j)
				// May or may not exist yet
				ddb.Get([]byte(key))
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify all data written
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numOps; j++ {
			key := fmt.Sprintf("concurrent-%d-%d", i, j)
			has, err := ddb.Has([]byte(key))
			assert.NoError(t, err)
			assert.True(t, has)
		}
	}
}

// TestDualDatabaseSnapshot tests snapshot behavior
func TestDualDatabaseSnapshot(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Snapshots not supported
	_, err = ddb.NewSnapshot()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "snapshots not supported")
}

// TestDualDatabaseClose tests proper cleanup
func TestDualDatabaseClose(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath: t.TempDir(),
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	
	// Close should succeed
	err = ddb.Close()
	assert.NoError(t, err)
	
	// Operations after close should fail
	_, err = ddb.Get([]byte("test"))
	assert.Error(t, err)
	
	err = ddb.Put([]byte("test"), []byte("value"))
	assert.Error(t, err)
}

// TestDualIterator tests the dual iterator implementation
func TestDualIterator(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	// Create test iterators
	archiveData := map[string]string{
		"a-1": "archive-1",
		"a-2": "archive-2",
		"a-3": "archive-3",
	}
	
	currentData := map[string]string{
		"c-1": "current-1",
		"c-2": "current-2",
		"c-3": "current-3",
	}
	
	// Setup databases
	archiveDir := t.TempDir()
	currentDir := t.TempDir()
	
	archiveDB, err := NewBadgerDatabase(archiveDir, false, false)
	require.NoError(t, err)
	for k, v := range archiveData {
		err := archiveDB.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}
	archiveDB.Close()
	
	currentDB, err := NewBadgerDatabase(currentDir, false, false)
	require.NoError(t, err)
	for k, v := range currentData {
		err := currentDB.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}
	currentDB.Close()
	
	// Create dual database
	config := DualDatabaseConfig{
		ArchivePath: archiveDir,
		CurrentPath: currentDir,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Test iteration
	iter := ddb.NewIterator(nil, nil)
	defer iter.Release()
	
	collected := make(map[string]string)
	for iter.Next() {
		collected[string(iter.Key())] = string(iter.Value())
	}
	
	// Should have all data from both databases
	assert.Equal(t, len(archiveData)+len(currentData), len(collected))
	
	for k, v := range archiveData {
		assert.Equal(t, v, collected[k])
	}
	for k, v := range currentData {
		assert.Equal(t, v, collected[k])
	}
}

// TestReadBlock tests block reading with routing
func TestReadBlock(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath:    t.TempDir(),
		CurrentPath:    t.TempDir(),
		FinalityHeight: 100,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Test reading finalized block (should check archive first)
	hash := common.HexToHash("0x1234")
	block := ddb.ReadBlock(hash, 50)
	assert.Nil(t, block) // No data, but should have checked archive
	assert.Greater(t, ddb.archiveReads.Load(), uint64(0))
	
	// Test reading recent block (should go to current)
	prevCurrentReads := ddb.currentReads.Load()
	block = ddb.ReadBlock(hash, 150)
	assert.Nil(t, block) // No data, but should have checked current
	assert.Greater(t, ddb.currentReads.Load(), prevCurrentReads)
}

// TestDualDatabaseSharedArchive tests shared archive access
func TestDualDatabaseSharedArchive(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	
	// Create and populate archive
	archiveDB, err := NewBadgerDatabase(archiveDir, false, false)
	require.NoError(t, err)
	
	testData := map[string]string{
		"shared-1": "value-1",
		"shared-2": "value-2",
		"shared-3": "value-3",
	}
	
	for k, v := range testData {
		err := archiveDB.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}
	archiveDB.Close()
	
	// Open multiple dual databases with shared archive
	var ddbs []*DualBadgerDatabase
	for i := 0; i < 3; i++ {
		config := DualDatabaseConfig{
			ArchivePath:   archiveDir,
			CurrentPath:   t.TempDir(), // Each gets its own current
			ArchiveShared: true,
		}
		
		ddb, err := NewDualBadgerDatabase(config)
		require.NoError(t, err)
		ddbs = append(ddbs, ddb)
	}
	
	// All should be able to read archive data
	for i, ddb := range ddbs {
		for k, expectedV := range testData {
			v, err := ddb.Get([]byte(k))
			assert.NoError(t, err, "instance %d, key %s", i, k)
			assert.Equal(t, expectedV, string(v))
		}
	}
	
	// Each can write to its own current
	for i, ddb := range ddbs {
		key := fmt.Sprintf("current-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}
	
	// Clean up
	for _, ddb := range ddbs {
		ddb.Close()
	}
}

// TestArchiver tests the archiver functionality
func TestArchiver(t *testing.T) {
	t.Skip("Dual database is obsolete with ancient store architecture")
	config := DualDatabaseConfig{
		ArchivePath:    t.TempDir(),
		CurrentPath:    t.TempDir(),
		FinalityHeight: 5,
		ArchiveShared:  false,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Create archiver with short interval for testing
	archiver := NewArchiver(ddb, 5, 100*time.Millisecond)
	archiver.Start()
	defer archiver.Stop()
	
	// Write some blocks
	for i := uint64(0); i < 20; i++ {
		key := append([]byte("block-"), encodeBlockNumber(i)...)
		value := []byte(fmt.Sprintf("block-%d-data", i))
		
		err := ddb.Put(key, value)
		require.NoError(t, err)
		
		// Simulate block height tracking
		heightKey := []byte("LastBlock")
		err = ddb.Put(heightKey, encodeBlockNumber(i))
		require.NoError(t, err)
	}
	
	// Wait for archiving to happen
	time.Sleep(500 * time.Millisecond)
	
	// Check that archiving happened
	assert.Greater(t, archiver.totalArchived.Load(), uint64(0))
	
	// Verify old blocks can still be read
	for i := uint64(0); i < 10; i++ {
		key := append([]byte("block-"), encodeBlockNumber(i)...)
		value, err := ddb.Get(key)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("block-%d-data", i), string(value))
	}
}

// BenchmarkDualDatabaseGet benchmarks Get operations
func BenchmarkDualDatabaseGet(b *testing.B) {
	config := DualDatabaseConfig{
		ArchivePath: b.TempDir(),
		CurrentPath: b.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(b, err)
	defer ddb.Close()
	
	// Populate with test data
	testKey := []byte("benchmark-key")
	testValue := []byte("benchmark-value")
	err = ddb.Put(testKey, testValue)
	require.NoError(b, err)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := ddb.Get(testKey)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDualDatabasePut benchmarks Put operations
func BenchmarkDualDatabasePut(b *testing.B) {
	config := DualDatabaseConfig{
		CurrentPath: b.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(b, err)
	defer ddb.Close()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-key-%d", i)
		value := fmt.Sprintf("bench-value-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestIDEInterface ensures DualBadgerDatabase implements ethdb.Database
func TestIDEInterface(t *testing.T) {
	config := DualDatabaseConfig{
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// This will fail to compile if DualBadgerDatabase doesn't implement ethdb.Database
	var _ ethdb.Database = ddb
}