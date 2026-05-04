// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"path/filepath"
	"testing"
)

// TestKATBundle_RoundTrip — round-trip the same Go-side KAT to
// validate the bundle format itself.
func TestKATBundle_RoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "codec_kat.json")
	bundle, err := ReadKATBundleFile(path)
	if err != nil {
		t.Skipf("KAT bundle not present at %s; run cmd/emit_codec_kat: %v",
			path, err)
		return
	}
	if bundle.Schema != KATSchemaV1 {
		t.Fatalf("schema = %q, want %q", bundle.Schema, KATSchemaV1)
	}
	if len(bundle.Entries) == 0 {
		t.Fatal("bundle has zero entries")
	}
	for i, entry := range bundle.Entries {
		if err := entry.Header.Validate(); err != nil {
			t.Errorf("entry[%d] header: %v", i, err)
		}
		// Decode the input and replay it through Reader. Output must
		// match the recorded OutputHex byte-for-byte (or be "REJECTED"
		// sentinel for entries that are expected to reject).
		input, err := HexDecode(entry.InputHex)
		if err != nil {
			t.Errorf("entry[%d] input_hex: %v", i, err)
			continue
		}
		expected, err := HexDecode(entry.OutputHex)
		if err != nil {
			t.Errorf("entry[%d] output_hex: %v", i, err)
			continue
		}
		// Reset reader for each entry.
		r, err := NewReader(bytes.NewReader(input), DefaultLimitsLatticeWire)
		if err != nil {
			t.Errorf("entry[%d] NewReader: %v", i, err)
			continue
		}

		actual, rejected := replayEntry(t, r, entry.Test)
		if rejected {
			if string(expected) != "REJECTED" {
				t.Errorf("entry[%d] %q: expected non-rejection, got reject",
					i, entry.Test)
			}
			continue
		}

		// Verify SHA-256 matches.
		digest := sha256.Sum256(actual)
		got := HexEncode(digest[:])
		if got != entry.OutputSHA256 {
			t.Errorf("entry[%d] %q: SHA-256 mismatch:\n want %s\n  got %s",
				i, entry.Test, entry.OutputSHA256, got)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("entry[%d] %q: byte-stream mismatch", i, entry.Test)
		}
	}
}

// replayEntry routes the entry's test name to the right Reader call.
// The case list MUST mirror cmd/emit_codec_kat/main.go.
func replayEntry(t *testing.T, r *Reader, name string) ([]byte, bool) {
	t.Helper()
	switch {
	case startsWith(name, "ReadUint64Slice/"):
		out, err := r.ReadUint64Slice()
		if err != nil {
			if errors.Is(err, ErrLimitExceeded) {
				return nil, true // rejected
			}
			t.Errorf("%s: %v", name, err)
			return nil, false
		}
		return uint64sToBytes(out), false
	case startsWith(name, "ReadUint16Slice/"):
		out, err := r.ReadUint16Slice()
		if err != nil {
			if errors.Is(err, ErrLimitExceeded) {
				return nil, true
			}
			t.Errorf("%s: %v", name, err)
			return nil, false
		}
		return uint16sToBytes(out), false
	case startsWith(name, "ReadUint32Slice/"):
		out, err := r.ReadUint32Slice()
		if err != nil {
			if errors.Is(err, ErrLimitExceeded) {
				return nil, true
			}
			t.Errorf("%s: %v", name, err)
			return nil, false
		}
		return uint32sToBytes(out), false
	}
	t.Errorf("unknown KAT test name: %s", name)
	return nil, false
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func uint64sToBytes(s []uint64) []byte {
	b := make([]byte, len(s)*8)
	for i, v := range s {
		binary.LittleEndian.PutUint64(b[i*8:], v)
	}
	return b
}

func uint32sToBytes(s []uint32) []byte {
	b := make([]byte, len(s)*4)
	for i, v := range s {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	return b
}

func uint16sToBytes(s []uint16) []byte {
	b := make([]byte, len(s)*2)
	for i, v := range s {
		binary.LittleEndian.PutUint16(b[i*2:], v)
	}
	return b
}
