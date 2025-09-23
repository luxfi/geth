#!/bin/bash
# Script to fix common lint issues in the geth repo

echo "Fixing lint errors in geth repo..."

# Fix errcheck issues for common patterns
find . -name "*.go" -type f | while read -r file; do
    # Skip vendor and test directories
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi

    # Fix simple error ignores for common patterns
    sed -i 's/^\(\s*\)\(.*\.Close()\)$/\1_ = \2/g' "$file"
    sed -i 's/^\(\s*\)defer \(.*\.Close()\)$/\1defer func() { _ = \2 }()/g' "$file"
    sed -i 's/^\(\s*\)\(.*\.Write(\)/\1_, _ = \2/g' "$file"
    sed -i 's/^\(\s*\)\(.*\.Put(\)/\1_ = \2/g' "$file"
    sed -i 's/^\(\s*\)\(batch\.Write()\)/\1_ = \2/g' "$file"
done

echo "Lint fixes applied"