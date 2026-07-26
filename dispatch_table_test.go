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

// Hand-authored Go-only tests for the exact six-field implementation-table
// shape, selected function identities, and all-or-none archsimd providers. The
// dispatch contract is pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:src/implementation.cpp
// and .omx/plans/port-simdutf-dec3aad192f4-go.md section 5.5; these are not
// upstream test vectors.

func TestImplementationTableExactFields(t *testing.T) {
	typ := reflect.TypeFor[implementation]()
	want := []struct {
		name string
		typ  reflect.Type
	}{
		{name: "validateUTF8", typ: reflect.TypeFor[func([]byte) bool]()},
		{name: "validateUTF8WithErrors", typ: reflect.TypeFor[func([]byte) Result]()},
		{name: "validateASCII", typ: reflect.TypeFor[func([]byte) bool]()},
		{name: "validateASCIIWithErrors", typ: reflect.TypeFor[func([]byte) Result]()},
		{name: "validateUTF16LEAsASCII", typ: reflect.TypeFor[func([]uint16) bool]()},
		{name: "validateUTF16BEAsASCII", typ: reflect.TypeFor[func([]uint16) bool]()},
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

func checkUTF8ImplementationFunctions(t *testing.T, got implementation) {
	t.Helper()
	if !sameFunction(got.validateUTF8, validateUTF8Scalar) {
		t.Errorf("validateUTF8 selected %x, want scalar %x", reflect.ValueOf(got.validateUTF8).Pointer(), reflect.ValueOf(validateUTF8Scalar).Pointer())
	}
	if !sameFunction(got.validateUTF8WithErrors, validateUTF8WithErrorsScalar) {
		t.Errorf("validateUTF8WithErrors selected %x, want scalar %x", reflect.ValueOf(got.validateUTF8WithErrors).Pointer(), reflect.ValueOf(validateUTF8WithErrorsScalar).Pointer())
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
