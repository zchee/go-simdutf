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
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"testing"
)

// Test vectors translated and adapted from
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f (tree
// eb5429bb160dfdf1a7d208f0184d3379940e69ee):
// tests/null_safety_tests.cpp:65-73, tests/simdutf_c_tests.cpp:254-265,
// tests/readme_tests.cpp:122-141, and include/simdutf/scalar/utf8.h:258-325.

func TestUTF8LengthsUpstreamCases(t *testing.T) {
	cases := map[string]struct {
		input                []byte
		latin1, utf16, utf32 int
	}{
		"nil":            {input: nil},
		"empty":          {input: []byte{}},
		"hello":          {input: []byte("hello"), latin1: 5, utf16: 5, utf32: 5},
		"one-to-four":    {input: []byte{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88}, latin1: 4, utf16: 5, utf32: 4},
		"stream-literal": {input: []byte("école d'été")[:10], latin1: 9, utf16: 9, utf32: 9},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := Latin1LengthFromUTF8(test.input); got != test.latin1 {
				t.Errorf("Latin1LengthFromUTF8() = %d, want %d", got, test.latin1)
			}
			if got := UTF16LengthFromUTF8(test.input); got != test.utf16 {
				t.Errorf("UTF16LengthFromUTF8() = %d, want %d", got, test.utf16)
			}
			if got := UTF32LengthFromUTF8(test.input); got != test.utf32 {
				t.Errorf("UTF32LengthFromUTF8() = %d, want %d", got, test.utf32)
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("UTF-8 length helper modified input")
			}
		})
	}
}

func TestUTF8LengthsAreNotBOMAware(t *testing.T) {
	input := []byte{0xef, 0xbb, 0xbf}
	before := bytes.Clone(input)
	tests := map[string]struct {
		public func([]byte) int
		scalar func([]byte) int
	}{
		"Latin1LengthFromUTF8": {public: Latin1LengthFromUTF8, scalar: latin1LengthFromUTF8Scalar},
		"UTF16LengthFromUTF8":  {public: UTF16LengthFromUTF8, scalar: utf16LengthFromUTF8Scalar},
		"UTF32LengthFromUTF8":  {public: UTF32LengthFromUTF8, scalar: utf32LengthFromUTF8Scalar},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := test.public(input); got != 1 {
				t.Errorf("public result = %d, want 1", got)
			}
			if got := test.scalar(input); got != 1 {
				t.Errorf("scalar result = %d, want 1", got)
			}
		})
	}
	if !bytes.Equal(input, before) {
		t.Fatal("UTF-8 length helper modified BOM input")
	}
}

func TestUTF8LengthScalarArbitraryByteFormula(t *testing.T) {
	input := []byte{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xf4, 0xf8, 0xff}
	if got, want := latin1LengthFromUTF8Scalar(input), 8; got != want {
		t.Errorf("latin1LengthFromUTF8Scalar() = %d, want %d", got, want)
	}
	if got, want := utf16LengthFromUTF8Scalar(input), 12; got != want {
		t.Errorf("utf16LengthFromUTF8Scalar() = %d, want %d", got, want)
	}
	if got, want := utf32LengthFromUTF8Scalar(input), 8; got != want {
		t.Errorf("utf32LengthFromUTF8Scalar() = %d, want %d", got, want)
	}
	for value := 0; value <= 0xff; value++ {
		input := []byte{byte(value)}
		codePoints := 1
		if value >= 0x80 && value <= 0xbf {
			codePoints = 0
		}
		utf16 := codePoints
		if value >= 0xf0 {
			utf16++
		}
		if got := Latin1LengthFromUTF8(input); got != codePoints {
			t.Errorf("Latin1LengthFromUTF8({%#02x}) = %d, want %d", value, got, codePoints)
		}
		if got := UTF16LengthFromUTF8(input); got != utf16 {
			t.Errorf("UTF16LengthFromUTF8({%#02x}) = %d, want %d", value, got, utf16)
		}
		if got := UTF32LengthFromUTF8(input); got != codePoints {
			t.Errorf("UTF32LengthFromUTF8({%#02x}) = %d, want %d", value, got, codePoints)
		}
	}
}

func TestUTF8LengthPublicScalarBoundaryParity(t *testing.T) {
	tests := []struct {
		name    string
		lengths []int
		public  func([]byte) int
		scalar  func([]byte) int
	}{
		{
			name:    "UTF16",
			lengths: []int{0, 1, 14, 15, 16, 17, 31, 32, 33},
			public:  UTF16LengthFromUTF8,
			scalar:  utf16LengthFromUTF8Scalar,
		},
		{
			name:    "UTF32",
			lengths: []int{0, 1, 62, 63, 64, 65, 127, 128, 129},
			public:  UTF32LengthFromUTF8,
			scalar:  utf32LengthFromUTF8Scalar,
		},
	}
	for _, test := range tests {
		for _, length := range test.lengths {
			t.Run(test.name+"/length="+strconv.Itoa(length), func(t *testing.T) {
				input := make([]byte, length)
				for i := range input {
					input[i] = byte(i*131 + length*17)
				}
				before := slices.Clone(input)
				if got, want := test.public(input), test.scalar(input); got != want {
					t.Errorf("public = %d, scalar = %d", got, want)
				}
				if !slices.Equal(input, before) {
					t.Fatal("length helper modified input")
				}
			})
		}
	}
}

