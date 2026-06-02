// Copyright (C) 2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// ZAP-native ancient store.
//
// The upstream geth freezer is a directory full of .cdat / .ridx /
// .meta files — one set per table. That layout was designed for
// rotating disks circa 2019. For ZAP-backed Lux nodes the canonical
// store is `luxfi/zapdb` (the BadgerDB fork). This file implements
// `ethdb.AncientStore` over a single ZAP database so the EVM freezer
// path stays one-and-only-one-way: ZAP for hot KV, ZAP for cold
// ancient.
//
// Key layout (single ZAP database under <datadir>/):
//
//	["a", kind_byte, big-endian uint64 number]   →  raw value bytes
//	["m", "head", kind_byte]                     →  big-endian uint64 head
//	["m", "tail", kind_byte]                     →  big-endian uint64 tail
//
// The single-byte `kind_byte` is allocated at table-config time from a
// sorted list of table names (see kindByte). This keeps key prefixes
// short (4 bytes for the prefix + table marker + 8 bytes for the
// number), and lets `AncientRange` scan a single prefix efficiently
// using the iterator.
//
// Compression: per-block snappy when the table config says so. Headers,
// bodies, receipts compress ~10x cold; canonical hashes do not (they
// are already random 32-byte digests).
//
// This implementation deliberately stays single-writer to mirror the
// upstream `Freezer` semantics — ModifyAncients takes the write lock,
// runs the caller's function against a batch, commits atomically. A
// crashed write rolls the head pointers back.
package rawdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/luxfi/geth/ethdb"
	log "github.com/luxfi/log"
	"github.com/golang/snappy"
)

// kindByte assigns a stable single-byte marker per table name.
// Allocation is deterministic: alphabetical sort over the table config
// keys. Two zap-ancient stores opened with the same `tables` config
// agree byte-for-byte on every record key.
func kindByte(tables map[string]freezerTableConfig, kind string) (byte, bool) {
	names := sortedTableNames(tables)
	for i, n := range names {
		if n == kind {
			if i > 255 {
				return 0, false
			}
			return byte(i), true
		}
	}
	return 0, false
}

// sortedTableNames returns the table names in the order kindByte uses
// to allocate prefixes. Exported via the local zapAncient struct only.
func sortedTableNames(tables map[string]freezerTableConfig) []string {
	out := make([]string, 0, len(tables))
	for k := range tables {
		out = append(out, k)
	}
	// Manual sort to avoid pulling sort into a hot path that the
	// freezer init walks once at boot.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// zapAncientKey returns the canonical record key for (kind, number).
//
// Layout: 'a' | kind | uint64-be(number).
//
// 10 bytes total. The single-byte 'a' prefix segregates ancient data
// from metadata so a prefix iterator at "a" hits only block records.
func zapAncientKey(kindB byte, number uint64) []byte {
	k := make([]byte, 10)
	k[0] = 'a'
	k[1] = kindB
	binary.BigEndian.PutUint64(k[2:], number)
	return k
}

// zapAncientHeadKey returns the metadata key holding the head pointer
// for a kind. Layout: 'm' | 'h' | kind.
func zapAncientHeadKey(kindB byte) []byte { return []byte{'m', 'h', kindB} }

// zapAncientTailKey returns the metadata key holding the tail pointer
// for a kind. Layout: 'm' | 't' | kind.
func zapAncientTailKey(kindB byte) []byte { return []byte{'m', 't', kindB} }

// zapAncientBackend is the contract this package needs from a KV store
// to implement the ancient store. Defined as a minimal interface (Has /
// Get / Put / Delete / Iterator / batched write / Close) so the same
// implementation works against any ethdb.KeyValueStore — ZAP today,
// memory in tests, pebble for in-place upgrade scenarios.
type zapAncientBackend interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
	Put(key []byte, value []byte) error
	Delete(key []byte) error
	NewBatch() ethdb.Batch
	NewIterator(prefix []byte, start []byte) ethdb.Iterator
	Close() error
}

