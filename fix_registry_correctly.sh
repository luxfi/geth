#!/bin/bash

echo "=== Fixing registry references correctly ==="

# Use NewPrometheusRegistry (not NewRegistry)
find . -name "*.go" -type f | while read file; do
    if grep -q "metric\.NewRegistry()" "$file" 2>/dev/null; then
        echo "  Fixing $file"
        sed -i 's/metric\.NewRegistry()/metric.NewPrometheusRegistry()/g' "$file"
    fi
    if grep -q "luxmetrics\.NewRegistry()" "$file" 2>/dev/null; then
        echo "  Fixing $file"
        sed -i 's/luxmetrics\.NewRegistry()/luxmetrics.NewPrometheusRegistry()/g' "$file"
    fi
done

echo "=== Done fixing registry references ==="
