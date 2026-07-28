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
	"bytes"
	"encoding/hex"
	"math/bits"
	"math/rand"
	"slices"
	"testing"
	"unicode/utf16"
)

// Upstream-derived vectors and semantics come from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// tests/validate_ascii_basic_tests.cpp:8-125,
// tests/validate_ascii_with_errors_tests.cpp:7-38,
// tests/validate_utf16be_basic_tests.cpp:12-20,158-174, and
// tests/validate_utf16le_basic_tests.cpp:31-41,257-265. The randomized tests
// reproduce the pinned seed domain and mutation loops with Go's deterministic
// generator; they do not claim byte-identical output to C++ mt19937 fixtures.
// Boundary, nil/empty, raw-storage endian, and canary cases are narrow Go-only
// slice checks, not additional upstream vectors. Canaries prove read-only input
// preservation; they do not prove absence of overreads.

func TestValidateASCIIBasicVectors(t *testing.T) {
	good := map[string]string{
		"good-00": "a",
		"good-01": "abcde12345",
		"good-02": "q",
		"good-03": "uL",
		"good-04": "\x7fL#<:o]D\x13p",
	}
	bad := map[string]string{
		"bad-00": "\xc3(",
		"bad-01": "\xa0\xa1",
		"bad-02": "\xe2(\xa1",
		"bad-03": "\xe2\x82(",
		"bad-04": "\xf0(\x8c\xbc",
		"bad-05": "\xf0\x90(\xbc",
		"bad-06": "\xf0(\x8c(",
		"bad-07": "\xc0\x9f",
		"bad-08": "\xf5\xff\xff\xff",
		"bad-09": "\xed\xa0\x81",
		"bad-10": "\xf8\x90\x80\x80\x80",
		"bad-11": "123456789012345\xed",
		"bad-12": "123456789012345\xf1",
		"bad-13": "123456789012345\xc2",
		"bad-14": "\xc2\x7f",
		"bad-15": "\xce",
		"bad-16": "\xce\xba\xe1",
		"bad-17": "\xce\xba\xe1\xbd",
		"bad-18": "\xce\xba\xe1\xbd\xb9\xcf",
		"bad-19": "\xce\xba\xe1\xbd\xb9\xcf\x83\xce",
		"bad-20": "\xce\xba\xe1\xbd\xb9\xcf\x83\xce\xbc\xce",
		"bad-21": "\xdf",
		"bad-22": "\xef\xbf",
		"bad-23": "\x80",
		"bad-24": "\x91\x85\x95\x9e",
		"bad-25": "l\x02\x8e\x18",
		"bad-26": mustDecodeASCIIHex(
			"255b6e2c322c5b5b332c342c05292c330101010101010101010101010101010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101010101010101015b5b5b5b5b5b5b5b5b5b5b5b5b5b" +
				"5b5b5b5d2c352e332c392e332c372e332c392e342c372e332c392e332c372e332c392e345d5d5d5d5d5d5d5d5d5d5d5d" +
				"5d5d5d5d0101010101010101010101010101010101010101010101010101010101200101010101020101010101010101" +
				"01010101010101010101010101010101010101010101010101230a010101010101010101010101010101010101010101" +
				"01017e7e0a0a010101010101010101010101010101010101010101015b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5d2c37" +
				"2e332c392e332c372e332c392e342c372e332c392e332c372e332c392e345d5d5d5d5d5d5d5d5d5d5d5d5d5d5d010180" +
				"010101790101010101010101010101010101010101010101010101010101010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101"),
		"bad-27": mustDecodeASCIIHex(
			"5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b800101010101010101010101010101010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101" +
				"01010101010101010101010101010101010101100101010101010101010101"),
		"bad-28": mustDecodeASCIIHex(
			"200b010101643a643a643a5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b5b" +
				"5b5b5b5b3001010101010101010101010101010101010101010101010101010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101018001010101010101010101010101010101010101" +
				"010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101010101" +
				"0101010101010101010101010101010101"),
		"bad-29": "\x80", "bad-30": "\x90", "bad-31": "\xa1", "bad-32": "\xb2",
		"bad-33": "\xc3", "bad-34": "\xd4", "bad-35": "\xe5", "bad-36": "\xf6",
		"bad-37": "\xc3\xb1", "bad-38": "\xe2\x82\xa1", "bad-39": "\xf0\x90\x8c\xbc",
		"bad-40": "\xc2\x80", "bad-41": "\xf0\x90\x80\x80", "bad-42": "\xee\x80\x80",
		"bad-43": "\xef\xbb\xbf",
	}

	validators := map[string]func([]byte) bool{
		"public": ValidateASCII,
		"scalar": validateASCIIScalar,
	}
	for validatorName, validator := range validators {
		for name, input := range good {
			t.Run(validatorName+"/"+name, func(t *testing.T) {
				if !validator([]byte(input)) {
					t.Error("valid upstream ASCII vector was rejected")
				}
			})
		}
		for name, input := range bad {
			t.Run(validatorName+"/"+name, func(t *testing.T) {
				if validator([]byte(input)) {
					t.Error("invalid upstream ASCII vector was accepted")
				}
			})
		}
	}
}

