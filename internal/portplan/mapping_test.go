// Copyright 2026 The go-simdutf Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package portplan

import "testing"

func TestFamilyContractDisplayIDV1(t *testing.T) {
	if got := FamilyContractDisplayIDV1("Base64"); got != "FC-v1-base64" {
		t.Fatalf("FamilyContractDisplayIDV1(Base64) = %q", got)
	}
}

func TestClassifyFamilyContractV1(t *testing.T) {
	tests := []struct {
		family, symbol, want string
	}{
		{"UTF-8", "ValidateUTF16", "FC-v1-helper-validation"},
		{"Transcoding/length", "ConvertLatin1ToUTF16", "FC-v1-latin1-source"},
		{"UTF-8", "CountUTF8", "FC-v1-utf8-source"},
		{"UTF-16", "Latin1LengthFromUTF16", "FC-v1-utf16-source"},
		{"UTF-32", "ChangeEndiannessUTF32", "FC-v1-utf32-source"},
		{"Encoding detection", "AutodetectEncoding", "FC-v1-detection"},
		{"UTF-16", "FindUTF16", "FC-v1-find"},
		{"Base64", "UnrelatedName", "FC-v1-base64"},
	}
	for _, tt := range tests {
		got, err := ClassifyFamilyContractV1(tt.family, tt.symbol)
		if err != nil || got != tt.want {
			t.Fatalf("ClassifyFamilyContractV1(%q, %q) = %q, %v; want %q", tt.family, tt.symbol, got, err, tt.want)
		}
	}
	for _, symbol := range []string{"FindAll", "UnknownOperation"} {
		if _, err := ClassifyFamilyContractV1("UTF-8", symbol); err == nil {
			t.Fatalf("ClassifyFamilyContractV1 accepted %q", symbol)
		}
	}
}

func TestDirectSymbolLowerFirstV1(t *testing.T) {
	for _, tt := range []struct{ in, want string }{
		{"UTF8Kernel", "utf8Kernel"},
		{"UTF16Kernel", "utf16Kernel"},
		{"UTF32Kernel", "utf32Kernel"},
		{"Base64Kernel", "base64Kernel"},
	} {
		if got := lowerFirst(tt.in); got != tt.want {
			t.Fatalf("lowerFirst(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateBackendMappingV1RejectsSourceNAEligibility(t *testing.T) {
	var manifest ManifestRowV1
	manifest.Cells[15] = "not_applicable: no source"
	mapping := ReviewedMappingV1{
		ISAOrdinalOrScalar:  "1",
		CanonicalKernelName: "utf8Kernel",
		Backends:            [4]BackendMappingV1{{Outcome: "eligible", DirectSymbol: "utf8Kernel", EvidenceAnchor: "source"}},
	}
	if err := validateBackendMapping(mapping, 0, manifest); err == nil {
		t.Fatal("eligible backend accepted despite source-declared N/A")
	}
}

func TestValidateBackendMappingV1UsesGoSymbolAndPinnedLedgerSource(t *testing.T) {
	var manifest ManifestRowV1
	manifest.Cells[3] = "src/api.h:1"
	manifest.Cells[15] = "src/amd64.go"
	var ledger ISARowV1
	ledger.Cells[3] = "src/westmere.go; annotated"
	mapping := ReviewedMappingV1{
		GoSymbol:            "ValidateUTF8",
		ISAOrdinalOrScalar:  "1",
		CanonicalKernelName: "SharedKernel",
		Backends: [4]BackendMappingV1{{
			Outcome:        "eligible",
			DirectSymbol:   "validateUTF8Westmere",
			EvidenceAnchor: "611becc2a08c27a4edc77d9a45ff74c97130129b src/api.h:1 src/westmere.go",
		}},
	}
	if err := validateBackendMapping(mapping, 0, manifest, &ledger); err != nil {
		t.Fatalf("validateBackendMapping() = %v", err)
	}
	mapping.Backends[0].DirectSymbol = "sharedKernelWestmere"
	if err := validateBackendMapping(mapping, 0, manifest, &ledger); err == nil {
		t.Fatal("canonical kernel-derived direct symbol accepted")
	}
}

func TestSourcePathTokenV1(t *testing.T) {
	if got := sourcePathToken("src/foo.go; generated (detail)"); got != "src/foo.go" {
		t.Fatalf("sourcePathToken() = %q", got)
	}
}
