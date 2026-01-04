// Copyright (C) 2019-2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package math

import (
	"fmt"
	"math/bits"
	"strconv"
)

// HexOrDecimal64 marshals uint64 as hex or decimal.
type HexOrDecimal64 uint64

// UnmarshalJSON implements json.Unmarshaler.
func (i *HexOrDecimal64) UnmarshalJSON(input []byte) error {
	if len(input) > 1 && input[0] == '"' {
		input = input[1 : len(input)-1]
	}
	return i.UnmarshalText(input)
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (i *HexOrDecimal64) UnmarshalText(input []byte) error {
	n, ok := ParseUint64(string(input))
	if !ok {
		return fmt.Errorf("invalid hex or decimal integer %q", input)
	}
	*i = HexOrDecimal64(n)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (i HexOrDecimal64) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%#x", uint64(i)), nil
}

// ParseUint64 parses s as an integer in decimal or hexadecimal syntax.
func ParseUint64(s string) (uint64, bool) {
	if s == "" {
		return 0, true
	}
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		v, err := strconv.ParseUint(s[2:], 16, 64)
		return v, err == nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	return v, err == nil
}

// MustParseUint64 parses s as an integer and panics if invalid.
func MustParseUint64(s string) uint64 {
	v, ok := ParseUint64(s)
	if !ok {
		panic("invalid unsigned 64 bit integer: " + s)
	}
	return v
}

// SafeSub returns x-y and checks for overflow.
func SafeSub(x, y uint64) (uint64, bool) {
	diff, borrowOut := bits.Sub64(x, y, 0)
	return diff, borrowOut != 0
}

// SafeAdd returns x+y and checks for overflow.
func SafeAdd(x, y uint64) (uint64, bool) {
	sum, carryOut := bits.Add64(x, y, 0)
	return sum, carryOut != 0
}

// SafeMul returns x*y and checks for overflow.
func SafeMul(x, y uint64) (uint64, bool) {
	hi, lo := bits.Mul64(x, y)
	return lo, hi != 0
}
