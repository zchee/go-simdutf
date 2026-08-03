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

package simdutf

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/base64.h and include/simdutf/base64_implementation.h.

func base64IgnoreGarbage(options Base64Options) bool {
	return options == Base64URLAcceptGarbage ||
		options == Base64DefaultAcceptGarbage ||
		options == Base64DefaultOrURLAcceptGarbage
}

func base64ToValueTable(options Base64Options) *[256]byte {
	if options&Base64DefaultOrURL != 0 {
		return &toBase64DefaultOrUrlValue
	}
	if options&Base64URL != 0 {
		return &toBase64UrlValue
	}
	return &toBase64Value
}

func base64DecodeTables(options Base64Options) (d0, d1, d2, d3 *[256]uint32) {
	if options&Base64DefaultOrURL != 0 {
		return &base64DefaultOrURLD0, &base64DefaultOrURLD1, &base64DefaultOrURLD2, &base64DefaultOrURLD3
	}
	if options&Base64URL != 0 {
		return &base64URLD0, &base64URLD1, &base64URLD2, &base64URLD3
	}
	return &base64DefaultD0, &base64DefaultD1, &base64DefaultD2, &base64DefaultD3
}

func base64EncodeTables(options Base64Options) (e0, e1, e2 *[256]byte) {
	if options&Base64URL != 0 {
		return &base64URLE0, &base64URLE1, &base64URLE2
	}
	return &base64DefaultE0, &base64DefaultE1, &base64DefaultE2
}

func base64UsePadding(options Base64Options) bool {
	url := (options & Base64URL) == 0
	reverse := (options & Base64Options(Base64ReversePadding)) == Base64Options(Base64ReversePadding)
	return url != reverse // XOR: ((url==0) ^ reverse_padding)
}

func isEightByteUTF16(c uint16) bool { return c <= 0xff }

func isIgnorableByte(c byte, options Base64Options) bool {
	toBase64 := base64ToValueTable(options)
	code := toBase64[c]
	if code <= 63 {
		return false
	}
	if code == 64 {
		return true
	}
	return base64IgnoreGarbage(options)
}

func isIgnorableUTF16(c uint16, options Base64Options) bool {
	if !isEightByteUTF16(c) {
		return base64IgnoreGarbage(options)
	}
	return isIgnorableByte(byte(c), options)
}

func isBase64Byte(c byte, options Base64Options) bool {
	return base64ToValueTable(options)[c] <= 63
}

func isBase64UTF16(c uint16, options Base64Options) bool {
	return isEightByteUTF16(c) && isBase64Byte(byte(c), options)
}

func isBase64OrPaddingByte(c byte, options Base64Options) bool {
	if c == '=' {
		return true
	}
	return isBase64Byte(c, options)
}

func isBase64OrPaddingUTF16(c uint16, options Base64Options) bool {
	if c == '=' {
		return true
	}
	return isBase64UTF16(c, options)
}

type base64ReducedInput struct {
	equalSigns      int
	equalLocation   int
	srcLen          int
	fullInputLength int
}

func findEndBase64Byte(src []byte, options Base64Options) base64ReducedInput {
	toBase64 := base64ToValueTable(options)
	ignoreGarbage := base64IgnoreGarbage(options)
	srclen := len(src)
	fullInputLength := srclen
	equalSigns := 0
	for !ignoreGarbage && srclen > 0 && toBase64[src[srclen-1]] == 64 {
		srclen--
	}
	equalLocation := srclen
	if ignoreGarbage {
		for i := 0; i < srclen; i++ {
			if src[i] == '=' {
				equalLocation = i
				equalSigns = 1
				srclen = equalLocation
				fullInputLength = equalLocation + 1
				break
			}
		}
		return base64ReducedInput{equalSigns, equalLocation, srclen, fullInputLength}
	}
	if srclen > 0 && src[srclen-1] == '=' {
		equalLocation = srclen - 1
		srclen--
		equalSigns = 1
		for srclen > 0 && toBase64[src[srclen-1]] == 64 {
			srclen--
		}
		if srclen > 0 && src[srclen-1] == '=' {
			equalLocation = srclen - 1
			srclen--
			equalSigns = 2
		}
	}
	return base64ReducedInput{equalSigns, equalLocation, srclen, fullInputLength}
}

