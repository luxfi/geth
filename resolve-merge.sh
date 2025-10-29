#!/bin/bash
set -e

echo "=== Resolving remaining merge conflicts ==="

# List of files with conflicts (excluding already resolved ones)
CONFLICT_FILES=(
  "build/ci.go"
  "cmd/utils/cmd.go"
  "core/evm.go"
  "core/state/state_object.go"
  "core/txpool/blobpool/conversion.go"
  "core/txpool/blobpool/conversion_test.go"
  "core/txpool/validation_test.go"
  "core/types/bloom9.go"
  "core/types/hashing_test.go"
  "core/types/transaction_signing_test.go"
  "crypto/bn256/bn256_fast.go"
  "crypto/crypto.go"
  "crypto/kzg4844/kzg4844_test.go"
  "eth/ethconfig/gen_config.go"
  "eth/fetcher/tx_fetcher.go"
  "eth/filters/filter.go"
  "eth/filters/filter_system_test.go"
  "eth/protocols/eth/handshake.go"
  "eth/tracers/native/keccak256_preimage.go"
  "eth/tracers/native/keccak256_preimage_test.go"
  "interfaces.go"
  "internal/ethapi/errors.go"
  "node/defaults.go"
  "params/config.go"
  "trie/list_hasher.go"
)

# Resolve conflicts by accepting upstream changes
echo "Accepting upstream changes for conflicted files..."
for file in "${CONFLICT_FILES[@]}"; do
  if [ -f "$file" ]; then
    echo "  Resolving $file"
    git checkout --theirs "$file"
    git add "$file"
  fi
done

echo ""
echo "=== Converting ethereum imports to luxfi imports ==="

# Find all Go files and replace imports
find . -name "*.go" -type f ! -path "./vendor/*" ! -path "./.git/*" | while read -r file; do
  # Replace module imports
  sed -i '' 's|github\.com/ethereum/go-ethereum|github.com/luxfi/geth|g' "$file"
done

# Also update go.mod
sed -i '' 's|github\.com/ethereum/go-ethereum|github.com/luxfi/geth|g' go.mod

# Ensure luxfi/crypto is used in crypto/signature_cgo.go
if [ -f "crypto/signature_cgo.go" ]; then
  sed -i '' 's|github\.com/ethereum/go-ethereum/crypto/secp256k1|github.com/luxfi/crypto/secp256k1|g' crypto/signature_cgo.go
fi

echo ""
echo "=== Verifying luxfi/crypto package usage ==="
grep -r "luxfi/crypto" crypto/ || echo "  luxfi/crypto package found"

echo ""
echo "=== Resolution complete! ==="
echo ""
echo "Next steps:"
echo "1. Review changes with: git diff --cached"
echo "2. Run tests: go test ./..."
echo "3. Build: go build -v ./cmd/geth"
