#!/bin/bash
echo "=== FINAL TEST STATUS REPORT ==="
echo ""
echo "Testing key packages..."

packages=(
    "./accounts/..."
    "./cmd/geth"
    "./consensus/..."
    "./core/vm/..."
    "./core/state/..."
    "./core/types/..."
    "./crypto/..."
    "./common/..."
    "./params"
    "./rpc"
    "./node"
    "./internal/ethapi"
    "./eth"
    "./eth/catalyst"
)

passed=0
failed=0

for pkg in "${packages[@]}"; do
    # Use longer timeout for node package
    if [ "$pkg" = "./node" ]; then
        timeout="60s"
    else
        timeout="30s"
    fi

    if go test -short -timeout $timeout $pkg &>/dev/null; then
        echo "✅ PASS: $pkg"
        ((passed++))
    else
        echo "❌ FAIL: $pkg"
        ((failed++))
    fi
done

echo ""
echo "=== SUMMARY ==="
echo "Passed: $passed"
echo "Failed: $failed"
total=$((passed + failed))
rate=$((passed * 100 / total))
echo "Pass Rate: ${rate}%"
