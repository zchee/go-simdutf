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
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// Hand-authored Go-only public-dispatch qualification harness pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b and
// docs/porting/benchmark-contract.md. It adds no product behavior or upstream
// algorithm translation. Corpus setup, integrity checks, and dispatch-provider
// qualification stay outside timed b.Loop bodies.

const (
	dispatchQualificationZeroSHA256   = "ad7facb2586fc6e966c004d7d1d16b024f5805ff7cb47c7a85dabd8b48892ca7"
	dispatchQualificationOperationEnv = "SIMDUTF_BENCH_EXPECT_OPERATION"
	dispatchQualificationTierEnv      = "SIMDUTF_BENCH_EXPECT_TIER"
)

type dispatchQualificationRow struct {
	operation string
	corpus    string
	class     string
	size      int
	bytes     []byte
	uint16s   []uint16
}

func (row dispatchQualificationRow) name() string {
	return fmt.Sprintf("%s/%s/%s/%04d", row.operation, row.corpus, row.class, row.size)
}

func (row dispatchQualificationRow) inputBytes() int64 {
	if row.uint16s != nil {
		return int64(2 * len(row.uint16s))
	}
	return int64(len(row.bytes))
}

var dispatchQualificationByteSizes = [...]struct {
	size  int
	class string
}{
	{1, "short"}, {15, "short"}, {16, "short"}, {17, "short"},
	{31, "short"}, {32, "short"}, {33, "short"},
	{63, "boundary"}, {64, "boundary"}, {65, "boundary"},
	{127, "boundary"}, {128, "boundary"}, {129, "boundary"},
	{4096, "bulk"},
}

var dispatchQualificationUint16Sizes = [...]struct {
	size  int
	class string
}{
	{1, "short"}, {7, "short"}, {8, "short"}, {9, "short"},
	{15, "short"}, {16, "short"}, {17, "short"},
	{31, "boundary"}, {32, "boundary"}, {33, "boundary"},
	{63, "boundary"}, {64, "boundary"}, {65, "boundary"},
	{127, "boundary"}, {128, "boundary"}, {129, "boundary"},
	{2048, "bulk"},
}

func materializeDispatchQualificationCorpora() ([]byte, []byte, []uint16) {
	byteZero := make([]byte, 4096)
	uint16Raw := make([]byte, 4096)
	uint16Zero := make([]uint16, len(uint16Raw)/2)
	for i := range uint16Zero {
		uint16Zero[i] = binary.NativeEndian.Uint16(uint16Raw[2*i:])
	}
	return byteZero, uint16Raw, uint16Zero
}

func dispatchQualificationRows() []dispatchQualificationRow {
	byteZero, _, uint16Zero := materializeDispatchQualificationCorpora()
	rows := make([]dispatchQualificationRow, 0, 169)
	for _, operation := range [...]string{"ValidateASCII", "ValidateASCIIWithErrors"} {
		for _, input := range dispatchQualificationByteSizes {
			rows = append(rows, dispatchQualificationRow{
				operation: operation,
				corpus:    "Q-byte-zero",
				class:     input.class,
				size:      input.size,
				bytes:     byteZero[:input.size],
			})
		}
	}
	for _, operation := range [...]string{
		"ValidateUTF16LEAsASCII",
		"ValidateUTF16BEAsASCII",
		"ValidateUTF16AsASCII",
	} {
		for _, input := range dispatchQualificationUint16Sizes {
			rows = append(rows, dispatchQualificationRow{
				operation: operation,
				corpus:    "Q-u16-zero",
				class:     input.class,
				size:      input.size,
				uint16s:   uint16Zero[:input.size],
			})
		}
	}
	for _, operation := range [...]string{
		"ValidateUTF8",
		"ValidateUTF8WithErrors",
		"CountUTF8",
		"Latin1LengthFromUTF8",
		"UTF16LengthFromUTF8",
		"UTF32LengthFromUTF8",
	} {
		for _, input := range dispatchQualificationByteSizes {
			rows = append(rows, dispatchQualificationRow{
				operation: operation,
				corpus:    "Q-byte-zero",
				class:     input.class,
				size:      input.size,
				bytes:     byteZero[:input.size],
			})
		}
		rows = append(rows, dispatchQualificationRow{
			operation: operation,
			corpus:    "Q-emoji",
			class:     "bulk",
			size:      len(upstreamEmojiUTF8),
			bytes:     upstreamEmojiUTF8,
		})
	}
	return rows
}

