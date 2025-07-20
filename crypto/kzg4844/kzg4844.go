// Package kzg4844 provides wrapper types for go-ethereum's kzg4844 implementation
package kzg4844

import (
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// Re-export types
type (
	Blob = kzg4844.Blob
	Commitment = kzg4844.Commitment
	Proof = kzg4844.Proof
	Point = kzg4844.Point
	Claim = kzg4844.Claim
)

// Re-export functions
var (
	BlobToCommitment = kzg4844.BlobToCommitment
	ComputeProof = kzg4844.ComputeProof
	VerifyProof = kzg4844.VerifyProof
	CalcBlobHashV1 = kzg4844.CalcBlobHashV1
)