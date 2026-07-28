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

// Independent Go assembly translations of the Westmere and Haswell
// count_code_points_bytemask loops in
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/utf8.h:21-68,
// src/westmere/implementation.cpp:1142-1146, and
// src/haswell/implementation.cpp:1115-1119. PCMPGTB/VPCMPGTB implement the
// pinned signed int8 predicate input > -65. Four masks are subtracted into
// byte lanes per iteration, and those lanes are widened after exactly 63
// iterations (255/4) before they can wrap.

#include "textflag.h"

DATA ·countUTF8Minus65<>+0(SB)/8, $0xbfbfbfbfbfbfbfbf
DATA ·countUTF8Minus65<>+8(SB)/8, $0xbfbfbfbfbfbfbfbf
DATA ·countUTF8Minus65<>+16(SB)/8, $0xbfbfbfbfbfbfbfbf
DATA ·countUTF8Minus65<>+24(SB)/8, $0xbfbfbfbfbfbfbfbf
GLOBL ·countUTF8Minus65<>(SB), RODATA|NOPTR, $32

// func countUTF8BlocksWestmere(input []byte) int
TEXT ·countUTF8BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-64, CX
	PXOR  X0, X0                      // local byte counters
	PXOR  X4, X4                      // widened 64-bit counters
	XORQ  R8, R8                      // iterations since the last widening
	TESTQ CX, CX
	JE    count_utf8_westmere_finish
	MOVOU ·countUTF8Minus65<>(SB), X2

count_utf8_westmere_loop:
	MOVOU   0(SI), X3
	PCMPGTB X2, X3
	PSUBB   X3, X0
	MOVOU   16(SI), X3
	PCMPGTB X2, X3
	PSUBB   X3, X0
	MOVOU   32(SI), X3
	PCMPGTB X2, X3
	PSUBB   X3, X0
	MOVOU   48(SI), X3
	PCMPGTB X2, X3
	PSUBB   X3, X0
	ADDQ    $64, SI
	SUBQ    $64, CX
	INCQ    R8
	CMPQ    R8, $63
	JNE     count_utf8_westmere_continue
	PXOR    X1, X1
	PSADBW  X1, X0
	PADDQ   X0, X4
	PXOR    X0, X0
	XORQ    R8, R8

count_utf8_westmere_continue:
	TESTQ CX, CX
	JNE   count_utf8_westmere_loop

count_utf8_westmere_finish:
	PXOR   X1, X1
	PSADBW X1, X0
	PADDQ  X0, X4
	PSHUFD $0x4e, X4, X5
	PADDQ  X5, X4
	MOVQ   X4, AX
	MOVQ   AX, count+24(FP)
	RET

// func countUTF8BlocksHaswell(input []byte) int
TEXT ·countUTF8BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), CX
	ANDQ    $-128, CX
	VPXOR   Y0, Y0, Y0                  // local byte counters
	VPXOR   Y4, Y4, Y4                  // widened 64-bit counters
	XORQ    R8, R8                      // iterations since the last widening
	TESTQ   CX, CX
	JE      count_utf8_haswell_finish
	VMOVDQU ·countUTF8Minus65<>(SB), Y2

count_utf8_haswell_loop:
	VMOVDQU  0(SI), Y3
	VPCMPGTB Y2, Y3, Y3
	VPSUBB   Y3, Y0, Y0
	VMOVDQU  32(SI), Y3
	VPCMPGTB Y2, Y3, Y3
	VPSUBB   Y3, Y0, Y0
	VMOVDQU  64(SI), Y3
	VPCMPGTB Y2, Y3, Y3
	VPSUBB   Y3, Y0, Y0
	VMOVDQU  96(SI), Y3
	VPCMPGTB Y2, Y3, Y3
	VPSUBB   Y3, Y0, Y0
	ADDQ     $128, SI
	SUBQ     $128, CX
	INCQ     R8
	CMPQ     R8, $63
	JNE      count_utf8_haswell_continue
	VPXOR    Y1, Y1, Y1
	VPSADBW  Y1, Y0, Y3
	VPADDQ   Y3, Y4, Y4
	VPXOR    Y0, Y0, Y0
	XORQ     R8, R8

count_utf8_haswell_continue:
	TESTQ CX, CX
	JNE   count_utf8_haswell_loop

count_utf8_haswell_finish:
	VPXOR        Y1, Y1, Y1
	VPSADBW      Y1, Y0, Y3
	VPADDQ       Y3, Y4, Y4
	VEXTRACTI128 $1, Y4, X5
	VPADDQ       X5, X4, X4
	VPSHUFD      $0x4e, X4, X5
	VPADDQ       X5, X4, X4
	MOVQ         X4, AX
	VZEROUPPER
	MOVQ         AX, count+24(FP)
	RET
