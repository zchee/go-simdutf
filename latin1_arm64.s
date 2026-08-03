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

#include "textflag.h"

// Handwritten NEON complete-block kernels adapted from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f).  Callers pass lengths rounded
// down to the stated block size, so none of these routines over-read or
// over-write a slice.

// func utf8LengthFromLatin1BlocksNEON(input []byte) (length int)
TEXT ·utf8LengthFromLatin1BlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD $0, R2
	CBZ  R1, latin1_utf8_length_done

latin1_utf8_length_loop:
	VLD1.P  64(R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	VUSHR   $7, V0.B16, V4.B16
	VUSHR   $7, V1.B16, V5.B16
	VUSHR   $7, V2.B16, V6.B16
	VUSHR   $7, V3.B16, V7.B16
	VADDP   V5.B16, V4.B16, V8.B16
	VADDP   V7.B16, V6.B16, V9.B16
	VADDP   V9.B16, V8.B16, V8.B16
	VUADDLV V8.B16, V10
	VMOV    V10.D[0], R3
	ADD     $64, R3, R3
	ADD     R3, R2, R2
	SUB     $64, R1
	CBNZ    R1, latin1_utf8_length_loop

latin1_utf8_length_done:
	MOVD R2, length+24(FP)
	RET

// func convertLatin1ToUTF8ASCIIPrefixNEON(input, dst []byte) (consumed int)
TEXT ·convertLatin1ToUTF8ASCIIPrefixNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	AND  $-64, R1, R4
	MOVD $0, R3

latin1_utf8_ascii_loop:
	CMP    R4, R3
	BEQ    latin1_utf8_ascii_done
	VLD1   (R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	VORR   V1.B16, V0.B16, V4.B16
	VORR   V3.B16, V2.B16, V5.B16
	VORR   V5.B16, V4.B16, V4.B16
	VUSHR  $7, V4.B16, V4.B16
	VMOV   V4.D[0], R5
	VMOV   V4.D[1], R6
	ORR    R6, R5, R5
	CBNZ   R5, latin1_utf8_ascii_done
	VST1.P [V0.B16, V1.B16, V2.B16, V3.B16], 64(R2)
	ADD    $64, R0
	ADD    $64, R3
	B      latin1_utf8_ascii_loop

latin1_utf8_ascii_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertLatin1ToUTF16LEBlocksNEON(input []byte, dst []uint16) (consumed int)
TEXT ·convertLatin1ToUTF16LEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

latin1_utf16le_loop:
	CMP    R1, R3
	BEQ    latin1_utf16le_done
	VLD1.P 16(R0), [V0.B16]
	VUXTL  V0.B8, V1.H8
	VUXTL2 V0.B16, V2.H8
	VST1.P [V1.H8, V2.H8], 32(R2)
	ADD    $16, R3
	B      latin1_utf16le_loop

latin1_utf16le_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertLatin1ToUTF16BEBlocksNEON(input []byte, dst []uint16) (consumed int)
TEXT ·convertLatin1ToUTF16BEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

latin1_utf16be_loop:
	CMP    R1, R3
	BEQ    latin1_utf16be_done
	VLD1.P 16(R0), [V0.B16]
	VUXTL  V0.B8, V1.H8
	VUXTL2 V0.B16, V2.H8
	VREV16 V1.B16, V1.B16
	VREV16 V2.B16, V2.B16
	VST1.P [V1.H8, V2.H8], 32(R2)
	ADD    $16, R3
	B      latin1_utf16be_loop

latin1_utf16be_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertLatin1ToUTF32BlocksNEON(input []byte, dst []uint32) (consumed int)
TEXT ·convertLatin1ToUTF32BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

latin1_utf32_loop:
	CMP    R1, R3
	BEQ    latin1_utf32_done
	VLD1.P 16(R0), [V0.B16]
	VUXTL  V0.B8, V1.H8
	VUXTL2 V0.B16, V2.H8
	VUXTL  V1.H4, V3.S4
	VUXTL2 V1.H8, V4.S4
	VUXTL  V2.H4, V5.S4
	VUXTL2 V2.H8, V6.S4
	VST1.P [V3.S4, V4.S4, V5.S4, V6.S4], 64(R2)
	ADD    $16, R3
	B      latin1_utf32_loop

latin1_utf32_done:
	MOVD R3, consumed+48(FP)
	RET
