// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// Post-quantum key support for keystore

package keystore

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/crypto"
	"github.com/google/uuid"
	"github.com/luxfi/crypto/mldsa"
	"github.com/luxfi/crypto/mlkem"
	"github.com/luxfi/crypto/slhdsa"
)

// SignatureAlgorithm represents the type of signature algorithm
type SignatureAlgorithm uint8

const (
	SignatureECDSA SignatureAlgorithm = iota
	SignatureMLDSA44
	SignatureMLDSA65
	SignatureMLDSA87
	SignatureSLHDSA128s
	SignatureSLHDSA128f
	SignatureSLHDSA192s
	SignatureSLHDSA192f
	SignatureSLHDSA256s
	SignatureSLHDSA256f
	SignatureBLS // For validator keys
)

// PostQuantumKey represents a key that can use different signature algorithms
type PostQuantumKey struct {
	Id              uuid.UUID          `json:"id"`
	Address         common.Address     `json:"address"`
	Algorithm       SignatureAlgorithm `json:"algorithm"`
	
	// Traditional ECDSA (optional, for backward compatibility)
	ECDSAPrivateKey *ecdsa.PrivateKey  `json:"-"`
	
	// Post-quantum keys (one of these will be set based on Algorithm)
	MLDSAPrivateKey *mldsa.PrivateKey  `json:"-"`
	SLHDSAPrivateKey *slhdsa.PrivateKey `json:"-"`
	MLKEMPrivateKey *mlkem.PrivateKey  `json:"-"`
	
	// Serialized form for storage
	PrivateKeyBytes []byte             `json:"-"`
	PublicKeyBytes  []byte             `json:"-"`
}

// PostQuantumKeyJSON is the JSON representation
type postQuantumKeyJSON struct {
	Address    string `json:"address"`
	Algorithm  uint8  `json:"algorithm"`
	PrivateKey string `json:"privatekey"`
	PublicKey  string `json:"publickey"`
	Id         string `json:"id"`
	Version    int    `json:"version"`
}

// NewPostQuantumKey generates a new post-quantum key
func NewPostQuantumKey(algorithm SignatureAlgorithm) (*PostQuantumKey, error) {
	key := &PostQuantumKey{
		Id:        uuid.New(),
		Algorithm: algorithm,
	}
	
	switch algorithm {
	case SignatureECDSA:
		// Generate ECDSA key as before
		privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		key.ECDSAPrivateKey = privateKeyECDSA
		key.Address = common.BytesToAddress(crypto.PubkeyToAddress(privateKeyECDSA.PublicKey).Bytes())
		
	case SignatureMLDSA44:
		privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA44)
		if err != nil {
			return nil, err
		}
		key.MLDSAPrivateKey = privKey
		key.PrivateKeyBytes = privKey.Bytes()
		key.PublicKeyBytes = privKey.PublicKey.Bytes()
		// Derive address from public key hash
		key.Address = common.BytesToAddress(crypto.Keccak256(key.PublicKeyBytes)[:20])
		
	case SignatureMLDSA65:
		privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA65)
		if err != nil {
			return nil, err
		}
		key.MLDSAPrivateKey = privKey
		key.PrivateKeyBytes = privKey.Bytes()
		key.PublicKeyBytes = privKey.PublicKey.Bytes()
		key.Address = common.BytesToAddress(crypto.Keccak256(key.PublicKeyBytes)[:20])
		
	case SignatureMLDSA87:
		privKey, err := mldsa.GenerateKey(rand.Reader, mldsa.MLDSA87)
		if err != nil {
			return nil, err
		}
		key.MLDSAPrivateKey = privKey
		key.PrivateKeyBytes = privKey.Bytes()
		key.PublicKeyBytes = privKey.PublicKey.Bytes()
		key.Address = common.BytesToAddress(crypto.Keccak256(key.PublicKeyBytes)[:20])
		
	case SignatureSLHDSA128f:
		privKey, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA128f)
		if err != nil {
			return nil, err
		}
		key.SLHDSAPrivateKey = privKey
		key.PrivateKeyBytes = privKey.Bytes()
		key.PublicKeyBytes = privKey.PublicKey.Bytes()
		key.Address = common.BytesToAddress(crypto.Keccak256(key.PublicKeyBytes)[:20])
		
	case SignatureSLHDSA192f:
		privKey, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA192f)
		if err != nil {
			return nil, err
		}
		key.SLHDSAPrivateKey = privKey
		key.PrivateKeyBytes = privKey.Bytes()
		key.PublicKeyBytes = privKey.PublicKey.Bytes()
		key.Address = common.BytesToAddress(crypto.Keccak256(key.PublicKeyBytes)[:20])
		
	case SignatureSLHDSA256f:
		privKey, err := slhdsa.GenerateKey(rand.Reader, slhdsa.SLHDSA256f)
		if err != nil {
			return nil, err
		}
		key.SLHDSAPrivateKey = privKey
		key.PrivateKeyBytes = privKey.Bytes()
		key.PublicKeyBytes = privKey.PublicKey.Bytes()
		key.Address = common.BytesToAddress(crypto.Keccak256(key.PublicKeyBytes)[:20])
		
	default:
		return nil, fmt.Errorf("unsupported signature algorithm: %d", algorithm)
	}
	
	return key, nil
}