var dispatchQualificationProviderIdentifiers = map[string]map[string][]string{
	"ValidateASCII": {
		"scalar":   {"validateASCIIScalar"},
		"westmere": {"validateASCIIWestmere"},
		"haswell":  {"validateASCIIHaswell"},
		"neon":     {"validateASCIINEON"},
		"archsimd": {"validateASCIIArchsimd"},
	},
	"ValidateASCIIWithErrors": {
		"scalar":   {"validateASCIIWithErrorsScalar"},
		"westmere": {"validateASCIIWithErrorsWestmere"},
		"haswell":  {"validateASCIIWithErrorsHaswell"},
		"neon":     {"validateASCIIWithErrorsNEON"},
		"archsimd": {"validateASCIIWithErrorsArchsimd"},
	},
	"ValidateUTF16LEAsASCII": {
		"scalar":   {"validateUTF16LEAsASCIIScalar"},
		"westmere": {"validateUTF16LEAsASCIIWestmere"},
		"haswell":  {"validateUTF16LEAsASCIIHaswell"},
		"neon":     {"validateUTF16LEAsASCIINEON"},
		"archsimd": {"validateUTF16LEAsASCIIArchsimd"},
	},
	"ValidateUTF16BEAsASCII": {
		"scalar":   {"validateUTF16BEAsASCIIScalar"},
		"westmere": {"validateUTF16BEAsASCIIWestmere"},
		"haswell":  {"validateUTF16BEAsASCIIHaswell"},
		"neon":     {"validateUTF16BEAsASCIINEON"},
		"archsimd": {"validateUTF16BEAsASCIIArchsimd"},
	},
	"ValidateUTF8": {
		"scalar":   {"validateUTF8Scalar"},
		"westmere": {"validateUTF8Westmere"},
		"haswell":  {"validateUTF8Haswell"},
		"neon":     {"validateUTF8NEON"},
		"archsimd": {"validateUTF8Archsimd"},
	},
	"ValidateUTF8WithErrors": {
		"scalar":   {"validateUTF8WithErrorsScalar"},
		"westmere": {"validateUTF8WithErrorsWestmere"},
		"haswell":  {"validateUTF8WithErrorsHaswell"},
		"neon":     {"validateUTF8WithErrorsNEON"},
		"archsimd": {"validateUTF8WithErrorsArchsimd"},
	},
	"CountUTF8": {
		"scalar":   {"countUTF8Scalar"},
		"westmere": {"countUTF8Westmere"},
		"haswell":  {"countUTF8Haswell"},
		"neon":     {"countUTF8NEON"},
		"archsimd": {"countUTF8Archsimd"},
	},
	"Latin1LengthFromUTF8": {
		"scalar":   {"countUTF8Scalar"},
		"westmere": {"countUTF8Westmere"},
		"haswell":  {"countUTF8Haswell"},
		"neon":     {"countUTF8NEON"},
		"archsimd": {"countUTF8Archsimd"},
	},
	"UTF16LengthFromUTF8": {
		"scalar":   {"utf16LengthFromUTF8Scalar"},
		"westmere": {"utf16LengthFromUTF8Westmere"},
		"haswell":  {"utf16LengthFromUTF8Haswell"},
		"neon":     {"utf16LengthFromUTF8NEON"},
		"archsimd": {"utf16LengthFromUTF8Archsimd"},
	},
	"UTF32LengthFromUTF8": {
		"scalar":   {"utf32LengthFromUTF8Scalar"},
		"westmere": {"utf32LengthFromUTF8Westmere"},
		"haswell":  {"utf32LengthFromUTF8Haswell"},
		"neon":     {"utf32LengthFromUTF8NEON"},
		"archsimd": {"utf32LengthFromUTF8Archsimd"},
	},
}

func dispatchQualificationGuard(operation string, fn any) error {
	expectedOperation, operationSet := os.LookupEnv(dispatchQualificationOperationEnv)
	expectedTier, tierSet := os.LookupEnv(dispatchQualificationTierEnv)
	return checkDispatchQualificationGuard(
		operation, expectedOperation, operationSet, expectedTier, tierSet, fn,
	)
}

