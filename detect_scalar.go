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

import "encoding/binary"

// Translated and adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/fallback/implementation.cpp:8-32 and the base
// implementation::autodetect_encoding in src/implementation.cpp:73-105.
// Byte buffers are reinterpreted as native UTF-16/UTF-32 code units to match
// the pinned C++ reinterpret_cast before validate_utf16le / validate_utf32.

func detectEncodingsScalar(input []byte) Encoding {
	if bom := CheckBOM(input); bom != Unspecified {
		return bom
	}
	var out Encoding
	if validateUTF8Scalar(input) {
		out |= UTF8
	}
	if len(input)%2 == 0 {
		if validateUTF16LEScalar(nativeUTF16UnitsFromBytes(input)) {
			out |= UTF16LE
		}
	}
	if len(input)%4 == 0 {
		if validateUTF32Scalar(nativeUTF32UnitsFromBytes(input)) {
			out |= UTF32LE
		}
	}
	return out
}

func autodetectEncodingFromDetected(encodings Encoding) Encoding {
	if encodings&UTF8 != 0 {
		return UTF8
	}
	if encodings&UTF16LE != 0 {
		return UTF16LE
	}
	if encodings&UTF32LE != 0 {
		return UTF32LE
	}
	return encodings
}

func nativeUTF16UnitsFromBytes(input []byte) []uint16 {
	n := len(input) / 2
	if n == 0 {
		return nil
	}
	out := make([]uint16, n)
	for i := range n {
		out[i] = binary.NativeEndian.Uint16(input[i*2:])
	}
	return out
}

func nativeUTF32UnitsFromBytes(input []byte) []uint32 {
	n := len(input) / 4
	if n == 0 {
		return nil
	}
	out := make([]uint32, n)
	for i := range n {
		out[i] = binary.NativeEndian.Uint32(input[i*4:])
	}
	return out
}
