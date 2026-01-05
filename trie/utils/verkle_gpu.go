// Copyright 2025 Lux Industries, Inc.
// This file is part of the lux geth library.

// Package utils provides Verkle tree utilities with GPU acceleration.
// GPU operations are delegated to github.com/luxfi/crypto/verkle/gpu.
package utils

import (
	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
	"github.com/ethereum/go-verkle"
	"github.com/luxfi/crypto/verkle/gpu"
)

// GPU acceleration thresholds - exposed for compatibility
const (
	MinPolyDegreeGPU     = gpu.MinPolyDegreeGPU
	BatchCommitThreshold = gpu.BatchCommitThreshold
)

// GPUAvailable returns true if GPU acceleration is available.
func GPUAvailable() bool {
	return gpu.Available()
}

// GPUCommitToPoly performs polynomial commitment using GPU acceleration.
func GPUCommitToPoly(poly []fr.Element) (*verkle.Point, error) {
	return gpu.CommitToPoly(poly)
}

// GPUBatchCommitToPoly performs batch polynomial commitments using GPU.
func GPUBatchCommitToPoly(polys [][]fr.Element) ([]*verkle.Point, error) {
	return gpu.BatchCommitToPoly(polys)
}

// GPUVerifyProof verifies an IPA proof using GPU acceleration.
func GPUVerifyProof(
	commitment *verkle.Point,
	proof []byte,
	point fr.Element,
	evaluation fr.Element,
) (bool, error) {
	return gpu.VerifyProof(commitment, proof, point, evaluation)
}

// GPUBatchVerifyProofs verifies multiple IPA proofs in parallel on GPU.
func GPUBatchVerifyProofs(
	commitments []*verkle.Point,
	proofs [][]byte,
	points []fr.Element,
	evaluations []fr.Element,
) ([]bool, error) {
	return gpu.BatchVerifyProofs(commitments, proofs, points, evaluations)
}

// DestroyGPU releases the Metal IPA context.
func DestroyGPU() {
	gpu.Destroy()
}