func checkDispatchQualificationGuard(
	operation, expectedOperation string,
	operationSet bool,
	expectedTier string,
	tierSet bool,
	fn any,
) error {
	if !operationSet || !tierSet {
		return fmt.Errorf("%s and %s must both be set",
			dispatchQualificationOperationEnv, dispatchQualificationTierEnv)
	}
	if expectedOperation != operation {
		return fmt.Errorf("operation = %q, expected %q", operation, expectedOperation)
	}
	identifierOperation := operation
	if operation == "ValidateUTF16AsASCII" {
		if nativeLittleEndian() {
			identifierOperation = "ValidateUTF16LEAsASCII"
		} else {
			identifierOperation = "ValidateUTF16BEAsASCII"
		}
	}
	tiers, ok := dispatchQualificationProviderIdentifiers[identifierOperation]
	if !ok {
		return fmt.Errorf("operation %q has no provider allowlist", operation)
	}
	allowed, ok := tiers[expectedTier]
	if !ok {
		return fmt.Errorf("tier %q is not allowed for operation %q", expectedTier, operation)
	}
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func || value.IsNil() {
		return fmt.Errorf("operation %q has invalid dispatch function", operation)
	}
	runtimeFunction := runtime.FuncForPC(value.Pointer())
	if runtimeFunction == nil {
		return fmt.Errorf("operation %q dispatch function has no runtime symbol", operation)
	}
	name := runtimeFunction.Name()
	identifier := name[strings.LastIndexByte(name, '.')+1:]
	for _, candidate := range allowed {
		if identifier == candidate {
			return nil
		}
	}
	return fmt.Errorf("operation %q dispatches to %q, not tier %q allowlist %v",
		operation, identifier, expectedTier, allowed)
}

func dispatchQualificationFunction(operation string) any {
	switch operation {
	case "ValidateASCII":
		return activeImplementation.validateASCII
	case "ValidateASCIIWithErrors":
		return activeImplementation.validateASCIIWithErrors
	case "ValidateUTF16LEAsASCII":
		return activeImplementation.validateUTF16LEAsASCII
	case "ValidateUTF16BEAsASCII":
		return activeImplementation.validateUTF16BEAsASCII
	case "ValidateUTF16AsASCII":
		if nativeLittleEndian() {
			return activeImplementation.validateUTF16LEAsASCII
		}
		return activeImplementation.validateUTF16BEAsASCII
	case "ValidateUTF8":
		return activeImplementation.validateUTF8
	case "ValidateUTF8WithErrors":
		return activeImplementation.validateUTF8WithErrors
	case "CountUTF8":
		return activeImplementation.countUTF8
	case "Latin1LengthFromUTF8":
		return activeImplementation.latin1LengthFromUTF8
	case "UTF16LengthFromUTF8":
		return activeImplementation.utf16LengthFromUTF8
	case "UTF32LengthFromUTF8":
		return activeImplementation.utf32LengthFromUTF8
	default:
		panic("unknown dispatch qualification operation: " + operation)
	}
}

func BenchmarkDispatchQualification(b *testing.B) {
	for _, row := range dispatchQualificationRows() {
		b.Run(row.name(), func(b *testing.B) {
			if err := dispatchQualificationGuard(
				row.operation, dispatchQualificationFunction(row.operation),
			); err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(row.inputBytes())
			switch row.operation {
			case "ValidateASCII":
				for b.Loop() {
					benchmarkBoolSink = ValidateASCII(row.bytes)
				}
			case "ValidateASCIIWithErrors":
				for b.Loop() {
					benchmarkResultSink = ValidateASCIIWithErrors(row.bytes)
				}
			case "ValidateUTF16LEAsASCII":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF16LEAsASCII(row.uint16s)
				}
			case "ValidateUTF16BEAsASCII":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF16BEAsASCII(row.uint16s)
				}
			case "ValidateUTF16AsASCII":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF16AsASCII(row.uint16s)
				}
			case "ValidateUTF8":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF8(row.bytes)
				}
			case "ValidateUTF8WithErrors":
				for b.Loop() {
					benchmarkResultSink = ValidateUTF8WithErrors(row.bytes)
				}
			case "CountUTF8":
				for b.Loop() {
					benchmarkIntSink = CountUTF8(row.bytes)
				}
			case "Latin1LengthFromUTF8":
				for b.Loop() {
					benchmarkIntSink = Latin1LengthFromUTF8(row.bytes)
				}
			case "UTF16LengthFromUTF8":
				for b.Loop() {
					benchmarkIntSink = UTF16LengthFromUTF8(row.bytes)
				}
			case "UTF32LengthFromUTF8":
				for b.Loop() {
					benchmarkIntSink = UTF32LengthFromUTF8(row.bytes)
				}
			default:
				b.Fatalf("unknown operation %q", row.operation)
			}
		})
	}
}

