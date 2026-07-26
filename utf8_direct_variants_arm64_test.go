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

//go:build arm64

package simdutf

// Go-only registration of the direct arm64 lookup4 implementation pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f. It defines no
// product dispatch behavior and translates no additional upstream algorithm.

func init() {
	registerUTF8DirectVariant(utf8DirectVariant{
		name: "neon",
		validate: variant[func([]byte) bool]{
			value: validateUTF8NEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
		withErrors: variant[func([]byte) Result]{
			value: validateUTF8WithErrorsNEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
	})
}
