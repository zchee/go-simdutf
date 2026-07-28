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
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

// Test vectors translated and adapted from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// tests/validate_utf8_basic_tests.cpp:7-130,
// tests/validate_utf8_with_errors_tests.cpp:7-205, and
// tests/validate_utf8_puzzler_tests.cpp:5-39. Named map cases preserve the
// upstream edge-case categories; Go-only boundary and immutability checks are
// identified separately.

func TestValidateUTF8WithErrors(t *testing.T) {
	cases := map[string]struct {
		input []byte
		want  Result
	}{
		"nil":                             {input: nil, want: Result{Error: Success, Count: 0}},
		"empty":                           {input: []byte{}, want: Result{Error: Success, Count: 0}},
		"ASCII":                           {input: []byte("plain ASCII"), want: Result{Error: Success, Count: 11}},
		"copyright":                       {input: []byte{0xc2, 0xa9}, want: Result{Error: Success, Count: 2}},
		"two-byte minimum":                {input: []byte{0xc2, 0x80}, want: Result{Error: Success, Count: 2}},
		"two-byte maximum":                {input: []byte{0xdf, 0xbf}, want: Result{Error: Success, Count: 2}},
		"three-byte minimum":              {input: []byte{0xe0, 0xa0, 0x80}, want: Result{Error: Success, Count: 3}},
		"three-byte maximum":              {input: []byte{0xef, 0xbf, 0xbf}, want: Result{Error: Success, Count: 3}},
		"four-byte minimum":               {input: []byte{0xf0, 0x90, 0x80, 0x80}, want: Result{Error: Success, Count: 4}},
		"four-byte maximum":               {input: []byte{0xf4, 0x8f, 0xbf, 0xbf}, want: Result{Error: Success, Count: 4}},
		"replacement character":           {input: []byte{0xef, 0xbf, 0xbd}, want: Result{Error: Success, Count: 3}},
		"embedded NUL":                    {input: []byte{'a', 0, 'b'}, want: Result{Error: Success, Count: 3}},
		"prefixed multibyte success":      {input: []byte{'x', 0xc2, 0xa9, 0xe2, 0x82, 0xa1}, want: Result{Error: Success, Count: 6}},
		"header bits f8":                  {input: []byte{0xf8, 0x90, 0x80, 0x80, 0x80}, want: Result{Error: HeaderBits, Count: 0}},
		"prefixed header bits f8":         {input: []byte{'x', 0xf8}, want: Result{Error: HeaderBits, Count: 1}},
		"header bits ff":                  {input: []byte{0xff}, want: Result{Error: HeaderBits, Count: 0}},
		"too short two-byte end":          {input: []byte{0xc2}, want: Result{Error: TooShort, Count: 0}},
		"too short two-byte body":         {input: []byte{0xc3, 0x28}, want: Result{Error: TooShort, Count: 0}},
		"too short three-byte end":        {input: []byte{0xe1, 0x80}, want: Result{Error: TooShort, Count: 0}},
		"too short three-byte body":       {input: []byte{0xe2, 0x82, 0x28}, want: Result{Error: TooShort, Count: 0}},
		"too short four-byte end":         {input: []byte{0xf2, 0x80, 0x80}, want: Result{Error: TooShort, Count: 0}},
		"too long continuation":           {input: []byte{0x80}, want: Result{Error: TooLong, Count: 0}},
		"too long after ASCII":            {input: []byte{'a', 0x80}, want: Result{Error: TooLong, Count: 1}},
		"overlong two-byte":               {input: []byte{0xc0, 0x80}, want: Result{Error: Overlong, Count: 0}},
		"C0 missing continuation":         {input: []byte{0xc0}, want: Result{Error: TooShort, Count: 0}},
		"C0 bad continuation":             {input: []byte{0xc0, 0x7f}, want: Result{Error: TooShort, Count: 0}},
		"overlong three-byte":             {input: []byte{0xe0, 0x80, 0x80}, want: Result{Error: Overlong, Count: 0}},
		"overlong four-byte":              {input: []byte{0xf0, 0x80, 0x80, 0x80}, want: Result{Error: Overlong, Count: 0}},
		"too large":                       {input: []byte{0xf4, 0x90, 0x80, 0x80}, want: Result{Error: TooLarge, Count: 0}},
		"F4 90 incomplete":                {input: []byte{0xf4, 0x90, 0x80}, want: Result{Error: TooShort, Count: 0}},
		"F4 90 bad continuation":          {input: []byte{0xf4, 0x90, 0x80, 0x7f}, want: Result{Error: TooShort, Count: 0}},
		"F5 complete too large":           {input: []byte{0xf5, 0x80, 0x80, 0x80}, want: Result{Error: TooLarge, Count: 0}},
		"surrogate low boundary":          {input: []byte{0xed, 0xa0, 0x80}, want: Result{Error: Surrogate, Count: 0}},
		"surrogate high boundary":         {input: []byte{0xed, 0xbf, 0xbf}, want: Result{Error: Surrogate, Count: 0}},
		"ED A0 incomplete":                {input: []byte{0xed, 0xa0}, want: Result{Error: TooShort, Count: 0}},
		"ED A0 bad continuation":          {input: []byte{0xed, 0xa0, 0x7f}, want: Result{Error: TooShort, Count: 0}},
		"prefixed C0 complete":            {input: []byte{'x', 0xc0, 0x80}, want: Result{Error: Overlong, Count: 1}},
		"prefixed ED A0 complete":         {input: []byte{'x', 0xed, 0xa0, 0x80}, want: Result{Error: Surrogate, Count: 1}},
		"prefixed F4 90 complete":         {input: []byte{'x', 0xf4, 0x90, 0x80, 0x80}, want: Result{Error: TooLarge, Count: 1}},
		"first error 80 before FF":        {input: []byte{0x80, 0xff}, want: Result{Error: TooLong, Count: 0}},
		"first error FF before 80":        {input: []byte{0xff, 0x80}, want: Result{Error: HeaderBits, Count: 0}},
		"first error truncated before FF": {input: []byte{0xe1, 0x80, 0x41, 0xff}, want: Result{Error: TooShort, Count: 0}},
		"F0 incomplete":                   {input: []byte{0xf0, 0x90, 0x80}, want: Result{Error: TooShort, Count: 0}},
		"F0 bad continuation":             {input: []byte{0xf0, 0x90, 0x80, 0x41}, want: Result{Error: TooShort, Count: 0}},
		"F0 complete":                     {input: []byte{0xf0, 0x90, 0x80, 0x80}, want: Result{Error: Success, Count: 4}},
		"15-byte ASCII boundary":          {input: bytes.Repeat([]byte{'a'}, 15), want: Result{Error: Success, Count: 15}},
		"16-byte ASCII boundary":          {input: bytes.Repeat([]byte{'a'}, 16), want: Result{Error: Success, Count: 16}},
		"17-byte ASCII boundary":          {input: bytes.Repeat([]byte{'a'}, 17), want: Result{Error: Success, Count: 17}},
		"invalid at byte 15":              {input: append(bytes.Repeat([]byte{'a'}, 15), 0xff), want: Result{Error: HeaderBits, Count: 15}},
		"invalid at byte 16":              {input: append(bytes.Repeat([]byte{'a'}, 16), 0xff), want: Result{Error: HeaderBits, Count: 16}},
		"invalid at byte 31":              {input: append(bytes.Repeat([]byte{'a'}, 31), 0x80), want: Result{Error: TooLong, Count: 31}},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := ValidateUTF8WithErrors(test.input); got != test.want {
				t.Errorf("ValidateUTF8WithErrors() = %+v, want %+v", got, test.want)
			}
			if got := ValidateUTF8(test.input); got != test.want.IsOK() {
				t.Errorf("ValidateUTF8() = %t, want %t", got, test.want.IsOK())
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("validation modified input")
			}
		})
	}
}

