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

// Independent Go assembly translations of util::find from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/find.h.
// Westmere uses four 16-byte PCMPEQ/PMOVMSKB lanes per 64-byte block; Haswell
// uses two 32-byte VPCMPEQ/VPMOVMSKB lanes. Match index is BSFQ of the combined
// bitmask (divided by two for UTF-16). Empty/nil returns len (0).

#include "textflag.h"

// func findWestmere(input []byte, value byte) int
TEXT ·findWestmere(SB), NOSPLIT|NOFRAME, $0-40
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), R11
	MOVBLZX value+24(FP), R8
	TESTQ   R11, R11
	JE      find_westmere_ret_len

	MOVD    R8, X0
	PUNPCKLBW X0, X0
	PUNPCKLBW X0, X0
	PSHUFL  $0, X0, X0

	MOVQ SI, DI
	LEAQ (SI)(R11*1), R9

	MOVQ SI, AX
	ANDQ $63, AX
	JZ   find_westmere_main

	MOVQ $64, BX
	SUBQ AX, BX
	MOVQ R9, CX
	SUBQ DI, CX
	CMPQ CX, BX
	JAE  find_westmere_prologue
	MOVQ CX, BX

find_westmere_prologue:
	TESTQ   BX, BX
	JE      find_westmere_main
	MOVBLZX (DI), AX
	CMPQ    AX, R8
	JE      find_westmere_found
	INCQ    DI
	DECQ    BX
	JMP     find_westmere_prologue

find_westmere_main:
	MOVQ R9, CX
	SUBQ DI, CX
	CMPQ CX, $64
	JB   find_westmere_tail

	MOVOU    0(DI), X1
	PCMPEQB  X0, X1
	PMOVMSKB X1, AX
	MOVOU    16(DI), X1
	PCMPEQB  X0, X1
	PMOVMSKB X1, DX
	SHLQ     $16, DX
	ORQ      DX, AX
	MOVOU    32(DI), X1
	PCMPEQB  X0, X1
	PMOVMSKB X1, DX
	SHLQ     $32, DX
	ORQ      DX, AX
	MOVOU    48(DI), X1
	PCMPEQB  X0, X1
	PMOVMSKB X1, DX
	SHLQ     $48, DX
	ORQ      DX, AX
	TESTQ    AX, AX
	JNZ      find_westmere_mask
	ADDQ     $64, DI
	JMP      find_westmere_main

find_westmere_mask:
	BSFQ AX, AX
	ADDQ AX, DI
	JMP  find_westmere_found

find_westmere_tail:
	CMPQ    DI, R9
	JAE     find_westmere_ret_len
	MOVBLZX (DI), AX
	CMPQ    AX, R8
	JE      find_westmere_found
	INCQ    DI
	JMP     find_westmere_tail

find_westmere_found:
	SUBQ SI, DI
	MOVQ DI, ret+32(FP)
	RET

find_westmere_ret_len:
	MOVQ R11, ret+32(FP)
	RET

// func findHaswell(input []byte, value byte) int
TEXT ·findHaswell(SB), NOSPLIT|NOFRAME, $0-40
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), R11
	MOVBLZX value+24(FP), R8
	TESTQ   R11, R11
	JE      find_haswell_ret_len

	MOVD         R8, X0
	VPBROADCASTB X0, Y0

	MOVQ SI, DI
	LEAQ (SI)(R11*1), R9

	MOVQ SI, AX
	ANDQ $63, AX
	JZ   find_haswell_main

	MOVQ $64, BX
	SUBQ AX, BX
	MOVQ R9, CX
	SUBQ DI, CX
	CMPQ CX, BX
	JAE  find_haswell_prologue
	MOVQ CX, BX

find_haswell_prologue:
	TESTQ   BX, BX
	JE      find_haswell_main
	MOVBLZX (DI), AX
	CMPQ    AX, R8
	JE      find_haswell_found
	INCQ    DI
	DECQ    BX
	JMP     find_haswell_prologue

find_haswell_main:
	MOVQ R9, CX
	SUBQ DI, CX
	CMPQ CX, $64
	JB   find_haswell_tail

	VMOVDQU   0(DI), Y1
	VPCMPEQB  Y0, Y1, Y1
	VPMOVMSKB Y1, AX
	VMOVDQU   32(DI), Y1
	VPCMPEQB  Y0, Y1, Y1
	VPMOVMSKB Y1, DX
	SHLQ      $32, DX
	ORQ       DX, AX
	TESTQ     AX, AX
	JNZ       find_haswell_mask
	ADDQ      $64, DI
	JMP       find_haswell_main

find_haswell_mask:
	BSFQ AX, AX
	ADDQ AX, DI
	JMP  find_haswell_found

find_haswell_tail:
	CMPQ    DI, R9
	JAE     find_haswell_ret_len
	MOVBLZX (DI), AX
	CMPQ    AX, R8
	JE      find_haswell_found
	INCQ    DI
	JMP     find_haswell_tail

find_haswell_found:
	SUBQ       SI, DI
	MOVQ       DI, ret+32(FP)
	VZEROUPPER
	RET

find_haswell_ret_len:
	MOVQ       R11, ret+32(FP)
	VZEROUPPER
	RET

