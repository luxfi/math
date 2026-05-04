// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ntt

import (
	"math/rand/v2"
	"testing"

	"github.com/luxfi/math/backend"
	"github.com/luxfi/math/params"
)

// PulsarN256 — Pulsar/LP-073 NTT instance.
var PulsarN256 = &Params{
	N:  256,
	Q:  0x1000000004A01,
	ID: params.NTTPulsarN256,
}

func TestParams_Validate(t *testing.T) {
	if err := PulsarN256.Validate(); err != nil {
		t.Errorf("Pulsar N=256: %v", err)
	}
	bad := &Params{N: 0, Q: 7, ID: params.NTTPulsarN256}
	if err := bad.Validate(); err == nil {
		t.Error("N=0: no error")
	}
	bad2 := &Params{N: 257, Q: 7, ID: params.NTTPulsarN256}
	if err := bad2.Validate(); err == nil {
		t.Error("N=257 (not pow2): no error")
	}
	notNTT := &Params{N: 256, Q: 13, ID: params.NTTPulsarN256}
	if err := notNTT.Validate(); err == nil {
		t.Error("not NTT-friendly: no error")
	}
}

func TestService_PureGo_RoundTrip(t *testing.T) {
	s, err := NewService(PulsarN256, backend.PolicyPureGo)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	if s.Backend() != params.BackendPureGo {
		t.Errorf("Backend = %s, want %s", s.Backend(), params.BackendPureGo)
	}

	r := rand.New(rand.NewPCG(0xdeadbeef, 0x12345678))
	N := int(PulsarN256.N)
	a := make([]uint64, N)
	for i := range a {
		a[i] = r.Uint64() % PulsarN256.Q
	}
	saved := make([]uint64, N)
	copy(saved, a)

	if err := s.Forward(a, 1); err != nil {
		t.Fatalf("Forward: %v", err)
	}
	if err := s.Inverse(a, 1); err != nil {
		t.Fatalf("Inverse: %v", err)
	}
	for i := range a {
		if a[i] != saved[i] {
			t.Fatalf("round-trip [%d]: %d != %d", i, a[i], saved[i])
		}
	}
}

func TestService_BatchRoundTrip(t *testing.T) {
	s, err := NewService(PulsarN256, backend.PolicyPureGo)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	N := int(PulsarN256.N)
	const batch = 8
	a := make([]uint64, batch*N)
	r := rand.New(rand.NewPCG(0xfeedface, 1))
	for i := range a {
		a[i] = r.Uint64() % PulsarN256.Q
	}
	saved := make([]uint64, batch*N)
	copy(saved, a)

	if err := s.Forward(a, batch); err != nil {
		t.Fatalf("Forward: %v", err)
	}
	if err := s.Inverse(a, batch); err != nil {
		t.Fatalf("Inverse: %v", err)
	}
	for i := range a {
		if a[i] != saved[i] {
			t.Fatalf("batch round-trip [%d]: %d != %d", i, a[i], saved[i])
		}
	}
}

func TestPureGo_Determinism_AcrossInvocations(t *testing.T) {
	// Same input -> identical output across two Service instances.
	a := make([]uint64, PulsarN256.N)
	r := rand.New(rand.NewPCG(0xa5a5a5a5, 0))
	for i := range a {
		a[i] = r.Uint64() % PulsarN256.Q
	}
	b := make([]uint64, len(a))
	copy(b, a)

	s1, err := NewService(PulsarN256, backend.PolicyPureGo)
	if err != nil {
		t.Fatalf("NewService 1: %v", err)
	}
	s2, err := NewService(PulsarN256, backend.PolicyPureGo)
	if err != nil {
		t.Fatalf("NewService 2: %v", err)
	}
	if err := s1.Forward(a, 1); err != nil {
		t.Fatalf("Forward 1: %v", err)
	}
	if err := s2.Forward(b, 1); err != nil {
		t.Fatalf("Forward 2: %v", err)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("non-deterministic [%d]: %d != %d", i, a[i], b[i])
		}
	}
}
