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

// Hand-authored Go-only direct fuzz registration for the amd64 lookup4
// assembly ports pinned to simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f.
// It registers test functions only and adds no product behavior.

func init() {
	registerUTF8FuzzVariant(utf8FuzzVariant{
		name:       "westmere",
		validate:   variant[func([]byte) bool]{value: validateUTF8Westmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
		withErrors: variant[func([]byte) Result]{value: validateUTF8WithErrorsWestmere, kind: implementationWestmere, required: cpuSSSE3, available: true},
	})
	registerUTF8FuzzVariant(utf8FuzzVariant{
		name:       "haswell",
		validate:   variant[func([]byte) bool]{value: validateUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true},
		withErrors: variant[func([]byte) Result]{value: validateUTF8WithErrorsHaswell, kind: implementationHaswell, required: cpuAVX2, available: true},
	})
}