func TestValidateASCIIWithErrorsUpstreamSeedDomain(t *testing.T) {
	validators := map[string]func([]byte) Result{
		"public": ValidateASCIIWithErrors,
		"scalar": validateASCIIWithErrorsScalar,
	}
	for seed := int64(1234); seed <= 2233; seed++ {
		generator := rand.New(rand.NewSource(seed))
		input := make([]byte, 513)
		for i := range input[:512] {
			input[i] = byte(generator.Intn(127) + 1)
		}
		for name, validator := range validators {
			if got, want := validator(input), (Result{Error: Success, Count: 513}); got != want {
				t.Fatalf("seed %d %s valid result = %+v, want %+v", seed, name, got, want)
			}
			for i := range input {
				input[i] |= 0x80
				if got, want := validator(input), (Result{Error: TooLarge, Count: i}); got != want {
					t.Fatalf("seed %d %s invalid index %d result = %+v, want %+v", seed, name, i, got, want)
				}
				input[i] &^= 0x80
			}
		}
	}
}

func TestValidateUTF16AsASCIIUpstreamSeedDomain(t *testing.T) {
	type validator struct {
		public func([]uint16) bool
		scalar func([]uint16) bool
	}
	validators := map[string]validator{
		"little-endian": {ValidateUTF16LEAsASCII, validateUTF16LEAsASCIIScalar},
		"big-endian":    {ValidateUTF16BEAsASCII, validateUTF16BEAsASCIIScalar},
	}
	for seed := int64(1234); seed <= 2233; seed++ {
		generator := rand.New(rand.NewSource(seed))
		semantic := make([]uint16, 512)
		for i := range semantic {
			semantic[i] = uint16(generator.Intn(128))
		}
		for name, funcs := range validators {
			little := name == "little-endian"
			input := encodeUTF16Raw(semantic, little)
			if !funcs.public(input) || !funcs.scalar(input) {
				t.Fatalf("seed %d %s valid input was rejected", seed, name)
			}
			input[256] = 0xc0c0
			if funcs.public(input) || funcs.scalar(input) {
				t.Fatalf("seed %d %s invalid input was accepted", seed, name)
			}
		}
	}
}

