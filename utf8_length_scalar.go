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

// Translated and adapted from
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// include/simdutf/scalar/utf8.h:258-325 and
// src/fallback/implementation.cpp:436-440,476-480,525-529.

func latin1LengthFromUTF8Scalar(input []byte) int {
	return countUTF8Scalar(input)
}

func utf16LengthFromUTF8Scalar(input []byte) int {
	count := 0
	for _, value := range input {
		if int8(value) > -65 {
			count++
		}
		if value >= 0xf0 {
			count++
		}
	}
	return count
}

func utf32LengthFromUTF8Scalar(input []byte) int {
	return countUTF8Scalar(input)
}

func trimPartialUTF8Scalar(input []byte) int {
	length := len(input)
	if length < 3 {
		switch length {
		case 2:
			if input[length-1] >= 0xc0 {
				return length - 1
			}
			if input[length-2] >= 0xe0 {
				return length - 2
			}
			return length
		case 1:
			if input[length-1] >= 0xc0 {
				return length - 1
			}
			return length
		case 0:
			return length
		}
	}
	if input[length-1] >= 0xc0 {
		return length - 1
	}
	if input[length-2] >= 0xe0 {
		return length - 2
	}
	if input[length-3] >= 0xf0 {
		return length - 3
	}
	return length
}
