// Copyright 2024-2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

// transaction_signing_mldsa.go — strict-PQ FIPS 204 ML-DSA-65 signer.
//
// Recovers sender address from the (PubKey, Sig) tail of an
// MLDSATx, not from an ECDSA (V, R, S) triple. Sender =
// sha256(PubKey)[:20] — a new address space distinct from the
// classical keccak256(secp256k1-pub)[12:] EOA space.
//
// The Signer interface in this package returns (r, s, v) from
// SignatureValues — that shape is ECDSA-specific. MLDSASigner
// returns (0, 0, 0) for that method and exposes
// SignMLDSA / SenderMLDSA as the strict-PQ surface. Callers that
// hard-code the ECDSA shape get a clear empty triple rather than
// a silently-wrong recovery — they should be routed through the
// MLDSA-specific path instead.

package types

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/geth/common"
)

// MLDSASigner is the FIPS 204 ML-DSA-65 transaction signer. One
// instance per chain (binds to a specific ChainID so the same key
// signing the same tx body on a different chain produces a
// different digest — cross-chain replay protection without the
// EIP-155 v-byte trick).
type MLDSASigner struct {
	chainID *big.Int
}

// NewMLDSASigner returns a strict-PQ ML-DSA-65 signer for chainID.
func NewMLDSASigner(chainID *big.Int) MLDSASigner {
	if chainID == nil {
		chainID = new(big.Int)
	}
	return MLDSASigner{chainID: new(big.Int).Set(chainID)}
}

// ChainID returns the chain id this signer binds to.
func (s MLDSASigner) ChainID() *big.Int {
	return new(big.Int).Set(s.chainID)
}

// Equal reports whether two MLDSASigners bind to the same chain id.
func (s MLDSASigner) Equal(other Signer) bool {
	o, ok := other.(MLDSASigner)
	return ok && s.chainID.Cmp(o.chainID) == 0
}

// Hash returns the digest the ML-DSA-65 signature covers.
// Delegates to MLDSATx.sigHash which is SHAKE256 over
// (typeByte || RLP(body without PubKey/Sig)). Chain id binding
// makes the digest cross-chain unique.
func (s MLDSASigner) Hash(tx *Transaction) common.Hash {
	mldsaTx, ok := tx.inner.(*MLDSATx)
	if !ok {
		// Caller passed a classical tx to a strict-PQ signer —
		// return zero so the misuse is observable at the
		// verification site rather than producing a wrong
		// chain-bound digest.
		return common.Hash{}
	}
	return mldsaTx.sigHash(s.chainID)
}

// SignatureValues returns the empty classical (r, s, v) triple.
// MLDSA signatures don't fit in this shape; the strict-PQ
// signature lives on MLDSATx.{PubKey, Sig}. Returning (0, 0, 0)
// is the deliberate signal that "ECDSA values are not meaningful
// for this tx" — call SenderFromTx / WithMLDSASignature instead.
func (s MLDSASigner) SignatureValues(tx *Transaction, sig []byte) (r, ss, v *big.Int, err error) {
	return new(big.Int), new(big.Int), new(big.Int), nil
}

// Sender recovers the sender address from an MLDSATx's
// (PubKey, Sig) tail. The check is two-step:
//
//  1. Validate the signature with mldsa65.Verify(pub, sigHash, sig).
//  2. Derive Sender = sha256(pubKey)[:20] from the (now-verified)
//     pubkey — the address is the *pubkey's hash*, not a separate
//     field that could be lied about.
//
// Refuses any non-MLDSATx with ErrInvalidSigner so misuse on a
// classical tx fails loud.
func (s MLDSASigner) Sender(tx *Transaction) (common.Address, error) {
	mldsaTx, ok := tx.inner.(*MLDSATx)
	if !ok {
		return common.Address{}, fmt.Errorf("%w: MLDSASigner only handles MLDSATx (got type 0x%02x)",
			ErrInvalidSigner, tx.inner.txType())
	}
	if s.chainID.Cmp(mldsaTx.ChainID) != 0 {
		return common.Address{}, fmt.Errorf("%w: signer chain id %s, tx chain id %s",
			ErrInvalidChainId, s.chainID, mldsaTx.ChainID)
	}
	if len(mldsaTx.PubKey) != mldsa.MLDSA65PublicKeySize {
		return common.Address{}, fmt.Errorf("MLDSASigner: pubkey len=%d, want %d",
			len(mldsaTx.PubKey), mldsa.MLDSA65PublicKeySize)
	}
	if len(mldsaTx.Sig) == 0 {
		return common.Address{}, errors.New("MLDSASigner: empty ML-DSA signature")
	}

	pub, err := mldsa.PublicKeyFromBytes(mldsaTx.PubKey, mldsa.MLDSA65)
	if err != nil {
		return common.Address{}, fmt.Errorf("MLDSASigner: parse pubkey: %w", err)
	}
	hash := mldsaTx.sigHash(s.chainID)
	if !pub.VerifySignature(hash[:], mldsaTx.Sig) {
		return common.Address{}, errors.New("MLDSASigner: ML-DSA-65 signature verification failed")
	}
	return MLDSASenderFromPubKey(mldsaTx.PubKey), nil
}

// ErrInvalidSigner is returned when a Signer is asked to handle a
// transaction type it doesn't recognize. Mirrors the shape of
// EIP155Signer's mismatch error so callers can errors.Is against
// a single sentinel across all signers.
var ErrInvalidSigner = errors.New("invalid signer for transaction type")