func TestValidateUTF8VectorBoundaryRelocation(t *testing.T) {
	lengths := []int{31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129}
	for _, length := range lengths {
		t.Run(fmt.Sprintf("length-%03d", length), func(t *testing.T) {
			valid := bytes.Repeat([]byte{'a'}, length)
			if got, want := ValidateUTF8WithErrors(valid), (Result{Error: Success, Count: length}); got != want {
				t.Errorf("valid result = %+v, want %+v", got, want)
			}

			invalid := bytes.Clone(valid)
			invalid[length-1] = 0xff
			if got, want := ValidateUTF8WithErrors(invalid), (Result{Error: HeaderBits, Count: length - 1}); got != want {
				t.Errorf("relocated error result = %+v, want %+v", got, want)
			}
			if ValidateUTF8(invalid) {
				t.Error("ValidateUTF8(invalid) = true")
			}
		})
	}
}

// TestValidateUTF8WithErrorsDeterministicMutations ports the mutation domains
// from the pinned upstream validate_utf8_with_errors tests. Go's deterministic
// PRNG selects semantic mutation sites; it does not claim byte-identical output
// to the upstream C++ mt19937 fixtures. Every mutation is restored before the
// next category is exercised.
func TestValidateUTF8WithErrorsDeterministicMutations(t *testing.T) {
	const trials = 1000
	base := append(bytes.Repeat([]byte{'a', 0xc2, 0xa9, 0xe2, 0x82, 0xa1, 0xf0, 0x90, 0x8c, 0xbc}, 51), 0xc2, 0xa9)
	pristine := bytes.Clone(base)
	two, three, four, leads := utf8MutationSites(base)
	multibyte := append(append(append([]int(nil), two...), three...), four...)

	for trial := 0; trial < trials; trial++ {
		seed := int64(1234 + trial)
		rng := rand.New(rand.NewSource(seed))
		mutate := func(name string, start, width int, want Result, apply func([]byte)) {
			t.Helper()
			original := bytes.Clone(base[start : start+width])
			apply(base[start : start+width])
			if got := ValidateUTF8WithErrors(base); got != want {
				t.Fatalf("seed %d %s: result = %+v, want %+v", seed, name, got, want)
			}
			copy(base[start:start+width], original)
			if !bytes.Equal(base, pristine) {
				t.Fatalf("seed %d %s: mutation was not restored", seed, name)
			}
			if got := ValidateUTF8WithErrors(base); got != (Result{Error: Success, Count: len(base)}) {
				t.Fatalf("seed %d %s: restored input result = %+v", seed, name, got)
			}
		}

		header := leads[rng.Intn(len(leads))]
		mutate("HeaderBits", header, 1, Result{Error: HeaderBits, Count: header}, func(site []byte) { site[0] = 0xf8 })

		short := multibyte[rng.Intn(len(multibyte))]
		mutate("TooShort", short, 2, Result{Error: TooShort, Count: short}, func(site []byte) { site[1] = 0x41 })

		long := leads[1+rng.Intn(len(leads)-1)]
		mutate("TooLong", long, 1, Result{Error: TooLong, Count: long}, func(site []byte) { site[0] = 0x80 })

		overlongKinds := [][]int{two, three, four}
		kind := rng.Intn(len(overlongKinds))
		overlong := overlongKinds[kind][rng.Intn(len(overlongKinds[kind]))]
		width := kind + 2
		mutate("Overlong", overlong, width, Result{Error: Overlong, Count: overlong}, func(site []byte) {
			switch len(site) {
			case 2:
				site[0] = 0xc0
			case 3:
				site[0], site[1] = 0xe0, site[1]&0x9f
			case 4:
				site[0], site[1] = 0xf0, site[1]&0x8f
			}
		})

		large := four[rng.Intn(len(four))]
		mutate("TooLarge", large, 4, Result{Error: TooLarge, Count: large}, func(site []byte) { site[0] = 0xf5 })

		surrogate := three[rng.Intn(len(three))]
		mutate("Surrogate", surrogate, 3, Result{Error: Surrogate, Count: surrogate}, func(site []byte) {
			site[0], site[1] = 0xed, 0xa0|byte(rng.Intn(0x20))
		})
	}
}

