package genesis

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/luxfi/geth/ethdb/badgerdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSourceDB implements a simple in-memory database for testing
type mockSourceDB struct {
	data map[string][]byte
	ethdb.Database
}

func newMockSourceDB() *mockSourceDB {
	return &mockSourceDB{
		data: make(map[string][]byte),
	}
}

func (m *mockSourceDB) Get(key []byte) ([]byte, error) {
	if value, ok := m.data[string(key)]; ok {
		return value, nil
	}
	return nil, fmt.Errorf("key not found")
}

func (m *mockSourceDB) Put(key []byte, value []byte) error {
	m.data[string(key)] = value
	return nil
}

func (m *mockSourceDB) Has(key []byte) (bool, error) {
	_, ok := m.data[string(key)]
	return ok, nil
}

func (m *mockSourceDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return &mockIterator{
		data:    m.data,
		keys:    getSortedKeys(m.data, prefix, start),
		current: -1,
	}
}

func (m *mockSourceDB) Close() error {
	return nil
}

// mockIterator implements ethdb.Iterator for testing
type mockIterator struct {
	data    map[string][]byte
	keys    []string
	current int
}

func (it *mockIterator) Next() bool {
	it.current++
	return it.current < len(it.keys)
}

func (it *mockIterator) Key() []byte {
	if it.current >= 0 && it.current < len(it.keys) {
		return []byte(it.keys[it.current])
	}
	return nil
}

func (it *mockIterator) Value() []byte {
	if it.current >= 0 && it.current < len(it.keys) {
		return it.data[it.keys[it.current]]
	}
	return nil
}

func (it *mockIterator) Error() error {
	return nil
}

func (it *mockIterator) Release() {}

func getSortedKeys(data map[string][]byte, prefix []byte, start []byte) []string {
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	// Simple sorting for testing
	return keys
}

// TestNewGenesisReplayer tests replayer creation
func TestNewGenesisReplayer(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		GenesisPath:     "/path/to/genesis",
		TargetPath:      targetDir,
		BatchSize:       1000,
		VerifyBlocks:    true,
		ContinueOnError: false,
		ProgressLogInterval: 10 * time.Second,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	assert.NotNil(t, replayer)
	assert.Equal(t, config.BatchSize, replayer.config.BatchSize)
	assert.Equal(t, config.VerifyBlocks, replayer.config.VerifyBlocks)
}

// TestGenesisReplayerCheckReplayStatus tests replay status checking
func TestGenesisReplayerCheckReplayStatus(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize: 100,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Test when no replay has been done
	status, err := replayer.checkReplayStatus()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), status.LastReplayedHeight)
	assert.False(t, status.IsComplete)
	
	// Simulate a replay status
	testStatus := ReplayStatus{
		LastReplayedHeight: 1000,
		LastReplayedHash:   "0x1234",
		IsComplete:         false,
		StartTime:          time.Now(),
		LastUpdateTime:     time.Now(),
	}
	
	err = replayer.saveReplayStatus(testStatus)
	assert.NoError(t, err)
	
	// Check status again
	status, err = replayer.checkReplayStatus()
	assert.NoError(t, err)
	assert.Equal(t, testStatus.LastReplayedHeight, status.LastReplayedHeight)
	assert.Equal(t, testStatus.LastReplayedHash, status.LastReplayedHash)
	assert.Equal(t, testStatus.IsComplete, status.IsComplete)
}

// TestGenesisReplayerReplayBlocks tests block replay
func TestGenesisReplayerReplayBlocks(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add test data to source
	testData := map[string]string{
		"block-0": "genesis-block",
		"block-1": "first-block",
		"block-2": "second-block",
		"state-0": "genesis-state",
		"state-1": "first-state",
	}
	
	for k, v := range testData {
		err := sourceDB.Put([]byte(k), []byte(v))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize:    2,
		VerifyBlocks: false,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Replay blocks
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Verify data was copied
	for k, expectedValue := range testData {
		value, err := targetDB.Get([]byte(k))
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, string(value))
	}
	
	// Check replay status
	status, err := replayer.checkReplayStatus()
	assert.NoError(t, err)
	assert.True(t, status.IsComplete)
}

