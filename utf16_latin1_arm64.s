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

// Handwritten NEON complete-block kernels adapted from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// src/arm64/arm_convert_utf16_to_latin1.cpp.
//
// Each iteration consumes 8 uint16 code units. Go 1.26 Plan 9 arm64 has no
// VMOVN/UMAXV mnemonics, so the vmaxvq_u16 <= 0xff check is expressed with
// VUSHR $8 + GPR OR reduction (proven in latin1_arm64.s / utf8_convert_arm64.s)
// and the narrow pack uses VUZP1.B16. On non-Latin-1 input the kernel stops
// before storing that vector and returns the uint16 count already consumed.

// func convertUTF16LEToLatin1BlocksNEON(input []uint16, dst []byte) (consumed int)
TEXT ·convertUTF16LEToLatin1BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

utf16le_latin1_loop:
	CMP  $8, R1
	BLT  utf16le_latin1_done
	VLD1 (R0), [V0.H8]

	// vmax-style: any high byte non-zero => word > 0xff
	VUSHR $8, V0.H8, V1.H8
	VMOV  V1.D[0], R5
	VMOV  V1.D[1], R6
	ORR   R6, R5, R5
	CBNZ  R5, utf16le_latin1_done

	// VMOVN substitute: pack low bytes of 8 halfwords into 8 bytes
	VUZP1 V0.B16, V0.B16, V1.B16
	VMOV  V1.D[0], R5
	MOVD  R5, (R2)
	ADD   $8, R2
	ADD   $16, R0
	ADD   $8, R3
	SUB   $8, R1
	B     utf16le_latin1_loop

utf16le_latin1_done:
	MOVD R3, consumed+48(FP)
	RET

// func convertUTF16BEToLatin1BlocksNEON(input []uint16, dst []byte) (consumed int)
TEXT ·convertUTF16BEToLatin1BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	MOVD $0, R3

utf16be_latin1_loop:
	CMP    $8, R1
	BLT    utf16be_latin1_done
	VLD1   (R0), [V0.H8]
	VREV16 V0.B16, V0.B16
	VUSHR  $8, V0.H8, V1.H8
	VMOV   V1.D[0], R5
	VMOV   V1.D[1], R6
	ORR    R6, R5, R5
	CBNZ   R5, utf16be_latin1_done
	VUZP1  V0.B16, V0.B16, V1.B16
	VMOV   V1.D[0], R5
	MOVD   R5, (R2)
	ADD    $8, R2
	ADD    $16, R0
	ADD    $8, R3
	SUB    $8, R1
	B      utf16be_latin1_loop

utf16be_latin1_done:
	MOVD R3, consumed+48(FP)
	RET
