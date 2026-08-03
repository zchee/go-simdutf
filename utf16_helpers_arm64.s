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

#include "textflag.h"

// func changeEndiannessUTF16BlocksNEON(input, dst []uint16) (consumed int)
// Complete 32-uint16 groups via VREV16 on four 8-lane vectors.
TEXT ·changeEndiannessUTF16BlocksNEON(SB), NOSPLIT|NOFRAME, $0-56
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	AND  $-32, R1, R1
	MOVD $0, R3
	CBZ  R1, change_endian_neon_done

change_endian_neon_loop:
	VLD1.P 64(R0), [V0.H8, V1.H8, V2.H8, V3.H8]
	VREV16 V0.B16, V0.B16
	VREV16 V1.B16, V1.B16
	VREV16 V2.B16, V2.B16
	VREV16 V3.B16, V3.B16
	VST1.P [V0.H8, V1.H8, V2.H8, V3.H8], 64(R2)
	ADD    $32, R3
	SUB    $32, R1
	CBNZ   R1, change_endian_neon_loop

change_endian_neon_done:
	MOVD R3, consumed+48(FP)
	RET

// countUTF16* NEON uses the bytemask kernel:
//   t2 = (t1 == 0) ? 0 : 1 where t1 = (word & 0xfc00) ^ 0xdc00
// Go 1.26 Plan 9 arm64 has VCMEQ but no UMIN mnemonic in this tree, so the
// min(t1,1) step is expressed with VCMEQ/VEOR/VAND against proven opcodes.

// func countUTF16LEBlocksNEON(input []uint16) (count int)
TEXT ·countUTF16LEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-32, R1, R1
	MOVD $0, R2
	CBZ  R1, count_utf16le_neon_done

	MOVD $0xfc00, R3
	VMOV R3, V4.H8
	MOVD $0xdc00, R3
	VMOV R3, V5.H8
	MOVD $1, R3
	VMOV R3, V6.H8
	MOVD $0xffff, R3
	VMOV R3, V16.H8
	VEOR V7.B16, V7.B16, V7.B16 // 64-bit accumulator via VUADDLV path
	MOVD $0, R3                 // iterations

count_utf16le_neon_loop:
	VLD1.P 64(R0), [V0.H8, V1.H8, V2.H8, V3.H8]

	// V0
	VAND  V4.B16, V0.B16, V0.B16
	VEOR  V5.B16, V0.B16, V0.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V0.H8, V0.H8     // FFFF where t1==0 (low surrogate)
	VEOR  V16.B16, V0.B16, V0.B16 // invert
	VAND  V6.B16, V0.B16, V0.B16  // 1 where not low surrogate

	// V1
	VAND  V4.B16, V1.B16, V1.B16
	VEOR  V5.B16, V1.B16, V1.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V1.H8, V1.H8
	VEOR  V16.B16, V1.B16, V1.B16
	VAND  V6.B16, V1.B16, V1.B16

	// V2
	VAND  V4.B16, V2.B16, V2.B16
	VEOR  V5.B16, V2.B16, V2.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V2.H8, V2.H8
	VEOR  V16.B16, V2.B16, V2.B16
	VAND  V6.B16, V2.B16, V2.B16

	// V3
	VAND  V4.B16, V3.B16, V3.B16
	VEOR  V5.B16, V3.B16, V3.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V3.H8, V3.H8
	VEOR  V16.B16, V3.B16, V3.B16
	VAND  V6.B16, V3.B16, V3.B16

	// pairwise reduce 32x uint16 ones into a scalar addend, then accumulate.
	VADDP   V1.H8, V0.H8, V0.H8
	VADDP   V3.H8, V2.H8, V2.H8
	VADDP   V2.H8, V0.H8, V0.H8
	VUADDLV V0.H8, V1
	VADD    V1, V7

	SUB  $32, R1
	CBNZ R1, count_utf16le_neon_loop

	VMOV V7.D[0], R2

count_utf16le_neon_done:
	MOVD R2, count+24(FP)
	RET

// func countUTF16BEBlocksNEON(input []uint16) (count int)
TEXT ·countUTF16BEBlocksNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-32, R1, R1
	MOVD $0, R2
	CBZ  R1, count_utf16be_neon_done

	MOVD $0xfc00, R3
	VMOV R3, V4.H8
	MOVD $0xdc00, R3
	VMOV R3, V5.H8
	MOVD $1, R3
	VMOV R3, V6.H8
	MOVD $0xffff, R3
	VMOV R3, V16.H8
	VEOR V7.B16, V7.B16, V7.B16

count_utf16be_neon_loop:
	VLD1.P 64(R0), [V0.H8, V1.H8, V2.H8, V3.H8]
	VREV16 V0.B16, V0.B16
	VREV16 V1.B16, V1.B16
	VREV16 V2.B16, V2.B16
	VREV16 V3.B16, V3.B16

	VAND  V4.B16, V0.B16, V0.B16
	VEOR  V5.B16, V0.B16, V0.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V0.H8, V0.H8
	VEOR  V16.B16, V0.B16, V0.B16
	VAND  V6.B16, V0.B16, V0.B16

	VAND  V4.B16, V1.B16, V1.B16
	VEOR  V5.B16, V1.B16, V1.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V1.H8, V1.H8
	VEOR  V16.B16, V1.B16, V1.B16
	VAND  V6.B16, V1.B16, V1.B16

	VAND  V4.B16, V2.B16, V2.B16
	VEOR  V5.B16, V2.B16, V2.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V2.H8, V2.H8
	VEOR  V16.B16, V2.B16, V2.B16
	VAND  V6.B16, V2.B16, V2.B16

	VAND  V4.B16, V3.B16, V3.B16
	VEOR  V5.B16, V3.B16, V3.B16
	VEOR  V8.B16, V8.B16, V8.B16
	VCMEQ V8.H8, V3.H8, V3.H8
	VEOR  V16.B16, V3.B16, V3.B16
	VAND  V6.B16, V3.B16, V3.B16

	VADDP   V1.H8, V0.H8, V0.H8
	VADDP   V3.H8, V2.H8, V2.H8
	VADDP   V2.H8, V0.H8, V0.H8
	VUADDLV V0.H8, V1
	VADD    V1, V7

	SUB  $32, R1
	CBNZ R1, count_utf16be_neon_loop

	VMOV V7.D[0], R2

count_utf16be_neon_done:
	MOVD R2, count+24(FP)
	RET
