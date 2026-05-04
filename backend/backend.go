// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

// Package backend defines how `luxfi/math` selects between CPU and GPU
// implementations of the same primitive.
//
// LP-107 §"Backend dispatch" — the canonical motivation. Backends are
// interchangeable performance realizations of one canonical contract;
// KATs prove byte-equality across them; the choice of backend MUST
// NEVER alter transcript bytes for consensus paths.
//
// Policy enum:
//
//	BackendPureGo       — pure-Go reference implementation. Always works.
//	BackendNativeCPU    — native CPU implementation (cgo + C++/SIMD).
//	BackendGPUPreferred — try GPU; fall back to CPU on unavailability.
//	BackendGPURequired  — GPU MUST be available; error if not.
//
// Registry: each substrate package (math/ntt, math/poly, math/sample,
// math/codec, math/rns) exposes its own backend interface and a
// process-wide registry of registered backends. This package owns the
// shared Policy enum + lookup helper; per-primitive interfaces live in
// the consuming package.
package backend

import (
	"errors"
	"fmt"

	"github.com/luxfi/math/params"
)

// Policy selects a backend at dispatch time.
type Policy uint8

const (
	// PolicyPureGo forces the pure-Go reference path. Used for
	// debugging, incident response, and consensus-critical paths
	// where determinism trumps speed.
	PolicyPureGo Policy = 0

	// PolicyNativeCPU prefers the native-cpu (cgo / SIMD) backend if
	// available; falls back to pure-Go otherwise.
	PolicyNativeCPU Policy = 1

	// PolicyGPUPreferred prefers GPU if available, falls back to native-
	// CPU, then pure-Go. Used when the workload amortizes GPU dispatch.
	PolicyGPUPreferred Policy = 2

	// PolicyGPURequired demands GPU; the call errors out if no GPU
	// backend is registered.
	PolicyGPURequired Policy = 3
)

// String makes Policy printable.
func (p Policy) String() string {
	switch p {
	case PolicyPureGo:
		return "pure-go"
	case PolicyNativeCPU:
		return "native-cpu"
	case PolicyGPUPreferred:
		return "gpu-preferred"
	case PolicyGPURequired:
		return "gpu-required"
	default:
		return fmt.Sprintf("policy(%d)", uint8(p))
	}
}

// Validate reports whether p is a known policy.
func (p Policy) Validate() error {
	switch p {
	case PolicyPureGo, PolicyNativeCPU, PolicyGPUPreferred, PolicyGPURequired:
		return nil
	}
	return fmt.Errorf("backend: unknown Policy %d", uint8(p))
}

// ErrUnavailable signals that a required backend is not present in the
// process. Returned by ResolveOrError when PolicyGPURequired is set
// but no GPU backend is registered.
var ErrUnavailable = errors.New("backend: required backend unavailable")

// Resolve returns the appropriate BackendID for the given policy and
// the set of registered backends. The order of preference is:
//
//	PolicyGPURequired:  GPU only — error if no GPU registered
//	PolicyGPUPreferred: GPU > native-cpu > pure-go
//	PolicyNativeCPU:    native-cpu > pure-go
//	PolicyPureGo:       pure-go only
//
// This function is the single decision point for dispatch ordering.
// Per-primitive packages (math/ntt, math/poly, etc.) consult it before
// invoking a backend.
func Resolve(policy Policy, registered map[params.BackendID]bool) (params.BackendID, error) {
	if err := policy.Validate(); err != nil {
		return "", err
	}

	gpuFamily := []params.BackendID{
		params.BackendCUDA, params.BackendMetal, params.BackendWGSL,
	}
	cpuNative := []params.BackendID{
		params.BackendNative, params.BackendAVX2, params.BackendNEON,
	}

	first := func(family []params.BackendID) (params.BackendID, bool) {
		for _, id := range family {
			if registered[id] {
				return id, true
			}
		}
		return "", false
	}

	switch policy {
	case PolicyGPURequired:
		if id, ok := first(gpuFamily); ok {
			return id, nil
		}
		return "", fmt.Errorf("backend: %w (policy=GPURequired, registered=%v)",
			ErrUnavailable, keysOf(registered))
	case PolicyGPUPreferred:
		if id, ok := first(gpuFamily); ok {
			return id, nil
		}
		if id, ok := first(cpuNative); ok {
			return id, nil
		}
		if registered[params.BackendPureGo] {
			return params.BackendPureGo, nil
		}
		return "", fmt.Errorf("backend: no backend registered")
	case PolicyNativeCPU:
		if id, ok := first(cpuNative); ok {
			return id, nil
		}
		if registered[params.BackendPureGo] {
			return params.BackendPureGo, nil
		}
		return "", fmt.Errorf("backend: no backend registered")
	case PolicyPureGo:
		if registered[params.BackendPureGo] {
			return params.BackendPureGo, nil
		}
		return "", fmt.Errorf("backend: PureGo not registered (impossible — pure-Go is the canonical reference)")
	}
	return "", fmt.Errorf("backend: unreachable")
}

func keysOf(m map[params.BackendID]bool) []params.BackendID {
	out := make([]params.BackendID, 0, len(m))
	for k, v := range m {
		if v {
			out = append(out, k)
		}
	}
	return out
}
