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

// Portions Copyright 2021 The simdutf Authors.
//go:build amd64

package simdutf

// Independently translated to Go assembly from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/westmere/sse_validate_utf16.cpp, src/haswell/avx2_validate_utf16.cpp,
// src/westmere/implementation.cpp:138, and src/haswell/implementation.cpp:142.
// The vector kernels consume only complete surrogate-free (UTF-16) or ASCII
// (UTF-32) blocks. Go finishes at the first exceptional block so Result keeps
// the scalar oracle's exact first-error semantics.

//go:noescape
func utf16NoSurrogateWestmere(input []uint16, bigEndian bool) int

//go:noescape
func utf16NoSurrogateHaswell(input []uint16, bigEndian bool) int

//go:noescape
func utf32ASCIIPrefixWestmere(input []uint32) int

//go:noescape
func utf32ASCIIPrefixHaswell(input []uint32) int

//go:noescape
func utf16CopyWestmere(input, dst []uint16, n int)

//go:noescape
func utf16CopyHaswell(input, dst []uint16, n int)

func validateUTF16LEWestmere(input []uint16) bool {
	if len(input) < 8 {
		return validateUTF16LEScalar(input)
	}
	return validateUTF16AMD64(input, false, utf16NoSurrogateWestmere)
}

func validateUTF16BEWestmere(input []uint16) bool {
	if len(input) < 8 {
		return validateUTF16BEScalar(input)
	}
	return validateUTF16AMD64(input, true, utf16NoSurrogateWestmere)
}

func validateUTF16LEHaswell(input []uint16) bool {
	if len(input) < 16 {
		return validateUTF16LEScalar(input)
	}
	return validateUTF16AMD64(input, false, utf16NoSurrogateHaswell)
}

func validateUTF16BEHaswell(input []uint16) bool {
	if len(input) < 16 {
		return validateUTF16BEScalar(input)
	}
	return validateUTF16AMD64(input, true, utf16NoSurrogateHaswell)
}

func validateUTF16LEWithErrorsWestmere(input []uint16) Result {
	if len(input) < 8 {
		return validateUTF16LEWithErrorsScalar(input)
	}
	return validateUTF16WithErrorsAMD64(input, false, utf16NoSurrogateWestmere)
}

func validateUTF16BEWithErrorsWestmere(input []uint16) Result {
	if len(input) < 8 {
		return validateUTF16BEWithErrorsScalar(input)
	}
	return validateUTF16WithErrorsAMD64(input, true, utf16NoSurrogateWestmere)
}

func validateUTF16LEWithErrorsHaswell(input []uint16) Result {
	if len(input) < 16 {
		return validateUTF16LEWithErrorsScalar(input)
	}
	return validateUTF16WithErrorsAMD64(input, false, utf16NoSurrogateHaswell)
}

func validateUTF16BEWithErrorsHaswell(input []uint16) Result {
	if len(input) < 16 {
		return validateUTF16BEWithErrorsScalar(input)
	}
	return validateUTF16WithErrorsAMD64(input, true, utf16NoSurrogateHaswell)
}

func validateUTF16AMD64(input []uint16, bigEndian bool, prefix func([]uint16, bool) int) bool {
	n := prefix(input, bigEndian)
	return validateUTF16Scalar(input[n:], !bigEndian)
}

func validateUTF16WithErrorsAMD64(input []uint16, bigEndian bool, prefix func([]uint16, bool) int) Result {
	n := prefix(input, bigEndian)
	r := validateUTF16WithErrorsScalar(input[n:], !bigEndian)
	r.Count += n
	return r
}

func toWellFormedUTF16LEWestmere(input, dst []uint16) {
	if len(input) < 8 {
		toWellFormedUTF16LEScalar(input, dst)
		return
	}
	toWellFormedUTF16AMD64(input, dst, false, utf16NoSurrogateWestmere, utf16CopyWestmere)
}

func toWellFormedUTF16BEWestmere(input, dst []uint16) {
	if len(input) < 8 {
		toWellFormedUTF16BEScalar(input, dst)
		return
	}
	toWellFormedUTF16AMD64(input, dst, true, utf16NoSurrogateWestmere, utf16CopyWestmere)
}

func toWellFormedUTF16LEHaswell(input, dst []uint16) {
	if len(input) < 16 {
		toWellFormedUTF16LEScalar(input, dst)
		return
	}
	toWellFormedUTF16AMD64(input, dst, false, utf16NoSurrogateHaswell, utf16CopyHaswell)
}

func toWellFormedUTF16BEHaswell(input, dst []uint16) {
	if len(input) < 16 {
		toWellFormedUTF16BEScalar(input, dst)
		return
	}
	toWellFormedUTF16AMD64(input, dst, true, utf16NoSurrogateHaswell, utf16CopyHaswell)
}

func toWellFormedUTF16AMD64(input, dst []uint16, bigEndian bool, prefix func([]uint16, bool) int, copyPrefix func([]uint16, []uint16, int)) {
	if len(dst) < len(input) {
		panic("simdutf: UTF-16 destination too short")
	}
	n := prefix(input, bigEndian)
	// The vector copy is safe for the supported identical-slice case and leaves
	// the first surrogate for the scalar state machine, including a block edge.
	copyPrefix(input, dst, n)
	toWellFormedUTF16Scalar(input[n:], dst[n:], !bigEndian)
}

func validateUTF32Westmere(input []uint32) bool {
	if len(input) < 4 {
		return validateUTF32Scalar(input)
	}
	return validateUTF32AMD64(input, utf32ASCIIPrefixWestmere)
}

func validateUTF32Haswell(input []uint32) bool {
	if len(input) < 8 {
		return validateUTF32Scalar(input)
	}
	return validateUTF32AMD64(input, utf32ASCIIPrefixHaswell)
}

func validateUTF32WithErrorsWestmere(input []uint32) Result {
	if len(input) < 4 {
		return validateUTF32WithErrorsScalar(input)
	}
	return validateUTF32WithErrorsAMD64(input, utf32ASCIIPrefixWestmere)
}

func validateUTF32WithErrorsHaswell(input []uint32) Result {
	if len(input) < 8 {
		return validateUTF32WithErrorsScalar(input)
	}
	return validateUTF32WithErrorsAMD64(input, utf32ASCIIPrefixHaswell)
}

func validateUTF32AMD64(input []uint32, prefix func([]uint32) int) bool {
	n := prefix(input)
	return validateUTF32Scalar(input[n:])
}

func validateUTF32WithErrorsAMD64(input []uint32, prefix func([]uint32) int) Result {
	n := prefix(input)
	r := validateUTF32WithErrorsScalar(input[n:])
	r.Count += n
	return r
}
