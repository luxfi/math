// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package ntt is the canonical Number-Theoretic-Transform interface for
// luxfi/math.
//
// LP-107 §"NTT" — the canonical motivation. Production callers
// (luxfi/lattice, luxfi/pulsar, luxfi/fhe) consume this package's
// Service abstraction; concrete kernels live behind a Backend
// interface so AVX2 / NEON / CUDA / Metal / WGSL realizations are
// interchangeable.
//
// Phase 2 (this file): defines the public surface. The pure-Go
// reference Backend wraps github.com/luxfi/lattice/v7/ring's
// SubRing.NTT — the canonical Lattigo-derived Montgomery NTT — so
// callers see no behavior change. Phase 3 (LP-107) inverts the
// dependency: lattice/ring imports this package, and the Lattigo
// kernel body lives here.
//
// Determinism contract: for a fixed (Params, input []uint64), every
// registered Backend MUST produce byte-equal output. KATs in
// luxfi/math/ntt/test/kat enforce this across runtimes.
package ntt

import (
	"errors"
	"fmt"

	"github.com/luxfi/math/backend"
	"github.com/luxfi/math/params"
)

// Params identifies one NTT instance: ring degree N, modulus Q, and
// the canonical parameter ID for KAT lookup.
type Params struct {
	// N is the ring dimension. Must be a power of two.
	N uint32
	// Q is the prime modulus. Must satisfy (Q - 1) | 2N (NTT-friendly).
	Q uint64
	// ID is the canonical parameter identifier (e.g. NTTPulsarN256).
	ID params.NTTParamID
}

// Validate ensures (N is a power of two) AND ((Q - 1) | 2N).
func (p *Params) Validate() error {
	if p == nil {
		return fmt.Errorf("ntt: nil Params")
	}
	if p.N == 0 || p.N&(p.N-1) != 0 {
		return fmt.Errorf("ntt: N=%d not a power of two", p.N)
	}
	if p.Q <= 1 {
		return fmt.Errorf("ntt: Q=%d invalid (must be > 1)", p.Q)
	}
	if (p.Q-1)%(2*uint64(p.N)) != 0 {
		return fmt.Errorf("ntt: Q-1=%d not divisible by 2N=%d (NTT-unfriendly)",
			p.Q-1, 2*uint64(p.N))
	}
	if err := p.ID.Validate(); err != nil {
		return err
	}
	return nil
}

// ErrUnsupportedParams is returned by a Backend that does not support
// the requested Params (e.g. CUDA backend asked for N=32 when it only
// implements N >= 256).
var ErrUnsupportedParams = errors.New("ntt: backend does not support these Params")

// Backend is the kernel interface every NTT realization implements.
// Forward and Inverse operate in-place and MUST produce byte-equal
// output across all registered backends for the same (Params, input).
type Backend interface {
	// ID returns the BackendID for this backend.
	ID() params.BackendID
	// Supports reports whether this backend can handle p.
	Supports(p *Params) bool
	// Forward applies the forward NTT in-place. dst must have length
	// batch*p.N. Returns ErrUnsupportedParams if Supports returns false.
	Forward(dst []uint64, p *Params, batch uint32) error
	// Inverse applies the inverse NTT in-place. Same length contract.
	Inverse(dst []uint64, p *Params, batch uint32) error
}

// Service binds a Params to a chosen Backend (resolved via dispatch
// policy) and exposes the public Forward / Inverse methods every
// downstream caller uses.
type Service struct {
	params  *Params
	backend Backend
	policy  backend.Policy
}

// NewService builds a Service for p under the given dispatch policy.
// The Backend is resolved at construction time from the registered
// backends; if no backend supports p, returns an error.
func NewService(p *Params, policy backend.Policy) (*Service, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	registered := registeredFor(p)
	id, err := backend.Resolve(policy, registered)
	if err != nil {
		return nil, fmt.Errorf("ntt.NewService: %w", err)
	}
	b := lookup(id)
	if b == nil {
		return nil, fmt.Errorf("ntt: backend %s registered but lookup returned nil", id)
	}
	return &Service{params: p, backend: b, policy: policy}, nil
}

// Params returns the bound parameter set.
func (s *Service) Params() *Params { return s.params }

// Backend returns the resolved BackendID. Callers print this in logs
// to record which path executed.
func (s *Service) Backend() params.BackendID { return s.backend.ID() }

// Forward applies the forward NTT in-place via the resolved backend.
func (s *Service) Forward(dst []uint64, batch uint32) error {
	return s.backend.Forward(dst, s.params, batch)
}

// Inverse applies the inverse NTT in-place via the resolved backend.
func (s *Service) Inverse(dst []uint64, batch uint32) error {
	return s.backend.Inverse(dst, s.params, batch)
}
