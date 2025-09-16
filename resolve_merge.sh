#!/bin/bash

# Resolve merge conflicts by accepting upstream changes but keeping our import paths

# For each conflicted file
for file in $(git status --porcelain | grep "^UU" | awk '{print $2}'); do
    echo "Processing $file..."

    # Use sed to handle the merge conflicts
    # First, for imports, always use luxfi paths
    sed -i 's|"github.com/ethereum/go-ethereum|"github.com/luxfi/geth|g' "$file"

    # Remove conflict markers for now (we'll manually review important ones)
    sed -i '/^<<<<<<< HEAD$/d' "$file"
    sed -i '/^=======$/d' "$file"
    sed -i '/^>>>>>>> upstream\/master$/d' "$file"
done

echo "Basic merge conflict resolution done. Manual review needed for complex conflicts."