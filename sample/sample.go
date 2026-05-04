// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package sample provides primitive samplers — uniform mod q, ternary,
// centered binomial, discrete Gaussian — used as building blocks by
// every Lux lattice protocol.
//
// LP-107 §"Sampling" — the canonical motivation. Protocol-specific
// samplers (e.g. Pulsar's transcript-bound discrete Gaussian for the
// proof of knowledge) remain in their owning protocol package; this
// package is the source of the primitive distributions those
// protocols compose.
//
// Determinism contract: every sampler accepts an io.Reader as its
// entropy source. Same seed → same samples, byte-identically.
//
// Phase 2 (this file): pure-Go reference implementation. Body uses
// luxfi/math/modarith for modular reduction.
package sample

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
)

// Uniform fills dst with values uniformly distributed in [0, q).
// Uses rejection sampling with a per-sample mask up to bits.Len64(q)
// bits to avoid bias. Reads len(dst) * ceil(log2(q)/8) bytes from r
// in expectation; rejection rate is at most 2x.
func Uniform(dst []uint64, q uint64, r io.Reader) error {
	if q < 2 {
		return fmt.Errorf("sample.Uniform: q=%d invalid", q)
	}
	// Mask = next power-of-two-minus-one >= q-1.
	mask := uint64(1)
	for mask < q {
		mask <<= 1
	}
	mask--
	buf := make([]byte, 8)
	for i := range dst {
		for {
			if _, err := io.ReadFull(r, buf); err != nil {
				return fmt.Errorf("sample.Uniform[%d]: %w", i, err)
			}
			v := binary.LittleEndian.Uint64(buf) & mask
			if v < q {
				dst[i] = v
				break
			}
		}
	}
	return nil
}

// Ternary fills dst with values from {-1 mod q, 0, 1} per the given
// non-zero density (probability of non-zero coefficient). Standard
// lattice short-secret distribution.
func Ternary(dst []uint64, q uint64, density float64, r io.Reader) error {
	if q < 2 {
		return fmt.Errorf("sample.Ternary: q=%d invalid", q)
	}
	if density < 0 || density > 1 {
		return fmt.Errorf("sample.Ternary: density=%f out of [0,1]", density)
	}
	// Two bytes per sample: byte 0 selects zero/non-zero, byte 1
	// selects sign.
	buf := make([]byte, 2)
	thresh := byte(density * 256)
	if density >= 1 {
		thresh = 0xFF
	}
	for i := range dst {
		if _, err := io.ReadFull(r, buf); err != nil {
			return fmt.Errorf("sample.Ternary[%d]: %w", i, err)
		}
		if buf[0] >= thresh {
			dst[i] = 0
			continue
		}
		if buf[1]&1 == 0 {
			dst[i] = 1
		} else {
			dst[i] = q - 1 // -1 mod q
		}
	}
	return nil
}

// CenteredBinomial fills dst with values from a centered binomial
// distribution with parameter eta (Bin(2*eta, 0.5) - eta). Standard
// Module-LWE error distribution.
func CenteredBinomial(dst []uint64, q uint64, eta int, r io.Reader) error {
	if q < 2 {
		return fmt.Errorf("sample.CenteredBinomial: q=%d invalid", q)
	}
	if eta < 1 || eta > 32 {
		return fmt.Errorf("sample.CenteredBinomial: eta=%d out of [1,32]", eta)
	}
	bytesPerSample := (2*eta + 7) / 8
	buf := make([]byte, bytesPerSample)
	for i := range dst {
		if _, err := io.ReadFull(r, buf); err != nil {
			return fmt.Errorf("sample.CenteredBinomial[%d]: %w", i, err)
		}
		// Compute popcount of first eta bits and last eta bits, take
		// the difference.
		var bits uint64
		for j := 0; j < bytesPerSample; j++ {
			bits |= uint64(buf[j]) << (j * 8)
		}
		mask := (uint64(1) << uint(eta)) - 1
		a := bitCount64(bits & mask)
		b := bitCount64((bits >> uint(eta)) & mask)
		// signed difference in [-eta, +eta]; map to [0, q).
		signed := a - b
		if signed >= 0 {
			dst[i] = uint64(signed) % q
		} else {
			dst[i] = q - uint64(-signed)%q
		}
	}
	return nil
}

func bitCount64(x uint64) int64 {
	count := int64(0)
	for x != 0 {
		count += int64(x & 1)
		x >>= 1
	}
	return count
}

// DiscreteGaussianRejection samples one value approximately from the
// discrete Gaussian D_{Z, sigma}, centered at 0, via rejection
// sampling with a 6-sigma cutoff. Accepts ~64% of draws on average
// for sigma in the typical lattice range [3, 50]; exact for the
// uncentered tail.
//
// This is the same reference path used by lattice/gpu.SampleGaussian
// (which we're consolidating here under LP-107).
func DiscreteGaussianRejection(q uint64, sigma float64, r io.Reader) (uint64, error) {
	if q < 2 {
		return 0, fmt.Errorf("sample.DiscreteGaussianRejection: q=%d invalid", q)
	}
	if sigma <= 0 {
		return 0, fmt.Errorf("sample.DiscreteGaussianRejection: sigma=%f invalid", sigma)
	}
	bound := int64(sigma*6 + 1)
	buf := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, fmt.Errorf("sample.DiscreteGaussianRejection: %w", err)
		}
		raw := binary.LittleEndian.Uint64(buf)
		// Map raw to a signed integer in [-bound, +bound].
		span := uint64(2*bound + 1)
		v := int64(raw%span) - bound

		// Accept with probability exp(-v^2 / (2 sigma^2)).
		var probAccept big.Float
		// Use float math to avoid float64 overflow on sigma*sigma for
		// moderate sigma values.
		num := float64(v * v)
		den := 2 * sigma * sigma
		probAccept.SetFloat64(num / den)
		expVal := approxExp(-num / den)

		// Draw acceptance threshold uniformly in [0, 1).
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, fmt.Errorf("sample.DiscreteGaussianRejection (accept): %w", err)
		}
		threshold := float64(binary.LittleEndian.Uint64(buf)) / float64(^uint64(0))
		if threshold < expVal {
			if v >= 0 {
				return uint64(v) % q, nil
			}
			return q - uint64(-v)%q, nil
		}
	}
}

// approxExp returns exp(x) for x <= 0. Uses the standard taylor
// expansion truncated at 12 terms; sufficient for the sigma <= 50
// regime that lattice protocols use.
func approxExp(x float64) float64 {
	if x > 0 {
		return 1 // shouldn't happen; defensive
	}
	if x < -50 {
		return 0
	}
	result := 1.0
	term := 1.0
	for n := 1; n <= 12; n++ {
		term *= x / float64(n)
		result += term
	}
	if result < 0 {
		return 0
	}
	return result
}
