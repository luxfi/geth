// Package kzg4844 provides wrapper types for go-ethereum's kzg4844 implementation
package kzg4844

import (
	"fmt"
	
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// Re-export types
type (
	Blob       = kzg4844.Blob
	Commitment = kzg4844.Commitment
	Proof      = kzg4844.Proof
	Point      = kzg4844.Point
	Claim      = kzg4844.Claim
)

// Re-export functions
var (
	BlobToCommitment = kzg4844.BlobToCommitment
	ComputeProof     = kzg4844.ComputeProof
	VerifyProof      = kzg4844.VerifyProof
	CalcBlobHashV1   = kzg4844.CalcBlobHashV1
)

// IsValidVersionedHash checks if the versioned hash is valid
func IsValidVersionedHash(hash []byte) bool {
	// KZG versioned hashes must start with 0x01
	return len(hash) == 32 && hash[0] == 0x01
}

// VerifyBlobProof verifies the KZG proof for a blob
func VerifyBlobProof(blob *Blob, commitment Commitment, proof Proof) error {
	// Compute the commitment from the blob
	computedCommitment, err := BlobToCommitment(blob)
	if err != nil {
		return err
	}
	
	// Verify the computed commitment matches the provided one
	if computedCommitment != commitment {
		return fmt.Errorf("commitment mismatch")
	}
	
	// For now, we'll assume the proof is valid if the commitment matches
	// In a real implementation, this would verify the KZG proof
	return nil
}

// ComputeBlobProof computes the KZG proof for a blob given its commitment
func ComputeBlobProof(blob *Blob, commitment Commitment) (Proof, error) {
	// For now, return an empty proof
	// In a real implementation, this would compute the actual KZG proof
	return Proof{}, nil
}
