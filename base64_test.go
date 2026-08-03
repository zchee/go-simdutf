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
	"encoding/base64"
	"slices"
	"testing"
)

func TestBinaryToBase64RoundTripDefault(t *testing.T) {
	input := []byte("Hello, simdutf Base64!")
	dst := make([]byte, Base64LengthFromBinary(len(input), Base64Default))
	n := BinaryToBase64(input, dst, Base64Default)
	encoded := string(dst[:n])
	want := base64.StdEncoding.EncodeToString(input)
	if encoded != want {
		t.Fatalf("BinaryToBase64 = %q, want %q", encoded, want)
	}
	out := make([]byte, MaximalBinaryLengthFromBase64(dst[:n]))
	r := Base64ToBinary(dst[:n], out, Base64Default, Loose)
	if r.Error != Success {
		t.Fatalf("Base64ToBinary error = %v", r.Error)
	}
	if !bytes.Equal(out[:r.Count], input) {
		t.Fatalf("round-trip = %q, want %q", out[:r.Count], input)
	}
}

func TestBase64URLNoPadding(t *testing.T) {
	input := []byte{0x01, 0x02}
	dst := make([]byte, Base64LengthFromBinary(len(input), Base64URL))
	n := BinaryToBase64(input, dst, Base64URL)
	if bytes.Contains(dst[:n], []byte("=")) {
		t.Fatalf("URL encoding unexpectedly padded: %q", dst[:n])
	}
	out := make([]byte, MaximalBinaryLengthFromBase64(dst[:n]))
	r := Base64ToBinary(dst[:n], out, Base64URL, Loose)
	if r.Error != Success {
		t.Fatalf("decode error = %v", r.Error)
	}
	if !bytes.Equal(out[:r.Count], input) {
		t.Fatalf("got %v want %v", out[:r.Count], input)
	}
}

func TestBase64ValidHelpers(t *testing.T) {
	if !Base64Valid('A', Base64Default) {
		t.Fatal("A should be valid")
	}
	if Base64Valid('-', Base64Default) {
		t.Fatal("- invalid in default")
	}
	if !Base64Valid('-', Base64URL) {
		t.Fatal("- valid in url")
	}
	if !Base64ValidOrPadding('=', Base64Default) {
		t.Fatal("= should be valid or padding")
	}
	if !Base64Ignorable('\n', Base64Default) {
		t.Fatal("newline ignorable")
	}
}

func TestBase64ToBinarySafeShort(t *testing.T) {
	enc := []byte("AQID") // 0x01 0x02 0x03
	dst := make([]byte, 2)
	res, written := Base64ToBinarySafe(enc, dst, Base64Default, Loose, false)
	if res.Error != OutputBufferTooSmall {
		t.Fatalf("want OutputBufferTooSmall, got %v written=%d", res.Error, written)
	}
	if written != 0 {
		t.Fatalf("want written=0, got %d", written)
	}
}

func TestBase64UTF16RoundTrip(t *testing.T) {
	input := []byte("Go")
	enc := make([]byte, Base64LengthFromBinary(len(input), Base64Default))
	n := BinaryToBase64(input, enc, Base64Default)
	u16 := make([]uint16, n)
	for i := range n {
		u16[i] = uint16(enc[i])
	}
	out := make([]byte, MaximalBinaryLengthFromBase64UTF16(u16))
	r := Base64ToBinaryUTF16(u16, out, Base64Default, Loose)
	if r.Error != Success {
		t.Fatalf("utf16 decode error %v", r.Error)
	}
	if !bytes.Equal(out[:r.Count], input) {
		t.Fatalf("got %q want %q", out[:r.Count], input)
	}
}

