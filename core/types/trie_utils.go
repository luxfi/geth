// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package types

import (
	"github.com/luxfi/geth/common"
)

// TrieRootHash ensures the hash is a valid trie root
func TrieRootHash(h common.Hash) common.Hash {
	// For now, just return the hash as-is
	// TODO: Add validation if needed
	return h
}