// Copyright 2025 Lux Industries, Inc.
// This file is part of the lux geth library.

//go:build !cgo

// Package utils provides Verkle tree utilities.
// This is the pure Go fallback when CGO is not available.
package utils

import (
	"errors"

	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
	"github.com/ethereum/go-verkle"
)

// GPU acceleration thresholds (same as CGO version for API compatibility)
const (
	MinPolyDegreeGPU     = 4
	BatchCommitThreshold = 8
)

// GPUAvailable returns false when CGO is not available.
func GPUAvailable() bool {
	return false
}

// GPUCommitToPoly uses CPU-based polynomial commitment.
func GPUCommitToPoly(poly []fr.Element) (*verkle.Point, error) {
	if len(poly) == 0 {
		return nil, errors.New("empty polynomial")
	}
	cfg := verkle.GetConfig()
	return cfg.CommitToPoly(poly, 0), nil
}

// GPUBatchCommitToPoly uses sequential CPU-based polynomial commitment.
func GPUBatchCommitToPoly(polys [][]fr.Element) ([]*verkle.Point, error) {
	if len(polys) == 0 {
		return nil, errors.New("empty polynomial batch")
	}
	results := make([]*verkle.Point, len(polys))
	cfg := verkle.GetConfig()
	for i, poly := range polys {
		results[i] = cfg.CommitToPoly(poly, 0)
	}
	return results, nil
}

// GPUVerifyProof is not available without CGO.
func GPUVerifyProof(
	commitment *verkle.Point,
	proof []byte,
	point fr.Element,
	evaluation fr.Element,
) (bool, error) {
	return false, errors.New("GPU verification not available without CGO")
}

// GPUBatchVerifyProofs is not available without CGO.
func GPUBatchVerifyProofs(
	commitments []*verkle.Point,
	proofs [][]byte,
	points []fr.Element,
	evaluations []fr.Element,
) ([]bool, error) {
	return nil, errors.New("GPU batch verification not available without CGO")
}

// DestroyGPU is a no-op without CGO.
func DestroyGPU() {}
