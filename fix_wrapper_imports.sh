#!/bin/bash

# Fix self-imports in wrapper packages that should import from ethereum/go-ethereum

echo "Fixing self-imports in wrapper packages..."

# List of wrapper packages that should import from ethereum/go-ethereum
WRAPPER_PACKAGES=(
    "common"
    "ethdb"
    "rlp"
    "accounts"
    "metrics"
    "params"
    "crypto"
    "event"
    "log"
    "rpc"
)

for pkg in "${WRAPPER_PACKAGES[@]}"; do
    if [ -d "$pkg" ]; then
        echo "Processing $pkg package..."
        
        # Find all .go files in the package directory
        find "$pkg" -name "*.go" -type f | while read -r file; do
            # Skip test files and vendor
            if [[ "$file" == *"_test.go" ]] || [[ "$file" == *"/vendor/"* ]]; then
                continue
            fi
            
            # Replace self-imports with ethereum imports
            sed -i "s|\"github\.com/luxfi/geth/${pkg}\"|\"github.com/ethereum/go-ethereum/${pkg}\"|g" "$file"
            
            echo "  Fixed: $file"
        done
    fi
done

echo "Running goimports to fix any import ordering issues..."
goimports -w .

echo "Running go mod tidy..."
go mod tidy

echo "Done fixing wrapper imports!"