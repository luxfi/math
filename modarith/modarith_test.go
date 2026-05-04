// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package modarith

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

// PulsarQ — Pulsar/LP-073 canonical NTT-friendly prime.
// Q = 0x1000000004A01.
const PulsarQ = uint64(0x1000000004A01)

func TestNewModulus_RejectsZero(t *testing.T) {
	if _, err := NewModulus(0, "zero"); err == nil {
		t.Error("NewModulus(0): no error")
	}
}

func TestNewModulus_RejectsEven(t *testing.T) {
	if _, err := NewModulus(8, "even"); err == nil {
		t.Error("NewModulus(8): no error")
	}
}

func TestNewModulus_PulsarQ(t *testing.T) {
	m, err := NewModulus(PulsarQ, "pulsar-q")
	if err != nil {
		t.Fatalf("NewModulus: %v", err)
	}
	if m.Q != PulsarQ {
		t.Errorf("Q: %#x", m.Q)
	}
	if m.Bits != 49 {
		t.Errorf("Bits: %d, want 49", m.Bits)
	}
	// q * QInv ≡ -1 mod 2^64
	want := ^uint64(0) // -1 mod 2^64
	if got := PulsarQ * m.QInv; got != want {
		t.Errorf("q*QInv = %#x, want %#x", got, want)
	}
}

func TestAddMod(t *testing.T) {
	q := PulsarQ
	tests := []struct {
		a, b, want uint64
	}{
		{0, 0, 0},
		{q - 1, 1, 0},
		{q - 2, 1, q - 1},
		{1, 1, 2},
		{q - 1, q - 1, q - 2},
	}
	for _, tc := range tests {
		if got := AddMod(tc.a, tc.b, q); got != tc.want {
			t.Errorf("AddMod(%d, %d, %d) = %d, want %d",
				tc.a, tc.b, q, got, tc.want)
		}
	}
}

func TestSubMod(t *testing.T) {
	q := PulsarQ
	tests := []struct {
		a, b, want uint64
	}{
		{0, 0, 0},
		{1, 1, 0},
		{0, 1, q - 1},
		{5, 3, 2},
		{3, 5, q - 2},
	}
	for _, tc := range tests {
		if got := SubMod(tc.a, tc.b, q); got != tc.want {
			t.Errorf("SubMod(%d, %d, %d) = %d, want %d",
				tc.a, tc.b, q, got, tc.want)
		}
	}
}

func TestMulMod_VsBigInt(t *testing.T) {
	q := PulsarQ
	qBig := new(big.Int).SetUint64(q)
	r := rand.New(rand.NewPCG(0xdeadbeef, 0x12345678))

	for i := 0; i < 1000; i++ {
		a := r.Uint64() % q
		b := r.Uint64() % q
		got := MulMod(a, b, q)

		want := new(big.Int).Mul(
			new(big.Int).SetUint64(a),
			new(big.Int).SetUint64(b))
		want.Mod(want, qBig)

		if got != want.Uint64() {
			t.Fatalf("MulMod(%d, %d) = %d, want %d", a, b, got, want.Uint64())
		}
	}
}

func TestMontgomery_RoundTrip(t *testing.T) {
	m, err := NewModulus(PulsarQ, "pulsar-q")
	if err != nil {
		t.Fatalf("NewModulus: %v", err)
	}
	r := rand.New(rand.NewPCG(0xfeedface, 0xc0ffeebabe))

	for i := 0; i < 100; i++ {
		x := r.Uint64() % PulsarQ
		mont := ToMontgomery(x, m)
		back := FromMontgomery(mont, m)
		if back != x {
			t.Fatalf("round-trip [%d]: %d -> mont=%d -> %d", i, x, mont, back)
		}
	}
}

func TestMontMulMod_VsMulMod(t *testing.T) {
	m, err := NewModulus(PulsarQ, "pulsar-q")
	if err != nil {
		t.Fatalf("NewModulus: %v", err)
	}
	r := rand.New(rand.NewPCG(0xa5a5a5a5, 0x5a5a5a5a))

	for i := 0; i < 100; i++ {
		a := r.Uint64() % PulsarQ
		b := r.Uint64() % PulsarQ

		// Mont(a) * Mont(b) * R^-1 == Mont(a*b)
		aMont := ToMontgomery(a, m)
		bMont := ToMontgomery(b, m)
		productMont := MontMulMod(aMont, bMont, m)
		productStandard := FromMontgomery(productMont, m)

		want := MulMod(a, b, PulsarQ)
		if productStandard != want {
			t.Fatalf("[%d] a=%d b=%d: mont path got %d, MulMod got %d",
				i, a, b, productStandard, want)
		}
	}
}

func TestCondSubtract(t *testing.T) {
	q := PulsarQ
	tests := []struct {
		x, want uint64
	}{
		{0, 0},
		{q - 1, q - 1},
		{q, 0},
		{q + 1, 1},
		{2*q - 1, q - 1},
	}
	for _, tc := range tests {
		if got := CondSubtract(tc.x, q); got != tc.want {
			t.Errorf("CondSubtract(%d, %d) = %d, want %d",
				tc.x, q, got, tc.want)
		}
	}
}

func TestLazyModeFits(t *testing.T) {
	tests := []struct {
		mode ReductionMode
		q    uint64
		want bool
	}{
		{ReductionStrictEveryOp, 1 << 63, true},
		{ReductionStrictEveryOp, ^uint64(0), true},
		{ReductionLazy2, 1<<63 - 1, true},
		{ReductionLazy2, 1 << 63, false},
		{ReductionLazy4, 1<<62 - 1, true},
		{ReductionLazy4, 1 << 62, false},
		{ReductionLazy8, 1<<61 - 1, true},
		{ReductionLazy8, 1 << 61, false},
	}
	for _, tc := range tests {
		if got := LazyModeFits(tc.mode, tc.q); got != tc.want {
			t.Errorf("LazyModeFits(%d, %d) = %v, want %v",
				tc.mode, tc.q, got, tc.want)
		}
	}
}
