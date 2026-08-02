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
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
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

// Hand-authored Go-only direct UTF-8 length benchmark registry scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
// benchmarks/shortbench.cpp:29-65,419-422,493-497,520-526 and
// benchmarks/src/benchmark.cpp:167-169,999-1011. It defines test-only named
// variant slots and adds no product behavior or mutable dispatch override.

type utf8LengthDirectVariant struct {
	name   string
	latin1 variant[func([]byte) int]
	utf16  variant[func([]byte) int]
	utf32  variant[func([]byte) int]
}

var utf8LengthDirectVariants []utf8LengthDirectVariant

func registerUTF8LengthDirectVariant(candidate utf8LengthDirectVariant) {
	if candidate.name == "" || !validUTF8LengthVariantCells(candidate.latin1, candidate.utf16, candidate.utf32) {
		panic("simdutf: invalid direct UTF-8 length benchmark variant")
	}
	for _, registered := range utf8LengthDirectVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 length benchmark variant " + candidate.name)
		}
	}
	utf8LengthDirectVariants = append(utf8LengthDirectVariants, candidate)
}

func validUTF8LengthVariantCells(cells ...variant[func([]byte) int]) bool {
	implemented := false
	for _, cell := range cells {
		if cell.value == nil {
			if cell.available || cell.kind != implementationScalar || cell.required != 0 {
				panic("simdutf: inconsistent absent UTF-8 length variant cell")
			}
			continue
		}
		if !cell.available {
			panic("simdutf: unavailable UTF-8 length variant cell has a function")
		}
		implemented = true
	}
	return implemented
}

func TestRegisterUTF8LengthDirectVariant(t *testing.T) {
	saved := utf8LengthDirectVariants
	defer func() { utf8LengthDirectVariants = saved }()

	tests := []struct {
		name          string
		backend       string
		implemented   uint8
		availableNil  uint8
		unavailableFn uint8
		duplicate     bool
		wantPanic     bool
	}{
		{name: "partial UTF16 only", backend: "test-utf16", implemented: 1 << 1},
		{name: "partial UTF32 only", backend: "test-utf32", implemented: 1 << 2},
		{name: "mixed implemented and not applicable", backend: "test-mixed", implemented: 1<<0 | 1<<2},
		{name: "empty name", implemented: 1 << 1, wantPanic: true},
		{name: "no implemented cells", backend: "test-empty", wantPanic: true},
		{name: "available operation with nil value", backend: "test-nil", availableNil: 1 << 1, wantPanic: true},
		{name: "unavailable operation with value", backend: "test-unavailable", unavailableFn: 1 << 2, wantPanic: true},
		{name: "duplicate name", backend: "test-duplicate", implemented: 1 << 1, duplicate: true, wantPanic: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			utf8LengthDirectVariants = nil
			var invoked [3]int
			candidate := makeUTF8LengthDirectTestVariant(test.backend, test.implemented, &invoked)
			setInvalidUTF8LengthDirectTestCell(&candidate, test.availableNil, true)
			setInvalidUTF8LengthDirectTestCell(&candidate, test.unavailableFn, false)
			if test.duplicate {
				registerUTF8LengthDirectVariant(candidate)
			}

			panicked := didPanic(func() { registerUTF8LengthDirectVariant(candidate) })
			if panicked != test.wantPanic {
				t.Fatalf("register panic = %t, want %t", panicked, test.wantPanic)
			}
			if test.wantPanic {
				return
			}

			got := utf8LengthDirectVariants[len(utf8LengthDirectVariants)-1]
			input := []byte("a\xf0\x90\x8d\x88")
			want := [...]int{
				latin1LengthFromUTF8Scalar(input),
				utf16LengthFromUTF8Scalar(input),
				utf32LengthFromUTF8Scalar(input),
			}
			cells := [...]variant[func([]byte) int]{got.latin1, got.utf16, got.utf32}
			for i, cell := range cells {
				if !cell.supportedBy(selectionInput{}) {
					if cell.value != nil {
						t.Fatalf("unsupported cell %d has a function", i)
					}
					continue
				}
				if result := cell.value(input); result != want[i] {
					t.Errorf("cell %d result = %d, scalar = %d", i, result, want[i])
				}
			}
			for i := range invoked {
				wantInvoked := 0
				if test.implemented&(1<<i) != 0 {
					wantInvoked = 1
				}
				if invoked[i] != wantInvoked {
					t.Errorf("cell %d invoked %d times, want %d", i, invoked[i], wantInvoked)
				}
			}
		})
	}
}

