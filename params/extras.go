// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package params

import (
	"math/big"
	"sync"
)

// Extras is a generic container for extra chain configuration data
type Extras[C any, R any] struct {
	ReuseJSONRoot bool
	NewRules      func(*ChainConfig, *Rules, C, *big.Int, bool, uint64) R
}

// ExtraPayloads provides access to extra configuration payloads
type ExtraPayloads[C any, R any] struct {
	ChainConfig ExtraPayloadAccessor[*ChainConfig, C]
	Rules       ExtraPayloadAccessor[*Rules, R]
}

// ExtraPayloadAccessor provides Get/Set access to extra payloads
type ExtraPayloadAccessor[K comparable, V any] struct {
	mu    sync.RWMutex
	store map[K]V
}

// Get retrieves the extra payload for the given key
func (a *ExtraPayloadAccessor[K, V]) Get(key K) V {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.store == nil {
		var zero V
		return zero
	}
	return a.store[key]
}

// Set stores the extra payload for the given key
func (a *ExtraPayloadAccessor[K, V]) Set(key K, value V) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.store == nil {
		a.store = make(map[K]V)
	}
	a.store[key] = value
}

// FromChainConfig extracts extra config from chain config
func (p ExtraPayloads[C, R]) FromChainConfig(c *ChainConfig) C {
	return p.ChainConfig.Get(c)
}

// FromRules extracts extra rules from rules
func (p ExtraPayloads[C, R]) FromRules(r *Rules) R {
	return p.Rules.Get(r)
}

// RegisterExtras registers extra configuration handlers
func RegisterExtras[C any, R any](e Extras[C, R]) ExtraPayloads[C, R] {
	return ExtraPayloads[C, R]{
		ChainConfig: ExtraPayloadAccessor[*ChainConfig, C]{
			store: make(map[*ChainConfig]C),
		},
		Rules: ExtraPayloadAccessor[*Rules, R]{
			store: make(map[*Rules]R),
		},
	}
}
