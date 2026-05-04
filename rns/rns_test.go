// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package rns

import "testing"

func TestNewBasis_Pulsar(t *testing.T) {
	// Pulsar canonical Q is single-prime; basis with one element is
	// the degenerate RNS case (no chain reduction).
	b, err := NewBasis([]uint64{0x1000000004A01}, "pulsar-q")
	if err != nil {
		t.Fatalf("NewBasis: %v", err)
	}
	if b.Levels() != 1 {
		t.Errorf("Levels = %d, want 1", b.Levels())
	}
}

func TestNewBasis_TwoPrime(t *testing.T) {
	// Synthetic two-prime tower.
	b, err := NewBasis([]uint64{0x1000000004A01, 0x1000000007EE1}, "two-prime")
	if err != nil {
		t.Fatalf("NewBasis: %v", err)
	}
	if b.Levels() != 2 {
		t.Errorf("Levels = %d, want 2", b.Levels())
	}
	if b.Moduli[0].Q != 0x1000000004A01 {
		t.Errorf("Moduli[0].Q = %#x", b.Moduli[0].Q)
	}
	if b.Moduli[1].Q != 0x1000000007EE1 {
		t.Errorf("Moduli[1].Q = %#x", b.Moduli[1].Q)
	}
}

func TestNewBasis_Empty(t *testing.T) {
	if _, err := NewBasis(nil, "empty"); err == nil {
		t.Error("NewBasis(nil): no error")
	}
}

func TestNewBasis_RejectsEvenPrime(t *testing.T) {
	if _, err := NewBasis([]uint64{8}, "even"); err == nil {
		t.Error("NewBasis(even): no error")
	}
}
