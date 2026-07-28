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

// Independent Go assembly translations of the complete-block loops in
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// src/generic/ascii_validation.h:6-45, src/generic/validate_utf16.h:128-158,
// src/simdutf/westmere/simd.h:168-170,290-297, and
// src/simdutf/haswell/simd.h:177-179,293-300.

#include "textflag.h"

DATA ·utf16LEASCIIMask<>+0(SB)/8, $0xff80ff80ff80ff80
DATA ·utf16LEASCIIMask<>+8(SB)/8, $0xff80ff80ff80ff80
DATA ·utf16LEASCIIMask<>+16(SB)/8, $0xff80ff80ff80ff80
DATA ·utf16LEASCIIMask<>+24(SB)/8, $0xff80ff80ff80ff80
GLOBL ·utf16LEASCIIMask<>(SB), RODATA|NOPTR, $32

DATA ·utf16BEASCIIMask<>+0(SB)/8, $0x80ff80ff80ff80ff
DATA ·utf16BEASCIIMask<>+8(SB)/8, $0x80ff80ff80ff80ff
DATA ·utf16BEASCIIMask<>+16(SB)/8, $0x80ff80ff80ff80ff
DATA ·utf16BEASCIIMask<>+24(SB)/8, $0x80ff80ff80ff80ff
GLOBL ·utf16BEASCIIMask<>(SB), RODATA|NOPTR, $32

// func validateASCIIPrefixWestmere(input []byte) int
TEXT ·validateASCIIPrefixWestmere(SB), NOSPLIT, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-64, CX
	XORQ AX, AX
ascii_westmere_loop:
	CMPQ AX, CX
	JAE ascii_westmere_return
	MOVOU 0(SI)(AX*1), X0
	MOVOU 16(SI)(AX*1), X1
	MOVOU 32(SI)(AX*1), X2
	MOVOU 48(SI)(AX*1), X3
	POR X1, X0
	POR X2, X0
	POR X3, X0
	PMOVMSKB X0, DX
	TESTL DX, DX
	JNE ascii_westmere_return
	ADDQ $64, AX
	JMP ascii_westmere_loop
ascii_westmere_return:
	MOVQ AX, ret+24(FP)
	RET

// func validateASCIIPrefixHaswell(input []byte) int
TEXT ·validateASCIIPrefixHaswell(SB), NOSPLIT, $0-32
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-64, CX
	XORQ AX, AX
ascii_haswell_loop:
	CMPQ AX, CX
	JAE ascii_haswell_return
	VMOVDQU 0(SI)(AX*1), Y0
	VMOVDQU 32(SI)(AX*1), Y1
	VPOR Y1, Y0, Y0
	VPMOVMSKB Y0, DX
	TESTL DX, DX
	JNE ascii_haswell_return
	ADDQ $64, AX
	JMP ascii_haswell_loop
ascii_haswell_return:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET

// func validateUTF16LEASCIIPrefixWestmere(input []uint16) int
TEXT ·validateUTF16LEASCIIPrefixWestmere(SB), NOSPLIT, $0-32
	LEAQ ·utf16LEASCIIMask<>(SB), DI
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ AX, AX
utf16le_ascii_westmere_loop:
	CMPQ AX, CX
	JAE utf16le_ascii_westmere_return
	MOVOU 0(SI)(AX*2), X0
	MOVOU 16(SI)(AX*2), X1
	MOVOU 32(SI)(AX*2), X2
	MOVOU 48(SI)(AX*2), X3
	POR X1, X0
	POR X2, X0
	POR X3, X0
	PAND (DI), X0
	PXOR X1, X1
	PCMPEQW X1, X0
	PMOVMSKB X0, DX
	CMPL DX, $0xffff
	JNE utf16le_ascii_westmere_return
	ADDQ $32, AX
	JMP utf16le_ascii_westmere_loop
utf16le_ascii_westmere_return:
	MOVQ AX, ret+24(FP)
	RET

// func validateUTF16BEASCIIPrefixWestmere(input []uint16) int
TEXT ·validateUTF16BEASCIIPrefixWestmere(SB), NOSPLIT, $0-32
	LEAQ ·utf16BEASCIIMask<>(SB), DI
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ AX, AX
utf16_ascii_westmere_loop:
	CMPQ AX, CX
	JAE utf16_ascii_westmere_return
	MOVOU 0(SI)(AX*2), X0
	MOVOU 16(SI)(AX*2), X1
	MOVOU 32(SI)(AX*2), X2
	MOVOU 48(SI)(AX*2), X3
	POR X1, X0
	POR X2, X0
	POR X3, X0
	PAND (DI), X0
	PXOR X1, X1
	PCMPEQW X1, X0
	PMOVMSKB X0, DX
	CMPL DX, $0xffff
	JNE utf16_ascii_westmere_return
	ADDQ $32, AX
	JMP utf16_ascii_westmere_loop
utf16_ascii_westmere_return:
	MOVQ AX, ret+24(FP)
	RET

// func validateUTF16LEASCIIPrefixHaswell(input []uint16) int
TEXT ·validateUTF16LEASCIIPrefixHaswell(SB), NOSPLIT, $0-32
	LEAQ ·utf16LEASCIIMask<>(SB), DI
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ AX, AX
utf16le_ascii_haswell_loop:
	CMPQ AX, CX
	JAE utf16le_ascii_haswell_return
	VMOVDQU 0(SI)(AX*2), Y0
	VMOVDQU 32(SI)(AX*2), Y1
	VPOR Y1, Y0, Y0
	VPAND (DI), Y0, Y0
	VPXOR Y1, Y1, Y1
	VPCMPEQW Y1, Y0, Y0
	VPMOVMSKB Y0, DX
	CMPL DX, $-1
	JNE utf16le_ascii_haswell_return
	ADDQ $32, AX
	JMP utf16le_ascii_haswell_loop
utf16le_ascii_haswell_return:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET

// func validateUTF16BEASCIIPrefixHaswell(input []uint16) int
TEXT ·validateUTF16BEASCIIPrefixHaswell(SB), NOSPLIT, $0-32
	LEAQ ·utf16BEASCIIMask<>(SB), DI
	MOVQ input_base+0(FP), SI
	MOVQ input_len+8(FP), CX
	ANDQ $-32, CX
	XORQ AX, AX
utf16_ascii_haswell_loop:
	CMPQ AX, CX
	JAE utf16_ascii_haswell_return
	VMOVDQU 0(SI)(AX*2), Y0
	VMOVDQU 32(SI)(AX*2), Y1
	VPOR Y1, Y0, Y0
	VPAND (DI), Y0, Y0
	VPXOR Y1, Y1, Y1
	VPCMPEQW Y1, Y0, Y0
	VPMOVMSKB Y0, DX
	CMPL DX, $-1
	JNE utf16_ascii_haswell_return
	ADDQ $32, AX
	JMP utf16_ascii_haswell_loop
utf16_ascii_haswell_return:
	VZEROUPPER
	MOVQ AX, ret+24(FP)
	RET