// TestBase64UTF16ValidHelpers is hand-authored Go-only coverage for the
// UTF-16 classification helpers, mirroring TestBase64ValidHelpers.
func TestBase64UTF16ValidHelpers(t *testing.T) {
	tests := map[string]struct {
		value          uint16
		options        Base64Options
		valid          bool
		validOrPadding bool
		ignorable      bool
	}{
		"valid: alphabet unit":             {'A', Base64Default, true, true, false},
		"invalid: url dash in default":     {'-', Base64Default, false, false, false},
		"valid: url dash in url":           {'-', Base64URL, true, true, false},
		"padding: equals sign":             {'=', Base64Default, false, true, false},
		"ignorable: newline":               {'\n', Base64Default, false, false, true},
		"invalid: non-latin1 unit":         {0x263a, Base64Default, false, false, false},
		"ignorable: non-latin1 as garbage": {0x263a, Base64DefaultAcceptGarbage, false, false, true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := Base64ValidUTF16(test.value, test.options); got != test.valid {
				t.Fatalf("Base64ValidUTF16(%#x) = %v, want %v", test.value, got, test.valid)
			}
			if got := Base64ValidOrPaddingUTF16(test.value, test.options); got != test.validOrPadding {
				t.Fatalf("Base64ValidOrPaddingUTF16(%#x) = %v, want %v", test.value, got, test.validOrPadding)
			}
			if got := Base64IgnorableUTF16(test.value, test.options); got != test.ignorable {
				t.Fatalf("Base64IgnorableUTF16(%#x) = %v, want %v", test.value, got, test.ignorable)
			}
		})
	}
}

// TestBase64UTF16HelpersMatchByteHelpers locks the UTF-16 helpers to the
// tested byte helpers over the eight-byte range and to the ignore-garbage
// rule above it.
func TestBase64UTF16HelpersMatchByteHelpers(t *testing.T) {
	options := []Base64Options{
		Base64Default, Base64URL, Base64DefaultAcceptGarbage,
		Base64URLAcceptGarbage, Base64DefaultOrURL,
	}
	for c := range uint16(0x300) {
		for _, opts := range options {
			if c <= 0xff {
				if Base64ValidUTF16(c, opts) != Base64Valid(byte(c), opts) ||
					Base64ValidOrPaddingUTF16(c, opts) != Base64ValidOrPadding(byte(c), opts) ||
					Base64IgnorableUTF16(c, opts) != Base64Ignorable(byte(c), opts) {
					t.Fatalf("UTF-16 helper diverges from byte helper for %#x options %v", c, opts)
				}
				continue
			}
			wantIgnorable := Base64Ignorable(0xfe, opts) // any non-alphabet, non-space byte
			if Base64ValidUTF16(c, opts) || Base64ValidOrPaddingUTF16(c, opts) || Base64IgnorableUTF16(c, opts) != wantIgnorable {
				t.Fatalf("non-latin1 unit %#x misclassified under options %v", c, opts)
			}
		}
	}
}

// TestBase64ToBinarySafeUTF16 mirrors the byte-input safe-decode coverage for
// the UTF-16 entry point.
func TestBase64ToBinarySafeUTF16(t *testing.T) {
	input := []byte("Hello, simdutf Base64!")
	enc := make([]byte, Base64LengthFromBinary(len(input), Base64Default))
	n := BinaryToBase64(input, enc, Base64Default)
	u16 := make([]uint16, n)
	for i, b := range enc[:n] {
		u16[i] = uint16(b)
	}
	dst := make([]byte, MaximalBinaryLengthFromBase64UTF16(u16))
	res, written := Base64ToBinarySafeUTF16(u16, dst, Base64Default, Loose, false)
	if res.Error != Success || !bytes.Equal(dst[:written], input) {
		t.Fatalf("Base64ToBinarySafeUTF16 = %+v %q, want %q", res, dst[:written], input)
	}

	short := []uint16{'A', 'Q', 'I', 'D'} // 0x01 0x02 0x03
	res, written = Base64ToBinarySafeUTF16(short, make([]byte, 2), Base64Default, Loose, false)
	if res.Error != OutputBufferTooSmall || written != 0 {
		t.Fatalf("short dst = %+v written=%d, want OutputBufferTooSmall/0", res, written)
	}
}

// Hand-authored Go-only direct Base64 encode differential fuzz registry
// scaffolding for simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// fuzz/base64.cpp, fuzz/base64_details.cpp, fuzz/roundtrip.cpp,
// include/simdutf/scalar/base64.h, src/fallback/implementation.cpp,
// src/westmere/sse_base64.cpp, src/haswell/avx2_base64.cpp and
// src/arm64/arm_base64.cpp. It defines test metadata only and adds no product
// behavior.

