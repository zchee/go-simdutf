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

// Hand-authored Go-only direct fuzz registration for the arm64 assembly port
// pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/utf8.h:8-17 and src/arm64/implementation.cpp:1113-1117.
func init() {
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{
		name: "neon",
		variant: variant[func([]byte) int]{
			value: countUTF8NEON, kind: implementationNEON,
			required: cpuNEON, available: true,
		},
	})
}
