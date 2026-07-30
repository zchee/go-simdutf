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
	"reflect"
	"testing"
)

// Hand-authored Go-only tests for the exact implementation-table shape,
// selected function identities, and all-or-none archsimd providers. The
// dispatch contract is pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:src/implementation.cpp
// and the per-symbol ISA/object-proof policy in
// docs/porting/provenance.md; these are not
// upstream test vectors.

func TestImplementationTableExactFields(t *testing.T) {
	typ := reflect.TypeFor[implementation]()
	want := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "validateUTF8", typ: reflect.TypeFor[func([]byte) bool]()},
		{name: "validateUTF8WithErrors", typ: reflect.TypeFor[func([]byte) Result]()},
		{name: "countUTF8", typ: reflect.TypeFor[func([]byte) int]()},
		{name: "latin1LengthFromUTF8", typ: reflect.TypeFor[func([]byte) int]()},
		{name: "utf16LengthFromUTF8", typ: reflect.TypeFor[func([]byte) int]()},
		{name: "utf32LengthFromUTF8", typ: reflect.TypeFor[func([]byte) int]()},
		{name: "validateASCII", typ: reflect.TypeFor[func([]byte) bool]()},
		{name: "validateASCIIWithErrors", typ: reflect.TypeFor[func([]byte) Result]()},
		{name: "validateUTF16LEAsASCII", typ: reflect.TypeFor[func([]uint16) bool]()},
		{name: "validateUTF16BEAsASCII", typ: reflect.TypeFor[func([]uint16) bool]()},
		{name: "validateUTF16LE", typ: reflect.TypeFor[func([]uint16) bool]()},
		{name: "validateUTF16BE", typ: reflect.TypeFor[func([]uint16) bool]()},
		{name: "validateUTF16LEWithErrors", typ: reflect.TypeFor[func([]uint16) Result]()},
		{name: "validateUTF16BEWithErrors", typ: reflect.TypeFor[func([]uint16) Result]()},
		{name: "toWellFormedUTF16LE", typ: reflect.TypeFor[func([]uint16, []uint16)]()},
		{name: "toWellFormedUTF16BE", typ: reflect.TypeFor[func([]uint16, []uint16)]()},
		{name: "validateUTF32", typ: reflect.TypeFor[func([]uint32) bool]()},
		{name: "validateUTF32WithErrors", typ: reflect.TypeFor[func([]uint32) Result]()},
	}
	if typ.NumField() != len(want) {
		t.Fatalf("implementation has %d fields, want exactly %d", typ.NumField(), len(want))
	}
	for i, field := range want {
		got := typ.Field(i)
		if got.Name != field.name || got.Type != field.typ {
			t.Errorf("field %d = %s %v, want %s %v", i, got.Name, got.Type, field.name, field.typ)
		}
	}
}

func TestLatin1LengthDispatchMatchesCountUTF8(t *testing.T) {
	for _, input := range []selectionInput{
		{},
		{features: ^cpuFeatures(0)},
		{features: ^cpuFeatures(0), archsimdAVX2: true},
	} {
		got := makeImplementation(input)
		if !sameFunction(got.latin1LengthFromUTF8, got.countUTF8) {
			t.Errorf("input %+v selected Latin-1 %p and CountUTF8 %p", input, got.latin1LengthFromUTF8, got.countUTF8)
		}
	}
}

func checkUTF8LengthImplementationFunctions(
	t *testing.T,
	got implementation,
	wantUTF16 func([]byte) int,
	wantUTF32 func([]byte) int,
) {
	t.Helper()
	if !sameFunction(got.latin1LengthFromUTF8, got.countUTF8) {
		t.Errorf("latin1LengthFromUTF8 selected %p, countUTF8 selected %p", got.latin1LengthFromUTF8, got.countUTF8)
	}
	if !sameFunction(got.utf16LengthFromUTF8, wantUTF16) {
		t.Errorf("utf16LengthFromUTF8 selected %p, want %p", got.utf16LengthFromUTF8, wantUTF16)
	}
	if !sameFunction(got.utf32LengthFromUTF8, wantUTF32) {
		t.Errorf("utf32LengthFromUTF8 selected %p, want %p", got.utf32LengthFromUTF8, wantUTF32)
	}
}

func checkUTF8ImplementationFunctions(t *testing.T, got implementation) {
	checkUTF8ImplementationFunctionsWant(t, got, validateUTF8Scalar, validateUTF8WithErrorsScalar)
}

func checkUTF8ImplementationFunctionsWant(
	t *testing.T,
	got implementation,
	wantValidate func([]byte) bool,
	wantWithErrors func([]byte) Result,
) {
	t.Helper()
	if !sameFunction(got.validateUTF8, wantValidate) {
		t.Errorf("validateUTF8 selected %x, want %x", reflect.ValueOf(got.validateUTF8).Pointer(), reflect.ValueOf(wantValidate).Pointer())
	}
	if !sameFunction(got.validateUTF8WithErrors, wantWithErrors) {
		t.Errorf("validateUTF8WithErrors selected %x, want %x", reflect.ValueOf(got.validateUTF8WithErrors).Pointer(), reflect.ValueOf(wantWithErrors).Pointer())
	}
}

func TestArchsimdProvidersAreAllOrNone(t *testing.T) {
	want := archsimdValidateASCII() != nil
	if got := archsimdProvidersAvailable(); got != want {
		t.Fatalf("archsimdProvidersAvailable() = %t, want %t", got, want)
	}
	if (archsimdValidateASCIIWithErrors() != nil) != want ||
		(archsimdValidateUTF16LEAsASCII() != nil) != want ||
		(archsimdValidateUTF16BEAsASCII() != nil) != want {
		t.Fatal("archsimd providers are not all available or all unavailable")
	}
}

func sameFunction[F any](got, want F) bool {
	return reflect.ValueOf(got).Pointer() == reflect.ValueOf(want).Pointer()
}

func checkImplementationFunctions(
	t *testing.T,
	got implementation,
	wantASCII func([]byte) bool,
	wantASCIIWithErrors func([]byte) Result,
	wantUTF16LE func([]uint16) bool,
	wantUTF16BE func([]uint16) bool,
) {
	t.Helper()
	if !sameFunction(got.validateASCII, wantASCII) {
		t.Errorf("validateASCII selected %x, want %x", reflect.ValueOf(got.validateASCII).Pointer(), reflect.ValueOf(wantASCII).Pointer())
	}
	if !sameFunction(got.validateASCIIWithErrors, wantASCIIWithErrors) {
		t.Errorf("validateASCIIWithErrors selected %x, want %x", reflect.ValueOf(got.validateASCIIWithErrors).Pointer(), reflect.ValueOf(wantASCIIWithErrors).Pointer())
	}
	if !sameFunction(got.validateUTF16LEAsASCII, wantUTF16LE) {
		t.Errorf("validateUTF16LEAsASCII selected %x, want %x", reflect.ValueOf(got.validateUTF16LEAsASCII).Pointer(), reflect.ValueOf(wantUTF16LE).Pointer())
	}
	if !sameFunction(got.validateUTF16BEAsASCII, wantUTF16BE) {
		t.Errorf("validateUTF16BEAsASCII selected %x, want %x", reflect.ValueOf(got.validateUTF16BEAsASCII).Pointer(), reflect.ValueOf(wantUTF16BE).Pointer())
	}
}