// func findUTF16Westmere(input []uint16, value uint16) int
TEXT ·findUTF16Westmere(SB), NOSPLIT|NOFRAME, $0-40
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), R11
	MOVWQZX value+24(FP), R8
	TESTQ   R11, R11
	JE      find_utf16_westmere_ret_len

	MOVD    R8, X0
	PSHUFLW $0, X0, X0
	PSHUFL  $0, X0, X0

	MOVQ SI, DI
	LEAQ (SI)(R11*2), R9

	MOVQ SI, AX
	ANDQ $63, AX
	JZ   find_utf16_westmere_main
	TESTQ $1, AX
	JNZ  find_utf16_westmere_main

	MOVQ $64, BX
	SUBQ AX, BX
	SHRQ $1, BX
	MOVQ R9, CX
	SUBQ DI, CX
	SHRQ $1, CX
	CMPQ CX, BX
	JAE  find_utf16_westmere_prologue
	MOVQ CX, BX

find_utf16_westmere_prologue:
	TESTQ   BX, BX
	JE      find_utf16_westmere_main
	MOVWQZX (DI), AX
	CMPQ    AX, R8
	JE      find_utf16_westmere_found
	ADDQ    $2, DI
	DECQ    BX
	JMP     find_utf16_westmere_prologue

find_utf16_westmere_main:
	MOVQ R9, CX
	SUBQ DI, CX
	CMPQ CX, $64
	JB   find_utf16_westmere_tail

	MOVOU    0(DI), X1
	PCMPEQW  X0, X1
	PMOVMSKB X1, AX
	MOVOU    16(DI), X1
	PCMPEQW  X0, X1
	PMOVMSKB X1, DX
	SHLQ     $16, DX
	ORQ      DX, AX
	MOVOU    32(DI), X1
	PCMPEQW  X0, X1
	PMOVMSKB X1, DX
	SHLQ     $32, DX
	ORQ      DX, AX
	MOVOU    48(DI), X1
	PCMPEQW  X0, X1
	PMOVMSKB X1, DX
	SHLQ     $48, DX
	ORQ      DX, AX
	TESTQ    AX, AX
	JNZ      find_utf16_westmere_mask
	ADDQ     $64, DI
	JMP      find_utf16_westmere_main

find_utf16_westmere_mask:
	BSFQ AX, AX
	SHRQ $1, AX
	LEAQ (DI)(AX*2), DI
	JMP  find_utf16_westmere_found

find_utf16_westmere_tail:
	CMPQ    DI, R9
	JAE     find_utf16_westmere_ret_len
	MOVWQZX (DI), AX
	CMPQ    AX, R8
	JE      find_utf16_westmere_found
	ADDQ    $2, DI
	JMP     find_utf16_westmere_tail

find_utf16_westmere_found:
	SUBQ SI, DI
	SHRQ $1, DI
	MOVQ DI, ret+32(FP)
	RET

find_utf16_westmere_ret_len:
	MOVQ R11, ret+32(FP)
	RET

// func findUTF16Haswell(input []uint16, value uint16) int
TEXT ·findUTF16Haswell(SB), NOSPLIT|NOFRAME, $0-40
	MOVQ    input_base+0(FP), SI
	MOVQ    input_len+8(FP), R11
	MOVWQZX value+24(FP), R8
	TESTQ   R11, R11
	JE      find_utf16_haswell_ret_len

	MOVD         R8, X0
	VPBROADCASTW X0, Y0

	MOVQ SI, DI
	LEAQ (SI)(R11*2), R9

	MOVQ SI, AX
	ANDQ $63, AX
	JZ   find_utf16_haswell_main
	TESTQ $1, AX
	JNZ  find_utf16_haswell_main

	MOVQ $64, BX
	SUBQ AX, BX
	SHRQ $1, BX
	MOVQ R9, CX
	SUBQ DI, CX
	SHRQ $1, CX
	CMPQ CX, BX
	JAE  find_utf16_haswell_prologue
	MOVQ CX, BX

find_utf16_haswell_prologue:
	TESTQ   BX, BX
	JE      find_utf16_haswell_main
	MOVWQZX (DI), AX
	CMPQ    AX, R8
	JE      find_utf16_haswell_found
	ADDQ    $2, DI
	DECQ    BX
	JMP     find_utf16_haswell_prologue

find_utf16_haswell_main:
	MOVQ R9, CX
	SUBQ DI, CX
	CMPQ CX, $64
	JB   find_utf16_haswell_tail

	VMOVDQU   0(DI), Y1
	VPCMPEQW  Y0, Y1, Y1
	VPMOVMSKB Y1, AX
	VMOVDQU   32(DI), Y1
	VPCMPEQW  Y0, Y1, Y1
	VPMOVMSKB Y1, DX
	SHLQ      $32, DX
	ORQ       DX, AX
	TESTQ     AX, AX
	JNZ       find_utf16_haswell_mask
	ADDQ      $64, DI
	JMP       find_utf16_haswell_main

find_utf16_haswell_mask:
	BSFQ AX, AX
	SHRQ $1, AX
	LEAQ (DI)(AX*2), DI
	JMP  find_utf16_haswell_found

find_utf16_haswell_tail:
	CMPQ    DI, R9
	JAE     find_utf16_haswell_ret_len
	MOVWQZX (DI), AX
	CMPQ    AX, R8
	JE      find_utf16_haswell_found
	ADDQ    $2, DI
	JMP     find_utf16_haswell_tail

find_utf16_haswell_found:
	SUBQ       SI, DI
	SHRQ       $1, DI
	MOVQ       DI, ret+32(FP)
	VZEROUPPER
	RET

find_utf16_haswell_ret_len:
	MOVQ       R11, ret+32(FP)
	VZEROUPPER
	RET