func utf8MutationSites(input []byte) (two, three, four, leads []int) {
	for pos := 0; pos < len(input); {
		leads = append(leads, pos)
		switch {
		case input[pos] < 0x80:
			pos++
		case input[pos]&0xe0 == 0xc0:
			two = append(two, pos)
			pos += 2
		case input[pos]&0xf0 == 0xe0:
			three = append(three, pos)
			pos += 3
		default:
			four = append(four, pos)
			pos += 4
		}
	}
	return two, three, four, leads
}

func TestValidateUTF8UpstreamGoodAndBadSequences(t *testing.T) {
	good := map[string][]byte{
		"ASCII":             {'a'},
		"two-byte":          {0xc3, 0xb1},
		"three-byte":        {0xe2, 0x82, 0xa1},
		"four-byte":         {0xf0, 0x90, 0x8c, 0xbc},
		"two-byte minimum":  {0xc2, 0x80},
		"four-byte minimum": {0xf0, 0x90, 0x80, 0x80},
		"private use":       {0xee, 0x80, 0x80},
		"BOM":               {0xef, 0xbb, 0xbf},
	}
	for name, input := range good {
		t.Run("good/"+name, func(t *testing.T) {
			if got := ValidateUTF8WithErrors(input); got != (Result{Error: Success, Count: len(input)}) {
				t.Errorf("ValidateUTF8WithErrors() = %+v, want success at %d", got, len(input))
			}
			if !ValidateUTF8(input) {
				t.Error("ValidateUTF8() = false, want true")
			}
		})
	}

	// These are the compact badsequences entries at pinned upstream
	// tests/validate_utf8_basic_tests.cpp:32-58. Expected Result values also
	// lock the scalar validate_with_errors check order: structural continuation
	// errors precede range errors when a sequence is malformed in both ways.
	bad := map[string]struct {
		input []byte
		want  Result
	}{
		"bad second byte of two":         {[]byte{0xc3, 0x28}, Result{Error: TooShort, Count: 0}},
		"continuations only":             {[]byte{0xa0, 0xa1}, Result{Error: TooLong, Count: 0}},
		"bad second byte of three":       {[]byte{0xe2, 0x28, 0xa1}, Result{Error: TooShort, Count: 0}},
		"bad third byte of three":        {[]byte{0xe2, 0x82, 0x28}, Result{Error: TooShort, Count: 0}},
		"bad second byte of four":        {[]byte{0xf0, 0x28, 0x8c, 0xbc}, Result{Error: TooShort, Count: 0}},
		"bad third byte of four":         {[]byte{0xf0, 0x90, 0x28, 0xbc}, Result{Error: TooShort, Count: 0}},
		"two bad bytes of four":          {[]byte{0xf0, 0x28, 0x8c, 0x28}, Result{Error: TooShort, Count: 0}},
		"overlong two-byte":              {[]byte{0xc0, 0x9f}, Result{Error: Overlong, Count: 0}},
		"too-large with bad body":        {[]byte{0xf5, 0xff, 0xff, 0xff}, Result{Error: TooShort, Count: 0}},
		"surrogate":                      {[]byte{0xed, 0xa0, 0x81}, Result{Error: Surrogate, Count: 0}},
		"five-byte header":               {[]byte{0xf8, 0x90, 0x80, 0x80, 0x80}, Result{Error: HeaderBits, Count: 0}},
		"three-byte truncated at 15":     {append(bytes.Repeat([]byte{'1'}, 15), 0xed), Result{Error: TooShort, Count: 15}},
		"four-byte truncated at 15":      {append(bytes.Repeat([]byte{'1'}, 15), 0xf1), Result{Error: TooShort, Count: 15}},
		"two-byte truncated at 15":       {append(bytes.Repeat([]byte{'1'}, 15), 0xc2), Result{Error: TooShort, Count: 15}},
		"two-byte bad continuation":      {[]byte{0xc2, 0x7f}, Result{Error: TooShort, Count: 0}},
		"two-byte truncated":             {[]byte{0xce}, Result{Error: TooShort, Count: 0}},
		"three-byte truncated after two": {[]byte{0xce, 0xba, 0xe1}, Result{Error: TooShort, Count: 2}},
		"three-byte truncated after one": {[]byte{0xce, 0xba, 0xe1, 0xbd}, Result{Error: TooShort, Count: 2}},
		"continuation":                   {[]byte{0x80}, Result{Error: TooLong, Count: 0}},
	}
	for name, test := range bad {
		t.Run("bad/"+name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := ValidateUTF8WithErrors(test.input); got != test.want {
				t.Errorf("ValidateUTF8WithErrors() = %+v, want %+v", got, test.want)
			}
			if ValidateUTF8(test.input) {
				t.Error("ValidateUTF8() = true, want false")
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("validation modified input")
			}
		})
	}
}

