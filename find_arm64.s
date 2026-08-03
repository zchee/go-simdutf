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

#include "textflag.h"

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_find.cpp util_find. Returns the first match index, or len on miss.
// Go 1.26 Plan 9 arm64 exposes VCMEQ/VORR/VDUP/VLD1 but no VSHRN, so a hit window
// is refined by scanning the matching 16-byte / 8-uint16 chunk.

// func findNEON(input []byte, value byte) int
TEXT ·findNEON(SB), NOSPLIT|NOFRAME, $0-40
	MOVD  input_base+0(FP), R0
	MOVD  input_len+8(FP), R9
	MOVBU value+24(FP), R2
	MOVD  R0, R8
	ADD   R0, R9, R3
	CBZ   R9, find_byte_miss
	VDUP  R2, V7.B16

	AND  $15, R0, R4
	CBZ  R4, find_byte_aligned
	MOVD $16, R5
	SUB  R4, R5
	SUB  R0, R3, R6
	CMP  R5, R6
	BGE  find_byte_align_ok
	MOVD R6, R5

find_byte_align_ok:
	MOVD $0, R4

find_byte_align_loop:
	CMP   R5, R4
	BEQ   find_byte_align_done
	MOVBU (R0)(R4), R6
	CMP   R2, R6
	BEQ   find_byte_align_hit
	ADD   $1, R4
	B     find_byte_align_loop

find_byte_align_hit:
	SUB R8, R0, R7
	ADD R4, R7
	B   find_byte_return

find_byte_align_done:
	ADD R5, R0

