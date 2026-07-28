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

// Independent Go assembly translations of the lookup4 complete-block checker
// in simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216,
// src/generic/utf8_validation/utf8_validator.h:10-80,
// src/westmere/implementation.cpp:19-29, and
// src/haswell/implementation.cpp:19-29. These routines read complete 64-byte
// blocks only and return the start of the first block with cumulative lookup4
// error, leaving exact public error classification to the Go scalar oracle.

#include "textflag.h"

// lookup4 tables, repeated in both 128-bit lanes for AVX2 VPSHUFB.
DATA ·utf8LookupHigh<>+0(SB)/8, $0x0202020202020202
DATA ·utf8LookupHigh<>+8(SB)/8, $0x4915012180808080
DATA ·utf8LookupHigh<>+16(SB)/8, $0x0202020202020202
DATA ·utf8LookupHigh<>+24(SB)/8, $0x4915012180808080
GLOBL ·utf8LookupHigh<>(SB), RODATA|NOPTR, $32

DATA ·utf8LookupLow<>+0(SB)/8, $0xcbcbcb8b8383a3e7
DATA ·utf8LookupLow<>+8(SB)/8, $0xcbcbdbcbcbcbcbcb
DATA ·utf8LookupLow<>+16(SB)/8, $0xcbcbcb8b8383a3e7
DATA ·utf8LookupLow<>+24(SB)/8, $0xcbcbdbcbcbcbcbcb
GLOBL ·utf8LookupLow<>(SB), RODATA|NOPTR, $32

DATA ·utf8LookupInput<>+0(SB)/8, $0x0101010101010101
DATA ·utf8LookupInput<>+8(SB)/8, $0x01010101babaaee6
DATA ·utf8LookupInput<>+16(SB)/8, $0x0101010101010101
DATA ·utf8LookupInput<>+24(SB)/8, $0x01010101babaaee6
GLOBL ·utf8LookupInput<>(SB), RODATA|NOPTR, $32

