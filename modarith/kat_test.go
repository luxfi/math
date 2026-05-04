// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package modarith

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"
)

// TestKATBundle_RoundTrip — load the JSON KAT bundle emitted by
// cmd/emit_modarith_kat, replay every entry through the Go-side
// modarith primitives, and assert byte-equal output + SHA-256 match.
//
// Cross-runtime contract: the same JSON drives the C++ replay test in
// luxcpp/crypto/math/test/modarith_cross_runtime_test.cpp.
func TestKATBundle_RoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "modarith_kat.json")
	bundle, err := ReadKATBundleFile(path)
	if err != nil {
		t.Skipf("KAT bundle not present at %s; run cmd/emit_modarith_kat: %v",
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

		actual, ok := replayEntry(t, entry.Test, input)
		if !ok {
			continue
		}

		digest := sha256.Sum256(actual)
		if got := HexEncode(digest[:]); got != entry.OutputSHA256 {
			t.Errorf("entry[%d] %q: SHA-256 mismatch:\n want %s\n  got %s",
				i, entry.Test, entry.OutputSHA256, got)
		}
		if !bytes.Equal(actual, expected) {
			t.Errorf("entry[%d] %q: byte-stream mismatch", i, entry.Test)
		}
	}
}

// replayEntry routes an entry's test name to the right modarith call.
// Mirrors cmd/emit_modarith_kat/main.go.
func replayEntry(t *testing.T, name string, input []byte) ([]byte, bool) {
	t.Helper()
	switch {
	case strings.HasPrefix(name, "MontMulMod/"):
		if len(input) != 24 {
			t.Errorf("%s: want 24 bytes input, got %d", name, len(input))
			return nil, false
		}
		q := binary.LittleEndian.Uint64(input[0:8])
		a := binary.LittleEndian.Uint64(input[8:16])
		b := binary.LittleEndian.Uint64(input[16:24])
		mod, err := NewModulus(q, "kat")
		if err != nil {
			t.Errorf("%s: NewModulus: %v", name, err)
			return nil, false
		}
		aMont := ToMontgomery(a, mod)
		bMont := ToMontgomery(b, mod)
		productMont := MontMulMod(aMont, bMont, mod)
		productStandard := FromMontgomery(productMont, mod)
		out := make([]byte, 8)
		binary.LittleEndian.PutUint64(out, productStandard)
		return out, true
	case strings.HasPrefix(name, "AddMod/"):
		if len(input) != 24 {
			t.Errorf("%s: want 24 bytes input, got %d", name, len(input))
			return nil, false
		}
		q := binary.LittleEndian.Uint64(input[0:8])
		a := binary.LittleEndian.Uint64(input[8:16])
		b := binary.LittleEndian.Uint64(input[16:24])
		out := make([]byte, 8)
		binary.LittleEndian.PutUint64(out, AddMod(a, b, q))
		return out, true
	case strings.HasPrefix(name, "MontgomeryRoundTrip/"):
		if len(input) != 16 {
			t.Errorf("%s: want 16 bytes input, got %d", name, len(input))
			return nil, false
		}
		q := binary.LittleEndian.Uint64(input[0:8])
		x := binary.LittleEndian.Uint64(input[8:16])
		mod, err := NewModulus(q, "kat")
		if err != nil {
			t.Errorf("%s: NewModulus: %v", name, err)
			return nil, false
		}
		mont := ToMontgomery(x, mod)
		back := FromMontgomery(mont, mod)
		out := make([]byte, 8)
		binary.LittleEndian.PutUint64(out, back)
		return out, true
	}
	t.Errorf("unknown KAT test name: %s", name)
	return nil, false
}
