#!/bin/bash

# This script updates internal imports within geth to use luxfi/geth instead of ethereum/go-ethereum
# It only updates imports for packages that exist within our geth module

echo "Updating internal imports from ethereum/go-ethereum to luxfi/geth..."

# Find all Go files and update imports
find . -name "*.go" -type f | while read -r file; do
    # Skip vendor and other special directories
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi
    
    # Create a temporary file for the modifications
    temp_file="${file}.tmp"
    
    # Process the file and replace imports
    sed -E 's|"github\.com/ethereum/go-ethereum/([^"]+)"|"github.com/luxfi/geth/\1"|g' "$file" > "$temp_file"
    
    # Now we need to check if any of these imports are for actual ethereum/go-ethereum dependencies
    # and revert those back
    while IFS= read -r line; do
        # Check if this line has a luxfi/geth import
        if [[ "$line" =~ \"github\.com/luxfi/geth/([^\"]+)\" ]]; then
            import_path="${BASH_REMATCH[1]}"
            
            # Check if this path exists in our module
            if [[ ! -d "./$import_path" ]] && [[ ! -f "./${import_path}.go" ]]; then
                # This is an external ethereum dependency, revert it back
                line="${line//\"github.com\/luxfi\/geth\/$import_path\"/\"github.com\/ethereum\/go-ethereum\/$import_path\"}"
            fi
        fi
        echo "$line"
    done < "$temp_file" > "${temp_file}.2"
    
    # Replace the original file
    mv "${temp_file}.2" "$file"
    rm -f "$temp_file"
done

echo "Internal imports updated!"