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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf32_to_*.cpp
// and src/haswell/avx2_convert_utf32_to_*.cpp.
// Each kernel converts only complete fast-path vector groups
// (4 uint32 Westmere / 8 uint32 Haswell) and returns the number of uint32
// code units consumed. The first vector that fails the fast-path predicate
// (or a short remainder) stops the loop so Go wrappers can remount scalar.

// PSHUFB mask: extract bytes 0,4,8,12 from four uint32 lanes.
DATA ·utf32ByteExtract<>(SB)/8, $0xffffffff0c080400
DATA ·utf32ByteExtract<>+8(SB)/8, $0xffffffffffffffff
DATA ·utf32ByteExtract<>+16(SB)/8, $0xffffffff0c080400
DATA ·utf32ByteExtract<>+24(SB)/8, $0xffffffffffffffff
GLOBL ·utf32ByteExtract<>(SB), RODATA|NOPTR, $32

// Byte-swap mask for packed UTF-16 words (PSHUFB / VPSHUFB).
DATA ·utf32UTF16ByteSwap<>(SB)/8, $0x0607040502030001
DATA ·utf32UTF16ByteSwap<>+8(SB)/8, $0x0e0f0c0d0a0b0809
DATA ·utf32UTF16ByteSwap<>+16(SB)/8, $0x0607040502030001
DATA ·utf32UTF16ByteSwap<>+24(SB)/8, $0x0e0f0c0d0a0b0809
GLOBL ·utf32UTF16ByteSwap<>(SB), RODATA|NOPTR, $32

// Surrogate probe: (word & 0xfffff800) == 0xd800
DATA ·utf32F800<>(SB)/8, $0xfffff800fffff800
DATA ·utf32F800<>+8(SB)/8, $0xfffff800fffff800
DATA ·utf32F800<>+16(SB)/8, $0xfffff800fffff800
DATA ·utf32F800<>+24(SB)/8, $0xfffff800fffff800
GLOBL ·utf32F800<>(SB), RODATA|NOPTR, $32

DATA ·utf32D800<>(SB)/8, $0x0000d8000000d800
DATA ·utf32D800<>+8(SB)/8, $0x0000d8000000d800
DATA ·utf32D800<>+16(SB)/8, $0x0000d8000000d800
DATA ·utf32D800<>+24(SB)/8, $0x0000d8000000d800
GLOBL ·utf32D800<>(SB), RODATA|NOPTR, $32

// func utf32ToLatin1BlocksWestmere(input []uint32, dst []byte) (consumed int)
TEXT ·utf32ToLatin1BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	MOVOU ·utf32ByteExtract<>(SB), X7

loop_utf32ToLatin1BlocksWestmere:
	CMPQ CX, $4
	JB done_utf32ToLatin1BlocksWestmere
	MOVOU (SI), X0
	MOVOU X0, X1
	PSRLL $8, X1
	PXOR X2, X2
	PCMPEQL X2, X1
	PMOVMSKB X1, DX
	CMPL DX, $0xffff
	JNE done_utf32ToLatin1BlocksWestmere
	PSHUFB X7, X0
	MOVL X0, (DI)
	ADDQ $16, SI
	ADDQ $4, DI
	ADDQ $4, AX
	SUBQ $4, CX
	JMP loop_utf32ToLatin1BlocksWestmere

done_utf32ToLatin1BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToLatin1BlocksHaswell(input []uint32, dst []byte) (consumed int)
TEXT ·utf32ToLatin1BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	VMOVDQU ·utf32ByteExtract<>(SB), Y7

loop_utf32ToLatin1BlocksHaswell:
	CMPQ CX, $8
	JB done_utf32ToLatin1BlocksHaswell
	VMOVDQU (SI), Y0
	VPSRLD $8, Y0, Y1
	VPXOR Y2, Y2, Y2
	VPCMPEQD Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	CMPL DX, $0xffffffff
	JNE done_utf32ToLatin1BlocksHaswell
	VPSHUFB Y7, Y0, Y0
	VEXTRACTI128 $1, Y0, X1
	MOVL X0, (DI)
	MOVL X1, 4(DI)
	ADDQ $32, SI
	ADDQ $8, DI
	ADDQ $8, AX
	SUBQ $8, CX
	JMP loop_utf32ToLatin1BlocksHaswell

