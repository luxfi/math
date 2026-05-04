// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package codec is the bounded-decode contract for every wire format
// in luxfi/math (and downstream luxfi/lattice, luxfi/pulsar, luxfi/fhe).
//
// LP-107 §"Codec and bounded reader should be centralized" — the
// canonical motivation. The lattigo `ReadUint64Slice` recursion bug
// (issue #2) and `Vector[T].ReadFrom` OOM bug (issue #4) both stemmed
// from unbounded slice decode on untrusted wire data; this package
// fixes that class permanently.
//
// Contract:
//
//   - No recursion. Slice readers are iterative; depth is bounded by
//     the configured Limits.
//   - No hidden growth. Every `make([]T, n)` is preceded by a `n <= cap`
//     check against caller-supplied Limits.
//   - No unbounded allocation. The largest cap is application-supplied
//     and surfaces in error messages.
//   - All readers are deterministic and reentrant; failure leaves the
//     reader at the byte where the bound was exceeded.
package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
)

// Limits caps the largest slice / depth a Reader will accept on a
// single decode call. Callers MUST construct Limits explicitly; there
// is no implicit default.
type Limits struct {
	// MaxFrameBytes caps the total number of input bytes the Reader
	// will consume in a single decode call. Used to bound peek-ahead
	// buffers. 0 means unset and is treated as an error.
	MaxFrameBytes int

	// MaxUint16SliceLen caps the number of elements in a uint16 slice.
	MaxUint16SliceLen int
	// MaxUint32SliceLen caps the number of elements in a uint32 slice.
	MaxUint32SliceLen int
	// MaxUint64SliceLen caps the number of elements in a uint64 slice.
	MaxUint64SliceLen int

	// MaxDepth caps how deeply a recursively-shaped wire format may
	// nest before the reader rejects (Vector[Poly] etc. are 2 levels).
	MaxDepth int
}

// Validate reports whether the limits are coherent (all positive).
// Returns an error listing every zero/negative field.
func (l Limits) Validate() error {
	var problems []string
	if l.MaxFrameBytes <= 0 {
		problems = append(problems, "MaxFrameBytes")
	}
	if l.MaxUint16SliceLen <= 0 {
		problems = append(problems, "MaxUint16SliceLen")
	}
	if l.MaxUint32SliceLen <= 0 {
		problems = append(problems, "MaxUint32SliceLen")
	}
	if l.MaxUint64SliceLen <= 0 {
		problems = append(problems, "MaxUint64SliceLen")
	}
	if l.MaxDepth <= 0 {
		problems = append(problems, "MaxDepth")
	}
	if len(problems) > 0 {
		return fmt.Errorf("codec.Limits: zero/negative fields: %v", problems)
	}
	return nil
}

// DefaultLimitsLatticeWire is the conservative default for wire-format
// decoding of lattice polynomials at the canonical Pulsar parameters
// (R_q = Z_q[X]/(X^256 + 1), Q ≈ 2^48). Callers SHOULD use a
// configuration tuned to their parameter set rather than this default.
//
//	MaxUint64SliceLen = 4096   matches Pulsar Vector[Poly] cap.
//	MaxFrameBytes = 16 MiB     allows a worst-case threshold ceremony
//	                           transcript without truncation.
//	MaxDepth = 4               Pulsar wire is 2 levels (Vector + Poly);
//	                           4 leaves headroom for FHE chains.
var DefaultLimitsLatticeWire = Limits{
	MaxFrameBytes:     16 * 1024 * 1024,
	MaxUint16SliceLen: 4096,
	MaxUint32SliceLen: 4096,
	MaxUint64SliceLen: 4096,
	MaxDepth:          4,
}

// ErrLimitExceeded is the sentinel for any limit-bound rejection.
// errors.Is(err, ErrLimitExceeded) holds for every cap violation.
var ErrLimitExceeded = errors.New("codec: limit exceeded")

// LimitError carries the specific limit that was exceeded plus the
// observed value. Wraps ErrLimitExceeded.
type LimitError struct {
	What  string // human-readable name of the cap, e.g. "MaxUint64SliceLen"
	Limit int
	Got   uint64
}

// Error implements error.
func (e *LimitError) Error() string {
	return fmt.Sprintf("codec: %s exceeded: limit=%d got=%d",
		e.What, e.Limit, e.Got)
}

// Unwrap implements errors.Unwrap.
func (e *LimitError) Unwrap() error { return ErrLimitExceeded }

// Reader wraps an io.Reader and a Limits config. Every slice-reading
// method on Reader is bounded by Limits.
type Reader struct {
	r        io.Reader
	limits   Limits
	consumed int
	depth    int
}

// NewReader constructs a Reader from an io.Reader and a Limits config.
// Returns an error if Limits is invalid.
func NewReader(r io.Reader, l Limits) (*Reader, error) {
	if r == nil {
		return nil, fmt.Errorf("codec: nil io.Reader")
	}
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return &Reader{r: r, limits: l}, nil
}

// Consumed returns the number of bytes read from the underlying io.Reader.
func (r *Reader) Consumed() int { return r.consumed }

// EnterDepth bumps the nesting counter and returns an error if the
// configured MaxDepth is exceeded. Caller MUST pair with ExitDepth.
func (r *Reader) EnterDepth() error {
	r.depth++
	if r.depth > r.limits.MaxDepth {
		return &LimitError{What: "MaxDepth", Limit: r.limits.MaxDepth, Got: uint64(r.depth)}
	}
	return nil
}

