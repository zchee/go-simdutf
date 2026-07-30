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
// Independent Go arm64 assembly translation of the ASCII NEON fast path from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_convert_utf16_to_utf8.cpp. Each routine returns the number of
// UTF-16 code units consumed before the first complete 8-lane vector that is
// not pure ASCII. ASCII packs 1:1 into UTF-8 bytes (written == consumed).
// Non-ASCII / surrogate / variable-width handling is left to Go scalar remount.
// Big-endian raw storage is byte-swapped with VREV16 before the ASCII check.

#include "textflag.h"

// func convertUTF16LEToUTF8BlocksNEON(input []uint16, dst []byte) (consumed int)
TEXT ·convertUTF16LEToUTF8BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

utf16le_utf8_loop:
	CMP  $8, R1
	BLT  utf16le_utf8_done
	VLD1 (R0), [V0.H8]
	// non-ASCII when any halfword >> 7 is non-zero (word > 0x7f)
	VUSHR $7, V0.H8, V1.H8
	VMOV V1.D[0], R5
	VMOV V1.D[1], R6
	ORR  R6, R5, R5
	CBNZ R5, utf16le_utf8_done
	// pack low bytes of 8 halfwords into 8 UTF-8 ASCII bytes
	VUZP1 V0.B16, V0.B16, V1.B16
	VMOV V1.D[0], R5
	MOVD R5, (R2)
	ADD  $8, R2
	ADD  $16, R0
	ADD  $8, R3
	SUB  $8, R1
	B    utf16le_utf8_loop

utf16le_utf8_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertUTF16BEToUTF8BlocksNEON(input []uint16, dst []byte) (consumed int)
TEXT ·convertUTF16BEToUTF8BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

utf16be_utf8_loop:
	CMP  $8, R1
	BLT  utf16be_utf8_done
	VLD1 (R0), [V0.H8]
	VREV16 V0.B16, V0.B16
	VUSHR $7, V0.H8, V1.H8
	VMOV V1.D[0], R5
	VMOV V1.D[1], R6
	ORR  R6, R5, R5
	CBNZ R5, utf16be_utf8_done
	VUZP1 V0.B16, V0.B16, V1.B16
	VMOV V1.D[0], R5
	MOVD R5, (R2)
	ADD  $8, R2
	ADD  $16, R0
	ADD  $8, R3
	SUB  $8, R1
	B    utf16be_utf8_loop

utf16be_utf8_done:
	MOVD R3, consumed+48(FP)
	RET
