// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package sample

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

const (
	katNumSamples   = 64
	katPulsarQ      = uint64(0x1000000004A01)
	katNTT998       = uint64(998244353)
)

// TestKATBundle_RoundTrip — load the JSON KAT bundle emitted by
// cmd/emit_sample_kat, replay every entry through the Go-side
// samplers driven by a byte-buffer reader, and assert byte-equal
// sample output.
//
// Cross-runtime contract: the same JSON drives the C++ replay test in
// luxcpp/crypto/math/test/sample_cross_runtime_test.cpp.
func TestKATBundle_RoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "sample_kat.json")
	bundle, err := ReadKATBundleFile(path)
	if err != nil {
		t.Skipf("KAT bundle not present at %s; run cmd/emit_sample_kat: %v",
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

// replayEntry parses the test name and dispatches to the right sampler.
// Test name format mirrors cmd/emit_sample_kat:
//
//	Uniform/q=PulsarQ/i=N
//	Uniform/q=NTT998/i=N
//	Ternary/q=PulsarQ/density=0.50/i=N
//	CenteredBinomial/q=PulsarQ/eta=2/i=N
func replayEntry(t *testing.T, name string, input []byte) ([]byte, bool) {
	t.Helper()
	r := newBBReader(input)
	dst := make([]uint64, katNumSamples)

	q := parseQ(name)
	if q == 0 {
		t.Errorf("%s: cannot parse q", name)
		return nil, false
	}

	switch {
	case strings.HasPrefix(name, "Uniform/"):
		if err := Uniform(dst, q, r); err != nil {
			t.Errorf("%s: Uniform: %v", name, err)
			return nil, false
		}
	case strings.HasPrefix(name, "Ternary/"):
		density, ok := parseDensity(name)
		if !ok {
			t.Errorf("%s: cannot parse density", name)
			return nil, false
		}
		if err := Ternary(dst, q, density, r); err != nil {
			t.Errorf("%s: Ternary: %v", name, err)
			return nil, false
		}
	case strings.HasPrefix(name, "CenteredBinomial/"):
		eta, ok := parseEta(name)
		if !ok {
			t.Errorf("%s: cannot parse eta", name)
			return nil, false
		}
		if err := CenteredBinomial(dst, q, eta, r); err != nil {
			t.Errorf("%s: CenteredBinomial: %v", name, err)
			return nil, false
		}
	default:
		t.Errorf("unknown KAT test name: %s", name)
		return nil, false
	}
	out := make([]byte, len(dst)*8)
	for i, v := range dst {
		binary.LittleEndian.PutUint64(out[i*8:], v)
	}
	return out, true
}

func parseQ(name string) uint64 {
	if strings.Contains(name, "q=PulsarQ") {
		return katPulsarQ
	}
	if strings.Contains(name, "q=NTT998") {
		return katNTT998
	}
	return 0
}

func parseDensity(name string) (float64, bool) {
	idx := strings.Index(name, "density=")
	if idx < 0 {
		return 0, false
	}
	rest := name[idx+len("density="):]
	end := strings.IndexByte(rest, '/')
	if end < 0 {
		return 0, false
	}
	var d float64
	if _, err := fmt.Sscanf(rest[:end], "%f", &d); err != nil {
		return 0, false
	}
	return d, true
}

func parseEta(name string) (int, bool) {
	idx := strings.Index(name, "eta=")
	if idx < 0 {
		return 0, false
	}
	rest := name[idx+len("eta="):]
	end := strings.IndexByte(rest, '/')
	if end < 0 {
		return 0, false
	}
	var e int
	if _, err := fmt.Sscanf(rest[:end], "%d", &e); err != nil {
		return 0, false
	}
	return e, true
}

// bbReader is a stateful byte-buffer io.Reader.
type bbReader struct {
	buf []byte
	off int
}

func newBBReader(b []byte) *bbReader { return &bbReader{buf: b} }

func (r *bbReader) Read(p []byte) (int, error) {
	if r.off >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}
