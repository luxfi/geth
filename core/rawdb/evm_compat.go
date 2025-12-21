// Copyright (C) 2024, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package rawdb

import (
	"fmt"

	"github.com/luxfi/geth/ethdb"
)

// BuildCanonicalMappingsIfMissing checks if canonical mappings exist in database.
// For normal geth/C-Chain databases (no namespace).
// This is a no-op check - actual mapping is built during normal blockchain operation.
func BuildCanonicalMappingsIfMissing(db ethdb.Database) error {
	// Check if mappings exist
	testKey := append([]byte{headerNumberPrefix[0]}, make([]byte, 32)...)
	if val, err := db.Get(testKey); err == nil && len(val) > 0 {
		return nil // Mappings exist
	}

	// For normal databases, mappings are built during blockchain operations
	// This is just a compatibility check for EVM migration
	return nil
}

// BuildCanonicalMappingsFromEVM builds canonical hash mappings from EVM database.
// For EVM migration: scans raw DB with namespace, writes to target DB without namespace.
//
// Simple composition:
// 1. Check if already done
// 2. Scan raw DB for headers (with namespace)
// 3. Write mappings to target DB (without namespace)
// 4. Verify
func BuildCanonicalMappingsFromEVM(targetDB ethdb.Database, rawDB ethdb.Database, namespace []byte) error {
	// Already have mappings?
	testKey := append([]byte{headerNumberPrefix[0]}, make([]byte, 32)...)
	if val, err := targetDB.Get(testKey); err == nil && len(val) > 0 {
		fmt.Printf("✅ Canonical mappings exist, skipping\n")
		return nil
	}

	fmt.Printf("🔍 Building canonical mappings from EVM database\n")

	// Scan headers from raw DB (has namespace prefix)
	headers, err := ScanEVMHeaders(rawDB, namespace)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	if len(headers) == 0 {
		return fmt.Errorf("no headers found")
	}

	// Write mappings to target DB (SubnetNamespaceStripper handles namespace automatically)
	// The stripper will add the namespace prefix when writing and strip it when reading
	fmt.Printf("✍️  Writing %d canonical mappings\n", len(headers))
	if err := WriteMappings(targetDB, headers); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	// Sync if possible
	if syncer, ok := targetDB.(interface{ Sync() error }); ok {
		if err := syncer.Sync(); err != nil {
			fmt.Printf("⚠️  Sync failed: %v\n", err)
		}
	}

	// Verify sample blocks
	var maxBlock uint64
	for num := range headers {
		if num > maxBlock {
			maxBlock = num
		}
	}
	checkBlocks := []uint64{0, 1, 100, 1000, 10000, maxBlock}

	fmt.Printf("🔍 Verifying mappings\n")
	if err := VerifyMappings(targetDB, headers, checkBlocks); err != nil {
		return fmt.Errorf("verify failed: %w", err)
	}

	fmt.Printf("✅ Built and verified %d canonical mappings\n", len(headers))
	fmt.Printf("🎯 Highest block: %d\n", maxBlock)

	return nil
}
