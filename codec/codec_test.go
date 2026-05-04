// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func newReader(t *testing.T, data []byte, l Limits) *Reader {
	t.Helper()
	r, err := NewReader(bytes.NewReader(data), l)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	return r
}

func TestLimits_Validate(t *testing.T) {
	if err := DefaultLimitsLatticeWire.Validate(); err != nil {
		t.Errorf("DefaultLimitsLatticeWire.Validate(): %v", err)
	}
	if err := (Limits{}).Validate(); err == nil {
		t.Error("empty Limits.Validate() returned nil")
	}
}

func TestNewReader_NilArgs(t *testing.T) {
	if _, err := NewReader(nil, DefaultLimitsLatticeWire); err == nil {
		t.Error("nil io.Reader: no error")
	}
	if _, err := NewReader(bytes.NewReader(nil), Limits{}); err == nil {
		t.Error("zero Limits: no error")
	}
}

func encodeUvarint(out *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		out.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	out.WriteByte(byte(v))
}

func TestReadUint64Slice_HappyPath(t *testing.T) {
	want := []uint64{0xdeadbeef, 0xcafebabe, 0x1122334455667788}
	var buf bytes.Buffer
	encodeUvarint(&buf, uint64(len(want)))
	for _, v := range want {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}

	r := newReader(t, buf.Bytes(), DefaultLimitsLatticeWire)
	got, err := r.ReadUint64Slice()
	if err != nil {
		t.Fatalf("ReadUint64Slice: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len: want %d got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: want %#x got %#x", i, want[i], got[i])
		}
	}
}

// TestReadUint64Slice_RejectsHugeLength is the regression test for
// lattigo issue #4: 9-byte input asking for 70 trillion uint64s must
// be rejected before any allocation.
func TestReadUint64Slice_RejectsHugeLength(t *testing.T) {
	// 9-byte attack input from the lattice issue #4 reproducer
	// produces a varint whose decoded value is way beyond MaxUint64SliceLen.
	// We synthesize: varint = 70_368_955_777_453 (~70T), then expect
	// a LimitError on MaxUint64SliceLen.
	huge := uint64(70_368_955_777_453)
	var buf bytes.Buffer
	encodeUvarint(&buf, huge)

	r := newReader(t, buf.Bytes(), DefaultLimitsLatticeWire)
	_, err := r.ReadUint64Slice()
	if err == nil {
		t.Fatal("ReadUint64Slice with 70T length: returned nil error")
	}
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("want ErrLimitExceeded, got %v", err)
	}
	var le *LimitError
	if !errors.As(err, &le) {
		t.Fatalf("error not a *LimitError: %T %v", err, err)
	}
	if le.What != "MaxUint64SliceLen" {
		t.Errorf("LimitError.What = %q, want MaxUint64SliceLen", le.What)
	}
	if le.Got != huge {
		t.Errorf("LimitError.Got = %d, want %d", le.Got, huge)
	}
}

func TestReadUint32Slice_RejectsHugeLength(t *testing.T) {
	huge := uint64(1_000_000_000)
	var buf bytes.Buffer
	encodeUvarint(&buf, huge)

	r := newReader(t, buf.Bytes(), DefaultLimitsLatticeWire)
	_, err := r.ReadUint32Slice()
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("want ErrLimitExceeded, got %v", err)
	}
}

func TestReadUint16Slice_RejectsHugeLength(t *testing.T) {
	huge := uint64(1_000_000_000)
	var buf bytes.Buffer
	encodeUvarint(&buf, huge)

	r := newReader(t, buf.Bytes(), DefaultLimitsLatticeWire)
	_, err := r.ReadUint16Slice()
	if !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("want ErrLimitExceeded, got %v", err)
	}
}

func TestReadUint16(t *testing.T) {
	r := newReader(t, []byte{0xCD, 0xAB}, DefaultLimitsLatticeWire)
	v, err := r.ReadUint16()
	if err != nil || v != 0xABCD {
		t.Errorf("ReadUint16: %v %#x", err, v)
	}
}

func TestReadUint32(t *testing.T) {
	r := newReader(t, []byte{0x78, 0x56, 0x34, 0x12}, DefaultLimitsLatticeWire)
	v, err := r.ReadUint32()
	if err != nil || v != 0x12345678 {
		t.Errorf("ReadUint32: %v %#x", err, v)
	}
}

func TestReadUint64(t *testing.T) {
	r := newReader(t, []byte{
		0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11,
	}, DefaultLimitsLatticeWire)
	v, err := r.ReadUint64()
	if err != nil || v != 0x1122334455667788 {
		t.Errorf("ReadUint64: %v %#x", err, v)
	}
}

func TestDepth(t *testing.T) {
	limits := DefaultLimitsLatticeWire
	limits.MaxDepth = 2

	r := newReader(t, []byte{}, limits)
	if err := r.EnterDepth(); err != nil {
		t.Fatalf("depth 1: %v", err)
	}
	if err := r.EnterDepth(); err != nil {
		t.Fatalf("depth 2: %v", err)
	}
	if err := r.EnterDepth(); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("depth 3: want ErrLimitExceeded, got %v", err)
	}
	r.ExitDepth()
	r.ExitDepth()
	r.ExitDepth() // safe to over-exit
}

func TestFrameBytesCap(t *testing.T) {
	limits := DefaultLimitsLatticeWire
	limits.MaxFrameBytes = 4

	r := newReader(t, []byte{1, 2, 3, 4, 5}, limits)
	if _, err := r.ReadUint32(); err != nil {
		t.Fatalf("first 4 bytes: %v", err)
	}
	// Next byte read must hit MaxFrameBytes.
	if _, err := r.ReadUint16(); !errors.Is(err, ErrLimitExceeded) {
		t.Errorf("over-cap: want ErrLimitExceeded, got %v", err)
	}
}
