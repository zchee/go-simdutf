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

//go:build amd64 && goexperiment.simd

package simdutf

// Hand-authored Go-only benchmark registration for the independent archsimd
// adaptation of simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de
// (tree 4cbac4c5d1ce0d7f98cc35360d53725433f12811),
// src/generic/ascii_validation.h:6-45. It uses the test-only registry defined
// by ascii_direct_variants_test.go and adds no benchmark procedure or result.

func init() {
	registerASCIIDirectBenchmarkVariants(
		"archsimd",
		variant[func([]byte) bool]{
			value:     validateASCIIArchsimd,
			kind:      implementationArchsimd,
			required:  cpuAVX2,
			available: true,
		},
		variant[func([]byte) Result]{
			value:     validateASCIIWithErrorsArchsimd,
			kind:      implementationArchsimd,
			required:  cpuAVX2,
			available: true,
		},
	)
}