DATA ·utf8NibbleMask<>+0(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA ·utf8NibbleMask<>+8(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA ·utf8NibbleMask<>+16(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA ·utf8NibbleMask<>+24(SB)/8, $0x0f0f0f0f0f0f0f0f
GLOBL ·utf8NibbleMask<>(SB), RODATA|NOPTR, $32

DATA ·utf8Sub60<>+0(SB)/8, $0x6060606060606060
DATA ·utf8Sub60<>+8(SB)/8, $0x6060606060606060
DATA ·utf8Sub60<>+16(SB)/8, $0x6060606060606060
DATA ·utf8Sub60<>+24(SB)/8, $0x6060606060606060
GLOBL ·utf8Sub60<>(SB), RODATA|NOPTR, $32

DATA ·utf8Sub70<>+0(SB)/8, $0x7070707070707070
DATA ·utf8Sub70<>+8(SB)/8, $0x7070707070707070
DATA ·utf8Sub70<>+16(SB)/8, $0x7070707070707070
DATA ·utf8Sub70<>+24(SB)/8, $0x7070707070707070
GLOBL ·utf8Sub70<>(SB), RODATA|NOPTR, $32

DATA ·utf8Bit80<>+0(SB)/8, $0x8080808080808080
DATA ·utf8Bit80<>+8(SB)/8, $0x8080808080808080
DATA ·utf8Bit80<>+16(SB)/8, $0x8080808080808080
DATA ·utf8Bit80<>+24(SB)/8, $0x8080808080808080
GLOBL ·utf8Bit80<>(SB), RODATA|NOPTR, $32

// X0 is current input, X1 is previous input, and X10 accumulates errors.
#define CHECK_UTF8_XMM() \
	MOVO X0, X2; PALIGNR $15, X1, X2;                                                                             \
	MOVO X2, X3; PSRLW $4, X3; PAND ·utf8NibbleMask<>(SB), X3; MOVOU ·utf8LookupHigh<>(SB), X8; PSHUFB X3, X8;    \
	MOVO X2, X4; PAND ·utf8NibbleMask<>(SB), X4; MOVOU ·utf8LookupLow<>(SB), X9; PSHUFB X4, X9;                   \
	MOVO X0, X5; PSRLW $4, X5; PAND ·utf8NibbleMask<>(SB), X5; MOVOU ·utf8LookupInput<>(SB), X11; PSHUFB X5, X11; \
	PAND X9, X8; PAND X11, X8;                                                                                    \
	MOVO X0, X6; PALIGNR $14, X1, X6; PSUBUSB ·utf8Sub60<>(SB), X6;                                               \
	MOVO X0, X7; PALIGNR $13, X1, X7; PSUBUSB ·utf8Sub70<>(SB), X7;                                               \
	POR  X7, X6; PAND ·utf8Bit80<>(SB), X6; PXOR X8, X6; POR X6, X10;                                             \
	MOVO X0, X1

// func validateUTF8PrefixWestmere(input []byte) int
TEXT ·validateUTF8PrefixWestmere(SB), NOSPLIT, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-64, CX
	XORQ AX, AX
	PXOR X1, X1
	PXOR X10, X10

utf8_westmere_loop:
	CMPQ     AX, CX
	JAE      utf8_westmere_return
	MOVOU    0(SI)(AX*1), X0
	CHECK_UTF8_XMM()
	MOVOU    16(SI)(AX*1), X0
	CHECK_UTF8_XMM()
	MOVOU    32(SI)(AX*1), X0
	CHECK_UTF8_XMM()
	MOVOU    48(SI)(AX*1), X0
	CHECK_UTF8_XMM()
	PXOR     X11, X11
	MOVO     X10, X9
	PCMPEQB  X11, X9
	PMOVMSKB X9, DX
	CMPL     DX, $0xffff
	JNE      utf8_westmere_return
	ADDQ     $64, AX
	JMP      utf8_westmere_loop

utf8_westmere_return:
	MOVQ AX, ret+24(FP)
	RET

// Y0 is current input, Y1 is previous input, and Y10 accumulates errors.
// Y2 bridges the preceding high lane and current low lane before VPALIGNR.
#define CHECK_UTF8_YMM() \
	VPERM2I128 $0x21, Y0, Y1, Y2;                                                                                         \
	VPALIGNR   $15, Y2, Y0, Y3;                                                                                           \
	VPSRLW     $4, Y3, Y4; VPAND ·utf8NibbleMask<>(SB), Y4, Y4; VMOVDQU ·utf8LookupHigh<>(SB), Y12; VPSHUFB Y4, Y12, Y4;  \
	VPAND      ·utf8NibbleMask<>(SB), Y3, Y5; VMOVDQU ·utf8LookupLow<>(SB), Y13; VPSHUFB Y5, Y13, Y5;                     \
	VPSRLW     $4, Y0, Y6; VPAND ·utf8NibbleMask<>(SB), Y6, Y6; VMOVDQU ·utf8LookupInput<>(SB), Y14; VPSHUFB Y6, Y14, Y6; \
	VPAND      Y5, Y4, Y4; VPAND Y6, Y4, Y4;                                                                              \
	VPALIGNR   $14, Y2, Y0, Y7; VPSUBUSB ·utf8Sub60<>(SB), Y7, Y7;                                                        \
	VPALIGNR   $13, Y2, Y0, Y8; VPSUBUSB ·utf8Sub70<>(SB), Y8, Y8;                                                        \
	VPOR       Y8, Y7, Y7; VPAND ·utf8Bit80<>(SB), Y7, Y7; VPXOR Y4, Y7, Y7; VPOR Y7, Y10, Y10;                           \
	VMOVDQA    Y0, Y1

// func validateUTF8PrefixHaswell(input []byte) int
TEXT ·validateUTF8PrefixHaswell(SB), NOSPLIT, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-64, CX
	XORQ  AX, AX
	VPXOR Y1, Y1, Y1
	VPXOR Y10, Y10, Y10

utf8_haswell_loop:
	CMPQ      AX, CX
	JAE       utf8_haswell_return
	VMOVDQU   0(SI)(AX*1), Y0
	CHECK_UTF8_YMM()
	VMOVDQU   32(SI)(AX*1), Y0
	CHECK_UTF8_YMM()
	VPXOR     Y11, Y11, Y11
	VPCMPEQB  Y11, Y10, Y9
	VPMOVMSKB Y9, DX
	CMPL      DX, $-1
	JNE       utf8_haswell_return
	ADDQ      $64, AX
	JMP       utf8_haswell_loop

utf8_haswell_return:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET
