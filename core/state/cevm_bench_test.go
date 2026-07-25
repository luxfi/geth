// BenchmarkStateLayerTransfers is the Go side of the cevm-vs-geth state-layer
// comparison. It mirrors luxcpp/cevm's lib/evm/state/bench_state.cpp exactly so
// the two numbers are directly comparable on the same machine:
//
//	workload : N transfers, sender(i) -> recipient(i+N), 1 wei, 21000 gas @ price 1
//	setup    : pre-fund every sender with gas_limit*gas_price + value, commit (untimed)
//	timed    : apply all N transfers, then compute the state root
//
// The C++ bench used to print a HARDCODED "luxfi/geth StateDB (Go): 62.31 ms"
// reference measured on some other machine at some other time. That number
// rots silently. This benchmark exists so the Go side is re-measured on the
// same box, in the same session, as the C++ side.
//
//	go test ./core/state/ -run=NONE -bench=StateLayerTransfers -benchtime=5x
package state

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/tracing"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/triedb"
)

// benchAddr derives the same address bench_state.cpp does: the low 3 bytes of
// the index land in bytes 19, 18, 17 of a 20-byte address.
func benchAddr(i int) common.Address {
	var a common.Address
	a[19] = byte(i & 0xFF)
	a[18] = byte((i >> 8) & 0xFF)
	a[17] = byte((i >> 16) & 0xFF)
	return a
}

func BenchmarkStateLayerTransfers(b *testing.B) {
	const (
		numTxs   = 10000
		gasLimit = 21000
		gasPrice = 1
		value    = 1
	)
	coinbase := common.Address{19: 0xFF}
	funding := uint256.NewInt(gasLimit*gasPrice + value)
	transfer := uint256.NewInt(value)
	fee := uint256.NewInt(gasLimit * gasPrice)

	for b.Loop() {
		b.StopTimer()
		sdb, err := New(types.EmptyRootHash,
			NewDatabase(triedb.NewDatabase(rawdb.NewMemoryDatabase(), nil), nil))
		if err != nil {
			b.Fatal(err)
		}
		// Pre-fund senders and commit, mirroring setup_state() — untimed.
		for i := 0; i < numTxs; i++ {
			sdb.SetBalance(benchAddr(i), funding, tracing.BalanceChangeUnspecified)
		}
		sdb.IntermediateRoot(false)
		b.StartTimer()

		// Timed: the state transitions plus the root, i.e. what process_block()
		// reports as execution_time_ms on the C++ side.
		for i := 0; i < numTxs; i++ {
			from, to := benchAddr(i), benchAddr(i+numTxs)
			sdb.SubBalance(from, fee, tracing.BalanceChangeUnspecified)
			sdb.SubBalance(from, transfer, tracing.BalanceChangeUnspecified)
			sdb.SetNonce(from, 1, tracing.NonceChangeUnspecified)
			sdb.AddBalance(to, transfer, tracing.BalanceChangeUnspecified)
			sdb.AddBalance(coinbase, fee, tracing.BalanceChangeUnspecified)
		}
		root := sdb.IntermediateRoot(false)
		if root == (common.Hash{}) {
			b.Fatal("empty state root")
		}
	}
	b.ReportMetric(float64(numTxs*gasLimit)/1e6, "Mgas/op")
}
