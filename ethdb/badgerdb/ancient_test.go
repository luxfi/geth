package badgerdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	
	"github.com/luxfi/geth/ethdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to append a complete ancient block
func appendAncientBlock(op ethdb.AncientWriteOp, number uint64, hash, header, body, receipts, td []byte) error {
	if err := op.AppendRaw(chainFreezerHashTable, number, hash); err != nil {
		return err
	}
	if err := op.AppendRaw(chainFreezerHeaderTable, number, header); err != nil {
		return err
	}
	if err := op.AppendRaw(chainFreezerBodiesTable, number, body); err != nil {
		return err
	}
	if err := op.AppendRaw(chainFreezerReceiptTable, number, receipts); err != nil {
		return err
	}
	if err := op.AppendRaw(chainFreezerDifficultyTable, number, td); err != nil {
		return err
	}
	return nil
}

// TestBadgerAncientStore tests basic ancient store functionality
func TestBadgerAncientStore(t *testing.T) {
	dir := t.TempDir()
	
	store, err := NewBadgerAncientStore(dir)
	require.NoError(t, err)
	defer store.Close()
	
	// Test initial state
	ancients, err := store.Ancients()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), ancients)
	
	tail, err := store.Tail()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), tail)
	
	// Test writing ancient data
	written, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 10; i++ {
			hash := make([]byte, 32)
			binary.BigEndian.PutUint64(hash, i)
			
			header := []byte(fmt.Sprintf("header-%d", i))
			body := []byte(fmt.Sprintf("body-%d", i))
			receipts := []byte(fmt.Sprintf("receipts-%d", i))
			td := []byte(fmt.Sprintf("td-%d", i))
			
			// Append each type of data separately
			err := op.AppendRaw(chainFreezerHashTable, i, hash)
			assert.NoError(t, err)
			err = op.AppendRaw(chainFreezerHeaderTable, i, header)
			assert.NoError(t, err)
			err = op.AppendRaw(chainFreezerBodiesTable, i, body)
			assert.NoError(t, err)
			err = op.AppendRaw(chainFreezerReceiptTable, i, receipts)
			assert.NoError(t, err)
			err = op.AppendRaw(chainFreezerDifficultyTable, i, td)
			assert.NoError(t, err)
		}
		return nil
	})
	
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))
	
	// Verify data was written
	ancients, err = store.Ancients()
	assert.NoError(t, err)
	assert.Equal(t, uint64(10), ancients)
	
	// Test reading ancient data
	for i := uint64(0); i < 10; i++ {
		// Check HasAncient
		has, err := store.HasAncient(chainFreezerHeaderTable, i)
		assert.NoError(t, err)
		assert.True(t, has)
		
		// Read header
		header, err := store.Ancient(chainFreezerHeaderTable, i)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("header-%d", i), string(header))
		
		// Read body
		body, err := store.Ancient(chainFreezerBodiesTable, i)
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("body-%d", i), string(body))
	}
	
	// Test AncientRange
	headers, err := store.AncientRange(chainFreezerHeaderTable, 2, 5, 0)
	assert.NoError(t, err)
	assert.Len(t, headers, 5)
	for i := 0; i < 5; i++ {
		assert.Equal(t, fmt.Sprintf("header-%d", i+2), string(headers[i]))
	}
	
	// Test size limits
	headers, err = store.AncientRange(chainFreezerHeaderTable, 0, 10, 50)
	assert.NoError(t, err)
	assert.Greater(t, len(headers), 0)
	assert.Less(t, len(headers), 10) // Should stop due to size limit
}

// TestBadgerAncientTruncation tests truncation functionality
func TestBadgerAncientTruncation(t *testing.T) {
	dir := t.TempDir()
	
	store, err := NewBadgerAncientStore(dir)
	require.NoError(t, err)
	defer store.Close()
	
	// Write 20 ancient items
	_, err = store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 20; i++ {
			hash := make([]byte, 32)
			binary.BigEndian.PutUint64(hash, i)
			
			err := appendAncientBlock(op, i, hash,
				[]byte(fmt.Sprintf("header-%d", i)),
				[]byte(fmt.Sprintf("body-%d", i)),
				[]byte(fmt.Sprintf("receipts-%d", i)),
				[]byte(fmt.Sprintf("td-%d", i)))
			assert.NoError(t, err)
		}
		return nil
	})
	require.NoError(t, err)
	
	// Test TruncateHead
	oldHead, err := store.TruncateHead(15)
	assert.NoError(t, err)
	assert.Equal(t, uint64(20), oldHead)
	
	ancients, err := store.Ancients()
	assert.NoError(t, err)
	assert.Equal(t, uint64(15), ancients)
	
	// Verify items 15-19 are gone
	for i := uint64(15); i < 20; i++ {
		has, err := store.HasAncient(chainFreezerHeaderTable, i)
		assert.NoError(t, err)
		assert.False(t, has)
	}
	
	// Test TruncateTail
	oldTail, err := store.TruncateTail(5)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), oldTail)
	
	tail, err := store.Tail()
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), tail)
	
	// Verify items 0-4 are gone
	for i := uint64(0); i < 5; i++ {
		has, err := store.HasAncient(chainFreezerHeaderTable, i)
		assert.NoError(t, err)
		assert.False(t, has)
	}
	
	// Items 5-14 should still exist
	for i := uint64(5); i < 15; i++ {
		has, err := store.HasAncient(chainFreezerHeaderTable, i)
		assert.NoError(t, err)
		assert.True(t, has)
	}
}

