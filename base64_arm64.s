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

//go:build arm64

// Handwritten NEON Base64 block kernels adapted from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/arm64/arm_base64.cpp (encode_base64_impl, base64_decode_block) and
// src/generic/base64lengths.h (binary_length_from_base64). Callers pass
// lengths rounded down to the stated block size so these routines neither
// over-read nor over-write a slice.

#include "textflag.h"

// Standard Base64 alphabet (64 bytes).
DATA ·base64AlphabetDefault<>+0(SB)/8, $0x4847464544434241
DATA ·base64AlphabetDefault<>+8(SB)/8, $0x504f4e4d4c4b4a49
DATA ·base64AlphabetDefault<>+16(SB)/8, $0x5857565554535251
DATA ·base64AlphabetDefault<>+24(SB)/8, $0x6665646362615a59
DATA ·base64AlphabetDefault<>+32(SB)/8, $0x6e6d6c6b6a696867
DATA ·base64AlphabetDefault<>+40(SB)/8, $0x767574737271706f
DATA ·base64AlphabetDefault<>+48(SB)/8, $0x333231307a797877
DATA ·base64AlphabetDefault<>+56(SB)/8, $0x2f2b393837363534
GLOBL ·base64AlphabetDefault<>(SB), RODATA|NOPTR, $64

// URL Base64 alphabet (64 bytes).
DATA ·base64AlphabetURL<>+0(SB)/8, $0x4847464544434241
DATA ·base64AlphabetURL<>+8(SB)/8, $0x504f4e4d4c4b4a49
DATA ·base64AlphabetURL<>+16(SB)/8, $0x5857565554535251
DATA ·base64AlphabetURL<>+24(SB)/8, $0x6665646362615a59
DATA ·base64AlphabetURL<>+32(SB)/8, $0x6e6d6c6b6a696867
DATA ·base64AlphabetURL<>+40(SB)/8, $0x767574737271706f
DATA ·base64AlphabetURL<>+48(SB)/8, $0x333231307a797877
DATA ·base64AlphabetURL<>+56(SB)/8, $0x5f2d393837363534
GLOBL ·base64AlphabetURL<>(SB), RODATA|NOPTR, $64

// func countBase64SignificantBytesNEON(input []byte) int
TEXT ·countBase64SignificantBytesNEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-64, R1, R1
	MOVD $0, R2
	CBZ  R1, count_b64_bytes_done

	MOVD $0x21, R3
	VMOV R3, V16.B16
	MOVD $1, R3
	VMOV R3, V17.B16

