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

func archsimdConvertUTF32ToLatin1() func([]uint32, []byte) int {
	return convertUTF32ToLatin1Archsimd
}

func archsimdConvertUTF32ToLatin1WithErrors() func([]uint32, []byte) Result {
	return convertUTF32ToLatin1WithErrorsArchsimd
}

func archsimdConvertValidUTF32ToLatin1() func([]uint32, []byte) int {
	return convertValidUTF32ToLatin1Archsimd
}

func archsimdConvertUTF32ToUTF8() func([]uint32, []byte) int {
	return convertUTF32ToUTF8Archsimd
}

func archsimdConvertUTF32ToUTF8WithErrors() func([]uint32, []byte) Result {
	return convertUTF32ToUTF8WithErrorsArchsimd
}

func archsimdConvertValidUTF32ToUTF8() func([]uint32, []byte) int {
	return convertValidUTF32ToUTF8Archsimd
}

func archsimdConvertUTF32ToUTF16LE() func([]uint32, []uint16) int {
	return convertUTF32ToUTF16LEArchsimd
}

func archsimdConvertUTF32ToUTF16BE() func([]uint32, []uint16) int {
	return convertUTF32ToUTF16BEArchsimd
}

func archsimdConvertUTF32ToUTF16LEWithErrors() func([]uint32, []uint16) Result {
	return convertUTF32ToUTF16LEWithErrorsArchsimd
}

func archsimdConvertUTF32ToUTF16BEWithErrors() func([]uint32, []uint16) Result {
	return convertUTF32ToUTF16BEWithErrorsArchsimd
}

func archsimdConvertValidUTF32ToUTF16LE() func([]uint32, []uint16) int {
	return convertValidUTF32ToUTF16LEArchsimd
}

func archsimdConvertValidUTF32ToUTF16BE() func([]uint32, []uint16) int {
	return convertValidUTF32ToUTF16BEArchsimd
}

func archsimdUTF8LengthFromUTF32() func([]uint32) int {
	return utf8LengthFromUTF32Archsimd
}

func archsimdUTF16LengthFromUTF32() func([]uint32) int {
	return utf16LengthFromUTF32Archsimd
}