func TestValidateUTF8GuardedInputUnchanged(t *testing.T) {
	for name, payload := range map[string][]byte{
		"valid":   []byte("a\xc2\xa9\xef\xbf\xbd"),
		"invalid": []byte("a\xed\xa0\x80"),
	} {
		t.Run(name, func(t *testing.T) {
			backing := append([]byte{0xa5}, payload...)
			backing = append(backing, 0x5a)
			before := bytes.Clone(backing)
			input := backing[1 : len(backing)-1]
			result := ValidateUTF8WithErrors(input)
			if got := ValidateUTF8(input); got != result.IsOK() {
				t.Errorf("ValidateUTF8() = %t, result.IsOK() = %t", got, result.IsOK())
			}
			if !bytes.Equal(backing, before) {
				t.Fatal("validation modified payload or canaries")
			}
		})
	}
}

func TestValidateUTF8UpstreamRewindRegressions(t *testing.T) {
	tooShort := append(bytes.Repeat([]byte{' '}, 64), 0xf2, 0x80, 0x80)
	for offset := 0; offset < 5; offset++ {
		want := Result{Error: TooShort, Count: 64 - offset}
		if got := ValidateUTF8WithErrors(tooShort[offset:]); got != want {
			t.Errorf("offset %d: ValidateUTF8WithErrors() = %+v, want %+v", offset, got, want)
		}
	}

	headerAt63 := append(bytes.Repeat([]byte{' '}, 63), 0xff)
	if got, want := ValidateUTF8WithErrors(headerAt63), (Result{Error: HeaderBits, Count: 63}); got != want {
		t.Errorf("64-byte header regression = %+v, want %+v", got, want)
	}

	puzzlers := map[string]struct {
		input []byte
		want  Result
	}{
		"bad64": {
			input: []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
			want:  Result{Error: TooLong, Count: 30},
		},
		"bad102": {
			input: []byte("\x0a\x04\x00\x00\xdb\xa1\xdd\xa1\xf1\xa0\xb6\x95\xe4\xb5\x89\xe7\x8f\x95\xe4\xa2\x83\xe7\x95\x89\xe7\x95\x91\xe7\x95\x89\x00\x01\x01\x1a\x20\x28\x00\x00\x60\x00\x00\x23\x00\xf1\xa0\xb6\x95\xe4\xb5\x89\xe7\x8f\x95\xe4\xa2\x83\xe7\x95\x89\xe7\x95\x91\xe7\x81\x00\x00\x01\x01\x1a\x20\x28\x00\x00\x60\x00\x00\x23\x00\x2f\x00\x00\x00\x00\x07\x04\x75\xc2\xa0\x34\x2f\x00\x00\x00\x00\x07\x04\x75\xc2\xa0\x33\x53\x2b"),
			want:  Result{Error: TooShort, Count: 62},
		},
	}
	for name, test := range puzzlers {
		t.Run(name, func(t *testing.T) {
			wantLength := map[string]int{"bad64": 64, "bad102": 102}[name]
			if len(test.input) != wantLength {
				t.Fatalf("fixture length = %d, want %d", len(test.input), wantLength)
			}
			if got := ValidateUTF8WithErrors(test.input); got != test.want {
				t.Errorf("ValidateUTF8WithErrors() = %+v, want %+v", got, test.want)
			}
			if ValidateUTF8(test.input) {
				t.Error("ValidateUTF8() = true, want false")
			}
		})
	}
}
