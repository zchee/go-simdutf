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

package simdutf

// Ported from simdutf commit dec3aad192f47081110d9c766d4917bad243906f,
// include/simdutf/encoding_types.h:15-24 and src/encoding_types.cpp:3-64.

type Encoding uint8

const (
	Unspecified Encoding = 0
	UTF8        Encoding = 1
	UTF16LE     Encoding = 2
	UTF16BE     Encoding = 4
	UTF32LE     Encoding = 8
	UTF32BE     Encoding = 16
	Latin1      Encoding = 32
)

func EncodingString(encoding Encoding) string {
	switch encoding {
	case UTF16LE:
		return "UTF16 little-endian"
	case UTF16BE:
		return "UTF16 big-endian"
	case UTF32LE:
		return "UTF32 little-endian"
	case UTF32BE:
		return "UTF32 big-endian"
	case UTF8:
		return "UTF8"
	case Unspecified:
		return "unknown"
	default:
		return "error"
	}
}

func CheckBOM(input []byte) Encoding {
	if len(input) >= 2 && input[0] == 0xff && input[1] == 0xfe {
		if len(input) >= 4 && input[2] == 0x00 && input[3] == 0x00 {
			return UTF32LE
		}
		return UTF16LE
	}
	if len(input) >= 2 && input[0] == 0xfe && input[1] == 0xff {
		return UTF16BE
	}
	if len(input) >= 4 && input[0] == 0x00 && input[1] == 0x00 && input[2] == 0xfe && input[3] == 0xff {
		return UTF32BE
	}
	if len(input) >= 3 && input[0] == 0xef && input[1] == 0xbb && input[2] == 0xbf {
		return UTF8
	}
	return Unspecified
}

func BOMByteSize(encoding Encoding) int {
	switch encoding {
	case UTF16LE, UTF16BE:
		return 2
	case UTF32LE, UTF32BE:
		return 4
	case UTF8:
		return 3
	default:
		return 0
	}
}
