// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package legacy

import (
	"github.com/luxfi/geth/core/vm"
)

// PrecompiledStatefulContract represents a stateful precompiled contract for legacy compatibility
type PrecompiledStatefulContract func(env vm.PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error)

// Upgrade converts a legacy stateful precompiled contract to the new interface
func (p PrecompiledStatefulContract) Upgrade() vm.StatefulPrecompiledContract {
	return &upgradedContract{run: p}
}

type upgradedContract struct {
	run PrecompiledStatefulContract
}

func (u *upgradedContract) Run(input []byte) ([]byte, error) {
	// Legacy contracts need environment, can't run without it
	return nil, vm.ErrExecutionReverted
}

func (u *upgradedContract) RequiredGas(input []byte) uint64 {
	// Legacy contracts determine gas during execution
	return 0
}

func (u *upgradedContract) RunStateful(env vm.PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	return u.run(env, input, suppliedGas)
}