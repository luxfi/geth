// Copyright (C) 2026, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/params"
)

// pq_dispatch_test.go — the profile gate as seen from the four EVM
// entry points, rather than from runPrecompile directly.
//
// This is the file that would have caught the hole it exists to close.
// The gate used to live inside runPrecompile, and Call, CallCode,
// DelegateCall and StaticCall each forked on the stateful-precompile
// interface *before* reaching it. Every test drove runPrecompile
// directly, so all four bypasses were invisible: a strict-PQ chain
// refused the native bn256Pairing at 0x08 and, in the same block,
// verified a Groth16 proof through a stateful precompile doing the
// same BN254 pairing internally.
//
// So these tests call evm.Call and friends, not runPrecompile.

// declaredPrecompile is a stateful precompile that names the classical
// family its soundness reduces to. It models luxfi/precompile/zk:
// that module reaches BN254 through luxfi/crypto/bn256.PairingCheck
// inside its own Run, so gating the native op at 0x08 does nothing for
// it — only its own declaration can.
type declaredPrecompile struct {
	op  Op
	ran *bool
}

func (p *declaredPrecompile) Name() string               { return "declared" }
func (p *declaredPrecompile) RequiredGas([]byte) uint64  { return 100 }
func (p *declaredPrecompile) Run([]byte) ([]byte, error) { return nil, ErrExecutionReverted }
func (p *declaredPrecompile) PQOp() Op                   { return p.op }

func (p *declaredPrecompile) RunStateful(env PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	*p.ran = true
	return []byte{0x01}, suppliedGas - 100, nil
}

// silentPrecompile declares nothing. Every custom stateful precompile
// was this shape before the gate moved, and a newly written one is this
// shape if its author forgets. Under a constraining profile it must be
// refused, not admitted: unknown means deny.
type silentPrecompile struct {
	ran *bool
}

func (p *silentPrecompile) Name() string               { return "silent" }
func (p *silentPrecompile) RequiredGas([]byte) uint64  { return 100 }
func (p *silentPrecompile) Run([]byte) ([]byte, error) { return nil, ErrExecutionReverted }

func (p *silentPrecompile) RunStateful(env PrecompileEnvironment, input []byte, suppliedGas uint64) ([]byte, uint64, error) {
	*p.ran = true
	return []byte{0x01}, suppliedGas - 100, nil
}

// statefulAddr is where the test precompiles are installed. The value
// is arbitrary: classification is by type, not by address.
var statefulAddr = common.HexToAddress("0x0900000000000000000000000000000000000000")

// newDispatchEVM builds an EVM with real state, the supplied profile,
// and exactly one precompile installed at statefulAddr.
func newDispatchEVM(t *testing.T, profile *PQProfile, p PrecompiledContract) *EVM {
	t.Helper()
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		t.Fatalf("state.New: %v", err)
	}
	cc := *params.TestChainConfig
	cc.PQ = profile
	bc := BlockContext{
		CanTransfer: func(StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(StateDB, common.Address, common.Address, *uint256.Int) {},
		BlockNumber: big.NewInt(0),
	}
	evm := NewEVM(bc, statedb, &cc, Config{})
	evm.SetPrecompiles(PrecompiledContracts{statefulAddr: p})
	return evm
}

// entryPoint is one of the four EVM call opcodes that can reach a
// precompile. Each used to carry its own copy of the stateful fork.
type entryPoint struct {
	name string
	call func(evm *EVM, gas uint64) ([]byte, uint64, error)
}

func entryPoints() []entryPoint {
	caller := common.HexToAddress("0x1111111111111111111111111111111111111111")
	return []entryPoint{
		{"Call", func(evm *EVM, gas uint64) ([]byte, uint64, error) {
			return evm.Call(caller, statefulAddr, nil, gas, new(uint256.Int))
		}},
		{"CallCode", func(evm *EVM, gas uint64) ([]byte, uint64, error) {
			return evm.CallCode(caller, statefulAddr, nil, gas, new(uint256.Int))
		}},
		{"DelegateCall", func(evm *EVM, gas uint64) ([]byte, uint64, error) {
			return evm.DelegateCall(caller, caller, statefulAddr, nil, gas, new(uint256.Int))
		}},
		{"StaticCall", func(evm *EVM, gas uint64) ([]byte, uint64, error) {
			return evm.StaticCall(caller, statefulAddr, nil, gas)
		}},
	}
}

