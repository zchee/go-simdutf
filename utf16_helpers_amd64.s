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

// Byte-swap mask for PSHUFB / VPSHUFB: swap adjacent bytes in each uint16.
DATA ·utf16ByteSwapMask<>(SB)/8, $0x0607040502030001
DATA ·utf16ByteSwapMask<>+8(SB)/8, $0x0e0f0c0d0a0b0809
DATA ·utf16ByteSwapMask<>+16(SB)/8, $0x0607040502030001
DATA ·utf16ByteSwapMask<>+24(SB)/8, $0x0e0f0c0d0a0b0809
GLOBL ·utf16ByteSwapMask<>(SB), RODATA|NOPTR, $32

DATA ·utf16FC00<>(SB)/8, $0xfc00fc00fc00fc00
DATA ·utf16FC00<>+8(SB)/8, $0xfc00fc00fc00fc00
DATA ·utf16FC00<>+16(SB)/8, $0xfc00fc00fc00fc00
DATA ·utf16FC00<>+24(SB)/8, $0xfc00fc00fc00fc00
GLOBL ·utf16FC00<>(SB), RODATA|NOPTR, $32

DATA ·utf16DC00<>(SB)/8, $0xdc00dc00dc00dc00
DATA ·utf16DC00<>+8(SB)/8, $0xdc00dc00dc00dc00
DATA ·utf16DC00<>+16(SB)/8, $0xdc00dc00dc00dc00
DATA ·utf16DC00<>+24(SB)/8, $0xdc00dc00dc00dc00
GLOBL ·utf16DC00<>(SB), RODATA|NOPTR, $32

DATA ·utf16Ones<>(SB)/8, $0x0001000100010001
DATA ·utf16Ones<>+8(SB)/8, $0x0001000100010001
DATA ·utf16Ones<>+16(SB)/8, $0x0001000100010001
DATA ·utf16Ones<>+24(SB)/8, $0x0001000100010001
GLOBL ·utf16Ones<>(SB), RODATA|NOPTR, $32

// func changeEndiannessUTF16BlocksWestmere(input, dst []uint16) (consumed int)
// Processes complete 32-uint16 (64-byte) groups with SSSE3 PSHUFB.
TEXT ·changeEndiannessUTF16BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	ANDQ $-32, CX
	XORQ AX, AX
	TESTQ CX, CX
	JE change_endian_westmere_done
	MOVOU ·utf16ByteSwapMask<>(SB), X7

change_endian_westmere_loop:
	MOVOU 0(SI), X0
	MOVOU 16(SI), X1
	MOVOU 32(SI), X2
	MOVOU 48(SI), X3
	PSHUFB X7, X0
	PSHUFB X7, X1
	PSHUFB X7, X2
	PSHUFB X7, X3
	MOVOU X0, 0(DI)
	MOVOU X1, 16(DI)
	MOVOU X2, 32(DI)
	MOVOU X3, 48(DI)
	ADDQ $64, SI
	ADDQ $64, DI
	ADDQ $32, AX
	SUBQ $32, CX
	JNE change_endian_westmere_loop

change_endian_westmere_done:
	MOVQ AX, consumed+48(FP)
	RET

// func changeEndiannessUTF16BlocksHaswell(input, dst []uint16) (consumed int)
// Processes complete 32-uint16 groups with AVX2 VPSHUFB.
TEXT ·changeEndiannessUTF16BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	ANDQ $-32, CX
	XORQ AX, AX
	TESTQ CX, CX
	JE change_endian_haswell_done
	VMOVDQU ·utf16ByteSwapMask<>(SB), Y7

change_endian_haswell_loop:
	VMOVDQU 0(SI), Y0
	VMOVDQU 32(SI), Y1
	VPSHUFB Y7, Y0, Y0
	VPSHUFB Y7, Y1, Y1
	VMOVDQU Y0, 0(DI)
	VMOVDQU Y1, 32(DI)
	ADDQ $64, SI
	ADDQ $64, DI
	ADDQ $32, AX
	SUBQ $32, CX
	JNE change_endian_haswell_loop

