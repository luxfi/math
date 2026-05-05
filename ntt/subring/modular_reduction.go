// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// Body migrated from github.com/luxfi/lattice/v7/ring/modular_reduction.go
// (Lattigo-derived). LP-107 Phase 3 establishes luxfi/math as the
// owner of the Montgomery / Barrett scalar primitives; the
// lattice copy now delegates to this package.

package subring

import (
	"math/big"
	"math/bits"
)

// MForm switches a to the Montgomery domain by computing
// a*2^64 mod q.
func MForm(a, q uint64, bredconstant [2]uint64) (r uint64) {
	mhi, _ := bits.Mul64(a, bredconstant[1])
	r = -(a*bredconstant[0] + mhi) * q
	if r >= q {
		r -= q
	}
	return
}

// MFormLazy switches a to the Montgomery domain by computing
// a*2^64 mod q in constant time.
// The result is between 0 and 2*q-1.
func MFormLazy(a, q uint64, bredconstant [2]uint64) (r uint64) {
	mhi, _ := bits.Mul64(a, bredconstant[1])
	r = -(a*bredconstant[0] + mhi) * q
	return
}

// IMForm switches a from the Montgomery domain back to the
// standard domain by computing a*(1/2^64) mod q.
func IMForm(a, q, mredconstant uint64) (r uint64) {
	r, _ = bits.Mul64(a*mredconstant, q)
	r = q - r
	if r >= q {
		r -= q
	}
	return
}

// IMFormLazy switches a from the Montgomery domain back to the
// standard domain by computing a*(1/2^64) mod q in constant time.
// The result is between 0 and 2*q-1.
func IMFormLazy(a, q, mredconstant uint64) (r uint64) {
	r, _ = bits.Mul64(a*mredconstant, q)
	r = q - r
	return
}

// GenMRedConstant computes the constant mredconstant = (q^-1) mod 2^64 required for MRed.
func GenMRedConstant(q uint64) (mredconstant uint64) {
	mredconstant = 1
	for i := 0; i < 63; i++ {
		mredconstant *= q
		q *= q
	}
	return mredconstant
}

// MRed computes x * y * (1/2^64) mod q.
func MRed(x, y, q, mredconstant uint64) (r uint64) {
	mhi, mlo := bits.Mul64(x, y)
	hhi, _ := bits.Mul64(mlo*mredconstant, q)
	r = mhi - hhi + q
	if r >= q {
		r -= q
	}
	return
}

// MRedLazy computes x * y * (1/2^64) mod q in constant time.
// The result is between 0 and 2*q-1.
func MRedLazy(x, y, q, mredconstant uint64) (r uint64) {
	ahi, alo := bits.Mul64(x, y)
	H, _ := bits.Mul64(alo*mredconstant, q)
	r = ahi - H + q
	return
}

// GenBRedConstant computes the constant for the BRed algorithm.
// Returns ((2^128)/q)/(2^64) and (2^128)/q mod 2^64.
func GenBRedConstant(q uint64) [2]uint64 {
	bigR, _ := new(big.Int).SetString("100000000000000000000000000000000", 16)
	bigR.Quo(bigR, new(big.Int).SetUint64(q))

	mlo := bigR.Uint64()
	mhi := bigR.Rsh(bigR, 64).Uint64()

	return [2]uint64{mhi, mlo}
}

// BRedAdd computes a mod q.
func BRedAdd(a, q uint64, bredconstant [2]uint64) (r uint64) {
	mhi, _ := bits.Mul64(a, bredconstant[0])
	r = a - mhi*q
	if r >= q {
		r -= q
	}
	return
}

// BRedAddLazy computes a mod q in constant time.
// The result is between 0 and 2*q-1.
func BRedAddLazy(x, q uint64, bredconstant [2]uint64) uint64 {
	s0, _ := bits.Mul64(x, bredconstant[0])
	return x - s0*q
}

// BRed computes x*y mod q.
func BRed(x, y, q uint64, bredconstant [2]uint64) (r uint64) {
	var mhi, mlo, lhi, hhi, hlo, s0, carry uint64

	mhi, mlo = bits.Mul64(x, y)

	r = mhi * bredconstant[0]

	hhi, hlo = bits.Mul64(mlo, bredconstant[0])

	r += hhi

	lhi, _ = bits.Mul64(mlo, bredconstant[1])

	s0, carry = bits.Add64(hlo, lhi, 0)

	r += carry

	hhi, hlo = bits.Mul64(mhi, bredconstant[1])

	r += hhi

	_, carry = bits.Add64(hlo, s0, 0)

	r += carry

	r = mlo - r*q

	if r >= q {
		r -= q
	}

	return
}

// BRedLazy computes x*y mod q in constant time.
// The result is between 0 and 2*q-1.
func BRedLazy(x, y, q uint64, bredconstant [2]uint64) (r uint64) {
	var mhi, mlo, lhi, hhi, hlo, s0, carry uint64

	mhi, mlo = bits.Mul64(x, y)

	r = mhi * bredconstant[0]

	hhi, hlo = bits.Mul64(mlo, bredconstant[0])

	r += hhi

	lhi, _ = bits.Mul64(mlo, bredconstant[1])

	s0, carry = bits.Add64(hlo, lhi, 0)

	r += carry

	hhi, hlo = bits.Mul64(mhi, bredconstant[1])

	r += hhi

	_, carry = bits.Add64(hlo, s0, 0)

	r += carry

	r = mlo - r*q

	return
}

// CRed reduce returns a mod q where a is between 0 and 2*q-1.
func CRed(a, q uint64) uint64 {
	if a >= q {
		return a - q
	}
	return a
}
