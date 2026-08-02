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
	"math/bits"
	"math/rand"
	"slices"
	"testing"
	"unicode/utf8"
)

// Test vectors translated and adapted from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
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

	for trial := range trials {
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
	for offset := range 5 {
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

// Hand-authored Go-only direct UTF-8 benchmark registry scaffolding for the
// port pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b.
// It defines test-only variant slots and adds no product behavior.

type utf8DirectVariant struct {
	name       string
	validate   variant[func([]byte) bool]
	withErrors variant[func([]byte) Result]
}

var utf8DirectVariants []utf8DirectVariant

func registerUTF8DirectVariant(candidate utf8DirectVariant) {
	if candidate.name == "" || candidate.validate.value == nil || candidate.withErrors.value == nil {
		panic("simdutf: invalid direct UTF-8 benchmark variant")
	}
	for _, registered := range utf8DirectVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 benchmark variant " + candidate.name)
		}
	}
	utf8DirectVariants = append(utf8DirectVariants, candidate)
}

// Hand-authored Go-only direct UTF-8 differential fuzz registry scaffolding
// for simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b. It defines
// test metadata only and adds no product behavior.

type utf8FuzzVariant struct {
	name       string
	validate   variant[func([]byte) bool]
	withErrors variant[func([]byte) Result]
}

var utf8FuzzVariants []utf8FuzzVariant

func registerUTF8FuzzVariant(candidate utf8FuzzVariant) {
	if candidate.name == "" || candidate.validate.value == nil || candidate.withErrors.value == nil {
		panic("simdutf: invalid direct UTF-8 fuzz variant")
	}
	for _, registered := range utf8FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 fuzz variant " + candidate.name)
		}
	}
	utf8FuzzVariants = append(utf8FuzzVariants, candidate)
}

// Go-only public-versus-scalar differential fuzz scaffold. The pinned upstream
// fuzz target exercises both validation entry points at
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// fuzz/conversion.cpp:68-74. Scalar functions are the explicit oracle for the
// public entry points and every registered direct accelerated implementation.

func FuzzValidateUTF8(f *testing.F) {
	seeds := [][]byte{
		nil,
		{},
		[]byte("ASCII\x00suffix"),
		{0xc2, 0x80},
		{0xdf, 0xbf},
		{0xe0, 0xa0, 0x80},
		{0xef, 0xbf, 0xbf},
		{0xf0, 0x90, 0x80, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
		{0xef, 0xbf, 0xbd},
		{0xf8},
		{0xe1, 0x80},
		{0x80},
		{0xc0, 0x80},
		{0xf4, 0x90, 0x80, 0x80},
		{0xed, 0xa0, 0x80},
		{0x80, 0xff},
		{0xff, 0x80},
		{0xe1, 0x80, 0x41, 0xff},
		{0xf0, 0x90, 0x80},
		{0xf0, 0x90, 0x80, 0x41},
		append(bytes.Repeat([]byte{' '}, 64), 0xf2, 0x80, 0x80),
		append(bytes.Repeat([]byte{' '}, 63), 0xff),
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x1c\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x80\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
		[]byte("\x0a\x04\x00\x00\xdb\xa1\xdd\xa1\xf1\xa0\xb6\x95\xe4\xb5\x89\xe7\x8f\x95\xe4\xa2\x83\xe7\x95\x89\xe7\x95\x91\xe7\x95\x89\x00\x01\x01\x1a\x20\x28\x00\x00\x60\x00\x00\x23\x00\xf1\xa0\xb6\x95\xe4\xb5\x89\xe7\x8f\x95\xe4\xa2\x83\xe7\x95\x89\xe7\x95\x91\xe7\x81\x00\x00\x01\x01\x1a\x20\x28\x00\x00\x60\x00\x00\x23\x00\x2f\x00\x00\x00\x00\x07\x04\x75\xc2\xa0\x34\x2f\x00\x00\x00\x00\x07\x04\x75\xc2\xa0\x33\x53\x2b"),
	}
	for _, length := range []int{31, 32, 33, 61, 62, 63, 64, 65, 66, 67, 68, 95, 96, 97, 127, 128, 129} {
		valid := bytes.Repeat([]byte{'a'}, length)
		seeds = append(seeds, valid)
		invalid := bytes.Clone(valid)
		invalid[length-1] = 0xff
		seeds = append(seeds, invalid)
	}
	for _, seed := range seeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		backing := make([]byte, len(input)+2)
		backing[0], backing[len(backing)-1] = 0xa5, 0x5a
		copy(backing[1:], input)
		before := bytes.Clone(backing)
		guarded := backing[1 : len(backing)-1]
		if got, want := ValidateUTF8WithErrors(guarded), validateUTF8WithErrorsScalar(guarded); got != want {
			t.Fatalf("ValidateUTF8WithErrors() = %+v, scalar = %+v", got, want)
		}
		if got, want := ValidateUTF8(guarded), validateUTF8Scalar(guarded); got != want {
			t.Fatalf("ValidateUTF8() = %t, scalar = %t", got, want)
		}
		for _, candidate := range utf8FuzzVariants {
			if !candidate.validate.supportedBy(detectSelectionInput()) || !candidate.withErrors.supportedBy(detectSelectionInput()) {
				continue
			}
			if got, want := candidate.withErrors.value(guarded), validateUTF8WithErrorsScalar(guarded); got != want {
				t.Fatalf("%s ValidateUTF8WithErrors() = %+v, scalar = %+v", candidate.name, got, want)
			}
			if got, want := candidate.validate.value(guarded), validateUTF8Scalar(guarded); got != want {
				t.Fatalf("%s ValidateUTF8() = %t, scalar = %t", candidate.name, got, want)
			}
		}
		if !bytes.Equal(backing, before) {
			t.Fatal("validation modified input or canaries")
		}
	})
}

