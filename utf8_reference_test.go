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

import (
	"fmt"
	"math/rand"
	"testing"
)

// Translated from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// tests/reference/validate_utf8.cpp:7-78 and validate_utf8.h:3-8.
// credit: based on code from Google Fuchsia (Apache Licensed)
func validateUTF8PinnedReference(input []byte) bool {
	for pos := 0; pos < len(input); {
		b := input[pos]
		switch {
		case b < 0x80:
			pos++
		case b&0xe0 == 0xc0:
			next := pos + 2
			if next > len(input) || input[pos+1]&0xc0 != 0x80 {
				return false
			}
			cp := uint32(b&0x1f)<<6 | uint32(input[pos+1]&0x3f)
			if cp < 0x80 || cp > 0x7ff {
				return false
			}
			pos = next
		case b&0xf0 == 0xe0:
			next := pos + 3
			if next > len(input) || input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 {
				return false
			}
			cp := uint32(b&0x0f)<<12 | uint32(input[pos+1]&0x3f)<<6 | uint32(input[pos+2]&0x3f)
			if cp < 0x800 || cp > 0xffff || cp > 0xd7ff && cp < 0xe000 {
				return false
			}
			pos = next
		case b&0xf8 == 0xf0:
			next := pos + 4
			if next > len(input) || input[pos+1]&0xc0 != 0x80 || input[pos+2]&0xc0 != 0x80 || input[pos+3]&0xc0 != 0x80 {
				return false
			}
			cp := uint32(b&7)<<18 | uint32(input[pos+1]&0x3f)<<12 | uint32(input[pos+2]&0x3f)<<6 | uint32(input[pos+3]&0x3f)
			if cp <= 0xffff || cp > 0x10ffff {
				return false
			}
			pos = next
		default:
			return false
		}
	}
	return true
}

// additional tests are from autobahn websocket testsuite
// https://github.com/crossbario/autobahn-testsuite/tree/master/autobahntestsuite/autobahntestsuite/case
// Exact byte fixtures from pinned tests/validate_utf8_basic_tests.cpp:24-108.
// Each C++ adjacent string-literal expression is represented by one Go string.
var pinnedUTF8GoodSequences = [][]byte{
	[]byte("\x61"),
	[]byte("\xc3\xb1"),
	[]byte("\xe2\x82\xa1"),
	[]byte("\xf0\x90\x8c\xbc"),
	[]byte("\xc2\x80"),
	[]byte("\xf0\x90\x80\x80"),
	[]byte("\xee\x80\x80"),
	[]byte("\xef\xbb\xbf"),
}

var pinnedUTF8BadSequences = [][]byte{
	[]byte("\xc3\x28"),
	[]byte("\xa0\xa1"),
	[]byte("\xe2\x28\xa1"),
	[]byte("\xe2\x82\x28"),
	[]byte("\xf0\x28\x8c\xbc"),
	[]byte("\xf0\x90\x28\xbc"),
	[]byte("\xf0\x28\x8c\x28"),
	[]byte("\xc0\x9f"),
	[]byte("\xf5\xff\xff\xff"),
	[]byte("\xed\xa0\x81"),
	[]byte("\xf8\x90\x80\x80\x80"),
	[]byte("\x31\x32\x33\x34\x35\x36\x37\x38\x39\x30\x31\x32\x33\x34\x35\xed"),
	[]byte("\x31\x32\x33\x34\x35\x36\x37\x38\x39\x30\x31\x32\x33\x34\x35\xf1"),
	[]byte("\x31\x32\x33\x34\x35\x36\x37\x38\x39\x30\x31\x32\x33\x34\x35\xc2"),
	[]byte("\xc2\x7f"),
	[]byte("\xce"),
	[]byte("\xce\xba\xe1"),
	[]byte("\xce\xba\xe1\xbd"),
	[]byte("\xce\xba\xe1\xbd\xb9\xcf"),
	[]byte("\xce\xba\xe1\xbd\xb9\xcf\x83\xce"),
	[]byte("\xce\xba\xe1\xbd\xb9\xcf\x83\xce\xbc\xce"),
	[]byte("\xdf"),
	[]byte("\xef\xbf"),
	[]byte("\x80"),
	[]byte("\x91\x85\x95\x9e"),
	[]byte("\x6c\x02\x8e\x18"),
	[]byte("\x25\x5b\x6e\x2c\x32\x2c\x5b\x5b\x33\x2c\x34\x2c\x05\x29\x2c\x33\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b" +
		"\x5b\x5b\x5b\x5d\x2c\x35\x2e\x33\x2c\x39\x2e\x33\x2c\x37\x2e\x33\x2c\x39\x2e\x34\x2c\x37\x2e\x33\x2c\x39\x2e\x33\x2c\x37\x2e\x33" +
		"\x2c\x39\x2e\x34\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x20\x01\x01\x01\x01\x01\x02\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x23\x0a\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x7e\x7e\x0a\x0a\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5d\x2c\x37" +
		"\x2e\x33\x2c\x39\x2e\x33\x2c\x37\x2e\x33\x2c\x39\x2e\x34\x2c\x37\x2e\x33\x2c\x39\x2e\x33\x2c\x37\x2e\x33\x2c\x39\x2e\x34\x5d\x5d" +
		"\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x5d\x01\x01\x80\x01\x01\x01\x79\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01"),
	[]byte("\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x80\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x10\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01"),
	[]byte("\x20\x0b\x01\x01\x01\x64\x3a\x64\x3a\x64\x3a\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b" +
		"\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x5b\x30\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x80\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01\x01" +
		"\x01"),
}

