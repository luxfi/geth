// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// fleet-bench drives a NodeFleet-shaped workload against an in-process
// ZAP-native ancient store and records the per-block write size + read
// latency. Decouples the operator (k8s reconciliation) from the
// storage engine so we can tune the storage engine without spinning a
// kind cluster.
//
// Usage:
//
//	fleet-bench --blocks=10000 --bodies-bytes=2048 --out=fleet.csv
//
// Output CSV columns: block, write_us, write_bytes, read_us.
package main

import (
	"crypto/rand"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/memorydb"
)

func main() {
	blocks := flag.Int("blocks", 10000, "number of synthetic blocks to write")
	bodyBytes := flag.Int("bodies-bytes", 2048, "size of synthetic body")
	outPath := flag.String("out", "", "CSV output path (defaults to stdout)")
	flag.Parse()

	tables := map[string]rawdb.FreezerTableConfig{
		rawdb.ChainFreezerHeaderTable:  {NoSnappy: false, Prunable: false},
		rawdb.ChainFreezerHashTable:    {NoSnappy: true, Prunable: false},
		rawdb.ChainFreezerBodiesTable:  {NoSnappy: false, Prunable: true},
		rawdb.ChainFreezerReceiptTable: {NoSnappy: false, Prunable: true},
	}

	dir, err := os.MkdirTemp("", "fleet-bench-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tempdir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	store, err := rawdb.NewZapAncientStoreForMigration(dir, tables, memorydb.New())
	if err != nil {
		fmt.Fprintf(os.Stderr, "open store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	header := make([]byte, 600)
	hash := make([]byte, 32)
	body := make([]byte, *bodyBytes)
	receipt := make([]byte, 500)
	_, _ = rand.Read(header)
	_, _ = rand.Read(hash)
	_, _ = rand.Read(body)
	_, _ = rand.Read(receipt)

	var w *csv.Writer
	if *outPath == "" {
		w = csv.NewWriter(os.Stdout)
	} else {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create out: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		w = csv.NewWriter(f)
	}
	defer w.Flush()
	_ = w.Write([]string{"block", "write_us", "write_bytes", "read_us"})

	for i := 0; i < *blocks; i++ {
		num := uint64(i)
		writeStart := time.Now()
		written, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			if err := op.AppendRaw(rawdb.ChainFreezerHeaderTable, num, header); err != nil {
				return err
			}
			if err := op.AppendRaw(rawdb.ChainFreezerHashTable, num, hash); err != nil {
				return err
			}
			if err := op.AppendRaw(rawdb.ChainFreezerBodiesTable, num, body); err != nil {
				return err
			}
			return op.AppendRaw(rawdb.ChainFreezerReceiptTable, num, receipt)
		})
		writeUs := time.Since(writeStart).Microseconds()
		if err != nil {
			fmt.Fprintf(os.Stderr, "write block %d: %v\n", i, err)
			os.Exit(1)
		}
		// Hot read against the just-written block — models a replica
		// state-syncing the latest blocks.
		readStart := time.Now()
		if _, err := store.Ancient(rawdb.ChainFreezerHeaderTable, num); err != nil {
			fmt.Fprintf(os.Stderr, "read block %d: %v\n", i, err)
			os.Exit(1)
		}
		readUs := time.Since(readStart).Microseconds()

		_ = w.Write([]string{
			strconv.Itoa(i),
			strconv.FormatInt(writeUs, 10),
			strconv.FormatInt(written, 10),
			strconv.FormatInt(readUs, 10),
		})
	}
}