func findEndBase64UTF16(src []uint16, options Base64Options) base64ReducedInput {
	toBase64 := base64ToValueTable(options)
	ignoreGarbage := base64IgnoreGarbage(options)
	srclen := len(src)
	fullInputLength := srclen
	equalSigns := 0
	for !ignoreGarbage && srclen > 0 && isEightByteUTF16(src[srclen-1]) && toBase64[byte(src[srclen-1])] == 64 {
		srclen--
	}
	equalLocation := srclen
	if ignoreGarbage {
		for i := 0; i < srclen; i++ {
			if src[i] == '=' {
				equalLocation = i
				equalSigns = 1
				srclen = equalLocation
				fullInputLength = equalLocation + 1
				break
			}
		}
		return base64ReducedInput{equalSigns, equalLocation, srclen, fullInputLength}
	}
	if srclen > 0 && src[srclen-1] == '=' {
		equalLocation = srclen - 1
		srclen--
		equalSigns = 1
		for srclen > 0 && isEightByteUTF16(src[srclen-1]) && toBase64[byte(src[srclen-1])] == 64 {
			srclen--
		}
		if srclen > 0 && src[srclen-1] == '=' {
			equalLocation = srclen - 1
			srclen--
			equalSigns = 2
		}
	}
	return base64ReducedInput{equalSigns, equalLocation, srclen, fullInputLength}
}

func patchTailResult(r FullResult, previousInput, previousOutput, equalLocation, fullInputLength int, lastChunk LastChunkHandlingOptions) FullResult {
	r.InputCount += previousInput
	r.OutputCount += previousOutput
	if r.PaddingError {
		r.InputCount = equalLocation
	}
	if r.Error == Success {
		if !IsPartial(lastChunk) {
			r.InputCount = fullInputLength
		} else if r.OutputCount%3 != 0 {
			r.InputCount = fullInputLength
		}
	}
	return r
}

func base64TailDecodeByte(dst, src []byte, paddingCharacters int, options Base64Options, lastChunk LastChunkHandlingOptions, checkCapacity bool) FullResult {
	return base64TailDecodeImplByte(dst, src, paddingCharacters, options, lastChunk, checkCapacity)
}

func base64TailDecodeUTF16(dst []byte, src []uint16, paddingCharacters int, options Base64Options, lastChunk LastChunkHandlingOptions, checkCapacity bool) FullResult {
	return base64TailDecodeImplUTF16(dst, src, paddingCharacters, options, lastChunk, checkCapacity)
}

func maximalBinaryLengthFromBase64Scalar(input []byte) int {
	padding := 0
	if n := len(input); n > 0 {
		if input[n-1] == '=' {
			padding++
			if n > 1 && input[n-2] == '=' {
				padding++
			}
		}
	}
	actual := len(input) - padding
	if actual%4 <= 1 {
		return actual / 4 * 3
	}
	return actual/4*3 + actual%4 - 1
}

func maximalBinaryLengthFromBase64UTF16Scalar(input []uint16) int {
	padding := 0
	if n := len(input); n > 0 {
		if input[n-1] == '=' {
			padding++
			if n > 1 && input[n-2] == '=' {
				padding++
			}
		}
	}
	actual := len(input) - padding
	if actual%4 <= 1 {
		return actual / 4 * 3
	}
	return actual/4*3 + actual%4 - 1
}