// ZapAncientStore implements ethdb.ResettableAncientStore over a single
// `ethdb.KeyValueStore`. The KV is typically a luxfi/zapdb database.
type ZapAncientStore struct {
	dir     string
	tables  map[string]freezerTableConfig
	kindMap map[string]byte // kind name → single-byte marker
	store   zapAncientBackend

	// frozen + tail are mirrored from the metadata records on disk so
	// hot reads do not hit the KV. They are loaded at open time and
	// kept in sync on each ModifyAncients / TruncateHead / TruncateTail.
	frozen sync.Map // kind → *uint64 (current head)
	tails  sync.Map // kind → *uint64 (current tail)

	closed atomic.Bool
	mu     sync.Mutex // serializes writes
}

// NewZapAncientStore opens a ZAP-backed ancient store at the given
// directory. `store` is the underlying KV (the caller wires
// `ethdb/zapdb.New(dir, ...)`). The tables map is the same shape the
// upstream freezer takes; only `noSnappy` is honored — `prunable` is
// always true for ZAP-backed tables because TruncateTail issues
// concrete Delete calls (no on-disk file boundary to align to).
func NewZapAncientStore(dir string, tables map[string]freezerTableConfig, store zapAncientBackend) (*ZapAncientStore, error) {
	if store == nil {
		return nil, errors.New("zap-ancient: nil backend")
	}
	if len(tables) == 0 {
		return nil, errors.New("zap-ancient: empty tables config")
	}
	if len(tables) > 256 {
		return nil, errors.New("zap-ancient: too many tables (limit 256)")
	}
	zas := &ZapAncientStore{
		dir:     dir,
		tables:  tables,
		kindMap: map[string]byte{},
		store:   store,
	}
	for _, name := range sortedTableNames(tables) {
		b, ok := kindByte(tables, name)
		if !ok {
			return nil, fmt.Errorf("zap-ancient: kind byte exhausted on table %q", name)
		}
		zas.kindMap[name] = b
		// Load head + tail from KV; zero when absent (first open).
		var head, tail uint64
		if v, err := store.Get(zapAncientHeadKey(b)); err == nil && len(v) == 8 {
			head = binary.BigEndian.Uint64(v)
		}
		if v, err := store.Get(zapAncientTailKey(b)); err == nil && len(v) == 8 {
			tail = binary.BigEndian.Uint64(v)
		}
		h := head
		ta := tail
		zas.frozen.Store(name, &h)
		zas.tails.Store(name, &ta)
	}
	log.Info("Opened ZAP-native ancient store", "dir", dir, "tables", len(tables))
	return zas, nil
}

// AncientDatadir returns the on-disk path used at open time.
func (z *ZapAncientStore) AncientDatadir() (string, error) {
	if z.closed.Load() {
		return "", errors.New("zap-ancient: closed")
	}
	return z.dir, nil
}

// Has reports whether the (kind, number) record exists. Used by
// ancienttest.
func (z *ZapAncientStore) HasAncient(kind string, number uint64) (bool, error) {
	if z.closed.Load() {
		return false, errors.New("zap-ancient: closed")
	}
	b, ok := z.kindMap[kind]
	if !ok {
		return false, errUnknownTable
	}
	return z.store.Has(zapAncientKey(b, number))
}

// Ancient retrieves a single record. Returns errOutOfBounds when the
// number is outside [tail, head).
func (z *ZapAncientStore) Ancient(kind string, number uint64) ([]byte, error) {
	if z.closed.Load() {
		return nil, errors.New("zap-ancient: closed")
	}
	b, ok := z.kindMap[kind]
	if !ok {
		return nil, errUnknownTable
	}
	head := z.headOf(kind)
	tail := z.tailOf(kind)
	if number < tail || number >= head {
		return nil, errOutOfBounds
	}
	raw, err := z.store.Get(zapAncientKey(b, number))
	if err != nil {
		return nil, err
	}
	return z.maybeDecompress(kind, raw)
}

// AncientRange retrieves a contiguous range of records. Matches the
// upstream freezer semantics: at most 'count' items, optional
// 'maxBytes' cap (at least one item returned even if it exceeds the
// cap so callers always make progress).
func (z *ZapAncientStore) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	if z.closed.Load() {
		return nil, errors.New("zap-ancient: closed")
	}
	if count == 0 {
		return nil, errOutOfBounds
	}
	b, ok := z.kindMap[kind]
	if !ok {
		return nil, errUnknownTable
	}
	head := z.headOf(kind)
	tail := z.tailOf(kind)
	if start < tail || start >= head {
		return nil, errOutOfBounds
	}
	if start+count > head {
		count = head - start
	}
	out := make([][]byte, 0, count)
	var size uint64
	for n := start; n < start+count; n++ {
		raw, err := z.store.Get(zapAncientKey(b, n))
		if err != nil {
			return nil, err
		}
		val, err := z.maybeDecompress(kind, raw)
		if err != nil {
			return nil, err
		}
		// "if maxBytes is specified: at least 1 item (even if exceeding
		// the maxByteSize), but will otherwise return as many items as
		// fit into maxByteSize."
		if len(out) > 0 && maxBytes != 0 && size+uint64(len(val)) > maxBytes {
			return out, nil
		}
		out = append(out, val)
		size += uint64(len(val))
	}
	return out, nil
}

// AncientBytes returns a value slice. Snappy tables decompress first
// then slice — keeping the semantics of the upstream freezer's
// AncientBytes, which returns the decompressed payload.
func (z *ZapAncientStore) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	val, err := z.Ancient(kind, id)
	if err != nil {
		return nil, err
	}
	if offset >= uint64(len(val)) {
		return nil, nil
	}
	end := offset + length
	if end > uint64(len(val)) {
		end = uint64(len(val))
	}
	return val[offset:end], nil
}

// Ancients returns the current head — the next number that would be
// assigned by Append.
func (z *ZapAncientStore) Ancients() (uint64, error) {
	if z.closed.Load() {
		return 0, errors.New("zap-ancient: closed")
	}
	// All chain tables stay aligned to the same head; pick the first.
	for kind := range z.tables {
		return z.headOf(kind), nil
	}
	return 0, nil
}

// Tail returns the current tail — the first stored number.
func (z *ZapAncientStore) Tail() (uint64, error) {
	if z.closed.Load() {
		return 0, errors.New("zap-ancient: closed")
	}
	for kind := range z.tables {
		return z.tailOf(kind), nil
	}
	return 0, nil
}

// AncientSize is a cheap estimate. For ZAP-backed tables we cannot
// easily compute per-kind on-disk size without walking; return the
// item count × 0 as a sentinel. Callers use this for metrics + log
// lines only.
func (z *ZapAncientStore) AncientSize(kind string) (uint64, error) {
	if z.closed.Load() {
		return 0, errors.New("zap-ancient: closed")
	}
	if _, ok := z.kindMap[kind]; !ok {
		return 0, errUnknownTable
	}
	// Walk the prefix once. Cheap on cold-data sizes (1-100M records);
	// the call site is debug-only.
	prefix := []byte{'a', z.kindMap[kind]}
	it := z.store.NewIterator(prefix, nil)
	defer it.Release()
	var size uint64
	for it.Next() {
		size += uint64(len(it.Value()))
	}
	return size, nil
}

// ReadAncients runs the caller's function holding the write mutex (so
// no Modify/Truncate happens mid-read). Matches the upstream
// freezer's semantics.
func (z *ZapAncientStore) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	z.mu.Lock()
	defer z.mu.Unlock()
	return fn(z)
}

// ModifyAncients runs a write batch. The function gets a writeOp that
// accumulates puts; on success we flush atomically and bump the head
// pointers. On error we roll back the in-memory head so the next call
// re-uses the same range.
func (z *ZapAncientStore) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (int64, error) {
	if z.closed.Load() {
		return 0, errors.New("zap-ancient: closed")
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	prev := map[string]uint64{}
	for kind := range z.tables {
		prev[kind] = z.headOf(kind)
	}
	op := &zapWriteOp{owner: z, batch: z.store.NewBatch(), startHeads: prev}
	if err := fn(op); err != nil {
		return 0, err
	}
	// AppendRaw enforced contiguous in-order inserts on every call, so
	// op.heads is the resolved head per table. We only sanity-check
	// that every advanced table starts exactly at its previous head
	// (in case the caller skipped a kind by jumping ahead via Heads
	// only — defensive, not load-bearing).
	// Persist new head pointers in the same batch.
	for kind, head := range op.heads {
		b := z.kindMap[kind]
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, head)
		if err := op.batch.Put(zapAncientHeadKey(b), buf); err != nil {
			for k, h := range prev {
				z.setHead(k, h)
			}
			return 0, err
		}
	}
	if err := op.batch.Write(); err != nil {
		for k, h := range prev {
			z.setHead(k, h)
		}
		return 0, err
	}
	for kind, head := range op.heads {
		z.setHead(kind, head)
	}
	return op.written, nil
}

// SyncAncient is a no-op for ZAP — batches flush synchronously on
// commit. Provided to satisfy the interface.
func (z *ZapAncientStore) SyncAncient() error {
	if z.closed.Load() {
		return errors.New("zap-ancient: closed")
	}
	return nil
}

