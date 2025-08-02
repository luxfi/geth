package badgerdb

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupArchiveDatabase creates an archive database with initial data
func setupArchiveDatabase(t *testing.T, dir string) {
	archiveDB, err := NewBadgerDatabase(dir, false, false)
	require.NoError(t, err)
	err = archiveDB.Put([]byte("init"), []byte("data"))
	require.NoError(t, err)
	archiveDB.Close()
}

// TestNewArchiver tests archiver creation
func TestNewArchiver(t *testing.T) {
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 32,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	tests := []struct {
		name            string
		finalityDelay   uint64
		archiveInterval time.Duration
		expectError     bool
	}{
		{
			name:            "Valid configuration",
			finalityDelay:   32,
			archiveInterval: time.Minute,
			expectError:     false,
		},
		{
			name:            "Zero finality delay",
			finalityDelay:   0,
			archiveInterval: time.Minute,
			expectError:     false,
		},
		{
			name:            "Very short interval",
			finalityDelay:   32,
			archiveInterval: time.Millisecond,
			expectError:     false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archiver := NewArchiver(ddb, tt.finalityDelay, tt.archiveInterval)
			assert.NotNil(t, archiver)
			assert.Equal(t, tt.finalityDelay, archiver.finalityDelay)
			assert.Equal(t, tt.archiveInterval, archiver.archiveInterval)
		})
	}
}

// TestArchiverStartStop tests starting and stopping the archiver
func TestArchiverStartStop(t *testing.T) {
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath: archiveDir,
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 32, 100*time.Millisecond)
	
	// Start archiver
	archiver.Start()
	
	// Wait a bit
	time.Sleep(50 * time.Millisecond)
	
	// Stop archiver
	archiver.Stop()
	
	// Try to stop again (should be safe)
	archiver.Stop()
}

// TestArchiverArchiveFinalized tests the archiveFinalized method
func TestArchiverArchiveFinalized(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 5, time.Hour) // Long interval so it doesn't run automatically
	
	// Simulate blocks in current database
	for i := uint64(0); i < 20; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	// Set current height
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(19))
	require.NoError(t, err)
	
	// Run archiving
	err = archiver.archiveFinalized()
	assert.NoError(t, err)
	
	// Check that blocks below finality were moved
	// Blocks 0-13 should be archived (19 - 5 - 1 = 13)
	for i := uint64(0); i <= 13; i++ {
		key := fmt.Sprintf("block-%d", i)
		
		// Should be in archive
		value, err := ddb.archiveDB.Get([]byte(key))
		assert.NoError(t, err, "block %d should be in archive", i)
		assert.Equal(t, fmt.Sprintf("data-%d", i), string(value))
		
		// Should not be in current
		_, err = ddb.currentDB.Get([]byte(key))
		assert.Error(t, err, "block %d should not be in current", i)
	}
	
	// Blocks 14-19 should still be in current only
	for i := uint64(14); i < 20; i++ {
		key := fmt.Sprintf("block-%d", i)
		
		// Should be in current
		value, err := ddb.currentDB.Get([]byte(key))
		assert.NoError(t, err, "block %d should be in current", i)
		assert.Equal(t, fmt.Sprintf("data-%d", i), string(value))
		
		// Should not be in archive
		_, err = ddb.archiveDB.Get([]byte(key))
		assert.Error(t, err, "block %d should not be in archive", i)
	}
	
	// Check metrics
	assert.Greater(t, archiver.totalArchived.Load(), uint64(0))
	assert.Greater(t, archiver.lastArchived.Load(), int64(0))
}

// TestArchiverGetCurrentHeight tests height retrieval
func TestArchiverGetCurrentHeight(t *testing.T) {
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath: archiveDir,
		CurrentPath: t.TempDir(),
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 32, time.Hour)
	
	// Test when no height is set
	height, err := archiver.getCurrentHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), height)
	
	// Set a height
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(12345))
	require.NoError(t, err)
	
	// Test retrieval
	height, err = archiver.getCurrentHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(12345), height)
}

