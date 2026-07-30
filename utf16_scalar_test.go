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
	"math/bits"
	"slices"
	"testing"
)

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// include/simdutf/scalar/utf16.h:20-76,183-213.

func TestValidateUTF16Scalar(t *testing.T) {
	tests := []struct {
		name  string
		words []uint16
		want  Result
	}{
		{"nil", nil, Result{Error: Success, Count: 0}},
		{"empty", []uint16{}, Result{Error: Success, Count: 0}},
		{"bmp-extrema", []uint16{0, 0xd7ff, 0xe000, 0xffff}, Result{Error: Success, Count: 4}},
		{"pair-extrema", []uint16{0xd800, 0xdc00, 0xdbff, 0xdfff}, Result{Error: Success, Count: 4}},
		{"stray-low", []uint16{0x61, 0xdc00}, Result{Error: Surrogate, Count: 1}},
		{"terminal-high", []uint16{0x61, 0xdbff}, Result{Error: Surrogate, Count: 1}},
		{"high-non-low", []uint16{0xd800, 0x61}, Result{Error: Surrogate, Count: 0}},
		{"consecutive-highs", []uint16{0xd800, 0xdbff, 0xdc00}, Result{Error: Surrogate, Count: 0}},
		{"mixed-first-error", []uint16{0x61, 0xd800, 0xdc00, 0x62, 0xdc00, 0xd800}, Result{Error: Surrogate, Count: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, little := range []bool{true, false} {
				input := rawUTF16Scalar(tt.words, little)
				var gotBool bool
				var got Result
				if little {
					gotBool = validateUTF16LEScalar(input)
					got = validateUTF16LEWithErrorsScalar(input)
				} else {
					gotBool = validateUTF16BEScalar(input)
					got = validateUTF16BEWithErrorsScalar(input)
				}
				var publicBool bool
				var publicResult Result
				if little {
					publicBool = ValidateUTF16LE(input)
					publicResult = ValidateUTF16LEWithErrors(input)
				} else {
					publicBool = ValidateUTF16BE(input)
					publicResult = ValidateUTF16BEWithErrors(input)
				}
				if publicBool != gotBool || publicResult != got {
					t.Fatalf("little=%t: public bool=%t result=%+v, scalar bool=%t result=%+v", little, publicBool, publicResult, gotBool, got)
				}
				if got != tt.want || gotBool != (tt.want.Error == Success) {
					t.Fatalf("little=%t: bool=%t result=%+v, want valid=%t result=%+v", little, gotBool, got, tt.want.Error == Success, tt.want)
				}
			}
		})
	}
}

func TestToWellFormedUTF16Scalar(t *testing.T) {
	semantic := []uint16{0x61, 0xdc00, 0xd800, 0xdc00, 0xdbff, 0x62, 0xdfff, 0x63, 0xd800}
	wantSemantic := []uint16{0x61, 0xfffd, 0xd800, 0xdc00, 0xfffd, 0x62, 0xfffd, 0x63, 0xfffd}
	for _, little := range []bool{true, false} {
		input := rawUTF16Scalar(semantic, little)
		want := rawUTF16Scalar(wantSemantic, little)
		canary := uint16(0xa55a)
		dst := make([]uint16, len(input)+2)
		dst[0], dst[len(dst)-1] = canary, canary
		if little {
			toWellFormedUTF16LEScalar(input, dst[1:len(dst)-1])
		} else {
			toWellFormedUTF16BEScalar(input, dst[1:len(dst)-1])
		}
		publicDst := make([]uint16, len(input))
		if little {
			ToWellFormedUTF16LE(input, publicDst)
		} else {
			ToWellFormedUTF16BE(input, publicDst)
		}
		if !slices.Equal(publicDst, want) {
			t.Fatalf("little=%t: public output=%x want=%x", little, publicDst, want)
		}
		if dst[0] != canary || dst[len(dst)-1] != canary {
			t.Fatalf("little=%t: wrote outside destination: %x", little, dst)
		}
		if !slices.Equal(dst[1:len(dst)-1], want) {
			t.Fatalf("little=%t: output=%x want=%x", little, dst[1:len(dst)-1], want)
		}

		inPlace := append([]uint16(nil), input...)
		if little {
			toWellFormedUTF16LEScalar(inPlace, inPlace)
		} else {
			toWellFormedUTF16BEScalar(inPlace, inPlace)
		}
		if !slices.Equal(inPlace, want) {
			t.Fatalf("little=%t: in-place output=%x want=%x", little, inPlace, want)
		}
	}
}

func TestToWellFormedUTF16ScalarShortDestinationPanicsBeforeWrite(t *testing.T) {
	for _, little := range []bool{true, false} {
		input := rawUTF16Scalar([]uint16{0xd800, 0x61}, little)
		dst := []uint16{0xa55a}
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			if little {
				toWellFormedUTF16LEScalar(input, dst)
			} else {
				toWellFormedUTF16BEScalar(input, dst)
			}
		}()
		if dst[0] != 0xa55a {
			t.Fatalf("little=%t: short destination was modified: %x", little, dst)
		}
	}
}

func TestUTF16NativeWrappers(t *testing.T) {
	input := []uint16{0x61, 0xd800, 0xdc00}
	if !ValidateUTF16(input) || ValidateUTF16WithErrors(input) != (Result{Error: Success, Count: len(input)}) {
		t.Fatal("native validation rejected well-formed input")
	}

	input = []uint16{0x61, 0xd800}
	dst := make([]uint16, len(input))
	ToWellFormedUTF16(input, dst)
	if !slices.Equal(dst, []uint16{0x61, 0xfffd}) {
		t.Fatalf("native repair=%x", dst)
	}
}

func rawUTF16Scalar(semantic []uint16, little bool) []uint16 {
	raw := append([]uint16(nil), semantic...)
	if little != nativeLittleEndian() {
		for i := range raw {
			raw[i] = bits.ReverseBytes16(raw[i])
		}
	}
	return raw
}
