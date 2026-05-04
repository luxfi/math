// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// emit_codec_kat — produces the canonical cross-runtime KAT bundle
// for luxfi/math/codec. The C++ side at luxcpp/crypto/math/test/
// codec_cross_runtime_test.cpp reads the same JSON and asserts
// byte-equal Reader behavior on every entry.
//
// LP-107 Phase 7: Go emits → C++ verifies. This is the first
// cross-runtime release gate for the math substrate.
//
// Usage:
//
//	go run ./cmd/emit_codec_kat --out testdata/codec_kat.json
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/luxfi/math/codec"
	"github.com/luxfi/math/params"
)

func main() {
	out := flag.String("out", "codec_kat.json", "output JSON path")
	flag.Parse()

	header := params.KATHeader{
		ParameterSet:          "math.codec.v1",
		ModulusID:             params.ModPulsarQ, // any valid ID; codec is modulus-independent
		BackendID:             params.BackendPureGo,
		HashSuiteID:           params.HashBLAKE3,
		ImplementationName:    "luxfi/math/codec",
		ImplementationVersion: "v0.2.0",
	}

	entries := []codec.KATEntry{}

	// Entry 1: happy-path uint64 slice with 3 elements.
	{
		payload := []uint64{0xdeadbeef, 0xcafebabe, 0x1122334455667788}
		input := codec.MakeUvarintFrame(uint64(len(payload)), payload)
		// Output is the same uint64 stream emitted as little-endian bytes
		// (canonical wire form). Verifier replays the input through
		// ReadUint64Slice and re-serializes the result; bytes must match.
		var output []byte
		for _, v := range payload {
			b := make([]byte, 8)
			binary.LittleEndian.PutUint64(b, v)
			output = append(output, b...)
		}
		entries = append(entries, mkEntry(header,
			"ReadUint64Slice/happy-path/3-elements", input, output))
	}

	// Entry 2: empty slice (length 0).
	{
		input := codec.MakeUvarintFrame(0, nil)
		entries = append(entries, mkEntry(header,
			"ReadUint64Slice/empty", input, nil))
	}

	// Entry 3: huge length attack input (regression for lattice issue #4).
	// The Reader MUST reject this; verifier records the rejection
	// distinctly via Output = "REJECTED" sentinel.
	{
		input := codec.MakeUvarintFrame(70_368_955_777_453, nil)
		entries = append(entries, mkRejectEntry(header,
			"ReadUint64Slice/reject/lattice-issue-4-70T", input))
	}

	// Entry 4: uint16 happy-path with 5 elements.
	{
		// Manually emit varint(5) then 10 bytes of payload.
		input := []byte{}
		// uvarint(5):
		input = append(input, 5)
		// 5 little-endian uint16 values: 0x0102, 0x0304, 0x0506, 0x0708, 0x090A.
		input = append(input, 0x02, 0x01, 0x04, 0x03, 0x06, 0x05, 0x08, 0x07, 0x0A, 0x09)
		// Output is the values re-serialized.
		output := []byte{0x02, 0x01, 0x04, 0x03, 0x06, 0x05, 0x08, 0x07, 0x0A, 0x09}
		entries = append(entries, mkEntry(header,
			"ReadUint16Slice/happy-path/5-elements", input, output))
	}

	// Entry 5: uint32 happy-path with 4 elements.
	{
		input := []byte{4}
		// 4 little-endian uint32 values: 0x12345678, 0x87654321, 0x00112233, 0xFFEEDDCC.
		input = append(input,
			0x78, 0x56, 0x34, 0x12,
			0x21, 0x43, 0x65, 0x87,
			0x33, 0x22, 0x11, 0x00,
			0xCC, 0xDD, 0xEE, 0xFF)
		output := input[1:] // payload bytes
		entries = append(entries, mkEntry(header,
			"ReadUint32Slice/happy-path/4-elements", input, output))
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

// mkRejectEntry records an input that the Reader MUST reject. The
// canonical "output" for a rejected input is the literal byte string
// "REJECTED" so cross-runtime verifiers can compare a single value.
func mkRejectEntry(h params.KATHeader, name string, input []byte) codec.KATEntry {
	const sentinel = "REJECTED"
	digest := sha256.Sum256([]byte(sentinel))
	return codec.KATEntry{
		Header:       h,
		Test:         name,
		InputHex:     codec.HexEncode(input),
		OutputHex:    codec.HexEncode([]byte(sentinel)),
		OutputSHA256: codec.HexEncode(digest[:]),
	}
}
