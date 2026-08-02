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

//go:build amd64 && goexperiment.simd

package simdutf

import (
	"math/bits"
	"simd/archsimd"
)

// Independently adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// (tree c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): src/generic/base64lengths.h and
// src/haswell/avx2_base64.cpp. Length counts units > 0x20 with AVX2 compare /
// OnesCount; encode owns complete 24→32 groups via VPSHUFB expand, MulHigh /
// Mul index packing, and lookup_pshufb_improved. Contiguous decode owns
// complete 64→48 groups via DotProductPairsSaturated / DotProductPairs and
// pack shuffle; residual/error paths stay scalar. Direct callers must satisfy
// the archsimd AVX2 guard. Public selection stays scalar-first until
// qualification promotes.
//
// Go 1.26.5 archsimd API provenance:
// src/simd/archsimd/slice_gen_amd64.go:149-162,50-53;
// src/simd/archsimd/ops_amd64.go:297,602,2052,2032,4050,4202,4505,4830,5992,6094,7143,
// 7334,8343,8355,8436,8517,8520,8644,8674; src/simd/archsimd/compare_gen_amd64.go:452;
// src/simd/archsimd/other_gen_amd64.go:101,137,155,297-300;
// src/simd/archsimd/slicepart_amd64.go; and src/simd/archsimd/extra_amd64.go:9-17.

// base64ArchsimdShuf is the AVX2 lane-expand control (duplicated 16-byte pattern).
// Memory order matches _mm256_set_epi8(10,11,9,10,...,1,2,0,1, ...) reversed per lane.
var base64ArchsimdShuf = [32]int8{
	1, 0, 2, 1, 4, 3, 5, 4, 7, 6, 8, 7, 10, 9, 11, 10,
	1, 0, 2, 1, 4, 3, 5, 4, 7, 6, 8, 7, 10, 9, 11, 10,
}

// setr_epi8 standard / URL shift LUTs from lookup_pshufb_improved, duplicated for AVX2.
var base64ArchsimdShiftStd = [32]uint8{
	71, 252, 252, 252, 252, 252, 252, 252, 252, 252, 252, 237, 240, 65, 0, 0,
	71, 252, 252, 252, 252, 252, 252, 252, 252, 252, 252, 237, 240, 65, 0, 0,
}

var base64ArchsimdShiftURL = [32]uint8{
	71, 252, 252, 252, 252, 252, 252, 252, 252, 252, 252, 239, 32, 65, 0, 0,
	71, 252, 252, 252, 252, 252, 252, 252, 252, 252, 252, 239, 32, 65, 0, 0,
}

// Decode block constants from haswell/avx2_base64.cpp (duplicated 16-byte lanes).
// Input bytes are pre-mapped 6-bit values; VPMADDUBSW/VPMADDWD pack four 6-bit
// indices into three output bytes, then VPSHUFB gathers them.
var base64ArchsimdDecMul0 = [32]int8{
	0x40, 0x01, 0x40, 0x01, 0x40, 0x01, 0x40, 0x01,
	0x40, 0x01, 0x40, 0x01, 0x40, 0x01, 0x40, 0x01,
	0x40, 0x01, 0x40, 0x01, 0x40, 0x01, 0x40, 0x01,
	0x40, 0x01, 0x40, 0x01, 0x40, 0x01, 0x40, 0x01,
}

var base64ArchsimdDecMul1 = [16]int16{
	0x1000, 0x0001, 0x1000, 0x0001, 0x1000, 0x0001, 0x1000, 0x0001,
	0x1000, 0x0001, 0x1000, 0x0001, 0x1000, 0x0001, 0x1000, 0x0001,
}

var base64ArchsimdDecPack = [32]int8{
	2, 1, 0, 6, 5, 4, 10, 9, 8, 14, 13, 12, -1, -1, -1, -1,
	2, 1, 0, 6, 5, 4, 10, 9, 8, 14, 13, 12, -1, -1, -1, -1,
}

