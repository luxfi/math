// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package canonical owns the canonical Lattigo-derived Montgomery NTT
// body — SubRing layout, root-table generator, scalar primitives
// (MRed/MRedLazy/MForm/BRed/CRed/...), the loop-unrolled-by-16 NTT
// kernel, and the SIMD AVX2 dispatch hooks.
//
// LP-107 Phase 3: this package is the single source of truth for the
// Montgomery NTT body. luxfi/lattice/v7/ring re-exports it as a thin
// shim so its downstream consumers (luxfi/pulsar, luxfi/fhe,
// luxfi/threshold, ...) compile unchanged. Application callers should
// route through one of:
//
//   - github.com/luxfi/math/ntt           — the canonical Service /
//     Backend interface (recommended for new code).
//   - github.com/luxfi/lattice/v7/ring    — the historical SubRing API
//     for FHE-flavoured callers.
//
// This package is NOT a general-purpose import target: its surface is
// optimized for the two consumers above and is not part of the public
// luxfi/math semver contract for downstream code.
package canonical
