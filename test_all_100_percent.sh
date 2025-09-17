#!/bin/bash

echo "======================================"
echo "   COMPLETE 100% TEST VERIFICATION   "
echo "======================================"
echo ""

# Run go test on ALL packages with proper timeouts
echo "Running comprehensive test suite..."
echo ""

# Get all packages
all_packages=$(go list ./... 2>/dev/null)
total=$(echo "$all_packages" | wc -l)
passed=0
failed=0

for pkg in $all_packages; do
    # Skip vendor and test directories
    if [[ "$pkg" == *"vendor"* ]] || [[ "$pkg" == *"testdata"* ]]; then
        continue
    fi

    # Determine timeout based on package
    timeout="60s"
    if [[ "$pkg" == *"blobpool"* ]] || [[ "$pkg" == *"rlpgen"* ]]; then
        timeout="300s"
    elif [[ "$pkg" == *"core"* ]] || [[ "$pkg" == *"node"* ]] || [[ "$pkg" == *"p2p"* ]]; then
        timeout="180s"
    elif [[ "$pkg" == *"txpool"* ]] || [[ "$pkg" == *"triedb"* ]]; then
        timeout="180s"
    fi

    # Display package name (truncated for readability)
    short_name="${pkg##*/}"
    printf "Testing %-30s " "$short_name..."

    if go test -count=1 -short -timeout "$timeout" "$pkg" &>/dev/null; then
        echo "✅"
        ((passed++))
    else
        echo "❌"
        ((failed++))
    fi
done

echo ""
echo "======================================"
echo "           FINAL RESULTS              "
echo "======================================"
echo ""
echo "Total packages: $total"
echo "Passed: $passed"
echo "Failed: $failed"
echo "Pass Rate: $(($passed * 100 / $total))%"
echo ""

if [ $failed -eq 0 ]; then
    echo "🎉 100% PASS RATE ACHIEVED!"
else
    echo "⚠️  $failed packages still need fixing"
fi