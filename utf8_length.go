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

// Public API adapted from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// include/simdutf/implementation.h:1673-1778,3954-3983. Go slices replace C++
// pointer/length pairs.

// Latin1LengthFromUTF8 returns the number of bytes needed to represent input
// as Latin-1. It does not validate input; for arbitrary bytes it follows the
// pinned scalar byte-counting formula. It is not BOM-aware.
func Latin1LengthFromUTF8(input []byte) int {
	return activeImplementation.latin1LengthFromUTF8(input)
}

// UTF16LengthFromUTF8 returns the number of UTF-16 code units needed to
// represent input. It does not validate input; for arbitrary bytes it follows
// the pinned scalar byte-counting formula. It is not BOM-aware.
func UTF16LengthFromUTF8(input []byte) int {
	if len(input) < utf16LengthFromUTF8DispatchCutoff {
		return utf16LengthFromUTF8Scalar(input)
	}
	return activeImplementation.utf16LengthFromUTF8(input)
}

// UTF32LengthFromUTF8 returns the number of UTF-32 code units needed to
// represent input. It is equivalent to CountUTF8 and does not validate input.
// It is not BOM-aware.
func UTF32LengthFromUTF8(input []byte) int {
	if len(input) < utf32LengthFromUTF8DispatchCutoff {
		return utf32LengthFromUTF8Scalar(input)
	}
	return activeImplementation.utf32LengthFromUTF8(input)
}

// TrimPartialUTF8 returns the length of the longest prefix that excludes a
// final partial code point. Input must be valid UTF-8 except that its last code
// point may be truncated.
func TrimPartialUTF8(input []byte) int {
	return trimPartialUTF8Scalar(input)
}