count_b64_bytes_loop:
	VLD1.P 64(R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	VUMAX V16.B16, V0.B16, V4.B16
	VUMAX V16.B16, V1.B16, V5.B16
	VUMAX V16.B16, V2.B16, V6.B16
	VUMAX V16.B16, V3.B16, V7.B16
	VCMEQ V0.B16, V4.B16, V4.B16
	VCMEQ V1.B16, V5.B16, V5.B16
	VCMEQ V2.B16, V6.B16, V6.B16
	VCMEQ V3.B16, V7.B16, V7.B16
	VAND  V17.B16, V4.B16, V4.B16
	VAND  V17.B16, V5.B16, V5.B16
	VAND  V17.B16, V6.B16, V6.B16
	VAND  V17.B16, V7.B16, V7.B16
	VADDP V5.B16, V4.B16, V8.B16
	VADDP V7.B16, V6.B16, V9.B16
	VADDP V9.B16, V8.B16, V8.B16
	VUADDLV V8.B16, V10
	VMOV V10.D[0], R3
	ADD  R3, R2, R2
	SUB  $64, R1
	CBNZ R1, count_b64_bytes_loop

count_b64_bytes_done:
	MOVD R2, ret+24(FP)
	RET

// func countBase64SignificantUTF16NEON(input []uint16) int
TEXT ·countBase64SignificantUTF16NEON(SB), NOSPLIT|NOFRAME, $0-32
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	AND  $-32, R1, R1
	MOVD $0, R2
	CBZ  R1, count_b64_u16_done

	MOVD $0x21, R3
	VMOV R3, V16.H8
	MOVD $1, R3
	VMOV R3, V17.H8

count_b64_u16_loop:
	VLD1.P 64(R0), [V0.H8, V1.H8, V2.H8, V3.H8]
	VUMAX V16.H8, V0.H8, V4.H8
	VUMAX V16.H8, V1.H8, V5.H8
	VUMAX V16.H8, V2.H8, V6.H8
	VUMAX V16.H8, V3.H8, V7.H8
	VCMEQ V0.H8, V4.H8, V4.H8
	VCMEQ V1.H8, V5.H8, V5.H8
	VCMEQ V2.H8, V6.H8, V6.H8
	VCMEQ V3.H8, V7.H8, V7.H8
	VAND  V17.B16, V4.B16, V4.B16
	VAND  V17.B16, V5.B16, V5.B16
	VAND  V17.B16, V6.B16, V6.B16
	VAND  V17.B16, V7.B16, V7.B16
	VADDP V5.H8, V4.H8, V8.H8
	VADDP V7.H8, V6.H8, V9.H8
	VADDP V9.H8, V8.H8, V8.H8
	VUADDLV V8.H8, V10
	VMOV V10.D[0], R3
	ADD  R3, R2, R2
	SUB  $32, R1
	CBNZ R1, count_b64_u16_loop

count_b64_u16_done:
	MOVD R2, ret+24(FP)
	RET

// func binaryToBase64BlocksDefaultNEON(input, dst []byte)
// Caller must pass input_len as a multiple of 48 and dst large enough.
TEXT ·binaryToBase64BlocksDefaultNEON(SB), NOSPLIT|NOFRAME, $0-48
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	CBZ  R1, b64_enc_def_done
	MOVD $·base64AlphabetDefault<>(SB), R5
	VLD1 (R5), [V16.B16, V17.B16, V18.B16, V19.B16]
	MOVD $0x3f, R5
	VMOV R5, V20.B16
b64_enc_def_loop:
	VLD3 (R0), [V0.B16, V1.B16, V2.B16]
	ADD  $48, R0
	VUSHR $2, V0.B16, V4.B16
	VUSHR $4, V1.B16, V5.B16
	VSLI  $4, V0.B16, V5.B16
	VAND  V20.B16, V5.B16, V5.B16
	VUSHR $6, V2.B16, V6.B16
	VSLI  $2, V1.B16, V6.B16
	VAND  V20.B16, V6.B16, V6.B16
	VAND  V20.B16, V2.B16, V7.B16
	VTBL V4.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V4.B16
	VTBL V5.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V5.B16
	VTBL V6.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V6.B16
	VTBL V7.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V7.B16
	VST4 [V4.B16, V5.B16, V6.B16, V7.B16], (R2)
	ADD  $64, R2
	SUB  $48, R1
	CBNZ R1, b64_enc_def_loop
b64_enc_def_done:
	RET

// func binaryToBase64BlocksURLNEON(input, dst []byte)
TEXT ·binaryToBase64BlocksURLNEON(SB), NOSPLIT|NOFRAME, $0-48
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	CBZ  R1, b64_enc_url_done
	MOVD $·base64AlphabetURL<>(SB), R5
	VLD1 (R5), [V16.B16, V17.B16, V18.B16, V19.B16]
	MOVD $0x3f, R5
	VMOV R5, V20.B16
b64_enc_url_loop:
	VLD3 (R0), [V0.B16, V1.B16, V2.B16]
	ADD  $48, R0
	VUSHR $2, V0.B16, V4.B16
	VUSHR $4, V1.B16, V5.B16
	VSLI  $4, V0.B16, V5.B16
	VAND  V20.B16, V5.B16, V5.B16
	VUSHR $6, V2.B16, V6.B16
	VSLI  $2, V1.B16, V6.B16
	VAND  V20.B16, V6.B16, V6.B16
	VAND  V20.B16, V2.B16, V7.B16
	VTBL V4.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V4.B16
	VTBL V5.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V5.B16
	VTBL V6.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V6.B16
	VTBL V7.B16, [V16.B16, V17.B16, V18.B16, V19.B16], V7.B16
	VST4 [V4.B16, V5.B16, V6.B16, V7.B16], (R2)
	ADD  $64, R2
	SUB  $48, R1
	CBNZ R1, b64_enc_url_loop
b64_enc_url_done:
	RET

// func base64DecodeBlocksNEON(input, dst []byte)
// Caller must pass input_len as a multiple of 64 and dst large enough.
TEXT ·base64DecodeBlocksNEON(SB), NOSPLIT|NOFRAME, $0-48
	MOVD input_base+0(FP), R0
	MOVD input_len+8(FP), R1
	MOVD dst_base+24(FP), R2
	CBZ  R1, b64_decode_done
b64_decode_loop:
	VLD4 (R0), [V0.B16, V1.B16, V2.B16, V3.B16]
	ADD  $64, R0
	VUSHR $4, V1.B16, V4.B16
	VSLI  $2, V0.B16, V4.B16
	VUSHR $2, V2.B16, V5.B16
	VSLI  $4, V1.B16, V5.B16
	VORR V3.B16, V3.B16, V6.B16
	VSLI $6, V2.B16, V6.B16
	VST3 [V4.B16, V5.B16, V6.B16], (R2)
	ADD  $48, R2
	SUB  $64, R1
	CBNZ R1, b64_decode_loop
b64_decode_done:
	RET
