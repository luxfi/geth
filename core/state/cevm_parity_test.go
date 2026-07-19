// TestCevmMPTParity anchors the cross-engine state-root parity: luxfi/geth's
// StateDB must produce the same root that cevm's C++ StateDB::commit() asserts
// against in luxcpp/cevm/test/state_root_parity.cpp. If this golden changes,
// update it in lockstep on both sides — a silent drift is a chain-fork risk.
package state

import (
	"math/big"
	"testing"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/tracing"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/triedb"
	"github.com/holiman/uint256"
)

func TestCevmMPTParity(t *testing.T) {
	const golden = "0x549b895566f2d74e35e415bca2997f06c94d8f454dc55ac19cf009a8a247c021"
	sdb, err := New(types.EmptyRootHash, NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil))
	if err != nil {
		t.Fatal(err)
	}
	a := func(x byte) common.Address { var v common.Address; v[19] = x; return v }
	sdb.SetBalance(a(1), uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	sdb.SetNonce(a(1), 1, tracing.NonceChangeUnspecified)
	sdb.SetBalance(a(2), uint256.NewInt(2000), tracing.BalanceChangeUnspecified)
	sdb.SetNonce(a(2), 5, tracing.NonceChangeUnspecified)
	sdb.SetState(a(2), common.BigToHash(big.NewInt(3)), common.BigToHash(big.NewInt(0x99)))
	sdb.SetCode(a(3), []byte{0x60, 0x01, 0x60, 0x02}, tracing.CodeChangeUnspecified)
	if got := sdb.IntermediateRoot(false).Hex(); got != golden {
		t.Fatalf("state root drift: got %s want %s (cevm C++ parity test will now diverge)", got, golden)
	}
}
