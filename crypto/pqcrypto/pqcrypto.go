// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// Post-Quantum Cryptography Integration for Geth

package pqcrypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/crypto"
	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/mlkem"
	"github.com/luxfi/crypto/slhdsa"
)

// Algorithm types
type Algorithm uint8

const (
	AlgoClassical Algorithm = iota
	AlgoMLDSA44
	AlgoMLDSA65
	AlgoMLDSA87
	AlgoMLKEM512
	AlgoMLKEM768
	AlgoMLKEM1024
	AlgoSLHDSA128s
	AlgoSLHDSA192s
	AlgoSLHDSA256s
	AlgoHybridSecp256k1MLDSA
	AlgoHybridSecp256k1MLKEM
)

// PQSigner represents a post-quantum capable signer
type PQSigner struct {
	algo       Algorithm
	classical  *ecdsa.PrivateKey  // For hybrid modes
	mldsaPriv  *mldsa.PrivateKey
	mlkemPriv  *mlkem.PrivateKey
	slhdsaPriv *slhdsa.PrivateKey
}

// NewPQSigner creates a new post-quantum signer
func NewPQSigner(algo Algorithm) (*PQSigner, error) {
	signer := &PQSigner{algo: algo}
	
	switch algo {
	case AlgoClassical:
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		signer.classical = key
		
	case AlgoMLDSA44:
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
		if err != nil {
			return nil, err
		}
		signer.mldsaPriv = priv
		
	case AlgoMLDSA65:
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
		if err != nil {
			return nil, err
		}
		signer.mldsaPriv = priv
		
	case AlgoMLDSA87:
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
		if err != nil {
			return nil, err
		}
		signer.mldsaPriv = priv
		
	case AlgoMLKEM512:
		priv, err := mlkem.GenerateKeyPair(rand.Reader, mlkem.MLKEM512)
		if err != nil {
			return nil, err
		}
		signer.mlkemPriv = priv
		
	case AlgoMLKEM768:
		priv, err := mlkem.GenerateKeyPair(rand.Reader, mlkem.MLKEM768)
		if err != nil {
			return nil, err
		}
		signer.mlkemPriv = priv
		
	case AlgoMLKEM1024:
		priv, err := mlkem.GenerateKeyPair(rand.Reader, mlkem.MLKEM1024)
		if err != nil {
			return nil, err
		}
		signer.mlkemPriv = priv
		
	case AlgoSLHDSA128s:
		priv, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA128s)
		if err != nil {
			return nil, err
		}
		signer.slhdsaPriv = priv
		
	case AlgoSLHDSA192s:
		priv, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA192s)
		if err != nil {
			return nil, err
		}
		signer.slhdsaPriv = priv
		
	case AlgoSLHDSA256s:
		priv, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA256s)
		if err != nil {
			return nil, err
		}
		signer.slhdsaPriv = priv
		
	case AlgoHybridSecp256k1MLDSA:
		// Generate both classical and PQ keys
		classical, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		signer.classical = classical
		
		priv, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
		if err != nil {
			return nil, err
		}
		signer.mldsaPriv = priv
		
	case AlgoHybridSecp256k1MLKEM:
		// Generate both classical and KEM keys
		classical, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		signer.classical = classical
		
		priv, err := mlkem.GenerateKeyPair(rand.Reader, mlkem.MLKEM512)
		if err != nil {
			return nil, err
		}
		signer.mlkemPriv = priv
		
	default:
		return nil, fmt.Errorf("unsupported algorithm: %d", algo)
	}
	
	return signer, nil
}

// Sign signs a message using the appropriate algorithm
func (s *PQSigner) Sign(message []byte) ([]byte, error) {
	switch s.algo {
	case AlgoClassical:
		hash := crypto.Keccak256Hash(message)
		return crypto.Sign(hash.Bytes(), s.classical)
		
	case AlgoMLDSA44, AlgoMLDSA65, AlgoMLDSA87:
		return s.mldsaPriv.Sign(rand.Reader, message, nil)
		
	case AlgoSLHDSA128s, AlgoSLHDSA192s, AlgoSLHDSA256s:
		return s.slhdsaPriv.Sign(rand.Reader, message, nil)
		
	case AlgoHybridSecp256k1MLDSA:
		// Sign with both algorithms
		hash := crypto.Keccak256Hash(message)
		classicalSig, err := crypto.Sign(hash.Bytes(), s.classical)
		if err != nil {
			return nil, err
		}
		
		pqSig, err := s.mldsaPriv.Sign(rand.Reader, message, nil)
		if err != nil {
			return nil, err
		}
		
		// Concatenate signatures
		return append(classicalSig, pqSig...), nil
		
	default:
		return nil, errors.New("signing not supported for this algorithm")
	}
}

// Address returns the Ethereum address for this signer
func (s *PQSigner) Address() common.Address {
	switch s.algo {
	case AlgoClassical, AlgoHybridSecp256k1MLDSA, AlgoHybridSecp256k1MLKEM:
		addr := crypto.PubkeyToAddress(s.classical.PublicKey)
		return common.BytesToAddress(addr[:])
		
	case AlgoMLDSA44, AlgoMLDSA65, AlgoMLDSA87:
		// Use first 20 bytes of public key hash
		pubBytes := s.mldsaPriv.PublicKey.Bytes()
		hash := crypto.Keccak256(pubBytes)
		return common.BytesToAddress(hash[12:])
		
	case AlgoMLKEM512, AlgoMLKEM768, AlgoMLKEM1024:
		// Use first 20 bytes of public key hash
		pubBytes := s.mlkemPriv.PublicKey.Bytes()
		hash := crypto.Keccak256(pubBytes)
		return common.BytesToAddress(hash[12:])
		
	case AlgoSLHDSA128s, AlgoSLHDSA192s, AlgoSLHDSA256s:
		// Use first 20 bytes of public key hash
		pubBytes := s.slhdsaPriv.PublicKey.Bytes()
		hash := crypto.Keccak256(pubBytes)
		return common.BytesToAddress(hash[12:])
		
	default:
		return common.Address{}
	}
}

// Encapsulate performs key encapsulation (for KEM algorithms)
func (s *PQSigner) Encapsulate(pubKey []byte) ([]byte, []byte, error) {
	if s.mlkemPriv == nil {
		return nil, nil, errors.New("encapsulation requires ML-KEM key")
	}
	
	// Parse public key based on algorithm
	var mode mlkem.Mode
	switch s.algo {
	case AlgoMLKEM512, AlgoHybridSecp256k1MLKEM:
		mode = mlkem.MLKEM512
	case AlgoMLKEM768:
		mode = mlkem.MLKEM768
	case AlgoMLKEM1024:
		mode = mlkem.MLKEM1024
	default:
		return nil, nil, errors.New("not a KEM algorithm")
	}
	
	pub, err := mlkem.PublicKeyFromBytes(pubKey, mode)
	if err != nil {
		return nil, nil, err
	}
	
	result, err := pub.Encapsulate(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	
	return result.Ciphertext, result.SharedSecret, nil
}

// Decapsulate performs key decapsulation
func (s *PQSigner) Decapsulate(ciphertext []byte) ([]byte, error) {
	if s.mlkemPriv == nil {
		return nil, errors.New("decapsulation requires ML-KEM key")
	}
	
	return s.mlkemPriv.Decapsulate(ciphertext)
}