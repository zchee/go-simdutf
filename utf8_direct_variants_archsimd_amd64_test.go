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

// Go-only direct benchmark and scalar-differential fuzz registration for the
// tagged lookup4 adaptation pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.

func init() {
	validate := variant[func([]byte) bool]{value: validateUTF8Archsimd, kind: implementationArchsimd, required: cpuAVX2, available: true}
	withErrors := variant[func([]byte) Result]{value: validateUTF8WithErrorsArchsimd, kind: implementationArchsimd, required: cpuAVX2, available: true}
	registerUTF8DirectVariant(utf8DirectVariant{name: "archsimd", validate: validate, withErrors: withErrors})
	registerUTF8FuzzVariant(utf8FuzzVariant{name: "archsimd", validate: validate, withErrors: withErrors})
}
