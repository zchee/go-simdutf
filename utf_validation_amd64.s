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
// Independent Go assembly translations of complete-block filters from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/westmere/sse_validate_utf16.cpp and src/haswell/avx2_validate_utf16.cpp.

#include "textflag.h"

DATA ·utf16NativeMask<>+0(SB)/8, $0xf800f800f800f800
DATA ·utf16NativeMask<>+8(SB)/8, $0xf800f800f800f800
DATA ·utf16NativeMask<>+16(SB)/8, $0xf800f800f800f800
DATA ·utf16NativeMask<>+24(SB)/8, $0xf800f800f800f800
GLOBL ·utf16NativeMask<>(SB), RODATA|NOPTR, $32
DATA ·utf16NativeSurrogate<>+0(SB)/8, $0xd800d800d800d800
DATA ·utf16NativeSurrogate<>+8(SB)/8, $0xd800d800d800d800
DATA ·utf16NativeSurrogate<>+16(SB)/8, $0xd800d800d800d800
DATA ·utf16NativeSurrogate<>+24(SB)/8, $0xd800d800d800d800
GLOBL ·utf16NativeSurrogate<>(SB), RODATA|NOPTR, $32
DATA ·utf16BigMask<>+0(SB)/8, $0x00f800f800f800f8
DATA ·utf16BigMask<>+8(SB)/8, $0x00f800f800f800f8
DATA ·utf16BigMask<>+16(SB)/8, $0x00f800f800f800f8
DATA ·utf16BigMask<>+24(SB)/8, $0x00f800f800f800f8
GLOBL ·utf16BigMask<>(SB), RODATA|NOPTR, $32
DATA ·utf16BigSurrogate<>+0(SB)/8, $0x00d800d800d800d8
DATA ·utf16BigSurrogate<>+8(SB)/8, $0x00d800d800d800d8
DATA ·utf16BigSurrogate<>+16(SB)/8, $0x00d800d800d800d8
DATA ·utf16BigSurrogate<>+24(SB)/8, $0x00d800d800d800d8
GLOBL ·utf16BigSurrogate<>(SB), RODATA|NOPTR, $32
DATA ·utf32ASCIIMask<>+0(SB)/8, $0xffffff80ffffff80
DATA ·utf32ASCIIMask<>+8(SB)/8, $0xffffff80ffffff80
DATA ·utf32ASCIIMask<>+16(SB)/8, $0xffffff80ffffff80
DATA ·utf32ASCIIMask<>+24(SB)/8, $0xffffff80ffffff80
GLOBL ·utf32ASCIIMask<>(SB), RODATA|NOPTR, $32

// func utf16NoSurrogateWestmere(input []uint16, bigEndian bool) int
TEXT ·utf16NoSurrogateWestmere(SB), NOSPLIT, $0-40
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-8, CX
	XORQ AX, AX
	CMPB bigEndian+24(FP), $0
	JEQ  westmere_utf16_native
	LEAQ ·utf16BigMask<>(SB), DI
	LEAQ ·utf16BigSurrogate<>(SB), BX
	JMP  westmere_utf16_loop

westmere_utf16_native:
	LEAQ ·utf16NativeMask<>(SB), DI
	LEAQ ·utf16NativeSurrogate<>(SB), BX

westmere_utf16_loop:
	CMPQ     AX, CX
	JAE      westmere_utf16_done
	MOVOU    0(SI)(AX*2), X0
	PAND     (DI), X0
	PCMPEQW  (BX), X0
	PMOVMSKB X0, DX
	TESTL    DX, DX
	JNE      westmere_utf16_done
	ADDQ     $8, AX
	JMP      westmere_utf16_loop

westmere_utf16_done:
	MOVQ AX, ret+32(FP)
	RET

// func utf16NoSurrogateHaswell(input []uint16, bigEndian bool) int
TEXT ·utf16NoSurrogateHaswell(SB), NOSPLIT, $0-40
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-16, CX
	XORQ AX, AX
	CMPB bigEndian+24(FP), $0
	JEQ  haswell_utf16_native
	LEAQ ·utf16BigMask<>(SB), DI
	LEAQ ·utf16BigSurrogate<>(SB), BX
	JMP  haswell_utf16_loop

haswell_utf16_native:
	LEAQ ·utf16NativeMask<>(SB), DI
	LEAQ ·utf16NativeSurrogate<>(SB), BX

haswell_utf16_loop:
	CMPQ      AX, CX
	JAE       haswell_utf16_done
	VMOVDQU   0(SI)(AX*2), Y0
	VPAND     (DI), Y0, Y0
	VPCMPEQW  (BX), Y0, Y0
	VPMOVMSKB Y0, DX
	TESTL     DX, DX
	JNE       haswell_utf16_done
	ADDQ      $16, AX
	JMP       haswell_utf16_loop

haswell_utf16_done:
	VZEROUPPER
	MOVQ AX, ret+32(FP)
	RET

// func utf32ASCIIPrefixWestmere(input []uint32) int
TEXT ·utf32ASCIIPrefixWestmere(SB), NOSPLIT, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-4, CX
	XORQ AX, AX
	LEAQ ·utf32ASCIIMask<>(SB), DI

westmere_utf32_loop:
	CMPQ  AX, CX
	JAE   westmere_utf32_done
	MOVOU 0(SI)(AX*4), X0
	PAND  (DI), X0
	PTEST X0, X0
	JNE   westmere_utf32_done
	ADDQ  $4, AX
	JMP   westmere_utf32_loop

westmere_utf32_done:
	MOVQ AX, ret+24(FP)
	RET

// func utf32ASCIIPrefixHaswell(input []uint32) int
TEXT ·utf32ASCIIPrefixHaswell(SB), NOSPLIT, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-8, CX
	XORQ AX, AX
	LEAQ ·utf32ASCIIMask<>(SB), DI

haswell_utf32_loop:
	CMPQ    AX, CX
	JAE     haswell_utf32_done
	VMOVDQU 0(SI)(AX*4), Y0
	VPAND   (DI), Y0, Y0
	VPTEST  Y0, Y0
	JNE     haswell_utf32_done
	ADDQ    $8, AX
	JMP     haswell_utf32_loop

haswell_utf32_done:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET

// func utf16CopyWestmere(input, dst []uint16, n int)
TEXT ·utf16CopyWestmere(SB), NOSPLIT, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ dst_base+24(FP), DI
	MOVQ n+48(FP), CX
	XORQ AX, AX

westmere_utf16_copy_loop:
	CMPQ  AX, CX
	JAE   westmere_utf16_copy_done
	MOVOU 0(SI)(AX*2), X0
	MOVOU X0, 0(DI)(AX*2)
	ADDQ  $8, AX
	JMP   westmere_utf16_copy_loop

westmere_utf16_copy_done:
	RET

// func utf16CopyHaswell(input, dst []uint16, n int)
TEXT ·utf16CopyHaswell(SB), NOSPLIT, $0-56
	MOVQ input_base+0(FP), SI
	MOVQ dst_base+24(FP), DI
	MOVQ n+48(FP), CX
	XORQ AX, AX

haswell_utf16_copy_loop:
	CMPQ    AX, CX
	JAE     haswell_utf16_copy_done
	VMOVDQU 0(SI)(AX*2), Y0
	VMOVDQU Y0, 0(DI)(AX*2)
	ADDQ    $16, AX
	JMP     haswell_utf16_copy_loop

haswell_utf16_copy_done:
	VZEROUPPER
	RET