type binaryToBase64FuzzVariant struct {
	name string
	variant[func([]byte, []byte, Base64Options) int]
}

var binaryToBase64FuzzVariants []binaryToBase64FuzzVariant

func registerBinaryToBase64FuzzVariant(candidate binaryToBase64FuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct BinaryToBase64 fuzz variant")
	}
	for _, registered := range binaryToBase64FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct BinaryToBase64 fuzz variant " + candidate.name)
		}
	}
	binaryToBase64FuzzVariants = append(binaryToBase64FuzzVariants, candidate)
}

func TestRegisterBinaryToBase64FuzzVariant(t *testing.T) {
	saved := binaryToBase64FuzzVariants
	defer func() { binaryToBase64FuzzVariants = saved }()
	registerBinaryToBase64FuzzVariant(binaryToBase64FuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte, []byte, Base64Options) int]{
			value:     binaryToBase64Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := binaryToBase64FuzzVariants[len(binaryToBase64FuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, binaryToBase64Scalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, binaryToBase64Scalar)
	}
}

type binaryToBase64WithLinesFuzzVariant struct {
	name string
	variant[func([]byte, []byte, int, Base64Options) int]
}

var binaryToBase64WithLinesFuzzVariants []binaryToBase64WithLinesFuzzVariant

func registerBinaryToBase64WithLinesFuzzVariant(candidate binaryToBase64WithLinesFuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct BinaryToBase64WithLines fuzz variant")
	}
	for _, registered := range binaryToBase64WithLinesFuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct BinaryToBase64WithLines fuzz variant " + candidate.name)
		}
	}
	binaryToBase64WithLinesFuzzVariants = append(binaryToBase64WithLinesFuzzVariants, candidate)
}

func TestRegisterBinaryToBase64WithLinesFuzzVariant(t *testing.T) {
	saved := binaryToBase64WithLinesFuzzVariants
	defer func() { binaryToBase64WithLinesFuzzVariants = saved }()
	registerBinaryToBase64WithLinesFuzzVariant(binaryToBase64WithLinesFuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte, []byte, int, Base64Options) int]{
			value:     binaryToBase64WithLinesScalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := binaryToBase64WithLinesFuzzVariants[len(binaryToBase64WithLinesFuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, binaryToBase64WithLinesScalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, binaryToBase64WithLinesScalar)
	}
}

type binaryLengthFromBase64FuzzVariant struct {
	name string
	variant[func([]byte) int]
}

var binaryLengthFromBase64FuzzVariants []binaryLengthFromBase64FuzzVariant

func registerBinaryLengthFromBase64FuzzVariant(candidate binaryLengthFromBase64FuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct BinaryLengthFromBase64 fuzz variant")
	}
	for _, registered := range binaryLengthFromBase64FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct BinaryLengthFromBase64 fuzz variant " + candidate.name)
		}
	}
	binaryLengthFromBase64FuzzVariants = append(binaryLengthFromBase64FuzzVariants, candidate)
}

