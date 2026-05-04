// Copyright (c) 2026 Lux Industries Inc.
// SPDX-License-Identifier: BSD-3-Clause

package params

import "testing"

func TestModulusID_Validate(t *testing.T) {
	for _, id := range []ModulusID{
		ModPulsarQ, ModNTT998,
		ModFHE_PN10QP27, ModFHE_PN11QP54, ModFHE_PN9QP28_STD128,
	} {
		if err := id.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
	if err := ModulusID("not-a-real-id").Validate(); err == nil {
		t.Error("Validate(unknown) returned nil")
	}
}

func TestNTTParamID_Validate(t *testing.T) {
	for _, id := range []NTTParamID{
		NTTPulsarN256, NTTFHE_PN10QP27_N1024,
		NTTFHE_PN11QP54_N2048, NTTFHE_PN9QP28_N512,
	} {
		if err := id.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestFHEParamID_Validate(t *testing.T) {
	for _, id := range []FHEParamID{
		FHE_PN10QP27, FHE_PN11QP54, FHE_PN9QP28_STD128,
	} {
		if err := id.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestPulsarParamID_Validate(t *testing.T) {
	if err := PulsarLP073.Validate(); err != nil {
		t.Errorf("%s: %v", PulsarLP073, err)
	}
}

func TestHashSuiteID_Validate(t *testing.T) {
	for _, id := range []HashSuiteID{HashPulsarSHA3, HashBLAKE3} {
		if err := id.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestBackendID_Validate(t *testing.T) {
	for _, id := range []BackendID{
		BackendPureGo, BackendNative, BackendAVX2, BackendNEON,
		BackendCUDA, BackendMetal, BackendWGSL,
	} {
		if err := id.Validate(); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
}

func TestKATHeader_Validate(t *testing.T) {
	good := KATHeader{
		ParameterSet:          "pulsar-lp073",
		ModulusID:             ModPulsarQ,
		BackendID:             BackendPureGo,
		HashSuiteID:           HashPulsarSHA3,
		ImplementationName:    "luxfi/pulsar",
		ImplementationVersion: "v0.1.4",
	}
	if err := good.Validate(); err != nil {
		t.Errorf("good KATHeader: %v", err)
	}

	bad := KATHeader{}
	if err := bad.Validate(); err == nil {
		t.Error("empty KATHeader.Validate() returned nil")
	}

	noVer := good
	noVer.ImplementationVersion = ""
	if err := noVer.Validate(); err == nil {
		t.Error("missing ImplementationVersion returned nil")
	}

	var nilHdr *KATHeader
	if err := nilHdr.Validate(); err == nil {
		t.Error("nil KATHeader returned nil")
	}
}
