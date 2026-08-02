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
	const upstreamSHA = "611becc2a08c27a4edc77d9a45ff74c97130129b"
	expectations := []provenanceExpectation{
		{"utf8.go", []string{
			upstreamSHA,
			"include/simdutf/implementation.h:253-306,3931-3938",
			"Go slices replace C++\n// pointer/length",
		}},
		{"utf8_scalar.go", []string{
			upstreamSHA,
			"include/simdutf/scalar/utf8.h:9-218,258-268",
			"src/fallback/implementation.cpp:35-48,431-433",
			"credit: based on code from Google Fuchsia (Apache Licensed)",
			"bounds-checked loads",
		}},
		{"count_utf8_test.go", []string{
			upstreamSHA,
			"tests/count_utf8.cpp:11-84",
			"upstream byte sizes and ASCII/one-to-four-byte mixture categories",
			"it does\n// not claim byte-identical output",
			"counts every byte except UTF-8 continuation bytes 0x80..0xbf",
		}},
		{"count_utf8_test.go", []string{
			upstreamSHA,
			"fuzz/conversion.cpp and tests/count_utf8.cpp:11-84",
			"Go-only public-versus-scalar differential fuzz scaffold",
			"every registered direct\n// accelerated implementation",
		}},
		{"utf8_length.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"include/simdutf/implementation.h:1673-1778,3954-3983",
			"Go slices replace C++\n// pointer/length pairs",
		}},
		{"utf8_length_scalar.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Portions Copyright 2021 The simdutf Authors",
			"include/simdutf/scalar/utf8.h:258-325",
			"src/fallback/implementation.cpp:436-440,476-480,525-529",
		}},
		{"utf8_length_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"tests/null_safety_tests.cpp:65-73",
			"tests/simdutf_c_tests.cpp:254-265",
			"tests/readme_tests.cpp:122-141",
			"include/simdutf/scalar/utf8.h:258-325",
		}},
		{"utf8_length_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Go-only public/direct-dispatch-versus-scalar differential fuzz scaffold",
			"fuzz/conversion.cpp",
			"fuzz/roundtrip.cpp",
			"fuzz/misc.cpp",
			"The scalar functions are the permanent arbitrary-byte Go oracles",
			"ret + 3 < N",
		}},
		{"utf8_length_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Hand-authored Go-only direct UTF-8 length benchmark registry scaffolding",
			"benchmarks/shortbench.cpp:29-65,419-422,493-497,520-526",
			"test-only named\n// variant slots and adds no product behavior or mutable dispatch override",
		}},
		{"utf8_length_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Hand-authored Go-only direct UTF-8 length differential fuzz registry",
			"fuzz/conversion.cpp",
			"include/simdutf/scalar/utf8.h:258-325",
			"test metadata only and adds no product behavior or mutable",
		}},
		{"benchmark_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"benchmarks/shortbench.cpp:29-65,419-422,493-497,520-526",
			"benchmarks/src/benchmark.cpp:167-169,999-1011",
			"processes input_data.size()/4 input bytes",
			"Public, direct-dispatch, and scalar rows share identical setup",
			"Latin1LengthFromUTF8 and TrimPartialUTF8 have no registered",
		}},
		{"count_utf8_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct CountUTF8 benchmark registry scaffolding",
			"test-only variant slots and adds no product behavior",
		}},
		{"count_utf8_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct CountUTF8 differential fuzz registry",
			"test\n// metadata only and adds no product behavior",
		}},
		{"benchmark_test.go", []string{
			upstreamSHA,
			"benchmarks/shortbench.cpp:29-40,66-72,419-422,493-497,520-526",
			"benchmarks/src/benchmark.cpp:3428-3443",
			"shortbench's frozen\n// zero prefixes",
			"benchmarks/dataset/emoji.txt",
			"Public and scalar rows deliberately share identical corpus setup and names",
		}},
		{"count_utf8_arm64.go", []string{
			upstreamSHA,
			"Portions Copyright 2021 The simdutf Authors",
			"src/generic/utf8.h:8-17",
			"src/arm64/implementation.cpp:1113-1117",
			"src/simdutf/arm64/simd.h:446-555",
			"complete\n// 64-byte blocks only",
			"pinned scalar tail",
		}},
		{"count_utf8_arm64.s", []string{
			upstreamSHA,
			"Portions Copyright 2021 The simdutf Authors",
			"Independent Go arm64 assembly translation",
			"src/generic/utf8.h:8-17",
			"src/arm64/implementation.cpp:1113-1117",
			"src/simdutf/arm64/simd.h:446-555",
			"64-byte block",
			"Go 1.26's Plan 9 arm64 assembler exposes VCMEQ but no signed integer",
			"byte lanes therefore cannot overflow across iterations",
			"input.gt(-65)",
		}},
		{"count_utf8_arm64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct scalar-differential coverage",
			"src/generic/utf8.h:8-17",
			"src/arm64/implementation.cpp:1113-1117",
			"src/simdutf/arm64/simd.h:446-555",
			"Go 1.26's Plan 9 arm64 assembler has VCMEQ but no signed integer",
		}},
		{"count_utf8_arm64_test.go", []string{
			upstreamSHA,
			"Go-only direct benchmark registration",
			"changes no\n// frozen benchmark name, corpus, or setup",
		}},
		{"count_utf8_arm64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct fuzz registration",
			"src/generic/utf8.h:8-17",
			"src/arm64/implementation.cpp:1113-1117",
		}},
		{"page_guard_arm64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only guard-page coverage",
			"complete-block-only loads",
			"src/generic/utf8.h:8-17",
			"src/arm64/implementation.cpp:1113-1117",
		}},
		{"count_utf8_amd64.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Portions Copyright 2021 The simdutf Authors",
			"src/generic/utf8.h:8-68",
			"src/westmere/implementation.cpp:1142-1146",
			"src/haswell/implementation.cpp:1115-1119",
			"64-byte (Westmere) or 128-byte (Haswell) groups only",
			"pinned scalar tail",
		}},
		{"count_utf8_amd64.s", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Portions Copyright 2021 The simdutf Authors",
			"Independent Go assembly translations",
			"src/generic/utf8.h:21-68",
			"src/westmere/implementation.cpp:1142-1146",
			"src/haswell/implementation.cpp:1115-1119",
			"PCMPGTB/VPCMPGTB implement the\n// pinned signed int8 predicate input > -65",
			"exactly 63\n// iterations (255/4)",
		}},
		{"count_utf8_amd64_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Hand-authored Go-only direct differential coverage",
			"Westmere and Haswell count_code_points_bytemask ports",
			"src/generic/utf8.h:21-68",
		}},
		{"count_utf8_amd64_test.go", []string{
			upstreamSHA,
			"Go-only direct benchmark registration",
			"changes no\n// frozen benchmark name, corpus, or setup",
		}},
		{"count_utf8_amd64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct fuzz registration",
			"Westmere\n// and Haswell count_code_points_bytemask assembly ports",
		}},
		{"page_guard_amd64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only guard-page coverage",
			"complete-group-only loads",
			"Westmere and Haswell count_code_points_bytemask ports",
		}},
		{"count_utf8_archsimd_amd64.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Portions Copyright 2021 The simdutf Authors",
			"Independently adapted from the Haswell count_code_points_bytemask family",
			"src/generic/utf8.h:21-68",
			"src/haswell/implementation.cpp:1115-1119",
			"four\n// Uint8x32 vectors",
			"signed int8 predicate input > -65",
			"after exactly 63 iterations (255/4)",
			"pinned arbitrary-byte scalar oracle",
			"Go 1.26.5 archsimd API provenance",
			"src/simd/archsimd/extra_amd64.go:9-17",
		}},
		{"count_utf8_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"c8292790d793212ca0a1faf6ae42e7f8e7b70d4f",
			"Hand-authored Go-only direct scalar-differential coverage",
			"archsimd\n// Haswell count_code_points_bytemask adaptation",
			"src/generic/utf8.h:21-68",
			"src/haswell/implementation.cpp:1115-1119",
		}},
		{"count_utf8_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"Go-only direct benchmark and differential-fuzz registration",
			"changes no\n// frozen benchmark name, corpus, or setup",
		}},
		{"page_guard_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only physical guard-page coverage",
			"src/generic/utf8.h:21-68",
			"src/haswell/implementation.cpp:1115-1119",
		}},
		{"utf8_test.go", []string{
			upstreamSHA,
			"tests/validate_utf8_basic_tests.cpp:7-130",
			"tests/validate_utf8_with_errors_tests.cpp:7-205",
			"tests/validate_utf8_puzzler_tests.cpp:5-39",
			"Named map cases",
		}},
		{"utf8_reference_test.go", []string{
			upstreamSHA,
			"tests/reference/validate_utf8.cpp:7-78",
			"validate_utf8.h:3-8",
			"credit: based on code from Google Fuchsia (Apache Licensed)",
			"tests/validate_utf8_basic_tests.cpp:24-108",
			"additional tests are from autobahn websocket testsuite",
			"https://github.com/crossbario/autobahn-testsuite/tree/master/autobahntestsuite/autobahntestsuite/case",
			"tests/validate_utf8_brute_force_tests.cpp:7-86",
			"Go uses a deterministic Go",
		}},
		{"utf8_test.go", []string{
			upstreamSHA,
			"fuzz/conversion.cpp:68-74",
			"Go-only public-versus-scalar differential fuzz scaffold",
			"every registered direct accelerated implementation",
		}},
		{"utf8_amd64.go", []string{
			upstreamSHA,
			"Portions Copyright 2021 The simdutf Authors",
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
			"src/westmere/implementation.cpp:19-29",
			"src/haswell/implementation.cpp:19-29",
			"at most five continuation bytes",
		}},
		{"utf8_archsimd_amd64.go", []string{
			upstreamSHA,
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
			"src/haswell/implementation.cpp:19-29",
			"Independently adapted",
			"VPERM2I128, VPALIGNR, VPSHUFB, VPSRLW, and VPSUBUSB",
		}},
		{"utf8_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
			"Direct differential coverage",
		}},
		{"utf8_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"Go-only direct benchmark and scalar-differential fuzz registration",
			"tagged lookup4 adaptation",
		}},
		{"page_guard_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"Hand-authored Go-only direct no-overread coverage",
			"invokes\n// tagged test functions only and adds no product behavior",
		}},
		{"utf8_amd64.s", []string{
			upstreamSHA,
			"Independent Go assembly translations",
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
			"DATA ·utf8LookupHigh<>+0(SB)/8, $0x0202020202020202",
			"DATA ·utf8LookupLow<>+8(SB)/8, $0xcbcbdbcbcbcbcbcb",
			"DATA ·utf8LookupInput<>+8(SB)/8, $0x01010101babaaee6",
			"VZEROUPPER",
		}},
		{"utf8_amd64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct differential and complete-block contract",
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
		}},
		{"utf8_amd64_test.go", []string{
			upstreamSHA,
			"Go-only registration of the direct amd64 lookup4 implementations",
			"no\n// additional product behavior",
		}},
		{"utf8_amd64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct fuzz registration",
			"adds no product behavior",
		}},
		{"page_guard_amd64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only deterministic no-overread coverage",
			"invokes direct",
		}},
		{"utf8_arm64.go", []string{
			upstreamSHA,
			"Portions Copyright 2021 The simdutf Authors",
			"src/arm64/implementation.cpp:13-28",
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
			"include/simdutf/scalar/utf8.h:225-251",
			"direct Go adaptation of the pinned",
		}},
		{"utf8_arm64.s", []string{
			upstreamSHA,
			"src/arm64/implementation.cpp:13-28",
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"Independent Go arm64 assembly translation",
			"DATA ·utf8Lookup4Byte1HighNEON<>+0(SB)/8, $0x0202020202020202",
			"DATA ·utf8Lookup4Byte1LowNEON<>+8(SB)/8, $0xcbcbdbcbcbcbcbcb",
			"DATA ·utf8Lookup4Byte2HighNEON<>+8(SB)/8, $0x01010101babaaee6",
			"DATA ·utf8Lookup4IncompleteMaxNEON<>+8(SB)/8, $0xbfdfefffffffffff",
			"GLOBL ·utf8Lookup4IncompleteMaxNEON<>(SB), RODATA|NOPTR, $16",
			"four chunks while keeping checker state in vector registers",
		}},
		{"utf8_arm64_test.go", []string{
			upstreamSHA,
			"src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216",
			"src/generic/utf8_validation/utf8_validator.h:10-80",
			"Hand-authored Go-only direct differential coverage",
		}},
		{"utf8_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct UTF-8 differential fuzz registry scaffolding",
			"test metadata only and adds no product behavior",
		}},
		{"utf8_arm64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct fuzz registration",
			"registers test functions only and adds no product behavior",
		}},
		{"utf8_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only direct UTF-8 benchmark registry scaffolding",
			"test-only variant slots and adds no product behavior",
		}},
		{"utf8_arm64_test.go", []string{
			upstreamSHA,
			"Go-only registration of the direct arm64 lookup4 implementation",
			"defines no\n// product dispatch behavior and translates no additional upstream algorithm",
		}},
		{"page_guard_arm64_test.go", []string{
			upstreamSHA,
			"Hand-authored Go-only deterministic no-overread coverage",
			"invokes direct test functions only and adds no product behavior",
		}},
		{"benchmark_test.go", []string{
			upstreamSHA,
			"benchmarks/shortbench.cpp:29-40,419-422,493-497,520-526",
			"benchmarks/src/benchmark.cpp:611-645",
			"benchmarks/dataset/emoji.txt",
			"ValidateUTF8 keeps shortbench's frozen",
			"Public and scalar rows deliberately",
		}},
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
		{"ascii_amd64.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"src/simdutf/westmere/simd.h:168-170,290-297",
			"src/simdutf/haswell/simd.h:177-179,293-300",
			"Independently translated to Go assembly",
			"complete blocks only",
		}},
		{"ascii_amd64.s", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"src/simdutf/westmere/simd.h:168-170,290-297",
			"src/simdutf/haswell/simd.h:177-179,293-300",
			"Independent Go assembly translations",
			"complete-block loops",
		}},
		{"ascii_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go differential and block-contract tests",
			"independent\n// assembly translation",
		}},
		{"ascii_arm64.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/arm64/implementation.cpp:13-16",
			"src/arm64/implementation.cpp:253-298",
			"src/arm64/arm_validate_utf16.cpp:71-91",
			"Translated and adapted",
			"complete vectors",
		}},
		{"ascii_arm64.s", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/arm64/implementation.cpp:13-16",
			"src/arm64/arm_validate_utf16.cpp:71-91",
			"Independent Go arm64 assembly translation",
			"neither read the tail nor make calls",
		}},
		{"ascii_arm64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/arm64/implementation.cpp:13-16",
			"src/arm64/arm_validate_utf16.cpp:71-91",
			"Hand-authored Go-only tests",
			"independently translated",
		}},
		{"ascii_archsimd_amd64.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"src/haswell/implementation.cpp:278-307",
			"Independently adapted",
			"independent Go adaptation, not a mechanical translation",
		}},
		{"ascii_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/haswell/implementation.cpp:278-307",
			"Independently adapted direct differential coverage",
			"direct archsimd invocation guard",
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
		{"api_contract_test.go", []string{
			upstreamSHA,
			"include/simdutf/error.h:7-124",
			"Narrow Go-only scaffolding",
			"underlying type, unknown values",
			"Go zero values",
			"these are not upstream test vectors",
		}},
		{"api_contract_test.go", []string{
			upstreamSHA,
			"include/simdutf/encoding_types.h:15-24",
			"src/encoding_types.cpp:3-64",
			"Narrow Go-only scaffolding",
			"underlying type, unknown values",
			"truncated inputs, and non-prefix BOMs",
			"these are not upstream test vectors",
		}},
		{"api_contract_test.go", []string{
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
		{"dispatch_archsimd_utf8_stub_amd64.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Go-only dispatch stubs",
			"exists only in amd64 experiment builds",
			"this is not an algorithm translation",
		}},
		{"dispatch_archsimd_count_stub_amd64.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Go-only dispatch stub",
			"independently optional CountUTF8 backend exists only in amd64 experiment",
			"this is not an algorithm translation",
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
			"benchmarks/dataset/emoji.txt",
			"upstream-emoji-utf8",
			"Hand-authored Go-only benchmark scaffolding",
			"adds no product behavior, upstream algorithm vectors",
			"Benchmark function, or benchmark result",
		}},
		{"dispatch_qualification_benchmark_test.go", []string{
			upstreamSHA,
			"docs/porting/benchmark-contract.md",
			"Hand-authored Go-only public-dispatch qualification harness",
			"adds no product behavior or upstream",
			"Corpus setup, integrity checks, and dispatch-provider",
			"outside timed b.Loop bodies",
		}},
		{"ascii_test.go", []string{
			upstreamSHA,
			"docs/porting/benchmark-contract.md",
			"Hand-authored Go-only benchmark registry scaffolding",
			"test-only direct\n// variant slots",
			"defines no product behavior and translates no upstream\n// algorithm",
		}},
		{"ascii_variants_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"Test-only direct benchmark registration",
			"independent Go assembly\n// translation",
		}},
		{"ascii_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"Hand-authored Go-only benchmark registration",
			"adds no benchmark procedure or result",
		}},
		{"ascii_arm64_test.go", []string{
			"Go-only registration of the direct arm64 implementation",
			"defines no\n// product dispatch behavior and translates no upstream algorithm",
		}},
		{"dispatch_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"immutable dispatch selection",
			"not\n// upstream algorithm vectors",
		}},
		{"dispatch_amd64_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"amd64 implementation-table priority",
			"UTF-8\n// and ASCII feature gates",
			"not\n// upstream test vectors",
		}},
		{"dispatch_archsimd_amd64_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"compile-time, CPU-feature, and runtime dispatch gates",
			"not\n// upstream test vectors",
		}},
		{"dispatch_archsimd_stub_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"non-SIMD archsimd provider absence",
			"not\n// upstream test vectors",
		}},
		{"dispatch_arm64_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"arm64 NEON feature gating",
			"UTF-8 and ASCII-family fallback",
			"not\n// upstream test vectors",
		}},
		{"dispatch_generic_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"generic-target scalar dispatch",
			"not\n// upstream test vectors",
		}},
		{"dispatch_test.go", []string{
			upstreamSHA + ":src/implementation.cpp",
			"Hand-authored Go-only tests",
			"exact implementation-table shape",
			"not\n// upstream test vectors",
		}},
		{"ascii_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only direct differential fuzz registry scaffolding",
			"test metadata only and adds no product behavior",
		}},
		{"ascii_variants_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only direct fuzz registration",
			"registers test functions only and adds no product behavior",
		}},
		{"ascii_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only direct fuzz registration",
			"archsimd adaptation",
			"registers test functions only and adds no product behavior",
		}},
		{"ascii_arm64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/arm64/arm_validate_utf16.cpp:71-91",
			"Hand-authored Go-only direct fuzz registration",
			"assembly port",
			"registers test functions only and adds no product behavior",
		}},
		{"ascii_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only family differential fuzz coverage",
			"Scalar functions remain the explicit oracle",
			"adds no product behavior",
		}},
		{"page_guard_unix_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only PROT_NONE no-overread scaffolding",
			"Test-only unsafe",
			"adds no\n// product behavior",
		}},
		{"page_guard_mmap_unix_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only PROT_NONE mmap scaffolding",
			"mapping is test-only and adds no product behavior",
		}},
		{"page_guard_uint16_mmap_unix_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only uint16 PROT_NONE mmap adapter",
			"Test-only unsafe",
			"adds no\n// product behavior",
		}},
		{"page_guard_arm64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/arm64/arm_validate_utf16.cpp:71-91",
			"Hand-authored Go-only deterministic no-overread coverage",
			"invokes direct test functions only and adds no product behavior",
		}},
		{"page_guard_archsimd_amd64_test.go", []string{
			upstreamSHA,
			"src/generic/ascii_validation.h:6-45",
			"src/generic/validate_utf16.h:128-158",
			"Hand-authored Go-only deterministic no-overread coverage",
			"invokes direct test functions only and adds no product behavior",
		}},
		{"benchmark_test.go", []string{
			upstreamSHA,
			"benchmarks/shortbench.cpp:29-35,419-422,493-497,520-526",
			"benchmarks/src/benchmark.cpp:120-127,697-715",
			"docs/porting/benchmark-contract.md",
			"shortbench validate_ascii registration and zero-buffer prefix loop",
			"maps only the main benchmark registration\n// and runner",
			"main-zero-128 is a procedure label, not a corpus ID",
		}},
	}
	requireProvenance(t, expectations...)
}
