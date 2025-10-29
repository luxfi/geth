#!/bin/bash

# Strip namespace converter script
# Converts SubnetEVM database to C-Chain format

SOURCE_DB="/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
TARGET_DB="/home/z/.luxd-node1/chainData/C/db-converted"

# Build the tool
echo "Building strip-namespace tool..."
cd /home/z/work/lux/geth
go build -o bin/strip-namespace ./cmd/strip-namespace || exit 1

# Run the converter
echo "Converting database (first 1000 blocks)..."
echo "Source: $SOURCE_DB"
echo "Target: $TARGET_DB"

# Create target directory if it doesn't exist
mkdir -p "$(dirname "$TARGET_DB")"

# Run conversion
./bin/strip-namespace \
    -source "$SOURCE_DB" \
    -target "$TARGET_DB" \
    -blocks 1000 \
    "$@"

echo "Conversion complete!"
echo "To test the converted database:"
echo "  1. Stop any running luxd"
echo "  2. Backup existing C-Chain database"
echo "  3. Replace with converted database"
echo "  4. Start luxd and verify"