// Test vectors adapted from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b
// convert_utf8_to_* tests. Canary and short-destination checks are Go-specific
// slice-contract coverage.

func TestUTF8ConvertTable(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		latin1 []byte
		utf16  []uint16
		utf32  []uint32
	}{
		{"nil", "", nil, nil, nil},
		{"ascii", "Hello", []byte("Hello"), []uint16{'H', 'e', 'l', 'l', 'o'}, []uint32{'H', 'e', 'l', 'l', 'o'}},
		{"latin1", "caf\u00e9", []byte{'c', 'a', 'f', 0xe9}, []uint16{'c', 'a', 'f', 0xe9}, []uint32{'c', 'a', 'f', 0xe9}},
		{"emoji", "A\U0001F600B", nil, []uint16{'A', 0xd83d, 0xde00, 'B'}, []uint32{'A', 0x1f600, 'B'}},
		{"arabic", "\u0645\u0631\u062d\u0628\u0627", nil, []uint16{0x0645, 0x0631, 0x062d, 0x0628, 0x0627}, []uint32{0x0645, 0x0631, 0x062d, 0x0628, 0x0627}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := []byte(test.input)
			if test.name == "nil" {
				input = nil
			}

			if test.latin1 != nil {
				dst := make([]byte, Latin1LengthFromUTF8(input))
				if got := ConvertUTF8ToLatin1(input, dst); got != len(test.latin1) || !slices.Equal(dst[:got], test.latin1) {
					t.Fatalf("ConvertUTF8ToLatin1() = %d/%x, want %d/%x", got, dst[:got], len(test.latin1), test.latin1)
				}
				if got := ConvertUTF8ToLatin1WithErrors(input, dst); got != (Result{Error: Success, Count: len(test.latin1)}) {
					t.Fatalf("ConvertUTF8ToLatin1WithErrors() = %#v", got)
				}
				if got := ConvertValidUTF8ToLatin1(input, dst); got != len(test.latin1) || !slices.Equal(dst[:got], test.latin1) {
					t.Fatalf("ConvertValidUTF8ToLatin1() = %d/%x", got, dst[:got])
				}
			} else if len(input) > 0 {
				dst := make([]byte, Latin1LengthFromUTF8(input))
				if got := ConvertUTF8ToLatin1(input, dst); got != 0 {
					t.Fatalf("ConvertUTF8ToLatin1() = %d, want 0", got)
				}
				if got := ConvertUTF8ToLatin1WithErrors(input, dst); got.Error != TooLarge {
					t.Fatalf("ConvertUTF8ToLatin1WithErrors() = %#v, want TooLarge", got)
				}
			}

			dst16 := make([]uint16, UTF16LengthFromUTF8(input))
			if got := ConvertUTF8ToUTF16LE(input, dst16); got != len(test.utf16) || !slices.Equal(dst16[:got], rawUTF16(test.utf16, true)) {
				t.Fatalf("ConvertUTF8ToUTF16LE() = %d/%x, want %d/%x", got, dst16[:got], len(test.utf16), rawUTF16(test.utf16, true))
			}
			if got := ConvertUTF8ToUTF16BE(input, dst16); got != len(test.utf16) || !slices.Equal(dst16[:got], rawUTF16(test.utf16, false)) {
				t.Fatalf("ConvertUTF8ToUTF16BE() = %d/%x", got, dst16[:got])
			}
			if got := ConvertValidUTF8ToUTF16LE(input, dst16); got != len(test.utf16) || !slices.Equal(dst16[:got], rawUTF16(test.utf16, true)) {
				t.Fatalf("ConvertValidUTF8ToUTF16LE() = %d/%x", got, dst16[:got])
			}
			if got := ConvertUTF8ToUTF16LEWithErrors(input, dst16); got != (Result{Error: Success, Count: len(test.utf16)}) {
				t.Fatalf("ConvertUTF8ToUTF16LEWithErrors() = %#v", got)
			}

			dst32 := make([]uint32, UTF32LengthFromUTF8(input))
			if got := ConvertUTF8ToUTF32(input, dst32); got != len(test.utf32) || !slices.Equal(dst32[:got], test.utf32) {
				t.Fatalf("ConvertUTF8ToUTF32() = %d/%x, want %d/%x", got, dst32[:got], len(test.utf32), test.utf32)
			}
			if got := ConvertValidUTF8ToUTF32(input, dst32); got != len(test.utf32) || !slices.Equal(dst32[:got], test.utf32) {
				t.Fatalf("ConvertValidUTF8ToUTF32() = %d/%x", got, dst32[:got])
			}
			if got := ConvertUTF8ToUTF32WithErrors(input, dst32); got != (Result{Error: Success, Count: len(test.utf32)}) {
				t.Fatalf("ConvertUTF8ToUTF32WithErrors() = %#v", got)
			}

			native16 := make([]uint16, len(dst16))
			if got := ConvertUTF8ToUTF16(input, native16); got != len(test.utf16) {
				t.Fatalf("ConvertUTF8ToUTF16() = %d", got)
			}
			explicit := make([]uint16, len(dst16))
			if nativeLittleEndian() {
				ConvertUTF8ToUTF16LE(input, explicit)
			} else {
				ConvertUTF8ToUTF16BE(input, explicit)
			}
			if !slices.Equal(native16, explicit) {
				t.Fatalf("native UTF-16 mismatch")
			}
		})
	}
}

