// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package rns provides the Residue Number System primitives that FHE
// schemes use to operate over a chain of small primes instead of one
// large modulus.
//
// LP-107 §"Polynomial and RNS operations" — the canonical motivation.
// FHE PN10QP27 / PN11QP54 ring chains are RNS towers; their basis-
// extension and modulus-switching primitives live here.
//
// Phase 2 (this file): defines the public surface for RNS Basis,
// Tower, and basis-extension. Concrete bodies delegate to
// github.com/luxfi/lattice/v7/ringqp; Phase 3 of LP-107 inverts that.
package rns

import (
	"fmt"

	"github.com/luxfi/math/modarith"
)

// Basis describes one RNS basis: a list of pairwise coprime primes
// q_0, q_1, ..., q_{k-1}. Numbers are represented as their CRT
// projections (x mod q_0, x mod q_1, ..., x mod q_{k-1}).
type Basis struct {
	// Moduli is the tuple of primes. Each entry MUST satisfy
	// gcd(q_i, q_j) = 1 for i != j; we don't re-check this at every
	// op (it's a parameter-set property).
	Moduli []*modarith.Modulus
	// Name is a stable identifier (e.g. "fhe-pn10qp27").
	Name string
}

// NewBasis constructs an RNS basis from a list of primes. Returns an
// error if any modulus fails modarith.NewModulus.
func NewBasis(primes []uint64, name string) (*Basis, error) {
	if len(primes) == 0 {
		return nil, fmt.Errorf("rns.NewBasis: empty prime list")
	}
	mods := make([]*modarith.Modulus, len(primes))
	for i, q := range primes {
		m, err := modarith.NewModulus(q, fmt.Sprintf("%s.q[%d]", name, i))
		if err != nil {
			return nil, fmt.Errorf("rns.NewBasis: prime[%d]=%d: %w", i, q, err)
		}
		mods[i] = m
	}
	return &Basis{Moduli: mods, Name: name}, nil
}

// Levels returns the number of primes in the basis.
func (b *Basis) Levels() int { return len(b.Moduli) }
