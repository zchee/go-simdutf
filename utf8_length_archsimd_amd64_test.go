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

//go:build amd64 && goexperiment.simd

package simdutf

import (
	"bytes"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestUTF8LengthArchsimdAllByteValues(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	all := make([]byte, 256)
	for value := range all {
		all[value] = byte(value)
		checkUTF8LengthArchsimd(t, bytes.Repeat([]byte{byte(value)}, 129))
	}
	checkUTF8LengthArchsimd(t, all)
	checkUTF8LengthArchsimd(t, append(slices.Clone(all), all...))
	checkUTF8LengthArchsimd(t, bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xff}, 1050))
}

func TestUTF8LengthArchsimdAlignmentsAndBoundaries(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	lengths := []int{
		0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129,
		4063, 4064, 4065, 8063, 8064, 8065, 8127, 8128, 8129,
	}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				storage := make([]byte, alignment+length+32)
				input := storage[alignment : alignment+length]
				for i := range input {
					input[i] = byte(i*131 + length*17 + alignment)
				}
				checkUTF8LengthArchsimd(t, input)
			})
		}
	}
}

func TestUTF16LengthFromUTF8ArchsimdAccumulatorFlushes(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	for _, value := range []byte{0x00, 0x80, 0xf0, 0xff} {
		for _, length := range []int{4063, 4064, 4065, 8127, 8128, 8129, 3*4064 + 17, 1 << 20} {
			t.Run("byte="+strconv.Itoa(int(value))+"/length="+strconv.Itoa(length), func(t *testing.T) {
				input := bytes.Repeat([]byte{value}, length)
				if got, want := utf16LengthFromUTF8Archsimd(input), utf16LengthFromUTF8Scalar(input); got != want {
					t.Errorf("utf16 archsimd = %d, scalar = %d", got, want)
				}
			})
		}
	}
}

func TestUTF32LengthFromUTF8ArchsimdBlocksAndTails(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	for _, length := range []int{0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129, 191, 192, 193, 4095, 4096, 4097} {
		input := make([]byte, length)
		for i := range input {
			input[i] = []byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xff}[i&7]
		}
		if got, want := utf32LengthFromUTF8Archsimd(input), utf32LengthFromUTF8Scalar(input); got != want {
			t.Errorf("length %d: utf32 archsimd = %d, scalar = %d", length, got, want)
		}
	}
}

func TestUTF8LengthArchsimdCanariesAndImmutability(t *testing.T) {
	requireUTF8LengthArchsimdFeatures(t)
	for _, length := range []int{0, 1, 31, 32, 33, 63, 64, 65, 127, 128, 129, 4063, 4064, 4065, 8127, 8128, 8129} {
		guard := newGuardedSlice(37, length, 41, byte(0xa5))
		for i := range guard.body {
			guard.body[i] = byte(i*73 + length)
		}
		before := slices.Clone(guard.storage)
		checkUTF8LengthArchsimd(t, guard.body)
		guard.requireCanariesIntact(t)
		if !slices.Equal(guard.storage, before) {
			t.Fatalf("length %d input or canary modified", length)
		}
	}
}

func TestUTF8LengthArchsimdSourceContracts(t *testing.T) {
	source, err := os.ReadFile("utf8_length_archsimd_amd64.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	contracts := map[string]int{
		"return countUTF8Archsimd(input)":                       1,
		"chunk.Min(fourByteThreshold).Equal(fourByteThreshold)": 1,
		"if iterations == 127":                                  1,
		"local.SumAbsDiff(zero)":                                2,
		"mask := uint64(mask0) | uint64(mask1)<<32":             1,
		"bits.OnesCount64(mask)":                                1,
	}
	for contract, want := range contracts {
		if got := strings.Count(text, contract); got != want {
			t.Errorf("source contract %q occurs %d times, want %d", contract, got, want)
		}
	}
	if strings.Contains(text, ".GreaterEqual(") {
		t.Error("generic unsigned GreaterEqual must not implement the four-byte test")
	}
	if strings.Contains(text, "countUTF8Archsimd(input[offset:])") {
		t.Error("UTF-32 archsimd must not reuse CountUTF8 for its SIMD blocks")
	}
}

func checkUTF8LengthArchsimd(t *testing.T, input []byte) {
	t.Helper()
	checks := []struct {
		name string
		got  int
		want int
	}{
		{name: "latin1", got: latin1LengthFromUTF8Archsimd(input), want: latin1LengthFromUTF8Scalar(input)},
		{name: "utf16", got: utf16LengthFromUTF8Archsimd(input), want: utf16LengthFromUTF8Scalar(input)},
		{name: "utf32", got: utf32LengthFromUTF8Archsimd(input), want: utf32LengthFromUTF8Scalar(input)},
	}
	for _, check := range checks {
		if check.got != check.want {
			t.Errorf("%s archsimd = %d, scalar = %d for %d bytes", check.name, check.got, check.want, len(input))
		}
	}
}

func requireUTF8LengthArchsimdFeatures(t *testing.T) {
	t.Helper()
	selection := detectSelectionInput()
	if selection.features&(cpuAVX2|cpuPOPCNT) != cpuAVX2|cpuPOPCNT || !selection.archsimdAVX2 {
		t.Skip("archsimd UTF-8 length tests require repository AVX2/POPCNT and archsimd AVX2 gates")
	}
}
