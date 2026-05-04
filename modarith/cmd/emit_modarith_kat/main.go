// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// emit_modarith_kat — produces the canonical cross-runtime KAT bundle
// for luxfi/math/modarith. The C++ side at luxcpp/crypto/math/test/
// modarith_cross_runtime_test.cpp reads the same JSON and asserts
// byte-equal Montgomery / Add / round-trip behavior on every entry.
//
// LP-107 Phase 6.3: Go emits → C++ verifies. Cross-runtime release gate
// for modular arithmetic.
//
// Usage:
//
//	go run ./cmd/emit_modarith_kat --out testdata/modarith_kat.json
//
// Each entry's input is a packed little-endian byte stream:
//
//	"MontMulMod/...":  q (8) || a (8) || b (8)   -> output = MulMod(a,b,q) (8)
//	"AddMod/...":      q (8) || a (8) || b (8)   -> output = (a+b) mod q  (8)
//	"MontgomeryRoundTrip/...": q (8) || x (8)    -> output = x (8)
//
// Determinism: operand stream is the SHA-256 hash chain of a fixed
// seed string. Same emit run → byte-equal JSON.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/luxfi/math/codec"
	"github.com/luxfi/math/modarith"
	"github.com/luxfi/math/params"
)

// Three odd primes used as moduli. PulsarQ is the canonical NTT-friendly
// prime; the others stress the Montgomery path on smaller and larger
// odd q values.
const (
	PulsarQ uint64 = 0x1000000004A01 // 49-bit Pulsar prime
	NTT998  uint64 = 998244353       // 30-bit classical NTT prime
	Q40Bit  uint64 = 0xFFFFFFFFC5    // 40-bit prime: 2^40 - 59
)

func main() {
	out := flag.String("out", "modarith_kat.json", "output JSON path")
	flag.Parse()

	header := params.KATHeader{
		ParameterSet:          "math.modarith.v1",
		ModulusID:             params.ModPulsarQ,
		BackendID:             params.BackendPureGo,
		HashSuiteID:           params.HashBLAKE3,
		ImplementationName:    "luxfi/math/modarith",
		ImplementationVersion: "v1.3.0",
	}

	moduli := []struct {
		name string
		q    uint64
	}{
		{"PulsarQ", PulsarQ},
		{"NTT998", NTT998},
		{"Q40Bit", Q40Bit},
	}

	entries := []codec.KATEntry{}

	// Per-modulus seed schedules. Each (test, modulus) pair gets a
	// distinct deterministic stream so different moduli don't share
	// operand bytes.
	for _, M := range moduli {
		mod, err := modarith.NewModulus(M.q, M.name)
		if err != nil {
			log.Fatalf("NewModulus(%s): %v", M.name, err)
		}

		// 100 entries: MontMulMod path equals MulMod (slow canonical).
		stream := newSHAStream("modarith/MontMulMod/" + M.name)
		for i := 0; i < 100; i++ {
			a := stream.next() % M.q
			b := stream.next() % M.q
			input := packU64(M.q, a, b)
			// Reference output: standard-form MulMod result.
			result := modarith.MulMod(a, b, M.q)
			// Verify Montgomery path produces the same value.
			aMont := modarith.ToMontgomery(a, mod)
			bMont := modarith.ToMontgomery(b, mod)
			pmont := modarith.MontMulMod(aMont, bMont, mod)
			if got := modarith.FromMontgomery(pmont, mod); got != result {
				log.Fatalf("MontMulMod cross-check failed: a=%d b=%d q=%d: mont=%d mulmod=%d",
					a, b, M.q, got, result)
			}
			output := packU64(result)
			entries = append(entries, mkEntry(header,
				fmt.Sprintf("MontMulMod/%s/i=%d", M.name, i), input, output))
		}

		// 100 entries: AddMod.
		stream = newSHAStream("modarith/AddMod/" + M.name)
		for i := 0; i < 100; i++ {
			a := stream.next() % M.q
			b := stream.next() % M.q
			input := packU64(M.q, a, b)
			result := modarith.AddMod(a, b, M.q)
			output := packU64(result)
			entries = append(entries, mkEntry(header,
				fmt.Sprintf("AddMod/%s/i=%d", M.name, i), input, output))
		}

		// 100 entries: MontgomeryRoundTrip — ToMontgomery -> FromMontgomery
		// recovers x byte-for-byte.
		stream = newSHAStream("modarith/MontgomeryRoundTrip/" + M.name)
		for i := 0; i < 100; i++ {
			x := stream.next() % M.q
			input := packU64(M.q, x)
			mont := modarith.ToMontgomery(x, mod)
			back := modarith.FromMontgomery(mont, mod)
			if back != x {
				log.Fatalf("Montgomery round-trip failed: x=%d q=%d mont=%d back=%d",
					x, M.q, mont, back)
			}
			output := packU64(back)
			entries = append(entries, mkEntry(header,
				fmt.Sprintf("MontgomeryRoundTrip/%s/i=%d", M.name, i), input, output))
		}
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

// packU64 returns a little-endian byte stream of the given uint64s.
func packU64(vs ...uint64) []byte {
	out := make([]byte, 8*len(vs))
	for i, v := range vs {
		binary.LittleEndian.PutUint64(out[i*8:], v)
	}
	return out
}

// shaStream is a SHA-256 hash chain that yields fresh uint64s. The seed
// is hashed with a counter so emit runs are byte-stable.
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
