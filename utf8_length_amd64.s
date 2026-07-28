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

// Independent Go assembly translations of the pinned Westmere and Haswell
// UTF-8 length algorithms in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// src/generic/utf8/utf16_length_from_utf8_bytemask.h, src/generic/utf8.h:8-20,
// src/simdutf/westmere/simd.h:150-358, and
// src/simdutf/haswell/simd.h:150-345. UTF-16 subtracts the two all-ones byte
// masks and widens after exactly 127 vectors. UTF-32 builds the exact 64-bit
// simd8x64 mask and performs one POPCNTQ per 64-byte block.

#include "textflag.h"

DATA ·utf8LengthMinus65<>+0(SB)/8, $0xbfbfbfbfbfbfbfbf
DATA ·utf8LengthMinus65<>+8(SB)/8, $0xbfbfbfbfbfbfbfbf
DATA ·utf8LengthMinus65<>+16(SB)/8, $0xbfbfbfbfbfbfbfbf
DATA ·utf8LengthMinus65<>+24(SB)/8, $0xbfbfbfbfbfbfbfbf
GLOBL ·utf8LengthMinus65<>(SB), RODATA|NOPTR, $32

DATA ·utf8Length240<>+0(SB)/8, $0xf0f0f0f0f0f0f0f0
DATA ·utf8Length240<>+8(SB)/8, $0xf0f0f0f0f0f0f0f0
DATA ·utf8Length240<>+16(SB)/8, $0xf0f0f0f0f0f0f0f0
DATA ·utf8Length240<>+24(SB)/8, $0xf0f0f0f0f0f0f0f0
GLOBL ·utf8Length240<>(SB), RODATA|NOPTR, $32

// func utf16LengthFromUTF8BlocksWestmere(input []byte) int
TEXT ·utf16LengthFromUTF8BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-16, CX
	PXOR  X0, X0
	PXOR  X4, X4
	XORQ  R8, R8
	TESTQ CX, CX
	JE    utf16_length_westmere_finish
	MOVOU ·utf8LengthMinus65<>(SB), X2
	MOVOU ·utf8Length240<>(SB), X6

utf16_length_westmere_loop:
	MOVOU   0(SI), X3
	MOVO    X3, X5
	PCMPGTB X2, X3
	PMINUB  X6, X5
	PCMPEQB X6, X5
	PSUBB   X3, X0
	PSUBB   X5, X0
	ADDQ    $16, SI
	SUBQ    $16, CX
	INCQ    R8
	CMPQ    R8, $127
	JNE     utf16_length_westmere_continue
	PXOR    X1, X1
	PSADBW  X1, X0
	PADDQ   X0, X4
	PXOR    X0, X0
	XORQ    R8, R8

utf16_length_westmere_continue:
	TESTQ CX, CX
	JNE   utf16_length_westmere_loop

utf16_length_westmere_finish:
	PXOR   X1, X1
	PSADBW X1, X0
	PADDQ  X0, X4
	PSHUFD $0x4e, X4, X5
	PADDQ  X5, X4
	MOVQ   X4, AX
	MOVQ   AX, length+24(FP)
	RET

// func utf16LengthFromUTF8BlocksHaswell(input []byte) int
TEXT ·utf16LengthFromUTF8BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), CX
	ANDQ    $-32, CX
	VPXOR   Y0, Y0, Y0
	VPXOR   Y4, Y4, Y4
	XORQ    R8, R8
	TESTQ   CX, CX
	JE      utf16_length_haswell_finish
	VMOVDQU ·utf8LengthMinus65<>(SB), Y2
	VMOVDQU ·utf8Length240<>(SB), Y6

utf16_length_haswell_loop:
	VMOVDQU  0(SI), Y3
	VPCMPGTB Y2, Y3, Y5
	VPMINUB  Y6, Y3, Y7
	VPCMPEQB Y6, Y7, Y7
	VPSUBB   Y5, Y0, Y0
	VPSUBB   Y7, Y0, Y0
	ADDQ     $32, SI
	SUBQ     $32, CX
	INCQ     R8
	CMPQ     R8, $127
	JNE      utf16_length_haswell_continue
	VPXOR    Y1, Y1, Y1
	VPSADBW  Y1, Y0, Y3
	VPADDQ   Y3, Y4, Y4
	VPXOR    Y0, Y0, Y0
	XORQ     R8, R8

utf16_length_haswell_continue:
	TESTQ CX, CX
	JNE   utf16_length_haswell_loop

utf16_length_haswell_finish:
	VPXOR        Y1, Y1, Y1
	VPSADBW      Y1, Y0, Y3
	VPADDQ       Y3, Y4, Y4
	VEXTRACTI128 $1, Y4, X5
	VPADDQ       X5, X4, X4
	VPSHUFD      $0x4e, X4, X5
	VPADDQ       X5, X4, X4
	MOVQ         X4, AX
	VZEROUPPER
	MOVQ         AX, length+24(FP)
	RET

// func utf32LengthFromUTF8BlocksWestmere(input []byte) int
TEXT ·utf32LengthFromUTF8BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-64, CX
	XORQ  R8, R8
	TESTQ CX, CX
	JE    utf32_length_westmere_done
	MOVOU ·utf8LengthMinus65<>(SB), X2

utf32_length_westmere_loop:
	MOVOU    0(SI), X3
	PCMPGTB  X2, X3
	PMOVMSKB X3, AX
	MOVOU    16(SI), X3
	PCMPGTB  X2, X3
	PMOVMSKB X3, DX
	SHLQ     $16, DX
	ORQ      DX, AX
	MOVOU    32(SI), X3
	PCMPGTB  X2, X3
	PMOVMSKB X3, DX
	SHLQ     $32, DX
	ORQ      DX, AX
	MOVOU    48(SI), X3
	PCMPGTB  X2, X3
	PMOVMSKB X3, DX
	SHLQ     $48, DX
	ORQ      DX, AX
	POPCNTQ  AX, AX
	ADDQ     AX, R8
	ADDQ     $64, SI
	SUBQ     $64, CX
	JNE      utf32_length_westmere_loop

utf32_length_westmere_done:
	MOVQ R8, length+24(FP)
	RET

// func utf32LengthFromUTF8BlocksHaswell(input []byte) int
TEXT ·utf32LengthFromUTF8BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), CX
	ANDQ    $-64, CX
	XORQ    R8, R8
	TESTQ   CX, CX
	JE      utf32_length_haswell_done
	VMOVDQU ·utf8LengthMinus65<>(SB), Y2

utf32_length_haswell_loop:
	VMOVDQU   0(SI), Y3
	VPCMPGTB  Y2, Y3, Y3
	VPMOVMSKB Y3, AX
	VMOVDQU   32(SI), Y3
	VPCMPGTB  Y2, Y3, Y3
	VPMOVMSKB Y3, DX
	SHLQ      $32, DX
	ORQ       DX, AX
	POPCNTQ   AX, AX
	ADDQ      AX, R8
	ADDQ      $64, SI
	SUBQ      $64, CX
	JNE       utf32_length_haswell_loop

utf32_length_haswell_done:
	VZEROUPPER
	MOVQ R8, length+24(FP)
	RET
