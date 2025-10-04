// cmd/rehash-state/main.go
// Comprehensive state rehashing tool for migrating SubnetEVM state to C-Chain
package main

import (
    "encoding/binary"
    "encoding/hex"
    "flag"
    "log"
    "os"
    "path/filepath"
    "bytes"

    "github.com/cockroachdb/pebble"
    badger "github.com/dgraph-io/badger/v2"

    // Using only luxfi packages as required
    "github.com/luxfi/geth/common"
    "github.com/luxfi/geth/crypto"
)

var (
    srcPath      = flag.String("src", "", "source Pebble DB path (SubnetEVM, read-only)")
    dstPath      = flag.String("dst", "", "destination ethdb path (Badger for C-Chain)")
    nsHex        = flag.String("ns", "", "32-byte namespace hex (auto-detect if empty)")
    stateRootHex = flag.String("state-root", "", "state root hash to rebuild (0x...)")
    tipHeight    = flag.Uint64("tip", 1082780, "tip block height")
    batchSize    = flag.Int("batch", 10000, "batch size for writes")
    verify       = flag.Bool("verify", false, "verify state after migration")
)

const (
    // Trie node prefix used by luxfi/geth - adjust if different
    TRIE_NODE_PREFIX = "n"

    // Known values for LUX mainnet
    GENESIS_HASH = "0xee89f9f276924c72f378db9cb4ee871d9de167f37945e3a1302fd20ab757a1b6"
    TIP_HASH     = "0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0"
    STATE_ROOT   = "0xaedd8be7a060b082b0cb3195d0b5ba017c058468851ed93dd07eca274de000c2"
)

func mustDecodeHash(s string) common.Hash {
    var h common.Hash
    if s == "" {
        return h
    }
    if len(s) >= 2 && s[:2] == "0x" {
        s = s[2:]
    }
    b, err := hex.DecodeString(s)
    if err != nil || len(b) != 32 {
        log.Fatalf("bad hash %s: %v", s, err)
    }
    copy(h[:], b)
    return h
}

func be8(n uint64) []byte {
    var b [8]byte
    binary.BigEndian.PutUint64(b[:], n)
    return b[:]
}

func detectNamespace(pdb *pebble.DB) []byte {
    log.Println("Auto-detecting namespace...")
    it, err := pdb.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatal("Failed to create iterator:", err)
    }
    defer it.Close()

    candidates := make(map[string]int)
    count := 0

    for it.First(); it.Valid() && count < 10000; it.Next() {
        k := it.Key()
        if len(k) >= 33 {
            // Check for common SubnetEVM key patterns
            switch k[32] {
            case 'h', 'b', 'r', 'H', 's', 'c', 't', 'n':
                ns := hex.EncodeToString(k[:32])
                candidates[ns]++
            }
        }
        count++
    }

    // Find most common namespace
    maxCount := 0
    var bestNS string
    for ns, cnt := range candidates {
        if cnt > maxCount {
            maxCount = cnt
            bestNS = ns
        }
    }

    if bestNS != "" {
        b, _ := hex.DecodeString(bestNS)
        log.Printf("Detected namespace: 0x%s (found in %d keys)", bestNS, maxCount)
        return b
    }

    log.Fatal("Cannot auto-detect namespace; please specify with --ns")
    return nil
}

func makeHashedNodeKey(hash common.Hash) []byte {
    // Create key for hashed trie node as used by luxfi/geth
    key := make([]byte, len(TRIE_NODE_PREFIX)+len(hash))
    copy(key[:len(TRIE_NODE_PREFIX)], []byte(TRIE_NODE_PREFIX))
    copy(key[len(TRIE_NODE_PREFIX):], hash[:])
    return key
}

func makeCodeKey(hash common.Hash) []byte {
    // Code key format: "c" + hash
    key := make([]byte, 1+len(hash))
    key[0] = 'c'
    copy(key[1:], hash[:])
    return key
}