func TestUTF8ConvertErrors(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		err   ErrorCode
		count int
	}{
		{"too_short", []byte{0xc2}, TooShort, 0},
		{"overlong", []byte{0xc0, 0xaf}, Overlong, 0},
		{"surrogate", []byte{0xed, 0xa0, 0x80}, Surrogate, 0},
		{"header", []byte{0xff}, HeaderBits, 0},
		{"too_long", []byte{0x80}, TooLong, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dst16 := make([]uint16, UTF16LengthFromUTF8(test.input)+8)
			got := ConvertUTF8ToUTF16LEWithErrors(test.input, dst16)
			if got.Error != test.err || got.Count != test.count {
				t.Fatalf("UTF16 errors = %#v, want {%v %d}", got, test.err, test.count)
			}
			if ConvertUTF8ToUTF16LE(test.input, dst16) != 0 {
				t.Fatal("ConvertUTF8ToUTF16LE did not return 0")
			}
			dst32 := make([]uint32, UTF32LengthFromUTF8(test.input)+8)
			got = ConvertUTF8ToUTF32WithErrors(test.input, dst32)
			if got.Error != test.err || got.Count != test.count {
				t.Fatalf("UTF32 errors = %#v, want {%v %d}", got, test.err, test.count)
			}
		})
	}
}

