// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package poly provides polynomial-arithmetic primitives over R_q =
// Z_q[X] / (X^N + 1) used by every Lux lattice protocol.
//
// LP-107 §"Polynomial and RNS operations" — the canonical motivation.
// Add, sub, scalar-mul, NTT-domain mul; converted to /from NTT domain
// via package ntt.
//
// Phase 2 (this file): pure-Go reference implementation. Body uses
// luxfi/math/modarith for the field arithmetic and luxfi/math/ntt
// for the transform; no re-implementation, just composition.
package poly

import (
	"fmt"

	"github.com/luxfi/math/modarith"
	"github.com/luxfi/math/ntt"
)

// Add returns dst = a + b (mod q). All inputs must have length N.
func Add(dst, a, b []uint64, q uint64) error {
	if len(dst) != len(a) || len(a) != len(b) {
		return fmt.Errorf("poly.Add: length mismatch dst=%d a=%d b=%d",
			len(dst), len(a), len(b))
	}
	for i := range a {
		dst[i] = modarith.AddMod(a[i], b[i], q)
	}
	return nil
}

// Sub returns dst = a - b (mod q).
func Sub(dst, a, b []uint64, q uint64) error {
	if len(dst) != len(a) || len(a) != len(b) {
		return fmt.Errorf("poly.Sub: length mismatch dst=%d a=%d b=%d",
			len(dst), len(a), len(b))
	}
	for i := range a {
		dst[i] = modarith.SubMod(a[i], b[i], q)
	}
	return nil
}

// ScalarMul returns dst = a * scalar (mod q).
func ScalarMul(dst, a []uint64, scalar, q uint64) error {
	if len(dst) != len(a) {
		return fmt.Errorf("poly.ScalarMul: length mismatch dst=%d a=%d",
			len(dst), len(a))
	}
	for i := range a {
		dst[i] = modarith.MulMod(a[i], scalar, q)
	}
	return nil
}

// PointwiseMul returns dst = a * b (pointwise, NTT domain) (mod q).
// Inputs must already be in NTT domain; output is also in NTT domain.
// Use ntt.Service.Inverse to bring back to coefficient domain.
func PointwiseMul(dst, a, b []uint64, q uint64) error {
	if len(dst) != len(a) || len(a) != len(b) {
		return fmt.Errorf("poly.PointwiseMul: length mismatch dst=%d a=%d b=%d",
			len(dst), len(a), len(b))
	}
	for i := range a {
		dst[i] = modarith.MulMod(a[i], b[i], q)
	}
	return nil
}

// Mul computes the negacyclic polynomial product dst = a * b (mod q,
// mod X^N + 1) via NTT round-trip: NTT(a), NTT(b), pointwise mul,
// inverse NTT. dst, a, b must each have length p.N. Inputs are in
// coefficient domain; output is in coefficient domain.
//
// dst MAY alias a or b.
func Mul(dst, a, b []uint64, svc *ntt.Service) error {
	N := int(svc.Params().N)
	q := svc.Params().Q
	if len(dst) != N || len(a) != N || len(b) != N {
		return fmt.Errorf("poly.Mul: length mismatch dst=%d a=%d b=%d N=%d",
			len(dst), len(a), len(b), N)
	}
	aN := make([]uint64, N)
	bN := make([]uint64, N)
	copy(aN, a)
	copy(bN, b)
	if err := svc.Forward(aN, 1); err != nil {
		return fmt.Errorf("poly.Mul: NTT(a): %w", err)
	}
	if err := svc.Forward(bN, 1); err != nil {
		return fmt.Errorf("poly.Mul: NTT(b): %w", err)
	}
	if err := PointwiseMul(dst, aN, bN, q); err != nil {
		return err
	}
	if err := svc.Inverse(dst, 1); err != nil {
		return fmt.Errorf("poly.Mul: INTT(dst): %w", err)
	}
	return nil
}
