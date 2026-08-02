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

//go:build arm64

package simdutf

import (
	"bytes"
	"encoding/binary"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Hand-authored Go-only direct differential coverage for the lookup4 assembly
// port pinned to simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/generic/utf8_validation/utf8_lookup4_algorithm.h:12-216 and
// src/generic/utf8_validation/utf8_validator.h:10-80.

func TestValidateUTF8NEONLookupTablesMatchPinnedUpstream(t *testing.T) {
	// Pinned table bytes from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
	// src/generic/utf8_validation/utf8_lookup4_algorithm.h:37-153.
	tables := []struct {
		name string
		want [16]byte
	}{
		{
			name: "utf8Lookup4Byte1HighNEON",
			want: [16]byte{
				0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
				0x80, 0x80, 0x80, 0x80, 0x21, 0x01, 0x15, 0x49,
			},
		},
		{
			name: "utf8Lookup4Byte1LowNEON",
			want: [16]byte{
				0xe7, 0xa3, 0x83, 0x83, 0x8b, 0xcb, 0xcb, 0xcb,
				0xcb, 0xcb, 0xcb, 0xcb, 0xcb, 0xdb, 0xcb, 0xcb,
			},
		},
		{
			name: "utf8Lookup4Byte2HighNEON",
			want: [16]byte{
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0xe6, 0xae, 0xba, 0xba, 0x01, 0x01, 0x01, 0x01,
			},
		},
		{
			name: "utf8Lookup4IncompleteMaxNEON",
			want: [16]byte{
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xef, 0xdf, 0xbf,
			},
		},
	}

	source, err := os.ReadFile("utf8_arm64.s")
	if err != nil {
		t.Fatal(err)
	}
	dataPattern := regexp.MustCompile(`^DATA ·([[:alnum:]]+)<>\+([0-9]+)\(SB\)/8, \$(0x[0-9a-fA-F]{16})$`)
	globlPattern := regexp.MustCompile(`^GLOBL ·([[:alnum:]]+)<>\(SB\), RODATA\|NOPTR, \$16$`)
	var dataRecords, globlRecords [][]string
	for lineNumber, line := range strings.Split(string(source), "\n") {
		switch {
		case strings.HasPrefix(line, "DATA "):
			match := dataPattern.FindStringSubmatch(line)
			if match == nil {
				t.Fatalf("utf8_arm64.s:%d: malformed DATA declaration %q", lineNumber+1, line)
			}
			dataRecords = append(dataRecords, match)
		case strings.HasPrefix(line, "GLOBL "):
			match := globlPattern.FindStringSubmatch(line)
			if match == nil {
				t.Fatalf("utf8_arm64.s:%d: malformed GLOBL declaration %q", lineNumber+1, line)
			}
			globlRecords = append(globlRecords, match)
		}
	}

	if got, want := len(dataRecords), len(tables)*2; got != want {
		t.Fatalf("DATA /8 declaration count = %d, want %d", got, want)
	}
	if got, want := len(globlRecords), len(tables); got != want {
		t.Fatalf("GLOBL RODATA|NOPTR, $16 declaration count = %d, want %d", got, want)
	}
	for tableIndex, table := range tables {
		var got [16]byte
		for chunk := range 2 {
			record := dataRecords[tableIndex*2+chunk]
			if record[1] != table.name {
				t.Fatalf("DATA declaration %d symbol = %q, want %q", tableIndex*2+chunk, record[1], table.name)
			}
			wantOffset := strconv.Itoa(chunk * 8)
			if record[2] != wantOffset {
				t.Fatalf("DATA declaration %d offset = %q, want %q", tableIndex*2+chunk, record[2], wantOffset)
			}
			literal, err := strconv.ParseUint(record[3], 0, 64)
			if err != nil {
				t.Fatalf("DATA declaration %d literal %q: %v", tableIndex*2+chunk, record[3], err)
			}
			binary.LittleEndian.PutUint64(got[chunk*8:], literal)
		}
		if !slices.Equal(got[:], table.want[:]) {
			t.Errorf("%s bytes = % x, want % x", table.name, got, table.want)
		}
		if gotName := globlRecords[tableIndex][1]; gotName != table.name {
			t.Errorf("GLOBL declaration %d symbol = %q, want exact declaration for %q", tableIndex, gotName, table.name)
		}
	}
}

func TestValidateUTF8NEONScalarParity(t *testing.T) {
	inputs := [][]byte{nil, {}}
	for _, length := range []int{15, 16, 17, 31, 32, 33, 63, 64, 65, 127, 128, 129} {
		inputs = append(inputs, bytes.Repeat([]byte{'a'}, length))
	}
	validSequences := [][]byte{{0xc2, 0x80}, {0xe0, 0xa0, 0x80}, {0xf0, 0x90, 0x80, 0x80}}
	for _, boundary := range []int{16, 32, 48, 64, 80, 128} {
		for _, sequence := range validSequences {
			for split := 1; split < len(sequence); split++ {
				start := boundary - split
				input := bytes.Repeat([]byte{'a'}, start)
				input = append(input, sequence...)
				input = append(input, bytes.Repeat([]byte{'b'}, 17)...)
				inputs = append(inputs, input)
			}
		}
	}
	invalid := [][]byte{
		{0x80}, {0xff}, {0xc0, 0x80}, {0xe0, 0x80, 0x80},
		{0xed, 0xa0, 0x80}, {0xf0, 0x80, 0x80, 0x80}, {0xf4, 0x90, 0x80, 0x80},
		{0xc2}, {0xe1, 0x80}, {0xf0, 0x90, 0x80},
	}
	for _, prefix := range []int{0, 19, 64, 81, 128} {
		for _, suffix := range invalid {
			input := bytes.Repeat([]byte{'a'}, prefix)
			inputs = append(inputs, append(input, suffix...))
		}
	}
	for i, input := range inputs {
		t.Run(strconv.Itoa(i)+"/length="+strconv.Itoa(len(input)), func(t *testing.T) {
			checkUTF8NEON(t, input)
		})
	}
}

func TestValidateUTF8NEONIncompleteAndNextBlockErrors(t *testing.T) {
	for _, position := range []int{63, 64, 127} {
		for _, lead := range []byte{0xc2, 0xe1, 0xf1} {
			input := bytes.Repeat([]byte{'a'}, position)
			input = append(input, lead)
			checkUTF8NEON(t, input)
		}
	}
	for _, position := range []int{63, 127} {
		input := bytes.Repeat([]byte{'a'}, position)
		input = append(input, 0xe1, 0x80, 'x')
		var remainder [64]byte
		copy(remainder[:], input[len(input)&^63:])
		count, hasError := validateUTF8Lookup4NEON(input, &remainder)
		wantCount := (position + 1) &^ 63
		if hasError == 0 || count != wantCount {
			t.Errorf("error at %d raw result = (%d, %#x), want observing block %d", position, count, hasError, wantCount)
		}
		checkUTF8NEON(t, input)
	}
}

func TestValidateUTF8NEONDoesNotWriteInput(t *testing.T) {
	backing := make([]byte, 131)
	backing[0], backing[len(backing)-1] = 0xa5, 0x5a
	for i := 1; i < len(backing)-1; i++ {
		backing[i] = byte(i & 0x7f)
	}
	backing[64] = 0xf0
	before := slices.Clone(backing)
	input := backing[1 : len(backing)-1]
	validateUTF8NEON(input)
	validateUTF8WithErrorsNEON(input)
	if !slices.Equal(backing, before) {
		t.Fatal("NEON UTF-8 validators modified input or canaries")
	}
}

func checkUTF8NEON(t *testing.T, input []byte) {
	t.Helper()
	if got, want := validateUTF8NEON(input), validateUTF8Scalar(input); got != want {
		t.Errorf("validateUTF8NEON = %t, scalar = %t for %x", got, want, input)
	}
	if got, want := validateUTF8WithErrorsNEON(input), validateUTF8WithErrorsScalar(input); got != want {
		t.Errorf("validateUTF8WithErrorsNEON = %+v, scalar = %+v for %x", got, want, input)
	}
}
