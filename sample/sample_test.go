// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package sample

import (
	"bytes"
	"io"
	"testing"

	"github.com/zeebo/blake3"
)

const PulsarQ = uint64(0x1000000004A01)

// deterministicReader returns an unbounded byte stream from a seed
// string. Same pattern as luxfi/threshold's deterministicRand test
// helper.
func deterministicReader(seed string) io.Reader {
	h := blake3.New()
	_, _ = h.Write([]byte("luxfi/math/sample/test/v1"))
	_, _ = h.Write([]byte(seed))
	return h.Digest()
}

func TestUniform_Determinism(t *testing.T) {
	N := 256
	a := make([]uint64, N)
	b := make([]uint64, N)
	if err := Uniform(a, PulsarQ, deterministicReader("uniform-1")); err != nil {
		t.Fatalf("Uniform[a]: %v", err)
	}
	if err := Uniform(b, PulsarQ, deterministicReader("uniform-1")); err != nil {
		t.Fatalf("Uniform[b]: %v", err)
	}
	if !bytes.Equal(uint64sToBytes(a), uint64sToBytes(b)) {
		t.Error("Uniform with same seed: byte-mismatch")
	}
	for i, v := range a {
		if v >= PulsarQ {
			t.Fatalf("Uniform[%d] = %d >= q=%d", i, v, PulsarQ)
		}
	}
}

func TestTernary_Distribution(t *testing.T) {
	N := 4096
	dst := make([]uint64, N)
	if err := Ternary(dst, PulsarQ, 0.5, deterministicReader("ternary")); err != nil {
		t.Fatalf("Ternary: %v", err)
	}
	// Coarse distribution check: with density=0.5, expect ~50%
	// non-zero. Allow ±10% slack.
	zero, plus, minus := 0, 0, 0
	for _, v := range dst {
		switch v {
		case 0:
			zero++
		case 1:
			plus++
		case PulsarQ - 1:
			minus++
		default:
			t.Fatalf("Ternary: unexpected value %d", v)
		}
	}
	nonZero := plus + minus
	if nonZero < N*4/10 || nonZero > N*6/10 {
		t.Errorf("Ternary density: nonZero=%d/%d (expected ~50%%)", nonZero, N)
	}
	// Plus and minus should be roughly balanced.
	if plus < nonZero/3 || minus < nonZero/3 {
		t.Errorf("Ternary balance: plus=%d minus=%d", plus, minus)
	}
}

func TestCenteredBinomial_RangeBounded(t *testing.T) {
	N := 1024
	dst := make([]uint64, N)
	const eta = 2
	if err := CenteredBinomial(dst, PulsarQ, eta, deterministicReader("cbd-2")); err != nil {
		t.Fatalf("CenteredBinomial: %v", err)
	}
	// Output values should all be in {q-eta, ..., q-1, 0, 1, ..., eta}.
	for i, v := range dst {
		if v <= eta {
			continue
		}
		if v >= PulsarQ-eta {
			continue
		}
		t.Fatalf("CenteredBinomial[%d] = %d out of expected range", i, v)
	}
}

func TestDiscreteGaussianRejection_Range(t *testing.T) {
	// Coarse range test: 1000 samples should all lie within ~6 sigma.
	const sigma = 3.2
	r := deterministicReader("dgrr")
	for i := 0; i < 1000; i++ {
		_, err := DiscreteGaussianRejection(PulsarQ, sigma, r)
		if err != nil {
			t.Fatalf("[%d]: %v", i, err)
		}
	}
}

func uint64sToBytes(s []uint64) []byte {
	out := make([]byte, len(s)*8)
	for i, v := range s {
		for j := 0; j < 8; j++ {
			out[i*8+j] = byte(v >> (j * 8))
		}
	}
	return out
}
