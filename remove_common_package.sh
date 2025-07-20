#!/bin/bash

echo "Removing common package and updating imports to use top-level geth exports..."

# First, update all imports from github.com/luxfi/geth/common to just github.com/luxfi/geth
echo "Updating imports from github.com/luxfi/geth/common to github.com/luxfi/geth..."
find . -name "*.go" -type f | while read -r file; do
    # Skip vendor and test files
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi
    
    # Replace imports - handle all variations
    sed -i 's|"github\.com/luxfi/geth/common"|"github.com/luxfi/geth"|g' "$file"
    sed -i 's|"github\.com/luxfi/geth/common/bitutil"|"github.com/luxfi/geth"|g' "$file"
    sed -i 's|"github\.com/luxfi/geth/common/compiler"|"github.com/luxfi/geth"|g' "$file"
    sed -i 's|"github\.com/luxfi/geth/common/hexutil"|"github.com/luxfi/geth"|g' "$file"
    sed -i 's|"github\.com/luxfi/geth/common/lru"|"github.com/luxfi/geth"|g' "$file"
    sed -i 's|"github\.com/luxfi/geth/common/math"|"github.com/luxfi/geth"|g' "$file"
    sed -i 's|"github\.com/luxfi/geth/common/prque"|"github.com/luxfi/geth"|g' "$file"
done

# Update import aliases if any
echo "Updating import aliases..."
find . -name "*.go" -type f | while read -r file; do
    if [[ "$file" == *"/vendor/"* ]] || [[ "$file" == *"/.git/"* ]]; then
        continue
    fi
    
    # Handle aliased imports like: common "github.com/luxfi/geth/common"
    sed -i 's|common "github\.com/luxfi/geth/common"|geth "github.com/luxfi/geth"|g' "$file"
    
    # Then update the usage in the code
    sed -i 's|\bcommon\.Hash\b|geth.Hash|g' "$file"
    sed -i 's|\bcommon\.Address\b|geth.Address|g' "$file"
    sed -i 's|\bcommon\.HexToHash\b|geth.HexToHash|g' "$file"
    sed -i 's|\bcommon\.HexToAddress\b|geth.HexToAddress|g' "$file"
    sed -i 's|\bcommon\.BytesToHash\b|geth.BytesToHash|g' "$file"
    sed -i 's|\bcommon\.BytesToAddress\b|geth.BytesToAddress|g' "$file"
    sed -i 's|\bcommon\.BigToHash\b|geth.BigToHash|g' "$file"
    sed -i 's|\bcommon\.BigToAddress\b|geth.BigToAddress|g' "$file"
    sed -i 's|\bcommon\.IsHexAddress\b|geth.IsHexAddress|g' "$file"
    sed -i 's|\bcommon\.Hex2Bytes\b|geth.Hex2Bytes|g' "$file"
    sed -i 's|\bcommon\.FromHex\b|geth.FromHex|g' "$file"
    sed -i 's|\bcommon\.CopyBytes\b|geth.CopyBytes|g' "$file"
    sed -i 's|\bcommon\.LeftPadBytes\b|geth.LeftPadBytes|g' "$file"
    sed -i 's|\bcommon\.RightPadBytes\b|geth.RightPadBytes|g' "$file"
    sed -i 's|\bcommon\.TrimLeftZeroes\b|geth.TrimLeftZeroes|g' "$file"
    sed -i 's|\bcommon\.TrimRightZeroes\b|geth.TrimRightZeroes|g' "$file"
    sed -i 's|\bcommon\.Bytes2Hex\b|geth.Bytes2Hex|g' "$file"
    sed -i 's|\bcommon\.Big0\b|geth.Big0|g' "$file"
    sed -i 's|\bcommon\.Big1\b|geth.Big1|g' "$file"
    sed -i 's|\bcommon\.Big2\b|geth.Big2|g' "$file"
    sed -i 's|\bcommon\.Big3\b|geth.Big3|g' "$file"
    sed -i 's|\bcommon\.Big32\b|geth.Big32|g' "$file"
    sed -i 's|\bcommon\.Big256\b|geth.Big256|g' "$file"
    sed -i 's|\bcommon\.Big257\b|geth.Big257|g' "$file"
    sed -i 's|\bcommon\.HashLength\b|geth.HashLength|g' "$file"
    sed -i 's|\bcommon\.AddressLength\b|geth.AddressLength|g' "$file"
    sed -i 's|\bcommon\.StorageSize\b|geth.StorageSize|g' "$file"
    sed -i 's|\bcommon\.MixedcaseAddress\b|geth.MixedcaseAddress|g' "$file"
    sed -i 's|\bcommon\.NewMixedcaseAddress\b|geth.NewMixedcaseAddress|g' "$file"
    sed -i 's|\bcommon\.PrettyDuration\b|geth.PrettyDuration|g' "$file"
    sed -i 's|\bcommon\.PrettyAge\b|geth.PrettyAge|g' "$file"
    sed -i 's|\bcommon\.PrettyBytes\b|geth.PrettyBytes|g' "$file"
    
    # Handle hexutil
    sed -i 's|\bhexutil\.Encode\b|geth.Encode|g' "$file"
    sed -i 's|\bhexutil\.EncodeBig\b|geth.EncodeBig|g' "$file"
    sed -i 's|\bhexutil\.EncodeUint64\b|geth.EncodeUint64|g' "$file"
    sed -i 's|\bhexutil\.Decode\b|geth.Decode|g' "$file"
    sed -i 's|\bhexutil\.DecodeBig\b|geth.DecodeBig|g' "$file"
    sed -i 's|\bhexutil\.DecodeUint64\b|geth.DecodeUint64|g' "$file"
    sed -i 's|\bhexutil\.MustDecode\b|geth.MustDecode|g' "$file"
    sed -i 's|\bhexutil\.MustDecodeBig\b|geth.MustDecodeBig|g' "$file"
    sed -i 's|\bhexutil\.MustDecodeUint64\b|geth.MustDecodeUint64|g' "$file"
    sed -i 's|\bhexutil\.Bytes\b|geth.HexBytes|g' "$file"
    sed -i 's|\bhexutil\.Big\b|geth.HexBig|g' "$file"
    sed -i 's|\bhexutil\.Uint\b|geth.HexUint|g' "$file"
    sed -i 's|\bhexutil\.Uint64\b|geth.HexUint64|g' "$file"
    
    # Handle math
    sed -i 's|\bmath\.BigMax\b|geth.BigMax|g' "$file"
    sed -i 's|\bmath\.BigMin\b|geth.BigMin|g' "$file"
    sed -i 's|\bmath\.BigPow\b|geth.BigPow|g' "$file"
    sed -i 's|\bmath\.HexOrDecimal\b|geth.HexOrDecimal|g' "$file"
    sed -i 's|\bmath\.U256\b|geth.U256|g' "$file"
    sed -i 's|\bmath\.U256Bytes\b|geth.U256Bytes|g' "$file"
    sed -i 's|\bmath\.S256\b|geth.S256|g' "$file"
    sed -i 's|\bmath\.PaddedBigBytes\b|geth.PaddedBigBytes|g' "$file"
    sed -i 's|\bmath\.ReadBits\b|geth.ReadBits|g' "$file"
done

# Remove the common directory
echo "Removing common directory..."
rm -rf common/

echo "Running goimports to fix any import issues..."
goimports -w .

echo "Running go mod tidy..."
go mod tidy

echo "Done! The common package has been removed and all imports updated."