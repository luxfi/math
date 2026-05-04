// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ntt

import (
	"fmt"
	"sync"

	"github.com/luxfi/math/params"

	"github.com/luxfi/lattice/v7/ring"
)

// pureGoBackend is the canonical pure-Go NTT realization. It delegates
// to github.com/luxfi/lattice/v7/ring's SubRing.NTT / INTT — the
// canonical Lattigo-derived Montgomery NTT — so callers see no
// behavior change vs the v0.1.x lattice path.
//
// LP-107 Phase 3 will invert this dependency: the canonical kernel
// body will live in this package, and luxfi/lattice will import
// luxfi/math/ntt to expose ring.SubRing.NTT.
type pureGoBackend struct {
	mu    sync.RWMutex
	rings map[params.NTTParamID]*ring.Ring
}

// PureGoBackend returns the singleton pure-Go NTT backend. Always
// available; registered automatically by init().
func PureGoBackend() Backend {
	return &thePureGo
}

var thePureGo = pureGoBackend{
	rings: make(map[params.NTTParamID]*ring.Ring),
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

// resolveRing returns or builds the cached *ring.Ring for p.
func (b *pureGoBackend) resolveRing(p *Params) (*ring.Ring, error) {
	b.mu.RLock()
	r, ok := b.rings[p.ID]
	b.mu.RUnlock()
	if ok {
		return r, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if r, ok := b.rings[p.ID]; ok {
		return r, nil
	}
	rr, err := ring.NewRing(int(p.N), []uint64{p.Q})
	if err != nil {
		return nil, fmt.Errorf("ntt(pure-go): ring.NewRing(N=%d, Q=%d): %w",
			p.N, p.Q, err)
	}
	b.rings[p.ID] = rr
	return rr, nil
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
	r, err := b.resolveRing(p)
	if err != nil {
		return err
	}
	sr := r.SubRings[0]
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
	r, err := b.resolveRing(p)
	if err != nil {
		return err
	}
	sr := r.SubRings[0]
	for i := uint32(0); i < batch; i++ {
		off := int(i) * N
		sr.INTT(dst[off:off+N], dst[off:off+N])
	}
	return nil
}
