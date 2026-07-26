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

package simdutf

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const apacheLicenseHeader = `// Copyright 2026 The go-simdutf Authors.
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

`

// TestGoSourcesBeginWithApacheLicenseHeader is hand-authored Go-only license
// enforcement, not an upstream test vector.
func TestGoSourcesBeginWithApacheLicenseHeader(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		contents, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Errorf("read %s: %v", entry.Name(), err)
			continue
		}
		if !bytes.HasPrefix(contents, []byte(apacheLicenseHeader)) {
			t.Errorf("%s does not begin with the exact Apache license header from doc.go", entry.Name())
		}
	}
}

// TestAssemblySourcesBeginWithApacheLicenseHeader is hand-authored Go-only
// license enforcement, not an upstream test vector.
func TestAssemblySourcesBeginWithApacheLicenseHeader(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".s") {
			continue
		}
		contents, err := os.ReadFile(entry.Name())
		if err != nil {
			t.Errorf("read %s: %v", entry.Name(), err)
			continue
		}
		if !bytes.HasPrefix(contents, []byte(apacheLicenseHeader)) {
			t.Errorf("%s does not begin with the exact Apache license header from doc.go", entry.Name())
		}
	}
}

// TestSourcesRecordPinnedProvenance is hand-authored Go-only provenance
// enforcement, not an upstream test vector.
func TestSourcesRecordPinnedProvenance(t *testing.T) {
	const upstreamSHA = "dec3aad192f47081110d9c766d4917bad243906f"
	expectations := []provenanceExpectation{
		{"ascii.go", []string{
			upstreamSHA,
			"include/simdutf/implementation.h:315-455",
			"raw []uint16 storage encoding",
		}},
		{"ascii_scalar.go", []string{
			upstreamSHA,
			"include/simdutf/scalar/ascii.h:15-81",
			"include/simdutf/scalar/utf16.h:8-18",
			"src/fallback/implementation.cpp:49-73",
			"bounds-checked loads",
		}},
		{"ascii_test.go", []string{
			upstreamSHA,
			"tests/validate_ascii_basic_tests.cpp:8-125",
			"tests/validate_ascii_with_errors_tests.cpp:7-38",
			"tests/validate_utf16be_basic_tests.cpp:12-20,158-174",
			"tests/validate_utf16le_basic_tests.cpp:31-41,257-265",
			"do not claim byte-identical output to C++ mt19937 fixtures",
			"they do not prove absence of overreads",
		}},
		{"errors.go", []string{
			upstreamSHA,
			"include/simdutf/error.h",
		}},
		{"encoding.go", []string{
			upstreamSHA,
			"include/simdutf/encoding_types.h",
			"src/encoding_types.cpp",
		}},
		{"options.go", []string{
			upstreamSHA,
			"include/simdutf/implementation.h",
		}},
		{"errors_test.go", []string{
			upstreamSHA,
			"include/simdutf/error.h:7-124",
			"Narrow Go-only scaffolding",
			"underlying type, unknown values",
			"Go zero values",
			"these are not upstream test vectors",
		}},
		{"encoding_test.go", []string{
			upstreamSHA,
			"include/simdutf/encoding_types.h:15-24",
			"src/encoding_types.cpp:3-64",
			"Narrow Go-only scaffolding",
			"underlying type, unknown values",
			"truncated inputs, and non-prefix BOMs",
			"these are not upstream test vectors",
		}},
		{"options_test.go", []string{
			upstreamSHA,
			"include/simdutf/implementation.h:187-188,4094-4138,4194-4228",
			"Narrow Go-only scaffolding",
			"underlying types, unknown values",
			"typed bit composition",
			"these are not upstream test vectors",
		}},
		{"cpu_amd64.go", []string{
			upstreamSHA,
			"src/simdutf/westmere.h",
			"src/simdutf/haswell.h",
			"independent implementation",
			"not translated or structurally copied from",
			"include/simdutf/internal/isadetection.h",
		}},
		{"cpu_amd64.s", []string{
			upstreamSHA,
			"src/simdutf/westmere.h",
			"src/simdutf/haswell.h",
			"independently written",
			"No policy or code from",
			"include/simdutf/internal/isadetection.h",
		}},
		{"dispatch.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_amd64.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_archsimd_amd64.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_archsimd_stub.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_arm64.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"dispatch_generic.go", []string{
			upstreamSHA,
			"src/implementation.cpp",
			"Go-only dispatch glue",
			"this is not an\n// algorithm translation",
		}},
		{"test_helpers_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only test scaffolding",
			"test guards, direct-variant invocation, and provenance enforcement only",
			"does not define product behavior or port upstream algorithm vectors",
		}},
		{"benchmark_test.go", []string{
			upstreamSHA,
			"benchmarks/shortbench.cpp:419-422,493-497,520-526",
			"docs/porting/benchmark-contract.md",
			"Hand-authored Go-only benchmark scaffolding",
			"adds no product behavior, upstream algorithm vectors",
			"Benchmark function, or benchmark result",
		}},
	}
	requireProvenance(t, expectations...)
}
