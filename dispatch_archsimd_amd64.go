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
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; this is not an
// algorithm translation.

func archsimdAVX2Available() bool {
	return archsimd.X86.AVX2()
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
