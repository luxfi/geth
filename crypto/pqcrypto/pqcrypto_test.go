// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// Post-Quantum Cryptography Integration Tests

package pqcrypto

import (
	"bytes"
	"testing"

	"github.com/luxfi/crypto"
	"github.com/stretchr/testify/require"
)

func TestPQSigner(t *testing.T) {
	algorithms := []Algorithm{
		AlgoClassical,
		AlgoMLDSA44,
		AlgoMLDSA65,
		AlgoMLDSA87,
		AlgoMLKEM512,
		AlgoMLKEM768,
		AlgoMLKEM1024,
		AlgoSLHDSA128s,
		AlgoSLHDSA192s,
		AlgoSLHDSA256s,
		AlgoHybridSecp256k1MLDSA,
		AlgoHybridSecp256k1MLKEM,
	}

	for _, algo := range algorithms {
		t.Run(algo.String(), func(t *testing.T) {
			require := require.New(t)

			// Create signer
			signer, err := NewPQSigner(algo)
			require.NoError(err)
			require.NotNil(signer)

			// Get address
			addr := signer.Address()
			require.NotEqual(common.Address{}, addr)

			// Test signing (skip KEM-only algorithms)
			if algo != AlgoMLKEM512 && algo != AlgoMLKEM768 && algo != AlgoMLKEM1024 && algo != AlgoHybridSecp256k1MLKEM {
				message := []byte("Test message for PQ signing")
				signature, err := signer.Sign(message)
				require.NoError(err)
				require.NotEmpty(signature)
			}

			// Test KEM operations
			if algo == AlgoMLKEM512 || algo == AlgoMLKEM768 || algo == AlgoMLKEM1024 || algo == AlgoHybridSecp256k1MLKEM {
				// Create another signer for key exchange
				signer2, err := NewPQSigner(algo)
				require.NoError(err)

				// Get public key
				var pubKey []byte
				if algo == AlgoHybridSecp256k1MLKEM {
					pubKey = signer2.mlkemPriv.PublicKey.Bytes()
				} else {
					pubKey = signer2.mlkemPriv.PublicKey.Bytes()
				}

				// Encapsulate
				ciphertext, sharedSecret1, err := signer.Encapsulate(pubKey)
				require.NoError(err)
				require.NotEmpty(ciphertext)
				require.NotEmpty(sharedSecret1)

				// Decapsulate
				sharedSecret2, err := signer2.Decapsulate(ciphertext)
				require.NoError(err)
				require.Equal(sharedSecret1, sharedSecret2)
			}
		})
	}
}

func TestHybridMode(t *testing.T) {
	require := require.New(t)

	// Create hybrid signer
	signer, err := NewPQSigner(AlgoHybridSecp256k1MLDSA)
	require.NoError(err)

	// Test message
	message := []byte("Hybrid signature test message")

	// Sign with hybrid mode
	signature, err := signer.Sign(message)
	require.NoError(err)
	require.NotEmpty(signature)

	// Signature should contain both classical and PQ parts
	// Classical ECDSA signature is 65 bytes
	require.Greater(len(signature), 65)
}

func TestAddressGeneration(t *testing.T) {
	require := require.New(t)

	// Test that different algorithms generate different addresses
	addresses := make(map[string]bool)

	algorithms := []Algorithm{
		AlgoClassical,
		AlgoMLDSA44,
		AlgoSLHDSA128s,
	}

	for _, algo := range algorithms {
		signer, err := NewPQSigner(algo)
		require.NoError(err)

		addr := signer.Address().Hex()
		require.NotEmpty(addr)
		
		// Check uniqueness
		require.False(addresses[addr], "Duplicate address generated")
		addresses[addr] = true
	}
}

func BenchmarkPQSigning(b *testing.B) {
	algorithms := []Algorithm{
		AlgoClassical,
		AlgoMLDSA44,
		AlgoSLHDSA128s,
	}

	message := []byte("Benchmark message for signing")

	for _, algo := range algorithms {
		b.Run(algo.String(), func(b *testing.B) {
			signer, _ := NewPQSigner(algo)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = signer.Sign(message)
			}
		})
	}
}

func BenchmarkKEM(b *testing.B) {
	algorithms := []Algorithm{
		AlgoMLKEM512,
		AlgoMLKEM768,
		AlgoMLKEM1024,
	}

	for _, algo := range algorithms {
		b.Run(algo.String()+"_Encapsulate", func(b *testing.B) {
			signer1, _ := NewPQSigner(algo)
			signer2, _ := NewPQSigner(algo)
			pubKey := signer2.mlkemPriv.PublicKey.Bytes()
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, _ = signer1.Encapsulate(pubKey)
			}
		})
	}
}

// String returns the string representation of the algorithm
func (a Algorithm) String() string {
	switch a {
	case AlgoClassical:
		return "Classical"
	case AlgoMLDSA44:
		return "ML-DSA-44"
	case AlgoMLDSA65:
		return "ML-DSA-65"
	case AlgoMLDSA87:
		return "ML-DSA-87"
	case AlgoMLKEM512:
		return "ML-KEM-512"
	case AlgoMLKEM768:
		return "ML-KEM-768"
	case AlgoMLKEM1024:
		return "ML-KEM-1024"
	case AlgoSLHDSA128s:
		return "SLH-DSA-128s"
	case AlgoSLHDSA192s:
		return "SLH-DSA-192s"
	case AlgoSLHDSA256s:
		return "SLH-DSA-256s"
	case AlgoHybridSecp256k1MLDSA:
		return "Hybrid-Secp256k1-MLDSA"
	case AlgoHybridSecp256k1MLKEM:
		return "Hybrid-Secp256k1-MLKEM"
	default:
		return "Unknown"
	}
}