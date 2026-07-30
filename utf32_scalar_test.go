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

import "testing"

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/utf32.h:8-50.

func TestValidateUTF32Scalar(t *testing.T) {
	tests := []struct {
		name  string
		input []uint32
		want  Result
	}{
		{"nil", nil, Result{Error: Success, Count: 0}},
		{"empty", []uint32{}, Result{Error: Success, Count: 0}},
		{"unicode-extrema", []uint32{0, 0x7f, 0x80, 0xffff, 0x10000, 0x10ffff}, Result{Error: Success, Count: 6}},
		{"surrogate-lower-bound", []uint32{0x61, 0xd800}, Result{Error: Surrogate, Count: 1}},
		{"surrogate-upper-bound", []uint32{0x61, 0xdfff}, Result{Error: Surrogate, Count: 1}},
		{"too-large", []uint32{0x61, 0x110000}, Result{Error: TooLarge, Count: 1}},
		{"first-error", []uint32{0x61, 0xd800, 0x110000}, Result{Error: Surrogate, Count: 1}},
		{"too-large-before-surrogate", []uint32{0x61, 0x110000, 0xd800}, Result{Error: TooLarge, Count: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateUTF32WithErrorsScalar(tt.input)
			gotBool := validateUTF32Scalar(tt.input)
			publicResult := ValidateUTF32WithErrors(tt.input)
			publicBool := ValidateUTF32(tt.input)
			if publicResult != got || publicBool != gotBool {
				t.Fatalf("public bool=%t result=%+v, scalar bool=%t result=%+v", publicBool, publicResult, gotBool, got)
			}
			if got != tt.want || gotBool != (tt.want.Error == Success) {
				t.Fatalf("bool=%t result=%+v, want valid=%t result=%+v", gotBool, got, tt.want.Error == Success, tt.want)
			}
		})
	}
}