func binaryLengthFromBase64Archsimd(input []byte) int {
	space := archsimd.BroadcastUint8x32(0x20)
	count := 0
	offset := 0
	for ; offset+32 <= len(input); offset += 32 {
		v := archsimd.LoadUint8x32Slice(input[offset:])
		count += bits.OnesCount32(v.Greater(space).ToBits())
	}
	archsimd.ClearAVXUpperBits()
	for _, c := range input[offset:] {
		if c > ' ' {
			count++
		}
	}
	padding := 0
	pos := len(input)
	for pos > 0 && padding < 2 {
		pos--
		c := input[pos]
		if c == '=' {
			padding++
		} else if c > ' ' {
			break
		}
	}
	return ((count - padding) * 3) / 4
}

func binaryLengthFromBase64UTF16Archsimd(input []uint16) int {
	space := archsimd.BroadcastUint16x16(0x20)
	count := 0
	offset := 0
	for ; offset+16 <= len(input); offset += 16 {
		v := archsimd.LoadUint16x16Slice(input[offset:])
		// Mask16x16.ToBits is AVX-512-only; reinterpret as bytes for AVX2 VPMOVMSKB.
		mask := v.Greater(space).ToInt16x16().AsInt8x32().ToMask().ToBits()
		count += bits.OnesCount32(mask)
	}
	archsimd.ClearAVXUpperBits()
	// Each matching uint16 sets two bits in the byte mask.
	count /= 2
	for _, c := range input[offset:] {
		if c > ' ' {
			count++
		}
	}
	padding := 0
	pos := len(input)
	for pos > 0 && padding < 2 {
		pos--
		c := input[pos]
		if c == '=' {
			padding++
		} else if c > ' ' {
			break
		}
	}
	return ((count - padding) * 3) / 4
}

func base64EncodeBlocksArchsimd(input, dst []byte, url bool) (consumed, written int) {
	shuf := archsimd.LoadInt8x32(&base64ArchsimdShuf)
	mask0 := archsimd.BroadcastUint32x8(0x0fc0fc00).AsUint8x32()
	mulHi := archsimd.BroadcastUint32x8(0x04000040).AsUint16x16()
	mask1 := archsimd.BroadcastUint32x8(0x003f03f0).AsUint8x32()
	mulLo := archsimd.BroadcastUint32x8(0x01000010).AsUint16x16()
	spl51 := archsimd.BroadcastUint8x32(51)
	spl26 := archsimd.BroadcastInt8x32(26)
	spl13 := archsimd.BroadcastUint8x32(13)
	shift := archsimd.LoadUint8x32(&base64ArchsimdShiftStd)
	if url {
		shift = archsimd.LoadUint8x32(&base64ArchsimdShiftURL)
	}

	var zero archsimd.Uint8x32
	i, o := 0, 0
	for i+28 <= len(input) && o+32 <= len(dst) {
		in := zero.
			SetLo(archsimd.LoadUint8x16Slice(input[i:])).
			SetHi(archsimd.LoadUint8x16Slice(input[i+12:])).
			PermuteOrZeroGrouped(shuf)

		t0 := in.And(mask0).AsUint16x16().MulHigh(mulHi).AsUint8x32()
		t1 := in.And(mask1).AsUint16x16().Mul(mulLo).AsUint8x32()
		indices := t0.Or(t1)

		result := indices.SubSaturated(spl51)
		less := spl26.Greater(indices.AsInt8x32()).ToInt8x32().AsUint8x32().And(spl13)
		result = result.Or(less)
		out := shift.PermuteOrZeroGrouped(result.AsInt8x32()).Add(indices)
		out.StoreSlice(dst[o:])

		i += 24
		o += 32
	}
	archsimd.ClearAVXUpperBits()
	return i, o
}

