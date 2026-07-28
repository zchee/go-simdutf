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

import "simd/archsimd"

// Go-only dispatch glue based on the first-supported priority semantics in
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; this is not an
// algorithm translation.

func archsimdAVX2Available() bool {
	return archsimd.X86.AVX2()
}

func archsimdCountUTF8() func([]byte) int {
	return countUTF8Archsimd
}

func archsimdValidateASCII() func([]byte) bool {
	return validateASCIIArchsimd
}

func archsimdValidateASCIIWithErrors() func([]byte) Result {
	return validateASCIIWithErrorsArchsimd
}

func archsimdValidateUTF16LEAsASCII() func([]uint16) bool {
	return validateUTF16LEAsASCIIArchsimd
}

func archsimdValidateUTF16BEAsASCII() func([]uint16) bool {
	return validateUTF16BEAsASCIIArchsimd
}

func archsimdValidateUTF8() func([]byte) bool {
	// Keep the direct variant available for differential testing, but do not
	// select it in production while the Go 1.26.5 lowering significantly
	// regresses both short full-block and pinned bulk workloads.
	return nil
}

func archsimdValidateUTF8WithErrors() func([]byte) Result {
	return nil
}