func makeUTF8LengthDirectTestVariant(name string, implemented uint8, invoked *[3]int) utf8LengthDirectVariant {
	functions := [...]func([]byte) int{
		func(input []byte) int { invoked[0]++; return latin1LengthFromUTF8Scalar(input) },
		func(input []byte) int { invoked[1]++; return utf16LengthFromUTF8Scalar(input) },
		func(input []byte) int { invoked[2]++; return utf32LengthFromUTF8Scalar(input) },
	}
	cells := [3]variant[func([]byte) int]{}
	for i := range cells {
		if implemented&(1<<i) != 0 {
			cells[i] = variant[func([]byte) int]{value: functions[i], available: true}
		}
	}
	return utf8LengthDirectVariant{name: name, latin1: cells[0], utf16: cells[1], utf32: cells[2]}
}

func setInvalidUTF8LengthDirectTestCell(candidate *utf8LengthDirectVariant, mask uint8, available bool) {
	if mask == 0 {
		return
	}
	cell := variant[func([]byte) int]{available: available}
	if !available {
		cell.value = utf32LengthFromUTF8Scalar
	}
	switch mask {
	case 1 << 0:
		candidate.latin1 = cell
	case 1 << 1:
		candidate.utf16 = cell
	case 1 << 2:
		candidate.utf32 = cell
	}
}

func didPanic(run func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	run()
	return false
}

// Hand-authored Go-only direct UTF-8 length differential fuzz registry
// scaffolding for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): fuzz/conversion.cpp,
// fuzz/roundtrip.cpp, fuzz/misc.cpp, and include/simdutf/scalar/utf8.h:258-325.
// It defines test metadata only and adds no product behavior or mutable
// dispatch override.

type utf8LengthFuzzVariant struct {
	name   string
	latin1 variant[func([]byte) int]
	utf16  variant[func([]byte) int]
	utf32  variant[func([]byte) int]
}

var utf8LengthFuzzVariants []utf8LengthFuzzVariant

func registerUTF8LengthFuzzVariant(candidate utf8LengthFuzzVariant) {
	if candidate.name == "" || !validUTF8LengthVariantCells(candidate.latin1, candidate.utf16, candidate.utf32) {
		panic("simdutf: invalid direct UTF-8 length fuzz variant")
	}
	for _, registered := range utf8LengthFuzzVariants {
		if registered.name == candidate.name {
			panic("simdutf: duplicate direct UTF-8 length fuzz variant " + candidate.name)
		}
	}
	utf8LengthFuzzVariants = append(utf8LengthFuzzVariants, candidate)
}

