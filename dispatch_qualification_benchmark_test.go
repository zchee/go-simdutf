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
	dispatchQualificationZeroSHA256         = "ad7facb2586fc6e966c004d7d1d16b024f5805ff7cb47c7a85dabd8b48892ca7"
	dispatchQualificationLatin1RampSHA256   = "c8f5d0341d54d951a71b136e6e2afcb14d11ed8489a7ae126a8fee0df6ecf193"
	dispatchQualificationArabicLipsumSHA256 = "b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb"
	dispatchQualificationArabicLipsumPath   = ".omx/artifacts/phase0/benchmark-corpora/corpus/unicode_lipsum/lipsum/Arabic-Lipsum.utf8.txt"
	dispatchQualificationArabicLipsumSize   = 81685
	dispatchQualificationOperationEnv       = "SIMDUTF_BENCH_EXPECT_OPERATION"
	dispatchQualificationTierEnv            = "SIMDUTF_BENCH_EXPECT_TIER"
)

type dispatchQualificationRow struct {
	operation string
	corpus    string
	class     string
	size      int
	bytes     []byte
	uint16s   []uint16
	uint32s   []uint32
}

func (row dispatchQualificationRow) name() string {
	return fmt.Sprintf("%s/%s/%s/%04d", row.operation, row.corpus, row.class, row.size)
}

