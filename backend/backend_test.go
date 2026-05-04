// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package backend

import (
	"errors"
	"testing"

	"github.com/luxfi/math/params"
)

func TestPolicy_String(t *testing.T) {
	for _, tc := range []struct {
		p    Policy
		want string
	}{
		{PolicyPureGo, "pure-go"},
		{PolicyNativeCPU, "native-cpu"},
		{PolicyGPUPreferred, "gpu-preferred"},
		{PolicyGPURequired, "gpu-required"},
	} {
		if got := tc.p.String(); got != tc.want {
			t.Errorf("Policy(%d).String() = %q, want %q", tc.p, got, tc.want)
		}
	}
}

func TestPolicy_Validate(t *testing.T) {
	for _, p := range []Policy{
		PolicyPureGo, PolicyNativeCPU, PolicyGPUPreferred, PolicyGPURequired,
	} {
		if err := p.Validate(); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	if err := Policy(99).Validate(); err == nil {
		t.Error("Policy(99).Validate() returned nil")
	}
}

func TestResolve_PureGo(t *testing.T) {
	r := map[params.BackendID]bool{params.BackendPureGo: true}
	got, err := Resolve(PolicyPureGo, r)
	if err != nil || got != params.BackendPureGo {
		t.Errorf("PureGo resolve: %v %s", err, got)
	}
}

func TestResolve_NativeCPU_Fallback(t *testing.T) {
	// Only pure-go registered; native-cpu policy must fall back.
	r := map[params.BackendID]bool{params.BackendPureGo: true}
	got, err := Resolve(PolicyNativeCPU, r)
	if err != nil || got != params.BackendPureGo {
		t.Errorf("NativeCPU fallback: %v %s", err, got)
	}

	// AVX2 registered: native-cpu should pick it.
	r2 := map[params.BackendID]bool{
		params.BackendPureGo: true, params.BackendAVX2: true,
	}
	got, err = Resolve(PolicyNativeCPU, r2)
	if err != nil || got != params.BackendNative && got != params.BackendAVX2 {
		t.Errorf("NativeCPU with AVX2: %v %s", err, got)
	}
}

func TestResolve_GPUPreferred_FallbackChain(t *testing.T) {
	// No GPU, no native — falls back to pure-go.
	r := map[params.BackendID]bool{params.BackendPureGo: true}
	got, err := Resolve(PolicyGPUPreferred, r)
	if err != nil || got != params.BackendPureGo {
		t.Errorf("GPUPreferred → pure-go fallback: %v %s", err, got)
	}

	// CUDA registered: GPUPreferred picks CUDA.
	r2 := map[params.BackendID]bool{
		params.BackendPureGo: true, params.BackendCUDA: true,
	}
	got, err = Resolve(PolicyGPUPreferred, r2)
	if err != nil || got != params.BackendCUDA {
		t.Errorf("GPUPreferred with CUDA: %v %s", err, got)
	}
}

func TestResolve_GPURequired_NoGPU_Errors(t *testing.T) {
	r := map[params.BackendID]bool{params.BackendPureGo: true}
	_, err := Resolve(PolicyGPURequired, r)
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("GPURequired with no GPU: want ErrUnavailable, got %v", err)
	}
}

func TestResolve_GPURequired_Metal_OK(t *testing.T) {
	r := map[params.BackendID]bool{params.BackendMetal: true}
	got, err := Resolve(PolicyGPURequired, r)
	if err != nil || got != params.BackendMetal {
		t.Errorf("GPURequired with Metal: %v %s", err, got)
	}
}

func TestResolve_UnknownPolicy(t *testing.T) {
	r := map[params.BackendID]bool{params.BackendPureGo: true}
	_, err := Resolve(Policy(99), r)
	if err == nil {
		t.Error("Resolve(unknown) returned nil")
	}
}
