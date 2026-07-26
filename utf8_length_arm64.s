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

// Independent Go arm64 assembly translation of utf16_length_from_utf8 in
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee): src/generic/utf8.h:72-86,
// src/arm64/implementation.cpp:1178-1181, and
// src/simdutf/arm64/simd.h:420-529. Each iteration loads exactly one complete
// 64-byte block, counts non-continuations, and adds the four-byte-lead count.

#include "textflag.h"

// func utf16LengthFromUTF8BlocksNEON(input []byte) int
TEXT ·utf16LengthFromUTF8BlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD	input_base+0(FP), R0
	MOVD	input_len+8(FP), R1
	AND	$-64, R1, R1
	MOVD	$0, R2
	CBZ	R1, utf16_length_neon_done

	// Plan 9 has no signed integer VCMGT mnemonic. For every byte value,
	// (byte >> 6) == 2 is exactly the continuation class 0x80..0xbf, the
	// complement of upstream's signed input >= -64 predicate. Likewise,
	// (byte >> 4) == 15 is exactly upstream's unsigned input >= 0xf0 predicate.
	// Converting both all-ones masks to byte values of one lets each block be
	// reduced without byte-lane accumulation across iterations.
	MOVD	$2, R3
	VMOV	R3, V0.B16
	MOVD	$15, R3
	VMOV	R3, V14.B16
	MOVD	$1, R3
	VMOV	R3, V15.B16

utf16_length_neon_loop:
	VLD1.P	64(R0), [V1.B16, V2.B16, V3.B16, V4.B16]

	VUSHR	$6, V1.B16, V5.B16
	VUSHR	$6, V2.B16, V6.B16
	VUSHR	$6, V3.B16, V7.B16
	VUSHR	$6, V4.B16, V8.B16
	VCMEQ	V0.B16, V5.B16, V5.B16
	VCMEQ	V0.B16, V6.B16, V6.B16
	VCMEQ	V0.B16, V7.B16, V7.B16
	VCMEQ	V0.B16, V8.B16, V8.B16
	VAND	V15.B16, V5.B16, V5.B16
	VAND	V15.B16, V6.B16, V6.B16
	VAND	V15.B16, V7.B16, V7.B16
	VAND	V15.B16, V8.B16, V8.B16
	VADDP	V6.B16, V5.B16, V9.B16
	VADDP	V8.B16, V7.B16, V10.B16
	VADDP	V10.B16, V9.B16, V9.B16
	VUADDLV	V9.B16, V11
	VMOV	V11.D[0], R3

	VUSHR	$4, V1.B16, V1.B16
	VUSHR	$4, V2.B16, V2.B16
	VUSHR	$4, V3.B16, V3.B16
	VUSHR	$4, V4.B16, V4.B16
	VCMEQ	V14.B16, V1.B16, V1.B16
	VCMEQ	V14.B16, V2.B16, V2.B16
	VCMEQ	V14.B16, V3.B16, V3.B16
	VCMEQ	V14.B16, V4.B16, V4.B16
	VAND	V15.B16, V1.B16, V1.B16
	VAND	V15.B16, V2.B16, V2.B16
	VAND	V15.B16, V3.B16, V3.B16
	VAND	V15.B16, V4.B16, V4.B16
	VADDP	V2.B16, V1.B16, V9.B16
	VADDP	V4.B16, V3.B16, V10.B16
	VADDP	V10.B16, V9.B16, V9.B16
	VUADDLV	V9.B16, V11
	VMOV	V11.D[0], R4

	MOVD	$64, R5
	SUB	R3, R5, R5
	ADD	R4, R5, R5
	ADD	R5, R2, R2
	SUB	$64, R1
	CBNZ	R1, utf16_length_neon_loop

utf16_length_neon_done:
	MOVD	R2, length+24(FP)
	RET
