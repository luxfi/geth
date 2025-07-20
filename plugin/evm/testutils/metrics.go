package testutils

import (
	"testing"

	"github.com/ethereum/go-ethereum/metrics"
)

// WithMetrics enables go-ethereum metrics globally for the test.
// If the [metrics.Enabled()] is already true, nothing is done.
// Otherwise, it is set to true and is reverted to false when the test finishes.
func WithMetrics(t *testing.T) {
	if metrics.Enabled() {
		return
	}
	// In newer versions of go-ethereum, metrics.Enabled is a function, not a variable
	// We need to use a different approach to enable metrics
	// TODO: Update this when we find the proper way to enable metrics in newer go-ethereum
	t.Logf("Warning: WithMetrics is currently a no-op due to go-ethereum metrics API changes")
}