// TestStrictPQRefusesClassicalStatefulPrecompile is the headline case:
// a stateful precompile whose soundness rests on BN254 must be refused
// by a strict-PQ chain through every entry point, and its body must not
// run. Before the gate moved, all four admitted it.
func TestStrictPQRefusesClassicalStatefulPrecompile(t *testing.T) {
	for _, ep := range entryPoints() {
		t.Run(ep.name, func(t *testing.T) {
			var ran bool
			p := &declaredPrecompile{op: OpBn256Pairing, ran: &ran}
			evm := newDispatchEVM(t, AllForbidden(), p)

			_, _, err := ep.call(evm, 100_000)
			if !errors.Is(err, ErrBn256Forbidden) {
				t.Fatalf("want ErrBn256Forbidden, got %v", err)
			}
			if ran {
				t.Fatal("precompile body ran despite refusal")
			}
		})
	}
}

// TestNonStrictAdmitsClassicalStatefulPrecompile is the other half of
// the contract, and the one that keeps this change from being a
// consensus break: a chain that has not opted into a profile must
// behave exactly as before. Both the nil profile (the default) and the
// zero profile admit the same precompile and run its body.
func TestNonStrictAdmitsClassicalStatefulPrecompile(t *testing.T) {
	profiles := map[string]*PQProfile{
		"nil-profile":  nil,
		"zero-profile": {},
	}
	for pname, profile := range profiles {
		for _, ep := range entryPoints() {
			t.Run(pname+"/"+ep.name, func(t *testing.T) {
				var ran bool
				p := &declaredPrecompile{op: OpBn256Pairing, ran: &ran}
				evm := newDispatchEVM(t, profile, p)

				ret, _, err := ep.call(evm, 100_000)
				if err != nil {
					t.Fatalf("non-strict chain must admit; got %v", err)
				}
				if !ran {
					t.Fatal("precompile body did not run")
				}
				if len(ret) != 1 || ret[0] != 0x01 {
					t.Fatalf("want precompile output, got %x", ret)
				}
			})
		}
	}
}

// TestStrictPQRefusesUnclassifiedStatefulPrecompile pins the rule that
// makes the gate hold for precompiles nobody has classified yet: an
// undeclared stateful precompile is refused under a constraining
// profile. This is what stops the next module from re-opening the hole
// by simply not mentioning the profile.
func TestStrictPQRefusesUnclassifiedStatefulPrecompile(t *testing.T) {
	for _, ep := range entryPoints() {
		t.Run(ep.name, func(t *testing.T) {
			var ran bool
			evm := newDispatchEVM(t, AllForbidden(), &silentPrecompile{ran: &ran})

			_, _, err := ep.call(evm, 100_000)
			if !errors.Is(err, ErrUnclassifiedForbidden) {
				t.Fatalf("want ErrUnclassifiedForbidden, got %v", err)
			}
			if ran {
				t.Fatal("unclassified precompile body ran despite refusal")
			}
		})
	}
}

// TestNonStrictAdmitsUnclassifiedStatefulPrecompile asserts deny-by-
// default lives strictly inside the profile. A chain with no profile
// runs an undeclared precompile exactly as it always did — the new
// refusal narrows nothing outside an opt-in.
func TestNonStrictAdmitsUnclassifiedStatefulPrecompile(t *testing.T) {
	profiles := map[string]*PQProfile{
		"nil-profile":  nil,
		"zero-profile": {},
	}
	for pname, profile := range profiles {
		for _, ep := range entryPoints() {
			t.Run(pname+"/"+ep.name, func(t *testing.T) {
				var ran bool
				evm := newDispatchEVM(t, profile, &silentPrecompile{ran: &ran})

				if _, _, err := ep.call(evm, 100_000); err != nil {
					t.Fatalf("non-strict chain must admit unclassified; got %v", err)
				}
				if !ran {
					t.Fatal("precompile body did not run")
				}
			})
		}
	}
}

// TestDeclaredPQStatefulPrecompileSurvivesStrict asserts the gate cuts
// where the discriminator says and nowhere else: a stateful precompile
// declaring OpNone — a lattice, code or hash-based verifier — runs
// under the strict profile. A gate that refused these would take the
// post-quantum precompiles down with the classical ones.
func TestDeclaredPQStatefulPrecompileSurvivesStrict(t *testing.T) {
	for _, ep := range entryPoints() {
		t.Run(ep.name, func(t *testing.T) {
			var ran bool
			p := &declaredPrecompile{op: OpNone, ran: &ran}
			evm := newDispatchEVM(t, AllForbidden(), p)

			if _, _, err := ep.call(evm, 100_000); err != nil {
				t.Fatalf("strict-PQ must admit a declared PQ precompile; got %v", err)
			}
			if !ran {
				t.Fatal("precompile body did not run")
			}
		})
	}
}