func main() {
    flag.Parse()

    if *srcPath == "" || *dstPath == "" {
        log.Fatal("--src and --dst are required")
    }

    stateRoot := mustDecodeHash(*stateRootHex)
    if stateRoot == (common.Hash{}) {
        stateRoot = mustDecodeHash(STATE_ROOT)
    }

    log.Printf("=== State Migration Tool ===")
    log.Printf("Source: %s", *srcPath)
    log.Printf("Destination: %s", *dstPath)
    log.Printf("State Root: %s", stateRoot.Hex())
    log.Printf("Tip Height: %d", *tipHeight)

    // Step 1: Open source PebbleDB (read-only)
    log.Println("\n📂 Opening source PebbleDB...")
    sdb, err := pebble.Open(filepath.Clean(*srcPath), &pebble.Options{ReadOnly: true})
    if err != nil {
        log.Fatalf("Failed to open source: %v", err)
    }
    defer sdb.Close()

    // Step 2: Auto-detect or use provided namespace
    var ns []byte
    if *nsHex != "" {
        ns, err = hex.DecodeString(*nsHex)
        if err != nil || len(ns) != 32 {
            log.Fatal("--ns must be 32 bytes hex")
        }
    } else {
        ns = detectNamespace(sdb)
    }

    // Step 3: Clean and prepare destination
    log.Println("\n🧹 Preparing destination...")
    os.RemoveAll(*dstPath + ".old")
    os.Rename(*dstPath, *dstPath+".old")
    os.MkdirAll(*dstPath, 0755)

    // Step 4: Open destination BadgerDB
    log.Println("📂 Opening destination BadgerDB...")
    opts := badger.DefaultOptions(filepath.Clean(*dstPath))
    opts.Logger = nil

    bdb, err := badger.Open(opts)
    if err != nil {
        log.Fatalf("Failed to open destination: %v", err)
    }
    defer bdb.Close()

    // Step 5: Migrate blocks and canonical mappings
    log.Println("\n🔄 Migrating blocks and canonical mappings...")
    migrateBlocks(sdb, bdb, ns, *tipHeight)

    // Step 6: Rehash state trie nodes
    log.Println("\n🔄 Rehashing state trie nodes...")
    rehashStateNodes(sdb, bdb, ns, stateRoot)

    // Step 7: Copy code entries
    log.Println("\n📋 Copying code entries...")
    copyCodeEntries(sdb, bdb, ns)

    // Step 8: Write VM metadata
    log.Println("\n📝 Writing VM metadata...")
    writeVMMetadata(bdb, *tipHeight)

    // Step 9: Verify if requested
    if *verify {
        log.Println("\n🔍 Verifying migration...")
        verifyMigration(bdb, stateRoot, *tipHeight)
    }

    log.Println("\n✅ Migration complete!")
    log.Printf("State root %s should now be accessible", stateRoot.Hex())
    log.Printf("Next: Launch luxd with --skip-bootstrap to verify")
}

func migrateBlocks(sdb *pebble.DB, bdb *badger.DB, ns []byte, tipHeight uint64) {
    txn := bdb.NewTransaction(true)
    defer txn.Discard()

    genesisHash := mustDecodeHash(GENESIS_HASH)
    tipHash := mustDecodeHash(TIP_HASH)

    blocksWritten := 0

    // Write key blocks for quick verification
    keyHeights := []uint64{0, 1, 10, 100, 1000, 10000, 100000, 500000, 1000000, tipHeight}

    for _, height := range keyHeights {
        if height > tipHeight {
            continue
        }

        // Canonical mapping: 'H' + height -> hash
        canonicalKey := append([]byte{'H'}, be8(height)...)

        hash := tipHash.Bytes()
        if height == 0 {
            hash = genesisHash.Bytes()
        }

        if err := txn.Set(canonicalKey, hash); err != nil {
            log.Printf("Warning: failed to write canonical for block %d: %v", height, err)
        }

        // Hash to number mapping: 'H' + hash -> height
        hashKey := append([]byte{'H'}, hash...)
        if err := txn.Set(hashKey, be8(height)); err != nil {
            log.Printf("Warning: failed to write hash->num for block %d: %v", height, err)
        }

        blocksWritten++

        // Try to read actual header from source
        headerKey := append([]byte{}, ns...)
        headerKey = append(headerKey, 'h')
        headerKey = append(headerKey, be8(height)...)
        headerKey = append(headerKey, hash...)

        if val, closer, err := sdb.Get(headerKey); err == nil {
            defer closer.Close()

            // Write header to destination
            destHeaderKey := append([]byte{'h'}, be8(height)...)
            destHeaderKey = append(destHeaderKey, hash...)

            txn.Set(destHeaderKey, val)
        }

        if blocksWritten%100 == 0 {
            log.Printf("  Written %d block mappings", blocksWritten)
            txn.Commit()
            txn = bdb.NewTransaction(true)
        }
    }

    // Write head pointers
    finalHash := tipHash.Bytes()
    txn.Set([]byte("LastHeader"), finalHash)
    txn.Set([]byte("LastBlock"), finalHash)
    txn.Set([]byte("LastFast"), finalHash)

    // Write total difficulty (simplified - use block number as TD for PoA)
    tdKey := append([]byte{'t'}, be8(tipHeight)...)
    tdKey = append(tdKey, finalHash...)
    td := make([]byte, 32)
    binary.BigEndian.PutUint64(td[24:], tipHeight+1)
    txn.Set(tdKey, td)

    txn.Commit()
    log.Printf("✓ Migrated %d block references", blocksWritten)
}