// TestSharedBadgerAncientStore tests multiple read-only instances
func TestSharedBadgerAncientStore(t *testing.T) {
	dir := t.TempDir()
	
	// First, create and populate an ancient store
	store, err := NewBadgerAncientStore(dir)
	require.NoError(t, err)
	
	// Write test data
	_, err = store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < 100; i++ {
			hash := make([]byte, 32)
			binary.BigEndian.PutUint64(hash, i)
			
			err := appendAncientBlock(op, i, hash,
				[]byte(fmt.Sprintf("header-%d", i)),
				[]byte(fmt.Sprintf("body-%d", i)),
				[]byte(fmt.Sprintf("receipts-%d", i)),
				[]byte(fmt.Sprintf("td-%d", i)))
			require.NoError(t, err)
		}
		return nil
	})
	require.NoError(t, err)
	
	// Ensure all data is synced before closing
	err = store.SyncAncient()
	require.NoError(t, err)
	
	err = store.Close()
	require.NoError(t, err)
	
	// Longer delay to ensure BadgerDB has fully written manifest files
	time.Sleep(500 * time.Millisecond)
	
	// The ancient store creates an "ancient" subdirectory
	ancientDir := filepath.Join(dir, "ancient")
	
	// Debug: List files in the ancient directory
	files, err := os.ReadDir(ancientDir)
	require.NoError(t, err)
	t.Logf("Files in ancient directory %s:", ancientDir)
	for _, f := range files {
		t.Logf("  - %s", f.Name())
	}
	
	// Now open multiple read-only instances
	numReaders := 5
	readers := make([]*BadgerAncientStore, numReaders)
	
	for i := 0; i < numReaders; i++ {
		reader, err := NewSharedBadgerAncientStore(ancientDir)
		require.NoError(t, err)
		readers[i] = reader
	}
	
	// Test concurrent reads from all instances
	var wg sync.WaitGroup
	for i, reader := range readers {
		wg.Add(1)
		go func(id int, r *BadgerAncientStore) {
			defer wg.Done()
			
			// Each reader reads different blocks
			start := uint64(id * 20)
			end := start + 20
			
			for j := start; j < end && j < 100; j++ {
				// Read header
				header, err := r.Ancient(chainFreezerHeaderTable, j)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("header-%d", j), string(header))
				
				// Read body
				body, err := r.Ancient(chainFreezerBodiesTable, j)
				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("body-%d", j), string(body))
			}
			
			// Test range reads
			headers, err := r.AncientRange(chainFreezerHeaderTable, start, 10, 0)
			assert.NoError(t, err)
			assert.Len(t, headers, 10)
		}(i, reader)
	}
	
	wg.Wait()
	
	// Verify all readers see the same metadata
	for _, reader := range readers {
		ancients, err := reader.Ancients()
		assert.NoError(t, err)
		assert.Equal(t, uint64(100), ancients)
		
		tail, err := reader.Tail()
		assert.NoError(t, err)
		assert.Equal(t, uint64(0), tail)
	}
	
	// Clean up
	for _, reader := range readers {
		reader.Close()
	}
}

// TestBadgerDatabaseWithAncient tests the integrated database
func TestBadgerDatabaseWithAncient(t *testing.T) {
	dir := t.TempDir()
	
	// Create database with ancient store
	db, err := NewBadgerDatabaseWithAncient(dir, "", false, false)
	require.NoError(t, err)
	defer db.Close()
	
	// Test regular database operations
	testKey := []byte("test-key")
	testValue := []byte("test-value")
	
	err = db.Put(testKey, testValue)
	assert.NoError(t, err)
	
	value, err := db.Get(testKey)
	assert.NoError(t, err)
	assert.Equal(t, testValue, value)
	
	// Test ancient operations
	written, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		return appendAncientBlock(op, 0,
			[]byte("hash"),
			[]byte("header"),
			[]byte("body"),
			[]byte("receipts"),
			[]byte("td"))
	})
	assert.NoError(t, err)
	assert.Greater(t, written, int64(0))
	
	// Verify ancient data
	has, err := db.HasAncient(chainFreezerHeaderTable, 0)
	assert.NoError(t, err)
	assert.True(t, has)
	
	header, err := db.Ancient(chainFreezerHeaderTable, 0)
	assert.NoError(t, err)
	assert.Equal(t, []byte("header"), header)
}

// BenchmarkAncientWrites benchmarks ancient write performance
func BenchmarkAncientWrites(b *testing.B) {
	dir := b.TempDir()
	store, err := NewBadgerAncientStore(dir)
	require.NoError(b, err)
	defer store.Close()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			hash := make([]byte, 32)
			binary.BigEndian.PutUint64(hash, uint64(i))
			
			return appendAncientBlock(op, uint64(i), hash,
				[]byte(fmt.Sprintf("header-%d", i)),
				[]byte(fmt.Sprintf("body-%d", i)),
				[]byte(fmt.Sprintf("receipts-%d", i)),
				[]byte(fmt.Sprintf("td-%d", i)))
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAncientReads benchmarks ancient read performance
func BenchmarkAncientReads(b *testing.B) {
	dir := b.TempDir()
	store, err := NewBadgerAncientStore(dir)
	require.NoError(b, err)
	defer store.Close()
	
	// Populate with test data
	numItems := 10000
	_, err = store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 0; i < numItems; i++ {
			hash := make([]byte, 32)
			binary.BigEndian.PutUint64(hash, uint64(i))
			
			err := appendAncientBlock(op, uint64(i), hash,
				[]byte(fmt.Sprintf("header-%d", i)),
				[]byte(fmt.Sprintf("body-%d", i)),
				[]byte(fmt.Sprintf("receipts-%d", i)),
				[]byte(fmt.Sprintf("td-%d", i)))
			if err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(b, err)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		idx := uint64(i % numItems)
		_, err := store.Ancient(chainFreezerHeaderTable, idx)
		if err != nil {
			b.Fatal(err)
		}
	}
}