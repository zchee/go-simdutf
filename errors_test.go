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

import (
	"reflect"
	"testing"
)

func TestErrorCode(t *testing.T) {
	if kind := reflect.TypeOf(Success).Kind(); kind != reflect.Uint8 {
		t.Fatalf("ErrorCode underlying kind = %v, want uint8", kind)
	}

	tests := []struct {
		name string
		code ErrorCode
		want uint8
		text string
	}{
		{"success", Success, 0, "SUCCESS"},
		{"header bits", HeaderBits, 1, "HEADER_BITS"},
		{"too short", TooShort, 2, "TOO_SHORT"},
		{"too long", TooLong, 3, "TOO_LONG"},
		{"overlong", Overlong, 4, "OVERLONG"},
		{"too large", TooLarge, 5, "TOO_LARGE"},
		{"surrogate", Surrogate, 6, "SURROGATE"},
		{"invalid base64 character", InvalidBase64Character, 7, "INVALID_BASE64_CHARACTER"},
		{"base64 input remainder", Base64InputRemainder, 8, "BASE64_INPUT_REMAINDER"},
		{"base64 extra bits", Base64ExtraBits, 9, "BASE64_EXTRA_BITS"},
		{"output buffer too small", OutputBufferTooSmall, 10, "OUTPUT_BUFFER_TOO_SMALL"},
		{"other", Other, 11, "OTHER"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint8(test.code); got != test.want {
				t.Errorf("value = %d, want %d", got, test.want)
			}
			if got := ErrorToString(test.code); got != test.text {
				t.Errorf("ErrorToString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestErrorToStringUnknown(t *testing.T) {
	for _, code := range []ErrorCode{12, 255} {
		if got := ErrorToString(code); got != "OTHER" {
			t.Errorf("ErrorToString(%d) = %q, want OTHER", code, got)
		}
	}
}

func TestResultStatus(t *testing.T) {
	tests := []struct {
		name    string
		result  Result
		wantOK  bool
		wantErr bool
	}{
		{"zero value", Result{}, true, false},
		{"success", Result{Error: Success, Count: 9}, true, false},
		{"error", Result{Error: TooShort, Count: 3}, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.result.IsOK(); got != test.wantOK {
				t.Errorf("IsOK() = %v, want %v", got, test.wantOK)
			}
			if got := test.result.IsErr(); got != test.wantErr {
				t.Errorf("IsErr() = %v, want %v", got, test.wantErr)
			}
		})
	}
}

func TestFullResultResult(t *testing.T) {
	tests := []struct {
		name string
		full FullResult
		want Result
	}{
		{"zero value", FullResult{}, Result{}},
		{
			"success uses output count",
			FullResult{Error: Success, InputCount: 12, OutputCount: 7, PaddingError: true},
			Result{Error: Success, Count: 7},
		},
		{
			"error uses input count",
			FullResult{Error: InvalidBase64Character, InputCount: 5, OutputCount: 3, PaddingError: true},
			Result{Error: InvalidBase64Character, Count: 5},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.full.Result(); got != test.want {
				t.Errorf("Result() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestFullResultZeroValue(t *testing.T) {
	var result FullResult
	if result.Error != Success {
		t.Errorf("Error = %d, want Success", result.Error)
	}
	if result.InputCount != 0 {
		t.Errorf("InputCount = %d, want 0", result.InputCount)
	}
	if result.OutputCount != 0 {
		t.Errorf("OutputCount = %d, want 0", result.OutputCount)
	}
	if result.PaddingError {
		t.Error("PaddingError = true, want false")
	}
}