func TestRegisterBinaryLengthFromBase64FuzzVariant(t *testing.T) {
	saved := binaryLengthFromBase64FuzzVariants
	defer func() { binaryLengthFromBase64FuzzVariants = saved }()
	registerBinaryLengthFromBase64FuzzVariant(binaryLengthFromBase64FuzzVariant{
		name: "test-scalar",
		variant: variant[func([]byte) int]{
			value:     binaryLengthFromBase64Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := binaryLengthFromBase64FuzzVariants[len(binaryLengthFromBase64FuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, binaryLengthFromBase64Scalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, binaryLengthFromBase64Scalar)
	}
}

type binaryLengthFromBase64UTF16FuzzVariant struct {
	name string
	variant[func([]uint16) int]
}

var binaryLengthFromBase64UTF16FuzzVariants []binaryLengthFromBase64UTF16FuzzVariant

func registerBinaryLengthFromBase64UTF16FuzzVariant(candidate binaryLengthFromBase64UTF16FuzzVariant) {
	if candidate.name == "" || candidate.value == nil {
		panic("simdutf: invalid direct BinaryLengthFromBase64UTF16 fuzz variant")
	}
	for _, registered := range binaryLengthFromBase64UTF16FuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct BinaryLengthFromBase64UTF16 fuzz variant " + candidate.name)
		}
	}
	binaryLengthFromBase64UTF16FuzzVariants = append(binaryLengthFromBase64UTF16FuzzVariants, candidate)
}

func TestRegisterBinaryLengthFromBase64UTF16FuzzVariant(t *testing.T) {
	saved := binaryLengthFromBase64UTF16FuzzVariants
	defer func() { binaryLengthFromBase64UTF16FuzzVariants = saved }()
	registerBinaryLengthFromBase64UTF16FuzzVariant(binaryLengthFromBase64UTF16FuzzVariant{
		name: "test-scalar",
		variant: variant[func([]uint16) int]{
			value:     binaryLengthFromBase64UTF16Scalar,
			kind:      implementationScalar,
			available: true,
		},
	})
	got := binaryLengthFromBase64UTF16FuzzVariants[len(binaryLengthFromBase64UTF16FuzzVariants)-1]
	if got.name != "test-scalar" || !sameFunction(got.value, binaryLengthFromBase64UTF16Scalar) {
		t.Fatalf("registered fuzz variant = %q %p, want test-scalar %p", got.name, got.value, binaryLengthFromBase64UTF16Scalar)
	}
}

// base64FuzzOptions enumerates every Base64Options value defined at
// options.go:27-34 so a fuzzer-supplied selector byte can index it. Reducing a
// raw Base64Options value modulo eight is forbidden: the values are
// non-contiguous ({0, 1, 2, 3, 4, 5, 8, 12}), so modular reduction over the
// value space aliases 8 to 0 and 12 to 4 and would silently never reach
// Base64DefaultOrURL or Base64DefaultOrURLAcceptGarbage. Shared by the Base64
// encode and decode differential fuzz targets.
var base64FuzzOptions = []Base64Options{
	Base64Default, Base64URL,
	Base64DefaultNoPadding, Base64URLWithPadding,
	Base64DefaultAcceptGarbage, Base64URLAcceptGarbage,
	Base64DefaultOrURL, Base64DefaultOrURLAcceptGarbage,
}

const (
	base64FuzzSourceSentinel      byte   = 0xa5
	base64FuzzDestSentinel        byte   = 0x5a
	base64FuzzUTF16SourceSentinel uint16 = 0xa55a
)

// base64FuzzOption maps a fuzzer selector byte onto the option table.
func base64FuzzOption(selector byte) Base64Options {
	return base64FuzzOptions[int(selector)%len(base64FuzzOptions)]
}

// base64FuzzBinary builds a deterministic seed payload of the requested binary
// length whose byte values cycle through the whole 0..255 range, so alphabet
// and index-packing lanes are exercised rather than one repeated byte.
func base64FuzzBinary(length int) []byte {
	out := make([]byte, length)
	for i := range out {
		out[i] = byte(i*37 + 11)
	}
	return out
}

// base64FuzzEncodeUTF16 serializes code units as the little-endian byte pairs
// the UTF-16 fuzz targets decode, matching the shared input-encoding contract.
func base64FuzzEncodeUTF16(units ...uint16) []byte {
	raw := make([]byte, len(units)*2)
	for i, unit := range units {
		raw[2*i] = byte(unit)
		raw[2*i+1] = byte(unit >> 8)
	}
	return raw
}

// base64FuzzUTF16Text encodes ASCII base64 text as UTF-16 code units.
func base64FuzzUTF16Text(text string) []byte {
	units := make([]uint16, len(text))
	for i := range len(text) {
		units[i] = uint16(text[i])
	}
	return base64FuzzEncodeUTF16(units...)
}

// checkBase64Encode runs one encoder into a freshly allocated, canary-guarded
// destination sized to exactly the scalar-required length and differences the
// returned count, the written bytes and the untouched destination tail against
// the scalar oracle.
func checkBase64Encode(t *testing.T, name string, required, wantN int, want []byte, encode func(dst []byte) int) {
	t.Helper()
	dst := newGuardedSlice(2, required, 3, base64FuzzDestSentinel)
	got := encode(dst.body)
	if got != wantN {
		t.Errorf("%s returned %d, scalar returned %d", name, got, wantN)
		return
	}
	if !bytes.Equal(dst.body[:got], want[:wantN]) {
		t.Errorf("%s wrote %q, scalar wrote %q", name, dst.body[:got], want[:wantN])
	}
	for i, value := range dst.body[got:] {
		if value != base64FuzzDestSentinel {
			t.Errorf("%s modified destination byte %d beyond the returned count", name, got+i)
			break
		}
	}
	dst.requireCanariesIntact(t)
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the
// binaryToBase64 port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/base64.cpp,
// fuzz/roundtrip.cpp, src/fallback/implementation.cpp,
// src/westmere/sse_base64.cpp, src/haswell/avx2_base64.cpp and
// src/arm64/arm_base64.cpp. The scalar function is the explicit oracle for the
// public entry point and every registered direct accelerated implementation;
// the returned count, the encoded bytes and the untouched destination tail are
// all compared, and every differenced call gets its own destination.

func FuzzBinaryToBase64(f *testing.F) {
	// Boundary binary lengths around the 48-byte (NEON), 64-byte (Westmere) and
	// 96-byte (Haswell/archsimd) encode block strides, plus their neighbours.
	lengths := []int{0, 1, 2, 3, 47, 48, 49, 63, 64, 65, 95, 96, 97, 191, 192, 193}
	for i, length := range lengths {
		f.Add(base64FuzzBinary(length), byte(i))
	}
	// Saturate every option on a short tail-only payload and on a payload with
	// complete blocks so all eight modes start covered on both encode paths.
	for selector := range len(base64FuzzOptions) {
		f.Add(base64FuzzBinary(1), byte(selector))
		f.Add(base64FuzzBinary(100), byte(selector))
	}
	f.Fuzz(func(t *testing.T, data []byte, selector byte) {
		options := base64FuzzOption(selector)
		src := newGuardedSlice(2, len(data), 3, base64FuzzSourceSentinel)
		copy(src.body, data)
		before := bytes.Clone(src.storage)

		required := base64LengthFromBinaryScalar(len(data), options)
		oracle := newGuardedSlice(2, required, 3, base64FuzzDestSentinel)
		wantN := binaryToBase64Scalar(src.body, oracle.body, options)
		if wantN < 0 || wantN > required {
			t.Fatalf("binaryToBase64Scalar returned %d for a %d byte destination", wantN, required)
		}
		oracle.requireCanariesIntact(t)

		checkBase64Encode(t, "BinaryToBase64", required, wantN, oracle.body, func(dst []byte) int {
			return BinaryToBase64(src.body, dst, options)
		})
		selection := detectSelectionInput()
		for _, candidate := range binaryToBase64FuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			checkBase64Encode(t, candidate.name+" binaryToBase64", required, wantN, oracle.body, func(dst []byte) int {
				return candidate.value(src.body, dst, options)
			})
		}

		src.requireCanariesIntact(t)
		if !bytes.Equal(src.storage, before) {
			t.Fatal("BinaryToBase64 modified input or canaries")
		}
	})
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the
// binaryToBase64WithLines port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/base64.cpp,
// src/fallback/implementation.cpp, src/westmere/sse_base64.cpp,
// src/haswell/avx2_base64.cpp and src/arm64/arm_base64.cpp. The line length is
// derived from a fuzzer byte over the clamped 4..259 range so the seeds and the
// generated corpus straddle DefaultLineLength (76, options.go:20) on both sides
// as well as the single-line and multi-line layouts.

func FuzzBinaryToBase64WithLines(f *testing.F) {
	// 54 -> 72 base64 columns, 56 and 57 -> exactly 76, 58 -> 80, and 200 -> a
	// four-line layout, so line insertion is seeded below, at and above the
	// DefaultLineLength column boundary.
	lengths := []int{0, 1, 2, 3, 47, 48, 54, 56, 57, 58, 63, 64, 65, 96, 150, 200}
	const defaultLineSelector = byte(DefaultLineLength - 4)
	for i, length := range lengths {
		f.Add(base64FuzzBinary(length), byte(i), defaultLineSelector)
	}
	for selector := range len(base64FuzzOptions) {
		f.Add(base64FuzzBinary(57), byte(selector), defaultLineSelector)
		f.Add(base64FuzzBinary(200), byte(selector), defaultLineSelector)
	}
	// Line lengths away from the default, including the sub-4 clamp boundary.
	for _, lineSelector := range []byte{0, 1, 2, 71, 73, 200, 255} {
		f.Add(base64FuzzBinary(200), byte(0), lineSelector)
	}
	f.Fuzz(func(t *testing.T, data []byte, selector, lineSelector byte) {
		options := base64FuzzOption(selector)
		lineLength := 4 + int(lineSelector)
		src := newGuardedSlice(2, len(data), 3, base64FuzzSourceSentinel)
		copy(src.body, data)
		before := bytes.Clone(src.storage)

		required := base64LengthFromBinaryWithLinesScalar(len(data), options, lineLength)
		oracle := newGuardedSlice(2, required, 3, base64FuzzDestSentinel)
		wantN := binaryToBase64WithLinesScalar(src.body, oracle.body, lineLength, options)
		if wantN < 0 || wantN > required {
			t.Fatalf("binaryToBase64WithLinesScalar returned %d for a %d byte destination", wantN, required)
		}
		oracle.requireCanariesIntact(t)

		checkBase64Encode(t, "BinaryToBase64WithLines", required, wantN, oracle.body, func(dst []byte) int {
			return BinaryToBase64WithLines(src.body, dst, lineLength, options)
		})
		selection := detectSelectionInput()
		for _, candidate := range binaryToBase64WithLinesFuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			checkBase64Encode(t, candidate.name+" binaryToBase64WithLines", required, wantN, oracle.body, func(dst []byte) int {
				return candidate.value(src.body, dst, lineLength, options)
			})
		}

		src.requireCanariesIntact(t)
		if !bytes.Equal(src.storage, before) {
			t.Fatal("BinaryToBase64WithLines modified input or canaries")
		}
	})
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the
// binaryLengthFromBase64 port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/base64.cpp,
// fuzz/base64_details.cpp, fuzz/roundtrip.cpp, include/simdutf/scalar/base64.h,
// src/haswell/avx2_base64.cpp and src/arm64/implementation.cpp. The routine
// takes no Base64Options, so the corpus carries base64 text only; the scalar
// significant-unit count and reverse padding scan are the explicit oracle for
// the public entry point and every registered direct implementation.

func FuzzBinaryLengthFromBase64(f *testing.F) {
	padded := base64.StdEncoding.EncodeToString(base64FuzzBinary(64))
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("aGVsbG8="),
		[]byte("aGVsbG8"),
		[]byte("aGVsbG8gd29ybGQ="),
		[]byte("aGVs bG8=\n\t"),
		[]byte("   \t\r\n"),
		[]byte("===="),
		[]byte("="),
		[]byte("A="),
		[]byte("AA=="),
		[]byte("AA==   "),
		[]byte("!!!!"),
		[]byte("aGV$bG8="),
		[]byte("\x00\x01\x1f\x20\x21\x7f\xff"),
		[]byte(base64.StdEncoding.EncodeToString(base64FuzzBinary(47))),
		[]byte(base64.StdEncoding.EncodeToString(base64FuzzBinary(48))),
		[]byte(base64.RawURLEncoding.EncodeToString(base64FuzzBinary(49))),
		[]byte(padded[:63]),
		[]byte(padded[:64]),
		[]byte(padded),
		[]byte(base64.StdEncoding.EncodeToString(base64FuzzBinary(200))),
		bytes.Join([][]byte{[]byte(padded[:40]), []byte(padded[40:])}, []byte("\r\n")),
		append(bytes.Repeat([]byte(" "), 70), padded...),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		guard := newGuardedSlice(2, len(data), 3, base64FuzzSourceSentinel)
		copy(guard.body, data)
		before := bytes.Clone(guard.storage)
		want := binaryLengthFromBase64Scalar(guard.body)
		if got := BinaryLengthFromBase64(guard.body); got != want {
			t.Errorf("BinaryLengthFromBase64() = %d, scalar = %d", got, want)
		}
		selection := detectSelectionInput()
		for _, candidate := range binaryLengthFromBase64FuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			if got := candidate.value(guard.body); got != want {
				t.Errorf("%s binaryLengthFromBase64() = %d, scalar = %d", candidate.name, got, want)
			}
		}
		guard.requireCanariesIntact(t)
		if !bytes.Equal(guard.storage, before) {
			t.Fatal("BinaryLengthFromBase64 modified input or canaries")
		}
	})
}

