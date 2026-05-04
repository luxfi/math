// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package params is the single Lux registry of cryptographic parameter
// identifiers. Every other package in luxfi/math (and downstream
// luxfi/lattice, luxfi/pulsar, luxfi/fhe, luxfi/lens) keys off these
// IDs; every cross-runtime KAT carries them; every backend dispatch
// uses them to route work.
//
// LP-107 §"Parameter registry" — the canonical motivation. There must
// be exactly one place that names "Pulsar's modulus" or
// "FHE PN10QP27 ring dimension"; this package is that place.
//
// IDs are stable strings — wire-formatted, log-printable, KAT-keyed.
// Renaming an ID is a breaking change. New IDs append; existing IDs
// never change semantics.
package params

import "fmt"

// ModulusID names a single prime modulus.
//
// Production identifiers MUST satisfy: stable string, lowercase, hex
// representation of the modulus where applicable, prefixed by the
// owning protocol/scheme name. Validation: see Modulus.Validate.
type ModulusID string

const (
	// ModPulsarQ — Pulsar/LP-073 canonical NTT-friendly prime.
	// Q = 0x1000000004A01 ≈ 2^48; satisfies (Q - 1) | 2N for N = 256.
	ModPulsarQ ModulusID = "pulsar-q-0x1000000004a01"

	// ModNTT998 — classical NTT-friendly prime 998244353
	// (used by general vector kernels and tests; not production crypto).
	ModNTT998 ModulusID = "ntt-998244353"

	// ModFHE_PN10QP27 — first FHE production parameter set; 27-bit Q,
	// ring dimension N = 1024.
	ModFHE_PN10QP27 ModulusID = "fhe-pn10qp27"

	// ModFHE_PN11QP54 — second FHE production parameter set; 54-bit Q,
	// ring dimension N = 2048.
	ModFHE_PN11QP54 ModulusID = "fhe-pn11qp54"

	// ModFHE_PN9QP28_STD128 — STD128-tagged FHE parameter set;
	// ring dimension N = 512.
	ModFHE_PN9QP28_STD128 ModulusID = "fhe-pn9qp28-std128"
)

// String makes ModulusID printable.
func (m ModulusID) String() string { return string(m) }

// Validate reports whether m is a known modulus identifier in this
// process. Unknown IDs are rejected — there is no implicit registration.
func (m ModulusID) Validate() error {
	switch m {
	case ModPulsarQ,
		ModNTT998,
		ModFHE_PN10QP27,
		ModFHE_PN11QP54,
		ModFHE_PN9QP28_STD128:
		return nil
	}
	return fmt.Errorf("params: unknown ModulusID %q", string(m))
}

// NTTParamID names an (N, Q, root) triple for an NTT instance.
// One ModulusID may have multiple NTTParamID values (different N).
type NTTParamID string

const (
	// NTTPulsarN256 — Pulsar's R_q = Z_q[X]/(X^256 + 1) at Q = ModPulsarQ.
	NTTPulsarN256 NTTParamID = "pulsar-n256-q0x1000000004a01"

	// NTTFHE_PN10QP27_N1024 — FHE PN10QP27 ring at N = 1024.
	NTTFHE_PN10QP27_N1024 NTTParamID = "fhe-pn10qp27-n1024"

	// NTTFHE_PN11QP54_N2048 — FHE PN11QP54 ring at N = 2048.
	NTTFHE_PN11QP54_N2048 NTTParamID = "fhe-pn11qp54-n2048"

	// NTTFHE_PN9QP28_N512 — FHE PN9QP28 ring at N = 512.
	NTTFHE_PN9QP28_N512 NTTParamID = "fhe-pn9qp28-n512"
)

// String makes NTTParamID printable.
func (p NTTParamID) String() string { return string(p) }

// Validate reports whether p is a known NTT parameter identifier.
func (p NTTParamID) Validate() error {
	switch p {
	case NTTPulsarN256,
		NTTFHE_PN10QP27_N1024,
		NTTFHE_PN11QP54_N2048,
		NTTFHE_PN9QP28_N512:
		return nil
	}
	return fmt.Errorf("params: unknown NTTParamID %q", string(p))
}

// FHEParamID names a complete FHE scheme parameter set (ring + RNS
// chain + key-switching topology + bootstrap structure). Distinct
// from NTTParamID: one FHEParamID owns one or more NTTParamIDs.
type FHEParamID string

