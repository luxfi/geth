// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package triestate provides types for trie state management
package triestate

import (
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/triedb"
)

// Set is an alias for triedb.StateSet
type Set = triedb.StateSet

// New creates a new state set
func New() *Set {
	return triedb.NewStateSet()
}