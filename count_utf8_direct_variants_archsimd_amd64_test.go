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

// Go-only direct benchmark and differential-fuzz registration for the tagged
// CountUTF8 adaptation pinned to
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de. It changes no
// frozen benchmark name, corpus, or setup.
func init() {
	candidate := variant[func([]byte) int]{
		value: countUTF8Archsimd, kind: implementationArchsimd,
		required: cpuAVX2, available: true,
	}
	registerCountUTF8DirectVariant(countUTF8DirectVariant{name: "archsimd", variant: candidate})
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{name: "archsimd", variant: candidate})
}
