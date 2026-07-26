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
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// include/simdutf/implementation.h:1673-1778,3954-3983. Go slices replace C++
// pointer/length pairs.

// Latin1LengthFromUTF8 returns the number of bytes needed to represent input
// as Latin-1. It does not validate input; for arbitrary bytes it follows the
// pinned scalar byte-counting formula.
func Latin1LengthFromUTF8(input []byte) int {
	return activeImplementation.latin1LengthFromUTF8(input)
}

// UTF16LengthFromUTF8 returns the number of UTF-16 code units needed to
// represent input. It does not validate input; for arbitrary bytes it follows
// the pinned scalar byte-counting formula.
func UTF16LengthFromUTF8(input []byte) int {
	return activeImplementation.utf16LengthFromUTF8(input)
}

// UTF32LengthFromUTF8 returns the number of UTF-32 code units needed to
// represent input. It is equivalent to CountUTF8 and does not validate input.
func UTF32LengthFromUTF8(input []byte) int {
	return activeImplementation.utf32LengthFromUTF8(input)
}

// TrimPartialUTF8 returns the length of the longest prefix that excludes a
// final partial code point. Input must be valid UTF-8 except that its last code
// point may be truncated.
func TrimPartialUTF8(input []byte) int {
	return trimPartialUTF8Scalar(input)
}
