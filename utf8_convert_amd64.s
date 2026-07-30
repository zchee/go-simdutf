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

//go:build amd64

#include "textflag.h"

// ASCII-run accelerators for UTF-8 source conversion. Each routine consumes
// complete in-bounds vector groups of ASCII bytes only and returns the number
// of input bytes consumed. The first non-ASCII byte (or short remainder) stops
// the loop so Go wrappers can re-enter the scalar oracle.

TEXT ·utf8ASCIIToLatin1BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
loop_utf8ASCIIToLatin1BlocksWestmere:
	CMPQ CX, $16
	JB done_utf8ASCIIToLatin1BlocksWestmere
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToLatin1BlocksWestmere
	MOVOU X2, (DI)
	ADDQ $16, SI
	ADDQ $16, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToLatin1BlocksWestmere
done_utf8ASCIIToLatin1BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToLatin1BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
loop_utf8ASCIIToLatin1BlocksHaswell:
	CMPQ CX, $32
	JB done_utf8ASCIIToLatin1BlocksHaswell
	VMOVDQU (SI), Y0
	VPXOR Y1, Y1, Y1
	VPCMPGTB Y0, Y1, Y1
	VPMOVMSKB Y1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToLatin1BlocksHaswell
	VMOVDQU Y0, (DI)
	ADDQ $32, SI
	ADDQ $32, DI
	ADDQ $32, AX
	SUBQ $32, CX
	JMP loop_utf8ASCIIToLatin1BlocksHaswell
done_utf8ASCIIToLatin1BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToUTF16LEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
loop_utf8ASCIIToUTF16LEBlocksWestmere:
	CMPQ CX, $16
	JB done_utf8ASCIIToUTF16LEBlocksWestmere
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToUTF16LEBlocksWestmere

	PXOR X7, X7
	MOVOU X2, X0
	MOVOU X2, X1
	PUNPCKLBW X7, X0
	PUNPCKHBW X7, X1

	MOVOU X0, (DI)
	MOVOU X1, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToUTF16LEBlocksWestmere
done_utf8ASCIIToUTF16LEBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToUTF16BEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
loop_utf8ASCIIToUTF16BEBlocksWestmere:
	CMPQ CX, $16
	JB done_utf8ASCIIToUTF16BEBlocksWestmere
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToUTF16BEBlocksWestmere

	PXOR X7, X7
	MOVOU X2, X0
	MOVOU X2, X1
	PUNPCKLBW X7, X0
	PUNPCKHBW X7, X1

	MOVOU X0, X3
	PSLLW $8, X0
	PSRLW $8, X3
	POR X3, X0
	MOVOU X1, X3
	PSLLW $8, X1
	PSRLW $8, X3
	POR X3, X1

	MOVOU X0, (DI)
	MOVOU X1, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToUTF16BEBlocksWestmere
done_utf8ASCIIToUTF16BEBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToUTF16LEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
loop_utf8ASCIIToUTF16LEBlocksHaswell:
	CMPQ CX, $16
	JB done_utf8ASCIIToUTF16LEBlocksHaswell
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToUTF16LEBlocksHaswell

	PXOR X7, X7
	MOVOU X2, X0
	MOVOU X2, X1
	PUNPCKLBW X7, X0
	PUNPCKHBW X7, X1

	MOVOU X0, (DI)
	MOVOU X1, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToUTF16LEBlocksHaswell
done_utf8ASCIIToUTF16LEBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToUTF16BEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
loop_utf8ASCIIToUTF16BEBlocksHaswell:
	CMPQ CX, $16
	JB done_utf8ASCIIToUTF16BEBlocksHaswell
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToUTF16BEBlocksHaswell

	PXOR X7, X7
	MOVOU X2, X0
	MOVOU X2, X1
	PUNPCKLBW X7, X0
	PUNPCKHBW X7, X1

	MOVOU X0, X3
	PSLLW $8, X0
	PSRLW $8, X3
	POR X3, X0
	MOVOU X1, X3
	PSLLW $8, X1
	PSRLW $8, X3
	POR X3, X1

	MOVOU X0, (DI)
	MOVOU X1, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToUTF16BEBlocksHaswell
done_utf8ASCIIToUTF16BEBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToUTF32BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	PXOR X7, X7
loop_utf8ASCIIToUTF32BlocksWestmere:
	CMPQ CX, $16
	JB done_utf8ASCIIToUTF32BlocksWestmere
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToUTF32BlocksWestmere
	MOVOU X2, X0
	MOVOU X2, X1
	PUNPCKLBW X7, X0
	PUNPCKHBW X7, X1
	MOVOU X0, X2
	MOVOU X1, X3
	PUNPCKLWL X7, X0
	PUNPCKHWL X7, X2
	PUNPCKLWL X7, X1
	PUNPCKHWL X7, X3
	MOVOU X0, (DI)
	MOVOU X2, 16(DI)
	MOVOU X1, 32(DI)
	MOVOU X3, 48(DI)
	ADDQ $16, SI
	ADDQ $64, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToUTF32BlocksWestmere
done_utf8ASCIIToUTF32BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

TEXT ·utf8ASCIIToUTF32BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	PXOR X7, X7
loop_utf8ASCIIToUTF32BlocksHaswell:
	CMPQ CX, $16
	JB done_utf8ASCIIToUTF32BlocksHaswell
	MOVOU (SI), X0
	PXOR X1, X1
	MOVOU X0, X2
	PCMPGTB X0, X1
	PMOVMSKB X1, DX
	TESTQ DX, DX
	JNE done_utf8ASCIIToUTF32BlocksHaswell
	MOVOU X2, X0
	MOVOU X2, X1
	PUNPCKLBW X7, X0
	PUNPCKHBW X7, X1
	MOVOU X0, X2
	MOVOU X1, X3
	PUNPCKLWL X7, X0
	PUNPCKHWL X7, X2
	PUNPCKLWL X7, X1
	PUNPCKHWL X7, X3
	MOVOU X0, (DI)
	MOVOU X2, 16(DI)
	MOVOU X1, 32(DI)
	MOVOU X3, 48(DI)
	ADDQ $16, SI
	ADDQ $64, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf8ASCIIToUTF32BlocksHaswell
done_utf8ASCIIToUTF32BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