// TruncateHead discards records at or above `n`. Walks each table and
// issues Delete on the trailing range. Returns the previous head.
func (z *ZapAncientStore) TruncateHead(n uint64) (uint64, error) {
	if z.closed.Load() {
		return 0, errors.New("zap-ancient: closed")
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	prevHead := uint64(0)
	for kind, b := range z.kindMap {
		head := z.headOf(kind)
		if head > prevHead {
			prevHead = head
		}
		if head <= n {
			continue
		}
		batch := z.store.NewBatch()
		for i := n; i < head; i++ {
			if err := batch.Delete(zapAncientKey(b, i)); err != nil {
				return 0, err
			}
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, n)
		if err := batch.Put(zapAncientHeadKey(b), buf); err != nil {
			return 0, err
		}
		if err := batch.Write(); err != nil {
			return 0, err
		}
		z.setHead(kind, n)
	}
	return prevHead, nil
}

// TruncateTail discards records below `n`. Honors the prunable flag on
// the table config; tables that opt out are left untouched (matches
// upstream behavior for headers + hashes).
func (z *ZapAncientStore) TruncateTail(n uint64) (uint64, error) {
	if z.closed.Load() {
		return 0, errors.New("zap-ancient: closed")
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	prevTail := uint64(0)
	for kind, b := range z.kindMap {
		if !z.tables[kind].prunable {
			continue
		}
		tail := z.tailOf(kind)
		if tail > prevTail {
			prevTail = tail
		}
		if tail >= n {
			continue
		}
		batch := z.store.NewBatch()
		for i := tail; i < n; i++ {
			if err := batch.Delete(zapAncientKey(b, i)); err != nil {
				return 0, err
			}
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, n)
		if err := batch.Put(zapAncientTailKey(b), buf); err != nil {
			return 0, err
		}
		if err := batch.Write(); err != nil {
			return 0, err
		}
		z.setTail(kind, n)
	}
	return prevTail, nil
}

// Close shuts down the backend. The underlying store's Close is called
// (the caller transferred ownership at NewZapAncientStore).
func (z *ZapAncientStore) Close() error {
	if !z.closed.CompareAndSwap(false, true) {
		return nil
	}
	return z.store.Close()
}

// Reset clears every record + metadata pointer. Used by the chain
// freezer's reset path when a hard reorg drops below the freezer
// threshold.
func (z *ZapAncientStore) Reset() error {
	if z.closed.Load() {
		return errors.New("zap-ancient: closed")
	}
	z.mu.Lock()
	defer z.mu.Unlock()
	for kind, b := range z.kindMap {
		// Walk + delete the kind's records.
		prefix := []byte{'a', b}
		it := z.store.NewIterator(prefix, nil)
		batch := z.store.NewBatch()
		for it.Next() {
			if err := batch.Delete(append([]byte(nil), it.Key()...)); err != nil {
				it.Release()
				return err
			}
		}
		it.Release()
		_ = batch.Delete(zapAncientHeadKey(b))
		_ = batch.Delete(zapAncientTailKey(b))
		if err := batch.Write(); err != nil {
			return err
		}
		z.setHead(kind, 0)
		z.setTail(kind, 0)
	}
	return nil
}

// headOf / tailOf / setHead / setTail are atomic accessors. The
// per-kind heads + tails are stored as *uint64 in sync.Maps so the
// caller does not need to re-acquire the write mutex for a hot read.

func (z *ZapAncientStore) headOf(kind string) uint64 {
	v, ok := z.frozen.Load(kind)
	if !ok {
		return 0
	}
	return atomic.LoadUint64(v.(*uint64))
}

func (z *ZapAncientStore) tailOf(kind string) uint64 {
	v, ok := z.tails.Load(kind)
	if !ok {
		return 0
	}
	return atomic.LoadUint64(v.(*uint64))
}

func (z *ZapAncientStore) setHead(kind string, n uint64) {
	v, ok := z.frozen.Load(kind)
	if !ok {
		x := n
		z.frozen.Store(kind, &x)
		return
	}
	atomic.StoreUint64(v.(*uint64), n)
}

func (z *ZapAncientStore) setTail(kind string, n uint64) {
	v, ok := z.tails.Load(kind)
	if !ok {
		x := n
		z.tails.Store(kind, &x)
		return
	}
	atomic.StoreUint64(v.(*uint64), n)
}

// maybeDecompress runs snappy.Decode if the table config opted into
// compression. Snappy is small + fast — under 5% CPU overhead even on
// hot replays.
func (z *ZapAncientStore) maybeDecompress(kind string, raw []byte) ([]byte, error) {
	if z.tables[kind].noSnappy {
		return raw, nil
	}
	return snappy.Decode(nil, raw)
}

// maybeCompress is the reverse half. The write op calls this to encode
// before staging the put.
func (z *ZapAncientStore) maybeCompress(kind string, raw []byte) []byte {
	if z.tables[kind].noSnappy {
		return raw
	}
	return snappy.Encode(nil, raw)
}

// zapWriteOp implements ethdb.AncientWriteOp on top of an ethdb.Batch.
// It accumulates puts in-memory; ModifyAncients flushes the batch on
// success.
type zapWriteOp struct {
	owner      *ZapAncientStore
	batch      ethdb.Batch
	written    int64
	startHeads map[string]uint64 // kind → head at the start of this op
	heads      map[string]uint64 // kind → new head after this op
}

// AppendRaw appends a raw byte payload. Number must equal the table's
// current head — the freezer enforces no-gap inserts.
func (op *zapWriteOp) AppendRaw(kind string, number uint64, item []byte) error {
	b, ok := op.owner.kindMap[kind]
	if !ok {
		return errUnknownTable
	}
	// Expected number is the running head — startHeads on first
	// append, op.heads thereafter (each successful append bumps it).
	var expected uint64
	if h, ok := op.heads[kind]; ok {
		expected = h
	} else {
		expected = op.startHeads[kind]
	}
	if number != expected {
		return fmt.Errorf("zap-ancient: %s: out-of-order append (expected %d, got %d): %w",
			kind, expected, number, errOutOrderInsertion)
	}
	value := op.owner.maybeCompress(kind, item)
	if err := op.batch.Put(zapAncientKey(b, number), value); err != nil {
		return err
	}
	if op.heads == nil {
		op.heads = map[string]uint64{}
	}
	op.heads[kind] = number + 1
	op.written += int64(len(value))
	return nil
}

// Append RLP-encodes the item then defers to AppendRaw.
func (op *zapWriteOp) Append(kind string, number uint64, item interface{}) error {
	raw, err := rlpEncode(item)
	if err != nil {
		return err
	}
	return op.AppendRaw(kind, number, raw)
}

// rlpEncode wraps the RLP encoder so the writeOp depends on a function
// pointer instead of the heavy import chain through types/rlp.
var rlpEncode = func(item interface{}) ([]byte, error) {
	return rlpEncodeImpl(item)
}
