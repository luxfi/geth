// Package event provides wrapper types for go-ethereum's event system
package event

import (
	"github.com/luxfi/geth/event"
)

// Re-export all event types from go-ethereum
type (
	Feed = event.Feed
	Subscription = event.Subscription
	TypeMux = event.TypeMux
	TypeMuxEvent = event.TypeMuxEvent
	TypeMuxSubscription = event.TypeMuxSubscription
	SubscriptionScope = event.SubscriptionScope
)

// NewTypeMux creates a new type mux
func NewTypeMux() *TypeMux {
	return new(event.TypeMux)
}

// NewFeed creates a new event feed
func NewFeed() *Feed {
	return new(event.Feed)
}

// NewSubscription creates a new subscription
func NewSubscription(fn func(<-chan struct{}) error) Subscription {
	return event.NewSubscription(fn)
}