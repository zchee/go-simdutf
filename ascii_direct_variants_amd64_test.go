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

//go:build amd64

package simdutf

// Test-only direct benchmark registration for the independent Go assembly
// translation pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f),
// src/generic/ascii_validation.h:6-45.

func init() {
	registerASCIIDirectBenchmarkVariants(
		"westmere",
		variant[func([]byte) bool]{
			value:     validateASCIIWestmere,
			kind:      implementationWestmere,
			available: true,
		},
		variant[func([]byte) Result]{
			value:     validateASCIIWithErrorsWestmere,
			kind:      implementationWestmere,
			available: true,
		},
	)
	registerASCIIDirectBenchmarkVariants(
		"haswell",
		variant[func([]byte) bool]{
			value:     validateASCIIHaswell,
			kind:      implementationHaswell,
			required:  cpuAVX2,
			available: true,
		},
		variant[func([]byte) Result]{
			value:     validateASCIIWithErrorsHaswell,
			kind:      implementationHaswell,
			required:  cpuAVX2,
			available: true,
		},
	)
}