func (row dispatchQualificationRow) inputBytes() int64 {
	if row.uint32s != nil {
		return int64(4 * len(row.uint32s))
	}
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

var dispatchQualificationUint32Sizes = [...]struct {
	size  int
	class string
}{
	{1, "short"}, {3, "short"}, {4, "short"}, {5, "short"},
	{7, "short"}, {8, "short"}, {9, "short"},
	{15, "boundary"}, {16, "boundary"}, {17, "boundary"},
	{31, "boundary"}, {32, "boundary"}, {33, "boundary"},
	{1024, "bulk"},
}

func materializeDispatchQualificationCorpora() ([]byte, []byte, []uint16, []byte, []uint32, []byte) {
	byteZero := make([]byte, 4096)
	latin1Ramp := make([]byte, 4096)
	for i := range latin1Ramp {
		latin1Ramp[i] = byte(i % 256)
	}
	uint16Raw := make([]byte, 4096)
	uint16Zero := make([]uint16, len(uint16Raw)/2)
	for i := range uint16Zero {
		uint16Zero[i] = binary.NativeEndian.Uint16(uint16Raw[2*i:])
	}
	uint32Raw := make([]byte, 4096)
	uint32Zero := make([]uint32, len(uint32Raw)/4)
	for i := range uint32Zero {
		uint32Zero[i] = binary.NativeEndian.Uint32(uint32Raw[4*i:])
	}
	return byteZero, uint16Raw, uint16Zero, uint32Raw, uint32Zero, latin1Ramp
}

func loadDispatchQualificationArabicLipsum() []byte {
	data, err := os.ReadFile(dispatchQualificationArabicLipsumPath)
	if err != nil {
		panic(fmt.Sprintf(
			"Q-arabic-lipsum missing at %s: %v",
			dispatchQualificationArabicLipsumPath, err,
		))
	}
	if len(data) != dispatchQualificationArabicLipsumSize {
		panic(fmt.Sprintf(
			"Q-arabic-lipsum length = %d, want %d",
			len(data), dispatchQualificationArabicLipsumSize,
		))
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != dispatchQualificationArabicLipsumSHA256 {
		panic(fmt.Sprintf(
			"Q-arabic-lipsum SHA-256 = %s, want %s",
			got, dispatchQualificationArabicLipsumSHA256,
		))
	}
	return data
}

func dispatchQualificationRows() []dispatchQualificationRow {
	byteZero, _, uint16Zero, _, uint32Zero, latin1Ramp := materializeDispatchQualificationCorpora()
	arabicLipsum := loadDispatchQualificationArabicLipsum()
	rows := make([]dispatchQualificationRow, 0, 561)
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
	for _, operation := range [...]string{
		"ValidateUTF16LE",
		"ValidateUTF16BE",
		"ValidateUTF16LEWithErrors",
		"ValidateUTF16BEWithErrors",
		"ToWellFormedUTF16LE",
		"ToWellFormedUTF16BE",
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
		"ValidateUTF32",
		"ValidateUTF32WithErrors",
	} {
		for _, input := range dispatchQualificationUint32Sizes {
			rows = append(rows, dispatchQualificationRow{
				operation: operation,
				corpus:    "Q-u32-zero",
				class:     input.class,
				size:      input.size,
				uint32s:   uint32Zero[:input.size],
			})
		}
	}
	for _, operation := range [...]string{
		"UTF8LengthFromLatin1",
		"ConvertLatin1ToUTF8",
		"ConvertLatin1ToUTF16LE",
		"ConvertLatin1ToUTF16BE",
		"ConvertLatin1ToUTF32",
	} {
		for _, input := range dispatchQualificationByteSizes {
			rows = append(rows, dispatchQualificationRow{
				operation: operation,
				corpus:    "Q-latin1-ramp",
				class:     input.class,
				size:      input.size,
				bytes:     latin1Ramp[:input.size],
			})
		}
	}
	for _, operation := range [...]string{
		"ConvertUTF8ToLatin1",
		"ConvertUTF8ToLatin1WithErrors",
		"ConvertValidUTF8ToLatin1",
		"ConvertUTF8ToUTF16LE",
		"ConvertUTF8ToUTF16BE",
		"ConvertUTF8ToUTF16LEWithErrors",
		"ConvertUTF8ToUTF16BEWithErrors",
		"ConvertValidUTF8ToUTF16LE",
		"ConvertValidUTF8ToUTF16BE",
		"ConvertUTF8ToUTF32",
		"ConvertUTF8ToUTF32WithErrors",
		"ConvertValidUTF8ToUTF32",
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
		rows = append(rows, dispatchQualificationRow{
			operation: operation,
			corpus:    "Q-arabic-lipsum",
			class:     "bulk",
			size:      len(arabicLipsum),
			bytes:     arabicLipsum,
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
	"ValidateUTF16LE": {
		"scalar":   {"validateUTF16LEScalar"},
		"westmere": {"validateUTF16LEWestmere"},
		"haswell":  {"validateUTF16LEHaswell"},
		"neon":     {"validateUTF16LENEON"},
		"archsimd": {"validateUTF16LEArchsimd"},
	},
	"ValidateUTF16BE": {
		"scalar":   {"validateUTF16BEScalar"},
		"westmere": {"validateUTF16BEWestmere"},
		"haswell":  {"validateUTF16BEHaswell"},
		"neon":     {"validateUTF16BENEON"},
		"archsimd": {"validateUTF16BEArchsimd"},
	},
	"ValidateUTF16LEWithErrors": {
		"scalar":   {"validateUTF16LEWithErrorsScalar"},
		"westmere": {"validateUTF16LEWithErrorsWestmere"},
		"haswell":  {"validateUTF16LEWithErrorsHaswell"},
		"neon":     {"validateUTF16LEWithErrorsNEON"},
		"archsimd": {"validateUTF16LEWithErrorsArchsimd"},
	},
	"ValidateUTF16BEWithErrors": {
		"scalar":   {"validateUTF16BEWithErrorsScalar"},
		"westmere": {"validateUTF16BEWithErrorsWestmere"},
		"haswell":  {"validateUTF16BEWithErrorsHaswell"},
		"neon":     {"validateUTF16BEWithErrorsNEON"},
		"archsimd": {"validateUTF16BEWithErrorsArchsimd"},
	},
	"ToWellFormedUTF16LE": {
		"scalar":   {"toWellFormedUTF16LEScalar"},
		"westmere": {"toWellFormedUTF16LEWestmere"},
		"haswell":  {"toWellFormedUTF16LEHaswell"},
		"neon":     {"toWellFormedUTF16LENEON"},
		"archsimd": {"toWellFormedUTF16LEArchsimd"},
	},
	"ToWellFormedUTF16BE": {
		"scalar":   {"toWellFormedUTF16BEScalar"},
		"westmere": {"toWellFormedUTF16BEWestmere"},
		"haswell":  {"toWellFormedUTF16BEHaswell"},
		"neon":     {"toWellFormedUTF16BENEON"},
		"archsimd": {"toWellFormedUTF16BEArchsimd"},
	},
	"ValidateUTF32": {
		"scalar":   {"validateUTF32Scalar"},
		"westmere": {"validateUTF32Westmere"},
		"haswell":  {"validateUTF32Haswell"},
		"neon":     {"validateUTF32NEON"},
		"archsimd": {"validateUTF32Archsimd"},
	},
	"ValidateUTF32WithErrors": {
		"scalar":   {"validateUTF32WithErrorsScalar"},
		"westmere": {"validateUTF32WithErrorsWestmere"},
		"haswell":  {"validateUTF32WithErrorsHaswell"},
		"neon":     {"validateUTF32WithErrorsNEON"},
		"archsimd": {"validateUTF32WithErrorsArchsimd"},
	},
	"UTF8LengthFromLatin1": {
		"scalar":   {"utf8LengthFromLatin1Scalar"},
		"westmere": {"utf8LengthFromLatin1Westmere"},
		"haswell":  {"utf8LengthFromLatin1Haswell"},
		"neon":     {"utf8LengthFromLatin1NEON"},
		"archsimd": {"utf8LengthFromLatin1Archsimd"},
	},
	"ConvertLatin1ToUTF8": {
		"scalar":   {"convertLatin1ToUTF8Scalar"},
		"westmere": {"convertLatin1ToUTF8Westmere"},
		"haswell":  {"convertLatin1ToUTF8Haswell"},
		"neon":     {"convertLatin1ToUTF8NEON"},
		"archsimd": {"convertLatin1ToUTF8Archsimd"},
	},
	"ConvertLatin1ToUTF16LE": {
		"scalar":   {"convertLatin1ToUTF16LEScalar"},
		"westmere": {"convertLatin1ToUTF16LEWestmere"},
		"haswell":  {"convertLatin1ToUTF16LEHaswell"},
		"neon":     {"convertLatin1ToUTF16LENEON"},
		"archsimd": {"convertLatin1ToUTF16LEArchsimd"},
	},
	"ConvertLatin1ToUTF16BE": {
		"scalar":   {"convertLatin1ToUTF16BEScalar"},
		"westmere": {"convertLatin1ToUTF16BEWestmere"},
		"haswell":  {"convertLatin1ToUTF16BEHaswell"},
		"neon":     {"convertLatin1ToUTF16BENEON"},
		"archsimd": {"convertLatin1ToUTF16BEArchsimd"},
	},
	"ConvertLatin1ToUTF32": {
		"scalar":   {"convertLatin1ToUTF32Scalar"},
		"westmere": {"convertLatin1ToUTF32Westmere"},
		"haswell":  {"convertLatin1ToUTF32Haswell"},
		"neon":     {"convertLatin1ToUTF32NEON"},
		"archsimd": {"convertLatin1ToUTF32Archsimd"},
	},
	"ConvertUTF8ToLatin1": {
		"scalar":   {"convertUTF8ToLatin1Scalar"},
		"westmere": {"convertUTF8ToLatin1Westmere"},
		"haswell":  {"convertUTF8ToLatin1Haswell"},
		"neon":     {"convertUTF8ToLatin1NEON"},
		"archsimd": {"convertUTF8ToLatin1Archsimd"},
	},
	"ConvertUTF8ToLatin1WithErrors": {
		"scalar":   {"convertUTF8ToLatin1WithErrorsScalar"},
		"westmere": {"convertUTF8ToLatin1WithErrorsWestmere"},
		"haswell":  {"convertUTF8ToLatin1WithErrorsHaswell"},
		"neon":     {"convertUTF8ToLatin1WithErrorsNEON"},
		"archsimd": {"convertUTF8ToLatin1WithErrorsArchsimd"},
	},
	"ConvertValidUTF8ToLatin1": {
		"scalar":   {"convertValidUTF8ToLatin1Scalar"},
		"westmere": {"convertValidUTF8ToLatin1Westmere"},
		"haswell":  {"convertValidUTF8ToLatin1Haswell"},
		"neon":     {"convertValidUTF8ToLatin1NEON"},
		"archsimd": {"convertValidUTF8ToLatin1Archsimd"},
	},
	"ConvertUTF8ToUTF16LE": {
		"scalar":   {"convertUTF8ToUTF16LEScalar"},
		"westmere": {"convertUTF8ToUTF16LEWestmere"},
		"haswell":  {"convertUTF8ToUTF16LEHaswell"},
		"neon":     {"convertUTF8ToUTF16LENEON"},
		"archsimd": {"convertUTF8ToUTF16LEArchsimd"},
	},
	"ConvertUTF8ToUTF16BE": {
		"scalar":   {"convertUTF8ToUTF16BEScalar"},
		"westmere": {"convertUTF8ToUTF16BEWestmere"},
		"haswell":  {"convertUTF8ToUTF16BEHaswell"},
		"neon":     {"convertUTF8ToUTF16BENEON"},
		"archsimd": {"convertUTF8ToUTF16BEArchsimd"},
	},
	"ConvertUTF8ToUTF16LEWithErrors": {
		"scalar":   {"convertUTF8ToUTF16LEWithErrorsScalar"},
		"westmere": {"convertUTF8ToUTF16LEWithErrorsWestmere"},
		"haswell":  {"convertUTF8ToUTF16LEWithErrorsHaswell"},
		"neon":     {"convertUTF8ToUTF16LEWithErrorsNEON"},
		"archsimd": {"convertUTF8ToUTF16LEWithErrorsArchsimd"},
	},
	"ConvertUTF8ToUTF16BEWithErrors": {
		"scalar":   {"convertUTF8ToUTF16BEWithErrorsScalar"},
		"westmere": {"convertUTF8ToUTF16BEWithErrorsWestmere"},
		"haswell":  {"convertUTF8ToUTF16BEWithErrorsHaswell"},
		"neon":     {"convertUTF8ToUTF16BEWithErrorsNEON"},
		"archsimd": {"convertUTF8ToUTF16BEWithErrorsArchsimd"},
	},
	"ConvertValidUTF8ToUTF16LE": {
		"scalar":   {"convertValidUTF8ToUTF16LEScalar"},
		"westmere": {"convertValidUTF8ToUTF16LEWestmere"},
		"haswell":  {"convertValidUTF8ToUTF16LEHaswell"},
		"neon":     {"convertValidUTF8ToUTF16LENEON"},
		"archsimd": {"convertValidUTF8ToUTF16LEArchsimd"},
	},
	"ConvertValidUTF8ToUTF16BE": {
		"scalar":   {"convertValidUTF8ToUTF16BEScalar"},
		"westmere": {"convertValidUTF8ToUTF16BEWestmere"},
		"haswell":  {"convertValidUTF8ToUTF16BEHaswell"},
		"neon":     {"convertValidUTF8ToUTF16BENEON"},
		"archsimd": {"convertValidUTF8ToUTF16BEArchsimd"},
	},
	"ConvertUTF8ToUTF32": {
		"scalar":   {"convertUTF8ToUTF32Scalar"},
		"westmere": {"convertUTF8ToUTF32Westmere"},
		"haswell":  {"convertUTF8ToUTF32Haswell"},
		"neon":     {"convertUTF8ToUTF32NEON"},
		"archsimd": {"convertUTF8ToUTF32Archsimd"},
	},
	"ConvertUTF8ToUTF32WithErrors": {
		"scalar":   {"convertUTF8ToUTF32WithErrorsScalar"},
		"westmere": {"convertUTF8ToUTF32WithErrorsWestmere"},
		"haswell":  {"convertUTF8ToUTF32WithErrorsHaswell"},
		"neon":     {"convertUTF8ToUTF32WithErrorsNEON"},
		"archsimd": {"convertUTF8ToUTF32WithErrorsArchsimd"},
	},
	"ConvertValidUTF8ToUTF32": {
		"scalar":   {"convertValidUTF8ToUTF32Scalar"},
		"westmere": {"convertValidUTF8ToUTF32Westmere"},
		"haswell":  {"convertValidUTF8ToUTF32Haswell"},
		"neon":     {"convertValidUTF8ToUTF32NEON"},
		"archsimd": {"convertValidUTF8ToUTF32Archsimd"},
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
	case "ValidateUTF16LE":
		return activeImplementation.validateUTF16LE
	case "ValidateUTF16BE":
		return activeImplementation.validateUTF16BE
	case "ValidateUTF16LEWithErrors":
		return activeImplementation.validateUTF16LEWithErrors
	case "ValidateUTF16BEWithErrors":
		return activeImplementation.validateUTF16BEWithErrors
	case "ToWellFormedUTF16LE":
		return activeImplementation.toWellFormedUTF16LE
	case "ToWellFormedUTF16BE":
		return activeImplementation.toWellFormedUTF16BE
	case "ValidateUTF32":
		return activeImplementation.validateUTF32
	case "ValidateUTF32WithErrors":
		return activeImplementation.validateUTF32WithErrors
	case "UTF8LengthFromLatin1":
		return activeImplementation.utf8LengthFromLatin1
	case "ConvertLatin1ToUTF8":
		return activeImplementation.convertLatin1ToUTF8
	case "ConvertLatin1ToUTF16LE":
		return activeImplementation.convertLatin1ToUTF16LE
	case "ConvertLatin1ToUTF16BE":
		return activeImplementation.convertLatin1ToUTF16BE
	case "ConvertLatin1ToUTF32":
		return activeImplementation.convertLatin1ToUTF32
	case "ConvertUTF8ToLatin1":
		return activeImplementation.convertUTF8ToLatin1
	case "ConvertUTF8ToLatin1WithErrors":
		return activeImplementation.convertUTF8ToLatin1WithErrors
	case "ConvertValidUTF8ToLatin1":
		return activeImplementation.convertValidUTF8ToLatin1
	case "ConvertUTF8ToUTF16LE":
		return activeImplementation.convertUTF8ToUTF16LE
	case "ConvertUTF8ToUTF16BE":
		return activeImplementation.convertUTF8ToUTF16BE
	case "ConvertUTF8ToUTF16LEWithErrors":
		return activeImplementation.convertUTF8ToUTF16LEWithErrors
	case "ConvertUTF8ToUTF16BEWithErrors":
		return activeImplementation.convertUTF8ToUTF16BEWithErrors
	case "ConvertValidUTF8ToUTF16LE":
		return activeImplementation.convertValidUTF8ToUTF16LE
	case "ConvertValidUTF8ToUTF16BE":
		return activeImplementation.convertValidUTF8ToUTF16BE
	case "ConvertUTF8ToUTF32":
		return activeImplementation.convertUTF8ToUTF32
	case "ConvertUTF8ToUTF32WithErrors":
		return activeImplementation.convertUTF8ToUTF32WithErrors
	case "ConvertValidUTF8ToUTF32":
		return activeImplementation.convertValidUTF8ToUTF32
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
			case "ValidateUTF16LE":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF16LE(row.uint16s)
				}
			case "ValidateUTF16BE":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF16BE(row.uint16s)
				}
			case "ValidateUTF16LEWithErrors":
				for b.Loop() {
					benchmarkResultSink = ValidateUTF16LEWithErrors(row.uint16s)
				}
			case "ValidateUTF16BEWithErrors":
				for b.Loop() {
					benchmarkResultSink = ValidateUTF16BEWithErrors(row.uint16s)
				}
			case "ToWellFormedUTF16LE":
				dst := make([]uint16, len(row.uint16s))
				for b.Loop() {
					ToWellFormedUTF16LE(row.uint16s, dst)
				}
			case "ToWellFormedUTF16BE":
				dst := make([]uint16, len(row.uint16s))
				for b.Loop() {
					ToWellFormedUTF16BE(row.uint16s, dst)
				}
			case "ValidateUTF32":
				for b.Loop() {
					benchmarkBoolSink = ValidateUTF32(row.uint32s)
				}
			case "ValidateUTF32WithErrors":
				for b.Loop() {
					benchmarkResultSink = ValidateUTF32WithErrors(row.uint32s)
				}
			case "UTF8LengthFromLatin1":
				for b.Loop() {
					benchmarkIntSink = UTF8LengthFromLatin1(row.bytes)
				}
			case "ConvertLatin1ToUTF8":
				dst := make([]byte, len(row.bytes)*2)
				for b.Loop() {
					benchmarkIntSink = ConvertLatin1ToUTF8(row.bytes, dst)
				}
			case "ConvertLatin1ToUTF16LE":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertLatin1ToUTF16LE(row.bytes, dst)
				}
			case "ConvertLatin1ToUTF16BE":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertLatin1ToUTF16BE(row.bytes, dst)
				}
			case "ConvertLatin1ToUTF32":
				dst := make([]uint32, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertLatin1ToUTF32(row.bytes, dst)
				}
			case "ConvertUTF8ToLatin1":
				dst := make([]byte, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF8ToLatin1(row.bytes, dst)
				}
			case "ConvertUTF8ToLatin1WithErrors":
				dst := make([]byte, len(row.bytes))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF8ToLatin1WithErrors(row.bytes, dst)
				}
			case "ConvertValidUTF8ToLatin1":
				dst := make([]byte, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF8ToLatin1(row.bytes, dst)
				}
			case "ConvertUTF8ToUTF16LE":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF8ToUTF16LE(row.bytes, dst)
				}
			case "ConvertUTF8ToUTF16BE":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF8ToUTF16BE(row.bytes, dst)
				}
			case "ConvertUTF8ToUTF16LEWithErrors":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF8ToUTF16LEWithErrors(row.bytes, dst)
				}
			case "ConvertUTF8ToUTF16BEWithErrors":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF8ToUTF16BEWithErrors(row.bytes, dst)
				}
			case "ConvertValidUTF8ToUTF16LE":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF8ToUTF16LE(row.bytes, dst)
				}
			case "ConvertValidUTF8ToUTF16BE":
				dst := make([]uint16, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF8ToUTF16BE(row.bytes, dst)
				}
			case "ConvertUTF8ToUTF32":
				dst := make([]uint32, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF8ToUTF32(row.bytes, dst)
				}
			case "ConvertUTF8ToUTF32WithErrors":
				dst := make([]uint32, len(row.bytes))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF8ToUTF32WithErrors(row.bytes, dst)
				}
			case "ConvertValidUTF8ToUTF32":
				dst := make([]uint32, len(row.bytes))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF8ToUTF32(row.bytes, dst)
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
ValidateUTF16LE/Q-u16-zero/short/0001
ValidateUTF16LE/Q-u16-zero/short/0007
ValidateUTF16LE/Q-u16-zero/short/0008
ValidateUTF16LE/Q-u16-zero/short/0009
ValidateUTF16LE/Q-u16-zero/short/0015
ValidateUTF16LE/Q-u16-zero/short/0016
ValidateUTF16LE/Q-u16-zero/short/0017
ValidateUTF16LE/Q-u16-zero/boundary/0031
ValidateUTF16LE/Q-u16-zero/boundary/0032
ValidateUTF16LE/Q-u16-zero/boundary/0033
ValidateUTF16LE/Q-u16-zero/boundary/0063
ValidateUTF16LE/Q-u16-zero/boundary/0064
ValidateUTF16LE/Q-u16-zero/boundary/0065
ValidateUTF16LE/Q-u16-zero/boundary/0127
ValidateUTF16LE/Q-u16-zero/boundary/0128
ValidateUTF16LE/Q-u16-zero/boundary/0129
ValidateUTF16LE/Q-u16-zero/bulk/2048
ValidateUTF16BE/Q-u16-zero/short/0001
ValidateUTF16BE/Q-u16-zero/short/0007
ValidateUTF16BE/Q-u16-zero/short/0008
ValidateUTF16BE/Q-u16-zero/short/0009
ValidateUTF16BE/Q-u16-zero/short/0015
ValidateUTF16BE/Q-u16-zero/short/0016
ValidateUTF16BE/Q-u16-zero/short/0017
ValidateUTF16BE/Q-u16-zero/boundary/0031
ValidateUTF16BE/Q-u16-zero/boundary/0032
ValidateUTF16BE/Q-u16-zero/boundary/0033
ValidateUTF16BE/Q-u16-zero/boundary/0063
ValidateUTF16BE/Q-u16-zero/boundary/0064
ValidateUTF16BE/Q-u16-zero/boundary/0065
ValidateUTF16BE/Q-u16-zero/boundary/0127
ValidateUTF16BE/Q-u16-zero/boundary/0128
ValidateUTF16BE/Q-u16-zero/boundary/0129
ValidateUTF16BE/Q-u16-zero/bulk/2048
ValidateUTF16LEWithErrors/Q-u16-zero/short/0001
ValidateUTF16LEWithErrors/Q-u16-zero/short/0007
ValidateUTF16LEWithErrors/Q-u16-zero/short/0008
ValidateUTF16LEWithErrors/Q-u16-zero/short/0009
ValidateUTF16LEWithErrors/Q-u16-zero/short/0015
ValidateUTF16LEWithErrors/Q-u16-zero/short/0016
ValidateUTF16LEWithErrors/Q-u16-zero/short/0017
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0031
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0032
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0033
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0063
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0064
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0065
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0127
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0128
ValidateUTF16LEWithErrors/Q-u16-zero/boundary/0129
ValidateUTF16LEWithErrors/Q-u16-zero/bulk/2048
ValidateUTF16BEWithErrors/Q-u16-zero/short/0001
ValidateUTF16BEWithErrors/Q-u16-zero/short/0007
ValidateUTF16BEWithErrors/Q-u16-zero/short/0008
ValidateUTF16BEWithErrors/Q-u16-zero/short/0009
ValidateUTF16BEWithErrors/Q-u16-zero/short/0015
ValidateUTF16BEWithErrors/Q-u16-zero/short/0016
ValidateUTF16BEWithErrors/Q-u16-zero/short/0017
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0031
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0032
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0033
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0063
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0064
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0065
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0127
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0128
ValidateUTF16BEWithErrors/Q-u16-zero/boundary/0129
ValidateUTF16BEWithErrors/Q-u16-zero/bulk/2048
ToWellFormedUTF16LE/Q-u16-zero/short/0001
ToWellFormedUTF16LE/Q-u16-zero/short/0007
ToWellFormedUTF16LE/Q-u16-zero/short/0008
ToWellFormedUTF16LE/Q-u16-zero/short/0009
ToWellFormedUTF16LE/Q-u16-zero/short/0015
ToWellFormedUTF16LE/Q-u16-zero/short/0016
ToWellFormedUTF16LE/Q-u16-zero/short/0017
ToWellFormedUTF16LE/Q-u16-zero/boundary/0031
ToWellFormedUTF16LE/Q-u16-zero/boundary/0032
ToWellFormedUTF16LE/Q-u16-zero/boundary/0033
ToWellFormedUTF16LE/Q-u16-zero/boundary/0063
ToWellFormedUTF16LE/Q-u16-zero/boundary/0064
ToWellFormedUTF16LE/Q-u16-zero/boundary/0065
ToWellFormedUTF16LE/Q-u16-zero/boundary/0127
ToWellFormedUTF16LE/Q-u16-zero/boundary/0128
ToWellFormedUTF16LE/Q-u16-zero/boundary/0129
ToWellFormedUTF16LE/Q-u16-zero/bulk/2048
ToWellFormedUTF16BE/Q-u16-zero/short/0001
ToWellFormedUTF16BE/Q-u16-zero/short/0007
ToWellFormedUTF16BE/Q-u16-zero/short/0008
ToWellFormedUTF16BE/Q-u16-zero/short/0009
ToWellFormedUTF16BE/Q-u16-zero/short/0015
ToWellFormedUTF16BE/Q-u16-zero/short/0016
ToWellFormedUTF16BE/Q-u16-zero/short/0017
ToWellFormedUTF16BE/Q-u16-zero/boundary/0031
ToWellFormedUTF16BE/Q-u16-zero/boundary/0032
ToWellFormedUTF16BE/Q-u16-zero/boundary/0033
ToWellFormedUTF16BE/Q-u16-zero/boundary/0063
ToWellFormedUTF16BE/Q-u16-zero/boundary/0064
ToWellFormedUTF16BE/Q-u16-zero/boundary/0065
ToWellFormedUTF16BE/Q-u16-zero/boundary/0127
ToWellFormedUTF16BE/Q-u16-zero/boundary/0128
ToWellFormedUTF16BE/Q-u16-zero/boundary/0129
ToWellFormedUTF16BE/Q-u16-zero/bulk/2048
ValidateUTF32/Q-u32-zero/short/0001
ValidateUTF32/Q-u32-zero/short/0003
ValidateUTF32/Q-u32-zero/short/0004
ValidateUTF32/Q-u32-zero/short/0005
ValidateUTF32/Q-u32-zero/short/0007
ValidateUTF32/Q-u32-zero/short/0008
ValidateUTF32/Q-u32-zero/short/0009
ValidateUTF32/Q-u32-zero/boundary/0015
ValidateUTF32/Q-u32-zero/boundary/0016
ValidateUTF32/Q-u32-zero/boundary/0017
ValidateUTF32/Q-u32-zero/boundary/0031
ValidateUTF32/Q-u32-zero/boundary/0032
ValidateUTF32/Q-u32-zero/boundary/0033
ValidateUTF32/Q-u32-zero/bulk/1024
ValidateUTF32WithErrors/Q-u32-zero/short/0001
ValidateUTF32WithErrors/Q-u32-zero/short/0003
ValidateUTF32WithErrors/Q-u32-zero/short/0004
ValidateUTF32WithErrors/Q-u32-zero/short/0005
ValidateUTF32WithErrors/Q-u32-zero/short/0007
ValidateUTF32WithErrors/Q-u32-zero/short/0008
ValidateUTF32WithErrors/Q-u32-zero/short/0009
ValidateUTF32WithErrors/Q-u32-zero/boundary/0015
ValidateUTF32WithErrors/Q-u32-zero/boundary/0016
ValidateUTF32WithErrors/Q-u32-zero/boundary/0017
ValidateUTF32WithErrors/Q-u32-zero/boundary/0031
ValidateUTF32WithErrors/Q-u32-zero/boundary/0032
ValidateUTF32WithErrors/Q-u32-zero/boundary/0033
ValidateUTF32WithErrors/Q-u32-zero/bulk/1024
UTF8LengthFromLatin1/Q-latin1-ramp/short/0001
UTF8LengthFromLatin1/Q-latin1-ramp/short/0015
UTF8LengthFromLatin1/Q-latin1-ramp/short/0016
UTF8LengthFromLatin1/Q-latin1-ramp/short/0017
UTF8LengthFromLatin1/Q-latin1-ramp/short/0031
UTF8LengthFromLatin1/Q-latin1-ramp/short/0032
UTF8LengthFromLatin1/Q-latin1-ramp/short/0033
UTF8LengthFromLatin1/Q-latin1-ramp/boundary/0063
UTF8LengthFromLatin1/Q-latin1-ramp/boundary/0064
UTF8LengthFromLatin1/Q-latin1-ramp/boundary/0065
UTF8LengthFromLatin1/Q-latin1-ramp/boundary/0127
UTF8LengthFromLatin1/Q-latin1-ramp/boundary/0128
UTF8LengthFromLatin1/Q-latin1-ramp/boundary/0129
UTF8LengthFromLatin1/Q-latin1-ramp/bulk/4096
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0001
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0015
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0016
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0017
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0031
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0032
ConvertLatin1ToUTF8/Q-latin1-ramp/short/0033
ConvertLatin1ToUTF8/Q-latin1-ramp/boundary/0063
ConvertLatin1ToUTF8/Q-latin1-ramp/boundary/0064
ConvertLatin1ToUTF8/Q-latin1-ramp/boundary/0065
ConvertLatin1ToUTF8/Q-latin1-ramp/boundary/0127
ConvertLatin1ToUTF8/Q-latin1-ramp/boundary/0128
ConvertLatin1ToUTF8/Q-latin1-ramp/boundary/0129
ConvertLatin1ToUTF8/Q-latin1-ramp/bulk/4096
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0001
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0015
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0016
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0017
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0031
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0032
ConvertLatin1ToUTF16LE/Q-latin1-ramp/short/0033
ConvertLatin1ToUTF16LE/Q-latin1-ramp/boundary/0063
ConvertLatin1ToUTF16LE/Q-latin1-ramp/boundary/0064
ConvertLatin1ToUTF16LE/Q-latin1-ramp/boundary/0065
ConvertLatin1ToUTF16LE/Q-latin1-ramp/boundary/0127
ConvertLatin1ToUTF16LE/Q-latin1-ramp/boundary/0128
ConvertLatin1ToUTF16LE/Q-latin1-ramp/boundary/0129
ConvertLatin1ToUTF16LE/Q-latin1-ramp/bulk/4096
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0001
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0015
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0016
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0017
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0031
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0032
ConvertLatin1ToUTF16BE/Q-latin1-ramp/short/0033
ConvertLatin1ToUTF16BE/Q-latin1-ramp/boundary/0063
ConvertLatin1ToUTF16BE/Q-latin1-ramp/boundary/0064
ConvertLatin1ToUTF16BE/Q-latin1-ramp/boundary/0065
ConvertLatin1ToUTF16BE/Q-latin1-ramp/boundary/0127
ConvertLatin1ToUTF16BE/Q-latin1-ramp/boundary/0128
ConvertLatin1ToUTF16BE/Q-latin1-ramp/boundary/0129
ConvertLatin1ToUTF16BE/Q-latin1-ramp/bulk/4096
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0001
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0015
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0016
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0017
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0031
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0032
ConvertLatin1ToUTF32/Q-latin1-ramp/short/0033
ConvertLatin1ToUTF32/Q-latin1-ramp/boundary/0063
ConvertLatin1ToUTF32/Q-latin1-ramp/boundary/0064
ConvertLatin1ToUTF32/Q-latin1-ramp/boundary/0065
ConvertLatin1ToUTF32/Q-latin1-ramp/boundary/0127
ConvertLatin1ToUTF32/Q-latin1-ramp/boundary/0128
ConvertLatin1ToUTF32/Q-latin1-ramp/boundary/0129
ConvertLatin1ToUTF32/Q-latin1-ramp/bulk/4096
ConvertUTF8ToLatin1/Q-byte-zero/short/0001
ConvertUTF8ToLatin1/Q-byte-zero/short/0015
ConvertUTF8ToLatin1/Q-byte-zero/short/0016
ConvertUTF8ToLatin1/Q-byte-zero/short/0017
ConvertUTF8ToLatin1/Q-byte-zero/short/0031
ConvertUTF8ToLatin1/Q-byte-zero/short/0032
ConvertUTF8ToLatin1/Q-byte-zero/short/0033
ConvertUTF8ToLatin1/Q-byte-zero/boundary/0063
ConvertUTF8ToLatin1/Q-byte-zero/boundary/0064
ConvertUTF8ToLatin1/Q-byte-zero/boundary/0065
ConvertUTF8ToLatin1/Q-byte-zero/boundary/0127
ConvertUTF8ToLatin1/Q-byte-zero/boundary/0128
ConvertUTF8ToLatin1/Q-byte-zero/boundary/0129
ConvertUTF8ToLatin1/Q-byte-zero/bulk/4096
ConvertUTF8ToLatin1/Q-emoji/bulk/3150
ConvertUTF8ToLatin1/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0001
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0015
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0016
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0017
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0031
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0032
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/short/0033
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/boundary/0063
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/boundary/0064
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/boundary/0065
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/boundary/0127
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/boundary/0128
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/boundary/0129
ConvertUTF8ToLatin1WithErrors/Q-byte-zero/bulk/4096
ConvertUTF8ToLatin1WithErrors/Q-emoji/bulk/3150
ConvertUTF8ToLatin1WithErrors/Q-arabic-lipsum/bulk/81685
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0001
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0015
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0016
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0017
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0031
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0032
ConvertValidUTF8ToLatin1/Q-byte-zero/short/0033
ConvertValidUTF8ToLatin1/Q-byte-zero/boundary/0063
ConvertValidUTF8ToLatin1/Q-byte-zero/boundary/0064
ConvertValidUTF8ToLatin1/Q-byte-zero/boundary/0065
ConvertValidUTF8ToLatin1/Q-byte-zero/boundary/0127
ConvertValidUTF8ToLatin1/Q-byte-zero/boundary/0128
ConvertValidUTF8ToLatin1/Q-byte-zero/boundary/0129
ConvertValidUTF8ToLatin1/Q-byte-zero/bulk/4096
ConvertValidUTF8ToLatin1/Q-emoji/bulk/3150
ConvertValidUTF8ToLatin1/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0001
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0015
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0016
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0017
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0031
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0032
ConvertUTF8ToUTF16LE/Q-byte-zero/short/0033
ConvertUTF8ToUTF16LE/Q-byte-zero/boundary/0063
ConvertUTF8ToUTF16LE/Q-byte-zero/boundary/0064
ConvertUTF8ToUTF16LE/Q-byte-zero/boundary/0065
ConvertUTF8ToUTF16LE/Q-byte-zero/boundary/0127
ConvertUTF8ToUTF16LE/Q-byte-zero/boundary/0128
ConvertUTF8ToUTF16LE/Q-byte-zero/boundary/0129
ConvertUTF8ToUTF16LE/Q-byte-zero/bulk/4096
ConvertUTF8ToUTF16LE/Q-emoji/bulk/3150
ConvertUTF8ToUTF16LE/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0001
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0015
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0016
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0017
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0031
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0032
ConvertUTF8ToUTF16BE/Q-byte-zero/short/0033
ConvertUTF8ToUTF16BE/Q-byte-zero/boundary/0063
ConvertUTF8ToUTF16BE/Q-byte-zero/boundary/0064
ConvertUTF8ToUTF16BE/Q-byte-zero/boundary/0065
ConvertUTF8ToUTF16BE/Q-byte-zero/boundary/0127
ConvertUTF8ToUTF16BE/Q-byte-zero/boundary/0128
ConvertUTF8ToUTF16BE/Q-byte-zero/boundary/0129
ConvertUTF8ToUTF16BE/Q-byte-zero/bulk/4096
ConvertUTF8ToUTF16BE/Q-emoji/bulk/3150
ConvertUTF8ToUTF16BE/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0001
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0015
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0016
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0017
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0031
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0032
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/short/0033
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/boundary/0063
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/boundary/0064
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/boundary/0065
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/boundary/0127
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/boundary/0128
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/boundary/0129
ConvertUTF8ToUTF16LEWithErrors/Q-byte-zero/bulk/4096
ConvertUTF8ToUTF16LEWithErrors/Q-emoji/bulk/3150
ConvertUTF8ToUTF16LEWithErrors/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0001
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0015
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0016
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0017
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0031
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0032
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/short/0033
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/boundary/0063
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/boundary/0064
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/boundary/0065
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/boundary/0127
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/boundary/0128
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/boundary/0129
ConvertUTF8ToUTF16BEWithErrors/Q-byte-zero/bulk/4096
ConvertUTF8ToUTF16BEWithErrors/Q-emoji/bulk/3150
ConvertUTF8ToUTF16BEWithErrors/Q-arabic-lipsum/bulk/81685
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0001
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0015
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0016
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0017
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0031
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0032
ConvertValidUTF8ToUTF16LE/Q-byte-zero/short/0033
ConvertValidUTF8ToUTF16LE/Q-byte-zero/boundary/0063
ConvertValidUTF8ToUTF16LE/Q-byte-zero/boundary/0064
ConvertValidUTF8ToUTF16LE/Q-byte-zero/boundary/0065
ConvertValidUTF8ToUTF16LE/Q-byte-zero/boundary/0127
ConvertValidUTF8ToUTF16LE/Q-byte-zero/boundary/0128
ConvertValidUTF8ToUTF16LE/Q-byte-zero/boundary/0129
ConvertValidUTF8ToUTF16LE/Q-byte-zero/bulk/4096
ConvertValidUTF8ToUTF16LE/Q-emoji/bulk/3150
ConvertValidUTF8ToUTF16LE/Q-arabic-lipsum/bulk/81685
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0001
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0015
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0016
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0017
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0031
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0032
ConvertValidUTF8ToUTF16BE/Q-byte-zero/short/0033
ConvertValidUTF8ToUTF16BE/Q-byte-zero/boundary/0063
ConvertValidUTF8ToUTF16BE/Q-byte-zero/boundary/0064
ConvertValidUTF8ToUTF16BE/Q-byte-zero/boundary/0065
ConvertValidUTF8ToUTF16BE/Q-byte-zero/boundary/0127
ConvertValidUTF8ToUTF16BE/Q-byte-zero/boundary/0128
ConvertValidUTF8ToUTF16BE/Q-byte-zero/boundary/0129
ConvertValidUTF8ToUTF16BE/Q-byte-zero/bulk/4096
ConvertValidUTF8ToUTF16BE/Q-emoji/bulk/3150
ConvertValidUTF8ToUTF16BE/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToUTF32/Q-byte-zero/short/0001
ConvertUTF8ToUTF32/Q-byte-zero/short/0015
ConvertUTF8ToUTF32/Q-byte-zero/short/0016
ConvertUTF8ToUTF32/Q-byte-zero/short/0017
ConvertUTF8ToUTF32/Q-byte-zero/short/0031
ConvertUTF8ToUTF32/Q-byte-zero/short/0032
ConvertUTF8ToUTF32/Q-byte-zero/short/0033
ConvertUTF8ToUTF32/Q-byte-zero/boundary/0063
ConvertUTF8ToUTF32/Q-byte-zero/boundary/0064
ConvertUTF8ToUTF32/Q-byte-zero/boundary/0065
ConvertUTF8ToUTF32/Q-byte-zero/boundary/0127
ConvertUTF8ToUTF32/Q-byte-zero/boundary/0128
ConvertUTF8ToUTF32/Q-byte-zero/boundary/0129
ConvertUTF8ToUTF32/Q-byte-zero/bulk/4096
ConvertUTF8ToUTF32/Q-emoji/bulk/3150
ConvertUTF8ToUTF32/Q-arabic-lipsum/bulk/81685
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0001
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0015
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0016
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0017
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0031
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0032
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/short/0033
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/boundary/0063
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/boundary/0064
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/boundary/0065
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/boundary/0127
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/boundary/0128
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/boundary/0129
ConvertUTF8ToUTF32WithErrors/Q-byte-zero/bulk/4096
ConvertUTF8ToUTF32WithErrors/Q-emoji/bulk/3150
ConvertUTF8ToUTF32WithErrors/Q-arabic-lipsum/bulk/81685
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0001
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0015
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0016
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0017
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0031
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0032
ConvertValidUTF8ToUTF32/Q-byte-zero/short/0033
ConvertValidUTF8ToUTF32/Q-byte-zero/boundary/0063
ConvertValidUTF8ToUTF32/Q-byte-zero/boundary/0064
ConvertValidUTF8ToUTF32/Q-byte-zero/boundary/0065
ConvertValidUTF8ToUTF32/Q-byte-zero/boundary/0127
ConvertValidUTF8ToUTF32/Q-byte-zero/boundary/0128
ConvertValidUTF8ToUTF32/Q-byte-zero/boundary/0129
ConvertValidUTF8ToUTF32/Q-byte-zero/bulk/4096
ConvertValidUTF8ToUTF32/Q-emoji/bulk/3150
ConvertValidUTF8ToUTF32/Q-arabic-lipsum/bulk/81685
`

func TestDispatchQualificationSurface(t *testing.T) {
	rows := dispatchQualificationRows()
	wantNames := strings.Fields(dispatchQualificationExpectedNames)
	if len(rows) != 561 || len(wantNames) != 561 {
		t.Fatalf("row counts = (%d, %d), want (561, 561)", len(rows), len(wantNames))
	}
	for i, row := range rows {
		if got, want := row.name(), wantNames[i]; got != want {
			t.Fatalf("row %d name = %q, want %q", i, got, want)
		}
	}
}

func TestDispatchQualificationInputs(t *testing.T) {
	byteZero, uint16Raw, uint16Zero, uint32Raw, uint32Zero, latin1Ramp := materializeDispatchQualificationCorpora()
	if got := fmt.Sprintf("%x", sha256.Sum256(byteZero)); got != dispatchQualificationZeroSHA256 {
		t.Fatalf("Q-byte-zero SHA-256 = %s, want %s", got, dispatchQualificationZeroSHA256)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(latin1Ramp)); got != dispatchQualificationLatin1RampSHA256 {
		t.Fatalf("Q-latin1-ramp SHA-256 = %s, want %s", got, dispatchQualificationLatin1RampSHA256)
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
	if got := fmt.Sprintf("%x", sha256.Sum256(uint32Raw)); got != dispatchQualificationZeroSHA256 {
		t.Fatalf("Q-u32-zero raw SHA-256 = %s, want %s", got, dispatchQualificationZeroSHA256)
	}
	encoded32 := make([]byte, 4*len(uint32Zero))
	for i, codeUnit := range uint32Zero {
		binary.NativeEndian.PutUint32(encoded32[4*i:], codeUnit)
	}
	if !reflect.DeepEqual(encoded32, uint32Raw) {
		t.Fatal("Q-u32-zero is not the native-endian decoding of its raw bytes")
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(upstreamEmojiUTF8)); got != upstreamEmojiUTF8SHA256 {
		t.Fatalf("Q-emoji SHA-256 = %s, want %s", got, upstreamEmojiUTF8SHA256)
	}
	arabicLipsum := loadDispatchQualificationArabicLipsum()
	if got := fmt.Sprintf("%x", sha256.Sum256(arabicLipsum)); got != dispatchQualificationArabicLipsumSHA256 {
		t.Fatalf("Q-arabic-lipsum SHA-256 = %s, want %s", got, dispatchQualificationArabicLipsumSHA256)
	}
	if len(arabicLipsum) != dispatchQualificationArabicLipsumSize {
		t.Fatalf("Q-arabic-lipsum length = %d, want %d", len(arabicLipsum), dispatchQualificationArabicLipsumSize)
	}
	for i, row := range dispatchQualificationRows() {
		wantBytes := int64(row.size)
		if row.corpus == "Q-u16-zero" {
			wantBytes *= 2
		}
		if row.corpus == "Q-u32-zero" {
			wantBytes *= 4
		}
		if got := row.inputBytes(); got != wantBytes {
			t.Fatalf("row %d (%s) denominator = %d, want %d", i, row.name(), got, wantBytes)
		}
		if row.corpus == "Q-emoji" && row.size != 3150 {
			t.Fatalf("row %d (%s) input length = %d, want 3150", i, row.name(), row.size)
		}
		if row.corpus == "Q-arabic-lipsum" && row.size != dispatchQualificationArabicLipsumSize {
			t.Fatalf("row %d (%s) input length = %d, want %d", i, row.name(), row.size, dispatchQualificationArabicLipsumSize)
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
		"benchmarkBoolSink = ValidateUTF16LE(row.uint16s)",
		"benchmarkBoolSink = ValidateUTF16BE(row.uint16s)",
		"benchmarkResultSink = ValidateUTF16LEWithErrors(row.uint16s)",
		"benchmarkResultSink = ValidateUTF16BEWithErrors(row.uint16s)",
		"ToWellFormedUTF16LE(row.uint16s, dst)",
		"ToWellFormedUTF16BE(row.uint16s, dst)",
		"benchmarkBoolSink = ValidateUTF32(row.uint32s)",
		"benchmarkResultSink = ValidateUTF32WithErrors(row.uint32s)",
		"benchmarkIntSink = UTF8LengthFromLatin1(row.bytes)",
		"benchmarkIntSink = ConvertLatin1ToUTF8(row.bytes, dst)",
		"benchmarkIntSink = ConvertLatin1ToUTF16LE(row.bytes, dst)",
		"benchmarkIntSink = ConvertLatin1ToUTF16BE(row.bytes, dst)",
		"benchmarkIntSink = ConvertLatin1ToUTF32(row.bytes, dst)",
		"benchmarkIntSink = ConvertUTF8ToLatin1(row.bytes, dst)",
		"benchmarkResultSink = ConvertUTF8ToLatin1WithErrors(row.bytes, dst)",
		"benchmarkIntSink = ConvertValidUTF8ToLatin1(row.bytes, dst)",
		"benchmarkIntSink = ConvertUTF8ToUTF16LE(row.bytes, dst)",
		"benchmarkIntSink = ConvertUTF8ToUTF16BE(row.bytes, dst)",
		"benchmarkResultSink = ConvertUTF8ToUTF16LEWithErrors(row.bytes, dst)",
		"benchmarkResultSink = ConvertUTF8ToUTF16BEWithErrors(row.bytes, dst)",
		"benchmarkIntSink = ConvertValidUTF8ToUTF16LE(row.bytes, dst)",
		"benchmarkIntSink = ConvertValidUTF8ToUTF16BE(row.bytes, dst)",
		"benchmarkIntSink = ConvertUTF8ToUTF32(row.bytes, dst)",
		"benchmarkResultSink = ConvertUTF8ToUTF32WithErrors(row.bytes, dst)",
		"benchmarkIntSink = ConvertValidUTF8ToUTF32(row.bytes, dst)",
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