find_byte_aligned:
find_byte_wide:
	SUB   R0, R3, R4
	CMP   $64, R4
	BLT   find_byte_step
	VLD1  (R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	VCMEQ V7.B16, V0.B16, V0.B16
	VCMEQ V7.B16, V1.B16, V1.B16
	VCMEQ V7.B16, V2.B16, V2.B16
	VCMEQ V7.B16, V3.B16, V3.B16
	VORR  V1.B16, V0.B16, V4.B16
	VORR  V3.B16, V2.B16, V5.B16
	VORR  V5.B16, V4.B16, V4.B16
	VMOV  V4.D[0], R4
	VMOV  V4.D[1], R5
	ORR   R5, R4, R4
	CBNZ  R4, find_byte_wide_hit
	ADD   $64, R0
	B     find_byte_wide

find_byte_wide_hit:
	VMOV V0.D[0], R4
	VMOV V0.D[1], R5
	ORR  R5, R4, R4
	MOVD $0, R5
	CBNZ R4, find_byte_refine
	VMOV V1.D[0], R4
	VMOV V1.D[1], R5
	ORR  R5, R4, R4
	MOVD $16, R5
	CBNZ R4, find_byte_refine
	VMOV V2.D[0], R4
	VMOV V2.D[1], R5
	ORR  R5, R4, R4
	MOVD $32, R5
	CBNZ R4, find_byte_refine
	MOVD $48, R5

find_byte_refine:
	ADD  R5, R0, R4
	MOVD $0, R6

find_byte_refine_loop:
	MOVBU (R4)(R6), R7
	CMP   R2, R7
	BEQ   find_byte_refine_found
	ADD   $1, R6
	CMP   $16, R6
	BLT   find_byte_refine_loop
	ADD   $64, R0
	B     find_byte_wide

find_byte_refine_found:
	SUB R8, R4, R7
	ADD R6, R7
	B   find_byte_return

find_byte_step:
	SUB   R0, R3, R4
	CMP   $16, R4
	BLT   find_byte_tail
	VLD1  (R0), [V0.B16]
	VCMEQ V7.B16, V0.B16, V0.B16
	VMOV  V0.D[0], R4
	VMOV  V0.D[1], R5
	ORR   R5, R4, R4
	MOVD  $0, R5
	CBNZ  R4, find_byte_refine_step
	ADD   $16, R0
	B     find_byte_step

find_byte_refine_step:
	ADD  R5, R0, R4
	MOVD $0, R6

find_byte_refine_step_loop:
	MOVBU (R4)(R6), R7
	CMP   R2, R7
	BEQ   find_byte_refine_found
	ADD   $1, R6
	CMP   $16, R6
	BLT   find_byte_refine_step_loop
	ADD   $16, R0
	B     find_byte_step

find_byte_tail:
	CMP   R3, R0
	BEQ   find_byte_miss
	MOVBU (R0), R4
	CMP   R2, R4
	BEQ   find_byte_tail_hit
	ADD   $1, R0
	B     find_byte_tail

find_byte_tail_hit:
	SUB R8, R0, R7
	B   find_byte_return

find_byte_miss:
	MOVD R9, ret+32(FP)
	RET

find_byte_return:
	MOVD R7, ret+32(FP)
	RET

// func findUTF16NEON(input []uint16, value uint16) int
TEXT ·findUTF16NEON(SB), NOSPLIT|NOFRAME, $0-40
	MOVD  input_base+0(FP), R0
	MOVD  input_len+8(FP), R9
	MOVHU value+24(FP), R2
	MOVD  R0, R8

	// end = base + len*2
	ADD  R9, R9, R3
	ADD  R0, R3
	CBZ  R9, find_u16_miss
	VDUP R2, V7.H8

	AND  $15, R0, R4
	CBZ  R4, find_u16_aligned
	AND  $1, R4, R5
	CBNZ R5, find_u16_aligned
	MOVD $16, R5
	SUB  R4, R5
	LSR  $1, R5
	SUB  R0, R3, R6
	LSR  $1, R6
	CMP  R5, R6
	BGE  find_u16_align_ok
	MOVD R6, R5

find_u16_align_ok:
	MOVD $0, R4

find_u16_align_loop:
	CMP   R5, R4
	BEQ   find_u16_align_done
	MOVHU (R0)(R4<<1), R6
	CMPW  R2, R6
	BEQ   find_u16_align_hit
	ADD   $1, R4
	B     find_u16_align_loop

find_u16_align_hit:
	SUB R8, R0, R7
	LSR $1, R7
	ADD R4, R7
	B   find_u16_return

find_u16_align_done:
	ADD R5<<1, R0

find_u16_aligned:
find_u16_wide:
	SUB   R0, R3, R4
	CMP   $64, R4
	BLT   find_u16_step
	VLD1  (R0), [V0.H8, V1.H8, V2.H8, V3.H8]
	VCMEQ V7.H8, V0.H8, V0.H8
	VCMEQ V7.H8, V1.H8, V1.H8
	VCMEQ V7.H8, V2.H8, V2.H8
	VCMEQ V7.H8, V3.H8, V3.H8
	VMOV  V0.D[0], R4
	VMOV  V0.D[1], R5
	ORR   R5, R4, R4
	MOVD  $0, R5
	CBNZ  R4, find_u16_refine
	VMOV  V1.D[0], R4
	VMOV  V1.D[1], R5
	ORR   R5, R4, R4
	MOVD  $16, R5
	CBNZ  R4, find_u16_refine
	VMOV  V2.D[0], R4
	VMOV  V2.D[1], R5
	ORR   R5, R4, R4
	MOVD  $32, R5
	CBNZ  R4, find_u16_refine
	VMOV  V3.D[0], R4
	VMOV  V3.D[1], R5
	ORR   R5, R4, R4
	MOVD  $48, R5
	CBNZ  R4, find_u16_refine
	ADD   $64, R0
	B     find_u16_wide

find_u16_refine:
	ADD  R5, R0, R4
	MOVD $0, R6

find_u16_refine_loop:
	MOVHU (R4)(R6<<1), R7
	CMPW  R2, R7
	BEQ   find_u16_refine_found
	ADD   $1, R6
	CMP   $8, R6
	BLT   find_u16_refine_loop
	ADD   $64, R0
	B     find_u16_wide

find_u16_refine_found:
	SUB R8, R4, R7
	LSR $1, R7
	ADD R6, R7
	B   find_u16_return

find_u16_step:
	SUB   R0, R3, R4
	CMP   $16, R4
	BLT   find_u16_tail
	VLD1  (R0), [V0.H8]
	VCMEQ V7.H8, V0.H8, V0.H8
	VMOV  V0.D[0], R4
	VMOV  V0.D[1], R5
	ORR   R5, R4, R4
	MOVD  $0, R5
	CBNZ  R4, find_u16_refine_step
	ADD   $16, R0
	B     find_u16_step

find_u16_refine_step:
	ADD  R5, R0, R4
	MOVD $0, R6

find_u16_refine_step_loop:
	MOVHU (R4)(R6<<1), R7
	CMPW  R2, R7
	BEQ   find_u16_refine_found
	ADD   $1, R6
	CMP   $8, R6
	BLT   find_u16_refine_step_loop
	ADD   $16, R0
	B     find_u16_step

find_u16_tail:
	CMP   R3, R0
	BEQ   find_u16_miss
	MOVHU (R0), R4
	CMPW  R2, R4
	BEQ   find_u16_tail_hit
	ADD   $2, R0
	B     find_u16_tail

find_u16_tail_hit:
	SUB R8, R0, R7
	LSR $1, R7
	B   find_u16_return

find_u16_miss:
	MOVD R9, ret+32(FP)
	RET

find_u16_return:
	MOVD R7, ret+32(FP)
	RET
