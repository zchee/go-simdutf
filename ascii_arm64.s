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

// Independent Go arm64 assembly translation of the 64-byte ASCII reduction in
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/generic/ascii_validation.h:6-45 and src/arm64/implementation.cpp:13-16,
// and the 16-code-unit kernels in src/arm64/arm_validate_utf16.cpp:71-91.
// The ABI0 wrappers return the start of the first failing complete block, or
// the complete-block prefix length. They neither read the tail nor make calls.

#include "textflag.h"

// func validateASCIIPrefixNEON(input []byte) int
TEXT ·validateASCIIPrefixNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD	input_base+0(FP), R0
	MOVD	input_len+8(FP), R1
	AND	$-64, R1, R2
	MOVD	$0, R3

ascii_loop:
	CMP	R2, R3
	BEQ	ascii_done
	VLD1	(R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	VORR	V1.B16, V0.B16, V0.B16
	VORR	V3.B16, V2.B16, V2.B16
	VORR	V2.B16, V0.B16, V0.B16
	VMOV	V0.D[0], R4
	VMOV	V0.D[1], R5
	ORR	R5, R4, R4
	TST	$0x8080808080808080, R4
	BNE	ascii_return
	ADD	$64, R0
	ADD	$64, R3
	B	ascii_loop

ascii_done:
	MOVD	R2, R3
ascii_return:
	MOVD	R3, ret+24(FP)
	RET

// func validateUTF16LEASCIIPrefixNEON(input []uint16) int
TEXT ·validateUTF16LEASCIIPrefixNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD	input_base+0(FP), R0
	MOVD	input_len+8(FP), R1
	AND	$-16, R1, R2
	MOVD	$0xff80, R6
	VDUP	R6, V3.H8
	MOVD	$0, R3

utf16le_loop:
	CMP	R2, R3
	BEQ	utf16le_done
	VLD1	(R0), [V0.H8, V1.H8]
	VORR	V1.B16, V0.B16, V0.B16
	VAND	V3.B16, V0.B16, V0.B16
	VMOV	V0.D[0], R4
	VMOV	V0.D[1], R5
	ORR	R5, R4, R4
	CBNZ	R4, utf16le_return
	ADD	$32, R0
	ADD	$16, R3
	B	utf16le_loop

utf16le_done:
	MOVD	R2, R3
utf16le_return:
	MOVD	R3, ret+24(FP)
	RET

// func validateUTF16BEASCIIPrefixNEON(input []uint16) int
TEXT ·validateUTF16BEASCIIPrefixNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD	input_base+0(FP), R0
	MOVD	input_len+8(FP), R1
	AND	$-16, R1, R2
	MOVD	$0x80ff, R6
	VDUP	R6, V3.H8
	MOVD	$0, R3

utf16be_loop:
	CMP	R2, R3
	BEQ	utf16be_done
	VLD1	(R0), [V0.H8, V1.H8]
	VORR	V1.B16, V0.B16, V0.B16
	VAND	V3.B16, V0.B16, V0.B16
	VMOV	V0.D[0], R4
	VMOV	V0.D[1], R5
	ORR	R5, R4, R4
	CBNZ	R4, utf16be_return
	ADD	$32, R0
	ADD	$16, R3
	B	utf16be_loop

utf16be_done:
	MOVD	R2, R3
utf16be_return:
	MOVD	R3, ret+24(FP)
	RET