done_utf32ToLatin1BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToUTF8ASCIIBlocksWestmere(input []uint32, dst []byte) (consumed int)
TEXT ·utf32ToUTF8ASCIIBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	MOVOU ·utf32ByteExtract<>(SB), X7

loop_utf32ToUTF8ASCIIBlocksWestmere:
	CMPQ CX, $4
	JB done_utf32ToUTF8ASCIIBlocksWestmere
	MOVOU (SI), X0
	MOVOU X0, X1
	PSRLL $7, X1
	PXOR X2, X2
	PCMPEQL X2, X1
	PMOVMSKB X1, DX
	CMPL DX, $0xffff
	JNE done_utf32ToUTF8ASCIIBlocksWestmere
	PSHUFB X7, X0
	MOVL X0, (DI)
	ADDQ $16, SI
	ADDQ $4, DI
	ADDQ $4, AX
	SUBQ $4, CX
	JMP loop_utf32ToUTF8ASCIIBlocksWestmere

done_utf32ToUTF8ASCIIBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToUTF8ASCIIBlocksHaswell(input []uint32, dst []byte) (consumed int)
TEXT ·utf32ToUTF8ASCIIBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	VMOVDQU ·utf32ByteExtract<>(SB), Y7

loop_utf32ToUTF8ASCIIBlocksHaswell:
	CMPQ CX, $8
	JB done_utf32ToUTF8ASCIIBlocksHaswell
	VMOVDQU (SI), Y0
	VPSRLD $7, Y0, Y1
	VPXOR Y2, Y2, Y2
	VPCMPEQD Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	CMPL DX, $0xffffffff
	JNE done_utf32ToUTF8ASCIIBlocksHaswell
	VPSHUFB Y7, Y0, Y0
	VEXTRACTI128 $1, Y0, X1
	MOVL X0, (DI)
	MOVL X1, 4(DI)
	ADDQ $32, SI
	ADDQ $8, DI
	ADDQ $8, AX
	SUBQ $8, CX
	JMP loop_utf32ToUTF8ASCIIBlocksHaswell

done_utf32ToUTF8ASCIIBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToUTF16LEBlocksWestmere(input []uint32, dst []uint16) (consumed int)
TEXT ·utf32ToUTF16LEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	MOVOU ·utf32F800<>(SB), X6
	MOVOU ·utf32D800<>(SB), X7

loop_utf32ToUTF16LEBlocksWestmere:
	CMPQ CX, $4
	JB done_utf32ToUTF16LEBlocksWestmere
	MOVOU (SI), X0
	// Require high 16 bits zero (BMP).
	MOVOU X0, X1
	PSRLL $16, X1
	PXOR X2, X2
	PCMPEQL X2, X1
	PMOVMSKB X1, DX
	CMPL DX, $0xffff
	JNE done_utf32ToUTF16LEBlocksWestmere
	// Reject surrogates: (word & 0xfffff800) == 0xd800.
	MOVOU X0, X1
	PAND X6, X1
	PCMPEQL X7, X1
	PMOVMSKB X1, DX
	TESTL DX, DX
	JNE done_utf32ToUTF16LEBlocksWestmere
	PXOR X1, X1
	PACKUSDW X1, X0
	MOVQ X0, (DI)
	ADDQ $16, SI
	ADDQ $8, DI
	ADDQ $4, AX
	SUBQ $4, CX
	JMP loop_utf32ToUTF16LEBlocksWestmere

done_utf32ToUTF16LEBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToUTF16BEBlocksWestmere(input []uint32, dst []uint16) (consumed int)
TEXT ·utf32ToUTF16BEBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	MOVOU ·utf32UTF16ByteSwap<>(SB), X5
	MOVOU ·utf32F800<>(SB), X6
	MOVOU ·utf32D800<>(SB), X7

loop_utf32ToUTF16BEBlocksWestmere:
	CMPQ CX, $4
	JB done_utf32ToUTF16BEBlocksWestmere
	MOVOU (SI), X0
	MOVOU X0, X1
	PSRLL $16, X1
	PXOR X2, X2
	PCMPEQL X2, X1
	PMOVMSKB X1, DX
	CMPL DX, $0xffff
	JNE done_utf32ToUTF16BEBlocksWestmere
	MOVOU X0, X1
	PAND X6, X1
	PCMPEQL X7, X1
	PMOVMSKB X1, DX
	TESTL DX, DX
	JNE done_utf32ToUTF16BEBlocksWestmere
	PXOR X1, X1
	PACKUSDW X1, X0
	PSHUFB X5, X0
	MOVQ X0, (DI)
	ADDQ $16, SI
	ADDQ $8, DI
	ADDQ $4, AX
	SUBQ $4, CX
	JMP loop_utf32ToUTF16BEBlocksWestmere

done_utf32ToUTF16BEBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToUTF16LEBlocksHaswell(input []uint32, dst []uint16) (consumed int)
TEXT ·utf32ToUTF16LEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	VMOVDQU ·utf32F800<>(SB), Y6
	VMOVDQU ·utf32D800<>(SB), Y7

loop_utf32ToUTF16LEBlocksHaswell:
	CMPQ CX, $8
	JB done_utf32ToUTF16LEBlocksHaswell
	VMOVDQU (SI), Y0
	VPSRLD $16, Y0, Y1
	VPXOR Y2, Y2, Y2
	VPCMPEQD Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	CMPL DX, $0xffffffff
	JNE done_utf32ToUTF16LEBlocksHaswell
	VPAND Y6, Y0, Y1
	VPCMPEQD Y7, Y1, Y1
	VPMOVMSKB Y1, DX
	TESTL DX, DX
	JNE done_utf32ToUTF16LEBlocksHaswell
	// Pack each 128-bit half: 4 dwords -> 4 words.
	VEXTRACTI128 $1, Y0, X1
	PXOR X2, X2
	PACKUSDW X2, X0
	PACKUSDW X2, X1
	MOVQ X0, (DI)
	MOVQ X1, 8(DI)
	ADDQ $32, SI
	ADDQ $16, DI
	ADDQ $8, AX
	SUBQ $8, CX
	JMP loop_utf32ToUTF16LEBlocksHaswell

done_utf32ToUTF16LEBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// func utf32ToUTF16BEBlocksHaswell(input []uint32, dst []uint16) (consumed int)
TEXT ·utf32ToUTF16BEBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX
	VMOVDQU ·utf32F800<>(SB), Y6
	VMOVDQU ·utf32D800<>(SB), Y7

loop_utf32ToUTF16BEBlocksHaswell:
	CMPQ CX, $8
	JB done_utf32ToUTF16BEBlocksHaswell
	VMOVDQU (SI), Y0
	VPSRLD $16, Y0, Y1
	VPXOR Y2, Y2, Y2
	VPCMPEQD Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	CMPL DX, $0xffffffff
	JNE done_utf32ToUTF16BEBlocksHaswell
	VPAND Y6, Y0, Y1
	VPCMPEQD Y7, Y1, Y1
	VPMOVMSKB Y1, DX
	TESTL DX, DX
	JNE done_utf32ToUTF16BEBlocksHaswell
	VEXTRACTI128 $1, Y0, X1
	PXOR X2, X2
	PACKUSDW X2, X0
	PACKUSDW X2, X1
	MOVOU ·utf32UTF16ByteSwap<>(SB), X3
	PSHUFB X3, X0
	PSHUFB X3, X1
	MOVQ X0, (DI)
	MOVQ X1, 8(DI)
	ADDQ $32, SI
	ADDQ $16, DI
	ADDQ $8, AX
	SUBQ $8, CX
	JMP loop_utf32ToUTF16BEBlocksHaswell

done_utf32ToUTF16BEBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET
