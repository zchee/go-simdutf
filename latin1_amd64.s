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

TEXT ·latin1UTF8ASCIIBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX
	loop_latin1UTF8ASCIIBlocksWestmere: CMPQ CX, $16; JB done_latin1UTF8ASCIIBlocksWestmere; MOVOU (SI), X0; PXOR X1, X1; MOVOU X0, X2; PCMPGTB X0, X1; PMOVMSKB X1, DX; TESTQ DX, DX; JNE done_latin1UTF8ASCIIBlocksWestmere; MOVOU X2, (DI); ADDQ $16, SI; ADDQ $16, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF8ASCIIBlocksWestmere
	done_latin1UTF8ASCIIBlocksWestmere: MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF8ASCIIBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX
	loop_latin1UTF8ASCIIBlocksHaswell: CMPQ CX, $32; JB done_latin1UTF8ASCIIBlocksHaswell; VMOVDQU (SI), Y0; VPXOR Y1, Y1, Y1; VPCMPGTB Y0, Y1, Y1; VPMOVMSKB Y1, DX; TESTQ DX, DX; JNE done_latin1UTF8ASCIIBlocksHaswell; VMOVDQU Y0, (DI); ADDQ $32, SI; ADDQ $32, DI; ADDQ $32, AX; SUBQ $32, CX; JMP loop_latin1UTF8ASCIIBlocksHaswell
	done_latin1UTF8ASCIIBlocksHaswell: VZEROUPPER; MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF16LEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX
	loop_latin1UTF16LEBlocksWestmere: CMPQ CX, $16; JB done_latin1UTF16LEBlocksWestmere; MOVOU (SI), X0; PXOR X7, X7; MOVOU X0, X1; PUNPCKLBW X7, X0; PUNPCKHBW X7, X1; MOVOU X0, (DI); MOVOU X1, 16(DI); ADDQ $16, SI; ADDQ $32, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF16LEBlocksWestmere
	done_latin1UTF16LEBlocksWestmere: MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF16BEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX
	loop_latin1UTF16BEBlocksWestmere: CMPQ CX, $16; JB done_latin1UTF16BEBlocksWestmere; MOVOU (SI), X0; PXOR X7, X7; MOVOU X0, X1; PUNPCKLBW X7, X0; PUNPCKHBW X7, X1; MOVOU X0, X2; PSLLW $8, X0; PSRLW $8, X2; POR X2, X0; MOVOU X1, X2; PSLLW $8, X1; PSRLW $8, X2; POR X2, X1; MOVOU X0, (DI); MOVOU X1, 16(DI); ADDQ $16, SI; ADDQ $32, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF16BEBlocksWestmere
	done_latin1UTF16BEBlocksWestmere: MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF16LEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX
	loop_latin1UTF16LEBlocksHaswell: CMPQ CX, $16; JB done_latin1UTF16LEBlocksHaswell; MOVOU (SI), X0; PXOR X7, X7; MOVOU X0, X1; PUNPCKLBW X7, X0; PUNPCKHBW X7, X1; MOVOU X0, (DI); MOVOU X1, 16(DI); ADDQ $16, SI; ADDQ $32, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF16LEBlocksHaswell
	done_latin1UTF16LEBlocksHaswell: VZEROUPPER; MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF16BEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX
	loop_latin1UTF16BEBlocksHaswell: CMPQ CX, $16; JB done_latin1UTF16BEBlocksHaswell; MOVOU (SI), X0; PXOR X7, X7; MOVOU X0, X1; PUNPCKLBW X7, X0; PUNPCKHBW X7, X1; MOVOU X0, X2; PSLLW $8, X0; PSRLW $8, X2; POR X2, X0; MOVOU X1, X2; PSLLW $8, X1; PSRLW $8, X2; POR X2, X1; MOVOU X0, (DI); MOVOU X1, 16(DI); ADDQ $16, SI; ADDQ $32, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF16BEBlocksHaswell
	done_latin1UTF16BEBlocksHaswell: VZEROUPPER; MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF32BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX; PXOR X7, X7
	loop_latin1UTF32BlocksWestmere: CMPQ CX, $16; JB done_latin1UTF32BlocksWestmere; MOVOU (SI), X0; MOVOU X0, X1; PUNPCKLBW X7, X0; PUNPCKHBW X7, X1; MOVOU X0, X2; MOVOU X1, X3; PUNPCKLWL X7, X0; PUNPCKHWL X7, X2; PUNPCKLWL X7, X1; PUNPCKHWL X7, X3; MOVOU X0, (DI); MOVOU X2, 16(DI); MOVOU X1, 32(DI); MOVOU X3, 48(DI); ADDQ $16, SI; ADDQ $64, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF32BlocksWestmere
	done_latin1UTF32BlocksWestmere: MOVQ AX, consumed+48(FP); RET

TEXT ·latin1UTF32BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; MOVQ dst_base+24(FP), DI; XORQ AX, AX; PXOR X7, X7
	loop_latin1UTF32BlocksHaswell: CMPQ CX, $16; JB done_latin1UTF32BlocksHaswell; MOVOU (SI), X0; MOVOU X0, X1; PUNPCKLBW X7, X0; PUNPCKHBW X7, X1; MOVOU X0, X2; MOVOU X1, X3; PUNPCKLWL X7, X0; PUNPCKHWL X7, X2; PUNPCKLWL X7, X1; PUNPCKHWL X7, X3; MOVOU X0, (DI); MOVOU X2, 16(DI); MOVOU X1, 32(DI); MOVOU X3, 48(DI); ADDQ $16, SI; ADDQ $64, DI; ADDQ $16, AX; SUBQ $16, CX; JMP loop_latin1UTF32BlocksHaswell
	done_latin1UTF32BlocksHaswell: VZEROUPPER; MOVQ AX, consumed+48(FP); RET

TEXT ·latin1HighByteBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; ANDQ $-64, CX; XORQ AX, AX
	loop_latin1HighByteBlocksWestmere: TESTQ CX, CX; JE done_latin1HighByteBlocksWestmere; MOVOU (SI), X0; PXOR X7, X7; PCMPGTB X0, X7; PMOVMSKB X7, DX; POPCNTQ DX, DX; ADDQ DX, AX; ADDQ $16, SI; SUBQ $16, CX; JMP loop_latin1HighByteBlocksWestmere
	done_latin1HighByteBlocksWestmere: MOVQ AX, count+24(FP); RET

TEXT ·latin1HighByteBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ input_base+0(FP), SI; MOVQ input_len+8(FP), CX; ANDQ $-128, CX; XORQ AX, AX
	loop_latin1HighByteBlocksHaswell: TESTQ CX, CX; JE done_latin1HighByteBlocksHaswell; VMOVDQU (SI), Y0; VPXOR Y7, Y7, Y7; VPCMPGTB Y0, Y7, Y1; VPMOVMSKB Y1, DX; POPCNTQ DX, DX; ADDQ DX, AX; ADDQ $32, SI; SUBQ $32, CX; JMP loop_latin1HighByteBlocksHaswell
	done_latin1HighByteBlocksHaswell: VZEROUPPER; MOVQ AX, count+24(FP); RET

