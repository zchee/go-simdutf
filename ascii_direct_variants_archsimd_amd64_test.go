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
// adaptation of simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f
// (tree eb5429bb160dfdf1a7d208f0184d3379940e69ee),
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
