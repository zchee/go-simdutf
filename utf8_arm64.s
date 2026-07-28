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

// Independent Go arm64 assembly translation of the lookup4 checker in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/implementation.cpp:13-28 and
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216.

#include "textflag.h"

DATA ·utf8Lookup4Byte1HighNEON<>+0(SB)/8, $0x0202020202020202
DATA ·utf8Lookup4Byte1HighNEON<>+8(SB)/8, $0x4915012180808080
GLOBL ·utf8Lookup4Byte1HighNEON<>(SB), RODATA|NOPTR, $16

DATA ·utf8Lookup4Byte1LowNEON<>+0(SB)/8, $0xcbcbcb8b8383a3e7
DATA ·utf8Lookup4Byte1LowNEON<>+8(SB)/8, $0xcbcbdbcbcbcbcbcb
GLOBL ·utf8Lookup4Byte1LowNEON<>(SB), RODATA|NOPTR, $16

DATA ·utf8Lookup4Byte2HighNEON<>+0(SB)/8, $0x0101010101010101
DATA ·utf8Lookup4Byte2HighNEON<>+8(SB)/8, $0x01010101babaaee6
GLOBL ·utf8Lookup4Byte2HighNEON<>(SB), RODATA|NOPTR, $16

DATA ·utf8Lookup4IncompleteMaxNEON<>+0(SB)/8, $0xffffffffffffffff
DATA ·utf8Lookup4IncompleteMaxNEON<>+8(SB)/8, $0xbfdfefffffffffff
GLOBL ·utf8Lookup4IncompleteMaxNEON<>(SB), RODATA|NOPTR, $16

// func validateUTF8Lookup4NEON(input []byte, remainder *[64]byte) (count int, hasError uint64)
TEXT ·validateUTF8Lookup4NEON(SB), NOSPLIT|NOFRAME, $0-48
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD remainder+24(FP), R2
	AND  $-64, R1, R4
	MOVD $0, R3
	MOVD $0, R9

	// V0=error, V1=prev_input_block, V2=prev_incomplete.
	VEOR V0.B16, V0.B16, V0.B16
	VEOR V1.B16, V1.B16, V1.B16
	VEOR V2.B16, V2.B16, V2.B16

	MOVD  $·utf8Lookup4Byte1HighNEON<>(SB), R5
	VLD1  (R5), [V20.B16]
	MOVD  $·utf8Lookup4Byte1LowNEON<>(SB), R5
	VLD1  (R5), [V21.B16]
	MOVD  $·utf8Lookup4Byte2HighNEON<>(SB), R5
	VLD1  (R5), [V22.B16]
	MOVD  $·utf8Lookup4IncompleteMaxNEON<>(SB), R5
	VLD1  (R5), [V23.B16]
	VMOVI $15, V24.B16
	VMOVI $128, V25.B16
	VMOVI $224, V26.B16
	VMOVI $240, V27.B16

utf8_full_next:
	CMP R4, R3
	BEQ utf8_remainder
	ADD R3, R0, R7
	B   utf8_check_block

utf8_remainder:
	MOVD R2, R7
	MOVD $1, R9

utf8_check_block:
	// Load the block once for the 64-byte ASCII reduction. The non-ASCII path
	// reloads one 16-byte chunk at a time so the same lookup4 body handles all
	// four chunks while keeping checker state in vector registers.
	VLD1 (R7), [V3.B16, V4.B16, V5.B16, V6.B16]
	VORR V4.B16, V3.B16, V16.B16
	VORR V6.B16, V5.B16, V17.B16
	VORR V17.B16, V16.B16, V16.B16
	VMOV V16.D[0], R10
	VMOV V16.D[1], R11
	ORR  R11, R10, R10
	TST  $0x8080808080808080, R10
	BNE  utf8_non_ascii
	VORR V2.B16, V0.B16, V0.B16
	B    utf8_block_checked

utf8_non_ascii:
	MOVD $4, R8
	MOVD R7, R6

utf8_chunk_loop:
	VLD1 (R6), [V3.B16]

	// prev1 plus the three exact lookup tables implement
	// check_special_cases(input, prev1).
	VEXT  $15, V3.B16, V1.B16, V4.B16
	VUSHR $4, V4.B16, V5.B16
	VTBL  V5.B16, [V20.B16], V5.B16
	VAND  V24.B16, V4.B16, V6.B16
	VTBL  V6.B16, [V21.B16], V6.B16
	VUSHR $4, V3.B16, V7.B16
	VTBL  V7.B16, [V22.B16], V7.B16
	VAND  V6.B16, V5.B16, V5.B16
	VAND  V7.B16, V5.B16, V5.B16

	// must_be_2_3_continuation(prev2, prev3): values >= E0 have
	// their top three bits set; values >= F0 have their top four bits set.
	VEXT  $14, V3.B16, V1.B16, V16.B16
	VEXT  $13, V3.B16, V1.B16, V17.B16
	VAND  V26.B16, V16.B16, V18.B16
	VCMEQ V26.B16, V18.B16, V18.B16
	VAND  V27.B16, V17.B16, V19.B16
	VCMEQ V27.B16, V19.B16, V19.B16
	VEOR  V19.B16, V18.B16, V30.B16
	VAND  V25.B16, V30.B16, V30.B16
	VEOR  V5.B16, V30.B16, V30.B16
	VORR  V30.B16, V0.B16, V0.B16

	VORR V3.B16, V3.B16, V1.B16
	ADD  $16, R6
	SUB  $1, R8
	CBNZ R8, utf8_chunk_loop

	// is_incomplete(last_chunk): compare the final three lanes with the exact
	// EF, DF, BF thresholds loaded in incompleteMax. Keep the resulting state
	// in V2 so the ASCII and EOF paths remain the pinned checker state machine.
	VEOR V2.B16, V2.B16, V2.B16
	VMOV V1.B[13], R12
	VMOV V23.B[13], R13
	CMP  R13, R12
	BHI  utf8_set_incomplete
	VMOV V1.B[14], R12
	VMOV V23.B[14], R13
	CMP  R13, R12
	BHI  utf8_set_incomplete
	VMOV V1.B[15], R12
	VMOV V23.B[15], R13
	CMP  R13, R12
	BLS  utf8_block_checked

utf8_set_incomplete:
	VMOVI $255, V2.B16

utf8_block_checked:
	CBNZ R9, utf8_eof
	VMOV V0.D[0], R10
	VMOV V0.D[1], R11
	ORR  R11, R10, R10
	CBNZ R10, utf8_return_error
	ADD  $64, R3
	B    utf8_full_next

utf8_eof:
	VORR V2.B16, V0.B16, V0.B16
	VMOV V0.D[0], R10
	VMOV V0.D[1], R11
	ORR  R11, R10, R10
	CBNZ R10, utf8_return_error
	MOVD R1, count+32(FP)
	MOVD $0, hasError+40(FP)
	RET

utf8_return_error:
	MOVD R3, count+32(FP)
	MOVD R10, hasError+40(FP)
	RET
