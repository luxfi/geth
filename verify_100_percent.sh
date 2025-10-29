#!/bin/bash

echo "======================================"
echo "    100% VERIFICATION TEST SUITE     "
echo "======================================"
echo ""

# Critical packages that must pass
packages=(
    "./core/vm"
    "./core/state"
    "./core/types"
    "./consensus/ethash"
    "./consensus/beacon"
    "./crypto"
    "./accounts/abi"
    "./accounts/keystore"
    "./eth/catalyst"
    "./eth/downloader"
    "./eth/filters"
    "./node"
    "./rpc"
    "./cmd/geth"
    "./internal/ethapi"
    "./common"
    "./params"
)

total=${#packages[@]}
passed=0

echo "Testing $total critical packages..."
echo ""

for pkg in "${packages[@]}"; do
    # Determine timeout
    case "$pkg" in
        "./node") timeout="120s" ;;
        "./core/state") timeout="120s" ;;
        *) timeout="60s" ;;
    esac

    printf "%-30s " "$pkg"

    if go test -count=1 -short -timeout "$timeout" "$pkg" &>/dev/null; then
        echo "✅ PASS"
        ((passed++))
    else
        echo "❌ FAIL"
    fi
done

echo ""
echo "======================================"
echo "             RESULTS                  "
echo "======================================"
echo ""
echo "Passed: $passed / $total"
echo "Pass Rate: $(($passed * 100 / $total))%"
echo ""

if [ $passed -eq $total ]; then
    echo "🎉 SUCCESS: 100% PASS RATE ACHIEVED!"
    echo "All critical packages are passing!"
    exit 0
else
    echo "⚠️  Some tests failed. Run with -v for details."
    exit 1
fi