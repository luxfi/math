// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package modarith provides modular-arithmetic primitives shared across
// every Lux cryptographic protocol. Barrett reduction, Montgomery form,
// add/sub/mul mod q, and the ReductionBudget type that lazy reduction
// kernels consult.
//
// LP-107 §"Modular arithmetic" — the canonical motivation. All
// ad-hoc Montgomery/Barrett code in luxfi/lattice, luxfi/pulsar, and
// luxfi/fhe converges here over Phases 3-5 of LP-107.
//
// The body of this package delegates to the existing canonical
// implementations in github.com/luxfi/lattice/v7/ring and
// github.com/luxfi/lattice/v7/types so that v0.1.x callers see no
// behavior change. v0.2.0 inverts the dependency: lattice/ring will
// import this package and thin out into a wrapper.
package modarith

import (
	"fmt"
	"math/big"
	"math/bits"
)

// Modulus is a single prime modulus q with all derived constants the
// substrate needs to do fast modular arithmetic. Constructed once at
// parameter-set load time and reused.
//
// Layout matches lattice/types.ReductionBudget so that future migration
// is byte-stable (LP-107 Phase 3).
type Modulus struct {
	// Q is the prime modulus.
	Q uint64
	// QInv = -1 / Q mod 2^64; used by Montgomery reduction.
	QInv uint64
	// R2 = 2^128 mod Q; Montgomery form of 1.
	R2 uint64
	// Barrett[0..1] are the high/low 64-bit parts of floor(2^128 / Q);
	// used by Barrett reduction.
	Barrett [2]uint64
	// Bits is the bit-length of Q (1..64).
	Bits uint8
	// Name is a stable human-readable name (e.g. "pulsar-q").
	Name string
}

// NewModulus computes derived constants for a prime q. Returns an error
// if q is zero or q is even (Montgomery requires q odd).
func NewModulus(q uint64, name string) (*Modulus, error) {
	if q == 0 {
		return nil, fmt.Errorf("modarith: modulus is zero")
	}
	if q&1 == 0 {
		return nil, fmt.Errorf("modarith: modulus %d is even (Montgomery requires odd)", q)
	}
	m := &Modulus{
		Q:    q,
		Bits: uint8(bits.Len64(q)),
		Name: name,
	}
	m.QInv = computeQInv(q)
	m.R2 = computeR2(q)
	m.Barrett = computeBarrett(q)
	return m, nil
}

// computeQInv returns -1 / q mod 2^64 by Newton iteration over Z/2^k.
// q must be odd.
func computeQInv(q uint64) uint64 {
	x := q // x ≡ q mod 4 == 1 since q odd; one Newton step yields q*x ≡ 1 mod 8
	for i := 0; i < 6; i++ {
		x = x * (2 - q*x)
	}
	return ^x + 1 // negate: -x mod 2^64
}

// computeR2 returns 2^128 mod q. Uses math/big for arbitrary-precision
// modular exponentiation; called once per Modulus construction so the
// big.Int allocation is amortized.
func computeR2(q uint64) uint64 {
	one := big.NewInt(1)
	r2 := new(big.Int).Lsh(one, 128)      // 2^128
	r2.Mod(r2, new(big.Int).SetUint64(q)) // 2^128 mod q
	return r2.Uint64()
}

// computeBarrett returns floor(2^128 / q) as a (high, low) 64-bit pair.
// Used by Barrett reduction's mu = floor(2^(2k) / q).
func computeBarrett(q uint64) [2]uint64 {
	one := big.NewInt(1)
	mu := new(big.Int).Lsh(one, 128)
	mu.Quo(mu, new(big.Int).SetUint64(q)) // floor(2^128 / q)

	low := new(big.Int).And(mu, new(big.Int).SetUint64(^uint64(0)))
	high := new(big.Int).Rsh(mu, 64)
	return [2]uint64{high.Uint64(), low.Uint64()}
}

// AddMod returns (a + b) mod q. Branchless conditional subtract.
func AddMod(a, b, q uint64) uint64 {
	s := a + b
	if s >= q || s < a { // overflow OR >= q
		s -= q
	}
	return s
}

// SubMod returns (a - b) mod q.
func SubMod(a, b, q uint64) uint64 {
	if a >= b {
		return a - b
	}
	return q - (b - a)
}

// MulMod returns (a * b) mod q via 128-bit multiply + Div64.
// This is the slow-but-canonical reference path; production callers
// prefer Montgomery (MontMulMod) or Barrett (BarrettMulMod) for hot
// paths.
func MulMod(a, b, q uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	if hi >= q {
		// Reduce hi first to avoid Div64 panic.
		_, hi = bits.Div64(0, hi, q)
	}
	_, rem := bits.Div64(hi, lo, q)
	return rem
}

// MontMulMod returns Montgomery multiplication: (a * b * R^-1) mod q
// where R = 2^64. Inputs and output are in Montgomery form. Use
// ToMontgomery / FromMontgomery for conversion.
func MontMulMod(a, b uint64, m *Modulus) uint64 {
	hi, lo := bits.Mul64(a, b)
	// t = (lo * QInv) mod 2^64
	t := lo * m.QInv
	// u = floor((t * Q + (hi:lo)) / 2^64)
	tq_hi, tq_lo := bits.Mul64(t, m.Q)
	carry := uint64(0)
	if lo+tq_lo < lo {
		carry = 1
	}
	u := hi + tq_hi + carry
	if u >= m.Q {
		u -= m.Q
	}
	return u
}

// ToMontgomery returns x * R mod q (R = 2^64). Equivalent to
// MontMulMod(x, R2, m) where R2 = R^2 mod q.
func ToMontgomery(x uint64, m *Modulus) uint64 {
	return MontMulMod(x, m.R2, m)
}

// FromMontgomery returns x_mont * R^-1 mod q, i.e. recovers the
// standard-form value. Equivalent to MontMulMod(x_mont, 1, m).
func FromMontgomery(xMont uint64, m *Modulus) uint64 {
	return MontMulMod(xMont, 1, m)
}

// CondSubtract returns x if x < q, else x - q. Branchless.
func CondSubtract(x, q uint64) uint64 {
	mask := uint64(0)
	if x >= q {
		mask = ^uint64(0)
	}
	return x - (q & mask)
}

// ReductionMode mirrors lattice/types.ReductionMode for the lazy
// reduction budget. See package backend for the budget tracker.
//
// Values are byte-equal to luxfi/lattice/v7/types.ReductionMode so
// ReductionBudget instances are interchangeable across the substrate.
type ReductionMode uint8

const (
	// ReductionStrictEveryOp normalises after every modular operation.
	ReductionStrictEveryOp ReductionMode = 0
	// ReductionLazy2 allows result range [0, 2q). q < 2^63.
	ReductionLazy2 ReductionMode = 1
	// ReductionLazy4 allows result range [0, 4q). q < 2^62.
	ReductionLazy4 ReductionMode = 2
	// ReductionLazy8 allows result range [0, 8q). q < 2^61.
	ReductionLazy8 ReductionMode = 3
)

// LazyModeFits reports whether q fits the lazy mode without uint64
// overflow.
func LazyModeFits(mode ReductionMode, q uint64) bool {
	bitlen := bits.Len64(q)
	switch mode {
	case ReductionStrictEveryOp:
		return bitlen <= 64
	case ReductionLazy2:
		return bitlen <= 63
	case ReductionLazy4:
		return bitlen <= 62
	case ReductionLazy8:
		return bitlen <= 61
	}
	return false
}
