#!/bin/bash
echo "=== FINAL COMPLETE TEST REPORT ==="
echo ""
echo "Testing ALL critical packages..."

packages=(
    "./accounts/abi"
    "./accounts/keystore"
    "./cmd/geth"
    "./consensus/..."
    "./core/vm"
    "./core/state"
    "./core/types"
    "./crypto/..."
    "./common/..."
    "./params"
    "./rpc"
    "./node"
    "./internal/ethapi"
    "./eth/catalyst"
    "./eth/downloader"
    "./eth/filters"
    "./core"
)

passed=0
failed=0

for pkg in "${packages[@]}"; do
    # Determine timeout based on package
    case "$pkg" in
        "./node") timeout="60s" ;;
        "./core") timeout="120s" ;;
        "./core/state") timeout="60s" ;;
        *) timeout="30s" ;;
    esac
    
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
echo "Total: $((passed + failed))"
echo "Pass Rate: $((passed * 100 / (passed + failed)))%"
