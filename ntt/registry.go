// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package ntt

import (
	"sync"

	"github.com/luxfi/math/params"
)

// Process-wide registry of NTT backends. The pure-Go backend
// registers itself in init(); other backends (CUDA, Metal, WGSL) are
// registered by the build that includes them.

var (
	registryMu sync.RWMutex
	registry   = map[params.BackendID]Backend{}
)

// Register installs a Backend under its ID. Re-registration replaces.
// Backends MUST be idempotent — registering the same ID twice with
// different bodies is a programming error caught at process start
// when two libraries each try to register the same ID.
func Register(b Backend) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[b.ID()] = b
}

// Unregister removes a Backend. Used in tests.
func Unregister(id params.BackendID) {
	registryMu.Lock()
	defer registryMu.Unlock()
	delete(registry, id)
}

// lookup returns the registered Backend for id, or nil.
func lookup(id params.BackendID) Backend {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[id]
}

// registeredFor returns the set of registered BackendIDs whose
// Supports(p) returns true.
func registeredFor(p *Params) map[params.BackendID]bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make(map[params.BackendID]bool, len(registry))
	for id, b := range registry {
		if b.Supports(p) {
			out[id] = true
		}
	}
	return out
}
