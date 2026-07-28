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
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"testing"
)

// Hand-authored Go-only direct scalar-differential coverage for the pinned
// Westmere and Haswell UTF-8 length families in
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// src/generic/utf8/utf16_length_from_utf8_bytemask.h,
// src/generic/utf8.h:8-20, and the corresponding target simd.h files.

func TestUTF8LengthAMD64ScalarParity(t *testing.T) {
	for _, input := range [][]byte{
		nil,
		{},
		[]byte("plain ASCII"),
		{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88},
		{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xf4, 0xf8, 0xff},
	} {
		checkUTF8LengthAMD64(t, input)
	}

	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257, 1024, 4097, 65536}
	for _, length := range lengths {
		for alignment := 0; alignment < 32; alignment++ {
			t.Run("length="+strconv.Itoa(length)+"/alignment="+strconv.Itoa(alignment), func(t *testing.T) {
				guard := newGuardedSlice(alignment, length, 33, byte(0xa5))
				for i := range guard.body {
					guard.body[i] = byte(i*131 + length*17 + alignment)
				}
				before := slices.Clone(guard.storage)
				checkUTF8LengthAMD64(t, guard.body)
				guard.requireCanariesIntact(t)
				if !slices.Equal(guard.storage, before) {
					t.Fatal("UTF-8 length amd64 input or canary modified")
				}
			})
		}
	}
}

func TestUTF8LengthAMD64ShortInputGuardContracts(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "utf8_length_amd64.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	functions := make(map[string]*ast.FuncDecl)
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			functions[function.Name.Name] = function
		}
	}
	tests := []struct {
		wrapper string
		scalar  string
		raw     string
	}{
		{"utf16LengthFromUTF8Westmere", "utf16LengthFromUTF8Scalar", "utf16LengthFromUTF8BlocksWestmere"},
		{"utf16LengthFromUTF8Haswell", "utf16LengthFromUTF8Scalar", "utf16LengthFromUTF8BlocksHaswell"},
		{"utf32LengthFromUTF8Westmere", "utf32LengthFromUTF8Scalar", "utf32LengthFromUTF8BlocksWestmere"},
		{"utf32LengthFromUTF8Haswell", "utf32LengthFromUTF8Scalar", "utf32LengthFromUTF8BlocksHaswell"},
	}
	for _, test := range tests {
		t.Run(test.wrapper, func(t *testing.T) {
			function := functions[test.wrapper]
			if function == nil || function.Body == nil {
				t.Fatalf("function %s not found", test.wrapper)
			}

			guardIndex, rawCallIndex := -1, -1
			for index, statement := range function.Body.List {
				if rawCallIndex < 0 && callsNamed(statement, test.raw) {
					rawCallIndex = index
				}
				guard, ok := statement.(*ast.IfStmt)
				if guardIndex >= 0 || !ok {
					continue
				}
				condition, ok := guard.Cond.(*ast.BinaryExpr)
				if !ok || condition.Op != token.EQL {
					continue
				}
				complete, completeOK := condition.X.(*ast.Ident)
				zero, zeroOK := condition.Y.(*ast.BasicLit)
				if !completeOK || complete.Name != "complete" || !zeroOK || zero.Kind != token.INT || zero.Value != "0" {
					continue
				}
				guardIndex = index
				if guard.Else != nil || len(guard.Body.List) != 1 {
					t.Fatal("complete == 0 guard must have one unconditional return")
				}
				result, ok := guard.Body.List[0].(*ast.ReturnStmt)
				if !ok || len(result.Results) != 1 || !callsNamed(result.Results[0], test.scalar) {
					t.Fatalf("complete == 0 guard must return %s(input)", test.scalar)
				}
			}
			if guardIndex < 0 {
				t.Fatal("complete == 0 guard not found")
			}
			if rawCallIndex < 0 {
				t.Fatalf("function does not call %s", test.raw)
			}
			if guardIndex >= rawCallIndex {
				t.Fatalf("complete == 0 guard must precede %s", test.raw)
			}
		})
	}
}

func callsNamed(node ast.Node, function string) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if ok && identifier.Name == function {
			found = true
		}
		return !found
	})
	return found
}

func TestUTF8LengthAMD64AllByteValues(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	if got := latin1LengthFromUTF8Scalar(all); got != 192 {
		t.Fatalf("one-cycle Latin-1 length = %d, want 192", got)
	}
	if got := utf16LengthFromUTF8Scalar(all); got != 208 {
		t.Fatalf("one-cycle UTF-16 length = %d, want 208", got)
	}
	if got := utf32LengthFromUTF8Scalar(all); got != 192 {
		t.Fatalf("one-cycle UTF-32 length = %d, want 192", got)
	}
	for _, input := range [][]byte{
		all,
		append(slices.Clone(all), all...),
		bytes.Repeat([]byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xff}, 257),
	} {
		checkUTF8LengthAMD64(t, input)
	}
	for value := 0; value <= 0xff; value++ {
		checkUTF8LengthAMD64(t, bytes.Repeat([]byte{byte(value)}, 257))
	}
}

