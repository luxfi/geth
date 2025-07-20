#!/bin/bash

# This script converts all internal imports in the geth module to use relative imports
# For example: "github.com/ethereum/go-ethereum/core" becomes "./core" or "../core" etc.

echo "Converting internal imports to relative imports..."

# First, let's find all Go files and process them
find . -name "*.go" -type f | while read -r file; do
    # Get the directory of the current file relative to the module root
    file_dir=$(dirname "$file")
    
    # Skip vendor and other special directories
    if [[ "$file_dir" == *"/vendor/"* ]] || [[ "$file_dir" == *"/.git/"* ]]; then
        continue
    fi
    
    # Create a temporary file for the modifications
    temp_file="${file}.tmp"
    
    # Process the file line by line
    while IFS= read -r line; do
        # Check if this is an import line with ethereum/go-ethereum
        if [[ "$line" =~ ^[[:space:]]*import[[:space:]]*\( ]] || [[ "$line" =~ ^[[:space:]]*import[[:space:]]+ ]]; then
            # We're in an import block, set flag
            in_import=true
        elif [[ "$line" =~ ^[[:space:]]*\) ]] && [[ "$in_import" == true ]]; then
            # End of import block
            in_import=false
        fi
        
        # Process import lines
        if [[ "$line" =~ \"github\.com/ethereum/go-ethereum/([^\"]+)\" ]]; then
            import_path="${BASH_REMATCH[1]}"
            
            # Skip if this is importing from actual ethereum/go-ethereum dependencies
            # Only convert imports for packages that exist in our module
            if [[ -d "./$import_path" ]]; then
                # Calculate relative path from current file to the imported package
                rel_path=$(realpath --relative-to="$file_dir" "./$import_path")
                
                # If the relative path doesn't start with . or .., add ./
                if [[ ! "$rel_path" =~ ^\. ]]; then
                    rel_path="./$rel_path"
                fi
                
                # Replace the import
                line="${line//\"github.com\/ethereum\/go-ethereum\/$import_path\"/\"$rel_path\"}"
            fi
        fi
        
        echo "$line"
    done < "$file" > "$temp_file"
    
    # Replace the original file with the modified one
    mv "$temp_file" "$file"
done

echo "Conversion complete!"