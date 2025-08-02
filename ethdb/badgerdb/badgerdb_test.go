package badgerdb

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBadgerDB runs the standard ethdb test suite
func TestBadgerDB(t *testing.T) {
	t.Run("DatabaseSuite", func(t *testing.T) {
		dbtest.TestDatabaseSuite(t, func() ethdb.KeyValueStore {
			dir := t.TempDir()
			db, err := NewBadgerDatabase(dir, false, false)
			if err != nil {
				t.Fatal(err)
			}
			return db
		})
	})
}

// TestNewBadgerDatabase tests database creation
func TestNewBadgerDatabase(t *testing.T) {
	tests := []struct {
		name        string
		readOnly    bool
		bypassLock  bool
		expectError bool
	}{
		{"ReadWrite", false, false, false},
		{"ReadOnly", true, false, false},
		{"ReadOnlyBypass", true, true, false},
		{"InvalidPath", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dir string
			if tt.expectError {
				dir = "/invalid/path/that/does/not/exist"
			} else {
				dir = t.TempDir()
				
				// For read-only tests, create a database first
				if tt.readOnly {
					initDB, err := NewBadgerDatabase(dir, false, false)
					require.NoError(t, err)
					err = initDB.Put([]byte("init"), []byte("data"))
					require.NoError(t, err)
					initDB.Close()
				}
			}

			db, err := NewBadgerDatabase(dir, tt.readOnly, tt.bypassLock)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, db)
			defer db.Close()

			// Test basic operations
			if !tt.readOnly {
				testKey := []byte("test-key")
				testValue := []byte("test-value")
				
				err = db.Put(testKey, testValue)
				assert.NoError(t, err)
				
				value, err := db.Get(testKey)
				assert.NoError(t, err)
				assert.Equal(t, testValue, value)
			}
		})
	}
}

// TestBadgerDatabaseConcurrency tests concurrent access
func TestBadgerDatabaseConcurrency(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Test concurrent writes
	var wg sync.WaitGroup
	numGoroutines := 10
	numWrites := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numWrites; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				value := fmt.Sprintf("value-%d-%d", id, j)
				err := db.Put([]byte(key), []byte(value))
				assert.NoError(t, err)
			}
		}(i)
	}
	wg.Wait()

	// Verify all writes
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < numWrites; j++ {
			key := fmt.Sprintf("key-%d-%d", i, j)
			expectedValue := fmt.Sprintf("value-%d-%d", i, j)
			value, err := db.Get([]byte(key))
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, string(value))
		}
	}
}

// TestBadgerDatabaseBatch tests batch operations
func TestBadgerDatabaseBatch(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Test batch writes
	batch := db.NewBatch()
	numEntries := 1000

	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("batch-key-%d", i)
		value := fmt.Sprintf("batch-value-%d", i)
		err := batch.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}

	// Write batch
	err = batch.Write()
	assert.NoError(t, err)

	// Verify all entries
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("batch-key-%d", i)
		expectedValue := fmt.Sprintf("batch-value-%d", i)
		value, err := db.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, string(value))
	}

	// Test batch reset
	batch.Reset()
	assert.Equal(t, 0, batch.ValueSize())
}

// TestBadgerDatabaseIterator tests iterator functionality
func TestBadgerDatabaseIterator(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Insert test data
	testData := map[string]string{
		"key-1": "value-1",
		"key-2": "value-2",
		"key-3": "value-3",
		"key-4": "value-4",
		"key-5": "value-5",
	}

	for k, v := range testData {
		err := db.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}

	// Test NewIterator
	t.Run("FullIterator", func(t *testing.T) {
		iter := db.NewIterator(nil, nil)
		defer iter.Release()

		count := 0
		for iter.Next() {
			count++
			key := string(iter.Key())
			value := string(iter.Value())
			expectedValue, ok := testData[key]
			assert.True(t, ok)
			assert.Equal(t, expectedValue, value)
		}
		assert.Equal(t, len(testData), count)
	})

	// Test range iterator
	t.Run("RangeIterator", func(t *testing.T) {
		start := []byte("key-2")
		limit := []byte("key-4")
		iter := db.NewIterator(nil, start)
		defer iter.Release()

		var keys []string
		for iter.Next() {
			key := iter.Key()
			if bytes.Compare(key, limit) >= 0 {
				break
			}
			keys = append(keys, string(key))
		}
		assert.Equal(t, []string{"key-2", "key-3"}, keys)
	})
}