change_endian_haswell_done:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// horizontalSumU16X: sum 8 uint16 lanes in X0 into AX (clobbers X0,X1).
#define HSUM_U16_X0_TO_AX \
	MOVOU X0, X1; \
	PSRLDQ $8, X1; \
	PADDW X1, X0; \
	MOVOU X0, X1; \
	PSRLDQ $4, X1; \
	PADDW X1, X0; \
	MOVOU X0, X1; \
	PSRLDQ $2, X1; \
	PADDW X1, X0; \
	PEXTRW $0, X0, AX

// func countUTF16LEBlocksWestmere(input []uint16) (count int)
// Bytemask count_code_points: min((word&0xfc00)^0xdc00, 1), LE/native words.
TEXT ·countUTF16LEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ R8, R8                      // total
	TESTQ CX, CX
	JE count_utf16le_westmere_done
	MOVOU ·utf16FC00<>(SB), X5
	MOVOU ·utf16DC00<>(SB), X6
	MOVOU ·utf16Ones<>(SB), X7
	PXOR X2, X2                      // lane counters
	XORQ R9, R9                      // iterations since flush

count_utf16le_westmere_loop:
	// 4x 8-uint16 vectors = 32 uint16s
	MOVOU 0(SI), X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	MOVOU 16(SI), X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	MOVOU 32(SI), X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	MOVOU 48(SI), X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	ADDQ $64, SI
	SUBQ $32, CX
	INCQ R9
	CMPQ R9, $65535
	JNE count_utf16le_westmere_continue
	MOVOU X2, X0
	HSUM_U16_X0_TO_AX
	ADDQ AX, R8
	PXOR X2, X2
	XORQ R9, R9

count_utf16le_westmere_continue:
	TESTQ CX, CX
	JNE count_utf16le_westmere_loop

	MOVOU X2, X0
	HSUM_U16_X0_TO_AX
	ADDQ AX, R8

count_utf16le_westmere_done:
	MOVQ R8, count+24(FP)
	RET

// func countUTF16BEBlocksWestmere(input []uint16) (count int)
// Same bytemask kernel after PSHUFB byte-swap for big-endian code units.
TEXT ·countUTF16BEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ R8, R8
	TESTQ CX, CX
	JE count_utf16be_westmere_done
	MOVOU ·utf16ByteSwapMask<>(SB), X4
	MOVOU ·utf16FC00<>(SB), X5
	MOVOU ·utf16DC00<>(SB), X6
	MOVOU ·utf16Ones<>(SB), X7
	PXOR X2, X2
	XORQ R9, R9

count_utf16be_westmere_loop:
	MOVOU 0(SI), X0
	PSHUFB X4, X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	MOVOU 16(SI), X0
	PSHUFB X4, X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	MOVOU 32(SI), X0
	PSHUFB X4, X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	MOVOU 48(SI), X0
	PSHUFB X4, X0
	PAND X5, X0
	PXOR X6, X0
	MOVOU X0, X1
	PXOR X3, X3
	PCMPEQW X3, X1
	PANDN X7, X1
	PADDW X1, X2

	ADDQ $64, SI
	SUBQ $32, CX
	INCQ R9
	CMPQ R9, $65535
	JNE count_utf16be_westmere_continue
	MOVOU X2, X0
	HSUM_U16_X0_TO_AX
	ADDQ AX, R8
	PXOR X2, X2
	XORQ R9, R9

count_utf16be_westmere_continue:
	TESTQ CX, CX
	JNE count_utf16be_westmere_loop

	MOVOU X2, X0
	HSUM_U16_X0_TO_AX
	ADDQ AX, R8

count_utf16be_westmere_done:
	MOVQ R8, count+24(FP)
	RET