// Sign signs a message with the appropriate algorithm
func (k *PostQuantumKey) Sign(message []byte) ([]byte, error) {
	switch k.Algorithm {
	case SignatureECDSA:
		if k.ECDSAPrivateKey == nil {
			return nil, errors.New("ECDSA private key not set")
		}
		hash := crypto.Keccak256(message)
		return crypto.Sign(hash, k.ECDSAPrivateKey)
		
	case SignatureMLDSA44, SignatureMLDSA65, SignatureMLDSA87:
		if k.MLDSAPrivateKey == nil {
			return nil, errors.New("ML-DSA private key not set")
		}
		return k.MLDSAPrivateKey.Sign(rand.Reader, message, nil)
		
	case SignatureSLHDSA128f, SignatureSLHDSA192f, SignatureSLHDSA256f:
		if k.SLHDSAPrivateKey == nil {
			return nil, errors.New("SLH-DSA private key not set")
		}
		return k.SLHDSAPrivateKey.Sign(rand.Reader, message, nil)
		
	default:
		return nil, fmt.Errorf("unsupported signature algorithm: %d", k.Algorithm)
	}
}

// MarshalJSON serializes the key for storage
func (k *PostQuantumKey) MarshalJSON() ([]byte, error) {
	var privKeyHex string
	var pubKeyHex string
	
	switch k.Algorithm {
	case SignatureECDSA:
		if k.ECDSAPrivateKey != nil {
			privKeyHex = hex.EncodeToString(crypto.FromECDSA(k.ECDSAPrivateKey))
			pubKeyHex = hex.EncodeToString(crypto.FromECDSAPub(&k.ECDSAPrivateKey.PublicKey))
		}
	default:
		privKeyHex = hex.EncodeToString(k.PrivateKeyBytes)
		pubKeyHex = hex.EncodeToString(k.PublicKeyBytes)
	}
	
	return json.Marshal(&postQuantumKeyJSON{
		Address:    k.Address.Hex(),
		Algorithm:  uint8(k.Algorithm),
		PrivateKey: privKeyHex,
		PublicKey:  pubKeyHex,
		Id:         k.Id.String(),
		Version:    4, // New version for PQ keys
	})
}

