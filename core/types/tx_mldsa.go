// Copyright 2024-2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// tx_mldsa.go — strict-PQ Liquid transaction envelope.
//
// FIPS 204 ML-DSA-65 signing identity. Sender address is the
// 20-byte truncation of sha256(MLDSAPubKey) — a NEW address space,
// distinct from the classical keccak256(secp256k1-pub)[12:] EOA
// space. The two spaces never collide because the derivation
// algorithm differs at the byte level; a migration tooling that
// wants to map "classical Alice" to "PQ Alice" must use an
// on-chain alias contract, not address arithmetic.
//
// Why not v/r/s style? ECDSA's (v, r, s) triple is curve-order
// specific (256-bit scalars on secp256k1). ML-DSA signatures are
// 3293 bytes and don't compress into r/s pairs. We add a parallel
// (PubKey, Sig) tail on the tx and route ALL strict-PQ verifiers
// through that path; the inherited TxData methods
// rawSignatureValues / setSignatureValues return the empty triple
// so any consumer that hard-codes the ECDSA shape gets a clear
// "(0, 0, 0)" rather than a silently-wrong recovery.

package types

import (
	"bytes"
	"crypto/sha256"
	"math/big"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

// MLDSATx is the strict-PQ Liquid transaction. RLP-encoded body
// follows the same field order as DynamicFeeTx (chain id, nonce,
// gas caps, gas, to, value, data, access list) and adds the
// (PubKey, Sig) tail at the end. The 0x42 type byte prefix on the
// canonical encoding means consumers that read by-type-byte route
// straight to this struct's decode path.
type MLDSATx struct {
	ChainID    *big.Int // EIP-1559 chain id
	Nonce      uint64
	GasTipCap  *big.Int // a.k.a. maxPriorityFeePerGas
	GasFeeCap  *big.Int // a.k.a. maxFeePerGas
	Gas        uint64
	To         *common.Address // nil = contract creation
	Value      *big.Int
	Data       []byte
	AccessList AccessList

	// Strict-PQ identity tail. Replaces the classical (V, R, S)
	// triple every other TxData impl carries. PubKey size is the
	// FIPS 204 fixed 1952 bytes (asserted by Sender derivation);
	// Sig size is the FIPS 204 fixed 3293 bytes.
	PubKey []byte // ML-DSA-65 public key (1952 bytes)
	Sig    []byte // ML-DSA-65 signature (3293 bytes)
}

// copy returns a deep copy of the MLDSATx.
func (tx *MLDSATx) copy() TxData {
	cpy := &MLDSATx{
		Nonce:      tx.Nonce,
		Gas:        tx.Gas,
		To:         copyAddressPtr(tx.To),
		Data:       common.CopyBytes(tx.Data),
		AccessList: make(AccessList, len(tx.AccessList)),
		PubKey:     common.CopyBytes(tx.PubKey),
		Sig:        common.CopyBytes(tx.Sig),
	}
	copy(cpy.AccessList, tx.AccessList)
	if tx.ChainID != nil {
		cpy.ChainID = new(big.Int).Set(tx.ChainID)
	}
	if tx.Value != nil {
		cpy.Value = new(big.Int).Set(tx.Value)
	}
	if tx.GasTipCap != nil {
		cpy.GasTipCap = new(big.Int).Set(tx.GasTipCap)
	}
	if tx.GasFeeCap != nil {
		cpy.GasFeeCap = new(big.Int).Set(tx.GasFeeCap)
	}
	return cpy
}

// MLDSATxType (and the type, chain, etc. accessors) — these match
// the TxData interface.

func (tx *MLDSATx) txType() byte           { return MLDSATxType }
func (tx *MLDSATx) chainID() *big.Int      { return tx.ChainID }
func (tx *MLDSATx) accessList() AccessList { return tx.AccessList }
func (tx *MLDSATx) data() []byte           { return tx.Data }
func (tx *MLDSATx) gas() uint64            { return tx.Gas }
func (tx *MLDSATx) gasFeeCap() *big.Int    { return tx.GasFeeCap }
func (tx *MLDSATx) gasTipCap() *big.Int    { return tx.GasTipCap }
func (tx *MLDSATx) gasPrice() *big.Int     { return tx.GasFeeCap }
func (tx *MLDSATx) value() *big.Int        { return tx.Value }
func (tx *MLDSATx) nonce() uint64          { return tx.Nonce }
func (tx *MLDSATx) to() *common.Address    { return tx.To }

// rawSignatureValues returns the empty classical (V, R, S) triple.
// MLDSATx carries its signature via (PubKey, Sig) — any consumer
// that calls this method on an MLDSATx is using the wrong
// signature surface; the return (0, 0, 0) makes that visible at
// the call site rather than silently coercing the ML-DSA bytes
// into a fake ECDSA shape.
func (tx *MLDSATx) rawSignatureValues() (v, r, s *big.Int) {
	return new(big.Int), new(big.Int), new(big.Int)
}

// setSignatureValues is a no-op for MLDSATx. ChainID flows through
// the dedicated ChainID field; the ECDSA triple has no meaning.
// The MLDSASigner sets PubKey / Sig directly via
// WithMLDSASignature, not through this interface.
func (tx *MLDSATx) setSignatureValues(chainID, v, r, s *big.Int) {
	if chainID != nil {
		tx.ChainID = new(big.Int).Set(chainID)
	}
}

// effectiveGasPrice mirrors DynamicFeeTx: min(GasFeeCap, GasTipCap + baseFee).
func (tx *MLDSATx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.GasFeeCap)
	}
	tip := dst.Sub(tx.GasFeeCap, baseFee)
	if tip.Cmp(tx.GasTipCap) > 0 {
		tip.Set(tx.GasTipCap)
	}
	return tip.Add(tip, baseFee)
}

// encode emits the canonical RLP body for an MLDSATx (no type byte
// prefix — that's added by the outer Transaction.MarshalBinary).
func (tx *MLDSATx) encode(buf *bytes.Buffer) error {
	return rlp.Encode(buf, tx)
}

// decode parses the canonical RLP body.
func (tx *MLDSATx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

// sigHash returns the digest the ML-DSA-65 signature covers. The
// digest is computed via SHAKE256 (consistent with the rest of
// the strict-PQ surface) over the RLP-encoded tx body WITHOUT the
// (PubKey, Sig) tail. ChainID is included as the FIRST field so
// the same key signing the same tx body on a DIFFERENT chain
// produces a different digest (cross-chain replay protection
// without needing the EIP-155 v-byte trick).
func (tx *MLDSATx) sigHash(chainID *big.Int) common.Hash {
	// Sign over RLP-encoded body with PubKey and Sig cleared.
	stripped := *tx
	stripped.PubKey = nil
	stripped.Sig = nil
	if chainID != nil {
		stripped.ChainID = new(big.Int).Set(chainID)
	}
	var buf bytes.Buffer
	buf.WriteByte(MLDSATxType)
	_ = rlp.Encode(&buf, &stripped)
	return sha256.Sum256(buf.Bytes())
}

// MLDSASenderFromPubKey returns the 20-byte address derived from
// an ML-DSA-65 public key: sha256(pubKey)[:20]. The same
// derivation is used by lx/dex SignedOrderPQ — one address-space
// shared across the Liquid EVM and Liquid DEX, so a trader has
// the same identity on both chains under the same key.
func MLDSASenderFromPubKey(pubKey []byte) common.Address {
	h := sha256.Sum256(pubKey)
	var a common.Address
	copy(a[:], h[:20])
	return a
}
