// Package crypto provides wrapper types for go-ethereum's crypto implementation
package crypto

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/luxfi/geth/common"
)

// Re-export constants
const (
	SignatureLength = crypto.SignatureLength
	DigestLength    = crypto.DigestLength
	HashLength      = 32 // SHA3-256 hash length
)

// Re-export types
type (
	KeccakState = crypto.KeccakState
)

// Re-export functions
var (
	Keccak256               = crypto.Keccak256
	Keccak256Hash           = crypto.Keccak256Hash
	CreateAddress           = crypto.CreateAddress
	CreateAddress2          = crypto.CreateAddress2
	ToECDSA                 = crypto.ToECDSA
	ToECDSAUnsafe           = crypto.ToECDSAUnsafe
	FromECDSA               = crypto.FromECDSA
	UnmarshalPubkey         = crypto.UnmarshalPubkey
	FromECDSAPub            = crypto.FromECDSAPub
	HexToECDSA              = crypto.HexToECDSA
	LoadECDSA               = crypto.LoadECDSA
	SaveECDSA               = crypto.SaveECDSA
	GenerateKey             = crypto.GenerateKey
	PubkeyToAddress         = crypto.PubkeyToAddress
	HashData                = crypto.HashData
	NewKeccakState          = crypto.NewKeccakState
	Sign                    = crypto.Sign
	VerifySignature         = crypto.VerifySignature
	Ecrecover               = crypto.Ecrecover
	SigToPub                = crypto.SigToPub
	ValidateSignatureValues = crypto.ValidateSignatureValues
)

// Type aliases for convenience
type Hash = common.Hash
