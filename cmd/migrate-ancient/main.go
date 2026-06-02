// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// migrate-ancient is a one-shot tool to copy an upstream-format
// ancient store (the .cdat/.ridx files written by core/rawdb.Freezer)
// into a ZAP-native ancient store (rawdb.ZapAncientStore).
//
// Usage:
//
//	migrate-ancient \
//	    --src=/data/db/<network>/chainData/<chainID>/ancient \
//	    --dst=/data/ancient/<network>/<chainID> \
//	    --tables=headers,hashes,bodies,receipts
//
// The tool walks every table in lockstep (chain tables share a head),
// reads each item from the upstream freezer, writes it to the new ZAP
// store via the same ethdb.AncientStore contract. After a successful
// migration the new store has the same (head, tail) pointers as the
// source. The source is NOT deleted — the operator is responsible for
// flipping the EVM config to point at the new path and then removing
// the legacy directory after validation.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/ethdb/memorydb"
)

func main() {
	src := flag.String("src", "", "source ancient datadir (upstream freezer)")
	dst := flag.String("dst", "", "destination zap-ancient datadir")
	tableList := flag.String("tables", "headers,hashes,bodies,receipts", "comma-separated list of tables to migrate")
	dryRun := flag.Bool("dry-run", false, "report what would be copied without writing")
	flag.Parse()

	if *src == "" || *dst == "" {
		fmt.Fprintln(os.Stderr, "usage: migrate-ancient --src <path> --dst <path> [--tables ...]")
		os.Exit(2)
	}

	tables := parseTables(*tableList)
	if err := migrate(*src, *dst, tables, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "migrate-ancient: %v\n", err)
		os.Exit(1)
	}
}

// parseTables maps the CLI string into the same config map shape the
// freezer + ZAP store consume. We only honor the four chain freezer
// tables; state + trienode freezers use different table layouts and
// will be migrated by a follow-up tool when needed.
func parseTables(raw string) map[string]rawdb.FreezerTableConfig {
	out := map[string]rawdb.FreezerTableConfig{}
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		switch name {
		case rawdb.ChainFreezerHeaderTable:
			out[name] = rawdb.FreezerTableConfig{NoSnappy: false, Prunable: false}
		case rawdb.ChainFreezerHashTable:
			out[name] = rawdb.FreezerTableConfig{NoSnappy: true, Prunable: false}
		case rawdb.ChainFreezerBodiesTable:
			out[name] = rawdb.FreezerTableConfig{NoSnappy: false, Prunable: true}
		case rawdb.ChainFreezerReceiptTable:
			out[name] = rawdb.FreezerTableConfig{NoSnappy: false, Prunable: true}
		default:
			fmt.Fprintf(os.Stderr, "warning: unknown table %q — skipped\n", name)
		}
	}
	return out
}

func migrate(src, dst string, tables map[string]rawdb.FreezerTableConfig, dryRun bool) error {
	// Open source freezer (read-only).
	srcStore, err := rawdb.OpenFreezerForMigration(src, tables)
	if err != nil {
		return fmt.Errorf("open source freezer: %w", err)
	}
	defer srcStore.Close()

	head, err := srcStore.Ancients()
	if err != nil {
		return fmt.Errorf("read source head: %w", err)
	}
	tail, err := srcStore.Tail()
	if err != nil {
		return fmt.Errorf("read source tail: %w", err)
	}
	fmt.Printf("source: %d items @ tail=%d head=%d\n", head-tail, tail, head)
	if dryRun {
		return nil
	}

	// Open destination ZAP store. The destination uses an in-process
	// KV — production paths swap in `ethdb/zapdb.New(dst, ...)` here.
	dstStore, err := rawdb.NewZapAncientStoreForMigration(dst, tables, memorydb.New())
	if err != nil {
		return fmt.Errorf("open dest zap-ancient: %w", err)
	}
	defer dstStore.Close()

	const batchSize = 1000
	start := time.Now()
	for n := tail; n < head; n += batchSize {
		hi := n + batchSize
		if hi > head {
			hi = head
		}
		if _, err := dstStore.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			for i := n; i < hi; i++ {
				for kind := range tables {
					val, err := srcStore.Ancient(kind, i)
					if err != nil {
						return fmt.Errorf("read %s @ %d: %w", kind, i, err)
					}
					if err := op.AppendRaw(kind, i, val); err != nil {
						return fmt.Errorf("write %s @ %d: %w", kind, i, err)
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
		fmt.Printf("\r  migrated %d / %d", hi-tail, head-tail)
	}
	fmt.Printf("\n  done in %s\n", time.Since(start))
	return nil
}
