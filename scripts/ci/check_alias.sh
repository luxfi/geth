#!/usr/bin/env bash
# CI script to check for interface divergence
# Fails if any alias accidentally became a distinct type

set -euo pipefail

echo "Checking for interface divergence and type mismatches..."

# Run go vet to check for interface issues
if ! GOOS= GOARCH= go vet ./... 2>&1 | grep -v 'does not implement'; then
    echo "✓ No interface divergence detected"
else
    echo "✗ Found interface divergence issues:"
    GOOS= GOARCH= go vet ./... 2>&1 | grep 'does not implement' || true
    exit 1
fi

# Check for import cycles
echo "Checking for import cycles..."
if go list -f '{{.ImportPath}} {{.Imports}}' ./... | grep -E 'import cycle|circular'; then
    echo "✗ Import cycles detected"
    exit 1
else
    echo "✓ No import cycles detected"
fi

# Check for missing type aliases
echo "Checking for potential missing type aliases..."
go vet ./... 2>&1 | grep 'does not implement' | cut -d'"' -f2 | sort -u || true

echo "CI checks passed!"