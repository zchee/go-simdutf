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

//go:build amd64 && !goexperiment.simd

package simdutf

// Go-only dispatch stubs based on the provider-availability semantics in
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp.
// The UTF-8 archsimd implementation exists only in amd64 experiment builds;
// this is not an algorithm translation.

func archsimdValidateUTF8() func([]byte) bool {
	return nil
}

func archsimdValidateUTF8WithErrors() func([]byte) Result {
	return nil
}