func binaryLengthFromBase64Scalar(input []byte) int {
	count := 0
	for _, c := range input {
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

func binaryLengthFromBase64UTF16Scalar(input []uint16) int {
	count := 0
	for _, c := range input {
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

func base64LengthFromBinaryScalar(length int, options Base64Options) int {
	if !base64UsePadding(options) {
		if length%3 != 0 {
			return length/3*4 + (length%3 + 1)
		}
		return length / 3 * 4
	}
	return (length + 2) / 3 * 4
}

func base64LengthFromBinaryWithLinesScalar(length int, options Base64Options, lineLength int) int {
	if length == 0 {
		return 0
	}
	base64Length := base64LengthFromBinaryScalar(length, options)
	if lineLength < 4 {
		lineLength = 4
	}
	lines := (base64Length + lineLength - 1) / lineLength
	return base64Length + lines - 1
}

func base64TailDecodeImplByte(dst, src []byte, paddingCharacters int, options Base64Options, lastChunk LastChunkHandlingOptions, checkCapacity bool) FullResult {
	return base64TailDecodeImplGeneric(
		dst,
		len(src),
		paddingCharacters,
		options,
		lastChunk,
		checkCapacity,
		func(i int) (byte, bool) { // value, okEight
			return src[i], true
		},
	)
}

func base64TailDecodeImplUTF16(dst []byte, src []uint16, paddingCharacters int, options Base64Options, lastChunk LastChunkHandlingOptions, checkCapacity bool) FullResult {
	return base64TailDecodeImplGeneric(
		dst,
		len(src),
		paddingCharacters,
		options,
		lastChunk,
		checkCapacity,
		func(i int) (byte, bool) {
			c := src[i]
			if c > 0xff {
				return 0, false
			}
			return byte(c), true
		},
	)
}

func base64TailDecodeImplGeneric(
	dst []byte,
	srcLen int,
	paddingCharacters int,
	options Base64Options,
	lastChunk LastChunkHandlingOptions,
	checkCapacity bool,
	at func(i int) (byte, bool),
) FullResult {
	toBase64 := base64ToValueTable(options)
	d0, d1, d2, d3 := base64DecodeTables(options)
	ignoreGarbage := base64IgnoreGarbage(options)

	src := 0
	dstPos := 0
	dstEnd := len(dst)

	for {
		for src+4 <= srcLen {
			c0, ok0 := at(src)
			c1, ok1 := at(src + 1)
			c2, ok2 := at(src + 2)
			c3, ok3 := at(src + 3)
			if !(ok0 && ok1 && ok2 && ok3) {
				break
			}
			x := d0[c0] | d1[c1] | d2[c2] | d3[c3]
			if x >= 0x01FFFFFF {
				break
			}
			if checkCapacity && dstEnd-dstPos < 3 {
				return FullResult{Error: OutputBufferTooSmall, InputCount: src, OutputCount: dstPos}
			}
			dst[dstPos] = byte(x)
			dst[dstPos+1] = byte(x >> 8)
			dst[dstPos+2] = byte(x >> 16)
			dstPos += 3
			src += 4
		}

		srcCur := src
		idx := 0
		var buffer [4]byte

		for idx < 4 && src < srcLen {
			c, okEight := at(src)
			code := byte(0xFF)
			if okEight {
				code = toBase64[c]
			}
			if okEight && code <= 63 {
				buffer[idx] = code
				idx++
				src++
				continue
			}
			if okEight && code == 64 { // whitespace/ignorable class in tables
				src++
				continue
			}
			if !okEight {
				if ignoreGarbage {
					src++
					continue
				}
				return FullResult{Error: InvalidBase64Character, InputCount: src, OutputCount: dstPos}
			}
			// code > 64 => '=' or invalid
			if c == '=' {
				break
			}
			if ignoreGarbage {
				src++
				continue
			}
			return FullResult{Error: InvalidBase64Character, InputCount: src, OutputCount: dstPos}
		}

		if idx < 4 {
			// Loose padding combination checks (match upstream scalar).
			if !ignoreGarbage && lastChunk == Loose && idx >= 2 && paddingCharacters > 0 && ((idx+paddingCharacters)&3) != 0 {
				return FullResult{Error: InvalidBase64Character, InputCount: src, OutputCount: dstPos, PaddingError: true}
			}
			if !ignoreGarbage && lastChunk == Strict && idx >= 2 && ((idx+paddingCharacters)&3) != 0 {
				return FullResult{Error: Base64InputRemainder, InputCount: src, OutputCount: dstPos, PaddingError: true}
			}
			if (lastChunk == StopBeforePartial && (paddingCharacters+idx < 4) && idx != 0 && (idx >= 2 || paddingCharacters == 0)) ||
				(lastChunk == OnlyFullChunks && (idx >= 2 || paddingCharacters == 0)) {
				src = srcCur
				return FullResult{Error: Success, InputCount: src, OutputCount: dstPos}
			}
			if idx == 2 {
				triple := (uint32(buffer[0]) << 18) + (uint32(buffer[1]) << 12)
				if !ignoreGarbage && lastChunk == Strict && (triple&0xffff) != 0 {
					return FullResult{Error: Base64ExtraBits, InputCount: src, OutputCount: dstPos}
				}
				if checkCapacity && dstEnd-dstPos < 1 {
					return FullResult{Error: OutputBufferTooSmall, InputCount: srcCur, OutputCount: dstPos}
				}
				dst[dstPos] = byte((triple >> 16) & 0xFF)
				dstPos++
			} else if idx == 3 {
				triple := (uint32(buffer[0]) << 18) + (uint32(buffer[1]) << 12) + (uint32(buffer[2]) << 6)
				if !ignoreGarbage && lastChunk == Strict && (triple&0xff) != 0 {
					return FullResult{Error: Base64ExtraBits, InputCount: src, OutputCount: dstPos}
				}
				if checkCapacity && dstEnd-dstPos < 2 {
					return FullResult{Error: OutputBufferTooSmall, InputCount: srcCur, OutputCount: dstPos}
				}
				dst[dstPos] = byte((triple >> 16) & 0xFF)
				dst[dstPos+1] = byte((triple >> 8) & 0xFF)
				dstPos += 2
			} else if !ignoreGarbage && idx == 1 && (!IsPartial(lastChunk) || (IsPartial(lastChunk) && paddingCharacters > 0)) {
				return FullResult{Error: Base64InputRemainder, InputCount: src, OutputCount: dstPos}
			} else if !ignoreGarbage && idx == 0 && paddingCharacters > 0 {
				return FullResult{Error: InvalidBase64Character, InputCount: src, OutputCount: dstPos, PaddingError: true}
			}
			return FullResult{Error: Success, InputCount: src, OutputCount: dstPos}
		}

		if checkCapacity && dstEnd-dstPos < 3 {
			return FullResult{Error: OutputBufferTooSmall, InputCount: srcCur, OutputCount: dstPos}
		}
		triple := (uint32(buffer[0]) << 18) + (uint32(buffer[1]) << 12) + (uint32(buffer[2]) << 6) + uint32(buffer[3])
		dst[dstPos] = byte((triple >> 16) & 0xFF)
		dst[dstPos+1] = byte((triple >> 8) & 0xFF)
		dst[dstPos+2] = byte(triple & 0xFF)
		dstPos += 3
	}
}

func base64ToBinaryDetailsScalar(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsByte(input, dst, options, lastChunk, false)
}

func base64ToBinaryDetailsUTF16Scalar(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsUTF16Inner(input, dst, options, lastChunk, false)
}

func base64ToBinaryDetailsSafeScalar(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsByte(input, dst, options, lastChunk, true)
}

func base64ToBinaryDetailsSafeUTF16Scalar(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) FullResult {
	return base64ToBinaryDetailsUTF16Inner(input, dst, options, lastChunk, true)
}

func base64ToBinaryDetailsByte(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions, checkCapacity bool) FullResult {
	ignoreGarbage := base64IgnoreGarbage(options)
	ri := findEndBase64Byte(input, options)
	equalsigns := ri.equalSigns
	equallocation := ri.equalLocation
	length := ri.srcLen
	fullInputLength := ri.fullInputLength
	if length == 0 {
		if !ignoreGarbage && equalsigns > 0 {
			return FullResult{Error: InvalidBase64Character, InputCount: equallocation, OutputCount: 0, PaddingError: true}
		}
		return FullResult{Error: Success, InputCount: fullInputLength, OutputCount: 0}
	}
	r := base64TailDecodeByte(dst, input[:length], equalsigns, options, lastChunk, checkCapacity)
	r = patchTailResult(r, 0, 0, equallocation, fullInputLength, lastChunk)
	if !IsPartial(lastChunk) && r.Error == Success && equalsigns > 0 && !ignoreGarbage {
		if (r.OutputCount%3 == 0) || ((r.OutputCount%3)+1+equalsigns != 4) {
			return FullResult{Error: InvalidBase64Character, InputCount: equallocation, OutputCount: r.OutputCount, PaddingError: true}
		}
	}
	if IsPartial(lastChunk) && r.Error == Success && r.InputCount < fullInputLength {
		for r.InputCount < fullInputLength && isIgnorableByte(input[r.InputCount], options) {
			r.InputCount++
		}
		if r.InputCount < fullInputLength {
			for r.InputCount > 0 && isIgnorableByte(input[r.InputCount-1], options) {
				r.InputCount--
			}
		}
	}
	return r
}

func base64ToBinaryDetailsUTF16Inner(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions, checkCapacity bool) FullResult {
	ignoreGarbage := base64IgnoreGarbage(options)
	ri := findEndBase64UTF16(input, options)
	equalsigns := ri.equalSigns
	equallocation := ri.equalLocation
	length := ri.srcLen
	fullInputLength := ri.fullInputLength
	if length == 0 {
		if !ignoreGarbage && equalsigns > 0 {
			return FullResult{Error: InvalidBase64Character, InputCount: equallocation, OutputCount: 0, PaddingError: true}
		}
		return FullResult{Error: Success, InputCount: fullInputLength, OutputCount: 0}
	}
	r := base64TailDecodeUTF16(dst, input[:length], equalsigns, options, lastChunk, checkCapacity)
	r = patchTailResult(r, 0, 0, equallocation, fullInputLength, lastChunk)
	if !IsPartial(lastChunk) && r.Error == Success && equalsigns > 0 && !ignoreGarbage {
		if (r.OutputCount%3 == 0) || ((r.OutputCount%3)+1+equalsigns != 4) {
			return FullResult{Error: InvalidBase64Character, InputCount: equallocation, OutputCount: r.OutputCount, PaddingError: true}
		}
	}
	if IsPartial(lastChunk) && r.Error == Success && r.InputCount < fullInputLength {
		for r.InputCount < fullInputLength && isIgnorableUTF16(input[r.InputCount], options) {
			r.InputCount++
		}
		if r.InputCount < fullInputLength {
			for r.InputCount > 0 && isIgnorableUTF16(input[r.InputCount-1], options) {
				r.InputCount--
			}
		}
	}
	return r
}

func base64ToBinaryScalar(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	required := maximalBinaryLengthFromBase64Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return base64ToBinaryDetailsScalar(input, dst, options, lastChunk).Result()
}

func base64ToBinaryUTF16Scalar(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) Result {
	required := maximalBinaryLengthFromBase64UTF16Scalar(input)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return base64ToBinaryDetailsUTF16Scalar(input, dst, options, lastChunk).Result()
}

func binaryToBase64Scalar(input, dst []byte, options Base64Options) int {
	required := base64LengthFromBinaryScalar(len(input), options)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	return tailEncodeBase64(dst, input, options, false, 0)
}

func binaryToBase64WithLinesScalar(input, dst []byte, lineLength int, options Base64Options) int {
	required := base64LengthFromBinaryWithLinesScalar(len(input), options, lineLength)
	if len(dst) < required {
		panic("simdutf: destination is too short")
	}
	if lineLength < 4 {
		lineLength = 4
	}
	return tailEncodeBase64(dst, input, options, true, lineLength)
}

func tailEncodeBase64(dst, src []byte, options Base64Options, useLines bool, lineLength int) int {
	e0, e1, e2 := base64EncodeTables(options)
	usePadding := base64UsePadding(options)
	out := 0
	lineOffset := 0
	i := 0
	srclen := len(src)
	for ; i+2 < srclen; i += 3 {
		t1 := src[i]
		t2 := src[i+1]
		t3 := src[i+2]
		if useLines {
			if lineOffset+3 >= lineLength {
				if lineOffset == lineLength {
					dst[out] = '\n'
					out++
					dst[out] = e0[t1]
					dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
					dst[out+2] = e1[((t2&0x0F)<<2)|((t3>>6)&0x03)]
					dst[out+3] = e2[t3]
					out += 4
					lineOffset = 4
				} else if lineOffset+1 == lineLength {
					dst[out] = e0[t1]
					dst[out+1] = '\n'
					dst[out+2] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
					dst[out+3] = e1[((t2&0x0F)<<2)|((t3>>6)&0x03)]
					dst[out+4] = e2[t3]
					out += 5
					lineOffset = 3
				} else if lineOffset+2 == lineLength {
					dst[out] = e0[t1]
					dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
					dst[out+2] = '\n'
					dst[out+3] = e1[((t2&0x0F)<<2)|((t3>>6)&0x03)]
					dst[out+4] = e2[t3]
					out += 5
					lineOffset = 2
				} else { // lineOffset+3 == lineLength
					dst[out] = e0[t1]
					dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
					dst[out+2] = e1[((t2&0x0F)<<2)|((t3>>6)&0x03)]
					dst[out+3] = '\n'
					dst[out+4] = e2[t3]
					out += 5
					lineOffset = 1
				}
			} else {
				dst[out] = e0[t1]
				dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
				dst[out+2] = e1[((t2&0x0F)<<2)|((t3>>6)&0x03)]
				dst[out+3] = e2[t3]
				out += 4
				lineOffset += 4
			}
		} else {
			dst[out] = e0[t1]
			dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
			dst[out+2] = e1[((t2&0x0F)<<2)|((t3>>6)&0x03)]
			dst[out+3] = e2[t3]
			out += 4
		}
	}
	switch srclen - i {
	case 1:
		t1 := src[i]
		if useLines {
			if usePadding {
				if lineOffset+3 >= lineLength {
					if lineOffset == lineLength {
						dst[out] = '\n'
						out++
						dst[out] = e0[t1]
						dst[out+1] = e1[(t1&0x03)<<4]
						dst[out+2] = '='
						dst[out+3] = '='
						out += 4
					} else if lineOffset+1 == lineLength {
						dst[out] = e0[t1]
						dst[out+1] = '\n'
						dst[out+2] = e1[(t1&0x03)<<4]
						dst[out+3] = '='
						dst[out+4] = '='
						out += 5
					} else if lineOffset+2 == lineLength {
						dst[out] = e0[t1]
						dst[out+1] = e1[(t1&0x03)<<4]
						dst[out+2] = '\n'
						dst[out+3] = '='
						dst[out+4] = '='
						out += 5
					} else {
						dst[out] = e0[t1]
						dst[out+1] = e1[(t1&0x03)<<4]
						dst[out+2] = '='
						dst[out+3] = '\n'
						dst[out+4] = '='
						out += 5
					}
				} else {
					dst[out] = e0[t1]
					dst[out+1] = e1[(t1&0x03)<<4]
					dst[out+2] = '='
					dst[out+3] = '='
					out += 4
				}
			} else {
				if lineOffset+1 >= lineLength {
					if lineOffset == lineLength {
						dst[out] = '\n'
						out++
						dst[out] = e0[t1]
						dst[out+1] = e1[(t1&0x03)<<4]
						out += 2
					} else {
						dst[out] = e0[t1]
						out++
						if lineOffset+1 == lineLength {
							dst[out] = '\n'
							out++
							dst[out] = e1[(t1&0x03)<<4]
							out++
						} else {
							dst[out] = e1[(t1&0x03)<<4]
							out++
						}
					}
				} else {
					dst[out] = e0[t1]
					dst[out+1] = e1[(t1&0x03)<<4]
					out += 2
				}
			}
		} else {
			dst[out] = e0[t1]
			dst[out+1] = e1[(t1&0x03)<<4]
			out += 2
			if usePadding {
				dst[out] = '='
				dst[out+1] = '='
				out += 2
			}
		}
	case 2:
		t1 := src[i]
		t2 := src[i+1]
		if useLines {
			if usePadding {
				if lineOffset+3 >= lineLength {
					if lineOffset == lineLength {
						dst[out] = '\n'
						out++
						dst[out] = e0[t1]
						dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+2] = e2[(t2&0x0F)<<2]
						dst[out+3] = '='
						out += 4
					} else if lineOffset+1 == lineLength {
						dst[out] = e0[t1]
						dst[out+1] = '\n'
						dst[out+2] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+3] = e2[(t2&0x0F)<<2]
						dst[out+4] = '='
						out += 5
					} else if lineOffset+2 == lineLength {
						dst[out] = e0[t1]
						dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+2] = '\n'
						dst[out+3] = e2[(t2&0x0F)<<2]
						dst[out+4] = '='
						out += 5
					} else {
						dst[out] = e0[t1]
						dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+2] = e2[(t2&0x0F)<<2]
						dst[out+3] = '\n'
						dst[out+4] = '='
						out += 5
					}
				} else {
					dst[out] = e0[t1]
					dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
					dst[out+2] = e2[(t2&0x0F)<<2]
					dst[out+3] = '='
					out += 4
				}
			} else {
				if lineOffset+2 >= lineLength {
					if lineOffset == lineLength {
						dst[out] = '\n'
						out++
						dst[out] = e0[t1]
						dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+2] = e2[(t2&0x0F)<<2]
						out += 3
					} else if lineOffset+1 == lineLength {
						dst[out] = e0[t1]
						dst[out+1] = '\n'
						dst[out+2] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+3] = e2[(t2&0x0F)<<2]
						out += 4
					} else if lineOffset+2 == lineLength {
						dst[out] = e0[t1]
						dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+2] = '\n'
						dst[out+3] = e2[(t2&0x0F)<<2]
						out += 4
					} else {
						dst[out] = e0[t1]
						dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
						dst[out+2] = e2[(t2&0x0F)<<2]
						out += 3
					}
				} else {
					dst[out] = e0[t1]
					dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
					dst[out+2] = e2[(t2&0x0F)<<2]
					out += 3
				}
			}
		} else {
			dst[out] = e0[t1]
			dst[out+1] = e1[((t1&0x03)<<4)|((t2>>4)&0x0F)]
			dst[out+2] = e2[(t2&0x0F)<<2]
			out += 3
			if usePadding {
				dst[out] = '='
				out++
			}
		}
	}
	return out
}