func binaryToBase64Archsimd(input, dst []byte, options Base64Options) int {
	required := base64LengthFromBinaryScalar(len(input), options)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	consumed, written := base64EncodeBlocksArchsimd(input, dst, options&Base64URL != 0)
	if consumed >= len(input) {
		return written
	}
	return written + tailEncodeBase64(dst[written:], input[consumed:], options, false, 0)
}

func binaryToBase64WithLinesArchsimd(input, dst []byte, lineLength int, options Base64Options) int {
	required := base64LengthFromBinaryWithLinesScalar(len(input), options, lineLength)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	if lineLength < 4 {
		lineLength = 4
	}
	url := options&Base64URL != 0
	out := 0
	col := 0
	inOff := 0
	for {
		var buf [128]byte
		consumed, written := base64EncodeBlocksArchsimd(input[inOff:], buf[:], url)
		if consumed == 0 {
			break
		}
		for j := 0; j < written; j++ {
			if col == lineLength {
				dst[out] = '\n'
				out++
				col = 0
			}
			dst[out] = buf[j]
			out++
			col++
		}
		inOff += consumed
	}
	if inOff < len(input) {
		var rem [72]byte
		n := tailEncodeBase64(rem[:], input[inOff:], options, false, 0)
		for j := 0; j < n; j++ {
			if col == lineLength {
				dst[out] = '\n'
				out++
				col = 0
			}
			dst[out] = rem[j]
			out++
			col++
		}
	}
	return out
}

// base64DecodeBlocksArchsimd converts pre-mapped 6-bit Base64 values to binary.
// Caller must pass input_len as a multiple of 64 and dst large enough for
// input_len/4*3 bytes. The final 12-byte store of each 48-byte group is
// width-safe (no +4 overrun).
func base64DecodeBlocksArchsimd(input, dst []byte) {
	mul0 := archsimd.LoadInt8x32(&base64ArchsimdDecMul0)
	mul1 := archsimd.LoadInt16x16(&base64ArchsimdDecMul1)
	pack := archsimd.LoadInt8x32(&base64ArchsimdDecPack)

	i, o := 0, 0
	for i+64 <= len(input) {
		in0 := archsimd.LoadUint8x32Slice(input[i:])
		p0 := in0.DotProductPairsSaturated(mul0).DotProductPairs(mul1).AsUint8x32().PermuteOrZeroGrouped(pack)
		p0.GetLo().StoreSlicePart(dst[o : o+12])
		p0.GetHi().StoreSlicePart(dst[o+12 : o+24])

		in1 := archsimd.LoadUint8x32Slice(input[i+32:])
		p1 := in1.DotProductPairsSaturated(mul0).DotProductPairs(mul1).AsUint8x32().PermuteOrZeroGrouped(pack)
		p1.GetLo().StoreSlicePart(dst[o+24 : o+36])
		p1.GetHi().StoreSlicePart(dst[o+36 : o+48])

		i += 64
		o += 48
	}
	archsimd.ClearAVXUpperBits()
}

// Decode/details: contiguous hot path uses AVX2 64→48 blocks; short,
// whitespace, garbage-accept, and ignore-garbage payloads stay scalar.

//go:noinline
func base64ToBinaryArchsimd(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	required := maximalBinaryLengthFromBase64Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return base64ToBinaryDetailsArchsimd(input, dst, options, lastChunk).Result()
}

//go:noinline
func base64ToBinaryUTF16Archsimd(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	required := maximalBinaryLengthFromBase64UTF16Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return base64ToBinaryDetailsUTF16Archsimd(input, dst, options, lastChunk).Result()
}

//go:noinline
func base64ToBinaryDetailsArchsimd(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	if fr, ok := base64ToBinaryDetailsAMD64Contiguous(input, dst, options, lastChunk, base64DecodeBlocksArchsimd); ok {
		return fr
	}
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk)
}

//go:noinline
func base64ToBinaryDetailsUTF16Archsimd(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	if fr, ok := base64ToBinaryDetailsUTF16AMD64Contiguous(input, dst, options, lastChunk, base64DecodeBlocksArchsimd); ok {
		return fr
	}
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk)
}
