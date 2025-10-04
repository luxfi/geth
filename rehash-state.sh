#!/bin/bash

# Rehash State Migration Tool - SubnetEVM to C-Chain
# Converts path-based trie nodes to hash-based (keccak256)

set -e

# Default paths
SRC_PATH="/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
DST_PATH="/home/z/work/lux/geth/cchain-badger-db"
NAMESPACE="337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
STATE_ROOT="0xaedd8be7a060b082b0cb3195d0b5ba017c058468851ed93dd07eca274de000c2"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --src)
            SRC_PATH="$2"
            shift 2
            ;;
        --dst)
            DST_PATH="$2"
            shift 2
            ;;
        --ns)
            NAMESPACE="$2"
            shift 2
            ;;
        --state-root)
            STATE_ROOT="$2"
            shift 2
            ;;
        --verify)
            VERIFY="--verify"
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --src PATH         Source PebbleDB path (default: $SRC_PATH)"
            echo "  --dst PATH         Destination BadgerDB path (default: $DST_PATH)"
            echo "  --ns HEX          32-byte namespace hex (default: $NAMESPACE)"
            echo "  --state-root HEX  State root hash to rebuild (default: $STATE_ROOT)"
            echo "  --verify          Verify state after migration"
            echo ""
            echo "Example:"
            echo "  $0 --verify"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if source exists
if [ ! -d "$SRC_PATH" ]; then
    echo "Error: Source path does not exist: $SRC_PATH"
    exit 1
fi

# Create destination directory if needed
mkdir -p "$(dirname "$DST_PATH")"

echo "====================================="
echo "State Rehashing Tool"
echo "====================================="
echo "Source:      $SRC_PATH"
echo "Destination: $DST_PATH"
echo "Namespace:   $NAMESPACE"
echo "State Root:  $STATE_ROOT"
echo "====================================="
echo ""

# Run the rehash-state tool
exec /home/z/work/lux/geth/bin/rehash-state \
    -src "$SRC_PATH" \
    -dst "$DST_PATH" \
    -ns "$NAMESPACE" \
    -state-root "$STATE_ROOT" \
    -tip 1082780 \
    -batch 10000 \
    $VERIFY