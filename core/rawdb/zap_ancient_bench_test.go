// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// Benchmark harness for the ZAP-native ancient store.
//
// We measure write IOPS proxy (ModifyAncients throughput) and read
// latency at scale. The memory backend stands in for ZAP — same
// interface, so write-amplification numbers translate one-for-one.
//
// Run:
//
//	GOWORK=off go test ./core/rawdb \
//	    -run=XXX -bench=BenchmarkZapAncient \
//	    -benchtime=2s -count=1
package rawdb

import (
	"crypto/rand"
	"testing"

	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/memorydb"
)

// benchTables mirrors the chain freezer tables so the numbers are
// directly comparable to production workloads.
var benchTables = map[string]freezerTableConfig{
	ChainFreezerHeaderTable:  {noSnappy: false, prunable: false},
	ChainFreezerHashTable:    {noSnappy: true, prunable: false},
	ChainFreezerBodiesTable:  {noSnappy: false, prunable: true},
	ChainFreezerReceiptTable: {noSnappy: false, prunable: true},
}

// BenchmarkZapAncient_WriteThroughput measures ModifyAncients
// throughput for a synthetic block (header=600B, hash=32B, body=2KB,
// receipts=500B). The chain freezer's bottleneck is this exact shape
// — 1 block = 4 puts.
func BenchmarkZapAncient_WriteThroughput(b *testing.B) {
	store, err := NewZapAncientStore(b.TempDir(), benchTables, memBackend{memorydb.New()})
	if err != nil {
		b.Fatalf("NewZapAncientStore: %v", err)
	}
	defer store.Close()

	header := make([]byte, 600)
	hash := make([]byte, 32)
	body := make([]byte, 2048)
	receipt := make([]byte, 500)
	_, _ = rand.Read(header)
	_, _ = rand.Read(hash)
	_, _ = rand.Read(body)
	_, _ = rand.Read(receipt)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		num := uint64(i)
		if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			if err := op.AppendRaw(ChainFreezerHeaderTable, num, header); err != nil {
				return err
			}
			if err := op.AppendRaw(ChainFreezerHashTable, num, hash); err != nil {
				return err
			}
			if err := op.AppendRaw(ChainFreezerBodiesTable, num, body); err != nil {
				return err
			}
			return op.AppendRaw(ChainFreezerReceiptTable, num, receipt)
		}); err != nil {
			b.Fatalf("ModifyAncients: %v", err)
		}
	}
}

// BenchmarkZapAncient_ReadHot measures Ancient(headers, n) latency on
// a pre-loaded store. Models a state-sync server serving a recent
// block.
func BenchmarkZapAncient_ReadHot(b *testing.B) {
	store, err := NewZapAncientStore(b.TempDir(), benchTables, memBackend{memorydb.New()})
	if err != nil {
		b.Fatalf("NewZapAncientStore: %v", err)
	}
	defer store.Close()

	const n = 100_000
	val := make([]byte, 600)
	if _, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := uint64(0); i < n; i++ {
			if err := op.AppendRaw(ChainFreezerHeaderTable, i, val); err != nil {
				return err
			}
			if err := op.AppendRaw(ChainFreezerHashTable, i, val[:32]); err != nil {
				return err
			}
			if err := op.AppendRaw(ChainFreezerBodiesTable, i, val); err != nil {
				return err
			}
			if err := op.AppendRaw(ChainFreezerReceiptTable, i, val); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		b.Fatalf("preload: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		num := uint64(i) % n
		if _, err := store.Ancient(ChainFreezerHeaderTable, num); err != nil {
			b.Fatalf("Ancient: %v", err)
		}
	}
}
