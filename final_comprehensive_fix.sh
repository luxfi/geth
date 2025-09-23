#!/bin/bash

# Fix all remaining errors
echo "Applying final comprehensive fixes..."

# Fix assignment mismatches
find . -name "*.go" -exec sed -i 's/_, _ = binary.Write/_ = binary.Write/g' {} \;

# Fix Pool.Put errors
find . -name "*.go" -exec sed -i 's/_ = \(.*\.Put(.*)\)/\1/g' {} \;

# Fix unexpected := statements
find . -name "*.go" -exec sed -i 's/_ = \(.*err :=\)/\1/g' {} \;
find . -name "*.go" -exec sed -i 's/_, _ = \(.*:=\)/\1/g' {} \;

# Fix unexpected return
find . -name "*.go" -exec sed -i 's/_, return/return/g' {} \;

# Fix unexpected = statements
find . -name "*.go" -exec sed -i 's/_, _ = \(.*n, err =\)/\1/g' {} \;

echo "Done!"
