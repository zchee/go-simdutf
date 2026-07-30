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

// Public API adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/implementation.h:202-247. AutodetectEncoding is the
// DetectEncodings priority wrapper (UTF8, then UTF16LE, then UTF32LE, else the
// BOM-only encoding or Unspecified). Nil slices are empty input.

// AutodetectEncoding returns one guessed encoding for input.
func AutodetectEncoding(input []byte) Encoding {
	return autodetectEncodingFromDetected(DetectEncodings(input))
}

// DetectEncodings returns the Encoding bitset of encodings that accept input.
// A leading BOM short-circuits to that single encoding.
func DetectEncodings(input []byte) Encoding {
	return activeImplementation.detectEncodings(input)
}