// horizontalSumU16Y: sum 16 uint16 lanes in Y0 into AX (clobbers Y0,X1,X0).
#define HSUM_U16_Y0_TO_AX \
	VEXTRACTI128 $1, Y0, X1; \
	VPADDW X1, X0, X0; \
	MOVOU X0, X1; \
	PSRLDQ $8, X1; \
	PADDW X1, X0; \
	MOVOU X0, X1; \
	PSRLDQ $4, X1; \
	PADDW X1, X0; \
	MOVOU X0, X1; \
	PSRLDQ $2, X1; \
	PADDW X1, X0; \
	PEXTRW $0, X0, AX

// func countUTF16LEBlocksHaswell(input []uint16) (count int)
TEXT ·countUTF16LEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ R8, R8
	TESTQ CX, CX
	JE count_utf16le_haswell_done
	VMOVDQU ·utf16FC00<>(SB), Y5
	VMOVDQU ·utf16DC00<>(SB), Y6
	VMOVDQU ·utf16Ones<>(SB), Y7
	VPXOR Y2, Y2, Y2
	XORQ R9, R9

count_utf16le_haswell_loop:
	VMOVDQU 0(SI), Y0
	VPAND Y5, Y0, Y0
	VPXOR Y6, Y0, Y0
	VPMINUW Y7, Y0, Y0
	VPADDW Y0, Y2, Y2

	VMOVDQU 32(SI), Y0
	VPAND Y5, Y0, Y0
	VPXOR Y6, Y0, Y0
	VPMINUW Y7, Y0, Y0
	VPADDW Y0, Y2, Y2

	ADDQ $64, SI
	SUBQ $32, CX
	INCQ R9
	CMPQ R9, $65535
	JNE count_utf16le_haswell_continue
	VMOVDQU Y2, Y0
	HSUM_U16_Y0_TO_AX
	ADDQ AX, R8
	VPXOR Y2, Y2, Y2
	XORQ R9, R9

count_utf16le_haswell_continue:
	TESTQ CX, CX
	JNE count_utf16le_haswell_loop

	VMOVDQU Y2, Y0
	HSUM_U16_Y0_TO_AX
	ADDQ AX, R8

count_utf16le_haswell_done:
	VZEROUPPER
	MOVQ R8, count+24(FP)
	RET

// func countUTF16BEBlocksHaswell(input []uint16) (count int)
TEXT ·countUTF16BEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ R8, R8
	TESTQ CX, CX
	JE count_utf16be_haswell_done
	VMOVDQU ·utf16ByteSwapMask<>(SB), Y4
	VMOVDQU ·utf16FC00<>(SB), Y5
	VMOVDQU ·utf16DC00<>(SB), Y6
	VMOVDQU ·utf16Ones<>(SB), Y7
	VPXOR Y2, Y2, Y2
	XORQ R9, R9

count_utf16be_haswell_loop:
	VMOVDQU 0(SI), Y0
	VPSHUFB Y4, Y0, Y0
	VPAND Y5, Y0, Y0
	VPXOR Y6, Y0, Y0
	VPMINUW Y7, Y0, Y0
	VPADDW Y0, Y2, Y2

	VMOVDQU 32(SI), Y0
	VPSHUFB Y4, Y0, Y0
	VPAND Y5, Y0, Y0
	VPXOR Y6, Y0, Y0
	VPMINUW Y7, Y0, Y0
	VPADDW Y0, Y2, Y2

	ADDQ $64, SI
	SUBQ $32, CX
	INCQ R9
	CMPQ R9, $65535
	JNE count_utf16be_haswell_continue
	VMOVDQU Y2, Y0
	HSUM_U16_Y0_TO_AX
	ADDQ AX, R8
	VPXOR Y2, Y2, Y2
	XORQ R9, R9

count_utf16be_haswell_continue:
	TESTQ CX, CX
	JNE count_utf16be_haswell_loop

	VMOVDQU Y2, Y0
	HSUM_U16_Y0_TO_AX
	ADDQ AX, R8

count_utf16be_haswell_done:
	VZEROUPPER
	MOVQ R8, count+24(FP)
	RET
