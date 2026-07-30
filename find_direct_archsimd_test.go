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
	"fmt"
	"testing"
)

func requireFindArchsimdAVX2(t *testing.T) {
	t.Helper()
	candidate := variant[func()]{
		value:     func() {},
		kind:      implementationArchsimd,
		required:  cpuAVX2,
		available: true,
	}
	if !candidate.supportedBy(detectSelectionInput()) {
		t.Skip("direct archsimd AVX2 implementation is unsupported")
	}
}

func TestDirectArchsimdFindAgainstScalar(t *testing.T) {
	requireFindArchsimdAVX2(t)

	byteCases := []struct {
		name  string
		input []byte
		value byte
	}{
		{name: "nil", value: 'a'},
		{name: "empty", input: []byte{}, value: 'a'},
		{name: "hit-first", input: []byte("abc"), value: 'a'},
		{name: "hit-middle", input: []byte("bac"), value: 'a'},
		{name: "hit-last", input: []byte("bca"), value: 'a'},
		{name: "miss", input: []byte("bcd"), value: 'a'},
		{name: "nul-hit", input: []byte{'A', 'B', 0, 'C'}, value: 0},
		{name: "short", input: []byte{7}, value: 7},
	}
	for _, length := range [...]int{31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257} {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i%251 + 1)
		}
		byteCases = append(byteCases,
			struct {
				name  string
				input []byte
				value byte
			}{name: fmt.Sprintf("miss-length-%d", length), input: input, value: 0},
			struct {
				name  string
				input []byte
				value byte
			}{name: fmt.Sprintf("hit-end-length-%d", length), input: append(append([]byte{}, input[:length-1]...), 0), value: 0},
			struct {
				name  string
				input []byte
				value byte
			}{name: fmt.Sprintf("hit-mid-length-%d", length), input: func() []byte {
				out := append([]byte{}, input...)
				out[length/2] = 0
				return out
			}(), value: 0},
		)
	}
	for _, tc := range byteCases {
		t.Run("byte/"+tc.name, func(t *testing.T) {
			got := findArchsimd(tc.input, tc.value)
			want := findScalar(tc.input, tc.value)
			if got != want {
				t.Fatalf("findArchsimd = %d, scalar = %d", got, want)
			}
		})
	}

	utf16Cases := []struct {
		name  string
		input []uint16
		value uint16
	}{
		{name: "nil", value: 'a'},
		{name: "empty", input: []uint16{}, value: 'a'},
		{name: "hit-first", input: []uint16{'a', 'b', 'c'}, value: 'a'},
		{name: "hit-middle", input: []uint16{'b', 'a', 'c'}, value: 'a'},
		{name: "hit-last", input: []uint16{'b', 'c', 'a'}, value: 'a'},
		{name: "miss", input: []uint16{'b', 'c', 'd'}, value: 'a'},
		{name: "nul-hit", input: []uint16{'A', 0, 'C'}, value: 0},
		{name: "high-unit", input: []uint16{0x20, 0xd800, 0x20}, value: 0xd800},
		{name: "short", input: []uint16{7}, value: 7},
	}
	for _, length := range [...]int{15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129} {
		input := make([]uint16, length)
		for i := range input {
			input[i] = uint16(i%1009 + 1)
		}
		utf16Cases = append(utf16Cases,
			struct {
				name  string
				input []uint16
				value uint16
			}{name: fmt.Sprintf("miss-length-%d", length), input: input, value: 0},
			struct {
				name  string
				input []uint16
				value uint16
			}{name: fmt.Sprintf("hit-end-length-%d", length), input: append(append([]uint16{}, input[:length-1]...), 0), value: 0},
			struct {
				name  string
				input []uint16
				value uint16
			}{name: fmt.Sprintf("hit-mid-length-%d", length), input: func() []uint16 {
				out := append([]uint16{}, input...)
				out[length/2] = 0
				return out
			}(), value: 0},
		)
	}
	for _, tc := range utf16Cases {
		t.Run("utf16/"+tc.name, func(t *testing.T) {
			got := findUTF16Archsimd(tc.input, tc.value)
			want := findUTF16Scalar(tc.input, tc.value)
			if got != want {
				t.Fatalf("findUTF16Archsimd = %d, scalar = %d", got, want)
			}
		})
	}
}

func TestMakeImplementationAMD64FindArchsimdForceable(t *testing.T) {
	requireFindArchsimdAVX2(t)

	// Scalar-first coverage lives in dispatch_amd64_test.go; this checks force selection.
	t.Setenv("SIMDUTF_FORCE_PROVIDER", "archsimd")
	forced := makeImplementation(selectionInput{features: cpuAVX2, archsimdAVX2: true})
	if !sameFunction(forced.find, findArchsimd) {
		t.Fatalf("forced find selected %p, want archsimd %p", forced.find, findArchsimd)
	}
	if !sameFunction(forced.findUTF16, findUTF16Archsimd) {
		t.Fatalf("forced findUTF16 selected %p, want archsimd %p", forced.findUTF16, findUTF16Archsimd)
	}
}

func FuzzFindArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]byte{
		nil,
		{},
		{0},
		{1, 2, 3},
		make([]byte, 64),
		make([]byte, 65),
		make([]byte, 128),
	} {
		f.Add(input, byte(0))
		f.Add(input, byte(1))
	}
	f.Fuzz(func(t *testing.T, input []byte, value byte) {
		requireFindArchsimdAVX2(t)
		if got, want := findArchsimd(input, value), findScalar(input, value); got != want {
			t.Fatalf("findArchsimd = %d, scalar = %d (len=%d value=%d)", got, want, len(input), value)
		}
	})
}

func FuzzFindUTF16ArchsimdAgainstScalar(f *testing.F) {
	for _, input := range [][]uint16{
		nil,
		{},
		{0},
		{1, 2, 3},
		make([]uint16, 32),
		make([]uint16, 33),
		make([]uint16, 64),
	} {
		// Seed via bytes: encode little-endian pairs roughly by adding length+value.
		raw := make([]byte, len(input)*2+1)
		raw[0] = 0
		for i, v := range input {
			raw[1+2*i] = byte(v)
			raw[1+2*i+1] = byte(v >> 8)
		}
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		requireFindArchsimdAVX2(t)
		if len(raw) == 0 {
			if got, want := findUTF16Archsimd(nil, 0), findUTF16Scalar(nil, 0); got != want {
				t.Fatalf("findUTF16Archsimd = %d, scalar = %d", got, want)
			}
			return
		}
		value := uint16(raw[0]) | uint16(raw[0])<<8
		units := make([]uint16, (len(raw)-1)/2)
		for i := range units {
			units[i] = uint16(raw[1+2*i]) | uint16(raw[1+2*i+1])<<8
		}
		if got, want := findUTF16Archsimd(units, value), findUTF16Scalar(units, value); got != want {
			t.Fatalf("findUTF16Archsimd = %d, scalar = %d (len=%d value=%#x)", got, want, len(units), value)
		}
	})
}