func TestValidateUTF16AsASCIIFixedUpstreamVectors(t *testing.T) {
	positive := utf16.Encode([]rune("I am ascii, I promise!"))
	negative := utf16.Encode([]rune("But this isn't: köttbulle"))
	for _, test := range []struct {
		name   string
		little bool
		public func([]uint16) bool
		scalar func([]uint16) bool
	}{
		{name: "little-endian", little: true, public: ValidateUTF16LEAsASCII, scalar: validateUTF16LEAsASCIIScalar},
		{name: "big-endian", little: false, public: ValidateUTF16BEAsASCII, scalar: validateUTF16BEAsASCIIScalar},
	} {
		t.Run(test.name, func(t *testing.T) {
			if input := encodeUTF16Raw(positive, test.little); !test.public(input) || !test.scalar(input) {
				t.Error("positive upstream UTF-16 vector was rejected")
			}
			if input := encodeUTF16Raw(negative, test.little); test.public(input) || test.scalar(input) {
				t.Error("negative upstream UTF-16 vector was accepted")
			}
		})
	}
	if nativeLittleEndian() {
		if ValidateUTF16AsASCII(positive) != ValidateUTF16LEAsASCII(positive) || ValidateUTF16AsASCII(negative) != ValidateUTF16LEAsASCII(negative) {
			t.Error("native wrapper differs from explicit little-endian wrapper")
		}
	} else if ValidateUTF16AsASCII(positive) != ValidateUTF16BEAsASCII(positive) || ValidateUTF16AsASCII(negative) != ValidateUTF16BEAsASCII(negative) {
		t.Error("native wrapper differs from explicit big-endian wrapper")
	}
}

func TestASCIIAPIsNilAndEmpty(t *testing.T) {
	for name, input := range map[string][]byte{"nil": nil, "empty": {}} {
		t.Run("bytes/"+name, func(t *testing.T) {
			if !ValidateASCII(input) || !validateASCIIScalar(input) {
				t.Error("empty byte input was rejected")
			}
			want := Result{Error: Success, Count: 0}
			if got := ValidateASCIIWithErrors(input); got != want {
				t.Errorf("public result = %+v, want %+v", got, want)
			}
			if got := validateASCIIWithErrorsScalar(input); got != want {
				t.Errorf("scalar result = %+v, want %+v", got, want)
			}
		})
	}
	for name, input := range map[string][]uint16{"nil": nil, "empty": {}} {
		t.Run("utf16/"+name, func(t *testing.T) {
			for validatorName, validator := range map[string]func([]uint16) bool{
				"native":    ValidateUTF16AsASCII,
				"le":        ValidateUTF16LEAsASCII,
				"be":        ValidateUTF16BEAsASCII,
				"scalar-le": validateUTF16LEAsASCIIScalar,
				"scalar-be": validateUTF16BEAsASCIIScalar,
			} {
				if !validator(input) {
					t.Errorf("%s rejected empty UTF-16 input", validatorName)
				}
			}
		})
	}
}

func TestValidateASCIIBoundariesAndFirstError(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129}
	for _, length := range lengths {
		valid := bytes.Repeat([]byte{0x7f}, length)
		if !ValidateASCII(valid) || !validateASCIIScalar(valid) {
			t.Fatalf("length %d valid input was rejected", length)
		}
		if got, want := ValidateASCIIWithErrors(valid), (Result{Error: Success, Count: length}); got != want {
			t.Fatalf("length %d valid result = %+v, want %+v", length, got, want)
		}
		if length == 0 {
			continue
		}
		for positionName, position := range map[string]int{"first": 0, "middle": length / 2, "last": length - 1} {
			input := slices.Clone(valid)
			input[position] = 0x80
			if length > 1 {
				input[length-1] = 0xff
			}
			if ValidateASCII(input) || validateASCIIScalar(input) {
				t.Fatalf("length %d %s invalid input was accepted", length, positionName)
			}
			want := Result{Error: TooLarge, Count: position}
			if got := ValidateASCIIWithErrors(input); got != want {
				t.Fatalf("length %d %s public result = %+v, want %+v", length, positionName, got, want)
			}
			if got := validateASCIIWithErrorsScalar(input); got != want {
				t.Fatalf("length %d %s scalar result = %+v, want %+v", length, positionName, got, want)
			}
		}
	}
}

