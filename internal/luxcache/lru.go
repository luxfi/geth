// (c) 2023, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package luxcache provides backward compatibility types for cache
// that were present in older Lux versions
package luxcache

import (
	"github.com/luxfi/node/cache/lru"
)

// LRU provides a backward compatibility type for cache.LRU
// In newer Lux versions, use cache/lru.Cache instead
type LRU[K comparable, V any] struct {
	Size  int
	cache *lru.Cache[K, V]
}

// NewLRU creates a new LRU cache with initialization
func NewLRU[K comparable, V any](size int) *LRU[K, V] {
	return &LRU[K, V]{
		Size:  size,
		cache: lru.NewCache[K, V](size),
	}
}

// Get returns the value for a key
func (l *LRU[K, V]) Get(key K) (V, bool) {
	if l.cache == nil {
		l.cache = lru.NewCache[K, V](l.Size)
	}
	return l.cache.Get(key)
}

// Put adds a key-value pair
func (l *LRU[K, V]) Put(key K, value V) {
	if l.cache == nil {
		l.cache = lru.NewCache[K, V](l.Size)
	}
	l.cache.Put(key, value)
}

// Evict removes a key
func (l *LRU[K, V]) Evict(key K) {
	if l.cache == nil {
		l.cache = lru.NewCache[K, V](l.Size)
	}
	l.cache.Evict(key)
}

// Flush removes all entries
func (l *LRU[K, V]) Flush() {
	if l.cache == nil {
		l.cache = lru.NewCache[K, V](l.Size)
	}
	l.cache.Flush()
}

// Len returns the number of elements
func (l *LRU[K, V]) Len() int {
	if l.cache == nil {
		return 0
	}
	return l.cache.Len()
}

// PortionFilled returns the fraction of cache filled
func (l *LRU[K, V]) PortionFilled() float64 {
	if l.cache == nil {
		return 0
	}
	return l.cache.PortionFilled()
}
