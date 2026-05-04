// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// emit_ntt_kat — produces the canonical cross-runtime KAT bundle for
// luxfi/math/ntt. The C++ side at luxcpp/crypto/math/test/
// ntt_cross_runtime_test.cpp reads the same JSON and asserts byte-equal
// Forward-NTT output for every entry.
//
// LP-107 Phase 6.4: Go emits → C++ verifies. Cross-runtime release gate
// for the Number-Theoretic Transform.
//
// Usage:
//
//	go run ./cmd/emit_ntt_kat --out testdata/ntt_kat.json
//
// Each entry's input is the 256-element uint64 polynomial in standard
// (non-Montgomery) form, packed little-endian (256*8 = 2048 bytes).
// Output is the forward-NTT result, same packing (2048 bytes).
//
// Determinism: input coefficients are drawn from a SHA-256 hash chain
// of a fixed seed string, reduced mod PulsarQ.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/luxfi/math/backend"
	"github.com/luxfi/math/codec"
	"github.com/luxfi/math/ntt"
	"github.com/luxfi/math/params"
)

const PulsarQ uint64 = 0x1000000004A01

func main() {
	out := flag.String("out", "ntt_kat.json", "output JSON path")
	flag.Parse()

	header := params.KATHeader{
		ParameterSet:          "math.ntt.pulsar-n256.v1",
		ModulusID:             params.ModPulsarQ,
		BackendID:             params.BackendPureGo,
		HashSuiteID:           params.HashBLAKE3,
		ImplementationName:    "luxfi/math/ntt",
		ImplementationVersion: "v1.3.0",
	}

	p := &ntt.Params{
		N:  256,
		Q:  PulsarQ,
		ID: params.NTTPulsarN256,
	}
	if err := p.Validate(); err != nil {
		log.Fatalf("Params: %v", err)
	}
	svc, err := ntt.NewService(p, backend.PolicyPureGo)
	if err != nil {
		log.Fatalf("NewService: %v", err)
	}

	N := int(p.N)

	entries := []codec.KATEntry{}

	const numEntries = 50
	for i := 0; i < numEntries; i++ {
		seed := fmt.Sprintf("ntt/PulsarN256/i=%d", i)
		stream := newSHAStream(seed)
		input := make([]uint64, N)
		for j := 0; j < N; j++ {
			input[j] = stream.next() % PulsarQ
		}
		// Run forward NTT.
		work := make([]uint64, N)
		copy(work, input)
		if err := svc.Forward(work, 1); err != nil {
			log.Fatalf("Forward[%d]: %v", i, err)
		}
		// Pack input and output as little-endian byte streams.
		entries = append(entries, mkEntry(header,
			fmt.Sprintf("Forward/PulsarN256/i=%d", i),
			packU64s(input), packU64s(work)))
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

func packU64s(vs []uint64) []byte {
	out := make([]byte, 8*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint64(out[i*8:], v)
	}
	return out
}

// shaStream — SHA-256 hash chain that yields uint64 values. Same seed
// → byte-equal stream.
type shaStream struct {
	seed    []byte
	counter uint64
	buf     []byte
	off     int
}

func newSHAStream(seed string) *shaStream {
	return &shaStream{seed: []byte(seed)}
}

func (s *shaStream) next() uint64 {
	if s.off+8 > len(s.buf) {
		s.refill()
	}
	v := binary.LittleEndian.Uint64(s.buf[s.off:])
	s.off += 8
	return v
}

func (s *shaStream) refill() {
	h := sha256.New()
	h.Write(s.seed)
	var ctr [8]byte
	binary.LittleEndian.PutUint64(ctr[:], s.counter)
	h.Write(ctr[:])
	s.buf = h.Sum(nil)
	s.off = 0
	s.counter++
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
