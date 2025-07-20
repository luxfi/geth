// Package lru provides wrapper types for go-ethereum's lru implementation
package lru

import (
	"sync"
)

// BasicLRU is a simple LRU cache
type BasicLRU[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]V
	capacity int
}

// Cache is a thread-safe LRU cache
type Cache[K comparable, V any] struct {
	*BasicLRU[K, V]
}

// SizeConstrainedCache is a size-constrained cache
type SizeConstrainedCache[K comparable, V any] struct {
	*BasicLRU[K, V]
	maxSize uint64
}

// NewBasicLRU creates a new BasicLRU
func NewBasicLRU[K comparable, V any](capacity int) *BasicLRU[K, V] {
	return &BasicLRU[K, V]{
		m: make(map[K]V),
		capacity: capacity,
	}
}

// NewCache creates a new cache
func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{NewBasicLRU[K, V](capacity)}
}

// NewSizeConstrainedCache creates a new size-constrained cache
func NewSizeConstrainedCache[K comparable, V any](maxSize uint64) *SizeConstrainedCache[K, V] {
	return &SizeConstrainedCache[K, V]{
		BasicLRU: NewBasicLRU[K, V](1000), // default capacity
		maxSize: maxSize,
	}
}

// Get retrieves a value from the cache
func (c *BasicLRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok
}

// Add adds a value to the cache
func (c *BasicLRU[K, V]) Add(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = value
}

// Len returns the number of items in the cache
func (c *BasicLRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m)