// TestStrictPQRefusalChargesGas asserts a refusal costs what execution
// would have cost. If refusing were free, the profile would be a cheap
// oracle for probing which precompiles a chain gates, and refusal would
// be cheaper than the work it replaces.
func TestStrictPQRefusalChargesGas(t *testing.T) {
	const supplied = 100_000
	cases := []struct {
		name string
		p    PrecompiledContract
	}{
		{"declared-classical", &declaredPrecompile{op: OpBn256Pairing, ran: new(bool)}},
		{"unclassified", &silentPrecompile{ran: new(bool)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			evm := newDispatchEVM(t, AllForbidden(), c.p)
			cost := c.p.RequiredGas(nil)

			_, remaining, err := evm.runPrecompile(c.p, common.Address{}, common.Address{}, nil, supplied, false)
			if err == nil {
				t.Fatal("want refusal")
			}
			if remaining != supplied-cost {
				t.Fatalf("refusal must charge RequiredGas: remaining=%d want=%d", remaining, supplied-cost)
			}
		})
	}
}

// TestEveryPrecompileIsClassified is the rule that stops an
// unclassified precompile being *added*. Deny-by-default at runtime
// means a forgotten declaration is safe; this test means it is also
// loud, naming the map, the address and the type at CI time rather
// than leaving the chain to discover it.
//
// It covers geth's own builtins too, so an upstream merge that adds a
// precompile fails here until somebody classifies it.
func TestEveryPrecompileIsClassified(t *testing.T) {
	maps := map[string]PrecompiledContracts{
		"Homestead":  PrecompiledContractsHomestead,
		"Byzantium":  PrecompiledContractsByzantium,
		"Istanbul":   PrecompiledContractsIstanbul,
		"Berlin":     PrecompiledContractsBerlin,
		"Cancun":     PrecompiledContractsCancun,
		"Prague":     PrecompiledContractsPrague,
		"Osaka":      PrecompiledContractsOsaka,
		"Verkle":     PrecompiledContractsVerkle,
		"BLS":        PrecompiledContractsBLS,
		"P256Verify": PrecompiledContractsP256Verify,
		"Lux":        PrecompiledContractsLux,
	}
	for name, m := range maps {
		for addr, p := range m {
			if _, ok := classify(p); !ok {
				t.Errorf("%s[%s]: %T is unclassified — give it a PQOp method, "+
					"or add a builtinOp case if it is one of geth's own", name, addr, p)
			}
		}
	}
}

// TestLuxPrecompilesDeclarePostQuantum asserts every precompile geth
// bakes in declares OpNone. All eight of the LP-4200 block are lattice,
// code or hash-based, so a strict-PQ chain keeps all of them; if one
// ever grows a classical dependency, its declaration must change here
// and this test is where that gets noticed.
func TestLuxPrecompilesDeclarePostQuantum(t *testing.T) {
	for addr, p := range LuxPrecompiles() {
		d, ok := p.(Classified)
		if !ok {
			t.Errorf("%s: %T does not declare an op", addr, p)
			continue
		}
		if got := d.PQOp(); got != OpNone {
			t.Errorf("%s: declares %v, want OpNone", addr, got)
		}
	}
}

// TestNewPrecompileAdapterCarriesItsDeclaration asserts the exported
// constructor — the one way an L1 chain installs a stateful precompile
// — propagates the op it was given all the way to the gate.
//
// The op parameter is what makes an ungated precompile fail to
// COMPILE rather than fail silently: there is no adapter without a
// declaration. This test pins that the value is actually used, not
// merely accepted.
func TestNewPrecompileAdapterCarriesItsDeclaration(t *testing.T) {
	for _, op := range []Op{OpNone, OpBn256Pairing, OpEcrecover, OpBLS12381Pairing} {
		a := NewPrecompileAdapter("t", statefulAddr, nil, func([]byte) uint64 { return 1 }, op)
		d, ok := a.(Classified)
		if !ok {
			t.Fatalf("adapter must implement Classified")
		}
		if got := d.PQOp(); got != op {
			t.Errorf("adapter built with %v reports %v", op, got)
		}
		if _, known := classify(a); !known {
			t.Errorf("adapter built with %v is unclassified", op)
		}
	}
}

// TestUnclassifiedRefusalNamesTheType asserts the refusal error carries
// the offending type. An operator reading a node log needs to know
// which precompile was refused, not merely that one was.
func TestUnclassifiedRefusalNamesTheType(t *testing.T) {
	evm := newDispatchEVM(t, AllForbidden(), &silentPrecompile{ran: new(bool)})
	_, _, err := evm.runPrecompile(&silentPrecompile{ran: new(bool)}, common.Address{}, common.Address{}, nil, 100_000, false)
	if err == nil {
		t.Fatal("want refusal")
	}
	if want := "vm.silentPrecompile"; !contains(err.Error(), want) {
		t.Errorf("refusal %q must name the type %q", err, want)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