func TestUTF8ConvertShortDestinationPanics(t *testing.T) {
	input := []byte("café")
	short := make([]byte, 1)
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	ConvertUTF8ToLatin1(input, short)
}

func TestUTF8ConvertRoundTripSample(t *testing.T) {
	input := []byte("ASCII \u00e9 \u0645 \U0001F600")
	if !utf8.Valid(input) {
		t.Fatal("fixture must be valid UTF-8")
	}
	dst32 := make([]uint32, UTF32LengthFromUTF8(input))
	n := ConvertUTF8ToUTF32(input, dst32)
	if n != utf8.RuneCount(input) {
		t.Fatalf("UTF32 count = %d, want %d", n, utf8.RuneCount(input))
	}
	dst16 := make([]uint16, UTF16LengthFromUTF8(input))
	if ConvertUTF8ToUTF16LE(input, dst16) == 0 {
		t.Fatal("UTF16 conversion failed")
	}
}

func rawUTF16(native []uint16, little bool) []uint16 {
	out := make([]uint16, len(native))
	needNative := little == nativeLittleEndian()
	for i, value := range native {
		if needNative {
			out[i] = value
		} else {
			out[i] = bits.ReverseBytes16(value)
		}
	}
	return out
}

// TestUTF8ToUTF16HostEndianWrappers is hand-authored Go-only coverage for the
// host-endian UTF-8 to UTF-16 wrappers; the explicit-endian variants stay the
// tested oracles, matching TestUTF16HostEndianWrappers.
func TestUTF8ToUTF16HostEndianWrappers(t *testing.T) {
	if !nativeLittleEndian() {
		t.Skip("host-endian wrapper LE-path coverage requires little-endian host")
	}

	input := []byte("ASCII \u00e9 \u0645 \U0001F600")
	length := UTF16LengthFromUTF8(input)

	dst := make([]uint16, length)
	leDst := make([]uint16, length)
	if got, want := ConvertUTF8ToUTF16WithErrors(input, dst), ConvertUTF8ToUTF16LEWithErrors(input, leDst); got != want || !slices.Equal(dst, leDst) {
		t.Fatalf("ConvertUTF8ToUTF16WithErrors = %+v, want %+v (payload equal %v)", got, want, slices.Equal(dst, leDst))
	}
	if got := ConvertUTF8ToUTF16WithErrors(input, dst); got != (Result{Error: Success, Count: length}) {
		t.Fatalf("ConvertUTF8ToUTF16WithErrors = %+v, want success count %d", got, length)
	}

	validDst := make([]uint16, length)
	validLEDst := make([]uint16, length)
	if got, want := ConvertValidUTF8ToUTF16(input, validDst), ConvertValidUTF8ToUTF16LE(input, validLEDst); got != want || !slices.Equal(validDst, validLEDst) {
		t.Fatalf("ConvertValidUTF8ToUTF16 = %d, want %d (payload equal %v)", got, want, slices.Equal(validDst, validLEDst))
	}

	invalid := []byte{'a', 0xff, 'b'}
	invalidDst := make([]uint16, UTF16LengthFromUTF8(invalid))
	invalidLEDst := make([]uint16, len(invalidDst))
	got, want := ConvertUTF8ToUTF16WithErrors(invalid, invalidDst), ConvertUTF8ToUTF16LEWithErrors(invalid, invalidLEDst)
	if got != want || got.Error == Success {
		t.Fatalf("invalid input = %+v, want %+v with non-Success error", got, want)
	}
}

// Portions Copyright 2021 The simdutf Authors.

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
			for sample := range 10 {
				input := generatePinnedReferenceUTF8(rng, 1000, profile)
				if !ValidateUTF8(input) || !validateUTF8Scalar(input) || !validateUTF8PinnedReference(input) {
					t.Fatal("generated input is not valid UTF-8")
				}
				for mutation := range 1000 {
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
	for sample := range 1000 {
		input := generatePinnedReferenceUTF8(rng, rng.Intn(256), utf8WidthWeights{1, 1, 1, 1})
		if !ValidateUTF8(input) || !validateUTF8Scalar(input) || !validateUTF8PinnedReference(input) {
			t.Fatal("generated input is not valid UTF-8")
		}
		for mutation := range 1000 {
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
