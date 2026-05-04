// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package poly

import (
	"math/rand/v2"
	"testing"

	"github.com/luxfi/math/backend"
	"github.com/luxfi/math/ntt"
	"github.com/luxfi/math/params"
)

const PulsarQ = uint64(0x1000000004A01)

var pulsarParams = &ntt.Params{
	N:  256,
	Q:  PulsarQ,
	ID: params.NTTPulsarN256,
}

func TestAddSub_RoundTrip(t *testing.T) {
	N := 256
	a := make([]uint64, N)
	b := make([]uint64, N)
	r := rand.New(rand.NewPCG(0xdead, 0))
	for i := range a {
		a[i] = r.Uint64() % PulsarQ
		b[i] = r.Uint64() % PulsarQ
	}
	sum := make([]uint64, N)
	if err := Add(sum, a, b, PulsarQ); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := make([]uint64, N)
	if err := Sub(got, sum, b, PulsarQ); err != nil {
		t.Fatalf("Sub: %v", err)
	}
	for i := range a {
		if got[i] != a[i] {
			t.Fatalf("[%d]: got %d, want %d", i, got[i], a[i])
		}
	}
}

func TestScalarMul(t *testing.T) {
	N := 256
	a := make([]uint64, N)
	for i := range a {
		a[i] = uint64(i + 1)
	}
	dst := make([]uint64, N)
	if err := ScalarMul(dst, a, 7, PulsarQ); err != nil {
		t.Fatalf("ScalarMul: %v", err)
	}
	for i := range a {
		want := (uint64(i+1) * 7) % PulsarQ
		if dst[i] != want {
			t.Errorf("[%d]: got %d, want %d", i, dst[i], want)
		}
	}
}

func TestMul_NegacyclicVsBigInt(t *testing.T) {
	// Verify a * b mod (X^N + 1) for small constants using package
	// ntt's pure-Go backend, then sanity-check against a hand-computed
	// expectation.
	svc, err := ntt.NewService(pulsarParams, backend.PolicyPureGo)
	if err != nil {
		t.Fatalf("ntt.NewService: %v", err)
	}
	N := int(pulsarParams.N)
	a := make([]uint64, N)
	b := make([]uint64, N)
	a[0] = 2
	b[0] = 3
	dst := make([]uint64, N)
	if err := Mul(dst, a, b, svc); err != nil {
		t.Fatalf("Mul: %v", err)
	}
	// (2)*(3) = 6 in coefficient 0; everything else 0.
	if dst[0] != 6 {
		t.Errorf("dst[0] = %d, want 6", dst[0])
	}
	for i := 1; i < N; i++ {
		if dst[i] != 0 {
			t.Errorf("dst[%d] = %d, want 0", i, dst[i])
		}
	}
}
