// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// Helpers migrated from luxfi/lattice/v7/ring (utils.go, primes.go) and
// luxfi/lattice/v7/utils (BitReverse64). LP-107 Phase 3.

package canonical

import (
	"math/big"
	"math/bits"
)

// BitReverse64 returns the bit-reverse value of the input value, within a context of 2^bitLen.
func BitReverse64(index uint64, bitLen int) uint64 {
	return bits.Reverse64(index) >> (64 - bitLen)
}

// IsPrime applies the Baillie-PSW, which is 100% accurate for numbers below 2^64.
func IsPrime(x uint64) bool {
	return new(big.Int).SetUint64(x).ProbablyPrime(0)
}

// ModExp performs the modular exponentiation x^e mod p,
// x and p are required to be at most 64 bits to avoid an overflow.
func ModExp(x, e, p uint64) (result uint64) {
	brc := GenBRedConstant(p)
	result = 1
	for i := e; i > 0; i >>= 1 {
		if i&1 == 1 {
			result = BRed(result, x, p, brc)
		}
		x = BRed(x, x, p, brc)
	}
	return result
}

// ModExpPow2 performs the modular exponentiation x^e mod p, where p is a power of two,
// x and p are required to be at most 64 bits to avoid an overflow.
func ModExpPow2(x, e, p uint64) (result uint64) {
	result = 1
	for i := e; i > 0; i >>= 1 {
		if i&1 == 1 {
			result *= x
		}
		x *= x
	}
	return result & (p - 1)
}

// ModexpMontgomery performs the modular exponentiation x^e mod p,
// where x is in Montgomery form, and returns x^e in Montgomery form.
func ModexpMontgomery(x uint64, e int, q, mredconstant uint64, bredconstant [2]uint64) (result uint64) {
	result = MForm(1, q, bredconstant)

	for i := e; i > 0; i >>= 1 {
		if i&1 == 1 {
			result = MRed(result, x, q, mredconstant)
		}
		x = MRed(x, x, q, mredconstant)
	}
	return result
}
