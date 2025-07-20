# ethdb Package

This package provides database interfaces and implementations for the Lux blockchain.

## Design Decision

Our `ethdb.Database` interface extends `github.com/ethereum/go-ethereum/ethdb.Database` with an additional `SyncAncient()` method. However, the concrete implementations (leveldb, pebble) from upstream don't implement the full `ethdb.Database` interface - they're lower-level implementations.

The proper way to get a full database is through the rawdb package which wraps these concrete implementations with additional functionality like ancient store support.

For now, the leveldb and pebble packages return the upstream types directly, which can be used with rawdb.NewDatabase() to get a full implementation.