func TestLatin1LengthFromUTF8MatchesCountUTF8(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	inputs := [][]byte{nil, {}, all}
	for _, length := range []int{1, 15, 16, 17, 63, 64, 65, 127, 128, 129} {
		input := make([]byte, length)
		for i := range input {
			input[i] = byte(i*29 + length)
		}
		inputs = append(inputs, input)
	}
	for index, input := range inputs {
		t.Run(strconv.Itoa(index)+"/length="+strconv.Itoa(len(input)), func(t *testing.T) {
			before := slices.Clone(input)
			want := countUTF8Scalar(input)
			if got := CountUTF8(input); got != want {
				t.Errorf("CountUTF8 = %d, scalar = %d", got, want)
			}
			if got := Latin1LengthFromUTF8(input); got != want {
				t.Errorf("Latin1LengthFromUTF8 = %d, scalar = %d", got, want)
			}
			if !slices.Equal(input, before) {
				t.Fatal("CountUTF8 or Latin1LengthFromUTF8 modified input")
			}
		})
	}
}

func TestUTF8LengthPublicShortInputGuardSourceShape(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "utf8_length.go", nil, 0)
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
		function, cutoff, scalar, field string
	}{
		{"UTF16LengthFromUTF8", "utf16LengthFromUTF8DispatchCutoff", "utf16LengthFromUTF8Scalar", "utf16LengthFromUTF8"},
		{"UTF32LengthFromUTF8", "utf32LengthFromUTF8DispatchCutoff", "utf32LengthFromUTF8Scalar", "utf32LengthFromUTF8"},
	}
	for _, test := range tests {
		t.Run(test.function, func(t *testing.T) {
			function := functions[test.function]
			if function == nil || function.Body == nil || len(function.Body.List) != 2 {
				t.Fatalf("%s must contain exactly a guard and dispatch return", test.function)
			}
			guard, ok := function.Body.List[0].(*ast.IfStmt)
			if !ok || guard.Else != nil || len(guard.Body.List) != 1 {
				t.Fatal("first statement must be an unconditional short-input guard")
			}
			condition, ok := guard.Cond.(*ast.BinaryExpr)
			if !ok || condition.Op != token.LSS || !isLenInputCall(condition.X) || !isIdentifier(condition.Y, test.cutoff) {
				t.Fatalf("guard must be len(input) < %s", test.cutoff)
			}
			guardReturn, ok := guard.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(guardReturn.Results) != 1 || !isDirectCall(guardReturn.Results[0], test.scalar) {
				t.Fatalf("short-input guard must return %s(input)", test.scalar)
			}
			dispatchReturn, ok := function.Body.List[1].(*ast.ReturnStmt)
			if !ok || len(dispatchReturn.Results) != 1 || !isImplementationCall(dispatchReturn.Results[0], test.field) {
				t.Fatalf("guard must precede activeImplementation.%s(input)", test.field)
			}
		})
	}
}

func isLenInputCall(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	return ok && len(call.Args) == 1 && isIdentifier(call.Fun, "len") && isIdentifier(call.Args[0], "input")
}

func isIdentifier(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name
}

func isDirectCall(expression ast.Expr, function string) bool {
	call, ok := expression.(*ast.CallExpr)
	return ok && len(call.Args) == 1 && isIdentifier(call.Fun, function) && isIdentifier(call.Args[0], "input")
}

func isImplementationCall(expression ast.Expr, field string) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !isIdentifier(call.Args[0], "input") {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && isIdentifier(selector.X, "activeImplementation") && selector.Sel.Name == field
}

func TestTrimPartialUTF8UpstreamTailCases(t *testing.T) {
	cases := map[string]struct {
		input []byte
		want  int
	}{
		"nil":                      {input: nil, want: 0},
		"empty":                    {input: []byte{}, want: 0},
		"ASCII":                    {input: []byte("abc"), want: 3},
		"complete two byte":        {input: []byte{0xc3, 0xa9}, want: 2},
		"partial two byte":         {input: []byte{'a', 0xc3}, want: 1},
		"partial three one byte":   {input: []byte{'a', 0xe2}, want: 1},
		"partial three two bytes":  {input: []byte{'a', 0xe2, 0x82}, want: 1},
		"partial four one byte":    {input: []byte{'a', 0xf0}, want: 1},
		"partial four two bytes":   {input: []byte{'a', 0xf0, 0x90}, want: 1},
		"partial four three bytes": {input: []byte{'a', 0xf0, 0x90, 0x8d}, want: 1},
		"streaming example":        {input: []byte("école d'été")[:10], want: 9},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			if got := TrimPartialUTF8(test.input); got != test.want {
				t.Errorf("TrimPartialUTF8() = %d, want %d", got, test.want)
			}
			if got := trimPartialUTF8Scalar(test.input); got != test.want {
				t.Errorf("trimPartialUTF8Scalar() = %d, want %d", got, test.want)
			}
			if !bytes.Equal(test.input, before) {
				t.Fatal("TrimPartialUTF8 modified input")
			}
		})
	}
}
