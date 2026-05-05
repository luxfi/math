//go:build !(goexperiment.simd && amd64)

// SIMD stub for non-AVX2 / non-experimental-simd builds.
// LP-107 Phase 3 — verbatim from luxfi/lattice/v7/ring/ntt_simd_stub.go.

package subring

// nttCoreLazyAccel is the stub for non-SIMD builds.
// Returns false to indicate no acceleration was applied.
func nttCoreLazyAccel(p1, p2 []uint64, N int, Q, MRedConstant uint64, roots []uint64) bool {
	return false
}

// inttCoreLazyAccel is the stub for non-SIMD builds.
// Returns false to indicate no acceleration was applied.
func inttCoreLazyAccel(p1, p2 []uint64, N int, Q, MRedConstant uint64, roots []uint64) bool {
	return false
}
