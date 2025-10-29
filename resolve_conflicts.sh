#!/bin/bash

# Script to resolve merge conflicts preserving luxfi imports

echo "Resolving merge conflicts..."

# Function to replace ethereum imports with luxfi
fix_imports() {
    local file=$1
    echo "Fixing imports in $file"

    # Replace ethereum imports with luxfi
    sed -i 's|"github.com/ethereum/go-ethereum/|"github.com/luxfi/geth/|g' "$file"
    sed -i 's|"github.com/luxfi/|"github.com/luxfi/|g' "$file"
}

# Resolve specific conflicts

# 1. Build files - keep upstream versions but will fix imports
echo "Resolving build files..."
git checkout --theirs build/checksums.txt build/ci.go
fix_imports build/ci.go

# 2. Fix core/types/state_types.go vs build/tools/tools.go rename conflict
# Keep our state_types.go, restore tools.go from upstream
echo "Resolving rename conflict..."
git rm build/tools/tools.go
git checkout --theirs core/rawdb/database_tablewriter_unix.go
git add core/types/state_types.go

# 3. Core conflicts - resolve carefully
echo "Resolving core conflicts..."

# For files where we mainly need import fixes
for file in cmd/evm/internal/t8ntool/execution.go \
            cmd/geth/config.go \
            core/rawdb/accessors_trie.go \
            core/rawdb/database.go \
            core/state/reader.go \
            core/state/statedb.go \
            core/state/statedb_hooked.go \
            core/txpool/blobpool/blobpool_test.go \
            core/txpool/blobpool/limbo.go \
            core/txpool/validation.go \
            core/types/transaction.go \
            core/types/transaction_test.go \
            core/vm/contracts.go \
            core/vm/gas_table_test.go \
            core/vm/interpreter_test.go \
            core/vm/runtime/runtime.go \
            eth/catalyst/api_test.go \
            eth/handler.go \
            eth/handler_test.go \
            eth/peerset.go \
            internal/ethapi/api.go \
            internal/ethapi/api_test.go \
            trie/committer.go \
            trie/proof.go \
            trie/tracer.go \
            trie/trie.go \
            trie/trienode/node_test.go \
            trie/verkle.go \
            triedb/pathdb/database.go \
            triedb/pathdb/history.go \
            triedb/pathdb/history_inspect.go \
            triedb/pathdb/history_reader_test.go \
            triedb/pathdb/layertree.go \
            triedb/pathdb/layertree_test.go; do
    if [ -f "$file" ]; then
        echo "Processing $file..."
        # For most files, take upstream version and fix imports
        git checkout --theirs "$file" 2>/dev/null || true
        fix_imports "$file"
    fi
done

# Handle special cases with manual resolution needed
echo "Handling special cases..."

# For CHANGELOG and test files, just take upstream
git checkout --theirs core/tracing/CHANGELOG.md 2>/dev/null || true
git checkout --theirs core/tracing/journal_test.go 2>/dev/null || true

# Crypto files - remove as we use luxfi/crypto
echo "Handling crypto files..."
git rm -f crypto/bn256/cloudflare/gfp_decl.go 2>/dev/null || true
git rm -f crypto/bn256/google/gfp_decl.go 2>/dev/null || true
git rm -f crypto/secp256k1/curve.go 2>/dev/null || true
git rm -f crypto/secp256k1/scalar_mult_cgo.go 2>/dev/null || true

echo "Done with automated resolution. Manual review needed for go.mod and go.sum"