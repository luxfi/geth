// Copyright 2024-2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package types

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/geth/common"
)

func mldsaKeypair(t *testing.T) (*mldsa.PrivateKey, []byte) {
	t.Helper()
	priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
	if err != nil {
		t.Fatalf("mldsa keygen: %v", err)
	}
	return priv, priv.PublicKey.Bytes()
}

// TestMLDSATx_RoundTrip signs an MLDSATx with the FIPS 204
// primitive, verifies the recovered Sender matches the pubkey-
// derived address, and checks that signing the same tx body on a
// different chain id produces a different sigHash (cross-chain
// replay protection).
func TestMLDSATx_RoundTrip(t *testing.T) {
	priv, pubBytes := mldsaKeypair(t)
	chainID := big.NewInt(424242)

	to := common.HexToAddress("0xdeadbeef00000000000000000000000000000001")
	inner := &MLDSATx{
		ChainID:   chainID,
		Nonce:     7,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(2),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(100),
		Data:      []byte("hello"),
		PubKey:    pubBytes,
	}
	hash := inner.sigHash(chainID)
	sig, err := priv.Sign(rand.Reader, hash[:], nil)
	if err != nil {
		t.Fatalf("mldsa sign: %v", err)
	}
	inner.Sig = sig

	tx := NewTx(inner)
	signer := NewMLDSASigner(chainID)
	gotSender, err := signer.Sender(tx)
	if err != nil {
		t.Fatalf("MLDSASigner.Sender: %v", err)
	}
	wantSender := MLDSASenderFromPubKey(pubBytes)
	if gotSender != wantSender {
		t.Errorf("Sender = %s, want %s", gotSender.Hex(), wantSender.Hex())
	}

	// Different chain id → different digest. Sign + verify under
	// a wrong-chain signer should fail.
	wrongSigner := NewMLDSASigner(big.NewInt(131313))
	if _, err := wrongSigner.Sender(tx); err == nil {
		t.Error("wrong-chain MLDSASigner accepted tx")
	}
}

// TestMLDSATx_RawSignatureValuesEmpty pins the deliberate
// "classical ECDSA values are not meaningful" signal. Any consumer
// that hard-codes rawSignatureValues() on an MLDSATx gets (0, 0, 0)
// — the correct path is MLDSATx.{PubKey, Sig}.
func TestMLDSATx_RawSignatureValuesEmpty(t *testing.T) {
	inner := &MLDSATx{ChainID: big.NewInt(1)}
	v, r, s := inner.rawSignatureValues()
	if v.Sign() != 0 || r.Sign() != 0 || s.Sign() != 0 {
		t.Errorf("rawSignatureValues = (%s, %s, %s), want (0, 0, 0)",
			v, r, s)
	}
}

// TestMLDSASigner_RefuseClassicalTx asserts that asking the
// strict-PQ signer to recover from a classical tx fails loud —
// the wrong-shape misuse is observable, not silently coerced.
func TestMLDSASigner_RefuseClassicalTx(t *testing.T) {
	signer := NewMLDSASigner(big.NewInt(1))
	classical := NewTx(&LegacyTx{Nonce: 1, Gas: 21000, GasPrice: big.NewInt(1)})
	if _, err := signer.Sender(classical); err == nil {
		t.Fatal("MLDSASigner.Sender accepted a LegacyTx")
	}
}