// TestGenesisReplayerIdempotency tests idempotent replay
func TestGenesisReplayerIdempotency(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add test data
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize: 5,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// First replay
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Get metrics after first replay
	firstMetrics := replayer.GetMetrics()
	firstProcessed := firstMetrics.TotalKeysProcessed
	
	// Second replay (should be idempotent)
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Should not process any new keys
	secondMetrics := replayer.GetMetrics()
	assert.Equal(t, firstProcessed, secondMetrics.TotalKeysProcessed)
}

// TestGenesisReplayerBatchProcessing tests batch processing
func TestGenesisReplayerBatchProcessing(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add many test entries
	numEntries := 1000
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key-%05d", i)
		value := fmt.Sprintf("value-%d", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize: 100,
		ProgressLogInterval: time.Millisecond, // Fast logging for test
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Replay
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Verify all data was copied
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key-%05d", i)
		expectedValue := fmt.Sprintf("value-%d", i)
		
		value, err := targetDB.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, expectedValue, string(value))
	}
	
	// Check metrics
	metrics := replayer.GetMetrics()
	assert.Equal(t, uint64(numEntries), metrics.TotalKeysProcessed)
}

// TestGenesisReplayerErrorHandling tests error handling
func TestGenesisReplayerErrorHandling(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add test data
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	
	config := ReplayConfig{
		BatchSize:       5,
		ContinueOnError: true,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Close target DB to cause errors
	targetDB.Close()
	
	// Should handle error gracefully with ContinueOnError
	err = replayer.Replay()
	assert.Error(t, err)
	
	// Without ContinueOnError
	config.ContinueOnError = false
	replayer = NewGenesisReplayer(sourceDB, targetDB, config)
	
	err = replayer.Replay()
	assert.Error(t, err)
}

// TestGenesisReplayerVerifyTips tests tip verification
func TestGenesisReplayerVerifyTips(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add tip data
	tipKey := []byte("chain-tip")
	tipValue := []byte("tip-hash-12345")
	err := sourceDB.Put(tipKey, tipValue)
	require.NoError(t, err)
	
	// Add some blocks
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("block-%d", i)
		value := fmt.Sprintf("data-%d", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize:    10,
		VerifyBlocks: true,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Replay
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Verify tips match
	err = replayer.VerifyTips()
	assert.NoError(t, err)
	
	// Modify tip in target to cause mismatch
	err = targetDB.Put(tipKey, []byte("different-tip"))
	assert.NoError(t, err)
	
	// Should fail verification
	err = replayer.VerifyTips()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tip mismatch")
}

// TestGenesisReplayerMetrics tests metrics collection
func TestGenesisReplayerMetrics(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add test data
	numEntries := 100
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize: 20,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Get initial metrics
	metrics := replayer.GetMetrics()
	assert.Equal(t, uint64(0), metrics.TotalKeysProcessed)
	assert.Equal(t, uint64(0), metrics.TotalBytesProcessed)
	
	// Replay
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Check final metrics
	metrics = replayer.GetMetrics()
	assert.Equal(t, uint64(numEntries), metrics.TotalKeysProcessed)
	assert.Greater(t, metrics.TotalBytesProcessed, uint64(0))
	assert.Greater(t, metrics.Duration, time.Duration(0))
}

// TestGenesisReplayerConcurrentReplay tests concurrent replay attempts
func TestGenesisReplayerConcurrentReplay(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add test data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize: 10,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Try concurrent replays
	done := make(chan error, 2)
	
	go func() {
		done <- replayer.Replay()
	}()
	
	go func() {
		time.Sleep(10 * time.Millisecond) // Small delay
		done <- replayer.Replay()
	}()
	
	// Wait for both to complete
	err1 := <-done
	err2 := <-done
	
	// At least one should succeed
	assert.True(t, err1 == nil || err2 == nil)
}

// TestGenesisReplayerLargeValues tests handling of large values
func TestGenesisReplayerLargeValues(t *testing.T) {
	sourceDB := newMockSourceDB()
	targetDir := t.TempDir()
	
	// Add large values
	largeValue := make([]byte, 1024*1024) // 1MB
	for i := range largeValue {
		largeValue[i] = byte(i % 256)
	}
	
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("large-%d", i)
		err := sourceDB.Put([]byte(key), largeValue)
		require.NoError(t, err)
	}
	
	targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
	require.NoError(t, err)
	defer targetDB.Close()
	
	config := ReplayConfig{
		BatchSize: 2,
	}
	
	replayer := NewGenesisReplayer(sourceDB, targetDB, config)
	
	// Replay
	err = replayer.Replay()
	assert.NoError(t, err)
	
	// Verify large values
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("large-%d", i)
		value, err := targetDB.Get([]byte(key))
		assert.NoError(t, err)
		assert.Equal(t, len(largeValue), len(value))
	}
}

