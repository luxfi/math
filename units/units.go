// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package units provides Lux denomination constants
package units

// Lux denominations
const (
	NanoLux   uint64 = 1
	MicroLux  uint64 = 1000 * NanoLux
	Schmeckle uint64 = 49*MicroLux + 463*NanoLux // ≈ 49463 nLux
	MilliLux  uint64 = 1000 * MicroLux
	Lux       uint64 = 1000 * MilliLux
	KiloLux   uint64 = 1000 * Lux
	MegaLux   uint64 = 1000 * KiloLux
)

// Gas units
const (
	Wei   uint64 = 1
	GWei  uint64 = 1_000_000_000
	Ether uint64 = 1_000_000_000 * GWei
)

// Time units (for convenience)
const (
	Second uint64 = 1
	Minute uint64 = 60 * Second
	Hour   uint64 = 60 * Minute
	Day    uint64 = 24 * Hour
	Week   uint64 = 7 * Day
)

// Data size units
const (
	KiB uint64 = 1024
	MiB uint64 = 1024 * KiB
	GiB uint64 = 1024 * MiB
)