// Go-only public/direct-versus-scalar differential fuzz scaffold for the
// binaryLengthFromBase64UTF16 port pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b: fuzz/base64.cpp,
// fuzz/base64_details.cpp, fuzz/roundtrip.cpp, include/simdutf/scalar/base64.h,
// src/haswell/avx2_base64.cpp and src/arm64/implementation.cpp. Go's fuzz
// engine cannot take []uint16, so the input arrives as little-endian byte pairs
// with any odd trailing byte dropped; both bytes of each pair come from fuzzer
// input so code units strictly above 0xff stay reachable, which is the only way
// the 16-bit compare lanes are exercised beyond the ASCII range.

func FuzzBinaryLengthFromBase64UTF16(f *testing.F) {
	padded := base64.StdEncoding.EncodeToString(base64FuzzBinary(64))
	high := make([]uint16, 40)
	for i := range high {
		high[i] = uint16(0x0100 + i*997)
	}
	for _, seed := range [][]byte{
		nil,
		{},
		{0x41},
		base64FuzzUTF16Text("aGVsbG8="),
		base64FuzzUTF16Text("aGVsbG8"),
		base64FuzzUTF16Text("aGVs bG8=\n\t"),
		base64FuzzUTF16Text("===="),
		base64FuzzUTF16Text("AA=="),
		base64FuzzUTF16Text(padded[:31]),
		base64FuzzUTF16Text(padded[:32]),
		base64FuzzUTF16Text(padded[:33]),
		base64FuzzUTF16Text(padded),
		base64FuzzUTF16Text(base64.StdEncoding.EncodeToString(base64FuzzBinary(200))),
		base64FuzzEncodeUTF16(0x0100, 0x0020, 0x0021, 0x00ff, 0xd800, 0xdc00, 0xfffd),
		base64FuzzEncodeUTF16(high...),
		base64FuzzEncodeUTF16(append(high, '=', '=')...),
		append(base64FuzzUTF16Text(padded), 0x00),
		append(base64FuzzEncodeUTF16(high...), base64FuzzUTF16Text(padded)...),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		n := len(raw) / 2
		guard := newGuardedSlice(2, n, 3, base64FuzzUTF16SourceSentinel)
		for i := range n {
			guard.body[i] = uint16(raw[2*i]) | uint16(raw[2*i+1])<<8
		}
		before := slices.Clone(guard.storage)
		want := binaryLengthFromBase64UTF16Scalar(guard.body)
		if got := BinaryLengthFromBase64UTF16(guard.body); got != want {
			t.Errorf("BinaryLengthFromBase64UTF16() = %d, scalar = %d", got, want)
		}
		selection := detectSelectionInput()
		for _, candidate := range binaryLengthFromBase64UTF16FuzzVariants {
			if !candidate.supportedBy(selection) {
				continue
			}
			if got := candidate.value(guard.body); got != want {
				t.Errorf("%s binaryLengthFromBase64UTF16() = %d, scalar = %d", candidate.name, got, want)
			}
		}
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatal("BinaryLengthFromBase64UTF16 modified input or canaries")
		}
	})
}
