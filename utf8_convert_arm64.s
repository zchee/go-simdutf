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

// ASCII-run accelerators for UTF-8 source conversion on arm64. Each routine
// consumes complete in-bounds 16-byte NEON groups of ASCII bytes only.

// func utf8ASCIIToLatin1BlocksNEON(input, dst []byte) (consumed int)
TEXT ·utf8ASCIIToLatin1BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
utf8_ascii_l1_loop:
	CMP $16, R1
	BLT utf8_ascii_l1_done
	VLD1 (R0), [V0.B16]
	VUSHR $7, V0.B16, V1.B16
	VMOV V1.D[0], R5
	VMOV V1.D[1], R6
	ORR R6, R5, R5
	CBNZ R5, utf8_ascii_l1_done
	VST1.P [V0.B16], 16(R2)
	ADD $16, R0
	ADD $16, R3
	SUB $16, R1
	B utf8_ascii_l1_loop
utf8_ascii_l1_done:
	MOVD R3, consumed+48(FP)
	RET

// func utf8ASCIIToUTF16LEBlocksNEON(input []byte, dst []uint16) (consumed int)
TEXT ·utf8ASCIIToUTF16LEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
utf8_ascii_u16le_loop:
	CMP $16, R1
	BLT utf8_ascii_u16le_done
	VLD1 (R0), [V0.B16]
	VUSHR $7, V0.B16, V1.B16
	VMOV V1.D[0], R5
	VMOV V1.D[1], R6
	ORR R6, R5, R5
	CBNZ R5, utf8_ascii_u16le_done
	VUXTL V0.B8, V2.H8
	VUXTL2 V0.B16, V3.H8
	VST1.P [V2.H8, V3.H8], 32(R2)
	ADD $16, R0
	ADD $16, R3
	SUB $16, R1
	B utf8_ascii_u16le_loop
utf8_ascii_u16le_done:
	MOVD R3, consumed+48(FP)
	RET

// func utf8ASCIIToUTF16BEBlocksNEON(input []byte, dst []uint16) (consumed int)
TEXT ·utf8ASCIIToUTF16BEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
utf8_ascii_u16be_loop:
	CMP $16, R1
	BLT utf8_ascii_u16be_done
	VLD1 (R0), [V0.B16]
	VUSHR $7, V0.B16, V1.B16
	VMOV V1.D[0], R5
	VMOV V1.D[1], R6
	ORR R6, R5, R5
	CBNZ R5, utf8_ascii_u16be_done
	VUXTL V0.B8, V2.H8
	VUXTL2 V0.B16, V3.H8
	VREV16 V2.B16, V2.B16
	VREV16 V3.B16, V3.B16
	VST1.P [V2.H8, V3.H8], 32(R2)
	ADD $16, R0
	ADD $16, R3
	SUB $16, R1
	B utf8_ascii_u16be_loop
utf8_ascii_u16be_done:
	MOVD R3, consumed+48(FP)
	RET

// func utf8ASCIIToUTF32BlocksNEON(input []byte, dst []uint32) (consumed int)
TEXT ·utf8ASCIIToUTF32BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3
utf8_ascii_u32_loop:
	CMP $16, R1
	BLT utf8_ascii_u32_done
	VLD1 (R0), [V0.B16]
	VUSHR $7, V0.B16, V1.B16
	VMOV V1.D[0], R5
	VMOV V1.D[1], R6
	ORR R6, R5, R5
	CBNZ R5, utf8_ascii_u32_done
	VUXTL V0.B8, V1.H8
	VUXTL2 V0.B16, V2.H8
	VUXTL V1.H4, V3.S4
	VUXTL2 V1.H8, V4.S4
	VUXTL V2.H4, V5.S4
	VUXTL2 V2.H8, V6.S4
	VST1.P [V3.S4, V4.S4, V5.S4, V6.S4], 64(R2)
	ADD $16, R0
	ADD $16, R3
	SUB $16, R1
	B utf8_ascii_u32_loop
utf8_ascii_u32_done:
	MOVD R3, consumed+48(FP)
	RET