// ExitDepth decrements the nesting counter.
func (r *Reader) ExitDepth() {
	if r.depth > 0 {
		r.depth--
	}
}

// readN reads exactly n bytes, bumping the consumed counter and
// validating against MaxFrameBytes.
func (r *Reader) readN(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("codec: negative read length %d", n)
	}
	if r.consumed+n > r.limits.MaxFrameBytes {
		return nil, &LimitError{
			What:  "MaxFrameBytes",
			Limit: r.limits.MaxFrameBytes,
			Got:   uint64(r.consumed + n),
		}
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r.r, buf); err != nil {
		return nil, fmt.Errorf("codec: short read: %w", err)
	}
	r.consumed += n
	return buf, nil
}

// ReadUint16 reads a single little-endian uint16.
func (r *Reader) ReadUint16() (uint16, error) {
	b, err := r.readN(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

// ReadUint32 reads a single little-endian uint32.
func (r *Reader) ReadUint32() (uint32, error) {
	b, err := r.readN(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

// ReadUint64 reads a single little-endian uint64.
func (r *Reader) ReadUint64() (uint64, error) {
	b, err := r.readN(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

// ReadUint16Slice reads a length-prefixed slice of little-endian uint16.
// The length is read as a varint capped by MaxUint16SliceLen; iterative
// (no recursion).
func (r *Reader) ReadUint16Slice() ([]uint16, error) {
	n, err := r.readSliceLen("uint16", r.limits.MaxUint16SliceLen)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []uint16{}, nil
	}
	if err := overflowMul(n, 2, r.limits.MaxFrameBytes); err != nil {
		return nil, err
	}
	out := make([]uint16, n)
	buf, err := r.readN(int(n) * 2)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(buf[i*2:])
	}
	return out, nil
}

// ReadUint32Slice reads a length-prefixed slice of little-endian uint32.
func (r *Reader) ReadUint32Slice() ([]uint32, error) {
	n, err := r.readSliceLen("uint32", r.limits.MaxUint32SliceLen)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []uint32{}, nil
	}
	if err := overflowMul(n, 4, r.limits.MaxFrameBytes); err != nil {
		return nil, err
	}
	out := make([]uint32, n)
	buf, err := r.readN(int(n) * 4)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(buf[i*4:])
	}
	return out, nil
}

// ReadUint64Slice reads a length-prefixed slice of little-endian uint64.
// Bounded by MaxUint64SliceLen. The length-prefix is always read as a
// varint; values > MaxUint64SliceLen are rejected before any allocation.
//
// This is the centralized fix for both lattigo issue #2 (recursive
// `ReadUint64Slice`) and issue #4 (`Vector[T].ReadFrom` unbounded
// allocation). Callers consuming untrusted lattice wire data MUST go
// through this method, never lattigo's raw `utils/buffer.ReadUint64Slice`.
func (r *Reader) ReadUint64Slice() ([]uint64, error) {
	n, err := r.readSliceLen("uint64", r.limits.MaxUint64SliceLen)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return []uint64{}, nil
	}
	if err := overflowMul(n, 8, r.limits.MaxFrameBytes); err != nil {
		return nil, err
	}
	out := make([]uint64, n)
	buf, err := r.readN(int(n) * 8)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i] = binary.LittleEndian.Uint64(buf[i*8:])
	}
	return out, nil
}

// readSliceLen reads a varint length and validates against the cap.
// Returns the parsed length as uint32 (we never accept lengths that
// don't fit a 32-bit count for slice work).
func (r *Reader) readSliceLen(what string, cap int) (uint32, error) {
	v, err := r.readUvarint()
	if err != nil {
		return 0, err
	}
	if v > uint64(cap) {
		return 0, &LimitError{
			What:  fmt.Sprintf("Max%sSliceLen", capitalize(what)),
			Limit: cap,
			Got:   v,
		}
	}
	if v > uint64(^uint32(0)) {
		return 0, &LimitError{
			What:  "uint32 representable",
			Limit: int(^uint32(0)),
			Got:   v,
		}
	}
	return uint32(v), nil
}

// readUvarint reads a varint length. Iterative; bounded to 10 bytes
// (max varint encoding of uint64).
func (r *Reader) readUvarint() (uint64, error) {
	var v uint64
	var shift uint
	for i := 0; i < 10; i++ {
		b, err := r.readN(1)
		if err != nil {
			return 0, err
		}
		c := b[0]
		if c < 0x80 {
			if i == 9 && c > 1 {
				return 0, fmt.Errorf("codec: varint overflow")
			}
			v |= uint64(c) << shift
			return v, nil
		}
		v |= uint64(c&0x7f) << shift
		shift += 7
	}
	return 0, fmt.Errorf("codec: varint too long (>10 bytes)")
}

// overflowMul rejects n*size > frameMax even on multiplication overflow.
// Returns *LimitError on rejection.
func overflowMul(n uint32, size int, frameMax int) error {
	hi, lo := bits.Mul64(uint64(n), uint64(size))
	if hi != 0 || lo > uint64(frameMax) {
		return &LimitError{
			What:  "MaxFrameBytes (slice payload)",
			Limit: frameMax,
			Got:   lo,
		}
	}
	return nil
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	first := s[0]
	if first >= 'a' && first <= 'z' {
		first -= 'a' - 'A'
	}
	return string(first) + s[1:]
}