const dispatchQualificationExpectedNames = `
ValidateASCII/Q-byte-zero/short/0001
ValidateASCII/Q-byte-zero/short/0015
ValidateASCII/Q-byte-zero/short/0016
ValidateASCII/Q-byte-zero/short/0017
ValidateASCII/Q-byte-zero/short/0031
ValidateASCII/Q-byte-zero/short/0032
ValidateASCII/Q-byte-zero/short/0033
ValidateASCII/Q-byte-zero/boundary/0063
ValidateASCII/Q-byte-zero/boundary/0064
ValidateASCII/Q-byte-zero/boundary/0065
ValidateASCII/Q-byte-zero/boundary/0127
ValidateASCII/Q-byte-zero/boundary/0128
ValidateASCII/Q-byte-zero/boundary/0129
ValidateASCII/Q-byte-zero/bulk/4096
ValidateASCIIWithErrors/Q-byte-zero/short/0001
ValidateASCIIWithErrors/Q-byte-zero/short/0015
ValidateASCIIWithErrors/Q-byte-zero/short/0016
ValidateASCIIWithErrors/Q-byte-zero/short/0017
ValidateASCIIWithErrors/Q-byte-zero/short/0031
ValidateASCIIWithErrors/Q-byte-zero/short/0032
ValidateASCIIWithErrors/Q-byte-zero/short/0033
ValidateASCIIWithErrors/Q-byte-zero/boundary/0063
ValidateASCIIWithErrors/Q-byte-zero/boundary/0064
ValidateASCIIWithErrors/Q-byte-zero/boundary/0065
ValidateASCIIWithErrors/Q-byte-zero/boundary/0127
ValidateASCIIWithErrors/Q-byte-zero/boundary/0128
ValidateASCIIWithErrors/Q-byte-zero/boundary/0129
ValidateASCIIWithErrors/Q-byte-zero/bulk/4096
ValidateUTF16LEAsASCII/Q-u16-zero/short/0001
ValidateUTF16LEAsASCII/Q-u16-zero/short/0007
ValidateUTF16LEAsASCII/Q-u16-zero/short/0008
ValidateUTF16LEAsASCII/Q-u16-zero/short/0009
ValidateUTF16LEAsASCII/Q-u16-zero/short/0015
ValidateUTF16LEAsASCII/Q-u16-zero/short/0016
ValidateUTF16LEAsASCII/Q-u16-zero/short/0017
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0031
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0032
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0033
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0063
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0064
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0065
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0127
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0128
ValidateUTF16LEAsASCII/Q-u16-zero/boundary/0129
ValidateUTF16LEAsASCII/Q-u16-zero/bulk/2048
ValidateUTF16BEAsASCII/Q-u16-zero/short/0001
ValidateUTF16BEAsASCII/Q-u16-zero/short/0007
ValidateUTF16BEAsASCII/Q-u16-zero/short/0008
ValidateUTF16BEAsASCII/Q-u16-zero/short/0009
ValidateUTF16BEAsASCII/Q-u16-zero/short/0015
ValidateUTF16BEAsASCII/Q-u16-zero/short/0016
ValidateUTF16BEAsASCII/Q-u16-zero/short/0017
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0031
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0032
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0033
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0063
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0064
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0065
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0127
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0128
ValidateUTF16BEAsASCII/Q-u16-zero/boundary/0129
ValidateUTF16BEAsASCII/Q-u16-zero/bulk/2048
ValidateUTF16AsASCII/Q-u16-zero/short/0001
ValidateUTF16AsASCII/Q-u16-zero/short/0007
ValidateUTF16AsASCII/Q-u16-zero/short/0008
ValidateUTF16AsASCII/Q-u16-zero/short/0009
ValidateUTF16AsASCII/Q-u16-zero/short/0015
ValidateUTF16AsASCII/Q-u16-zero/short/0016
ValidateUTF16AsASCII/Q-u16-zero/short/0017
ValidateUTF16AsASCII/Q-u16-zero/boundary/0031
ValidateUTF16AsASCII/Q-u16-zero/boundary/0032
ValidateUTF16AsASCII/Q-u16-zero/boundary/0033
ValidateUTF16AsASCII/Q-u16-zero/boundary/0063
ValidateUTF16AsASCII/Q-u16-zero/boundary/0064
ValidateUTF16AsASCII/Q-u16-zero/boundary/0065
ValidateUTF16AsASCII/Q-u16-zero/boundary/0127
ValidateUTF16AsASCII/Q-u16-zero/boundary/0128
ValidateUTF16AsASCII/Q-u16-zero/boundary/0129
ValidateUTF16AsASCII/Q-u16-zero/bulk/2048
ValidateUTF8/Q-byte-zero/short/0001
ValidateUTF8/Q-byte-zero/short/0015
ValidateUTF8/Q-byte-zero/short/0016
ValidateUTF8/Q-byte-zero/short/0017
ValidateUTF8/Q-byte-zero/short/0031
ValidateUTF8/Q-byte-zero/short/0032
ValidateUTF8/Q-byte-zero/short/0033
ValidateUTF8/Q-byte-zero/boundary/0063
ValidateUTF8/Q-byte-zero/boundary/0064
ValidateUTF8/Q-byte-zero/boundary/0065
ValidateUTF8/Q-byte-zero/boundary/0127
ValidateUTF8/Q-byte-zero/boundary/0128
ValidateUTF8/Q-byte-zero/boundary/0129
ValidateUTF8/Q-byte-zero/bulk/4096
ValidateUTF8/Q-emoji/bulk/3150
ValidateUTF8WithErrors/Q-byte-zero/short/0001
ValidateUTF8WithErrors/Q-byte-zero/short/0015
ValidateUTF8WithErrors/Q-byte-zero/short/0016
ValidateUTF8WithErrors/Q-byte-zero/short/0017
ValidateUTF8WithErrors/Q-byte-zero/short/0031
ValidateUTF8WithErrors/Q-byte-zero/short/0032
ValidateUTF8WithErrors/Q-byte-zero/short/0033
ValidateUTF8WithErrors/Q-byte-zero/boundary/0063
ValidateUTF8WithErrors/Q-byte-zero/boundary/0064
ValidateUTF8WithErrors/Q-byte-zero/boundary/0065
ValidateUTF8WithErrors/Q-byte-zero/boundary/0127
ValidateUTF8WithErrors/Q-byte-zero/boundary/0128
ValidateUTF8WithErrors/Q-byte-zero/boundary/0129
ValidateUTF8WithErrors/Q-byte-zero/bulk/4096
ValidateUTF8WithErrors/Q-emoji/bulk/3150
CountUTF8/Q-byte-zero/short/0001
CountUTF8/Q-byte-zero/short/0015
CountUTF8/Q-byte-zero/short/0016
CountUTF8/Q-byte-zero/short/0017
CountUTF8/Q-byte-zero/short/0031
CountUTF8/Q-byte-zero/short/0032
CountUTF8/Q-byte-zero/short/0033
CountUTF8/Q-byte-zero/boundary/0063
CountUTF8/Q-byte-zero/boundary/0064
CountUTF8/Q-byte-zero/boundary/0065
CountUTF8/Q-byte-zero/boundary/0127
CountUTF8/Q-byte-zero/boundary/0128
CountUTF8/Q-byte-zero/boundary/0129
CountUTF8/Q-byte-zero/bulk/4096
CountUTF8/Q-emoji/bulk/3150
Latin1LengthFromUTF8/Q-byte-zero/short/0001
Latin1LengthFromUTF8/Q-byte-zero/short/0015
Latin1LengthFromUTF8/Q-byte-zero/short/0016
Latin1LengthFromUTF8/Q-byte-zero/short/0017
Latin1LengthFromUTF8/Q-byte-zero/short/0031
Latin1LengthFromUTF8/Q-byte-zero/short/0032
Latin1LengthFromUTF8/Q-byte-zero/short/0033
Latin1LengthFromUTF8/Q-byte-zero/boundary/0063
Latin1LengthFromUTF8/Q-byte-zero/boundary/0064
Latin1LengthFromUTF8/Q-byte-zero/boundary/0065
Latin1LengthFromUTF8/Q-byte-zero/boundary/0127
Latin1LengthFromUTF8/Q-byte-zero/boundary/0128
Latin1LengthFromUTF8/Q-byte-zero/boundary/0129
Latin1LengthFromUTF8/Q-byte-zero/bulk/4096
Latin1LengthFromUTF8/Q-emoji/bulk/3150
UTF16LengthFromUTF8/Q-byte-zero/short/0001
UTF16LengthFromUTF8/Q-byte-zero/short/0015
UTF16LengthFromUTF8/Q-byte-zero/short/0016
UTF16LengthFromUTF8/Q-byte-zero/short/0017
UTF16LengthFromUTF8/Q-byte-zero/short/0031
UTF16LengthFromUTF8/Q-byte-zero/short/0032
UTF16LengthFromUTF8/Q-byte-zero/short/0033
UTF16LengthFromUTF8/Q-byte-zero/boundary/0063
UTF16LengthFromUTF8/Q-byte-zero/boundary/0064
UTF16LengthFromUTF8/Q-byte-zero/boundary/0065
UTF16LengthFromUTF8/Q-byte-zero/boundary/0127
UTF16LengthFromUTF8/Q-byte-zero/boundary/0128
UTF16LengthFromUTF8/Q-byte-zero/boundary/0129
UTF16LengthFromUTF8/Q-byte-zero/bulk/4096
UTF16LengthFromUTF8/Q-emoji/bulk/3150
UTF32LengthFromUTF8/Q-byte-zero/short/0001
UTF32LengthFromUTF8/Q-byte-zero/short/0015
UTF32LengthFromUTF8/Q-byte-zero/short/0016
UTF32LengthFromUTF8/Q-byte-zero/short/0017
UTF32LengthFromUTF8/Q-byte-zero/short/0031
UTF32LengthFromUTF8/Q-byte-zero/short/0032
UTF32LengthFromUTF8/Q-byte-zero/short/0033
UTF32LengthFromUTF8/Q-byte-zero/boundary/0063
UTF32LengthFromUTF8/Q-byte-zero/boundary/0064
UTF32LengthFromUTF8/Q-byte-zero/boundary/0065
UTF32LengthFromUTF8/Q-byte-zero/boundary/0127
UTF32LengthFromUTF8/Q-byte-zero/boundary/0128
UTF32LengthFromUTF8/Q-byte-zero/boundary/0129
UTF32LengthFromUTF8/Q-byte-zero/bulk/4096
UTF32LengthFromUTF8/Q-emoji/bulk/3150
`

