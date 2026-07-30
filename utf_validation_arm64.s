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
// Independent Go arm64 assembly translation of the block classification used by
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_validate_utf16.cpp and src/arm64/arm_validate_utf32.cpp.
// Each routine returns the number of code units preceding the first complete
// vector which needs scalar inspection. It neither reads a tail nor calls out.

#include "textflag.h"

// func validateUTF16LEPrefixNEON(input []uint16) int
TEXT ·validateUTF16LEPrefixNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-16, R1, R1
	MOVD $0, R2
	MOVD $0xf800, R3
	VDUP R3, V2.H8
	MOVD $0xd800, R3
	VDUP R3, V3.H8
utf16le_prefix_loop:
	CMP R1, R2
	BEQ utf16le_prefix_done
	VLD1.P 32(R0), [V0.H8, V1.H8]
	VAND V2.B16, V0.B16, V0.B16
	VAND V2.B16, V1.B16, V1.B16
	VCMEQ V3.H8, V0.H8, V0.H8
	VCMEQ V3.H8, V1.H8, V1.H8
	VORR V1.B16, V0.B16, V0.B16
	VMOV V0.D[0], R4
	VMOV V0.D[1], R5
	ORR R5, R4, R4
	CBNZ R4, utf16le_prefix_done
	ADD $16, R2
	B utf16le_prefix_loop
utf16le_prefix_done:
	MOVD R2, ret+24(FP)
	RET

// func validateUTF16BEPrefixNEON(input []uint16) int
TEXT ·validateUTF16BEPrefixNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-16, R1, R1
	MOVD $0, R2
	MOVD $0x00f8, R3
	VDUP R3, V2.H8
	MOVD $0x00d8, R3
	VDUP R3, V3.H8
utf16be_prefix_loop:
	CMP R1, R2
	BEQ utf16be_prefix_done
	VLD1.P 32(R0), [V0.H8, V1.H8]
	VAND V2.B16, V0.B16, V0.B16
	VAND V2.B16, V1.B16, V1.B16
	VCMEQ V3.H8, V0.H8, V0.H8
	VCMEQ V3.H8, V1.H8, V1.H8
	VORR V1.B16, V0.B16, V0.B16
	VMOV V0.D[0], R4
	VMOV V0.D[1], R5
	ORR R5, R4, R4
	CBNZ R4, utf16be_prefix_done
	ADD $16, R2
	B utf16be_prefix_loop
utf16be_prefix_done:
	MOVD R2, ret+24(FP)
	RET

// func validateUTF32PrefixNEON(input []uint32) int
TEXT ·validateUTF32PrefixNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-4, R1, R1
	MOVD $0, R2
	// A nonzero high half is left to the scalar oracle. This conservative
	// vector fast path still rejects surrogate-containing BMP blocks in NEON.
	MOVD $27, R3
	VDUP R3, V3.S4
utf32_prefix_loop:
	CMP R1, R2
	BEQ utf32_prefix_done
	VLD1.P 16(R0), [V0.S4]
	VORR V0.B16, V0.B16, V1.B16
	VUSHR $16, V0.S4, V0.S4
	VUSHR $11, V1.S4, V1.S4
	VCMEQ V3.S4, V1.S4, V1.S4
	VORR V1.B16, V0.B16, V0.B16
	VMOV V0.D[0], R4
	VMOV V0.D[1], R5
	ORR R5, R4, R4
	CBNZ R4, utf32_prefix_done
	ADD $4, R2
	B utf32_prefix_loop
utf32_prefix_done:
	MOVD R2, ret+24(FP)
	RET