func TestUTF8LengthAMD64RawContracts(t *testing.T) {
	lengths := []int{0, 1, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129, 255, 256, 257, 2031, 2032, 2033, 2047, 2048, 2049, 4063, 4064, 4065, 4095, 4096, 4097, 65536}
	for _, length := range lengths {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i*29 + length)
		}
		if got, want := utf16LengthFromUTF8BlocksWestmere(input), utf16LengthFromUTF8Scalar(input[:length&^15]); got != want {
			t.Errorf("Westmere raw UTF-16 length %d = %d, scalar = %d", length, got, want)
		}
		if hasUTF8LengthAVX2() {
			if got, want := utf16LengthFromUTF8BlocksHaswell(input), utf16LengthFromUTF8Scalar(input[:length&^31]); got != want {
				t.Errorf("Haswell raw UTF-16 length %d = %d, scalar = %d", length, got, want)
			}
		}
		if hasUTF8LengthPOPCNT() {
			if got, want := utf32LengthFromUTF8BlocksWestmere(input), utf32LengthFromUTF8Scalar(input[:length&^63]); got != want {
				t.Errorf("Westmere raw UTF-32 length %d = %d, scalar = %d", length, got, want)
			}
		}
		if hasUTF8LengthAVX2() && hasUTF8LengthPOPCNT() {
			if got, want := utf32LengthFromUTF8BlocksHaswell(input), utf32LengthFromUTF8Scalar(input[:length&^63]); got != want {
				t.Errorf("Haswell raw UTF-32 length %d = %d, scalar = %d", length, got, want)
			}
		}
	}
}

func TestUTF16LengthFromUTF8AMD64FlushBoundaries(t *testing.T) {
	westmere := []int{2031, 2032, 2033, 2047, 2048, 2049, 2*2032 - 1, 2 * 2032, 2*2032 + 1, 1 << 20}
	haswell := []int{4063, 4064, 4065, 4095, 4096, 4097, 2*4064 - 1, 2 * 4064, 2*4064 + 1, 1 << 20}
	for _, value := range []byte{0x00, 0x80, 0xf0, 0xff} {
		for _, length := range westmere {
			input := bytes.Repeat([]byte{value}, length)
			if got, want := utf16LengthFromUTF8Westmere(input), utf16LengthFromUTF8Scalar(input); got != want {
				t.Errorf("Westmere byte %#02x length %d = %d, scalar = %d", value, length, got, want)
			}
		}
		if hasUTF8LengthAVX2() {
			for _, length := range haswell {
				input := bytes.Repeat([]byte{value}, length)
				if got, want := utf16LengthFromUTF8Haswell(input), utf16LengthFromUTF8Scalar(input); got != want {
					t.Errorf("Haswell byte %#02x length %d = %d, scalar = %d", value, length, got, want)
				}
			}
		}
	}
}

func checkUTF8LengthAMD64(t *testing.T, input []byte) {
	t.Helper()
	if got, want := latin1LengthFromUTF8Westmere(input), latin1LengthFromUTF8Scalar(input); got != want {
		t.Errorf("latin1LengthFromUTF8Westmere = %d, scalar = %d", got, want)
	}
	if got, want := utf16LengthFromUTF8Westmere(input), utf16LengthFromUTF8Scalar(input); got != want {
		t.Errorf("utf16LengthFromUTF8Westmere = %d, scalar = %d", got, want)
	}
	if hasUTF8LengthAVX2() {
		if got, want := latin1LengthFromUTF8Haswell(input), latin1LengthFromUTF8Scalar(input); got != want {
			t.Errorf("latin1LengthFromUTF8Haswell = %d, scalar = %d", got, want)
		}
		if got, want := utf16LengthFromUTF8Haswell(input), utf16LengthFromUTF8Scalar(input); got != want {
			t.Errorf("utf16LengthFromUTF8Haswell = %d, scalar = %d", got, want)
		}
	}
	if hasUTF8LengthPOPCNT() {
		if got, want := utf32LengthFromUTF8Westmere(input), utf32LengthFromUTF8Scalar(input); got != want {
			t.Errorf("utf32LengthFromUTF8Westmere = %d, scalar = %d", got, want)
		}
	}
	if hasUTF8LengthAVX2() && hasUTF8LengthPOPCNT() {
		if got, want := utf32LengthFromUTF8Haswell(input), utf32LengthFromUTF8Scalar(input); got != want {
			t.Errorf("utf32LengthFromUTF8Haswell = %d, scalar = %d", got, want)
		}
	}
}

func hasUTF8LengthAVX2() bool {
	return detectHostFeatures()&cpuAVX2 == cpuAVX2
}

func hasUTF8LengthPOPCNT() bool {
	return detectHostFeatures()&cpuPOPCNT == cpuPOPCNT
}