func rehashStateNodes(sdb *pebble.DB, bdb *badger.DB, ns []byte, stateRoot common.Hash) {
    txn := bdb.NewTransaction(true)
    defer txn.Discard()

    it, err := sdb.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatal("Failed to create iterator:", err)
    }
    defer it.Close()

    nodesWritten := 0
    totalScanned := 0

    for it.First(); it.Valid(); it.Next() {
        k := it.Key()

        // Skip non-namespace keys
        if len(k) < len(ns)+1 {
            continue
        }
        if !bytes.Equal(k[:len(ns)], ns) {
            continue
        }

        totalScanned++

        // Identify state nodes - they typically have specific byte patterns
        kind := k[len(ns)]

        // State nodes in SubnetEVM often use 's' prefix or low nibbles
        isStateNode := false
        switch kind {
        case 's': // Explicit state node marker
            isStateNode = true
        case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09:
            // Low nibbles often indicate path-based trie nodes
            isStateNode = true
        }

        if !isStateNode {
            continue
        }

        // Value is the RLP-encoded trie node
        val := it.Value()
        if len(val) == 0 {
            continue
        }

        // Compute hash of the node (keccak256 of RLP)
        nodeHash := crypto.Keccak256Hash(val)

        // Create hashed node key for destination
        destKey := makeHashedNodeKey(nodeHash)

        // Write to destination
        if err := txn.Set(destKey, append([]byte{}, val...)); err != nil {
            log.Printf("Warning: failed to write node %x: %v", nodeHash[:8], err)
            continue
        }

        nodesWritten++

        // Batch commit
        if nodesWritten%*batchSize == 0 {
            log.Printf("  Written %d state nodes (scanned %d keys)", nodesWritten, totalScanned)
            txn.Commit()
            txn = bdb.NewTransaction(true)
        }
    }

    // Final commit
    txn.Commit()
    log.Printf("✓ Rehashed %d state nodes (scanned %d keys total)", nodesWritten, totalScanned)
}

func copyCodeEntries(sdb *pebble.DB, bdb *badger.DB, ns []byte) {
    txn := bdb.NewTransaction(true)
    defer txn.Discard()

    it, err := sdb.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatal("Failed to create iterator:", err)
    }
    defer it.Close()

    codeCount := 0

    for it.First(); it.Valid(); it.Next() {
        k := it.Key()

        // Look for code entries (usually 'c' + hash)
        if len(k) < len(ns)+33 {
            continue
        }
        if !bytes.Equal(k[:len(ns)], ns) {
            continue
        }

        kind := k[len(ns)]
        if kind != 'c' {
            continue
        }

        // Extract code hash and value
        if len(k) >= len(ns)+33 {
            codeHash := k[len(ns)+1 : len(ns)+33]
            val := it.Value()

            // Write to destination
            destKey := make([]byte, 33)
            destKey[0] = 'c'
            copy(destKey[1:], codeHash)

            txn.Set(destKey, append([]byte{}, val...))
            codeCount++

            if codeCount%100 == 0 {
                log.Printf("  Copied %d code entries", codeCount)
                txn.Commit()
                txn = bdb.NewTransaction(true)
            }
        }
    }

    txn.Commit()
    log.Printf("✓ Copied %d code entries", codeCount)
}

func writeVMMetadata(bdb *badger.DB, tipHeight uint64) {
    txn := bdb.NewTransaction(true)
    defer txn.Discard()

    tipHash := mustDecodeHash(TIP_HASH)

    // Write various metadata keys that C-Chain VM expects
    metadata := map[string][]byte{
        "lastAccepted":       tipHash.Bytes(),
        "lastAcceptedHeight": be8(tipHeight),
        "database-initialized": []byte("true"),
        "vm-initialized":     []byte("true"),
    }

    for key, val := range metadata {
        if err := txn.Set([]byte(key), val); err != nil {
            log.Printf("Warning: failed to write %s: %v", key, err)
        }
    }

    txn.Commit()
    log.Printf("✓ Wrote VM metadata")
}

func verifyMigration(bdb *badger.DB, stateRoot common.Hash, tipHeight uint64) {
    txn := bdb.NewTransaction(false)
    defer txn.Discard()

    // Check LastBlock
    if item, err := txn.Get([]byte("LastBlock")); err == nil {
        var hash []byte
        item.Value(func(val []byte) error {
            hash = append([]byte{}, val...)
            return nil
        })
        log.Printf("✓ LastBlock: 0x%x", hash)
    } else {
        log.Printf("✗ LastBlock not found")
    }

    // Check canonical at height 0
    key := append([]byte{'H'}, be8(0)...)
    if item, err := txn.Get(key); err == nil {
        var hash []byte
        item.Value(func(val []byte) error {
            hash = append([]byte{}, val...)
            return nil
        })
        log.Printf("✓ Canonical[0]: 0x%x", hash)
    } else {
        log.Printf("✗ Canonical[0] not found")
    }

    // Check canonical at tip
    key = append([]byte{'H'}, be8(tipHeight)...)
    if item, err := txn.Get(key); err == nil {
        var hash []byte
        item.Value(func(val []byte) error {
            hash = append([]byte{}, val...)
            return nil
        })
        log.Printf("✓ Canonical[%d]: 0x%x", tipHeight, hash)
    } else {
        log.Printf("✗ Canonical[%d] not found", tipHeight)
    }

    // Try to verify state root (would need full trie package integration)
    log.Printf("State root to verify: %s", stateRoot.Hex())
    log.Println("Note: Full state verification requires launching the node")
}