// TestBadgerDatabaseSnapshot tests snapshot functionality
func TestBadgerDatabaseSnapshot(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Insert initial data
	err = db.Put([]byte("key1"), []byte("value1"))
	assert.NoError(t, err)

	// Take snapshot - BadgerDB now supports snapshots via read-only transactions
	snapshot, err := db.NewSnapshot()
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	
	// Test reading from snapshot
	value, err := snapshot.Get([]byte("key1"))
	assert.NoError(t, err)
	assert.Equal(t, "value1", string(value))
	
	// Test that snapshot is isolated from new writes
	err = db.Put([]byte("key2"), []byte("value2"))
	assert.NoError(t, err)
	
	// Snapshot should not see the new key
	_, err = snapshot.Get([]byte("key2"))
	assert.Error(t, err)
	
	// Release snapshot
	snapshot.Release()
}

// TestBadgerDatabaseMetrics tests metrics collection
func TestBadgerDatabaseMetrics(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Perform some operations
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("metric-key-%d", i)
		value := fmt.Sprintf("metric-value-%d", i)
		err := db.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
		
		_, err = db.Get([]byte(key))
		assert.NoError(t, err)
	}

	// Check stats
	stats, err := db.Stat()
	assert.NoError(t, err)
	assert.NotEmpty(t, stats)
	// BadgerDB should provide some statistics
}

// TestBadgerDatabaseCompact tests compaction
func TestBadgerDatabaseCompact(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Insert and delete data to create garbage
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("compact-key-%d", i)
		value := fmt.Sprintf("compact-value-%d", i)
		err := db.Put([]byte(key), []byte(value))
		assert.NoError(t, err)
	}

	// Delete half the keys
	for i := 0; i < 500; i++ {
		key := fmt.Sprintf("compact-key-%d", i)
		err := db.Delete([]byte(key))
		assert.NoError(t, err)
	}

	// Compact
	err = db.Compact(nil, nil)
	assert.NoError(t, err)

	// Verify remaining keys
	for i := 500; i < 1000; i++ {
		key := fmt.Sprintf("compact-key-%d", i)
		value, err := db.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("compact-value-%d", i), string(value))
	}
}

// TestBadgerDatabaseClose tests proper cleanup
func TestBadgerDatabaseClose(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)

	// Insert data
	err = db.Put([]byte("test"), []byte("value"))
	assert.NoError(t, err)

	// Close database
	err = db.Close()
	assert.NoError(t, err)

	// Operations after close should fail
	err = db.Put([]byte("test2"), []byte("value2"))
	assert.Error(t, err)

	_, err = db.Get([]byte("test"))
	assert.Error(t, err)
}

// TestBadgerDatabaseReadOnlyShared tests shared read-only access
func TestBadgerDatabaseReadOnlyShared(t *testing.T) {
	dir := t.TempDir()
	
	// Create and populate database
	db1, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	
	testData := map[string]string{
		"shared-1": "value-1",
		"shared-2": "value-2",
		"shared-3": "value-3",
	}
	
	for k, v := range testData {
		err := db1.Put([]byte(k), []byte(v))
		assert.NoError(t, err)
	}
	db1.Close()

	// Open multiple read-only instances with bypass lock
	var dbs []ethdb.Database
	for i := 0; i < 3; i++ {
		db, err := NewBadgerDatabase(dir, true, true)
		require.NoError(t, err)
		dbs = append(dbs, db)
	}

	// Verify all can read the same data
	for _, db := range dbs {
		for k, expectedValue := range testData {
			value, err := db.Get([]byte(k))
			assert.NoError(t, err)
			assert.Equal(t, expectedValue, string(value))
		}
	}

	// Clean up
	for _, db := range dbs {
		db.Close()
	}
}

// TestBadgerDatabaseLargeValues tests handling of large values
func TestBadgerDatabaseLargeValues(t *testing.T) {
	dir := t.TempDir()
	db, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	defer db.Close()

	// Test various sizes
	sizes := []int{
		1024,        // 1KB
		1024 * 100,  // 100KB
		1024 * 1024, // 1MB
	}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Size_%d", size), func(t *testing.T) {
			key := fmt.Sprintf("large-key-%d", size)
			value := make([]byte, size)
			for i := range value {
				value[i] = byte(i % 256)
			}

			err := db.Put([]byte(key), value)
			assert.NoError(t, err)

			retrieved, err := db.Get([]byte(key))
			assert.NoError(t, err)
			assert.Equal(t, value, retrieved)
		})
	}
}