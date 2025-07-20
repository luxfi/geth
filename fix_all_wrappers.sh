#!/bin/bash

echo "Fixing all wrapper package imports..."

# Fix common subdirectories
echo "Fixing common/bitutil..."
if [ -f "common/bitutil/bitutil.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/common/bitutil"|"github.com/ethereum/go-ethereum/common/bitutil"|g' common/bitutil/bitutil.go
    # Add closing brace if missing
    if ! grep -q "^}$" common/bitutil/bitutil.go; then
        echo ")" >> common/bitutil/bitutil.go
    fi
fi

echo "Fixing common/compiler..."
if [ -f "common/compiler/solidity.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/common/compiler"|"github.com/ethereum/go-ethereum/common/compiler"|g' common/compiler/solidity.go
    # Add closing brace if missing
    if ! grep -q "^}$" common/compiler/solidity.go; then
        echo ")" >> common/compiler/solidity.go
    fi
fi

echo "Fixing common/math..."
if [ -f "common/math/big.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/common/math"|"github.com/ethereum/go-ethereum/common/math"|g' common/math/big.go
    # Add closing brace if missing
    if ! grep -q "^}$" common/math/big.go; then
        echo ")" >> common/math/big.go
    fi
fi

echo "Fixing common/lru..."
if [ -f "common/lru/lru.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/common/lru"|"github.com/ethereum/go-ethereum/common/lru"|g' common/lru/lru.go
    # Add closing brace if missing
    if ! grep -q "^}$" common/lru/lru.go; then
        echo "}" >> common/lru/lru.go
    fi
fi

echo "Fixing common/prque..."
if [ -f "common/prque/prque.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/common/prque"|"github.com/ethereum/go-ethereum/common/prque"|g' common/prque/prque.go
fi

echo "Fixing common/types.go..."
if [ -f "common/types.go" ]; then
    # Add closing brace if missing
    if ! grep -q "^}$" common/types.go; then
        echo "}" >> common/types.go
    fi
fi

echo "Fixing common/format.go..."
if [ -f "common/format.go" ]; then
    # This file seems to be just a stub, let's complete it
    cat > common/format.go << 'EOF'
// Package common provides formatting utilities
package common

import (
	"time"
	"github.com/ethereum/go-ethereum/common"
)

// PrettyDuration is a wrapper for time.Duration for pretty printing
type PrettyDuration = common.PrettyDuration

// PrettyAge is a wrapper for time.Time for pretty printing
type PrettyAge = common.PrettyAge  

// PrettyBytes is a wrapper for byte sizes for pretty printing
type PrettyBytes = common.PrettyBytes
EOF
fi

# Fix crypto subdirectories
echo "Fixing crypto/blake2b..."
if [ -f "crypto/blake2b/blake2b.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/crypto/blake2b"|"github.com/ethereum/go-ethereum/crypto/blake2b"|g' crypto/blake2b/blake2b.go
    # Add closing brace if missing
    if ! grep -q "^}$" crypto/blake2b/blake2b.go; then
        echo ")" >> crypto/blake2b/blake2b.go
    fi
fi

echo "Fixing crypto/bn256..."
if [ -f "crypto/bn256/bn256.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/crypto/bn256"|"github.com/ethereum/go-ethereum/crypto/bn256"|g' crypto/bn256/bn256.go
    # Add closing brace if missing
    if ! grep -q "^}$" crypto/bn256/bn256.go; then
        echo ")" >> crypto/bn256/bn256.go
    fi
fi

echo "Fixing crypto/kzg4844..."
if [ -f "crypto/kzg4844/kzg4844.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/crypto/kzg4844"|"github.com/ethereum/go-ethereum/crypto/kzg4844"|g' crypto/kzg4844/kzg4844.go
    # Add closing brace if missing
    if ! grep -q "^}$" crypto/kzg4844/kzg4844.go; then
        echo ")" >> crypto/kzg4844/kzg4844.go
    fi
fi

echo "Fixing crypto/secp256k1.go..."
if [ -f "crypto/secp256k1.go" ]; then
    # Add closing brace if missing
    if ! grep -q "^}$" crypto/secp256k1.go; then
        echo "}" >> crypto/secp256k1.go
    fi
fi

# Fix core subdirectories
echo "Fixing core/tracing..."
if [ -f "core/tracing/tracing.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/core/tracing"|"github.com/ethereum/go-ethereum/core/tracing"|g' core/tracing/tracing.go
    # Add closing brace if missing
    if ! grep -q "^}$" core/tracing/tracing.go; then
        echo ")" >> core/tracing/tracing.go
    fi
fi

echo "Fixing core/vm/errors_wrapper.go..."
if [ -f "core/vm/errors_wrapper.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/core/vm"|"github.com/ethereum/go-ethereum/core/vm"|g' core/vm/errors_wrapper.go
    # Add closing brace if missing
    if ! grep -q "^}$" core/vm/errors_wrapper.go; then
        echo ")" >> core/vm/errors_wrapper.go
    fi
fi

# Fix ethdb subdirectories  
echo "Fixing ethdb/batch_ext.go..."
if [ -f "ethdb/batch_ext.go" ]; then
    # Add closing brace if missing
    if ! grep -q "^}$" ethdb/batch_ext.go; then
        echo "}" >> ethdb/batch_ext.go
    fi
fi

# Fix event
echo "Fixing event/event.go..."
if [ -f "event/event.go" ]; then
    # Add closing brace if missing
    if ! grep -q "^}$" event/event.go; then
        echo "}" >> event/event.go
    fi
fi

# Fix luxfi
echo "Fixing luxfi/types.go..."
if [ -f "luxfi/types.go" ]; then
    sed -i 's|"github\.com/luxfi/geth/common"|"github.com/ethereum/go-ethereum/common"|g' luxfi/types.go
    # Add closing brace if missing
    if ! grep -q "^}$" luxfi/types.go; then
        echo ")" >> luxfi/types.go
    fi
fi

echo "Running goimports..."
goimports -w .

echo "Running go mod tidy..."
go mod tidy

echo "Done fixing wrapper imports!"