// TestArchiverContinuousOperation tests continuous archiving
func TestArchiverContinuousOperation(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 5,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Create archiver with very short interval
	archiver := NewArchiver(ddb, 5, 50*time.Millisecond)
	archiver.Start()
	defer archiver.Stop()
	
	// Continuously add blocks
	for i := uint64(0); i < 30; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(t, err)
		
		// Update height
		heightKey := []byte("LastBlock")
		err = ddb.Put(heightKey, encodeBlockNumber(i))
		require.NoError(t, err)
		
		// Small delay to simulate block production
		time.Sleep(20 * time.Millisecond)
	}
	
	// Wait for archiving to catch up
	time.Sleep(200 * time.Millisecond)
	
	// Check that archiving happened
	assert.Greater(t, archiver.totalArchived.Load(), uint64(10))
	
	// Verify some archived blocks
	for i := uint64(0); i < 20; i++ {
		key := fmt.Sprintf("block-%d", i)
		
		// Should be readable from dual database
		value, err := ddb.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("data-%d", i), string(value))
	}
}

// TestArchiverBatchProcessing tests batch processing
func TestArchiverBatchProcessing(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 5, time.Hour)
	
	// Add many blocks
	numBlocks := 1000
	for i := 0; i < numBlocks; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	// Set height
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(uint64(numBlocks-1)))
	require.NoError(t, err)
	
	// Run archiving
	start := time.Now()
	err = archiver.archiveFinalized()
	duration := time.Since(start)
	
	assert.NoError(t, err)
	assert.Less(t, duration, 5*time.Second) // Should be reasonably fast
	
	// Check metrics
	expectedArchived := uint64(numBlocks - 5 - 1) // numBlocks - finalityDelay - 1
	assert.Equal(t, expectedArchived, archiver.totalArchived.Load())
}

// TestArchiverErrorHandling tests error handling
func TestArchiverErrorHandling(t *testing.T) {
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	
	archiver := NewArchiver(ddb, 5, 100*time.Millisecond)
	archiver.Start()
	
	// Close the database to cause errors
	ddb.Close()
	
	// Wait for archiver to try running
	time.Sleep(200 * time.Millisecond)
	
	// Should have logged errors but not crashed
	archiver.Stop()
}

// TestArchiverMetrics tests metrics collection
func TestArchiverMetrics(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 5,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 3, time.Hour)
	
	// Initial metrics
	assert.Equal(t, uint64(0), archiver.totalArchived.Load())
	assert.Equal(t, int64(0), archiver.lastArchived.Load())
	assert.Equal(t, uint64(0), archiver.totalDeleted.Load())
	
	// Add blocks and archive
	for i := uint64(0); i < 10; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(9))
	require.NoError(t, err)
	
	// Archive
	err = archiver.archiveFinalized()
	assert.NoError(t, err)
	
	// Check metrics
	assert.Greater(t, archiver.totalArchived.Load(), uint64(0))
	assert.Greater(t, archiver.lastArchived.Load(), int64(0))
	assert.Greater(t, archiver.totalDeleted.Load(), uint64(0))
}

// TestArchiverIdempotency tests that archiving is idempotent
func TestArchiverIdempotency(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 5, time.Hour)
	
	// Add blocks
	for i := uint64(0); i < 20; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(19))
	require.NoError(t, err)
	
	// Archive once
	err = archiver.archiveFinalized()
	assert.NoError(t, err)
	firstArchived := archiver.totalArchived.Load()
	
	// Archive again without adding new blocks
	err = archiver.archiveFinalized()
	assert.NoError(t, err)
	secondArchived := archiver.totalArchived.Load()
	
	// Should not archive anything new
	assert.Equal(t, firstArchived, secondArchived)
}

// TestArchiverConcurrentAccess tests concurrent access during archiving
func TestArchiverConcurrentAccess(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 5, 50*time.Millisecond)
	archiver.Start()
	defer archiver.Stop()
	
	// Writer goroutine
	done := make(chan bool)
	go func() {
		for i := uint64(0); i < 100; i++ {
			key := fmt.Sprintf("block-%d", i)
			value := fmt.Sprintf("data-%d", i)
			err := ddb.Put([]byte(key), []byte(value))
			assert.NoError(t, err)
			
			// Update height
			heightKey := []byte("LastBlock")
			err = ddb.Put(heightKey, encodeBlockNumber(i))
			assert.NoError(t, err)
			
			time.Sleep(5 * time.Millisecond)
		}
		done <- true
	}()
	
	// Reader goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				// Try to read random blocks
				for i := 0; i < 10; i++ {
					key := fmt.Sprintf("block-%d", i)
					ddb.Get([]byte(key)) // Ignore errors, block might not exist yet
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
	}()
	
	// Wait for writer to finish
	<-done
	
	// Wait for archiving to catch up
	time.Sleep(200 * time.Millisecond)
	
	// Verify data integrity
	for i := uint64(0); i < 100; i++ {
		key := fmt.Sprintf("block-%d", i)
		value, err := ddb.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("data-%d", i), string(value))
	}
}

