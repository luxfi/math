// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ntt

import (
	"fmt"
	"sync"

	"github.com/luxfi/math/ntt/subring"
	"github.com/luxfi/math/params"
)

// pureGoBackend is the pure-Go NTT realization. It owns the
// Lattigo-derived Montgomery NTT body via the subring package;
// lattice/v7/ring re-exports the same body for downstream
// consumers, so callers see no behavior change vs the v0.1.x
// lattice path.
//
// LP-107 Phase 3 (this file): the kernel body lives in
// luxfi/math/ntt/subring. luxfi/lattice/ring is a thin shim
// that delegates to it.
type pureGoBackend struct {
	mu       sync.RWMutex
	subRings map[params.NTTParamID]*subring.SubRing
}

// PureGoBackend returns the singleton pure-Go NTT backend. Always
// available; registered automatically by init().
func PureGoBackend() Backend {
	return &thePureGo
}

var thePureGo = pureGoBackend{
	subRings: make(map[params.NTTParamID]*subring.SubRing),
}

func init() {
	Register(&thePureGo)
}

// ID implements Backend.
func (b *pureGoBackend) ID() params.BackendID { return params.BackendPureGo }

// Supports implements Backend. The pure-Go path supports any
// NTT-friendly (N, Q) — the validation in Params.Validate is the
// definitive gate.
func (b *pureGoBackend) Supports(p *Params) bool {
	return p != nil && p.Validate() == nil
}

// resolveSubRing returns or builds the cached *subring.SubRing for p.
func (b *pureGoBackend) resolveSubRing(p *Params) (*subring.SubRing, error) {
	b.mu.RLock()
	sr, ok := b.subRings[p.ID]
	b.mu.RUnlock()
	if ok {
		return sr, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if sr, ok := b.subRings[p.ID]; ok {
		return sr, nil
	}
	sr, err := subring.NewSubRing(int(p.N), p.Q)
	if err != nil {
		return nil, fmt.Errorf("ntt(pure-go): subring.NewSubRing(N=%d, Q=%d): %w",
			p.N, p.Q, err)
	}
	if err := sr.GenerateNTTConstants(); err != nil {
		return nil, fmt.Errorf("ntt(pure-go): GenerateNTTConstants(N=%d, Q=%d): %w",
			p.N, p.Q, err)
	}
	b.subRings[p.ID] = sr
	return sr, nil
}

// Forward implements Backend.
func (b *pureGoBackend) Forward(dst []uint64, p *Params, batch uint32) error {
	if !b.Supports(p) {
		return ErrUnsupportedParams
	}
	N := int(p.N)
	if int(batch)*N > len(dst) {
		return fmt.Errorf("ntt(pure-go): buffer too small: need %d got %d",
			int(batch)*N, len(dst))
	}
	sr, err := b.resolveSubRing(p)
	if err != nil {
		return err
	}
	for i := uint32(0); i < batch; i++ {
		off := int(i) * N
		sr.NTT(dst[off:off+N], dst[off:off+N])
	}
	return nil
}

// Inverse implements Backend.
func (b *pureGoBackend) Inverse(dst []uint64, p *Params, batch uint32) error {
	if !b.Supports(p) {
		return ErrUnsupportedParams
	}
	N := int(p.N)
	if int(batch)*N > len(dst) {
		return fmt.Errorf("ntt(pure-go): buffer too small: need %d got %d",
			int(batch)*N, len(dst))
	}
	sr, err := b.resolveSubRing(p)
	if err != nil {
		return err
	}
	for i := uint32(0); i < batch; i++ {
		off := int(i) * N
		sr.INTT(dst[off:off+N], dst[off:off+N])
	}
	return nil
}