// UnmarshalJSON deserializes the key from storage
func (k *PostQuantumKey) UnmarshalJSON(data []byte) error {
	var keyJSON postQuantumKeyJSON
	if err := json.Unmarshal(data, &keyJSON); err != nil {
		return err
	}
	
	k.Id, _ = uuid.Parse(keyJSON.Id)
	k.Address = common.HexToAddress(keyJSON.Address)
	k.Algorithm = SignatureAlgorithm(keyJSON.Algorithm)
	
	privKeyBytes, err := hex.DecodeString(keyJSON.PrivateKey)
	if err != nil {
		return err
	}
	
	pubKeyBytes, err := hex.DecodeString(keyJSON.PublicKey)
	if err != nil {
		return err
	}
	
	k.PrivateKeyBytes = privKeyBytes
	k.PublicKeyBytes = pubKeyBytes
	
	// Reconstruct the actual key objects based on algorithm
	switch k.Algorithm {
	case SignatureECDSA:
		key, err := crypto.ToECDSA(privKeyBytes)
		if err != nil {
			return err
		}
		k.ECDSAPrivateKey = key
		
	case SignatureMLDSA44:
		key, err := mldsa.PrivateKeyFromBytes(privKeyBytes, mldsa.MLDSA44)
		if err != nil {
			return err
		}
		k.MLDSAPrivateKey = key
		
	case SignatureMLDSA65:
		key, err := mldsa.PrivateKeyFromBytes(privKeyBytes, mldsa.MLDSA65)
		if err != nil {
			return err
		}
		k.MLDSAPrivateKey = key
		
	case SignatureMLDSA87:
		key, err := mldsa.PrivateKeyFromBytes(privKeyBytes, mldsa.MLDSA87)
		if err != nil {
			return err
		}
		k.MLDSAPrivateKey = key
		
	case SignatureSLHDSA128f:
		key, err := slhdsa.PrivateKeyFromBytes(privKeyBytes, slhdsa.SLHDSA128f)
		if err != nil {
			return err
		}
		k.SLHDSAPrivateKey = key
		
	case SignatureSLHDSA192f:
		key, err := slhdsa.PrivateKeyFromBytes(privKeyBytes, slhdsa.SLHDSA192f)
		if err != nil {
			return err
		}
		k.SLHDSAPrivateKey = key
		
	case SignatureSLHDSA256f:
		key, err := slhdsa.PrivateKeyFromBytes(privKeyBytes, slhdsa.SLHDSA256f)
		if err != nil {
			return err
		}
		k.SLHDSAPrivateKey = key
	}
	
	return nil
}

// GetAlgorithmName returns human-readable algorithm name
func GetAlgorithmName(alg SignatureAlgorithm) string {
	switch alg {
	case SignatureECDSA:
		return "ECDSA (secp256k1)"
	case SignatureMLDSA44:
		return "ML-DSA-44 (NIST Level 2)"
	case SignatureMLDSA65:
		return "ML-DSA-65 (NIST Level 3)"
	case SignatureMLDSA87:
		return "ML-DSA-87 (NIST Level 5)"
	case SignatureSLHDSA128s:
		return "SLH-DSA-128s (Small)"
	case SignatureSLHDSA128f:
		return "SLH-DSA-128f (Fast)"
	case SignatureSLHDSA192s:
		return "SLH-DSA-192s (Small)"
	case SignatureSLHDSA192f:
		return "SLH-DSA-192f (Fast)"
	case SignatureSLHDSA256s:
		return "SLH-DSA-256s (Small)"
	case SignatureSLHDSA256f:
		return "SLH-DSA-256f (Fast)"
	case SignatureBLS:
		return "BLS12-381"
	default:
		return "Unknown"
	}
}

// GetKeySizes returns the key and signature sizes for an algorithm
func GetKeySizes(alg SignatureAlgorithm) (privKeySize, pubKeySize, sigSize int) {
	switch alg {
	case SignatureECDSA:
		return 32, 64, 65
	case SignatureMLDSA44:
		return mldsa.MLDSA44PrivateKeySize, mldsa.MLDSA44PublicKeySize, mldsa.MLDSA44SignatureSize
	case SignatureMLDSA65:
		return mldsa.MLDSA65PrivateKeySize, mldsa.MLDSA65PublicKeySize, mldsa.MLDSA65SignatureSize
	case SignatureMLDSA87:
		return mldsa.MLDSA87PrivateKeySize, mldsa.MLDSA87PublicKeySize, mldsa.MLDSA87SignatureSize
	case SignatureSLHDSA128f:
		return slhdsa.SLHDSA128fPrivateKeySize, slhdsa.SLHDSA128fPublicKeySize, slhdsa.SLHDSA128fSignatureSize
	case SignatureSLHDSA192f:
		return slhdsa.SLHDSA192fPrivateKeySize, slhdsa.SLHDSA192fPublicKeySize, slhdsa.SLHDSA192fSignatureSize
	case SignatureSLHDSA256f:
		return slhdsa.SLHDSA256fPrivateKeySize, slhdsa.SLHDSA256fPublicKeySize, slhdsa.SLHDSA256fSignatureSize
	default:
		return 0, 0, 0
	}
}