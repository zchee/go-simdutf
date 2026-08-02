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

//go:build amd64

package simdutf

import (
	"math/bits"
	"reflect"
	"testing"
)

func TestDirectAMD64UTFValidationAgainstScalar(t *testing.T) {
	utf16Cases := [][]uint16{
		nil,
		{0x41},
		{0xd800},
		{0xdc00},
		{0xd800, 0xdc00},
		{0x41, 0xd800, 0x42},
		{0x41, 0xdc00, 0x42},
		{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0xd800, 0xdc00, 0x48},
	}
	providers := []struct {
		name string
		le   func([]uint16) bool
		be   func([]uint16) bool
		leE  func([]uint16) Result
		beE  func([]uint16) Result
	}{
		{"westmere", validateUTF16LEWestmere, validateUTF16BEWestmere, validateUTF16LEWithErrorsWestmere, validateUTF16BEWithErrorsWestmere},
		{"haswell", validateUTF16LEHaswell, validateUTF16BEHaswell, validateUTF16LEWithErrorsHaswell, validateUTF16BEWithErrorsHaswell},
	}
	for _, p := range providers {
		for _, in := range utf16Cases {
			be := make([]uint16, len(in))
			for i := range in {
				be[i] = bits.ReverseBytes16(in[i])
			}
			if got, want := p.le(in), validateUTF16LEScalar(in); got != want {
				t.Fatalf("%s LE bool %#v: got %v want %v", p.name, in, got, want)
			}
			if got, want := p.be(be), validateUTF16BEScalar(be); got != want {
				t.Fatalf("%s BE bool %#v: got %v want %v", p.name, in, got, want)
			}
			if got, want := p.leE(in), validateUTF16LEWithErrorsScalar(in); got != want {
				t.Fatalf("%s LE result %#v: got %#v want %#v", p.name, in, got, want)
			}
			if got, want := p.beE(be), validateUTF16BEWithErrorsScalar(be); got != want {
				t.Fatalf("%s BE result %#v: got %#v want %#v", p.name, in, got, want)
			}
		}
	}
	utf32Cases := [][]uint32{nil, {0}, {0x7f, 0x80}, {0x10ffff}, {0x110000}, {0xd800}, {0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x110000}}
	for _, p := range []struct {
		name   string
		ok     func([]uint32) bool
		errors func([]uint32) Result
	}{
		{"westmere", validateUTF32Westmere, validateUTF32WithErrorsWestmere},
		{"haswell", validateUTF32Haswell, validateUTF32WithErrorsHaswell},
	} {
		for _, in := range utf32Cases {
			if got, want := p.ok(in), validateUTF32Scalar(in); got != want {
				t.Fatalf("%s UTF-32 bool %#v: got %v want %v", p.name, in, got, want)
			}
			if got, want := p.errors(in), validateUTF32WithErrorsScalar(in); got != want {
				t.Fatalf("%s UTF-32 result %#v: got %#v want %#v", p.name, in, got, want)
			}
		}
	}
}

func TestDirectAMD64ToWellFormedAgainstScalar(t *testing.T) {
	input := []uint16{0x41, 0xd800, 0xdc00, 0x42, 0xd800, 0x43, 0xdc00, 0x44, 0xd800}
	providers := []struct {
		name string
		big  bool
		fn   func([]uint16, []uint16)
	}{
		{"westmere-le", false, toWellFormedUTF16LEWestmere},
		{"haswell-le", false, toWellFormedUTF16LEHaswell},
		{"westmere-be", true, toWellFormedUTF16BEWestmere},
		{"haswell-be", true, toWellFormedUTF16BEHaswell},
	}
	for _, p := range providers {
		in := append([]uint16(nil), input...)
		if p.big {
			for i := range in {
				in[i] = bits.ReverseBytes16(in[i])
			}
		}
		want := make([]uint16, len(in))
		toWellFormedUTF16Scalar(in, want, !p.big)
		backing := make([]uint16, len(in)+2)
		backing[0], backing[len(backing)-1] = 0xaaaa, 0xbbbb
		got := backing[1 : len(backing)-1]
		p.fn(in, got)
		if !reflect.DeepEqual(got, want) || backing[0] != 0xaaaa || backing[len(backing)-1] != 0xbbbb {
			t.Fatalf("%s output/canary: got %#v backing %#v want %#v", p.name, got, backing, want)
		}
		inPlace := append([]uint16(nil), in...)
		p.fn(inPlace, inPlace)
		if !reflect.DeepEqual(inPlace, want) {
			t.Fatalf("%s in-place: got %#v want %#v", p.name, inPlace, want)
		}
		short := []uint16{0xaaaa, 0xbbbb}
		didPanic := false
		func() {
			defer func() { didPanic = recover() != nil }()
			p.fn(in, short)
		}()
		if !didPanic || short[0] != 0xaaaa || short[1] != 0xbbbb {
			t.Fatalf("%s short destination did not panic before storing: %#v", p.name, short)
		}
	}
}

func FuzzUTFValidationAMD64AgainstScalar(f *testing.F) {
	for _, seed := range [][]byte{nil, {0, 0}, {0, 0xd8, 0, 0xdc}, {0, 0, 0x11, 0}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		semantic := fuzzUTF16Words(data)
		input32 := make([]uint32, len(data)/4)
		for i := range input32 {
			input32[i] = uint32(data[4*i]) | uint32(data[4*i+1])<<8 |
				uint32(data[4*i+2])<<16 | uint32(data[4*i+3])<<24
		}
		providers := []struct {
			name     string
			le       func([]uint16) Result
			be       func([]uint16) Result
			repairLE func([]uint16, []uint16)
			repairBE func([]uint16, []uint16)
			utf32    func([]uint32) Result
		}{
			{"westmere", validateUTF16LEWithErrorsWestmere, validateUTF16BEWithErrorsWestmere, toWellFormedUTF16LEWestmere, toWellFormedUTF16BEWestmere, validateUTF32WithErrorsWestmere},
			{"haswell", validateUTF16LEWithErrorsHaswell, validateUTF16BEWithErrorsHaswell, toWellFormedUTF16LEHaswell, toWellFormedUTF16BEHaswell, validateUTF32WithErrorsHaswell},
		}
		for _, provider := range providers {
			for _, little := range []bool{true, false} {
				input := rawUTF16Scalar(semantic, little)
				gotDst, wantDst := make([]uint16, len(input)), make([]uint16, len(input))
				var gotResult, wantResult Result
				if little {
					gotResult, wantResult = provider.le(input), validateUTF16LEWithErrorsScalar(input)
					provider.repairLE(input, gotDst)
					toWellFormedUTF16LEScalar(input, wantDst)
				} else {
					gotResult, wantResult = provider.be(input), validateUTF16BEWithErrorsScalar(input)
					provider.repairBE(input, gotDst)
					toWellFormedUTF16BEScalar(input, wantDst)
				}
				if gotResult != wantResult || !reflect.DeepEqual(gotDst, wantDst) {
					t.Fatalf("%s UTF-16 little=%t: result=%+v/%+v output=%x/%x", provider.name, little, gotResult, wantResult, gotDst, wantDst)
				}
			}
			if got, want := provider.utf32(input32), validateUTF32WithErrorsScalar(input32); got != want {
				t.Fatalf("%s UTF-32 result=%+v want=%+v input=%x", provider.name, got, want, input32)
			}
		}
	})
}
