// (c) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package utils

import (
	"math/big"
)

// IsBlockForked returns whether a fork scheduled at block s is active at the given head block.
// Note: [s] and [head] can be either a block number or a block timestamp.
func IsBlockForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

// IsTimestampForked returns whether a fork scheduled at timestamp s is active
// at the given head timestamp. Whilst this method is the same as isBlockForked,
// they are explicitly separate for clearer reading.
func IsTimestampForked(s *uint64, head uint64) bool {
	if s == nil {
		return false
	}
	return *s <= head
}

// IsForkTransition returns true if [fork] activates during the transition from
// [parent] to [current].
// Taking [parent] as a pointer allows for us to pass nil when checking forks
// that activate during genesis.
// Note: this works for both block number and timestamp activated forks.
func IsForkTransition(fork *uint64, parent *uint64, current uint64) bool {
	var parentForked bool
	if parent != nil {
		parentForked = IsTimestampForked(fork, *parent)
	}
	currentForked := IsTimestampForked(fork, current)
	return !parentForked && currentForked
}

