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

// Independently translated from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf32.cpp
// and src/haswell/avx2_convert_utf16_to_utf32.cpp.
// Each kernel converts only complete no-surrogate vector groups
// (8 uint16 Westmere / 16 uint16 Haswell) with PMOVZXWD / VPMOVZXWD zero-extend
// stores and returns the number of uint16 code units consumed. The first vector
// containing a surrogate word (or a short remainder) stops the loop so Go
// wrappers can remount the scalar oracle without writing past the validated
// prefix.

// Byte-swap mask for PSHUFB / VPSHUFB (duplicated to avoid cross-file DATA deps).
DATA ·utf16UTF32ByteSwapMask<>(SB)/8, $0x0607040502030001
DATA ·utf16UTF32ByteSwapMask<>+8(SB)/8, $0x0e0f0c0d0a0b0809
DATA ·utf16UTF32ByteSwapMask<>+16(SB)/8, $0x0607040502030001
DATA ·utf16UTF32ByteSwapMask<>+24(SB)/8, $0x0e0f0c0d0a0b0809
GLOBL ·utf16UTF32ByteSwapMask<>(SB), RODATA|NOPTR, $32

// (in & 0xf800) == 0xd800 surrogate lane probe constants.
DATA ·utf16UTF32F800<>(SB)/8, $0xf800f800f800f800
DATA ·utf16UTF32F800<>+8(SB)/8, $0xf800f800f800f800
DATA ·utf16UTF32F800<>+16(SB)/8, $0xf800f800f800f800
DATA ·utf16UTF32F800<>+24(SB)/8, $0xf800f800f800f800
GLOBL ·utf16UTF32F800<>(SB), RODATA|NOPTR, $32

DATA ·utf16UTF32D800<>(SB)/8, $0xd800d800d800d800
DATA ·utf16UTF32D800<>+8(SB)/8, $0xd800d800d800d800
DATA ·utf16UTF32D800<>+16(SB)/8, $0xd800d800d800d800
DATA ·utf16UTF32D800<>+24(SB)/8, $0xd800d800d800d800
GLOBL ·utf16UTF32D800<>(SB), RODATA|NOPTR, $32

// func utf16LEToUTF32BlocksWestmere(input []uint16, dst []uint32) (consumed int)
// Processes complete 8-uint16 (16-byte) no-surrogate groups with SSE4.1 PMOVZXWD.
TEXT ·utf16LEToUTF32BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	MOVOU ·utf16UTF32F800<>(SB), X6
	MOVOU ·utf16UTF32D800<>(SB), X7

loop_utf16LEToUTF32BlocksWestmere:
	CMPQ CX, $8
	JB done_utf16LEToUTF32BlocksWestmere
	MOVOU (SI), X0
	MOVOU X0, X1
	PAND X6, X1
	PCMPEQW X7, X1
	PMOVMSKB X1, DX
	TESTL DX, DX
	JNE done_utf16LEToUTF32BlocksWestmere
	// Zero-extend eight uint16 → eight uint32 (two 128-bit stores).
	MOVOU X0, X1
	PMOVZXWD X0, X2
	PSRLDQ $8, X1
	PMOVZXWD X1, X3
	MOVOU X2, (DI)
	MOVOU X3, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	ADDQ $8, AX
	SUBQ $8, CX
	JMP loop_utf16LEToUTF32BlocksWestmere

done_utf16LEToUTF32BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf16BEToUTF32BlocksWestmere(input []uint16, dst []uint32) (consumed int)
TEXT ·utf16BEToUTF32BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	MOVOU ·utf16UTF32ByteSwapMask<>(SB), X5
	MOVOU ·utf16UTF32F800<>(SB), X6
	MOVOU ·utf16UTF32D800<>(SB), X7

loop_utf16BEToUTF32BlocksWestmere:
	CMPQ CX, $8
	JB done_utf16BEToUTF32BlocksWestmere
	MOVOU (SI), X0
	PSHUFB X5, X0
	MOVOU X0, X1
	PAND X6, X1
	PCMPEQW X7, X1
	PMOVMSKB X1, DX
	TESTL DX, DX
	JNE done_utf16BEToUTF32BlocksWestmere
	MOVOU X0, X1
	PMOVZXWD X0, X2
	PSRLDQ $8, X1
	PMOVZXWD X1, X3
	MOVOU X2, (DI)
	MOVOU X3, 16(DI)
	ADDQ $16, SI
	ADDQ $32, DI
	ADDQ $8, AX
	SUBQ $8, CX
	JMP loop_utf16BEToUTF32BlocksWestmere

done_utf16BEToUTF32BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf16LEToUTF32BlocksHaswell(input []uint16, dst []uint32) (consumed int)
// Processes complete 16-uint16 (32-byte) no-surrogate groups with AVX2 VPMOVZXWD.
TEXT ·utf16LEToUTF32BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	VMOVDQU ·utf16UTF32F800<>(SB), Y6
	VMOVDQU ·utf16UTF32D800<>(SB), Y7

loop_utf16LEToUTF32BlocksHaswell:
	CMPQ CX, $16
	JB done_utf16LEToUTF32BlocksHaswell
	VMOVDQU (SI), Y0
	VPAND Y6, Y0, Y1
	VPCMPEQW Y7, Y1, Y1
	VPMOVMSKB Y1, DX
	TESTL DX, DX
	JNE done_utf16LEToUTF32BlocksHaswell
	// Zero-extend sixteen uint16 → sixteen uint32 (two 256-bit stores).
	VPMOVZXWD X0, Y2
	VEXTRACTI128 $1, Y0, X1
	VPMOVZXWD X1, Y3
	VMOVDQU Y2, (DI)
	VMOVDQU Y3, 32(DI)
	ADDQ $32, SI
	ADDQ $64, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf16LEToUTF32BlocksHaswell

done_utf16LEToUTF32BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// func utf16BEToUTF32BlocksHaswell(input []uint16, dst []uint32) (consumed int)
TEXT ·utf16BEToUTF32BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	VMOVDQU ·utf16UTF32ByteSwapMask<>(SB), Y5
	VMOVDQU ·utf16UTF32F800<>(SB), Y6
	VMOVDQU ·utf16UTF32D800<>(SB), Y7

loop_utf16BEToUTF32BlocksHaswell:
	CMPQ CX, $16
	JB done_utf16BEToUTF32BlocksHaswell
	VMOVDQU (SI), Y0
	VPSHUFB Y5, Y0, Y0
	VPAND Y6, Y0, Y1
	VPCMPEQW Y7, Y1, Y1
	VPMOVMSKB Y1, DX
	TESTL DX, DX
	JNE done_utf16BEToUTF32BlocksHaswell
	VPMOVZXWD X0, Y2
	VEXTRACTI128 $1, Y0, X1
	VPMOVZXWD X1, Y3
	VMOVDQU Y2, (DI)
	VMOVDQU Y3, 32(DI)
	ADDQ $32, SI
	ADDQ $64, DI
	ADDQ $16, AX
	SUBQ $16, CX
	JMP loop_utf16BEToUTF32BlocksHaswell

done_utf16BEToUTF32BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET
