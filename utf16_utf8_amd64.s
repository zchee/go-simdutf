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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_utf8.cpp
// and src/haswell/avx2_convert_utf16_to_utf8.cpp (ASCII fast path only).
// Each kernel converts only complete ASCII vector groups
// (8 uint16 Westmere / 16 uint16 Haswell) via PACKUSWB and returns the number
// of uint16 code units consumed (== UTF-8 bytes written). The first non-ASCII
// vector (or short remainder) stops the loop so Go wrappers can remount the
// scalar oracle for non-ASCII / surrogate / incomplete blocks.

// Byte-swap mask for PSHUFB / VPSHUFB (duplicated to avoid cross-file DATA deps).
DATA ·utf16UTF8ByteSwapMask<>(SB)/8, $0x0607040502030001
DATA ·utf16UTF8ByteSwapMask<>+8(SB)/8, $0x0e0f0c0d0a0b0809
DATA ·utf16UTF8ByteSwapMask<>+16(SB)/8, $0x0607040502030001
DATA ·utf16UTF8ByteSwapMask<>+24(SB)/8, $0x0e0f0c0d0a0b0809
GLOBL ·utf16UTF8ByteSwapMask<>(SB), RODATA|NOPTR, $32

// ASCII probe: (word & 0xff80) == 0.
DATA ·utf16UTF8FF80<>(SB)/8, $0xff80ff80ff80ff80
DATA ·utf16UTF8FF80<>+8(SB)/8, $0xff80ff80ff80ff80
DATA ·utf16UTF8FF80<>+16(SB)/8, $0xff80ff80ff80ff80
DATA ·utf16UTF8FF80<>+24(SB)/8, $0xff80ff80ff80ff80
GLOBL ·utf16UTF8FF80<>(SB), RODATA|NOPTR, $32

// func utf16LEToUTF8ASCIIBlocksWestmere(input []uint16, dst []byte) (consumed int)
// Processes complete 8-uint16 ASCII-only groups with SSE2 PACKUSWB.
TEXT ·utf16LEToUTF8ASCIIBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	MOVQ  dst_base+24(FP), DI
	XORQ  AX, AX
	MOVOU ·utf16UTF8FF80<>(SB), X6

loop_utf16LEToUTF8ASCIIBlocksWestmere:
	CMPQ     CX, $8
	JB       done_utf16LEToUTF8ASCIIBlocksWestmere
	MOVOU    (SI), X0
	MOVOU    X0, X1
	PAND     X6, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	CMPL     DX, $0xffff
	JNE      done_utf16LEToUTF8ASCIIBlocksWestmere
	PACKUSWB X0, X0
	MOVQ     X0, (DI)
	ADDQ     $16, SI
	ADDQ     $8, DI
	ADDQ     $8, AX
	SUBQ     $8, CX
	JMP      loop_utf16LEToUTF8ASCIIBlocksWestmere

done_utf16LEToUTF8ASCIIBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf16BEToUTF8ASCIIBlocksWestmere(input []uint16, dst []byte) (consumed int)
TEXT ·utf16BEToUTF8ASCIIBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	MOVQ  dst_base+24(FP), DI
	XORQ  AX, AX
	MOVOU ·utf16UTF8ByteSwapMask<>(SB), X5
	MOVOU ·utf16UTF8FF80<>(SB), X6

loop_utf16BEToUTF8ASCIIBlocksWestmere:
	CMPQ     CX, $8
	JB       done_utf16BEToUTF8ASCIIBlocksWestmere
	MOVOU    (SI), X0
	PSHUFB   X5, X0
	MOVOU    X0, X1
	PAND     X6, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	CMPL     DX, $0xffff
	JNE      done_utf16BEToUTF8ASCIIBlocksWestmere
	PACKUSWB X0, X0
	MOVQ     X0, (DI)
	ADDQ     $16, SI
	ADDQ     $8, DI
	ADDQ     $8, AX
	SUBQ     $8, CX
	JMP      loop_utf16BEToUTF8ASCIIBlocksWestmere

done_utf16BEToUTF8ASCIIBlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf16LEToUTF8ASCIIBlocksHaswell(input []uint16, dst []byte) (consumed int)
// Processes complete 16-uint16 ASCII-only groups with AVX2.
TEXT ·utf16LEToUTF8ASCIIBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), CX
	MOVQ    dst_base+24(FP), DI
	XORQ    AX, AX
	VMOVDQU ·utf16UTF8FF80<>(SB), Y6

loop_utf16LEToUTF8ASCIIBlocksHaswell:
	CMPQ         CX, $16
	JB           done_utf16LEToUTF8ASCIIBlocksHaswell
	VMOVDQU      (SI), Y0
	VPAND        Y6, Y0, Y1
	VPXOR        Y2, Y2, Y2
	VPCMPEQW     Y2, Y1, Y1
	VPMOVMSKB    Y1, DX
	CMPL         DX, $0xffffffff
	JNE          done_utf16LEToUTF8ASCIIBlocksHaswell
	VEXTRACTI128 $1, Y0, X1
	VPACKUSWB    X1, X0, X0
	VMOVDQU      X0, (DI)
	ADDQ         $32, SI
	ADDQ         $16, DI
	ADDQ         $16, AX
	SUBQ         $16, CX
	JMP          loop_utf16LEToUTF8ASCIIBlocksHaswell

done_utf16LEToUTF8ASCIIBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// func utf16BEToUTF8ASCIIBlocksHaswell(input []uint16, dst []byte) (consumed int)
TEXT ·utf16BEToUTF8ASCIIBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), CX
	MOVQ    dst_base+24(FP), DI
	XORQ    AX, AX
	VMOVDQU ·utf16UTF8ByteSwapMask<>(SB), Y5
	VMOVDQU ·utf16UTF8FF80<>(SB), Y6

loop_utf16BEToUTF8ASCIIBlocksHaswell:
	CMPQ         CX, $16
	JB           done_utf16BEToUTF8ASCIIBlocksHaswell
	VMOVDQU      (SI), Y0
	VPSHUFB      Y5, Y0, Y0
	VPAND        Y6, Y0, Y1
	VPXOR        Y2, Y2, Y2
	VPCMPEQW     Y2, Y1, Y1
	VPMOVMSKB    Y1, DX
	CMPL         DX, $0xffffffff
	JNE          done_utf16BEToUTF8ASCIIBlocksHaswell
	VEXTRACTI128 $1, Y0, X1
	VPACKUSWB    X1, X0, X0
	VMOVDQU      X0, (DI)
	ADDQ         $32, SI
	ADDQ         $16, DI
	ADDQ         $16, AX
	SUBQ         $16, CX
	JMP          loop_utf16BEToUTF8ASCIIBlocksHaswell

done_utf16BEToUTF8ASCIIBlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET
