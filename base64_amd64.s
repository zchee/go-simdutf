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

// Independent Go assembly translations of Westmere/Haswell Base64 length and
// encode kernels from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
//   src/generic/base64lengths.h
//   src/westmere/sse_base64.cpp (encode 12→16 SSSE3 lane)
//   src/haswell/avx2_base64.cpp (encode 24→32 AVX2 lane)
// Length counts bytes/code units > 0x20 via unsigned compare + POPCNT.
// Encode owns complete vector groups only; Go wrappers retain scalar tails,
// padding, and line-break insertion.

#include "textflag.h"

// --- constants -------------------------------------------------------------

// PSHUFB shuffle for base64 lane expand (set_epi8 high→low):
// [1,0,2,1, 4,3,5,4, 7,6,8,7, 10,9,11,10]
DATA ·base64Shuf<>+0(SB)/8, $0x0405030401020001
DATA ·base64Shuf<>+8(SB)/8, $0x0a0b090a07080607
GLOBL ·base64Shuf<>(SB), RODATA|NOPTR, $16

DATA ·base64Mask0<>+0(SB)/8, $0x0fc0fc000fc0fc00
DATA ·base64Mask0<>+8(SB)/8, $0x0fc0fc000fc0fc00
GLOBL ·base64Mask0<>(SB), RODATA|NOPTR, $16

DATA ·base64MulHi<>+0(SB)/8, $0x0400004004000040
DATA ·base64MulHi<>+8(SB)/8, $0x0400004004000040
GLOBL ·base64MulHi<>(SB), RODATA|NOPTR, $16

DATA ·base64Mask1<>+0(SB)/8, $0x003f03f0003f03f0
DATA ·base64Mask1<>+8(SB)/8, $0x003f03f0003f03f0
GLOBL ·base64Mask1<>(SB), RODATA|NOPTR, $16

DATA ·base64MulLo<>+0(SB)/8, $0x0100001001000010
DATA ·base64MulLo<>+8(SB)/8, $0x0100001001000010
GLOBL ·base64MulLo<>(SB), RODATA|NOPTR, $16

DATA ·base64Spl51<>+0(SB)/8, $0x3333333333333333
DATA ·base64Spl51<>+8(SB)/8, $0x3333333333333333
GLOBL ·base64Spl51<>(SB), RODATA|NOPTR, $16

DATA ·base64Spl26<>+0(SB)/8, $0x1a1a1a1a1a1a1a1a
DATA ·base64Spl26<>+8(SB)/8, $0x1a1a1a1a1a1a1a1a
GLOBL ·base64Spl26<>(SB), RODATA|NOPTR, $16

DATA ·base64Spl13<>+0(SB)/8, $0x0d0d0d0d0d0d0d0d
DATA ·base64Spl13<>+8(SB)/8, $0x0d0d0d0d0d0d0d0d
GLOBL ·base64Spl13<>(SB), RODATA|NOPTR, $16

// setr_epi8 standard shift LUT
DATA ·base64ShiftStd<>+0(SB)/8, $0xfcfcfcfcfcfcfc47
DATA ·base64ShiftStd<>+8(SB)/8, $0x000041f0ebfcfcfc
GLOBL ·base64ShiftStd<>(SB), RODATA|NOPTR, $16

// setr_epi8 URL shift LUT
DATA ·base64ShiftURL<>+0(SB)/8, $0xfcfcfcfcfcfcfc47
DATA ·base64ShiftURL<>+8(SB)/8, $0x00004120effcfcfc
GLOBL ·base64ShiftURL<>(SB), RODATA|NOPTR, $16

DATA ·base64Space<>+0(SB)/8, $0x2020202020202020
DATA ·base64Space<>+8(SB)/8, $0x2020202020202020
GLOBL ·base64Space<>(SB), RODATA|NOPTR, $16

// AVX2 duplicates (32-byte)
DATA ·base64ShufY<>+0(SB)/8, $0x0405030401020001
DATA ·base64ShufY<>+8(SB)/8, $0x0a0b090a07080607
DATA ·base64ShufY<>+16(SB)/8, $0x0405030401020001
DATA ·base64ShufY<>+24(SB)/8, $0x0a0b090a07080607
GLOBL ·base64ShufY<>(SB), RODATA|NOPTR, $32

DATA ·base64Mask0Y<>+0(SB)/8, $0x0fc0fc000fc0fc00
DATA ·base64Mask0Y<>+8(SB)/8, $0x0fc0fc000fc0fc00
DATA ·base64Mask0Y<>+16(SB)/8, $0x0fc0fc000fc0fc00
DATA ·base64Mask0Y<>+24(SB)/8, $0x0fc0fc000fc0fc00
GLOBL ·base64Mask0Y<>(SB), RODATA|NOPTR, $32

DATA ·base64MulHiY<>+0(SB)/8, $0x0400004004000040
DATA ·base64MulHiY<>+8(SB)/8, $0x0400004004000040
DATA ·base64MulHiY<>+16(SB)/8, $0x0400004004000040
DATA ·base64MulHiY<>+24(SB)/8, $0x0400004004000040
GLOBL ·base64MulHiY<>(SB), RODATA|NOPTR, $32

DATA ·base64Mask1Y<>+0(SB)/8, $0x003f03f0003f03f0
DATA ·base64Mask1Y<>+8(SB)/8, $0x003f03f0003f03f0
DATA ·base64Mask1Y<>+16(SB)/8, $0x003f03f0003f03f0
DATA ·base64Mask1Y<>+24(SB)/8, $0x003f03f0003f03f0
GLOBL ·base64Mask1Y<>(SB), RODATA|NOPTR, $32

DATA ·base64MulLoY<>+0(SB)/8, $0x0100001001000010
DATA ·base64MulLoY<>+8(SB)/8, $0x0100001001000010
DATA ·base64MulLoY<>+16(SB)/8, $0x0100001001000010
DATA ·base64MulLoY<>+24(SB)/8, $0x0100001001000010
GLOBL ·base64MulLoY<>(SB), RODATA|NOPTR, $32

DATA ·base64Spl51Y<>+0(SB)/8, $0x3333333333333333
DATA ·base64Spl51Y<>+8(SB)/8, $0x3333333333333333
DATA ·base64Spl51Y<>+16(SB)/8, $0x3333333333333333
DATA ·base64Spl51Y<>+24(SB)/8, $0x3333333333333333
GLOBL ·base64Spl51Y<>(SB), RODATA|NOPTR, $32

DATA ·base64Spl26Y<>+0(SB)/8, $0x1a1a1a1a1a1a1a1a
DATA ·base64Spl26Y<>+8(SB)/8, $0x1a1a1a1a1a1a1a1a
DATA ·base64Spl26Y<>+16(SB)/8, $0x1a1a1a1a1a1a1a1a
DATA ·base64Spl26Y<>+24(SB)/8, $0x1a1a1a1a1a1a1a1a
GLOBL ·base64Spl26Y<>(SB), RODATA|NOPTR, $32

DATA ·base64Spl13Y<>+0(SB)/8, $0x0d0d0d0d0d0d0d0d
DATA ·base64Spl13Y<>+8(SB)/8, $0x0d0d0d0d0d0d0d0d
DATA ·base64Spl13Y<>+16(SB)/8, $0x0d0d0d0d0d0d0d0d
DATA ·base64Spl13Y<>+24(SB)/8, $0x0d0d0d0d0d0d0d0d
GLOBL ·base64Spl13Y<>(SB), RODATA|NOPTR, $32

DATA ·base64ShiftStdY<>+0(SB)/8, $0xfcfcfcfcfcfcfc47
DATA ·base64ShiftStdY<>+8(SB)/8, $0x000041f0ebfcfcfc
DATA ·base64ShiftStdY<>+16(SB)/8, $0xfcfcfcfcfcfcfc47
DATA ·base64ShiftStdY<>+24(SB)/8, $0x000041f0ebfcfcfc
GLOBL ·base64ShiftStdY<>(SB), RODATA|NOPTR, $32

DATA ·base64ShiftURLY<>+0(SB)/8, $0xfcfcfcfcfcfcfc47
DATA ·base64ShiftURLY<>+8(SB)/8, $0x00004120effcfcfc
DATA ·base64ShiftURLY<>+16(SB)/8, $0xfcfcfcfcfcfcfc47
DATA ·base64ShiftURLY<>+24(SB)/8, $0x00004120effcfcfc
GLOBL ·base64ShiftURLY<>(SB), RODATA|NOPTR, $32

DATA ·base64SpaceY<>+0(SB)/8, $0x2020202020202020
DATA ·base64SpaceY<>+8(SB)/8, $0x2020202020202020
DATA ·base64SpaceY<>+16(SB)/8, $0x2020202020202020
DATA ·base64SpaceY<>+24(SB)/8, $0x2020202020202020
GLOBL ·base64SpaceY<>(SB), RODATA|NOPTR, $32

DATA ·base64SpaceW<>+0(SB)/8, $0x0020002000200020
DATA ·base64SpaceW<>+8(SB)/8, $0x0020002000200020
GLOBL ·base64SpaceW<>(SB), RODATA|NOPTR, $16

DATA ·base64SpaceWY<>+0(SB)/8, $0x0020002000200020
DATA ·base64SpaceWY<>+8(SB)/8, $0x0020002000200020
DATA ·base64SpaceWY<>+16(SB)/8, $0x0020002000200020
DATA ·base64SpaceWY<>+24(SB)/8, $0x0020002000200020
GLOBL ·base64SpaceWY<>(SB), RODATA|NOPTR, $32

// --- length: bytes ---------------------------------------------------------

// func binaryLengthFromBase64BlocksWestmere(input []byte) int
// Counts units with value > 0x20 over complete 64-byte groups.
TEXT ·binaryLengthFromBase64BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-64, CX
	XORQ  AX, AX
	TESTQ CX, CX
	JE    b64_len_w_done
	MOVOU ·base64Space<>(SB), X7

b64_len_w_loop:
	MOVOU    0(SI), X0
	MOVOU    X0, X1
	PSUBUSB  X7, X1
	PXOR     X2, X2
	PCMPEQB  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	MOVOU    16(SI), X0
	MOVOU    X0, X1
	PSUBUSB  X7, X1
	PXOR     X2, X2
	PCMPEQB  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	MOVOU    32(SI), X0
	MOVOU    X0, X1
	PSUBUSB  X7, X1
	PXOR     X2, X2
	PCMPEQB  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	MOVOU    48(SI), X0
	MOVOU    X0, X1
	PSUBUSB  X7, X1
	PXOR     X2, X2
	PCMPEQB  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	ADDQ $64, SI
	SUBQ $64, CX
	JNE  b64_len_w_loop

b64_len_w_done:
	MOVQ AX, ret+24(FP)
	RET

// func binaryLengthFromBase64BlocksHaswell(input []byte) int
TEXT ·binaryLengthFromBase64BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-64, CX
	XORQ  AX, AX
	TESTQ CX, CX
	JE    b64_len_h_done
	VMOVDQU ·base64SpaceY<>(SB), Y7

b64_len_h_loop:
	VMOVDQU   0(SI), Y0
	VPSUBUSB  Y7, Y0, Y1
	VPXOR     Y2, Y2, Y2
	VPCMPEQB  Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	NOTL      DX
	POPCNTQ   DX, DX
	ADDQ      DX, AX

	VMOVDQU   32(SI), Y0
	VPSUBUSB  Y7, Y0, Y1
	VPXOR     Y2, Y2, Y2
	VPCMPEQB  Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	NOTL      DX
	POPCNTQ   DX, DX
	ADDQ      DX, AX

	ADDQ $64, SI
	SUBQ $64, CX
	JNE  b64_len_h_loop

b64_len_h_done:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET

// --- length: UTF-16 --------------------------------------------------------

// func binaryLengthFromBase64UTF16BlocksWestmere(input []uint16) int
// Counts code units > 0x20 over complete 32-unit (64-byte) groups.
// PMOVMSKB sets two bits per matching uint16 lane; divide by two at the end.
TEXT ·binaryLengthFromBase64UTF16BlocksWestmere(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-32, CX
	XORQ  AX, AX
	TESTQ CX, CX
	JE    b64_len16_w_done
	MOVOU ·base64SpaceW<>(SB), X7

b64_len16_w_loop:
	MOVOU    0(SI), X0
	MOVOU    X0, X1
	PSUBUSW  X7, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	MOVOU    16(SI), X0
	MOVOU    X0, X1
	PSUBUSW  X7, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	MOVOU    32(SI), X0
	MOVOU    X0, X1
	PSUBUSW  X7, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	MOVOU    48(SI), X0
	MOVOU    X0, X1
	PSUBUSW  X7, X1
	PXOR     X2, X2
	PCMPEQW  X2, X1
	PMOVMSKB X1, DX
	NOTL     DX
	ANDL     $0xffff, DX
	POPCNTQ  DX, DX
	ADDQ     DX, AX

	ADDQ $64, SI
	SUBQ $32, CX
	JNE  b64_len16_w_loop

	SHRQ $1, AX

b64_len16_w_done:
	MOVQ AX, ret+24(FP)
	RET

// func binaryLengthFromBase64UTF16BlocksHaswell(input []uint16) int
TEXT ·binaryLengthFromBase64UTF16BlocksHaswell(SB), NOSPLIT|NOFRAME, $0-32
	MOVQ  input_base+0(FP), SI
	MOVQ  input_len+8(FP), CX
	ANDQ  $-32, CX
	XORQ  AX, AX
	TESTQ CX, CX
	JE    b64_len16_h_done
	VMOVDQU ·base64SpaceWY<>(SB), Y7

b64_len16_h_loop:
	VMOVDQU   0(SI), Y0
	VPSUBUSW  Y7, Y0, Y1
	VPXOR     Y2, Y2, Y2
	VPCMPEQW  Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	NOTL      DX
	POPCNTQ   DX, DX
	ADDQ      DX, AX

	VMOVDQU   32(SI), Y0
	VPSUBUSW  Y7, Y0, Y1
	VPXOR     Y2, Y2, Y2
	VPCMPEQW  Y2, Y1, Y1
	VPMOVMSKB Y1, DX
	NOTL      DX
	POPCNTQ   DX, DX
	ADDQ      DX, AX

	ADDQ $64, SI
	SUBQ $32, CX
	JNE  b64_len16_h_loop

	SHRQ $1, AX

b64_len16_h_done:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET

// --- encode: Westmere 12→16 ------------------------------------------------

// func base64EncodeBlocksWestmere(input, dst []byte, url int) (consumed, written int)
// Encodes complete 12-byte input groups to 16-byte base64 output while
// input_len >= 16 (safe 16-byte load) and dst_len >= 16.
TEXT ·base64EncodeBlocksWestmere(SB), NOSPLIT|NOFRAME, $0-72
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), R8
	MOVQ dst_base+24(FP), DI
	MOVQ dst_len+32(FP), R9
	MOVQ url+48(FP), R10
	XORQ AX, AX // consumed
	XORQ BX, BX // written

	MOVOU ·base64Shuf<>(SB), X8
	MOVOU ·base64Mask0<>(SB), X9
	MOVOU ·base64MulHi<>(SB), X10
	MOVOU ·base64Mask1<>(SB), X11
	MOVOU ·base64MulLo<>(SB), X12
	MOVOU ·base64Spl51<>(SB), X14
	MOVOU ·base64Spl26<>(SB), X15
	TESTQ R10, R10
	JNZ   b64_enc_w_url
	MOVOU ·base64ShiftStd<>(SB), X13
	JMP   b64_enc_w_loop
b64_enc_w_url:
	MOVOU ·base64ShiftURL<>(SB), X13

b64_enc_w_loop:
	CMPQ R8, $16
	JB   b64_enc_w_done
	CMPQ R9, $16
	JB   b64_enc_w_done

	MOVOU  0(SI), X0
	PSHUFB X8, X0

	MOVOU   X0, X1
	PAND    X9, X1
	PMULHUW X10, X1

	MOVOU  X0, X2
	PAND   X11, X2
	PMULLW X12, X2

	POR X2, X1 // indices

	// lookup_pshufb_improved
	MOVOU   X1, X0
	PSUBUSB X14, X0 // result = subs_epu8(indices, 51)
	MOVOU   X15, X2
	PCMPGTB X1, X2 // X2 = (26 > indices)
	MOVOU   ·base64Spl13<>(SB), X3
	PAND    X3, X2
	POR     X2, X0
	MOVOU   X13, X2
	PSHUFB  X0, X2 // X2 = shuffle(shift_LUT, result)
	PADDB   X1, X2 // + indices

	MOVOU X2, 0(DI)

	ADDQ $12, SI
	ADDQ $16, DI
	ADDQ $12, AX
	ADDQ $16, BX
	SUBQ $12, R8
	SUBQ $16, R9
	JMP  b64_enc_w_loop

b64_enc_w_done:
	MOVQ AX, consumed+56(FP)
	MOVQ BX, written+64(FP)
	RET

// --- encode: Haswell 24→32 -------------------------------------------------

// func base64EncodeBlocksHaswell(input, dst []byte, url int) (consumed, written int)
// Encodes complete 24-byte input groups to 32-byte base64 output while
// input_len >= 28 (safe loads) and dst_len >= 32.
TEXT ·base64EncodeBlocksHaswell(SB), NOSPLIT|NOFRAME, $0-72
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), R8
	MOVQ dst_base+24(FP), DI
	MOVQ dst_len+32(FP), R9
	MOVQ url+48(FP), R10
	XORQ AX, AX
	XORQ BX, BX

	VMOVDQU ·base64ShufY<>(SB), Y8
	VMOVDQU ·base64Mask0Y<>(SB), Y9
	VMOVDQU ·base64MulHiY<>(SB), Y10
	VMOVDQU ·base64Mask1Y<>(SB), Y11
	VMOVDQU ·base64MulLoY<>(SB), Y12
	VMOVDQU ·base64Spl51Y<>(SB), Y14
	VMOVDQU ·base64Spl26Y<>(SB), Y15
	TESTQ   R10, R10
	JNZ     b64_enc_h_url
	VMOVDQU ·base64ShiftStdY<>(SB), Y13
	JMP     b64_enc_h_loop
b64_enc_h_url:
	VMOVDQU ·base64ShiftURLY<>(SB), Y13

b64_enc_h_loop:
	CMPQ R8, $28
	JB   b64_enc_h_done
	CMPQ R9, $32
	JB   b64_enc_h_done

	// lo = load(i), hi = load(i+12); in = set_m128i(hi, lo) then VPSHUFB
	VMOVDQU     0(SI), X0
	VMOVDQU     12(SI), X1
	VINSERTI128 $1, X1, Y0, Y0
	VPSHUFB     Y8, Y0, Y0

	VPAND    Y9, Y0, Y1
	VPMULHUW Y10, Y1, Y1
	VPAND    Y11, Y0, Y2
	VPMULLW  Y12, Y2, Y2
	VPOR     Y2, Y1, Y1

	// lookup: VPCMPGTB Y1, Y15, Y2 => Y2 = (Y15 > Y1) = (26 > indices)
	VPSUBUSB Y14, Y1, Y0
	VPCMPGTB Y1, Y15, Y2
	VPAND    ·base64Spl13Y<>(SB), Y2, Y2
	VPOR     Y2, Y0, Y0
	VPSHUFB  Y0, Y13, Y2
	VPADDB   Y1, Y2, Y2

	VMOVDQU Y2, 0(DI)

	ADDQ $24, SI
	ADDQ $32, DI
	ADDQ $24, AX
	ADDQ $32, BX
	SUBQ $24, R8
	SUBQ $32, R9
	JMP  b64_enc_h_loop

b64_enc_h_done:
	VZEROUPPER
	MOVQ AX, consumed+56(FP)
	MOVQ BX, written+64(FP)
	RET