func TestDispatchQualificationSurface(t *testing.T) {
	rows := dispatchQualificationRows()
	wantNames := strings.Fields(dispatchQualificationExpectedNames)
	if len(rows) != 169 || len(wantNames) != 169 {
		t.Fatalf("row counts = (%d, %d), want (169, 169)", len(rows), len(wantNames))
	}
	for i, row := range rows {
		if got, want := row.name(), wantNames[i]; got != want {
			t.Fatalf("row %d name = %q, want %q", i, got, want)
		}
	}
}

func TestDispatchQualificationInputs(t *testing.T) {
	byteZero, uint16Raw, uint16Zero := materializeDispatchQualificationCorpora()
	if got := fmt.Sprintf("%x", sha256.Sum256(byteZero)); got != dispatchQualificationZeroSHA256 {
		t.Fatalf("Q-byte-zero SHA-256 = %s, want %s", got, dispatchQualificationZeroSHA256)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(uint16Raw)); got != dispatchQualificationZeroSHA256 {
		t.Fatalf("Q-u16-zero raw SHA-256 = %s, want %s", got, dispatchQualificationZeroSHA256)
	}
	encoded := make([]byte, 2*len(uint16Zero))
	for i, codeUnit := range uint16Zero {
		binary.NativeEndian.PutUint16(encoded[2*i:], codeUnit)
	}
	if !reflect.DeepEqual(encoded, uint16Raw) {
		t.Fatal("Q-u16-zero is not the native-endian decoding of its raw bytes")
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(upstreamEmojiUTF8)); got != upstreamEmojiUTF8SHA256 {
		t.Fatalf("Q-emoji SHA-256 = %s, want %s", got, upstreamEmojiUTF8SHA256)
	}
	for i, row := range dispatchQualificationRows() {
		wantBytes := int64(row.size)
		if row.corpus == "Q-u16-zero" {
			wantBytes *= 2
		}
		if got := row.inputBytes(); got != wantBytes {
			t.Fatalf("row %d (%s) denominator = %d, want %d", i, row.name(), got, wantBytes)
		}
		if row.corpus == "Q-emoji" && row.size != 3150 {
			t.Fatalf("row %d (%s) input length = %d, want 3150", i, row.name(), row.size)
		}
	}
}

func TestDispatchQualificationGuard(t *testing.T) {
	tests := []struct {
		name              string
		operation         string
		expectedOperation string
		operationSet      bool
		tier              string
		tierSet           bool
		fn                any
		wantError         string
	}{
		{
			name:      "both environment values are required",
			operation: "ValidateASCII", fn: validateASCIIScalar,
			wantError: "must both be set",
		},
		{
			name:      "operation must match",
			operation: "ValidateASCII", expectedOperation: "CountUTF8",
			operationSet: true, tier: "scalar", tierSet: true,
			fn: validateASCIIScalar, wantError: "expected",
		},
		{
			name:      "tier must exist",
			operation: "ValidateASCII", expectedOperation: "ValidateASCII",
			operationSet: true, tier: "unknown", tierSet: true,
			fn: validateASCIIScalar, wantError: "is not allowed",
		},
		{
			name:      "provider must match tier",
			operation: "ValidateASCII", expectedOperation: "ValidateASCII",
			operationSet: true, tier: "neon", tierSet: true,
			fn: validateASCIIScalar, wantError: "not tier",
		},
		{
			name:      "scalar provider passes",
			operation: "ValidateASCII", expectedOperation: "ValidateASCII",
			operationSet: true, tier: "scalar", tierSet: true,
			fn: validateASCIIScalar,
		},
		{
			name:      "native UTF16 provider passes",
			operation: "ValidateUTF16AsASCII", expectedOperation: "ValidateUTF16AsASCII",
			operationSet: true, tier: "scalar", tierSet: true,
			fn: func() any {
				if nativeLittleEndian() {
					return validateUTF16LEAsASCIIScalar
				}
				return validateUTF16BEAsASCIIScalar
			}(),
		},
		{
			name:      "Latin1 uses CountUTF8 provider",
			operation: "Latin1LengthFromUTF8", expectedOperation: "Latin1LengthFromUTF8",
			operationSet: true, tier: "scalar", tierSet: true,
			fn: countUTF8Scalar,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := checkDispatchQualificationGuard(
				test.operation, test.expectedOperation, test.operationSet,
				test.tier, test.tierSet, test.fn,
			)
			if test.wantError == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("error = %v, want substring %q", err, test.wantError)
			}
		})
	}
}

func TestDispatchQualificationTimedCallsAreLiteral(t *testing.T) {
	source, err := os.ReadFile("dispatch_qualification_benchmark_test.go")
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "func BenchmarkDispatchQualification")
	end := strings.Index(string(source), "const dispatchQualificationExpectedNames")
	if start < 0 || end <= start {
		t.Fatalf("benchmark source boundaries = (%d, %d)", start, end)
	}
	timedSource := source[start:end]
	for _, call := range []string{
		"benchmarkBoolSink = ValidateASCII(row.bytes)",
		"benchmarkResultSink = ValidateASCIIWithErrors(row.bytes)",
		"benchmarkBoolSink = ValidateUTF16LEAsASCII(row.uint16s)",
		"benchmarkBoolSink = ValidateUTF16BEAsASCII(row.uint16s)",
		"benchmarkBoolSink = ValidateUTF16AsASCII(row.uint16s)",
		"benchmarkBoolSink = ValidateUTF8(row.bytes)",
		"benchmarkResultSink = ValidateUTF8WithErrors(row.bytes)",
		"benchmarkIntSink = CountUTF8(row.bytes)",
		"benchmarkIntSink = Latin1LengthFromUTF8(row.bytes)",
		"benchmarkIntSink = UTF16LengthFromUTF8(row.bytes)",
		"benchmarkIntSink = UTF32LengthFromUTF8(row.bytes)",
	} {
		if got := bytesCount(timedSource, []byte(call)); got != 1 {
			t.Errorf("literal timed call %q count = %d, want 1", call, got)
		}
	}
	if strings.Contains(string(timedSource), "activeImplementation.") {
		t.Error("timed qualification code uses activeImplementation instead of public wrappers")
	}
}

func bytesCount(contents, separator []byte) int {
	return strings.Count(string(contents), string(separator))
}
