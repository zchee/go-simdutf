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

// Hand-authored Go-only direct fuzz registration for the separate Westmere
// and Haswell count_code_points_bytemask assembly ports pinned to
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de.
func init() {
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{
		name: "westmere",
		variant: variant[func([]byte) int]{
			value: countUTF8Westmere, kind: implementationWestmere, available: true,
		},
	})
	registerCountUTF8FuzzVariant(countUTF8FuzzVariant{
		name: "haswell",
		variant: variant[func([]byte) int]{
			value: countUTF8Haswell, kind: implementationHaswell, required: cpuAVX2, available: true,
		},
	})
}
