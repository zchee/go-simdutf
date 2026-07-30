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
// Independent Go arm64 assembly translation of the NEON fast paths from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_convert_utf32_to_latin1.cpp (8×u32 Latin-1 pack),
// src/arm64/arm_convert_utf32_to_utf8.cpp (ASCII 1:1 prefix),
// src/arm64/arm_convert_utf32_to_utf16.cpp (BMP no-surrogate pack), and
// src/arm64/implementation.cpp (utf8/utf16_length_from_utf32). Go 1.26 Plan 9
// arm64 has no VMOVN/UMAXV/VCLE mnemonics, so range checks use VUSHR + GPR OR
// reduction and packs use VUZP1. Non-fast-path vectors stop before store;
// scalar remount handles tails and errors.

#include "textflag.h"

// func convertUTF32ToLatin1BlocksNEON(input []uint32, dst []byte) (consumed int)
TEXT ·convertUTF32ToLatin1BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

utf32_latin1_loop:
	CMP  $8, R1
	BLT  utf32_latin1_done
	VLD1 (R0), [V0.S4, V1.S4]
	// non-Latin-1 when any word >> 8 is non-zero
	VUSHR $8, V0.S4, V2.S4
	VUSHR $8, V1.S4, V3.S4
	VORR  V3.B16, V2.B16, V2.B16
	VMOV  V2.D[0], R5
	VMOV  V2.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf32_latin1_done
	// pack low bytes of 8 uint32 → 8 Latin-1 bytes
	VUZP1 V1.H8, V0.H8, V4.H8
	VUZP1 V4.B16, V4.B16, V5.B16
	VMOV  V5.D[0], R5
	MOVD  R5, (R2)
	ADD   $8, R2
	ADD   $32, R0
	ADD   $8, R3
	SUB   $8, R1
	B     utf32_latin1_loop

utf32_latin1_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertUTF32ToUTF8BlocksNEON(input []uint32, dst []byte) (consumed int)
TEXT ·convertUTF32ToUTF8BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

utf32_utf8_loop:
	CMP  $8, R1
	BLT  utf32_utf8_done
	VLD1 (R0), [V0.S4, V1.S4]
	// non-ASCII when any word >> 7 is non-zero
	VUSHR $7, V0.S4, V2.S4
	VUSHR $7, V1.S4, V3.S4
	VORR  V3.B16, V2.B16, V2.B16
	VMOV  V2.D[0], R5
	VMOV  V2.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf32_utf8_done
	VUZP1 V1.H8, V0.H8, V4.H8
	VUZP1 V4.B16, V4.B16, V5.B16
	VMOV  V5.D[0], R5
	MOVD  R5, (R2)
	ADD   $8, R2
	ADD   $32, R0
	ADD   $8, R3
	SUB   $8, R1
	B     utf32_utf8_loop

utf32_utf8_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertUTF32ToUTF16LEBlocksNEON(input []uint32, dst []uint16) (consumed int)
TEXT ·convertUTF32ToUTF16LEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
	MOVD $0xf800, R4
	VDUP R4, V6.H8
	MOVD $0xd800, R4
	VDUP R4, V7.H8

utf32_utf16le_loop:
	CMP  $8, R1
	BLT  utf32_utf16le_done
	VLD1 (R0), [V0.S4, V1.S4]
	// stop before supplementary / too-large words
	VUSHR $16, V0.S4, V2.S4
	VUSHR $16, V1.S4, V3.S4
	VORR  V3.B16, V2.B16, V2.B16
	VMOV  V2.D[0], R5
	VMOV  V2.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf32_utf16le_done
	// pack low 16 bits; reject surrogate BMP words
	VUZP1 V1.H8, V0.H8, V4.H8
	VAND  V6.B16, V4.B16, V5.B16
	VCMEQ V7.H8, V5.H8, V5.H8
	VMOV  V5.D[0], R5
	VMOV  V5.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf32_utf16le_done
	VST1  [V4.H8], (R2)
	ADD   $16, R2
	ADD   $32, R0
	ADD   $8, R3
	SUB   $8, R1
	B     utf32_utf16le_loop

