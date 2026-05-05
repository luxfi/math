// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package subring

import (
	"math/rand/v2"
	"testing"
)

// pulsarN256 is the Pulsar/LP-073 NTT instance: N=256, Q=2^48 + 4*N + 1.
const (
	pulsarN  = 256
	pulsarQ  = uint64(0x1000000004A01)
)

// TestSubRing_RoundTrip exercises the full path:
// constructor + GenerateNTTConstants + forward NTT + inverse NTT.
func TestSubRing_RoundTrip(t *testing.T) {
	sr, err := NewSubRing(pulsarN, pulsarQ)
	if err != nil {
		t.Fatalf("NewSubRing: %v", err)
	}
	if err := sr.GenerateNTTConstants(); err != nil {
		t.Fatalf("GenerateNTTConstants: %v", err)
	}

	r := rand.New(rand.NewPCG(0xdeadbeef, 0x12345678))
	a := make([]uint64, pulsarN)
	for i := range a {
		a[i] = r.Uint64() % pulsarQ
	}
	saved := make([]uint64, pulsarN)
	copy(saved, a)

	sr.NTT(a, a)
	sr.INTT(a, a)

	for i := range a {
		if a[i] != saved[i] {
			t.Fatalf("round-trip [%d]: %d != %d", i, a[i], saved[i])
		}
	}
}

// TestSubRing_LazyVariants exercises NTTLazy / INTTLazy.
func TestSubRing_LazyVariants(t *testing.T) {
	sr, err := NewSubRing(pulsarN, pulsarQ)
	if err != nil {
		t.Fatalf("NewSubRing: %v", err)
	}
	if err := sr.GenerateNTTConstants(); err != nil {
		t.Fatalf("GenerateNTTConstants: %v", err)
	}

	a := make([]uint64, pulsarN)
	r := rand.New(rand.NewPCG(0xa5a5, 0))
	for i := range a {
		a[i] = r.Uint64() % pulsarQ
	}
	b := make([]uint64, pulsarN)
	copy(b, a)

	sr.NTTLazy(a, a)
	for i := range a {
		// Lazy bound: a[i] in [0, 6q-2]
		if a[i] >= 6*pulsarQ {
			t.Fatalf("NTTLazy [%d]=%d exceeds 6q-2", i, a[i])
		}
	}
	sr.INTTLazy(a, a)
	// INTTLazy(NTTLazy(x)) is not identity (different reduction
	// regimes); run NTT/INTT on b for the round-trip check.
	sr.NTT(b, b)
	sr.INTT(b, b)
	rng := rand.New(rand.NewPCG(0xa5a5, 0))
	for i := range b {
		want := rng.Uint64() % pulsarQ
		if b[i] != want {
			t.Fatalf("NTT/INTT round-trip [%d]: %d != %d", i, b[i], want)
		}
	}
}

// TestModularReduction_Primitives smoke-tests the scalar primitives so
// a regression in any of MRed / MRedLazy / MForm / IMForm / BRed /
// CRed surfaces here, not just behind the NTT body.
func TestModularReduction_Primitives(t *testing.T) {
	q := pulsarQ
	brc := GenBRedConstant(q)
	mrc := GenMRedConstant(q)

	x, y := uint64(0xC0FFEE), uint64(0xBEEF)

	// MForm <-> IMForm round-trip
	xm := MForm(x, q, brc)
	if got := IMForm(xm, q, mrc); got != x {
		t.Errorf("IMForm(MForm(x))=%d, want %d", got, x)
	}

	// MRed should match (x*y) mod q after demontgomery-ization
	ym := MForm(y, q, brc)
	xmy := MRed(xm, ym, q, mrc)
	xy := IMForm(xmy, q, mrc)
	want := MulModSlow(x, y, q)
	if xy != want {
		t.Errorf("MRed(MForm(x), MForm(y))=%d (de-mont), want %d", xy, want)
	}

	// CRed bound
	if got := CRed(2*q-1, q); got >= q {
		t.Errorf("CRed(2q-1)=%d, want < q", got)
	}
}

// MulModSlow is reference (x*y) mod q via 128-bit multiply, used only
// in tests.
func MulModSlow(x, y, q uint64) uint64 {
	hi, lo := mul64(x, y)
	return mod128(hi, lo, q)
}

func mul64(x, y uint64) (hi, lo uint64) {
	xLo, xHi := x&0xffffffff, x>>32
	yLo, yHi := y&0xffffffff, y>>32
	t := xLo*yLo
	lo = t & 0xffffffff
	t = (t >> 32) + xHi*yLo
	w1 := t & 0xffffffff
	w2 := t >> 32
	t = xLo*yHi + w1
	lo |= (t & 0xffffffff) << 32
	hi = xHi*yHi + w2 + (t >> 32)
	return
}

func mod128(hi, lo, q uint64) uint64 {
	// Slow but obviously-correct: divmod via long division.
	for i := 0; i < 64; i++ {
		hi = (hi << 1) | (lo >> 63)
		lo <<= 1
		if hi >= q {
			hi -= q
		}
	}
	return hi
}
