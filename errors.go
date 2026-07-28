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

// Ported from simdutf commit c7bef0ff14a13fd6ea52e3347da2c659383392de,
// include/simdutf/error.h:7-124.

type ErrorCode uint8

const (
	Success ErrorCode = iota
	HeaderBits
	TooShort
	TooLong
	Overlong
	TooLarge
	Surrogate
	InvalidBase64Character
	Base64InputRemainder
	Base64ExtraBits
	OutputBufferTooSmall
	Other
)

func ErrorToString(code ErrorCode) string {
	switch code {
	case Success:
		return "SUCCESS"
	case HeaderBits:
		return "HEADER_BITS"
	case TooShort:
		return "TOO_SHORT"
	case TooLong:
		return "TOO_LONG"
	case Overlong:
		return "OVERLONG"
	case TooLarge:
		return "TOO_LARGE"
	case Surrogate:
		return "SURROGATE"
	case InvalidBase64Character:
		return "INVALID_BASE64_CHARACTER"
	case Base64InputRemainder:
		return "BASE64_INPUT_REMAINDER"
	case Base64ExtraBits:
		return "BASE64_EXTRA_BITS"
	case OutputBufferTooSmall:
		return "OUTPUT_BUFFER_TOO_SMALL"
	default:
		return "OTHER"
	}
}

type Result struct {
	Error ErrorCode
	Count int
}

func (result Result) IsOK() bool {
	return result.Error == Success
}

func (result Result) IsErr() bool {
	return result.Error != Success
}

type FullResult struct {
	Error        ErrorCode
	InputCount   int
	OutputCount  int
	PaddingError bool
}

func (result FullResult) Result() Result {
	if result.Error == Success {
		return Result{Error: result.Error, Count: result.OutputCount}
	}
	return Result{Error: result.Error, Count: result.InputCount}
}
