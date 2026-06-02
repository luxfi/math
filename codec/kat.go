// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package codec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/luxfi/math/params"
)

// KATEntry is one cross-runtime KAT vector. Each entry pins a specific
// (test, parameter set, backend, input) tuple and records the SHA-256
// (or BLAKE2b) digest of the byte-stream the substrate produces for
// that input.
//
// Cross-runtime contract: emitting `KATEntry.Output` from Go and
// replaying it from the C++ side (luxcpp/crypto/math/test) MUST
// produce a byte-equal stream. Any divergence is a release-gate
// failure.
type KATEntry struct {
	// Header carries the canonical (parameter, backend, hash-suite,
	// implementation, version) tuple every KAT must surface.
	Header params.KATHeader `json:"header"`

	// Test is the human-readable test name (e.g. "ReadUint64Slice/
	// happy-path-3-elements", "MontMul/q=PulsarQ/100-random-pairs").
	Test string `json:"test"`

	// InputHex is the hex-encoded input byte stream consumed by the
	// substrate primitive under test.
	InputHex string `json:"input_hex"`

	// OutputHex is the hex-encoded canonical output byte stream.
	OutputHex string `json:"output_hex"`

	// OutputSHA256 is the SHA-256 commitment over OutputHex's raw
	// bytes (NOT the hex string). Used as a fast cross-runtime
	// equality check; full byte-stream is in OutputHex for diffing
	// on mismatch.
	OutputSHA256 string `json:"output_sha256"`
}

// KATBundle is a collection of entries written to a JSON file at a
// stable path. C++ replay tests load the same file by path.
type KATBundle struct {
	Schema  string     `json:"schema"` // "lux.math.kat.v1"
	Entries []KATEntry `json:"entries"`
}

const KATSchemaV1 = "lux.math.kat.v1"

// WriteKATBundle serializes the bundle to a writer in canonical JSON
// (sorted keys, indent=2). Two runs produce byte-equal output when
// the entries are byte-equal.
func WriteKATBundle(w io.Writer, b *KATBundle) error {
	if b.Schema == "" {
		b.Schema = KATSchemaV1
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(b)
}

// WriteKATBundleFile writes the bundle to a file at path.
func WriteKATBundleFile(path string, b *KATBundle) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("codec.WriteKATBundleFile: %w", err)
	}
	defer f.Close()
	return WriteKATBundle(f, b)
}

// ReadKATBundle deserializes a bundle from a reader.
func ReadKATBundle(r io.Reader) (*KATBundle, error) {
	var b KATBundle
	if err := json.NewDecoder(r).Decode(&b); err != nil {
		return nil, fmt.Errorf("codec.ReadKATBundle: %w", err)
	}
	if b.Schema != KATSchemaV1 {
		return nil, fmt.Errorf("codec.ReadKATBundle: unknown schema %q", b.Schema)
	}
	return &b, nil
}

// ReadKATBundleFile reads a bundle from a file at path.
func ReadKATBundleFile(path string) (*KATBundle, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("codec.ReadKATBundleFile: %w", err)
	}
	defer f.Close()
	return ReadKATBundle(f)
}

// MakeUvarintFrame returns the wire-format byte stream for a length-
// prefixed slice of uint64 with the supplied length. Used by KAT
// emitters to construct the canonical input bytes that ReadUint64Slice
// consumes.
func MakeUvarintFrame(length uint64, payload []uint64) []byte {
	var buf bytes.Buffer
	encodeUvarintTo(&buf, length)
	for _, v := range payload {
		_ = binary.Write(&buf, binary.LittleEndian, v)
	}
	return buf.Bytes()
}

func encodeUvarintTo(out *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		out.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	out.WriteByte(byte(v))
}

// HexEncode is a thin wrapper around encoding/hex for KAT JSON authoring.
func HexEncode(b []byte) string { return hex.EncodeToString(b) }

// HexDecode is the inverse.
func HexDecode(s string) ([]byte, error) { return hex.DecodeString(s) }
