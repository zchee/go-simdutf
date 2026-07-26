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

// Go-only registration of the direct arm64 implementation. It defines no
// product dispatch behavior and translates no upstream algorithm.
func init() {
	registerASCIIDirectBenchmarkVariants(
		"neon",
		variant[func([]byte) bool]{
			value:     validateASCIINEON,
			kind:      implementationNEON,
			required:  cpuNEON,
			available: true,
		},
		variant[func([]byte) Result]{
			value:     validateASCIIWithErrorsNEON,
			kind:      implementationNEON,
			required:  cpuNEON,
			available: true,
		},
	)
}
