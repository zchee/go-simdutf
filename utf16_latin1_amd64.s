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
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/westmere/sse_convert_utf16_to_latin1.cpp
// and src/haswell/avx2_convert_utf16_to_latin1.cpp (with_errors 16-uint16 Haswell stride).
// Each kernel converts only complete latin1-only vector groups and returns the
// number of uint16 code units consumed. The first non-latin1 vector (or short
// remainder) stops the loop so Go wrappers can re-enter the scalar oracle.

// Byte-swap mask for PSHUFB / VPSHUFB (duplicated to avoid cross-file DATA deps).
DATA ·utf16Latin1ByteSwapMask<>(SB)/8, $0x0607040502030001
DATA ·utf16Latin1ByteSwapMask<>+8(SB)/8, $0x0e0f0c0d0a0b0809
DATA ·utf16Latin1ByteSwapMask<>+16(SB)/8, $0x0607040502030001
DATA ·utf16Latin1ByteSwapMask<>+24(SB)/8, $0x0e0f0c0d0a0b0809
GLOBL ·utf16Latin1ByteSwapMask<>(SB), RODATA|NOPTR, $32

// func utf16LEToLatin1BlocksWestmere(input []uint16, dst []byte) (consumed int)
// Processes complete 8-uint16 (16-byte) latin1-only groups with SSE2 PACKUSWB.
TEXT ·utf16LEToLatin1BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX

loop_utf16LEToLatin1BlocksWestmere:
	CMPQ  CX, $8
	JB    done_utf16LEToLatin1BlocksWestmere
	MOVOU (SI), X0

	// Isolate high bytes into low word lanes, then require every word == 0.
	// (PSRLW+PMOVMSKB alone only sees the MSB of each high byte.)
	MOVOU    X0, X1
	PSRLW    $8, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	CMPL     DX, $0xffff
	JNE      done_utf16LEToLatin1BlocksWestmere
	PACKUSWB X0, X0
	MOVQ     X0, (DI)
	ADDQ     $16, SI
	ADDQ     $8, DI
	ADDQ     $8, AX
	SUBQ     $8, CX
	JMP      loop_utf16LEToLatin1BlocksWestmere

done_utf16LEToLatin1BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf16BEToLatin1BlocksWestmere(input []uint16, dst []byte) (consumed int)
TEXT ·utf16BEToLatin1BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	MOVQ  dst_base+24(FP), DI
	XORQ  AX, AX
	MOVOU ·utf16Latin1ByteSwapMask<>(SB), X7

loop_utf16BEToLatin1BlocksWestmere:
	CMPQ     CX, $8
	JB       done_utf16BEToLatin1BlocksWestmere
	MOVOU    (SI), X0
	PSHUFB   X7, X0
	MOVOU    X0, X1
	PSRLW    $8, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	CMPL     DX, $0xffff
	JNE      done_utf16BEToLatin1BlocksWestmere
	PACKUSWB X0, X0
	MOVQ     X0, (DI)
	ADDQ     $16, SI
	ADDQ     $8, DI
	ADDQ     $8, AX
	SUBQ     $8, CX
	JMP      loop_utf16BEToLatin1BlocksWestmere

done_utf16BEToLatin1BlocksWestmere:
	MOVQ AX, consumed+48(FP)
	RET

// func utf16LEToLatin1BlocksHaswell(input []uint16, dst []byte) (consumed int)
// Processes complete 16-uint16 (32-byte) latin1-only groups with AVX2.
TEXT ·utf16LEToLatin1BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	MOVQ dst_base+24(FP), DI
	XORQ AX, AX

loop_utf16LEToLatin1BlocksHaswell:
	CMPQ    CX, $16
	JB      done_utf16LEToLatin1BlocksHaswell
	VMOVDQU (SI), Y0

	// Isolate high bytes; require every word == 0 (VPMOVMSKB-of-shift is MSB-only).
	VPSRLW       $8, Y0, Y1
	VPXOR        Y2, Y2, Y2
	VPCMPEQW     Y2, Y1, Y1
	VPMOVMSKB    Y1, DX
	CMPL         DX, $0xffffffff
	JNE          done_utf16LEToLatin1BlocksHaswell
	VEXTRACTI128 $1, Y0, X1
	VPACKUSWB    X1, X0, X0
	VMOVDQU      X0, (DI)
	ADDQ         $32, SI
	ADDQ         $16, DI
	ADDQ         $16, AX
	SUBQ         $16, CX
	JMP          loop_utf16LEToLatin1BlocksHaswell

done_utf16LEToLatin1BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET

// func utf16BEToLatin1BlocksHaswell(input []uint16, dst []byte) (consumed int)
TEXT ·utf16BEToLatin1BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-56
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), CX
	MOVQ    dst_base+24(FP), DI
	XORQ    AX, AX
	VMOVDQU ·utf16Latin1ByteSwapMask<>(SB), Y7

loop_utf16BEToLatin1BlocksHaswell:
	CMPQ         CX, $16
	JB           done_utf16BEToLatin1BlocksHaswell
	VMOVDQU      (SI), Y0
	VPSHUFB      Y7, Y0, Y0
	VPSRLW       $8, Y0, Y1
	VPXOR        Y2, Y2, Y2
	VPCMPEQW     Y2, Y1, Y1
	VPMOVMSKB    Y1, DX
	CMPL         DX, $0xffffffff
	JNE          done_utf16BEToLatin1BlocksHaswell
	VEXTRACTI128 $1, Y0, X1
	VPACKUSWB    X1, X0, X0
	VMOVDQU      X0, (DI)
	ADDQ         $32, SI
	ADDQ         $16, DI
	ADDQ         $16, AX
	SUBQ         $16, CX
	JMP          loop_utf16BEToLatin1BlocksHaswell

done_utf16BEToLatin1BlocksHaswell:
	VZEROUPPER
	MOVQ AX, consumed+48(FP)
	RET
