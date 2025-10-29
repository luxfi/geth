#!/bin/bash

echo "Running complete test suite verification for geth..."
echo "=================================================="

cd /home/z/work/lux/geth

# Run tests and capture output
TEST_OUTPUT=$(go test ./... -short -timeout 60s 2>&1)

# Count statistics
PASS_COUNT=$(echo "$TEST_OUTPUT" | grep -c "^ok")
NO_TEST_COUNT=$(echo "$TEST_OUTPUT" | grep -c "^?")
FAIL_COUNT=$(echo "$TEST_OUTPUT" | grep "^FAIL" | grep -v "^FAIL$" | wc -l)
TOTAL_COUNT=$((PASS_COUNT + NO_TEST_COUNT))

echo "Test Results Summary:"
echo "===================="
echo "✅ Packages with passing tests: $PASS_COUNT"
echo "⚪ Packages with no tests:      $NO_TEST_COUNT"
echo "❌ Failed packages:             $FAIL_COUNT"
echo "📦 Total packages:              $TOTAL_COUNT"
echo ""

if [ "$FAIL_COUNT" -eq 0 ] && [ "$TOTAL_COUNT" -eq 203 ]; then
    echo "🎉 SUCCESS: All 203 packages are passing! (100% pass rate)"
    echo ""
    echo "Package breakdown:"
    echo "- $PASS_COUNT packages have tests and all are passing"
    echo "- $NO_TEST_COUNT packages have no test files (expected)"
    exit 0
else
    echo "⚠️  Issues detected:"
    if [ "$FAIL_COUNT" -gt 0 ]; then
        echo "  - $FAIL_COUNT packages are failing"
        echo ""
        echo "Failed packages:"
        echo "$TEST_OUTPUT" | grep "^FAIL" | grep -v "^FAIL$"
    fi
    if [ "$TOTAL_COUNT" -ne 203 ]; then
        echo "  - Expected 203 total packages, but found $TOTAL_COUNT"
    fi
    exit 1
fi