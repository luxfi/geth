#!/bin/bash

# This script reverts standard ethereum imports back from luxfi/geth to ethereum/go-ethereum
# These are packages that come from the original ethereum module, not our custom packages

echo "Reverting standard ethereum imports..."

# List of standard ethereum packages that should NOT be changed to luxfi/geth
STANDARD_PACKAGES=(
    "accounts"
    "accounts/abi"
    "accounts/abi/bind"
    "accounts/abi/bind/backends"
    "common"
    "common/bitutil"
    "common/compiler"
    "common/hexutil"
    "common/math"
    "common/mclock"
    "common/prque"
    "core"
    "core/rawdb"
    "core/state"
    "core/tracing"
    "core/types"
    "core/vm"
    "crypto"
    "crypto/blake2b"
    "crypto/bn256"
    "crypto/bls12381"
    "crypto/kzg4844"
    "crypto/secp256k1"
    "ethclient"
    "ethclient/simulated"
    "ethdb"
    "event"
    "interfaces"
    "log"
    "metrics"
    "node"
    "p2p"
    "params"
    "rlp"
    "rpc"
    "trie"
    "trie/triedb"
    "trie/trienode"
    "triedb"
)

# Revert these imports back to ethereum/go-ethereum
for pkg in "${STANDARD_PACKAGES[@]}"; do
    echo "Reverting imports for $pkg..."
    find . -name "*.go" -type f -exec sed -i "s|\"github.com/luxfi/geth/${pkg}\"|\"github.com/ethereum/go-ethereum/${pkg}\"|g" {} +
done

echo "Standard imports reverted!"