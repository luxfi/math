// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package sample

// LP-107 Phase 6.5 cross-runtime KAT release gate.
//
// sample re-uses the canonical KATEntry / KATBundle types defined in
// luxfi/math/codec.

import "github.com/luxfi/math/codec"

// KATEntry is an alias for codec.KATEntry.
type KATEntry = codec.KATEntry

// KATBundle is an alias for codec.KATBundle.
type KATBundle = codec.KATBundle

// KATSchemaV1 is the canonical bundle schema string.
const KATSchemaV1 = codec.KATSchemaV1

// WriteKATBundleFile writes a bundle to disk; thin pass-through.
func WriteKATBundleFile(path string, b *KATBundle) error {
	return codec.WriteKATBundleFile(path, b)
}

// ReadKATBundleFile reads a bundle from disk; thin pass-through.
func ReadKATBundleFile(path string) (*KATBundle, error) {
	return codec.ReadKATBundleFile(path)
}

// HexEncode is the canonical hex encoder shared with codec.
func HexEncode(b []byte) string { return codec.HexEncode(b) }

// HexDecode is the inverse.
func HexDecode(s string) ([]byte, error) { return codec.HexDecode(s) }