func base64ToBinarySafeScalar(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions, decodeUpToBadChar bool) (Result, int) {
	if decodeUpToBadChar {
		return slowBase64ToBinarySafeByte(input, dst, options, lastChunk)
	}
	r := base64ToBinaryDetailsSafeScalar(input, dst, options, lastChunk)
	return Result{Error: r.Error, Count: r.InputCount}, r.OutputCount
}

func base64ToBinarySafeUTF16Scalar(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions, decodeUpToBadChar bool) (Result, int) {
	if decodeUpToBadChar {
		return slowBase64ToBinarySafeUTF16(input, dst, options, lastChunk)
	}
	r := base64ToBinaryDetailsSafeUTF16Scalar(input, dst, options, lastChunk)
	return Result{Error: r.Error, Count: r.InputCount}, r.OutputCount
}

func slowBase64ToBinarySafeByte(input, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) (Result, int) {
	ignoreGarbage := base64IgnoreGarbage(options)
	ri := findEndBase64Byte(input, options)
	if ri.srcLen == 0 {
		if !ignoreGarbage && ri.equalSigns > 0 {
			return Result{Error: InvalidBase64Character, Count: ri.equalLocation}, 0
		}
		return Result{Error: Success, Count: 0}, 0
	}
	r := base64TailDecodeByte(dst, input[:ri.srcLen], ri.equalSigns, options, lastChunk, true)
	r = patchTailResult(r, 0, 0, ri.equalLocation, ri.fullInputLength, lastChunk)
	if !IsPartial(lastChunk) && r.Error == Success && ri.equalSigns > 0 {
		outlen := r.OutputCount
		if (outlen%3 == 0) || ((outlen%3)+1+ri.equalSigns != 4) {
			r.Error = InvalidBase64Character
		}
	}
	return Result{Error: r.Error, Count: r.InputCount}, r.OutputCount
}

func slowBase64ToBinarySafeUTF16(input []uint16, dst []byte, options Base64Options, lastChunk LastChunkHandlingOptions) (Result, int) {
	ignoreGarbage := base64IgnoreGarbage(options)
	ri := findEndBase64UTF16(input, options)
	if ri.srcLen == 0 {
		if !ignoreGarbage && ri.equalSigns > 0 {
			return Result{Error: InvalidBase64Character, Count: ri.equalLocation}, 0
		}
		return Result{Error: Success, Count: 0}, 0
	}
	r := base64TailDecodeUTF16(dst, input[:ri.srcLen], ri.equalSigns, options, lastChunk, true)
	r = patchTailResult(r, 0, 0, ri.equalLocation, ri.fullInputLength, lastChunk)
	if !IsPartial(lastChunk) && r.Error == Success && ri.equalSigns > 0 {
		outlen := r.OutputCount
		if (outlen%3 == 0) || ((outlen%3)+1+ri.equalSigns != 4) {
			r.Error = InvalidBase64Character
		}
	}
	return Result{Error: r.Error, Count: r.InputCount}, r.OutputCount
}
