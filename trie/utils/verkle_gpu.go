// Copyright 2025 Lux Industries, Inc.
// This file is part of the lux geth library.

//go:build cgo

// Package utils provides GPU-accelerated Verkle tree operations.
// When CGO is enabled, polynomial commitments use Metal GPU acceleration
// for significant speedup on Apple Silicon and compatible hardware.
package utils

/*
#cgo CFLAGS: -I${SRCDIR}/../../../luxcpp/crypto/include
#cgo LDFLAGS: -L${SRCDIR}/../../../luxcpp/crypto/build-local -lluxcrypto -framework Metal -framework Foundation

#include <stdint.h>
#include <stdbool.h>
#include "lux/crypto/metal_ipa.h"
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"

	"github.com/crate-crypto/go-ipa/bandersnatch/fr"
	"github.com/ethereum/go-verkle"
)

// GPU acceleration thresholds
const (
	// MinPolyDegreeGPU is the minimum polynomial degree for GPU acceleration
	MinPolyDegreeGPU = 4

	// BatchCommitThreshold is minimum batch size for GPU batch commits
	BatchCommitThreshold = 8
)

var (
	gpuCtx      *C.MetalIPAContext
	gpuCtxOnce  sync.Once
	gpuCtxErr   error
	gpuCtxMu    sync.Mutex
	gpuBasisSet bool
)

// initGPU initializes the Metal IPA context for GPU acceleration.
func initGPU() error {
	gpuCtxOnce.Do(func() {
		gpuCtx = C.metal_ipa_init()
		if gpuCtx == nil {
			gpuCtxErr = errors.New("failed to initialize Metal IPA context")
		}
	})
	return gpuCtxErr
}

// GPUAvailable returns true if GPU acceleration is available.
func GPUAvailable() bool {
	return bool(C.metal_ipa_available())
}

// GPUCommitToPoly performs polynomial commitment using GPU acceleration.
// Falls back to CPU if GPU is unavailable or for small polynomials.
func GPUCommitToPoly(poly []fr.Element) (*verkle.Point, error) {
	n := len(poly)
	if n == 0 {
		return nil, errors.New("empty polynomial")
	}

	// Use CPU for small polynomials
	if n < MinPolyDegreeGPU || !GPUAvailable() {
		cfg := verkle.GetConfig()
		return cfg.CommitToPoly(poly, 0), nil
	}

	if err := initGPU(); err != nil {
		cfg := verkle.GetConfig()
		return cfg.CommitToPoly(poly, 0), nil
	}

	gpuCtxMu.Lock()
	defer gpuCtxMu.Unlock()

	// Convert scalars to C format
	cScalars := make([]C.BanderwagonScalar, n)
	for i := 0; i < n; i++ {
		bytes := poly[i].Bytes()
		for j := 0; j < 32; j++ {
			cScalars[i].bytes[j] = C.uint8_t(bytes[j])
		}
	}

	// Perform GPU commitment (uses precomputed Lagrange basis)
	var cResult C.BanderwagonExtended

	ret := C.metal_ipa_pedersen_commit(
		gpuCtx,
		&cResult,
		(*C.BanderwagonScalar)(unsafe.Pointer(&cScalars[0])),
		C.uint32_t(n),
	)

	if ret != C.METAL_IPA_SUCCESS {
		// Fall back to CPU
		cfg := verkle.GetConfig()
		return cfg.CommitToPoly(poly, 0), nil
	}

	// Convert result to verkle.Point
	var resultBytes [32]byte
	for i := 0; i < 32; i++ {
		resultBytes[i] = byte(cResult.x.bytes[i])
	}

	point := new(verkle.Point)
	if err := point.SetBytes(resultBytes[:]); err != nil {
		// Fall back to CPU on conversion error
		cfg := verkle.GetConfig()
		return cfg.CommitToPoly(poly, 0), nil
	}

	return point, nil
}

// GPUBatchCommitToPoly performs batch polynomial commitments using GPU.
// Much more efficient than sequential commits for multiple polynomials.
func GPUBatchCommitToPoly(polys [][]fr.Element) ([]*verkle.Point, error) {
	n := len(polys)
	if n == 0 {
		return nil, errors.New("empty polynomial batch")
	}

	// Use sequential CPU for small batches
	if n < BatchCommitThreshold || !GPUAvailable() {
		results := make([]*verkle.Point, n)
		cfg := verkle.GetConfig()
		for i, poly := range polys {
			results[i] = cfg.CommitToPoly(poly, 0)
		}
		return results, nil
	}

	if err := initGPU(); err != nil {
		results := make([]*verkle.Point, n)
		cfg := verkle.GetConfig()
		for i, poly := range polys {
			results[i] = cfg.CommitToPoly(poly, 0)
		}
		return results, nil
	}

	gpuCtxMu.Lock()
	defer gpuCtxMu.Unlock()

	// Prepare batch data
	counts := make([]C.uint32_t, n)
	totalScalars := 0
	for i, poly := range polys {
		counts[i] = C.uint32_t(len(poly))
		totalScalars += len(poly)
	}

	// Flatten all scalars
	allScalars := make([]C.BanderwagonScalar, totalScalars)
	idx := 0
	for _, poly := range polys {
		for _, s := range poly {
			bytes := s.Bytes()
			for j := 0; j < 32; j++ {
				allScalars[idx].bytes[j] = C.uint8_t(bytes[j])
			}
			idx++
		}
	}

	// Allocate results
	cResults := make([]C.BanderwagonExtended, n)

	ret := C.metal_ipa_batch_pedersen_commit(
		gpuCtx,
		(*C.BanderwagonExtended)(unsafe.Pointer(&cResults[0])),
		(*C.BanderwagonScalar)(unsafe.Pointer(&allScalars[0])),
		(*C.uint32_t)(unsafe.Pointer(&counts[0])),
		C.uint32_t(n),
	)

	if ret != C.METAL_IPA_SUCCESS {
		// Fall back to CPU
		results := make([]*verkle.Point, n)
		cfg := verkle.GetConfig()
		for i, poly := range polys {
			results[i] = cfg.CommitToPoly(poly, 0)
		}
		return results, nil
	}

	// Convert results
	results := make([]*verkle.Point, n)
	for i := 0; i < n; i++ {
		var resultBytes [32]byte
		for j := 0; j < 32; j++ {
			resultBytes[j] = byte(cResults[i].x.bytes[j])
		}

		point := new(verkle.Point)
		if err := point.SetBytes(resultBytes[:]); err != nil {
			// Fall back to CPU for this one
			cfg := verkle.GetConfig()
			results[i] = cfg.CommitToPoly(polys[i], 0)
		} else {
			results[i] = point
		}
	}

	return results, nil
}

// GPUVerifyProof verifies an IPA proof using GPU acceleration.
func GPUVerifyProof(
	commitment *verkle.Point,
	proof []byte,
	point fr.Element,
	evaluation fr.Element,
) (bool, error) {
	if !GPUAvailable() {
		// Fall back to CPU verification
		return false, errors.New("GPU not available, use CPU verification")
	}

	if err := initGPU(); err != nil {
		return false, err
	}

	gpuCtxMu.Lock()
	defer gpuCtxMu.Unlock()

	// Convert commitment
	var cCommit C.BanderwagonExtended
	commitBytes := commitment.Bytes()
	for i := 0; i < 32; i++ {
		cCommit.x.bytes[i] = C.uint8_t(commitBytes[i])
	}

	// Convert point and evaluation
	var cPoint, cEval C.BanderwagonScalar
	pointBytes := point.Bytes()
	evalBytes := evaluation.Bytes()
	for i := 0; i < 32; i++ {
		cPoint.bytes[i] = C.uint8_t(pointBytes[i])
		cEval.bytes[i] = C.uint8_t(evalBytes[i])
	}

	// Prepare proof
	if len(proof) == 0 {
		return false, errors.New("empty proof")
	}
	cProof := (*C.uint8_t)(unsafe.Pointer(&proof[0]))

	ret := C.metal_ipa_verify(
		gpuCtx,
		&cCommit,
		cProof,
		C.uint32_t(len(proof)),
		&cPoint,
		&cEval,
	)

	return ret == C.METAL_IPA_SUCCESS, nil
}

// GPUBatchVerifyProofs verifies multiple IPA proofs in parallel on GPU.
func GPUBatchVerifyProofs(
	commitments []*verkle.Point,
	proofs [][]byte,
	points []fr.Element,
	evaluations []fr.Element,
) ([]bool, error) {
	n := len(commitments)
	if n == 0 || n != len(proofs) || n != len(points) || n != len(evaluations) {
		return nil, errors.New("mismatched input lengths")
	}

	if !GPUAvailable() {
		return nil, errors.New("GPU not available")
	}

	if err := initGPU(); err != nil {
		return nil, err
	}

	gpuCtxMu.Lock()
	defer gpuCtxMu.Unlock()

	// Convert commitments
	cCommits := make([]C.BanderwagonExtended, n)
	for i := 0; i < n; i++ {
		bytes := commitments[i].Bytes()
		for j := 0; j < 32; j++ {
			cCommits[i].x.bytes[j] = C.uint8_t(bytes[j])
		}
	}

	// Convert points and evaluations
	cPoints := make([]C.BanderwagonScalar, n)
	cEvals := make([]C.BanderwagonScalar, n)
	for i := 0; i < n; i++ {
		pBytes := points[i].Bytes()
		eBytes := evaluations[i].Bytes()
		for j := 0; j < 32; j++ {
			cPoints[i].bytes[j] = C.uint8_t(pBytes[j])
			cEvals[i].bytes[j] = C.uint8_t(eBytes[j])
		}
	}

	// Prepare proofs
	proofPtrs := make([]*C.uint8_t, n)
	proofLens := make([]C.uint32_t, n)
	for i := 0; i < n; i++ {
		if len(proofs[i]) > 0 {
			proofPtrs[i] = (*C.uint8_t)(unsafe.Pointer(&proofs[i][0]))
		}
		proofLens[i] = C.uint32_t(len(proofs[i]))
	}

	// Allocate results
	cResults := make([]C.int, n)

	ret := C.metal_ipa_batch_verify(
		gpuCtx,
		(*C.BanderwagonExtended)(unsafe.Pointer(&cCommits[0])),
		(**C.uint8_t)(unsafe.Pointer(&proofPtrs[0])),
		(*C.uint32_t)(unsafe.Pointer(&proofLens[0])),
		(*C.BanderwagonScalar)(unsafe.Pointer(&cPoints[0])),
		(*C.BanderwagonScalar)(unsafe.Pointer(&cEvals[0])),
		C.uint32_t(n),
		(*C.int)(unsafe.Pointer(&cResults[0])),
	)

	if ret != C.METAL_IPA_SUCCESS {
		return nil, errors.New("batch verification failed")
	}

	results := make([]bool, n)
	for i := 0; i < n; i++ {
		results[i] = cResults[i] != 0
	}

	return results, nil
}

// DestroyGPU releases the Metal IPA context.
// Should be called when shutting down.
func DestroyGPU() {
	gpuCtxMu.Lock()
	defer gpuCtxMu.Unlock()

	if gpuCtx != nil {
		C.metal_ipa_destroy(gpuCtx)
		gpuCtx = nil
	}
}
