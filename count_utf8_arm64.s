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

//go:build arm64

// Independent Go arm64 assembly translation of count_code_points in
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// src/generic/utf8.h:8-17, src/arm64/implementation.cpp:1113-1117, and
// src/simdutf/arm64/simd.h:420-529. Each iteration loads exactly one complete
// 64-byte block and counts its UTF-8 continuation-byte masks with NEON.

#include "textflag.h"

// func countUTF8BlocksNEON(input []byte) int
TEXT ·countUTF8BlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-64, R1, R1
	MOVD R1, R2
	CBZ  R1, count_utf8_neon_done

	// Go 1.26's Plan 9 arm64 assembler exposes VCMEQ but no signed integer
	// VCMGT mnemonic. A continuation byte has the exact unsigned prefix
	// 10xxxxxx, so VUSHR $6 followed by VCMEQ 2 produces the complement of the
	// pinned signed comparison input.gt(-65). Subtracting that continuation
	// population from the block size yields upstream's non-continuation count;
	// this performs no validation.
	MOVD $2, R3
	VMOV R3, V0.B16
	MOVD $1, R3
	VMOV R3, V5.B16
	VEOR V8.B16, V8.B16, V8.B16

count_utf8_neon_loop:
	VLD1.P 64(R0), [V1.B16, V2.B16, V3.B16, V4.B16]
	VUSHR  $6, V1.B16, V1.B16
	VUSHR  $6, V2.B16, V2.B16
	VUSHR  $6, V3.B16, V3.B16
	VUSHR  $6, V4.B16, V4.B16
	VCMEQ  V0.B16, V1.B16, V1.B16
	VCMEQ  V0.B16, V2.B16, V2.B16
	VCMEQ  V0.B16, V3.B16, V3.B16
	VCMEQ  V0.B16, V4.B16, V4.B16
	VAND   V5.B16, V1.B16, V1.B16
	VAND   V5.B16, V2.B16, V2.B16
	VAND   V5.B16, V3.B16, V3.B16
	VAND   V5.B16, V4.B16, V4.B16
	VADDP  V2.B16, V1.B16, V6.B16
	VADDP  V4.B16, V3.B16, V7.B16
	VADDP  V7.B16, V6.B16, V6.B16

	// Reduce every block before adding its maximum value of 64 to the 64-bit
	// accumulator; byte lanes therefore cannot overflow across iterations.
	VUADDLV V6.B16, V7
	VADD    V7, V8
	SUB     $64, R1
	CBNZ    R1, count_utf8_neon_loop

	VMOV V8.D[0], R3
	SUB  R3, R2, R2

count_utf8_neon_done:
	MOVD R2, count+24(FP)
	RET
