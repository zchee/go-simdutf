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

func archsimdValidateUTF16LE() func([]uint16) bool { return validateUTF16LEArchsimd }
func archsimdValidateUTF16BE() func([]uint16) bool { return validateUTF16BEArchsimd }
func archsimdValidateUTF16LEWithErrors() func([]uint16) Result {
	return validateUTF16LEWithErrorsArchsimd
}
func archsimdValidateUTF16BEWithErrors() func([]uint16) Result {
	return validateUTF16BEWithErrorsArchsimd
}
func archsimdToWellFormedUTF16LE() func([]uint16, []uint16) { return toWellFormedUTF16LEArchsimd }
func archsimdToWellFormedUTF16BE() func([]uint16, []uint16) { return toWellFormedUTF16BEArchsimd }
func archsimdValidateUTF32() func([]uint32) bool            { return validateUTF32Archsimd }
func archsimdValidateUTF32WithErrors() func([]uint32) Result {
	return validateUTF32WithErrorsArchsimd
}
func archsimdUTF8LengthFromLatin1() func([]byte) int { return utf8LengthFromLatin1Archsimd }
func archsimdConvertLatin1ToUTF8() func([]byte, []byte) int {
	return convertLatin1ToUTF8Archsimd
}
func archsimdConvertLatin1ToUTF16LE() func([]byte, []uint16) int {
	return convertLatin1ToUTF16LEArchsimd
}
func archsimdConvertLatin1ToUTF16BE() func([]byte, []uint16) int {
	return convertLatin1ToUTF16BEArchsimd
}
func archsimdConvertLatin1ToUTF32() func([]byte, []uint32) int {
	return convertLatin1ToUTF32Archsimd
}

func archsimdConvertUTF8ToLatin1() func([]byte, []byte) int { return convertUTF8ToLatin1Archsimd }
func archsimdConvertUTF8ToLatin1WithErrors() func([]byte, []byte) Result {
	return convertUTF8ToLatin1WithErrorsArchsimd
}
func archsimdConvertValidUTF8ToLatin1() func([]byte, []byte) int {
	return convertValidUTF8ToLatin1Archsimd
}
func archsimdConvertUTF8ToUTF16LE() func([]byte, []uint16) int { return convertUTF8ToUTF16LEArchsimd }
func archsimdConvertUTF8ToUTF16BE() func([]byte, []uint16) int { return convertUTF8ToUTF16BEArchsimd }
func archsimdConvertUTF8ToUTF16LEWithErrors() func([]byte, []uint16) Result {
	return convertUTF8ToUTF16LEWithErrorsArchsimd
}
func archsimdConvertUTF8ToUTF16BEWithErrors() func([]byte, []uint16) Result {
	return convertUTF8ToUTF16BEWithErrorsArchsimd
}
func archsimdConvertValidUTF8ToUTF16LE() func([]byte, []uint16) int {
	return convertValidUTF8ToUTF16LEArchsimd
}
func archsimdConvertValidUTF8ToUTF16BE() func([]byte, []uint16) int {
	return convertValidUTF8ToUTF16BEArchsimd
}
func archsimdConvertUTF8ToUTF32() func([]byte, []uint32) int { return convertUTF8ToUTF32Archsimd }
func archsimdConvertUTF8ToUTF32WithErrors() func([]byte, []uint32) Result {
	return convertUTF8ToUTF32WithErrorsArchsimd
}
func archsimdConvertValidUTF8ToUTF32() func([]byte, []uint32) int {
	return convertValidUTF8ToUTF32Archsimd
}
