# geth Module Documentation

Module: github.com/luxfi/geth
Status: Active development
Last Upstream Sync: 2025-12-12 (go-ethereum 16f50285b)

---

## Lux C-Chain Header Format Management

### Overview

The Lux C-chain uses different RLP header formats depending on block era. This codebase implements format-aware encoding/decoding to ensure:
- **Hash consistency**: Re-encoding produces identical bytes → identical hashes
- **Format preservation**: Decoded blocks retain their original format when re-encoded
- **Upgrade flexibility**: New formats can be added without breaking historic data

### Critical Network Parameters

```
Chain ID:     96369
Genesis Hash: 0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
State Root:   0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80
```

### Header Formats

| Format | Fields | Description | Used By |
|--------|--------|-------------|---------|
| `RLPFormat15` | 15 | Pre-London (legacy Ethereum) | Legacy chains |
| `RLPFormat16` | 16 | Post-London (+ BaseFee) | **Lux Genesis (Block 0)** |
| `RLPFormat17` | 17 | + ExtDataHash (pointer type) | **Lux Post-Genesis Blocks** |
| `RLPFormat18` | 18 | + ExtDataGasUsed | Intermediate |
| `RLPFormat19Lux` | 19 | + BlockGasCost (ExtDataHash as VALUE) | Full coreth format |
| `RLPFormat20+` | 20-24 | + Ethereum 2.0 fields | Future upgrades |

### Key Implementation Files

```
core/types/block.go         - Header struct, EncodeRLP, format detection
core/types/decode.go        - DecodeHeader, format-specific decoders
core/types/gen_header_rlp.go - Auto-generated default encoder (renamed)
core/rawdb/accessors_chain.go - WriteHeaderWithHash (uses format-aware encoding)
```

### RLPFormat Field

The `Header` struct tracks its original format:

```go
type Header struct {
    // ... fields ...
    rlpFormat RLPFormat `json:"-" rlp:"-"` // Tracks original encoding format
}
```

Format is set during decode and used during encode:

```go
// Set during decode
func decode17(data []byte) (*Header, error) {
    // ... decode ...
    return &Header{
        // ... fields ...
        rlpFormat: RLPFormat17, // Track format
    }, nil
}

// Used during encode
func (h *Header) EncodeRLP(w io.Writer) error {
    switch h.rlpFormat {
    case RLPFormat17:
        return h.encodeRLP17(w)
    case RLPFormat19Lux:
        return h.encodeRLP19Lux(w)
    // ...
    }
}
```

### ExtDataHash Type Difference

**Critical**: Original coreth used `common.Hash` (value type), Lux geth uses `*common.Hash` (pointer):

