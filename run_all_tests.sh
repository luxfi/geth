#!/bin/bash

echo "=== COMPREHENSIVE TEST REPORT ==="
echo ""

failed_packages=""
total=0
passed=0

# Test each package with appropriate timeout
for package in $(go list ./...); do
    ((total++))
    echo -n "Testing $package... "
    
    # Use longer timeout for certain packages
    timeout="60s"
    if [[ "$package" == *"blobpool"* ]] || [[ "$package" == *"rlpgen"* ]]; then
        timeout="180s"
    fi
    
    if go test -short -timeout $timeout $package &>/dev/null; then
        echo "✅ PASS"
        ((passed++))
    else
        echo "❌ FAIL"
        failed_packages="$failed_packages\n$package"
    fi
done

echo ""
echo "=== SUMMARY ==="
echo "Total: $total"
echo "Passed: $passed"
echo "Failed: $((total - passed))"
echo "Pass Rate: $((passed * 100 / total))%"

if [ ! -z "$failed_packages" ]; then
    echo ""
    echo "Failed packages:"
    echo -e "$failed_packages"
fi
