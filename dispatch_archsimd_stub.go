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

//go:build !amd64 || !goexperiment.simd

package simdutf

// Go-only dispatch glue based on the first-supported priority semantics in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; this is not an
// algorithm translation.

func archsimdAVX2Available() bool {
	return false
}

func archsimdValidateASCII() func([]byte) bool {
	return nil
}

func archsimdValidateASCIIWithErrors() func([]byte) Result {
	return nil
}

func archsimdValidateUTF16LEAsASCII() func([]uint16) bool {
	return nil
}

func archsimdValidateUTF16BEAsASCII() func([]uint16) bool {
	return nil
}