// BenchmarkGenesisReplayer benchmarks replay performance
func BenchmarkGenesisReplayer(b *testing.B) {
	sourceDB := newMockSourceDB()
	
	// Add test data
	numEntries := 10000
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("key-%08d", i)
		value := fmt.Sprintf("value-%d-with-some-extra-data-to-make-it-larger", i)
		err := sourceDB.Put([]byte(key), []byte(value))
		require.NoError(b, err)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		targetDir := b.TempDir()
		targetDB, err := badgerdb.NewBadgerDatabase(targetDir, false, false)
		require.NoError(b, err)
		
		config := ReplayConfig{
			BatchSize: 1000,
		}
		
		replayer := NewGenesisReplayer(sourceDB, targetDB, config)
		b.StartTimer()
		
		err = replayer.Replay()
		if err != nil {
			b.Fatal(err)
		}
		
		b.StopTimer()
		targetDB.Close()
		os.RemoveAll(targetDir)
		b.StartTimer()
	}
}

// TestDetectDatabaseType tests database type detection
func TestDetectDatabaseType(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected DatabaseType
	}{
		{
			name:     "LevelDB",
			files:    []string{"CURRENT", "LOG", "MANIFEST-000001"},
			expected: LevelDB,
		},
		{
			name:     "PebbleDB with MANIFEST-000000",
			files:    []string{"MANIFEST-000000", "000001.sst"},
			expected: PebbleDB,
		},
		{
			name:     "PebbleDB with MANIFEST-000001",
			files:    []string{"MANIFEST-000001", "000001.sst"},
			expected: PebbleDB,
		},
		{
			name:     "BadgerDB",
			files:    []string{"MANIFEST", "000001.vlog"},
			expected: BadgerDB,
		},
		{
			name:     "Unknown",
			files:    []string{"random.txt", "data.bin"},
			expected: UnknownDB,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			
			// Create test files
			for _, file := range tt.files {
				path := filepath.Join(dir, file)
				err := os.WriteFile(path, []byte("test"), 0644)
				require.NoError(t, err)
			}
			
			// Detect type
			dbType := DetectDatabaseType(dir)
			assert.Equal(t, tt.expected, dbType)
		})
	}
}

// TestGetDatabaseInfo tests database info retrieval
func TestGetDatabaseInfo(t *testing.T) {
	dir := t.TempDir()
	
	// Create test files
	files := map[string]int{
		"CURRENT":   10,
		"LOG":       1024,
		"000001.sst": 1024 * 1024,
		"000002.sst": 2 * 1024 * 1024,
	}
	
	for name, size := range files {
		path := filepath.Join(dir, name)
		data := make([]byte, size)
		err := os.WriteFile(path, data, 0644)
		require.NoError(t, err)
	}
	
	// Get info
	info, err := GetDatabaseInfo(dir)
	assert.NoError(t, err)
	assert.Equal(t, LevelDB, info.Type)
	assert.Equal(t, dir, info.Path)
	assert.Equal(t, 4, info.FileCount)
	
	expectedSize := int64(10 + 1024 + 1024*1024 + 2*1024*1024)
	assert.Equal(t, expectedSize, info.Size)
}