func TestValidateUTF16AsASCIIBoundaries(t *testing.T) {
	lengths := []int{0, 1, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65}
	for _, length := range lengths {
		semantic := make([]uint16, length)
		for i := range semantic {
			semantic[i] = 0x7f
		}
		for _, test := range []struct {
			name   string
			little bool
			public func([]uint16) bool
			scalar func([]uint16) bool
		}{
			{name: "little-endian", little: true, public: ValidateUTF16LEAsASCII, scalar: validateUTF16LEAsASCIIScalar},
			{name: "big-endian", little: false, public: ValidateUTF16BEAsASCII, scalar: validateUTF16BEAsASCIIScalar},
		} {
			valid := encodeUTF16Raw(semantic, test.little)
			if !test.public(valid) || !test.scalar(valid) {
				t.Fatalf("length %d %s valid input was rejected", length, test.name)
			}
			if length == 0 {
				continue
			}
			for positionName, position := range map[string]int{"first": 0, "middle": length / 2, "last": length - 1} {
				invalidSemantic := slices.Clone(semantic)
				invalidSemantic[position] = 0x80
				input := encodeUTF16Raw(invalidSemantic, test.little)
				if test.public(input) || test.scalar(input) {
					t.Fatalf("length %d %s %s invalid input was accepted", length, test.name, positionName)
				}
			}
		}
	}
}

func TestASCIIValidatorsPreserveGuardedInputs(t *testing.T) {
	byteGuard := newGuardedSlice(3, 33, 5, byte(0xa5))
	for i := range byteGuard.body {
		byteGuard.body[i] = 0x7f
	}
	byteBefore := slices.Clone(byteGuard.storage)
	ValidateASCII(byteGuard.body)
	ValidateASCIIWithErrors(byteGuard.body)
	validateASCIIScalar(byteGuard.body)
	validateASCIIWithErrorsScalar(byteGuard.body)
	byteGuard.requireCanariesIntact(t)
	if !bytes.Equal(byteGuard.storage, byteBefore) {
		t.Error("byte validators modified guarded input")
	}

	semantic := make([]uint16, 17)
	for i := range semantic {
		semantic[i] = 0x7f
	}
	for _, test := range []struct {
		name       string
		input      []uint16
		validators map[string]func([]uint16) bool
	}{
		{
			name:  "native",
			input: slices.Clone(semantic),
			validators: map[string]func([]uint16) bool{
				"public": ValidateUTF16AsASCII,
			},
		},
		{
			name:  "little-endian",
			input: encodeUTF16Raw(semantic, true),
			validators: map[string]func([]uint16) bool{
				"public": ValidateUTF16LEAsASCII,
				"scalar": validateUTF16LEAsASCIIScalar,
			},
		},
		{
			name:  "big-endian",
			input: encodeUTF16Raw(semantic, false),
			validators: map[string]func([]uint16) bool{
				"public": ValidateUTF16BEAsASCII,
				"scalar": validateUTF16BEAsASCIIScalar,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			wordGuard := newGuardedSlice(3, len(test.input), 5, uint16(0xa55a))
			copy(wordGuard.body, test.input)
			wordBefore := slices.Clone(wordGuard.storage)
			for name, validator := range test.validators {
				if !validator(wordGuard.body) {
					t.Errorf("%s validator rejected valid input", name)
				}
			}
			wordGuard.requireCanariesIntact(t)
			if !slices.Equal(wordGuard.storage, wordBefore) {
				t.Error("UTF-16 validators modified guarded input")
			}
		})
	}
}

func encodeUTF16Raw(semantic []uint16, little bool) []uint16 {
	encoded := slices.Clone(semantic)
	if little != nativeLittleEndian() {
		for i := range encoded {
			encoded[i] = bits.ReverseBytes16(encoded[i])
		}
	}
	return encoded
}

func mustDecodeASCIIHex(encoded string) string {
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		panic(err)
	}
	return string(decoded)
}