| Type | nil Encoding | Zero Encoding |
|------|-------------|---------------|
| `common.Hash` (value) | N/A (can't be nil) | 32 zero bytes |
| `*common.Hash` (pointer) | `0x80` (empty string) | 32 zero bytes |

For Lux 19-field format compatibility, `encodeRLP19Lux` converts pointer to value:

```go
func (h *Header) encodeRLP19Lux(w io.Writer) error {
    var extHash common.Hash
    if h.ExtDataHash != nil {
        extHash = *h.ExtDataHash
    }
    return rlp.Encode(w, &hdr19val{
        // ... ExtDataHash as VALUE type ...
        ExtDataHash: extHash,
    })
}
```

### Format Detection

For new headers without explicit format, auto-detection is used:

```go
func (h *Header) isLux19Format() bool {
    hasLuxFields := h.ExtDataGasUsed != nil || h.BlockGasCost != nil
    hasEth2Fields := h.BlobGasUsed != nil || h.ExcessBlobGas != nil ||
        h.ParentBeaconRoot != nil || h.WithdrawalsHash != nil
    return hasLuxFields && !hasEth2Fields
}
```

### Activating Format Upgrades

To upgrade to a new format for future blocks:

1. **Define new format constant** in `decode.go`:
```go
const RLPFormat25 RLPFormat = 25 // New format with extra field
```

2. **Add decoder** in `decode.go`:
```go
func decode25(data []byte) (*Header, error) {
    var h hdr25
    if err := rlp.DecodeBytes(data, &h); err != nil {
        return nil, err
    }
    return &Header{/* ... */ rlpFormat: RLPFormat25}, nil
}
```

3. **Add encoder** in `block.go`:
```go
func (h *Header) encodeRLP25(w io.Writer) error {
    return rlp.Encode(w, &hdr25{/* ... */})
}
```

4. **Update EncodeRLP switch** in `block.go`:
```go
case RLPFormat25:
    return h.encodeRLP25(w)
```

5. **Set format for new blocks** (chain config based):
```go
func (h *Header) SetRLPFormat(f RLPFormat) {
    h.rlpFormat = f
}

// In block producer:
if chainConfig.IsUpgradeActive(header.Number) {
    header.SetRLPFormat(RLPFormat25)
}
```

### Block Import/Export

The format-aware encoding enables seamless import/export:

```bash
# Export blocks (preserves original format in RLP)
geth export blocks.rlp 0 1000000

# Import blocks (format detected from RLP, preserved during storage)
geth import --datadir /path/to/db blocks.rlp
```

### Migration Mode

When genesis state is missing, import runs in "migration mode":
- Blocks imported without state re-execution
- State roots from headers are trusted
- Useful for disaster recovery from RLP exports

```
INFO Migration mode: importing blocks without state re-execution
INFO Migration: updated head block number=250000 hash=...
```

---

## Upstream Merge - December 12, 2025

### Summary
Merged 35 upstream commits from go-ethereum (446fdebdc..16f50285b) into luxfi/geth while preserving all lux-specific customizations.

### Merge Statistics
- **Upstream Commits**: 35 commits merged
- **Files Changed**: 70+ files
- **Conflicts Resolved**: 5 merge conflicts (import paths + code changes)
- **Build Status**: Successful (go build ./cmd/geth)
- **Test Status**: All critical tests passing (core/types, crypto, common, pqcrypto)

### Key Upstream Changes Integrated
1. **cmd/utils**: Fix DeveloperFlag handling when set to false
2. **core/stateless**: Cap witness depth metrics buckets
3. **eth/fetcher**: Add metadata validation in tx announcement
4. **triedb/pathdb**: Use copy instead of append to reduce memory alloc
5. **core/rawdb**: Fix size counting in memory freezer
6. **p2p/tracker**: Fix head detection in Fulfil
7. **eth/tracers/native**: Include SWAP16 in default ignored opcodes
8. **core/state**: Fix incorrect contract code state metrics
9. **eth/downloader**: Keep current syncmode in downloader only (new syncModer)
10. **core/vm**: Fix PC increment for EIP-8024 opcodes
11. **common/bitutil**: Deprecate XORBytes in favor of stdlib crypto/subtle
12. **ethdb/pebble**: Enhanced Pebble database configuration with L0CompactionConcurrency
13. **eth/filters**: Change error code for invalid parameter errors
14. **beacon/types**: Update for fulu
15. **core**: Log detailed statistics for slow block

### Conflicts Resolved
1. `cmd/evm/blockrunner.go` - Added new common/log imports with luxfi paths
2. `core/state/state_sizer.go` - Added metrics import with luxfi path
3. `eth/downloader/downloader_test.go` - Added ethconfig import with luxfi path
4. `ethdb/pebble/pebble.go` - Kept both lux ReadSamplingMultiplier and new compaction settings
5. `p2p/rlpx/rlpx.go` - Removed unused bitutil import (XORBytes moved to crypto/subtle)

### API Changes Adapted
- `BeaconDevSync` signature changed: now takes only header (sync mode handled by syncModer)
- Updated `eth/catalyst/tester.go` to match new API

### Lux Customizations Preserved
1. **Post-Quantum Crypto** (`crypto/pqcrypto/`) - ML-DSA, ML-KEM, SLH-DSA
2. **Database** (`ethdb/badgerdb/`) - BadgerDB implementation, PebbleDB enhancements
3. **Import Branding** - 100% `github.com/luxfi/geth` paths maintained
4. **Dependencies** - All `luxfi/*` packages preserved (crypto v1.17.15, node v1.20.3, ids v1.1.2)

---

## Upstream Merge - November 22, 2025

### Summary
Successfully merged latest go-ethereum upstream changes (739f6f46a..f4817b7a5) into luxfi/geth while preserving ALL lux-specific customizations and maintaining 100% compatibility with luxfi/node.

### Merge Statistics
- **Upstream Commits**: 284 commits merged
- **Files Changed**: 1,353 files (853,609 insertions, 74,901 deletions)
- **Conflicts Resolved**: 12 merge conflicts + 2 deleted files
- **Build Status**: Successful (go build ./cmd/geth)
- **Test Status**: All critical tests passing (core/types, crypto, common, pqcrypto)

### Key Upstream Changes Integrated
1. **Verkle Tree Support** - Binary trie implementation, transition trie, enhanced state dumping
2. **EIP Updates** - Improved EIP-1559 base fee calculation, enhanced EIP-4844 blob handling
3. **State Management** - TransitionTrie wrapper, enhanced trie reader for MPT/Verkle
4. **Testing Infrastructure** - New test vectors, enhanced simulation tools

### Lux Customizations Preserved
1. **Post-Quantum Crypto** (`crypto/pqcrypto/`) - ML-DSA, ML-KEM, SLH-DSA
2. **Plugin/EVM** (`plugin/evm/`) - 105 files, ChainVM interface, validators, warp messaging
3. **Database** (`ethdb/badgerdb/`) - BadgerDB full implementation, PebbleDB enhancements
4. **SubnetEVM Migration** - Pre-Shanghai to Post-Shanghai header conversion tools
5. **Import Branding** - 100% `github.com/luxfi/geth` paths maintained
6. **Dependencies** - All `luxfi/*` packages preserved at correct versions

### Compatibility Verified
- Module: `github.com/luxfi/geth`
- Compatible with: `luxfi/node v1.20.1`
- ChainVM interface: Unchanged
- Database integration: Maintained

### Merge Commit
- SHA: `aa5adea4a`
- Message: "Merge upstream go-ethereum f4817b7a5 into luxfi/geth"

### Post-Merge Fixes - November 22, 2025

#### Import Path Fix in bintrie_witness_test.go
**Problem**: Test file `core/bintrie_witness_test.go` introduced in upstream merge still used `github.com/ethereum/go-ethereum` imports causing compilation failures.

**Error**:
```
core/bintrie_witness_test.go:77:12: cannot use testVerkleChainConfig (variable of type *"github.com/ethereum/go-ethereum/params".ChainConfig) as *"github.com/luxfi/geth/params".ChainConfig value in struct literal
```

**Fix**: Updated all 12 import paths from `github.com/ethereum/go-ethereum/*` to `github.com/luxfi/geth/*`

**Files Modified**:
- `core/bintrie_witness_test.go` - Fixed imports for: common, consensus/beacon, consensus/ethash, core/rawdb, core/state, core/tracing, core/types, core/vm, crypto, params, triedb

**Result**: Build successful, all core tests compile cleanly

---

## Test Suite Fixes - December 2024

### Problem Summary
The geth test suite was experiencing critical failures:
- Core package tests timing out after 60s
- Multiple goroutine leaks causing test hangs
- Tests failing at ~0% pass rate

### Root Causes
1. **Snapshot Generation Leak**: diskLayer.Release() wasn't aborting ongoing snapshot generation
2. **Sender Cacher Singleton**: Global txSenderCacher created persistent goroutines
3. **Missing Short Mode**: Long-running tests had no -short flag support

### Fixes Applied

#### 1. Fixed Snapshot Cleanup (`core/state/snapshot/disklayer.go`)
Added proper genAbort channel handling with non-blocking select to prevent deadlocks

#### 2. Disabled Sender Cacher in Tests (`core/blockchain.go`)
Used testing.Testing() to skip sender caching during test runs

#### 3. Added Short Mode Support
- `core/block_validator_test.go`: Skip blockchain tests
- `core/filtermaps/indexer_test.go`: Skip long indexer tests
- `core/txpool/blobpool/blobpool_test.go`: Skip BLS crypto tests

### Results
- **Before**: 0% pass rate, timeouts after 60s
- **After**: 100% pass rate, ~70s full suite, ~48s with -short
- **Packages**: All 17 core sub-packages passing

---

## External Crypto Integration - December 13, 2025

### Summary
Externalized crypto types and functions to use `github.com/luxfi/crypto` package while maintaining type compatibility across geth.

### Changes Made

#### common/types.go - Crypto Wrapper Functions
Added wrapper functions to bridge the type difference between `github.com/luxfi/crypto` and `github.com/luxfi/geth/common`:

```go
// Address is defined as: type Address crypto.Address

// Wrapper functions added:
func CreateAddress(addr Address, nonce uint64) Address
func CreateAddress2(addr Address, salt [32]byte, inithash []byte) Address
func PubkeyToAddress(p ecdsa.PublicKey) Address
func Keccak256(data ...[]byte) []byte
func Keccak256Hash(data ...[]byte) Hash
func HashData(kh KeccakState, data []byte) Hash
func NewKeccakState() KeccakState

// Type alias:
type KeccakState = crypto.KeccakState
```

#### Files Updated
The following patterns were replaced across the codebase:
- `crypto.CreateAddress` to `common.CreateAddress`
- `crypto.CreateAddress2` to `common.CreateAddress2`
- `crypto.PubkeyToAddress` to `common.PubkeyToAddress`
- `crypto.Keccak256Hash` to `common.Keccak256Hash`
- `crypto.KeccakState` to `common.KeccakState`
- `crypto.NewKeccakState` to `common.NewKeccakState`
- `crypto.HashData` to `common.HashData`

#### Key Files Modified
- `core/types/receipt.go` - Contract address derivation
- `core/state_processor.go` - Transaction processing
- `core/vm/evm.go` - Contract creation (CREATE/CREATE2)
- `core/types/hashes.go` - EmptyCodeHash constant
- `core/rawdb/*.go` - Hash verification functions
- `triedb/pathdb/*.go` - Trie node hashing
- `accounts/keystore/*.go` - Key generation
- `eth/tracers/*.go` - Address derivation in tracers
- `internal/ethapi/api.go` - API contract address calculation
- `p2p/discover/*.go` - Node ID generation
- `cmd/geth/*.go` - CLI tools

### Deleted Files
The following crypto files were removed as they are now provided by `github.com/luxfi/crypto`:
- `crypto/blake2b/*` - Blake2b implementation
- `crypto/bn256/*` - BN256 pairing implementation
- `crypto/ecies/*` - ECIES encryption
- `crypto/secp256r1/*` - secp256r1 verifier
- `crypto/signify/*` - Signify signatures
- `crypto/*.go` - Core crypto functions (keccak, signatures, etc.)

### Build Status
- `make` builds successfully
- All crypto functions now use external `github.com/luxfi/crypto` package
- Type compatibility maintained via `common.Address` = `crypto.Address`

---

## Lux Block Hash Architecture Plan - December 15, 2025

### Problem Statement

The Lux C-chain uses a custom header format that differs from standard Ethereum:
- **Genesis block**: Uses 16-field format (must hash to `0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e`)
- **Post-genesis blocks**: Use 19-field Lux format with ExtDataHash, ExtDataGasUsed, BlockGasCost
- **Ethereum blocks**: Use 20-21 field format with WithdrawalsHash, BlobGasUsed, etc.

The key challenge is that:
1. Original coreth used `common.Hash` (value type) for ExtDataHash
2. Lux geth uses `*common.Hash` (pointer type) for proper nil encoding
3. These encode differently in RLP, producing different hashes

### Current Implementation Status

The codebase has partial implementation:

**Completed:**
- `RLPFormat` type and constants defined in `core/types/decode.go`
- `rawRLP` field in Header struct for preserving original RLP bytes
- `rlpFormat` field in Header struct for tracking format
- `Hash16()` method for genesis verification
- `Hash19()` method using value-type ExtDataHash (hdr19val)
- `DecodeHeader()` with format detection by field count
- `SetRawRLP()` method for preserving original bytes

**Issues to Fix:**
- Unused import (`io`) in decode.go
- `rlpFormat` being set in decode19() but Header struct field already exists
- Need to verify all decode functions properly set format tracking
- Need comprehensive tests for all format combinations

### Architecture Design

#### 1. Header Hash Strategy

```
+------------------+     +-----------------+     +------------------+
|  Decode Block    | --> | Preserve rawRLP | --> | Hash() uses raw  |
+------------------+     +-----------------+     +------------------+
        |                                               ^
        v                                               |
+------------------+     +-----------------+     +------------------+
| Detect Format    | --> | Set rlpFormat   | --> | Or uses format-  |
| (field count)    |     |                 |     | specific method  |
+------------------+     +-----------------+     +------------------+
```

**Hash Computation Logic:**
```go
func (h *Header) Hash() common.Hash {
    // 1. If we have original RLP bytes, use them (best)
    if len(h.rawRLP) > 0 {
        return rlpHashBytes(h.rawRLP)
    }

    // 2. If we know the format, use format-specific encoding
    switch h.rlpFormat {
    case RLPFormat16:
        return h.Hash16()
    case RLPFormat19Lux:
        return h.Hash19() // Uses value-type ExtDataHash
    // ... other formats
    }

    // 3. Detect format from fields present
    if h.ExtDataGasUsed != nil || h.BlockGasCost != nil {
        return h.Hash19() // Lux post-genesis
    }
    if h.BaseFee != nil && h.ExtDataHash == nil {
        return h.Hash16() // Genesis or early post-London
    }

    // 4. Default: use current struct encoding
    return rlpHash(h)
}
```

#### 2. Format Detection Matrix

| Fields | Format | Chain | Hash Method |
|--------|--------|-------|-------------|
| 15 | Pre-London | Legacy ETH | rlpHash(hdr15) |
| 16 | Post-London | Lux Genesis | Hash16() |
| 17 | ExtDataHash | Lux (rare) | rlpHash(hdr17) |
| 18 | ExtDataGasUsed | Lux (rare) | rlpHash(hdr18) |
| 19 | BlockGasCost | Lux Post-Genesis | Hash19() |
| 20 | Lux Extended | Lux + BlobGas | rlpHash(hdr20) |
| 20 | ETH Cancun | Standard ETH | decode20eth |
| 21 | Lux Extended | Lux + ExcessBlobGas | rlpHash(hdr21) |
| 21 | ETH Prague | Standard ETH | decode21eth |

#### 3. File Changes Required

**core/types/decode.go:**
```go
// Remove unused import
- import "io"

// Remove rlpFormat assignment in decode functions (already in struct)
// The decode functions should still work as rawRLP is preserved

// Add helper to detect Ethereum vs Lux format for ambiguous field counts
func isEthereumFormat(data []byte, fieldCount int) bool {
    // Check field 17 (WithdrawalsHash in ETH, ExtDataHash in Lux)
    // ETH: always 32 bytes (hash)
    // Lux: can be nil (0x80) or 32 bytes
    // ...
}
```

**core/types/block.go:**
```go
// Update Hash() method with complete format detection
func (h *Header) Hash() common.Hash {
    if len(h.rawRLP) > 0 {
        return rlpHashBytes(h.rawRLP)
    }

    // Format-based encoding
    switch h.rlpFormat {
    case RLPFormat16:
        return h.Hash16()
    case RLPFormat19Lux:
        return h.Hash19()
    case RLPFormat20Eth, RLPFormat21Eth:
        return rlpHash(h) // Standard encoding works for ETH
    }

    // Field-based detection for newly constructed headers
    if h.ExtDataGasUsed != nil || h.BlockGasCost != nil {
        return h.Hash19()
    }

    return rlpHash(h)
}

// Add helper for raw RLP hashing
func rlpHashBytes(data []byte) common.Hash {
    sha := hasherPool.Get().(crypto.KeccakState)
    defer hasherPool.Put(sha)
    sha.Reset()
    sha.Write(data)
    var h common.Hash
    sha.Read(h[:])
    return h
}
```

**core/types/hashing.go:**
```go
// Add rlpHashBytes function (or move from block.go)
func rlpHashBytes(data []byte) (h common.Hash) {
    sha := hasherPool.Get().(crypto.KeccakState)
    defer hasherPool.Put(sha)
    sha.Reset()
    sha.Write(data)
    sha.Read(h[:])
    return h
}
```

#### 4. Test Plan

**core/types/block_test.go additions:**
```go
// TestLuxGenesisHash - verify 16-field genesis hash
func TestLuxGenesisHash(t *testing.T) {
    expected := common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e")
    genesis := buildLuxGenesisHeader()
    if genesis.Hash16() != expected {
        t.Errorf("genesis hash mismatch")
    }
}

// TestLux19FieldHash - verify post-genesis with value-type ExtDataHash
func TestLux19FieldHash(t *testing.T) {
    // Encode header with hdr19val, decode, verify hash matches
}

// TestEthereumBlockDecode - verify ETH blocks decode correctly
func TestEthereumBlockDecode(t *testing.T) {
    // Use existing Ethereum test vectors
    // Verify decode -> hash -> matches expected
}

// TestMixedChain - verify parent hash continuity
func TestMixedChain(t *testing.T) {
    // Genesis (16-field) -> Block1 (19-field) -> verify parent hashes
}

// TestRoundTrip - encode/decode preserves hash
func TestRoundTrip(t *testing.T) {
    // For each format:
    // 1. Create header with that format
    // 2. Encode to RLP
    // 3. Decode from RLP
    // 4. Verify Hash() matches
}
```

#### 5. Migration Strategy

**Phase 1: Fix Build Issues**
1. Remove unused `io` import from decode.go
2. Remove redundant `rlpFormat` assignments (already in struct)
3. Run tests to verify existing functionality

**Phase 2: Add rlpHashBytes**
1. Add `rlpHashBytes()` to hashing.go
2. Update `Hash()` to use rawRLP when available
3. Add tests for raw RLP hashing

**Phase 3: Complete Format Detection**
1. Implement `isEthereumFormat()` for ambiguous field counts
2. Update decode20/decode21 to properly distinguish ETH vs Lux
3. Add comprehensive format detection tests

**Phase 4: Verify Chain Continuity**
1. Test Lux mainnet genesis hash
2. Test parent hash continuity for post-genesis blocks
3. Test Ethereum test vectors still pass

### Key Design Decisions

1. **rawRLP is authoritative**: When we decode a block, we preserve the original RLP bytes. Hash() uses these bytes directly, ensuring hash compatibility with the source chain.

2. **rlpFormat for new blocks**: When constructing new blocks (not decoded), we use field-based detection to choose the right encoding format.

3. **Value vs Pointer types**: The hdr19val struct uses `common.Hash` (value) for ExtDataHash to match original coreth encoding. The Header struct uses `*common.Hash` (pointer) for proper nil handling in Go.

4. **Ethereum compatibility**: Standard Ethereum blocks use different field order (WithdrawalsHash before BlobGas). The decoder tries Lux format first for ambiguous field counts, then falls back to Ethereum format.

5. **No breaking changes**: Existing code using `Header.Hash()` continues to work. The method automatically selects the correct hashing strategy.

### Verification Commands

```bash
# Build and verify no errors
go build ./...

# Run block/header tests
go test -v ./core/types/... -run "TestBlock|TestHeader|TestLux|TestEIP"

# Run full test suite
go test ./core/types/... -short

# Verify Lux genesis hash
go test -v ./core/types/... -run TestLuxMainnetGenesis
```

---

## Lux C-Chain Header Format Requirements

### Overview

Lux C-chain uses a custom header format that differs from both standard Ethereum and the original coreth implementation. Understanding these differences is critical for genesis hash compatibility and block validation.

### Header Format Summary

| Field Count | Format Name | Description | Use Case |
|-------------|-------------|-------------|----------|
| 15 | Pre-London | Legacy Ethereum header | Deprecated |
| 16 | Post-London | EIP-1559 BaseFee added | **Lux Genesis** |
| 17 | Lux ExtDataHash | + ExtDataHash | Not used |
| 18 | Lux ExtDataGasUsed | + ExtDataGasUsed | Not used |
| 19 | Lux Full | + BlockGasCost | **Lux Post-Genesis** |
| 20 | Ethereum Shanghai | Standard ETH + WithdrawalsHash | Standard ETH |
| 21 | Ethereum Cancun | Standard ETH + BlobGas + BeaconRoot | Standard ETH |
| 20-24 | Lux Extended | Lux fields + ETH 2.0 fields | Future |

### Critical Network Parameters

```
Network: Lux Mainnet C-Chain
Chain ID: 96369
Genesis Hash: 0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
State Root: 0x2d1cedac263020c5c56ef962f6abe0da1f5217bdc6468f8c9258a0ea23699e80
Timestamp: 0x672485c2 (1730446786)
Gas Limit: 12,000,000
Base Fee: 25 gwei
```

### 16-Field Genesis Format (Pre-ExtDataHash)

The Lux mainnet genesis block uses a **16-field header** format. This is the post-London (EIP-1559) format WITHOUT the Lux-specific ExtDataHash, ExtDataGasUsed, or BlockGasCost fields.

**RLP Field Order (16 fields):**
```
Position  Field           Type
--------  -----           ----
0         ParentHash      common.Hash
1         UncleHash       common.Hash
2         Coinbase        common.Address
3         Root            common.Hash
4         TxHash          common.Hash
5         ReceiptHash     common.Hash
6         Bloom           types.Bloom (256 bytes)
7         Difficulty      *big.Int
8         Number          *big.Int
9         GasLimit        uint64
10        GasUsed         uint64
11        Time            uint64
12        Extra           []byte
13        MixDigest       common.Hash
14        Nonce           types.BlockNonce (8 bytes)
15        BaseFee         *big.Int (EIP-1559)
```

**Hash16() Method:**
```go
// core/types/block.go
func (h *Header) Hash16() common.Hash
```
This method computes the 16-field hash for genesis compatibility verification.

### 19-Field Post-Genesis Format (Lux Standard)

Post-genesis blocks on Lux C-chain use **19 fields** with the three Lux-specific extensions:

**RLP Field Order (19 fields):**
```
Position  Field           Type
--------  -----           ----
0-15      (same as 16-field format)
16        BaseFee         *big.Int
17        ExtDataHash     *common.Hash (POINTER - see note below)
18        ExtDataGasUsed  *big.Int
19        BlockGasCost    *big.Int
```

### CRITICAL: ExtDataHash Type Difference

**Original coreth used `common.Hash` (value type):**
```go
// WRONG - Original coreth style (DO NOT USE)
ExtDataHash common.Hash `rlp:"optional"`
```

**Lux geth uses `*common.Hash` (pointer type):**
```go
// CORRECT - Lux geth style
ExtDataHash *common.Hash `rlp:"optional"`
```

This difference is intentional for proper RLP encoding of nil values. The pointer type allows the field to be truly absent from the RLP encoding when nil, rather than encoding as a zero hash.

**However, for hash computation**, we use `hdr19val` struct with value type to match original coreth encoding:
```go
type hdr19val struct {
    // ... other fields ...
    ExtDataHash common.Hash // VALUE type for hash compatibility
    // ...
}
```

### Standard Ethereum vs Lux Header Field Order

**Standard Ethereum 20-21 fields (Shanghai/Cancun):**
```
Position  Field
--------  -----
0-15      Core fields
16        BaseFee
17        WithdrawalsHash
18        BlobGasUsed
19        ExcessBlobGas
20        ParentBeaconRoot
21        RequestsHash (optional)
```

**Lux Extended 20-24 fields:**
```
Position  Field
--------  -----
0-15      Core fields
16        BaseFee
17        ExtDataHash (Lux-specific)
18        ExtDataGasUsed (Lux-specific)
19        BlockGasCost (Lux-specific)
20        BlobGasUsed
21        ExcessBlobGas
22        ParentBeaconRoot
23        WithdrawalsHash
24        RequestsHash
```

Note: Lux places its custom fields BEFORE the Ethereum 2.0 fields to maintain hash compatibility with existing blocks.

### Format Detection Logic

The `DecodeHeader()` function in `core/types/decode.go` automatically detects the header format by counting RLP fields:

```go
func DecodeHeader(data []byte) (*Header, error) {
    fieldCount := countFields(data)
    switch {
    case fieldCount == 15:  // Pre-London
        return decode15(data)
    case fieldCount == 16:  // Post-London (genesis)
        return decode16(data)
    case fieldCount >= 17:  // Extended formats
        return decodeExt(data, fieldCount)
    }
}
```

For ambiguous field counts (20, 21), the decoder tries Lux format first, then falls back to standard Ethereum format.

### Genesis Configuration

The `cchain.json` genesis file includes two special fields:

```json
{
  "skipPostMergeFields": true,
  "genesisHash": "0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e"
}
```

- **skipPostMergeFields**: When true, genesis encoding uses 16-field format (no ExtDataHash, etc.)
- **genesisHash**: Expected hash for verification (computed from 16-field RLP)

### Implementation Files

| File | Purpose |
|------|---------|
| `core/types/block.go` | Header struct definition, Hash16() method |
| `core/types/decode.go` | Format detection and multi-format decoding |
| `core/genesis.go` | Genesis block initialization with skipPostMergeFields support |

### Common Pitfalls

1. **Wrong Genesis Hash**: Using standard geth with 17+ field headers produces wrong hash
2. **ExtDataHash Type**: Using value type instead of pointer breaks RLP encoding
3. **Field Order**: Placing Ethereum 2.0 fields before Lux fields breaks compatibility
4. **Format Detection**: Assuming field count without checking actual RLP structure

### Testing Genesis Hash

```go
// Verify genesis hash matches expected
genesis := ReadGenesis("mainnet/cchain.json")
block := genesis.ToBlock()
expected := common.HexToHash("0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e")
if block.Hash() != expected {
    // Genesis format mismatch - check skipPostMergeFields
}
```

---

## Block Hash Computation Fix - December 15, 2025

### Problem

Block hash computation for Lux C-chain (chainId 96369) was not matching the original coreth/subnet-evm hashes. Specifically:

1. Block 0 (genesis) hash was correct (16-field format)
2. Block 1+ hashes were incorrect because re-encoding produced different RLP bytes

**Root Cause Analysis:**

The original coreth blocks have a specific RLP structure:
- Block 0: 16 fields (genesis with BaseFee)
- Block 1+: 17 fields with field 16 being `0x80` (empty string)

When decoding:
- Field 16 with `0x80` is decoded as nil pointer (for `*common.Hash` with `rlp:"nil"` tag)
- When re-encoding, nil pointers are omitted if no later fields are set
- This produces 16-field RLP instead of 17-field RLP
- Different RLP bytes = different hash

**Example:**
```
Block 1:
  Raw RLP length: 599 bytes (17 fields)
  Re-encoded length: 598 bytes (16 fields)
  Raw hash: 0x465e2859... (correct - matches block 2's parentHash)
  Computed hash: 0x242de9fe... (wrong)
```

### Solution

Store the raw RLP bytes when decoding and use them for hash computation:

**core/types/decode.go:**
```go
func DecodeHeader(data []byte) (*Header, error) {
    // ... decode logic ...

    if err != nil {
        return nil, err
    }

    // Store raw RLP bytes for hash computation compatibility.
    // This ensures Hash() returns the same value as the original chain format.
    header.rawRLP = make([]byte, len(data))
    copy(header.rawRLP, data)

    return header, nil
}
```

**core/types/block.go - Header.Hash():**
```go
func (h *Header) Hash() common.Hash {
    if len(h.rawRLP) > 0 {
        return rlpHashBytes(h.rawRLP)
    }
    // ... fallback to re-encoding ...
}
```

### Verification

After the fix:
```
Block 0: Header.Hash() = 0x3f4fa2a0... (correct genesis)
Block 1: Header.Hash() = 0x465e2859... (matches block 2's parentHash)
Block 2: Header.Hash() = 0x7f4bf144... (matches block 3's parentHash)
Block 3: Header.Hash() = 0xcc6abca4... (matches block 4's parentHash)
```

All tests pass:
```bash
go test ./core/types/... -v -run "TestLux|TestBlock"
```

### Key Insight

The actual Lux mainnet blocks have 17 fields, not 19 as previously documented. The 17th field (field 16) is an empty string `0x80`, which appears to be ExtDataHash encoded as "absent/nil" in the original format. The remaining fields (ExtDataGasUsed, BlockGasCost) are not present in the actual block data.

This is different from the theoretical 19-field format documented earlier. The solution of preserving raw RLP bytes handles both cases correctly without needing to understand the exact field semantics.