// BenchmarkArchiver benchmarks archiving performance
func BenchmarkArchiver(b *testing.B) {
	archiveDir := b.TempDir()
	archiveDB, err := NewBadgerDatabase(archiveDir, false, false)
	require.NoError(b, err)
	err = archiveDB.Put([]byte("init"), []byte("data"))
	require.NoError(b, err)
	archiveDB.Close()
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    b.TempDir(),
		FinalityHeight: 100,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(b, err)
	defer ddb.Close()
	
	archiver := NewArchiver(ddb, 32, time.Hour)
	
	// Prepare data
	numBlocks := 10000
	for i := 0; i < numBlocks; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(b, err)
	}
	
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(uint64(numBlocks-1)))
	require.NoError(b, err)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Reset counter
		archiver.totalArchived.Store(0)
		
		// Run archiving
		err := archiver.archiveFinalized()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestArchiverNoArchiveDatabase tests archiver with no archive database
func TestArchiverNoArchiveDatabase(t *testing.T) {
	config := DualDatabaseConfig{
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
		// No archive path
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Should not create archiver if no archive
	assert.False(t, ddb.hasArchive)
}

// mockArchiverDatabase is a mock database for testing error scenarios
type mockArchiverDatabase struct {
	*DualBadgerDatabase
	failGet    bool
	failPut    bool
	failDelete bool
	failNewIterator bool
}

func (m *mockArchiverDatabase) Get(key []byte) ([]byte, error) {
	if m.failGet {
		return nil, fmt.Errorf("mock get error")
	}
	return m.DualBadgerDatabase.Get(key)
}

func (m *mockArchiverDatabase) Put(key []byte, value []byte) error {
	if m.failPut {
		return fmt.Errorf("mock put error")
	}
	return m.DualBadgerDatabase.Put(key, value)
}

func (m *mockArchiverDatabase) Delete(key []byte) error {
	if m.failDelete {
		return fmt.Errorf("mock delete error")
	}
	return m.DualBadgerDatabase.Delete(key)
}

// TestArchiverWithMockErrors tests error scenarios
func TestArchiverWithMockErrors(t *testing.T) {
	t.Skip("Archiver is obsolete with ancient store architecture")
	archiveDir := t.TempDir()
	setupArchiveDatabase(t, archiveDir)
	
	config := DualDatabaseConfig{
		ArchivePath:    archiveDir,
		CurrentPath:    t.TempDir(),
		FinalityHeight: 10,
	}
	
	ddb, err := NewDualBadgerDatabase(config)
	require.NoError(t, err)
	defer ddb.Close()
	
	// Add some data
	for i := uint64(0); i < 20; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := ddb.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	heightKey := []byte("LastBlock")
	err = ddb.Put(heightKey, encodeBlockNumber(19))
	require.NoError(t, err)
	
	// Create archiver with mock database
	mockDB := &mockArchiverDatabase{DualBadgerDatabase: ddb}
	archiver := &Archiver{
		db:              ddb,  // Use real db for most operations
		finalityDelay:   5,
		archiveInterval: time.Hour,
		totalArchived:   atomic.Uint64{},
		lastArchived:    atomic.Int64{},
		totalDeleted:    atomic.Uint64{},
	}
	
	// Test with get error
	mockDB.failGet = true
	err = archiver.archiveFinalized()
	assert.NoError(t, err) // Should handle error gracefully
	
	// Reset and test normal operation
	mockDB.failGet = false
	err = archiver.archiveFinalized()
	assert.NoError(t, err)
	assert.Greater(t, archiver.totalArchived.Load(), uint64(0))
}