utf32_utf16le_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertUTF32ToUTF16BEBlocksNEON(input []uint32, dst []uint16) (consumed int)
TEXT ·convertUTF32ToUTF16BEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
	MOVD $0xf800, R4
	VDUP R4, V6.H8
	MOVD $0xd800, R4
	VDUP R4, V7.H8

utf32_utf16be_loop:
	CMP  $8, R1
	BLT  utf32_utf16be_done
	VLD1 (R0), [V0.S4, V1.S4]
	VUSHR $16, V0.S4, V2.S4
	VUSHR $16, V1.S4, V3.S4
	VORR  V3.B16, V2.B16, V2.B16
	VMOV  V2.D[0], R5
	VMOV  V2.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf32_utf16be_done
	VUZP1 V1.H8, V0.H8, V4.H8
	VAND  V6.B16, V4.B16, V5.B16
	VCMEQ V7.H8, V5.H8, V5.H8
	VMOV  V5.D[0], R5
	VMOV  V5.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf32_utf16be_done
	VREV16 V4.B16, V4.B16
	VST1  [V4.H8], (R2)
	ADD   $16, R2
	ADD   $32, R0
	ADD   $8, R3
	SUB   $8, R1
	B     utf32_utf16be_loop

utf32_utf16be_done:
	MOVD R3, consumed+48(FP)
	RET

// func utf8LengthFromUTF32BlocksNEON(input []uint32) (length int)
// Caller passes a length that is a multiple of 4.
TEXT ·utf8LengthFromUTF32BlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD $0, R2
	VEOR V7.B16, V7.B16, V7.B16
	MOVD $1, R3
	VDUP R3, V6.S4
	MOVD $0xffffffff, R3
	VDUP R3, V5.S4

utf8_len_loop:
	CBZ  R1, utf8_len_done
	VLD1.P 16(R0), [V0.S4]
	// 1 per lane, plus one each for >0x7f / >0x7ff / >0xffff
	VUSHR $7, V0.S4, V1.S4
	VCMEQ V7.S4, V1.S4, V1.S4
	VEOR  V5.B16, V1.B16, V1.B16
	VAND  V6.B16, V1.B16, V1.B16
	VUSHR $11, V0.S4, V2.S4
	VCMEQ V7.S4, V2.S4, V2.S4
	VEOR  V5.B16, V2.B16, V2.B16
	VAND  V6.B16, V2.B16, V2.B16
	VUSHR $16, V0.S4, V3.S4
	VCMEQ V7.S4, V3.S4, V3.S4
	VEOR  V5.B16, V3.B16, V3.B16
	VAND  V6.B16, V3.B16, V3.B16
	VADD  V2.S4, V1.S4, V1.S4
	VADD  V3.S4, V1.S4, V1.S4
	VADD  V6.S4, V1.S4, V1.S4
	VADDP V1.S4, V1.S4, V1.S4
	VMOV  V1.S[0], R4
	VMOV  V1.S[1], R5
	ADD   R5, R4, R4
	ADD   R4, R2, R2
	SUB   $4, R1
	B     utf8_len_loop

utf8_len_done:
	MOVD R2, length+24(FP)
	RET

// func utf16LengthFromUTF32BlocksNEON(input []uint32) (length int)
// Caller passes a length that is a multiple of 4.
TEXT ·utf16LengthFromUTF32BlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD $0, R2
	VEOR V7.B16, V7.B16, V7.B16
	MOVD $1, R3
	VDUP R3, V6.S4
	MOVD $0xffffffff, R3
	VDUP R3, V5.S4

utf16_len_loop:
	CBZ  R1, utf16_len_done
	VLD1.P 16(R0), [V0.S4]
	// 4 + number of words > 0xffff
	VUSHR $16, V0.S4, V1.S4
	VCMEQ V7.S4, V1.S4, V1.S4
	VEOR  V5.B16, V1.B16, V1.B16
	VAND  V6.B16, V1.B16, V1.B16
	VADDP V1.S4, V1.S4, V1.S4
	VMOV  V1.S[0], R4
	VMOV  V1.S[1], R5
	ADD   R5, R4, R4
	ADD   $4, R4, R4
	ADD   R4, R2, R2
	SUB   $4, R1
	B     utf16_len_loop

utf16_len_done:
	MOVD R2, length+24(FP)
	RET
