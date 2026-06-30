// Copyright (C) 2025-2026, Lux Industries Inc. All rights reserved.
// THROWAWAY PROBE — delete after use. Isolates the mechanism that drops a
// stateful precompile's storage writes at the DEX 0x9999 address.
package vm

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/tracing"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/params"
)

type probePrecompile struct {
	token       common.Address
	doNested    bool
	nonceSentinel bool
}

func (p *probePrecompile) RequiredGas([]byte) uint64  { return 0 }
func (p *probePrecompile) Run([]byte) ([]byte, error) { return nil, nil }
func (p *probePrecompile) Name() string               { return "probe" }
func (p *probePrecompile) RunStateful(env PrecompileEnvironment, input []byte, gas uint64) ([]byte, uint64, error) {
	self := env.Addresses().Self
	sdb := env.StateDB()
	if p.nonceSentinel {
		sdb.SetNonce(self, 1, tracing.NonceChangeUnspecified) // make 0x9999 non-empty
	}
	sdb.SetState(self, common.HexToHash("0x0a"), common.HexToHash("0x01")) // pre
	if p.doNested {
		if _, _, err := env.Call(p.token, nil, 100000, big.NewInt(0)); err != nil {
			return nil, gas, err
		}
	}
	sdb.SetState(self, common.HexToHash("0x0b"), common.HexToHash("0x02")) // post
	return nil, gas, nil
}

func runProbe(t *testing.T, doNested bool, selfBalance int64, deleteEmpty bool, nonceSentinel bool) (a, b common.Hash) {
	t.Helper()
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	token := common.HexToAddress("0x00000000000000000000000000000000000000C0")
	self := common.HexToAddress("0x0000000000000000000000000000000000009999")
	origin := common.HexToAddress("0x1111111111111111111111111111111111111111")

	statedb.CreateAccount(token)
	statedb.SetCode(token, []byte{0x60, 0x01, 0x60, 0x00, 0x55, 0x00}, tracing.CodeChangeUnspecified)
	statedb.CreateAccount(self)
	if selfBalance > 0 {
		statedb.AddBalance(self, uint256.NewInt(uint64(selfBalance)), tracing.BalanceChangeUnspecified)
	}
	statedb.CreateAccount(origin)

	blockCtx := BlockContext{
		CanTransfer: func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(StateDB, common.Address, common.Address, *uint256.Int) {},
		BlockNumber: big.NewInt(1), Time: 1, Random: &common.Hash{},
	}
	evm := NewEVM(blockCtx, statedb, params.MergedTestChainConfig, Config{})
	pc := evm.precompiles
	if pc == nil {
		pc = make(map[common.Address]PrecompiledContract)
	}
	pc[self] = &probePrecompile{token: token, doNested: doNested}
	evm.SetPrecompiles(pc)

	statedb.SetTxContext(common.HexToHash("0xdead"), 0)
	if _, _, err := evm.Call(origin, self, nil, 1_000_000, uint256.NewInt(0)); err != nil {
		t.Fatalf("call errored: %v", err)
	}
	statedb.Finalise(deleteEmpty)
	statedb.IntermediateRoot(deleteEmpty)
	return statedb.GetState(self, common.HexToHash("0x0a")), statedb.GetState(self, common.HexToHash("0x0b"))
}

func TestReentryProbe_Matrix(t *testing.T) {
	cases := []struct {
		name        string
		doNested    bool
		selfBalance int64
		deleteEmpty bool
	}{
		{"nested_emptyself_delete", true, 0, true},      // ERC-20 deposit shape (live bug)
		{"nonested_emptyself_delete", false, 0, true},   // isolate: is the nested call required?
		{"nested_fundedself_delete", true, 5000, true},  // native deposit shape (gives balance)
		{"nested_emptyself_nodelete", true, 0, false},   // isolate: is it the deleteEmptyObjects flag?
	}
	for _, c := range cases {
		a, b := runProbe(t, c.doNested, c.selfBalance, c.deleteEmpty)
		persisted := a == common.HexToHash("0x01") && b == common.HexToHash("0x02")
		t.Logf("%-28s slotA=%s slotB=%s  PERSISTED=%v", c.name, a.Hex()[:6], b.Hex()[:6], persisted)
	}
}
