// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package safe provides overflow-safe arithmetic operations.
package safe

import (
	"errors"
	"math"
	"math/big"
	"math/bits"

	"golang.org/x/exp/constraints"
)

var (
	ErrOverflow       = errors.New("overflow")
	ErrUnderflow      = errors.New("underflow")
	ErrDivisionByZero = errors.New("division by zero")

	// Common big.Int values for performance.
	bigZero      = big.NewInt(0)
	bigOne       = big.NewInt(1)
	maxUint64Big = new(big.Int).SetUint64(math.MaxUint64)
)

// Add64 returns a + b, or error if overflow.
func Add64(a, b uint64) (uint64, error) {
	if a > math.MaxUint64-b {
		return 0, ErrOverflow
	}
	return a + b, nil
}

// Sub returns a - b, or error if underflow.
func Sub[T constraints.Unsigned](a, b T) (T, error) {
	if a < b {
		return 0, ErrUnderflow
	}
	return a - b, nil
}

// Mul64 returns a * b, or error if overflow.
func Mul64(a, b uint64) (uint64, error) {
	if b != 0 && a > math.MaxUint64/b {
		return 0, ErrOverflow
	}
	return a * b, nil
}

// SafeAdd returns x+y and whether overflow occurred.
func SafeAdd(x, y uint64) (uint64, bool) {
	sum, carryOut := bits.Add64(x, y, 0)
	return sum, carryOut != 0
}

// SafeSub returns x-y and whether underflow occurred.
func SafeSub(x, y uint64) (uint64, bool) {
	diff, borrowOut := bits.Sub64(x, y, 0)
	return diff, borrowOut != 0
}

// SafeMul returns x*y and whether overflow occurred.
func SafeMul(x, y uint64) (uint64, bool) {
	hi, lo := bits.Mul64(x, y)
	return lo, hi != 0
}

// AbsDiff returns |a - b|.
func AbsDiff[T constraints.Unsigned](a, b T) T {
	return max(a, b) - min(a, b)
}

// Min returns the minimum of two uint64 values.
func Min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two uint64 values.
func Max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// Div64 returns a / b, or error if division by zero.
func Div64(a, b uint64) (uint64, error) {
	if b == 0 {
		return 0, ErrDivisionByZero
	}
	return a / b, nil
}

// MulBig returns a * b as a big.Int (no overflow possible).
func MulBig(a, b uint64) *big.Int {
	x := new(big.Int).SetUint64(a)
	y := new(big.Int).SetUint64(b)
	return x.Mul(x, y)
}

// MulDiv64 returns (a * b) / c without intermediate overflow.
// Returns error if c is zero or result overflows uint64.
func MulDiv64(a, b, c uint64) (uint64, error) {
	if c == 0 {
		return 0, ErrDivisionByZero
	}
	product := MulBig(a, b)
	divisor := new(big.Int).SetUint64(c)
	result := new(big.Int).Div(product, divisor)
	if result.Cmp(maxUint64Big) > 0 {
		return 0, ErrOverflow
	}
	return result.Uint64(), nil
}

// MulDivRoundUp64 returns ceil((a * b) / c) without intermediate overflow.
// Returns error if c is zero or result overflows uint64.
func MulDivRoundUp64(a, b, c uint64) (uint64, error) {
	if c == 0 {
		return 0, ErrDivisionByZero
	}
	product := MulBig(a, b)
	divisor := new(big.Int).SetUint64(c)
	// (product + divisor - 1) / divisor for ceiling division
	product.Add(product, divisor)
	product.Sub(product, bigOne)
	result := product.Div(product, divisor)
	if result.Cmp(maxUint64Big) > 0 {
		return 0, ErrOverflow
	}
	return result.Uint64(), nil
}

// BigMulDiv returns (a * b) / c for big.Int values.
// Returns nil if c is zero.
func BigMulDiv(a, b, c *big.Int) *big.Int {
	if c.Cmp(bigZero) == 0 {
		return nil
	}
	result := new(big.Int).Mul(a, b)
	return result.Div(result, c)
}

// BigMulDivRoundUp returns ceil((a * b) / c) for big.Int values.
// Returns nil if c is zero.
func BigMulDivRoundUp(a, b, c *big.Int) *big.Int {
	if c.Cmp(bigZero) == 0 {
		return nil
	}
	result := new(big.Int).Mul(a, b)
	// (result + c - 1) / c for ceiling division
	result.Add(result, c)
	result.Sub(result, bigOne)
	return result.Div(result, c)
}

// Clamp returns value clamped to [minVal, maxVal].
func Clamp[T constraints.Ordered](value, minVal, maxVal T) T {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}