func TestValidateUTF8PinnedBasicSequences(t *testing.T) {
	for i, input := range pinnedUTF8GoodSequences {
		t.Run(fmt.Sprintf("good-%02d", i), func(t *testing.T) {
			if !validateUTF8PinnedReference(input) {
				t.Fatal("pinned reference rejected good sequence")
			}
			if !ValidateUTF8(input) {
				t.Fatal("ValidateUTF8 rejected good sequence")
			}
			if !validateUTF8Scalar(input) {
				t.Fatal("validateUTF8Scalar rejected good sequence")
			}
		})
	}
	for i, input := range pinnedUTF8BadSequences {
		t.Run(fmt.Sprintf("bad-%02d", i), func(t *testing.T) {
			if validateUTF8PinnedReference(input) {
				t.Fatal("pinned reference accepted bad sequence")
			}
			if ValidateUTF8(input) {
				t.Fatal("ValidateUTF8 accepted bad sequence")
			}
			if validateUTF8Scalar(input) {
				t.Fatal("validateUTF8Scalar accepted bad sequence")
			}
		})
	}
}

type utf8WidthWeights [4]int

func generatePinnedReferenceUTF8(rng *rand.Rand, outputBytes int, weights utf8WidthWeights) []byte {
	total := weights[0] + weights[1] + weights[2] + weights[3]
	output := make([]byte, 0, outputBytes+4)
	for len(output) < outputBytes {
		draw := rng.Intn(total)
		width := 1
		for draw >= weights[width-1] {
			draw -= weights[width-1]
			width++
		}
		switch width {
		case 1:
			output = append(output, byte(1+rng.Intn(0x7f)))
		case 2:
			cp := uint32(0x80 + rng.Intn(0x800-0x80))
			output = append(output, 0xc0|byte(cp>>6), 0x80|byte(cp&0x3f))
		case 3:
			cp := uint32(0x800 + rng.Intn(0x10000-0x800-0x800))
			if cp >= 0xd800 {
				cp += 0x800
			}
			output = append(output, 0xe0|byte(cp>>12), 0x80|byte(cp>>6&0x3f), 0x80|byte(cp&0x3f))
		case 4:
			cp := uint32(0x10000 + rng.Intn(0x110000-0x10000))
			output = append(output, 0xf0|byte(cp>>18), 0x80|byte(cp>>12&0x3f), 0x80|byte(cp>>6&0x3f), 0x80|byte(cp&0x3f))
		}
	}
	return append(output, 0) // Match pinned random_utf8's scalar-code EOS.
}

// This ports the behavior and iteration counts of pinned
// tests/validate_utf8_brute_force_tests.cpp:7-86. Go uses a deterministic Go
// PRNG rather than claiming byte identity with C++ mt19937/rand fixtures.
func TestValidateUTF8PinnedReferenceCorruption(t *testing.T) {
	profiles := []utf8WidthWeights{{1, 0, 0, 0}, {0, 1, 0, 0}, {1, 1, 0, 0}, {0, 0, 1, 0}, {0, 1, 1, 0}, {1, 0, 1, 0}, {1, 1, 1, 0}}
	for profileIndex, profile := range profiles {
		t.Run(fmt.Sprintf("profile-%d", profileIndex), func(t *testing.T) {
			rng := rand.New(rand.NewSource(1234))
			for sample := 0; sample < 10; sample++ {
				input := generatePinnedReferenceUTF8(rng, 1000, profile)
				if !ValidateUTF8(input) || !validateUTF8Scalar(input) || !validateUTF8PinnedReference(input) {
					t.Fatal("generated input is not valid UTF-8")
				}
				for mutation := 0; mutation < 1000; mutation++ {
					index := rng.Intn(len(input))
					original := input[index]
					input[index] = byte(rng.Uint32())
					want := validateUTF8PinnedReference(input)
					if got := ValidateUTF8(input); got != want {
						t.Fatalf("sample %d mutation %d public = %t, reference = %t", sample, mutation, got, want)
					}
					if got := validateUTF8Scalar(input); got != want {
						t.Fatalf("sample %d mutation %d scalar = %t, reference = %t", sample, mutation, got, want)
					}
					input[index] = original
				}
			}
		})
	}
}

func TestValidateUTF8PinnedReferenceBruteForce(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	for sample := 0; sample < 1000; sample++ {
		input := generatePinnedReferenceUTF8(rng, rng.Intn(256), utf8WidthWeights{1, 1, 1, 1})
		if !ValidateUTF8(input) || !validateUTF8Scalar(input) || !validateUTF8PinnedReference(input) {
			t.Fatal("generated input is not valid UTF-8")
		}
		for mutation := 0; mutation < 1000; mutation++ {
			input[rng.Intn(len(input))] = byte(1 << rng.Intn(8))
			want := validateUTF8PinnedReference(input)
			if got := ValidateUTF8(input); got != want {
				t.Fatalf("sample %d mutation %d public = %t, reference = %t", sample, mutation, got, want)
			}
			if got := validateUTF8Scalar(input); got != want {
				t.Fatalf("sample %d mutation %d scalar = %t, reference = %t", sample, mutation, got, want)
			}
		}
	}
}
