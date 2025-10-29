#!/bin/bash
# Script to help resolve merge conflicts and fix import paths

set -e

echo "Resolving merge conflicts for luxfi/geth..."

# 1. Resolve version.go - accept upstream but keep it as is
git checkout --theirs version/version.go
git add version/version.go

# 2. Resolve params/config.go - keep Lux configs, accept upstream for rest
# We'll handle this manually as it needs careful merging

# 3. For crypto/* files - always use luxfi import paths
echo "Fixing crypto package conflicts..."
for file in crypto/crypto.go crypto/bn256/bn256_fast.go crypto/kzg4844/kzg4844_test.go; do
  if [ -f "$file" ]; then
    # Checkout ours first, then we'll fix imports
    git checkout --ours "$file" || true
  fi
done

# 4. For test files that were deleted - accept deletion
if git status | grep -q "crypto/secp256k1/secp256_test.go"; then
  git rm crypto/secp256k1/secp256_test.go || true
fi

# 5. For new files from upstream - accept theirs but we'll fix imports later
for file in \
  core/txpool/blobpool/conversion.go \
  core/txpool/blobpool/conversion_test.go \
  core/txpool/validation_test.go \
  eth/tracers/native/keccak256_preimage.go \
  eth/tracers/native/keccak256_preimage_test.go \
  trie/list_hasher.go; do
  if [ -f "$file" ]; then
    git checkout --theirs "$file"
    git add "$file"
  fi
done

# 6. For go.work - we want to keep it (ours)
if [ -f "go.work" ]; then
  git checkout --ours go.work
  git add go.work
fi

echo "Phase 1 complete. Now will fix import paths..."

# Find and replace all ethereum imports with luxfi imports in conflicted files
find . -type f -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | while read file; do
  if grep -q "github.com/ethereum/go-ethereum" "$file" 2>/dev/null; then
    echo "Fixing imports in $file"
    sed -i '' 's|github.com/ethereum/go-ethereum|github.com/luxfi/geth|g' "$file"
  fi
done

echo "Import paths fixed. Check remaining conflicts with: git status"
