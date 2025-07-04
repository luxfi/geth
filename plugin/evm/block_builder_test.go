// (c) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"sync"
	"testing"
	"time"

	"github.com/luxfi/coreth/params"
	"github.com/luxfi/coreth/utils"
)

func TestBlockBuilderShutsDown(t *testing.T) {
	shutdownChan := make(chan struct{})
	wg := &sync.WaitGroup{}
	builder := &blockBuilder{
		ctx:          utils.TestSnowContext(),
		chainConfig:  params.TestChainConfig,
		shutdownChan: shutdownChan,
		shutdownWg:   wg,
	}

	builder.handleBlockBuilding()
	// Close [shutdownChan] and ensure that the wait group finishes in a reasonable
	// amount of time.
	close(shutdownChan)
	attemptAwait(t, wg, 5*time.Second)
}

func TestBlockBuilderSkipsTimerInitialization(t *testing.T) {
	shutdownChan := make(chan struct{})
	wg := &sync.WaitGroup{}
	builder := &blockBuilder{
		ctx:          utils.TestSnowContext(),
		chainConfig:  params.TestChainConfig,
		shutdownChan: shutdownChan,
		shutdownWg:   wg,
	}

	builder.handleBlockBuilding()

	if builder.buildBlockTimer == nil {
		t.Fatal("expected block timer to be non-nil")
	}

	// The wait group should finish immediately since no goroutine
	// should be created when all prices should be set from the start
	attemptAwait(t, wg, time.Millisecond)
}