func TestRegisterUTF8LengthFuzzVariant(t *testing.T) {
	saved := utf8LengthFuzzVariants
	defer func() { utf8LengthFuzzVariants = saved }()

	tests := []struct {
		name          string
		backend       string
		implemented   uint8
		availableNil  uint8
		unavailableFn uint8
		duplicate     bool
		wantPanic     bool
	}{
		{name: "partial UTF16 only", backend: "test-utf16", implemented: 1 << 1},
		{name: "partial UTF32 only", backend: "test-utf32", implemented: 1 << 2},
		{name: "mixed implemented and not applicable", backend: "test-mixed", implemented: 1<<0 | 1<<2},
		{name: "empty name", implemented: 1 << 1, wantPanic: true},
		{name: "no implemented cells", backend: "test-empty", wantPanic: true},
		{name: "available operation with nil value", backend: "test-nil", availableNil: 1 << 1, wantPanic: true},
		{name: "unavailable operation with value", backend: "test-unavailable", unavailableFn: 1 << 2, wantPanic: true},
		{name: "duplicate name", backend: "test-duplicate", implemented: 1 << 1, duplicate: true, wantPanic: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			utf8LengthFuzzVariants = nil
			var invoked [3]int
			candidate := makeUTF8LengthFuzzTestVariant(test.backend, test.implemented, &invoked)
			setInvalidUTF8LengthFuzzTestCell(&candidate, test.availableNil, true)
			setInvalidUTF8LengthFuzzTestCell(&candidate, test.unavailableFn, false)
			if test.duplicate {
				registerUTF8LengthFuzzVariant(candidate)
			}

			panicked := didPanic(func() { registerUTF8LengthFuzzVariant(candidate) })
			if panicked != test.wantPanic {
				t.Fatalf("register panic = %t, want %t", panicked, test.wantPanic)
			}
			if test.wantPanic {
				return
			}

			got := utf8LengthFuzzVariants[len(utf8LengthFuzzVariants)-1]
			input := []byte("a\xf0\x90\x8d\x88")
			want := [...]int{
				latin1LengthFromUTF8Scalar(input),
				utf16LengthFromUTF8Scalar(input),
				utf32LengthFromUTF8Scalar(input),
			}
			cells := [...]variant[func([]byte) int]{got.latin1, got.utf16, got.utf32}
			for i, cell := range cells {
				if !cell.supportedBy(selectionInput{}) {
					if cell.value != nil {
						t.Fatalf("unsupported cell %d has a function", i)
					}
					continue
				}
				if result := cell.value(input); result != want[i] {
					t.Errorf("cell %d result = %d, scalar = %d", i, result, want[i])
				}
			}
			for i := range invoked {
				wantInvoked := 0
				if test.implemented&(1<<i) != 0 {
					wantInvoked = 1
				}
				if invoked[i] != wantInvoked {
					t.Errorf("cell %d invoked %d times, want %d", i, invoked[i], wantInvoked)
				}
			}
		})
	}
}

func makeUTF8LengthFuzzTestVariant(name string, implemented uint8, invoked *[3]int) utf8LengthFuzzVariant {
	functions := [...]func([]byte) int{
		func(input []byte) int { invoked[0]++; return latin1LengthFromUTF8Scalar(input) },
		func(input []byte) int { invoked[1]++; return utf16LengthFromUTF8Scalar(input) },
		func(input []byte) int { invoked[2]++; return utf32LengthFromUTF8Scalar(input) },
	}
	cells := [3]variant[func([]byte) int]{}
	for i := range cells {
		if implemented&(1<<i) != 0 {
			cells[i] = variant[func([]byte) int]{value: functions[i], available: true}
		}
	}
	return utf8LengthFuzzVariant{name: name, latin1: cells[0], utf16: cells[1], utf32: cells[2]}
}

func setInvalidUTF8LengthFuzzTestCell(candidate *utf8LengthFuzzVariant, mask uint8, available bool) {
	if mask == 0 {
		return
	}
	cell := variant[func([]byte) int]{available: available}
	if !available {
		cell.value = utf32LengthFromUTF8Scalar
	}
	switch mask {
	case 1 << 0:
		candidate.latin1 = cell
	case 1 << 1:
		candidate.utf16 = cell
	case 1 << 2:
		candidate.utf32 = cell
	}
}

// Go-only public/direct-dispatch-versus-scalar differential fuzz scaffold for
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f): fuzz/conversion.cpp,
// fuzz/roundtrip.cpp, fuzz/misc.cpp, and include/simdutf/scalar/utf8.h:258-325.
// The scalar functions are the permanent arbitrary-byte Go oracles.

