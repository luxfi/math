// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ntt

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luxfi/math/backend"
	"github.com/luxfi/math/params"
)

// TestKATBundle_RoundTrip — load the JSON KAT bundle emitted by
// cmd/emit_ntt_kat, replay every entry through the pure-Go NTT
// backend, and assert byte-equal output + SHA-256 match.
//
// Cross-runtime contract: the same JSON drives the C++ replay test in
// luxcpp/crypto/math/test/ntt_cross_runtime_test.cpp.
func TestKATBundle_RoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "ntt_kat.json")
	bundle, err := ReadKATBundleFile(path)
	if err != nil {
		t.Skipf("KAT bundle not present at %s; run cmd/emit_ntt_kat: %v",
			path, err)
		return
	}
	if bundle.Schema != KATSchemaV1 {
		t.Fatalf("schema = %q, want %q", bundle.Schema, KATSchemaV1)
	}
	if len(bundle.Entries) == 0 {
		t.Fatal("bundle has zero entries")
	}

	p := &Params{
		N:  256,
		Q:  0x1000000004A01,
		ID: params.NTTPulsarN256,
	}
	svc, err := NewService(p, backend.PolicyPureGo)
	if err != nil {
		t.Fatalf("NewService: %v", err)
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
		actual, ok := replayEntry(t, svc, entry.Test, input)
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

func replayEntry(t *testing.T, svc *Service, name string, input []byte) ([]byte, bool) {
	t.Helper()
	N := int(svc.Params().N)
	if len(input) != N*8 {
		t.Errorf("%s: want %d bytes, got %d", name, N*8, len(input))
		return nil, false
	}
	if !strings.HasPrefix(name, "Forward/") {
		t.Errorf("unknown KAT test name: %s", name)
		return nil, false
	}
	work := make([]uint64, N)
	for i := 0; i < N; i++ {
		work[i] = binary.LittleEndian.Uint64(input[i*8:])
	}
	if err := svc.Forward(work, 1); err != nil {
		t.Errorf("%s: Forward: %v", name, err)
		return nil, false
	}
	out := make([]byte, N*8)
	for i, v := range work {
		binary.LittleEndian.PutUint64(out[i*8:], v)
	}
	return out, true
}