const (
	FHE_PN10QP27       FHEParamID = "fhe-pn10qp27"
	FHE_PN11QP54       FHEParamID = "fhe-pn11qp54"
	FHE_PN9QP28_STD128 FHEParamID = "fhe-pn9qp28-std128"
)

// String makes FHEParamID printable.
func (f FHEParamID) String() string { return string(f) }

// Validate reports whether f is a known FHE parameter set.
func (f FHEParamID) Validate() error {
	switch f {
	case FHE_PN10QP27, FHE_PN11QP54, FHE_PN9QP28_STD128:
		return nil
	}
	return fmt.Errorf("params: unknown FHEParamID %q", string(f))
}

// PulsarParamID names a Pulsar threshold-signature parameter set.
type PulsarParamID string

const (
	// PulsarLP073 — canonical LP-073 Pulsar parameter set.
	PulsarLP073 PulsarParamID = "pulsar-lp073"
)

// String makes PulsarParamID printable.
func (p PulsarParamID) String() string { return string(p) }

// Validate reports whether p is a known Pulsar parameter set.
func (p PulsarParamID) Validate() error {
	if p == PulsarLP073 {
		return nil
	}
	return fmt.Errorf("params: unknown PulsarParamID %q", string(p))
}

// HashSuiteID names a hash construction profile.
type HashSuiteID string

const (
	HashPulsarSHA3 HashSuiteID = "pulsar-sha3-v1"
	HashBLAKE3     HashSuiteID = "blake3-v1"
)

// String makes HashSuiteID printable.
func (h HashSuiteID) String() string { return string(h) }

// Validate reports whether h is a known hash suite.
func (h HashSuiteID) Validate() error {
	switch h {
	case HashPulsarSHA3, HashBLAKE3:
		return nil
	}
	return fmt.Errorf("params: unknown HashSuiteID %q", string(h))
}

// BackendID names a math-substrate backend (CPU pure-Go, native CPU,
// CUDA, Metal, WGSL). The same NTT/Modarith/Poly contract may be
// realized by multiple backends; KATs prove they produce byte-equal
// output.
type BackendID string

const (
	BackendPureGo BackendID = "pure-go"
	BackendNative BackendID = "native-cpu"
	BackendAVX2   BackendID = "avx2"
	BackendNEON   BackendID = "neon"
	BackendCUDA   BackendID = "cuda"
	BackendMetal  BackendID = "metal"
	BackendWGSL   BackendID = "wgsl"
)

// String makes BackendID printable.
func (b BackendID) String() string { return string(b) }

// Validate reports whether b is a known backend.
func (b BackendID) Validate() error {
	switch b {
	case BackendPureGo, BackendNative, BackendAVX2, BackendNEON,
		BackendCUDA, BackendMetal, BackendWGSL:
		return nil
	}
	return fmt.Errorf("params: unknown BackendID %q", string(b))
}

// KATHeader is the canonical key-set every KAT vector MUST carry.
// LP-107 §"Parameter registry" requirement: every KAT entry binds
// itself to a specific (parameter_set, modulus, backend, hash_suite,
// implementation_version) tuple so cross-runtime replay can match
// like-for-like.
type KATHeader struct {
	ParameterSet         string      `json:"parameter_set"`
	ModulusID            ModulusID   `json:"modulus_id"`
	BackendID            BackendID   `json:"backend_id"`
	HashSuiteID          HashSuiteID `json:"hash_suite_id"`
	ImplementationName   string      `json:"implementation_name"`
	ImplementationVersion string     `json:"implementation_version"`
}

// Validate ensures every required field is set and known.
func (h *KATHeader) Validate() error {
	if h == nil {
		return fmt.Errorf("params: nil KATHeader")
	}
	if h.ParameterSet == "" {
		return fmt.Errorf("params: KATHeader.ParameterSet is empty")
	}
	if err := h.ModulusID.Validate(); err != nil {
		return fmt.Errorf("KATHeader: %w", err)
	}
	if err := h.BackendID.Validate(); err != nil {
		return fmt.Errorf("KATHeader: %w", err)
	}
	if err := h.HashSuiteID.Validate(); err != nil {
		return fmt.Errorf("KATHeader: %w", err)
	}
	if h.ImplementationName == "" {
		return fmt.Errorf("params: KATHeader.ImplementationName is empty")
	}
	if h.ImplementationVersion == "" {
		return fmt.Errorf("params: KATHeader.ImplementationVersion is empty")
	}
	return nil
}
