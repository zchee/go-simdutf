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

import "simd/archsimd"

// Independently adapted from simdutf/simdutf at
// dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216,
// src/generic/utf8_validation/utf8_validator.h:10-80, and
// src/haswell/implementation.cpp:19-29. Complete 64-byte blocks use the
// pinned lookup4 checker. The existing scalar rewind wrapper supplies exact
// public error identity and Count and validates the final partial block.
//
// The Go 1.26.5 archsimd mapping is pinned to
// src/simd/archsimd/ops_amd64.go:1594-1613,4808-4830,5747-5807,
// 6416-6424,7310-7334; src/simd/archsimd/types_amd64.go:550-671; and
// src/simd/archsimd/slice_gen_amd64.go:149-162. These operations lower to
// VPERM2I128, VPALIGNR, VPSHUFB, VPSRLW, and VPSUBUSB under AVX2.

var utf8LookupHighArchsimd = [32]byte{
	0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
	0x80, 0x80, 0x80, 0x80, 0x21, 0x01, 0x15, 0x49,
	0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
	0x80, 0x80, 0x80, 0x80, 0x21, 0x01, 0x15, 0x49,
}

var utf8LookupLowArchsimd = [32]byte{
	0xe7, 0xa3, 0x83, 0x83, 0x8b, 0xcb, 0xcb, 0xcb,
	0xcb, 0xcb, 0xcb, 0xcb, 0xcb, 0xdb, 0xcb, 0xcb,
	0xe7, 0xa3, 0x83, 0x83, 0x8b, 0xcb, 0xcb, 0xcb,
	0xcb, 0xcb, 0xcb, 0xcb, 0xcb, 0xdb, 0xcb, 0xcb,
}

var utf8LookupInputArchsimd = [32]byte{
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0xe6, 0xae, 0xba, 0xba, 0x01, 0x01, 0x01, 0x01,
	0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
	0xe6, 0xae, 0xba, 0xba, 0x01, 0x01, 0x01, 0x01,
}

func validateUTF8Archsimd(input []byte) bool {
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixArchsimd(input)).Error == Success
}

func validateUTF8WithErrorsArchsimd(input []byte) Result {
	return validateUTF8AMD64FromPrefix(input, validateUTF8PrefixArchsimd(input))
}

func validateUTF8PrefixArchsimd(input []byte) int {
	limit := len(input) &^ 63
	var previous, errors archsimd.Uint8x32
	var zero archsimd.Int8x32
	for offset := 0; offset < limit; offset += 64 {
		current := archsimd.LoadUint8x32Slice(input[offset:])
		errors = errors.Or(checkUTF8BytesArchsimd(current, previous))
		previous = current
		current = archsimd.LoadUint8x32Slice(input[offset+32:])
		errors = errors.Or(checkUTF8BytesArchsimd(current, previous))
		previous = current
		if errors.AsInt8x32().Equal(zero).ToBits() != ^uint32(0) {
			return offset
		}
	}
	return limit
}

func checkUTF8BytesArchsimd(input, previous archsimd.Uint8x32) archsimd.Uint8x32 {
	bridge := previous.Select128FromPair(1, 2, input)
	prev1 := input.ConcatShiftBytesRightGrouped(15, bridge)
	prev2 := input.ConcatShiftBytesRightGrouped(14, bridge)
	prev3 := input.ConcatShiftBytesRightGrouped(13, bridge)

	nibbleMask := archsimd.BroadcastUint8x32(0x0f)
	prev1High := prev1.AsUint16x16().ShiftAllRight(4).AsUint8x32().And(nibbleMask)
	byte1High := archsimd.LoadUint8x32(&utf8LookupHighArchsimd).PermuteOrZeroGrouped(prev1High.AsInt8x32())
	byte1Low := archsimd.LoadUint8x32(&utf8LookupLowArchsimd).PermuteOrZeroGrouped(prev1.And(nibbleMask).AsInt8x32())
	inputHigh := input.AsUint16x16().ShiftAllRight(4).AsUint8x32().And(nibbleMask)
	byte2High := archsimd.LoadUint8x32(&utf8LookupInputArchsimd).PermuteOrZeroGrouped(inputHigh.AsInt8x32())
	specialCases := byte1High.And(byte1Low).And(byte2High)

	must23 := prev2.SubSaturated(archsimd.BroadcastUint8x32(0x60)).Or(
		prev3.SubSaturated(archsimd.BroadcastUint8x32(0x70)),
	).And(archsimd.BroadcastUint8x32(0x80))
	return must23.Xor(specialCases)
}
