// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause
//
// Body migrated from luxfi/lattice/v7/ring/vec_ops.go.
// LP-107 Phase 3 — only the helpers required by the NTT body
// (reducevec, mulscalarmontgomeryvec, mulscalarmontgomerylazyvec).

package canonical

import "unsafe"

// reducevec applies BRedAdd to every coefficient of p1 -> p2 (size multiple of 8).
func reducevec(p1, p2 []uint64, modulus uint64, bredconstant [2]uint64) {
	N := len(p1)
	for j := 0; j < N; j = j + 8 {
		/* #nosec G103 -- behavior and consequences well understood, possible buffer overflow if len(p1)%8 */
		x := (*[8]uint64)(unsafe.Pointer(&p1[j]))
		/* #nosec G103 -- behavior and consequences well understood, possible buffer overflow if len(p2)%8 */
		z := (*[8]uint64)(unsafe.Pointer(&p2[j]))

		z[0] = BRedAdd(x[0], modulus, bredconstant)
		z[1] = BRedAdd(x[1], modulus, bredconstant)
		z[2] = BRedAdd(x[2], modulus, bredconstant)
		z[3] = BRedAdd(x[3], modulus, bredconstant)
		z[4] = BRedAdd(x[4], modulus, bredconstant)
		z[5] = BRedAdd(x[5], modulus, bredconstant)
		z[6] = BRedAdd(x[6], modulus, bredconstant)
		z[7] = BRedAdd(x[7], modulus, bredconstant)
	}
}

// mulscalarmontgomeryvec multiplies p1 by scalarMont (Montgomery form) into p2.
func mulscalarmontgomeryvec(p1 []uint64, scalarMont uint64, p2 []uint64, modulus, mredconstant uint64) {
	N := len(p1)
	for j := 0; j < N; j = j + 8 {
		/* #nosec G103 -- behavior and consequences well understood, possible buffer overflow if len(p1)%8 */
		x := (*[8]uint64)(unsafe.Pointer(&p1[j]))
		/* #nosec G103 -- behavior and consequences well understood, possible buffer overflow if len(p2)%8 */
		z := (*[8]uint64)(unsafe.Pointer(&p2[j]))

		z[0] = MRed(x[0], scalarMont, modulus, mredconstant)
		z[1] = MRed(x[1], scalarMont, modulus, mredconstant)
		z[2] = MRed(x[2], scalarMont, modulus, mredconstant)
		z[3] = MRed(x[3], scalarMont, modulus, mredconstant)
		z[4] = MRed(x[4], scalarMont, modulus, mredconstant)
		z[5] = MRed(x[5], scalarMont, modulus, mredconstant)
		z[6] = MRed(x[6], scalarMont, modulus, mredconstant)
		z[7] = MRed(x[7], scalarMont, modulus, mredconstant)
	}
}

// mulscalarmontgomerylazyvec is the lazy variant of mulscalarmontgomeryvec.
func mulscalarmontgomerylazyvec(p1 []uint64, scalarMont uint64, p2 []uint64, modulus, mredconstant uint64) {
	N := len(p1)
	for j := 0; j < N; j = j + 8 {
		/* #nosec G103 -- behavior and consequences well understood, possible buffer overflow if len(p1)%8 */
		x := (*[8]uint64)(unsafe.Pointer(&p1[j]))
		/* #nosec G103 -- behavior and consequences well understood, possible buffer overflow if len(p2)%8 */
		z := (*[8]uint64)(unsafe.Pointer(&p2[j]))

		z[0] = MRedLazy(x[0], scalarMont, modulus, mredconstant)
		z[1] = MRedLazy(x[1], scalarMont, modulus, mredconstant)
		z[2] = MRedLazy(x[2], scalarMont, modulus, mredconstant)
		z[3] = MRedLazy(x[3], scalarMont, modulus, mredconstant)
		z[4] = MRedLazy(x[4], scalarMont, modulus, mredconstant)
		z[5] = MRedLazy(x[5], scalarMont, modulus, mredconstant)
		z[6] = MRedLazy(x[6], scalarMont, modulus, mredconstant)
		z[7] = MRedLazy(x[7], scalarMont, modulus, mredconstant)
	}
}