func FuzzUTF8Lengths(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("hello"),
		{'a', 0xc2, 0xa2, 0xe2, 0x82, 0xac, 0xf0, 0x90, 0x8d, 0x88},
		{0x00, 0x7f, 0x80, 0xbf, 0xc0, 0xef, 0xf0, 0xf4, 0xf8, 0xff},
	} {
		f.Add(seed)
	}
	for _, boundary := range []int{16, 32, 64, 128} {
		for _, size := range []int{boundary - 1, boundary, boundary + 1} {
			seed := bytes.Repeat([]byte{'a'}, size)
			seed[len(seed)-1] = 0x80
			// Start four bytes before the first byte past the boundary so the
			// complete boundary+1 seed carries a four-byte sequence across it.
			lead := boundary - 3
			if lead < len(seed) {
				seed[lead] = 0xf0
				for i := lead + 1; i < len(seed) && i < lead+4; i++ {
					seed[i] = 0x80
				}
			}
			f.Add(seed)
		}
	}
	selection := detectSelectionInput()
	f.Fuzz(func(t *testing.T, input []byte) {
		for _, prefix := range [...]int{1, 2, 3, 5, 7, 15, 31, 63} {
			guard := newGuardedSlice(prefix, len(input), 67, byte(0xa5))
			copy(guard.body, input)
			before := bytes.Clone(guard.storage)
			wantLatin1 := latin1LengthFromUTF8Scalar(guard.body)
			wantUTF16 := utf16LengthFromUTF8Scalar(guard.body)
			wantUTF32 := utf32LengthFromUTF8Scalar(guard.body)
			if got := Latin1LengthFromUTF8(guard.body); got != wantLatin1 {
				t.Errorf("prefix %d: Latin1LengthFromUTF8() = %d, scalar = %d", prefix, got, wantLatin1)
			}
			if got := activeImplementation.latin1LengthFromUTF8(guard.body); got != wantLatin1 {
				t.Errorf("prefix %d: direct latin1LengthFromUTF8() = %d, scalar = %d", prefix, got, wantLatin1)
			}
			if got := UTF16LengthFromUTF8(guard.body); got != wantUTF16 {
				t.Errorf("prefix %d: UTF16LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF16)
			}
			if got := activeImplementation.utf16LengthFromUTF8(guard.body); got != wantUTF16 {
				t.Errorf("prefix %d: direct utf16LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF16)
			}
			if got := UTF32LengthFromUTF8(guard.body); got != wantUTF32 {
				t.Errorf("prefix %d: UTF32LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF32)
			}
			if got := activeImplementation.utf32LengthFromUTF8(guard.body); got != wantUTF32 {
				t.Errorf("prefix %d: direct utf32LengthFromUTF8() = %d, scalar = %d", prefix, got, wantUTF32)
			}
			for _, candidate := range utf8LengthFuzzVariants {
				if candidate.latin1.supportedBy(selection) {
					if got := candidate.latin1.value(guard.body); got != wantLatin1 {
						t.Errorf("prefix %d: %s Latin1LengthFromUTF8() = %d, scalar = %d", prefix, candidate.name, got, wantLatin1)
					}
				}
				if candidate.utf16.supportedBy(selection) {
					if got := candidate.utf16.value(guard.body); got != wantUTF16 {
						t.Errorf("prefix %d: %s UTF16LengthFromUTF8() = %d, scalar = %d", prefix, candidate.name, got, wantUTF16)
					}
				}
				if candidate.utf32.supportedBy(selection) {
					if got := candidate.utf32.value(guard.body); got != wantUTF32 {
						t.Errorf("prefix %d: %s UTF32LengthFromUTF8() = %d, scalar = %d", prefix, candidate.name, got, wantUTF32)
					}
				}
			}
			guard.requireCanariesIntact(t)
			if !bytes.Equal(guard.storage, before) {
				t.Errorf("prefix %d: UTF-8 length helper modified input or canaries", prefix)
			}
		}
	})
}

func FuzzTrimPartialUTF8(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{},
		[]byte("abc"),
		{'a', 0xc3},
		{'a', 0xe2, 0x82},
		{'a', 0xf0, 0x90, 0x8d},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		guard := newGuardedSlice(5, len(input), 7, byte(0xa5))
		copy(guard.body, input)
		before := bytes.Clone(guard.storage)
		got := TrimPartialUTF8(guard.body)
		if want := trimPartialUTF8Scalar(guard.body); got != want {
			t.Errorf("TrimPartialUTF8() = %d, scalar = %d", got, want)
		}
		// Pinned fuzz/misc.cpp rejects ret + 3 < N or ret > N. Express the
		// same bounds without overflowing an addition to the returned length.
		if got > len(guard.body) {
			t.Errorf("TrimPartialUTF8() = %d, input length = %d", got, len(guard.body))
		} else if len(guard.body)-got > 3 {
			t.Errorf("TrimPartialUTF8() removed %d bytes, want at most 3", len(guard.body)-got)
		}
		guard.requireCanariesIntact(t)
		if !bytes.Equal(guard.storage, before) {
			t.Fatal("TrimPartialUTF8 modified input or canaries")
		}
	})
}
