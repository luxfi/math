// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// emit_sample_kat — produces the canonical cross-runtime KAT bundle
// for luxfi/math/sample. The C++ side at luxcpp/crypto/math/test/
// sample_cross_runtime_test.cpp reads the same JSON, drives a byte-
// buffer reader with the recorded entropy, and asserts byte-equal
// sample output.
//
// LP-107 Phase 6.5: Go emits → C++ verifies. Cross-runtime release gate
// for primitive distribution samplers.
//
// Usage:
//
//	go run ./cmd/emit_sample_kat --out testdata/sample_kat.json
//
// Each entry contains:
//   Test:       "Uniform/q=PulsarQ/i=N"            -> param: q
//               "Ternary/q=PulsarQ/density=0.50/i=N" -> param: q, density
//               "CenteredBinomial/q=PulsarQ/eta=2/i=N" -> param: q, eta
//   InputHex:   raw entropy bytes (4096) from a SHA-256 hash chain
//   OutputHex:  first 64 samples, packed little-endian (512 bytes)
//
// Entropy bytes are deterministic from the test name + index seed so
// emit runs are byte-stable.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/luxfi/math/codec"
	"github.com/luxfi/math/params"
	"github.com/luxfi/math/sample"
)

const (
	PulsarQ uint64 = 0x1000000004A01
	NTT998  uint64 = 998244353

	NumSamples    = 64
	EntropyBytes  = 4096
)

func main() {
	out := flag.String("out", "sample_kat.json", "output JSON path")
	flag.Parse()

	header := params.KATHeader{
		ParameterSet:          "math.sample.v1",
		ModulusID:             params.ModPulsarQ,
		BackendID:             params.BackendPureGo,
		HashSuiteID:           params.HashBLAKE3,
		ImplementationName:    "luxfi/math/sample",
		ImplementationVersion: "v1.3.0",
	}

	entries := []codec.KATEntry{}

	// 10 Uniform/PulsarQ entries.
	for i := 0; i < 10; i++ {
		seed := fmt.Sprintf("sample/Uniform/PulsarQ/i=%d", i)
		entropy := genEntropy(seed, EntropyBytes)
		dst := make([]uint64, NumSamples)
		if err := sample.Uniform(dst, PulsarQ, byteReader(entropy)); err != nil {
			log.Fatalf("Uniform[%d]: %v", i, err)
		}
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("Uniform/q=PulsarQ/i=%d", i),
			entropy, packU64s(dst)))
	}

	// 10 Uniform/NTT998 entries (smaller modulus, different mask).
	for i := 0; i < 10; i++ {
		seed := fmt.Sprintf("sample/Uniform/NTT998/i=%d", i)
		entropy := genEntropy(seed, EntropyBytes)
		dst := make([]uint64, NumSamples)
		if err := sample.Uniform(dst, NTT998, byteReader(entropy)); err != nil {
			log.Fatalf("Uniform/NTT998[%d]: %v", i, err)
		}
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("Uniform/q=NTT998/i=%d", i),
			entropy, packU64s(dst)))
	}

	// 5 Ternary entries at density=0.5.
	for i := 0; i < 5; i++ {
		seed := fmt.Sprintf("sample/Ternary/d=0.50/i=%d", i)
		entropy := genEntropy(seed, EntropyBytes)
		dst := make([]uint64, NumSamples)
		if err := sample.Ternary(dst, PulsarQ, 0.5, byteReader(entropy)); err != nil {
			log.Fatalf("Ternary[%d]: %v", i, err)
		}
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("Ternary/q=PulsarQ/density=0.50/i=%d", i),
			entropy, packU64s(dst)))
	}

	// 3 Ternary entries at density=0.25.
	for i := 0; i < 3; i++ {
		seed := fmt.Sprintf("sample/Ternary/d=0.25/i=%d", i)
		entropy := genEntropy(seed, EntropyBytes)
		dst := make([]uint64, NumSamples)
		if err := sample.Ternary(dst, PulsarQ, 0.25, byteReader(entropy)); err != nil {
			log.Fatalf("Ternary/0.25[%d]: %v", i, err)
		}
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("Ternary/q=PulsarQ/density=0.25/i=%d", i),
			entropy, packU64s(dst)))
	}

	// 5 CenteredBinomial entries at eta=2.
	for i := 0; i < 5; i++ {
		seed := fmt.Sprintf("sample/CBD/eta=2/i=%d", i)
		entropy := genEntropy(seed, EntropyBytes)
		dst := make([]uint64, NumSamples)
		if err := sample.CenteredBinomial(dst, PulsarQ, 2, byteReader(entropy)); err != nil {
			log.Fatalf("CBD[%d]: %v", i, err)
		}
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("CenteredBinomial/q=PulsarQ/eta=2/i=%d", i),
			entropy, packU64s(dst)))
	}

	// 3 CenteredBinomial entries at eta=4.
	for i := 0; i < 3; i++ {
		seed := fmt.Sprintf("sample/CBD/eta=4/i=%d", i)
		entropy := genEntropy(seed, EntropyBytes)
		dst := make([]uint64, NumSamples)
		if err := sample.CenteredBinomial(dst, PulsarQ, 4, byteReader(entropy)); err != nil {
			log.Fatalf("CBD/eta=4[%d]: %v", i, err)
		}
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("CenteredBinomial/q=PulsarQ/eta=4/i=%d", i),
			entropy, packU64s(dst)))
	}

	bundle := &codec.KATBundle{
		Schema:  codec.KATSchemaV1,
		Entries: entries,
	}
	if err := codec.WriteKATBundleFile(*out, bundle); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d entries to %s\n", len(entries), *out)
}

// genEntropy returns n bytes from a SHA-256 hash chain seeded by s.
func genEntropy(s string, n int) []byte {
	out := make([]byte, 0, n)
	var counter uint64
	for len(out) < n {
		h := sha256.New()
		h.Write([]byte(s))
		var ctr [8]byte
		binary.LittleEndian.PutUint64(ctr[:], counter)
		h.Write(ctr[:])
		out = append(out, h.Sum(nil)...)
		counter++
	}
	return out[:n]
}

// byteReader returns a stateful io.Reader over the supplied bytes.
// Each Read advances the internal offset.
func byteReader(buf []byte) io.Reader { return &bbReader{buf: buf} }

type bbReader struct {
	buf []byte
	off int
}

func (r *bbReader) Read(p []byte) (int, error) {
	if r.off >= len(r.buf) {
		return 0, io.EOF
	}
	n := copy(p, r.buf[r.off:])
	r.off += n
	return n, nil
}

func packU64s(vs []uint64) []byte {
	out := make([]byte, 8*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint64(out[i*8:], v)
	}
	return out
}

func mkEntry(h params.KATHeader, name string, input, output []byte) codec.KATEntry {
	digest := sha256.Sum256(output)
	return codec.KATEntry{
		Header:       h,
		Test:         name,
		InputHex:     codec.HexEncode(input),
		OutputHex:    codec.HexEncode(output),
		OutputSHA256: codec.HexEncode(digest[:]),
	}
}
