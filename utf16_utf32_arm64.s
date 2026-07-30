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
// Independent Go arm64 assembly translation of the no-surrogate NEON path from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_convert_utf16_to_utf32.cpp. Each routine returns the number of
// UTF-16 code units consumed before the first complete 8-lane vector that
// contains a surrogate. Surrogate/error handling is left to Go scalar remount.

#include "textflag.h"

// func convertUTF16LEToUTF32BlocksNEON(input []uint16, dst []uint32) (consumed int)
TEXT ·convertUTF16LEToUTF32BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
	MOVD $0xf800, R4
	VDUP R4, V2.H8
	MOVD $0xd800, R4
	VDUP R4, V3.H8

utf16le_utf32_loop:
	CMP  $8, R1
	BLT  utf16le_utf32_done
	VLD1 (R0), [V0.H8]
	// surrogates when (word & 0xf800) == 0xd800
	VAND  V2.B16, V0.B16, V1.B16
	VCMEQ V3.H8, V1.H8, V1.H8
	VMOV  V1.D[0], R5
	VMOV  V1.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf16le_utf32_done
	// widen eight u16 → eight u32 and store
	VUXTL  V0.H4, V4.S4
	VUXTL2 V0.H8, V5.S4
	VST1.P [V4.S4, V5.S4], 32(R2)
	ADD  $16, R0
	ADD  $8, R3
	SUB  $8, R1
	B    utf16le_utf32_loop

utf16le_utf32_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertUTF16BEToUTF32BlocksNEON(input []uint16, dst []uint32) (consumed int)
TEXT ·convertUTF16BEToUTF32BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
	MOVD $0xf800, R4
	VDUP R4, V2.H8
	MOVD $0xd800, R4
	VDUP R4, V3.H8

utf16be_utf32_loop:
	CMP  $8, R1
	BLT  utf16be_utf32_done
	VLD1 (R0), [V0.H8]
	VREV16 V0.B16, V0.B16
	VAND  V2.B16, V0.B16, V1.B16
	VCMEQ V3.H8, V1.H8, V1.H8
	VMOV  V1.D[0], R5
	VMOV  V1.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf16be_utf32_done
	VUXTL  V0.H4, V4.S4
	VUXTL2 V0.H8, V5.S4
	VST1.P [V4.S4, V5.S4], 32(R2)
	ADD  $16, R0
	ADD  $8, R3
	SUB  $8, R1
	B    utf16be_utf32_loop

utf16be_utf32_done:
	MOVD R3, consumed+48(FP)
	RET
