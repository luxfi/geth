#!/bin/bash

# CI Test Script - Runs ALL tests with NO skips
# Exit on any error
set -e

echo "======================================"
echo "       CI TEST SUITE - FULL RUN       "
echo "======================================"
echo ""
echo "Starting at: $(date)"
echo ""

# Track results
total=0
passed=0
failed=0
failed_packages=""

# Function to test a package
test_package() {
    local pkg=$1
    local timeout=$2
    ((total++))

    printf "Testing %-50s " "$pkg..."

    if go test -count=1 -timeout "$timeout" "$pkg" > /tmp/test_output_$$.log 2>&1; then
        echo "✅ PASS"
        ((passed++))
    else
        echo "❌ FAIL"
        ((failed++))
        failed_packages="${failed_packages}\n  - $pkg"
        echo "Error output for $pkg:" >> /tmp/ci_failures.log
        tail -20 /tmp/test_output_$$.log >> /tmp/ci_failures.log
        echo "---" >> /tmp/ci_failures.log
    fi
    rm -f /tmp/test_output_$$.log
}

echo "Phase 1: Core Packages"
echo "-----------------------"
test_package "./core" "180s"
test_package "./core/vm" "60s"
test_package "./core/state" "120s"
test_package "./core/types" "60s"
test_package "./core/rawdb" "60s"
test_package "./core/txpool/..." "180s"

echo ""
echo "Phase 2: Consensus & Crypto"
echo "---------------------------"
test_package "./consensus/..." "60s"
test_package "./crypto/..." "60s"

echo ""
echo "Phase 3: Accounts & Authentication"
echo "----------------------------------"
test_package "./accounts/..." "60s"
test_package "./accounts/abi/..." "60s"
test_package "./accounts/keystore" "60s"

echo ""
echo "Phase 4: Ethereum Protocol"
echo "--------------------------"
test_package "./eth" "60s"
test_package "./eth/catalyst" "60s"
test_package "./eth/downloader" "60s"
test_package "./eth/filters" "60s"
test_package "./eth/tracers/..." "60s"
test_package "./eth/protocols/..." "120s"

echo ""
echo "Phase 5: Node & Networking"
echo "--------------------------"
test_package "./node" "120s"
test_package "./p2p/..." "180s"
test_package "./rpc" "60s"

echo ""
echo "Phase 6: Commands & Tools"
echo "-------------------------"
test_package "./cmd/..." "120s"
test_package "./internal/..." "60s"

echo ""
echo "Phase 7: Common & Utils"
echo "-----------------------"
test_package "./common/..." "60s"
test_package "./params" "30s"
test_package "./log" "30s"
test_package "./metrics" "30s"
test_package "./rlp" "60s"
test_package "./trie/..." "60s"
test_package "./triedb/..." "120s"

echo ""
echo "======================================"
echo "           TEST RESULTS               "
echo "======================================"
echo ""
echo "Total packages tested: $total"
echo "Passed: $passed"
echo "Failed: $failed"
echo ""

if [ $failed -eq 0 ]; then
    echo "🎉 SUCCESS: 100% PASS RATE!"
    echo ""
    echo "All tests passed successfully."
    exit 0
else
    echo "❌ FAILURE: $(($failed * 100 / $total))% failure rate"
    echo ""
    echo "Failed packages:$failed_packages"
    echo ""
    echo "Check /tmp/ci_failures.log for details"
    exit 1
fi