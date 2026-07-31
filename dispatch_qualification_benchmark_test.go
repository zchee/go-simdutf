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
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

// Hand-authored Go-only public-dispatch qualification harness pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b and
// docs/porting/benchmark-contract.md. It adds no product behavior or upstream
// algorithm translation. Corpus setup, integrity checks, and dispatch-provider
// qualification stay outside timed b.Loop bodies.

const (
	dispatchQualificationZeroSHA256          = "ad7facb2586fc6e966c004d7d1d16b024f5805ff7cb47c7a85dabd8b48892ca7"
	dispatchQualificationLatin1RampSHA256    = "c8f5d0341d54d951a71b136e6e2afcb14d11ed8489a7ae126a8fee0df6ecf193"
	dispatchQualificationArabicLipsumSHA256  = "b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb"
	dispatchQualificationArabicLipsumPath    = ".omx/artifacts/phase0/benchmark-corpora/corpus/unicode_lipsum/lipsum/Arabic-Lipsum.utf8.txt"
	dispatchQualificationArabicLipsumSize    = 81685
	dispatchQualificationFindByteSHA256      = "d5adc7f2d7e4de5ff826cc0ba543bb5ba2e8aaf7609c3594df10af4c9af4f3d8"
	dispatchQualificationFindU16LESHA256     = "96e0c9f77ab0c6b07682cf5252710117ede55c74a5b30b06201602c714a10bfb"
	dispatchQualificationDNSNormalizedSHA256 = "79f1eba2fe0c187f1086f7534b74cd1dd4ef795a515d7db13d613eebafdb1d6f"
	dispatchQualificationDNSSourcePath       = ".omx/artifacts/phase0/benchmark-corpora/corpus/base64data/dns/swedenzonebase.txt"
	dispatchQualificationDNSSourceSize       = 35100000
	dispatchQualificationDNSNormalizedSize   = 35000000
	dispatchQualificationOperationEnv        = "SIMDUTF_BENCH_EXPECT_OPERATION"
	dispatchQualificationTierEnv             = "SIMDUTF_BENCH_EXPECT_TIER"
)

var (
	benchmarkEncodingSink   Encoding
	benchmarkFullResultSink FullResult
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

func materializeDispatchQualificationFindCorpora() (findByte []byte, findU16LE []uint16, findU16LERaw []byte) {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	findByte = make([]byte, 4096)
	for i := range findByte {
		findByte[i] = alphabet[i%len(alphabet)]
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(findByte)); got != dispatchQualificationFindByteSHA256 {
		panic(fmt.Sprintf("Q-find-byte SHA-256 = %s, want %s", got, dispatchQualificationFindByteSHA256))
	}
	findU16LE = make([]uint16, 2048)
	findU16LERaw = make([]byte, 4096)
	for i := range 2048 {
		findU16LE[i] = uint16(findByte[i])
		binary.LittleEndian.PutUint16(findU16LERaw[2*i:], findU16LE[i])
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(findU16LERaw)); got != dispatchQualificationFindU16LESHA256 {
		panic(fmt.Sprintf("Q-find-u16le SHA-256 = %s, want %s", got, dispatchQualificationFindU16LESHA256))
	}
	return findByte, findU16LE, findU16LERaw
}

func dispatchQualificationBytesToUTF16(input []byte) []uint16 {
	out := make([]uint16, len(input))
	for i, b := range input {
		out[i] = uint16(b)
	}
	return out
}

var (
	dispatchQualificationDNSNormalized     []byte
	dispatchQualificationDNSNormalizedU16  []uint16
	dispatchQualificationDNSNormalizedOnce sync.Once
)

func loadDispatchQualificationDNSNormalized() []byte {
	dispatchQualificationDNSNormalizedOnce.Do(func() {
		data, err := os.ReadFile(dispatchQualificationDNSSourcePath)
		if err != nil {
			panic(fmt.Sprintf(
				"Q-dns-normalized source missing at %s: %v",
				dispatchQualificationDNSSourcePath, err,
			))
		}
		if len(data) != dispatchQualificationDNSSourceSize {
			panic(fmt.Sprintf(
				"Q-dns-source length = %d, want %d",
				len(data), dispatchQualificationDNSSourceSize,
			))
		}
		normalized := bytes.ReplaceAll(data, []byte{10}, nil)
		if len(normalized) != dispatchQualificationDNSNormalizedSize {
			panic(fmt.Sprintf(
				"Q-dns-normalized length = %d, want %d",
				len(normalized), dispatchQualificationDNSNormalizedSize,
			))
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(normalized)); got != dispatchQualificationDNSNormalizedSHA256 {
			panic(fmt.Sprintf(
				"Q-dns-normalized SHA-256 = %s, want %s",
				got, dispatchQualificationDNSNormalizedSHA256,
			))
		}
		dispatchQualificationDNSNormalized = normalized
		dispatchQualificationDNSNormalizedU16 = dispatchQualificationBytesToUTF16(normalized)
	})
	return dispatchQualificationDNSNormalized
}

func loadDispatchQualificationDNSNormalizedUTF16() []uint16 {
	_ = loadDispatchQualificationDNSNormalized()
	return dispatchQualificationDNSNormalizedU16
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
	findByte, findU16LE, _ := materializeDispatchQualificationFindCorpora()
	detectionValid := findByte
	dnsNormalized := loadDispatchQualificationDNSNormalized()
	dnsNormalizedU16 := loadDispatchQualificationDNSNormalizedUTF16()
	byteZeroU16 := dispatchQualificationBytesToUTF16(byteZero)
	emojiU16 := dispatchQualificationBytesToUTF16(upstreamEmojiUTF8)
	rows := make([]dispatchQualificationRow, 0, 1359)
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
	for _, operation := range [...]string{
		"ConvertUTF16LEToLatin1",
		"ConvertUTF16BEToLatin1",
		"ConvertUTF16LEToLatin1WithErrors",
		"ConvertUTF16BEToLatin1WithErrors",
		"ConvertValidUTF16LEToLatin1",
		"ConvertValidUTF16BEToLatin1",
		"ConvertUTF16LEToUTF32",
		"ConvertUTF16BEToUTF32",
		"ConvertUTF16LEToUTF32WithErrors",
		"ConvertUTF16BEToUTF32WithErrors",
		"ConvertValidUTF16LEToUTF32",
		"ConvertValidUTF16BEToUTF32",
		"UTF32LengthFromUTF16LE",
		"UTF32LengthFromUTF16BE",
		"ConvertUTF16LEToUTF8",
		"ConvertUTF16BEToUTF8",
		"ConvertUTF16LEToUTF8WithErrors",
		"ConvertUTF16BEToUTF8WithErrors",
		"ConvertUTF16LEToUTF8WithReplacement",
		"ConvertUTF16BEToUTF8WithReplacement",
		"ConvertValidUTF16LEToUTF8",
		"ConvertValidUTF16BEToUTF8",
		"UTF8LengthFromUTF16LE",
		"UTF8LengthFromUTF16BE",
		"ChangeEndiannessUTF16",
		"CountUTF16LE",
		"CountUTF16BE",
		"UTF8LengthFromUTF16LEWithReplacement",
		"UTF8LengthFromUTF16BEWithReplacement",
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
		"ConvertUTF32ToLatin1",
		"ConvertUTF32ToLatin1WithErrors",
		"ConvertValidUTF32ToLatin1",
		"ConvertUTF32ToUTF8",
		"ConvertUTF32ToUTF8WithErrors",
		"ConvertValidUTF32ToUTF8",
		"ConvertUTF32ToUTF16LE",
		"ConvertUTF32ToUTF16BE",
		"ConvertUTF32ToUTF16LEWithErrors",
		"ConvertUTF32ToUTF16BEWithErrors",
		"ConvertValidUTF32ToUTF16LE",
		"ConvertValidUTF32ToUTF16BE",
		"UTF8LengthFromUTF32",
		"UTF16LengthFromUTF32",
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

	for _, input := range dispatchQualificationByteSizes {
		rows = append(rows, dispatchQualificationRow{
			operation: "Find",
			corpus:    "Q-find-byte",
			class:     input.class,
			size:      input.size,
			bytes:     findByte[:input.size],
		})
	}
	for _, input := range dispatchQualificationUint16Sizes {
		rows = append(rows, dispatchQualificationRow{
			operation: "FindUTF16",
			corpus:    "Q-find-u16le",
			class:     input.class,
			size:      input.size,
			uint16s:   findU16LE[:input.size],
		})
	}
	for _, input := range dispatchQualificationByteSizes {
		rows = append(rows, dispatchQualificationRow{
			operation: "DetectEncodings",
			corpus:    "Q-detection-valid",
			class:     input.class,
			size:      input.size,
			bytes:     detectionValid[:input.size],
		})
	}
	for _, operation := range [...]string{
		"BinaryLengthFromBase64",
		"BinaryToBase64",
		"BinaryToBase64WithLines",
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
	for _, input := range dispatchQualificationByteSizes {
		rows = append(rows, dispatchQualificationRow{
			operation: "BinaryLengthFromBase64UTF16",
			corpus:    "Q-byte-zero",
			class:     input.class,
			size:      input.size,
			uint16s:   byteZeroU16[:input.size],
		})
	}
	rows = append(rows, dispatchQualificationRow{
		operation: "BinaryLengthFromBase64UTF16",
		corpus:    "Q-emoji",
		class:     "bulk",
		size:      len(emojiU16),
		uint16s:   emojiU16,
	})
	for _, operation := range [...]string{
		"Base64ToBinary",
		"Base64ToBinaryDetails",
	} {
		rows = append(rows, dispatchQualificationRow{
			operation: operation,
			corpus:    "Q-dns-normalized",
			class:     "bulk",
			size:      len(dnsNormalized),
			bytes:     dnsNormalized,
		})
	}
	for _, operation := range [...]string{
		"Base64ToBinaryUTF16",
		"Base64ToBinaryDetailsUTF16",
	} {
		rows = append(rows, dispatchQualificationRow{
			operation: operation,
			corpus:    "Q-dns-normalized",
			class:     "bulk",
			size:      len(dnsNormalizedU16),
			uint16s:   dnsNormalizedU16,
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
	"ConvertUTF16LEToLatin1": {
		"scalar":   {"convertUTF16LEToLatin1Scalar"},
		"westmere": {"convertUTF16LEToLatin1Westmere"},
		"haswell":  {"convertUTF16LEToLatin1Haswell"},
		"neon":     {"convertUTF16LEToLatin1NEON"},
		"archsimd": {"convertUTF16LEToLatin1Archsimd"},
	},
	"ConvertUTF16BEToLatin1": {
		"scalar":   {"convertUTF16BEToLatin1Scalar"},
		"westmere": {"convertUTF16BEToLatin1Westmere"},
		"haswell":  {"convertUTF16BEToLatin1Haswell"},
		"neon":     {"convertUTF16BEToLatin1NEON"},
		"archsimd": {"convertUTF16BEToLatin1Archsimd"},
	},
	"ConvertUTF16LEToLatin1WithErrors": {
		"scalar":   {"convertUTF16LEToLatin1WithErrorsScalar"},
		"westmere": {"convertUTF16LEToLatin1WithErrorsWestmere"},
		"haswell":  {"convertUTF16LEToLatin1WithErrorsHaswell"},
		"neon":     {"convertUTF16LEToLatin1WithErrorsNEON"},
		"archsimd": {"convertUTF16LEToLatin1WithErrorsArchsimd"},
	},
	"ConvertUTF16BEToLatin1WithErrors": {
		"scalar":   {"convertUTF16BEToLatin1WithErrorsScalar"},
		"westmere": {"convertUTF16BEToLatin1WithErrorsWestmere"},
		"haswell":  {"convertUTF16BEToLatin1WithErrorsHaswell"},
		"neon":     {"convertUTF16BEToLatin1WithErrorsNEON"},
		"archsimd": {"convertUTF16BEToLatin1WithErrorsArchsimd"},
	},
	"ConvertValidUTF16LEToLatin1": {
		"scalar":   {"convertValidUTF16LEToLatin1Scalar"},
		"westmere": {"convertValidUTF16LEToLatin1Westmere"},
		"haswell":  {"convertValidUTF16LEToLatin1Haswell"},
		"neon":     {"convertValidUTF16LEToLatin1NEON"},
		"archsimd": {"convertValidUTF16LEToLatin1Archsimd"},
	},
	"ConvertValidUTF16BEToLatin1": {
		"scalar":   {"convertValidUTF16BEToLatin1Scalar"},
		"westmere": {"convertValidUTF16BEToLatin1Westmere"},
		"haswell":  {"convertValidUTF16BEToLatin1Haswell"},
		"neon":     {"convertValidUTF16BEToLatin1NEON"},
		"archsimd": {"convertValidUTF16BEToLatin1Archsimd"},
	},
	"ConvertUTF16LEToUTF32": {
		"scalar":   {"convertUTF16LEToUTF32Scalar"},
		"westmere": {"convertUTF16LEToUTF32Westmere"},
		"haswell":  {"convertUTF16LEToUTF32Haswell"},
		"neon":     {"convertUTF16LEToUTF32NEON"},
		"archsimd": {"convertUTF16LEToUTF32Archsimd"},
	},
	"ConvertUTF16BEToUTF32": {
		"scalar":   {"convertUTF16BEToUTF32Scalar"},
		"westmere": {"convertUTF16BEToUTF32Westmere"},
		"haswell":  {"convertUTF16BEToUTF32Haswell"},
		"neon":     {"convertUTF16BEToUTF32NEON"},
		"archsimd": {"convertUTF16BEToUTF32Archsimd"},
	},
	"ConvertUTF16LEToUTF32WithErrors": {
		"scalar":   {"convertUTF16LEToUTF32WithErrorsScalar"},
		"westmere": {"convertUTF16LEToUTF32WithErrorsWestmere"},
		"haswell":  {"convertUTF16LEToUTF32WithErrorsHaswell"},
		"neon":     {"convertUTF16LEToUTF32WithErrorsNEON"},
		"archsimd": {"convertUTF16LEToUTF32WithErrorsArchsimd"},
	},
	"ConvertUTF16BEToUTF32WithErrors": {
		"scalar":   {"convertUTF16BEToUTF32WithErrorsScalar"},
		"westmere": {"convertUTF16BEToUTF32WithErrorsWestmere"},
		"haswell":  {"convertUTF16BEToUTF32WithErrorsHaswell"},
		"neon":     {"convertUTF16BEToUTF32WithErrorsNEON"},
		"archsimd": {"convertUTF16BEToUTF32WithErrorsArchsimd"},
	},
	"ConvertValidUTF16LEToUTF32": {
		"scalar":   {"convertValidUTF16LEToUTF32Scalar"},
		"westmere": {"convertValidUTF16LEToUTF32Westmere"},
		"haswell":  {"convertValidUTF16LEToUTF32Haswell"},
		"neon":     {"convertValidUTF16LEToUTF32NEON"},
		"archsimd": {"convertValidUTF16LEToUTF32Archsimd"},
	},
	"ConvertValidUTF16BEToUTF32": {
		"scalar":   {"convertValidUTF16BEToUTF32Scalar"},
		"westmere": {"convertValidUTF16BEToUTF32Westmere"},
		"haswell":  {"convertValidUTF16BEToUTF32Haswell"},
		"neon":     {"convertValidUTF16BEToUTF32NEON"},
		"archsimd": {"convertValidUTF16BEToUTF32Archsimd"},
	},
	"UTF32LengthFromUTF16LE": {
		"scalar":   {"utf32LengthFromUTF16LEScalar"},
		"westmere": {"utf32LengthFromUTF16LEWestmere"},
		"haswell":  {"utf32LengthFromUTF16LEHaswell"},
		"neon":     {"utf32LengthFromUTF16LENEON"},
		"archsimd": {"utf32LengthFromUTF16LEArchsimd"},
	},
	"UTF32LengthFromUTF16BE": {
		"scalar":   {"utf32LengthFromUTF16BEScalar"},
		"westmere": {"utf32LengthFromUTF16BEWestmere"},
		"haswell":  {"utf32LengthFromUTF16BEHaswell"},
		"neon":     {"utf32LengthFromUTF16BENEON"},
		"archsimd": {"utf32LengthFromUTF16BEArchsimd"},
	},
	"ConvertUTF16LEToUTF8": {
		"scalar":   {"convertUTF16LEToUTF8Scalar"},
		"westmere": {"convertUTF16LEToUTF8Westmere"},
		"haswell":  {"convertUTF16LEToUTF8Haswell"},
		"neon":     {"convertUTF16LEToUTF8NEON"},
		"archsimd": {"convertUTF16LEToUTF8Archsimd"},
	},
	"ConvertUTF16BEToUTF8": {
		"scalar":   {"convertUTF16BEToUTF8Scalar"},
		"westmere": {"convertUTF16BEToUTF8Westmere"},
		"haswell":  {"convertUTF16BEToUTF8Haswell"},
		"neon":     {"convertUTF16BEToUTF8NEON"},
		"archsimd": {"convertUTF16BEToUTF8Archsimd"},
	},
	"ConvertUTF16LEToUTF8WithErrors": {
		"scalar":   {"convertUTF16LEToUTF8WithErrorsScalar"},
		"westmere": {"convertUTF16LEToUTF8WithErrorsWestmere"},
		"haswell":  {"convertUTF16LEToUTF8WithErrorsHaswell"},
		"neon":     {"convertUTF16LEToUTF8WithErrorsNEON"},
		"archsimd": {"convertUTF16LEToUTF8WithErrorsArchsimd"},
	},
	"ConvertUTF16BEToUTF8WithErrors": {
		"scalar":   {"convertUTF16BEToUTF8WithErrorsScalar"},
		"westmere": {"convertUTF16BEToUTF8WithErrorsWestmere"},
		"haswell":  {"convertUTF16BEToUTF8WithErrorsHaswell"},
		"neon":     {"convertUTF16BEToUTF8WithErrorsNEON"},
		"archsimd": {"convertUTF16BEToUTF8WithErrorsArchsimd"},
	},
	"ConvertUTF16LEToUTF8WithReplacement": {
		"scalar":   {"convertUTF16LEToUTF8WithReplacementScalar"},
		"westmere": {"convertUTF16LEToUTF8WithReplacementWestmere"},
		"haswell":  {"convertUTF16LEToUTF8WithReplacementHaswell"},
		"neon":     {"convertUTF16LEToUTF8WithReplacementNEON"},
		"archsimd": {"convertUTF16LEToUTF8WithReplacementArchsimd"},
	},
	"ConvertUTF16BEToUTF8WithReplacement": {
		"scalar":   {"convertUTF16BEToUTF8WithReplacementScalar"},
		"westmere": {"convertUTF16BEToUTF8WithReplacementWestmere"},
		"haswell":  {"convertUTF16BEToUTF8WithReplacementHaswell"},
		"neon":     {"convertUTF16BEToUTF8WithReplacementNEON"},
		"archsimd": {"convertUTF16BEToUTF8WithReplacementArchsimd"},
	},
	"ConvertValidUTF16LEToUTF8": {
		"scalar":   {"convertValidUTF16LEToUTF8Scalar"},
		"westmere": {"convertValidUTF16LEToUTF8Westmere"},
		"haswell":  {"convertValidUTF16LEToUTF8Haswell"},
		"neon":     {"convertValidUTF16LEToUTF8NEON"},
		"archsimd": {"convertValidUTF16LEToUTF8Archsimd"},
	},
	"ConvertValidUTF16BEToUTF8": {
		"scalar":   {"convertValidUTF16BEToUTF8Scalar"},
		"westmere": {"convertValidUTF16BEToUTF8Westmere"},
		"haswell":  {"convertValidUTF16BEToUTF8Haswell"},
		"neon":     {"convertValidUTF16BEToUTF8NEON"},
		"archsimd": {"convertValidUTF16BEToUTF8Archsimd"},
	},
	"UTF8LengthFromUTF16LE": {
		"scalar":   {"utf8LengthFromUTF16LEScalar"},
		"westmere": {"utf8LengthFromUTF16LEWestmere"},
		"haswell":  {"utf8LengthFromUTF16LEHaswell"},
		"neon":     {"utf8LengthFromUTF16LENEON"},
		"archsimd": {"utf8LengthFromUTF16LEArchsimd"},
	},
	"UTF8LengthFromUTF16BE": {
		"scalar":   {"utf8LengthFromUTF16BEScalar"},
		"westmere": {"utf8LengthFromUTF16BEWestmere"},
		"haswell":  {"utf8LengthFromUTF16BEHaswell"},
		"neon":     {"utf8LengthFromUTF16BENEON"},
		"archsimd": {"utf8LengthFromUTF16BEArchsimd"},
	},
	"ChangeEndiannessUTF16": {
		"scalar":   {"changeEndiannessUTF16Scalar"},
		"westmere": {"changeEndiannessUTF16Westmere"},
		"haswell":  {"changeEndiannessUTF16Haswell"},
		"neon":     {"changeEndiannessUTF16NEON"},
		"archsimd": {"changeEndiannessUTF16Archsimd"},
	},
	"CountUTF16LE": {
		"scalar":   {"countUTF16LEScalar"},
		"westmere": {"countUTF16LEWestmere"},
		"haswell":  {"countUTF16LEHaswell"},
		"neon":     {"countUTF16LENEON"},
		"archsimd": {"countUTF16LEArchsimd"},
	},
	"CountUTF16BE": {
		"scalar":   {"countUTF16BEScalar"},
		"westmere": {"countUTF16BEWestmere"},
		"haswell":  {"countUTF16BEHaswell"},
		"neon":     {"countUTF16BENEON"},
		"archsimd": {"countUTF16BEArchsimd"},
	},
	"UTF8LengthFromUTF16LEWithReplacement": {
		"scalar":   {"utf8LengthFromUTF16LEWithReplacementScalar"},
		"westmere": {"utf8LengthFromUTF16LEWithReplacementWestmere"},
		"haswell":  {"utf8LengthFromUTF16LEWithReplacementHaswell"},
		"neon":     {"utf8LengthFromUTF16LEWithReplacementNEON"},
		"archsimd": {"utf8LengthFromUTF16LEWithReplacementArchsimd"},
	},
	"UTF8LengthFromUTF16BEWithReplacement": {
		"scalar":   {"utf8LengthFromUTF16BEWithReplacementScalar"},
		"westmere": {"utf8LengthFromUTF16BEWithReplacementWestmere"},
		"haswell":  {"utf8LengthFromUTF16BEWithReplacementHaswell"},
		"neon":     {"utf8LengthFromUTF16BEWithReplacementNEON"},
		"archsimd": {"utf8LengthFromUTF16BEWithReplacementArchsimd"},
	},
	"ConvertUTF32ToLatin1": {
		"scalar":   {"convertUTF32ToLatin1Scalar"},
		"westmere": {"convertUTF32ToLatin1Westmere"},
		"haswell":  {"convertUTF32ToLatin1Haswell"},
		"neon":     {"convertUTF32ToLatin1NEON"},
		"archsimd": {"convertUTF32ToLatin1Archsimd"},
	},
	"ConvertUTF32ToLatin1WithErrors": {
		"scalar":   {"convertUTF32ToLatin1WithErrorsScalar"},
		"westmere": {"convertUTF32ToLatin1WithErrorsWestmere"},
		"haswell":  {"convertUTF32ToLatin1WithErrorsHaswell"},
		"neon":     {"convertUTF32ToLatin1WithErrorsNEON"},
		"archsimd": {"convertUTF32ToLatin1WithErrorsArchsimd"},
	},
	"ConvertValidUTF32ToLatin1": {
		"scalar":   {"convertValidUTF32ToLatin1Scalar"},
		"westmere": {"convertValidUTF32ToLatin1Westmere"},
		"haswell":  {"convertValidUTF32ToLatin1Haswell"},
		"neon":     {"convertValidUTF32ToLatin1NEON"},
		"archsimd": {"convertValidUTF32ToLatin1Archsimd"},
	},
	"ConvertUTF32ToUTF8": {
		"scalar":   {"convertUTF32ToUTF8Scalar"},
		"westmere": {"convertUTF32ToUTF8Westmere"},
		"haswell":  {"convertUTF32ToUTF8Haswell"},
		"neon":     {"convertUTF32ToUTF8NEON"},
		"archsimd": {"convertUTF32ToUTF8Archsimd"},
	},
	"ConvertUTF32ToUTF8WithErrors": {
		"scalar":   {"convertUTF32ToUTF8WithErrorsScalar"},
		"westmere": {"convertUTF32ToUTF8WithErrorsWestmere"},
		"haswell":  {"convertUTF32ToUTF8WithErrorsHaswell"},
		"neon":     {"convertUTF32ToUTF8WithErrorsNEON"},
		"archsimd": {"convertUTF32ToUTF8WithErrorsArchsimd"},
	},
	"ConvertValidUTF32ToUTF8": {
		"scalar":   {"convertValidUTF32ToUTF8Scalar"},
		"westmere": {"convertValidUTF32ToUTF8Westmere"},
		"haswell":  {"convertValidUTF32ToUTF8Haswell"},
		"neon":     {"convertValidUTF32ToUTF8NEON"},
		"archsimd": {"convertValidUTF32ToUTF8Archsimd"},
	},
	"ConvertUTF32ToUTF16LE": {
		"scalar":   {"convertUTF32ToUTF16LEScalar"},
		"westmere": {"convertUTF32ToUTF16LEWestmere"},
		"haswell":  {"convertUTF32ToUTF16LEHaswell"},
		"neon":     {"convertUTF32ToUTF16LENEON"},
		"archsimd": {"convertUTF32ToUTF16LEArchsimd"},
	},
	"ConvertUTF32ToUTF16BE": {
		"scalar":   {"convertUTF32ToUTF16BEScalar"},
		"westmere": {"convertUTF32ToUTF16BEWestmere"},
		"haswell":  {"convertUTF32ToUTF16BEHaswell"},
		"neon":     {"convertUTF32ToUTF16BENEON"},
		"archsimd": {"convertUTF32ToUTF16BEArchsimd"},
	},
	"ConvertUTF32ToUTF16LEWithErrors": {
		"scalar":   {"convertUTF32ToUTF16LEWithErrorsScalar"},
		"westmere": {"convertUTF32ToUTF16LEWithErrorsWestmere"},
		"haswell":  {"convertUTF32ToUTF16LEWithErrorsHaswell"},
		"neon":     {"convertUTF32ToUTF16LEWithErrorsNEON"},
		"archsimd": {"convertUTF32ToUTF16LEWithErrorsArchsimd"},
	},
	"ConvertUTF32ToUTF16BEWithErrors": {
		"scalar":   {"convertUTF32ToUTF16BEWithErrorsScalar"},
		"westmere": {"convertUTF32ToUTF16BEWithErrorsWestmere"},
		"haswell":  {"convertUTF32ToUTF16BEWithErrorsHaswell"},
		"neon":     {"convertUTF32ToUTF16BEWithErrorsNEON"},
		"archsimd": {"convertUTF32ToUTF16BEWithErrorsArchsimd"},
	},
	"ConvertValidUTF32ToUTF16LE": {
		"scalar":   {"convertValidUTF32ToUTF16LEScalar"},
		"westmere": {"convertValidUTF32ToUTF16LEWestmere"},
		"haswell":  {"convertValidUTF32ToUTF16LEHaswell"},
		"neon":     {"convertValidUTF32ToUTF16LENEON"},
		"archsimd": {"convertValidUTF32ToUTF16LEArchsimd"},
	},
	"ConvertValidUTF32ToUTF16BE": {
		"scalar":   {"convertValidUTF32ToUTF16BEScalar"},
		"westmere": {"convertValidUTF32ToUTF16BEWestmere"},
		"haswell":  {"convertValidUTF32ToUTF16BEHaswell"},
		"neon":     {"convertValidUTF32ToUTF16BENEON"},
		"archsimd": {"convertValidUTF32ToUTF16BEArchsimd"},
	},
	"UTF8LengthFromUTF32": {
		"scalar":   {"utf8LengthFromUTF32Scalar"},
		"westmere": {"utf8LengthFromUTF32Westmere"},
		"haswell":  {"utf8LengthFromUTF32Haswell"},
		"neon":     {"utf8LengthFromUTF32NEON"},
		"archsimd": {"utf8LengthFromUTF32Archsimd"},
	},
	"UTF16LengthFromUTF32": {
		"scalar":   {"utf16LengthFromUTF32Scalar"},
		"westmere": {"utf16LengthFromUTF32Westmere"},
		"haswell":  {"utf16LengthFromUTF32Haswell"},
		"neon":     {"utf16LengthFromUTF32NEON"},
		"archsimd": {"utf16LengthFromUTF32Archsimd"},
	},
	"Find": {
		"scalar":   {"findScalar"},
		"westmere": {"findWestmere"},
		"haswell":  {"findHaswell"},
		"neon":     {"findNEON"},
		"archsimd": {"findArchsimd"},
	},
	"FindUTF16": {
		"scalar":   {"findUTF16Scalar"},
		"westmere": {"findUTF16Westmere"},
		"haswell":  {"findUTF16Haswell"},
		"neon":     {"findUTF16NEON"},
		"archsimd": {"findUTF16Archsimd"},
	},
	"DetectEncodings": {
		"scalar":   {"detectEncodingsScalar"},
		"westmere": {"detectEncodingsWestmere"},
		"haswell":  {"detectEncodingsHaswell"},
		"neon":     {"detectEncodingsNEON"},
		"archsimd": {"detectEncodingsArchsimd"},
	},
	"BinaryLengthFromBase64": {
		"scalar":   {"binaryLengthFromBase64Scalar"},
		"westmere": {"binaryLengthFromBase64Westmere"},
		"haswell":  {"binaryLengthFromBase64Haswell"},
		"neon":     {"binaryLengthFromBase64NEON"},
		"archsimd": {"binaryLengthFromBase64Archsimd"},
	},
	"BinaryLengthFromBase64UTF16": {
		"scalar":   {"binaryLengthFromBase64UTF16Scalar"},
		"westmere": {"binaryLengthFromBase64UTF16Westmere"},
		"haswell":  {"binaryLengthFromBase64UTF16Haswell"},
		"neon":     {"binaryLengthFromBase64UTF16NEON"},
		"archsimd": {"binaryLengthFromBase64UTF16Archsimd"},
	},
	"Base64ToBinary": {
		"scalar":   {"base64ToBinaryScalar"},
		"westmere": {"base64ToBinaryWestmere"},
		"haswell":  {"base64ToBinaryHaswell"},
		"neon":     {"base64ToBinaryNEON"},
		"archsimd": {"base64ToBinaryArchsimd"},
	},
	"Base64ToBinaryUTF16": {
		"scalar":   {"base64ToBinaryUTF16Scalar"},
		"westmere": {"base64ToBinaryUTF16Westmere"},
		"haswell":  {"base64ToBinaryUTF16Haswell"},
		"neon":     {"base64ToBinaryUTF16NEON"},
		"archsimd": {"base64ToBinaryUTF16Archsimd"},
	},
	"Base64ToBinaryDetails": {
		"scalar":   {"base64ToBinaryDetailsScalar"},
		"westmere": {"base64ToBinaryDetailsWestmere"},
		"haswell":  {"base64ToBinaryDetailsHaswell"},
		"neon":     {"base64ToBinaryDetailsNEON"},
		"archsimd": {"base64ToBinaryDetailsArchsimd"},
	},
	"Base64ToBinaryDetailsUTF16": {
		"scalar":   {"base64ToBinaryDetailsUTF16Scalar"},
		"westmere": {"base64ToBinaryDetailsUTF16Westmere"},
		"haswell":  {"base64ToBinaryDetailsUTF16Haswell"},
		"neon":     {"base64ToBinaryDetailsUTF16NEON"},
		"archsimd": {"base64ToBinaryDetailsUTF16Archsimd"},
	},
	"BinaryToBase64": {
		"scalar":   {"binaryToBase64Scalar"},
		"westmere": {"binaryToBase64Westmere"},
		"haswell":  {"binaryToBase64Haswell"},
		"neon":     {"binaryToBase64NEON"},
		"archsimd": {"binaryToBase64Archsimd"},
	},
	"BinaryToBase64WithLines": {
		"scalar":   {"binaryToBase64WithLinesScalar"},
		"westmere": {"binaryToBase64WithLinesWestmere"},
		"haswell":  {"binaryToBase64WithLinesHaswell"},
		"neon":     {"binaryToBase64WithLinesNEON"},
		"archsimd": {"binaryToBase64WithLinesArchsimd"},
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
	if slices.Contains(allowed, identifier) {
		return nil
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
	case "ConvertUTF16LEToLatin1":
		return activeImplementation.convertUTF16LEToLatin1
	case "ConvertUTF16BEToLatin1":
		return activeImplementation.convertUTF16BEToLatin1
	case "ConvertUTF16LEToLatin1WithErrors":
		return activeImplementation.convertUTF16LEToLatin1WithErrors
	case "ConvertUTF16BEToLatin1WithErrors":
		return activeImplementation.convertUTF16BEToLatin1WithErrors
	case "ConvertValidUTF16LEToLatin1":
		return activeImplementation.convertValidUTF16LEToLatin1
	case "ConvertValidUTF16BEToLatin1":
		return activeImplementation.convertValidUTF16BEToLatin1
	case "ConvertUTF16LEToUTF32":
		return activeImplementation.convertUTF16LEToUTF32
	case "ConvertUTF16BEToUTF32":
		return activeImplementation.convertUTF16BEToUTF32
	case "ConvertUTF16LEToUTF32WithErrors":
		return activeImplementation.convertUTF16LEToUTF32WithErrors
	case "ConvertUTF16BEToUTF32WithErrors":
		return activeImplementation.convertUTF16BEToUTF32WithErrors
	case "ConvertValidUTF16LEToUTF32":
		return activeImplementation.convertValidUTF16LEToUTF32
	case "ConvertValidUTF16BEToUTF32":
		return activeImplementation.convertValidUTF16BEToUTF32
	case "UTF32LengthFromUTF16LE":
		return activeImplementation.utf32LengthFromUTF16LE
	case "UTF32LengthFromUTF16BE":
		return activeImplementation.utf32LengthFromUTF16BE
	case "ConvertUTF16LEToUTF8":
		return activeImplementation.convertUTF16LEToUTF8
	case "ConvertUTF16BEToUTF8":
		return activeImplementation.convertUTF16BEToUTF8
	case "ConvertUTF16LEToUTF8WithErrors":
		return activeImplementation.convertUTF16LEToUTF8WithErrors
	case "ConvertUTF16BEToUTF8WithErrors":
		return activeImplementation.convertUTF16BEToUTF8WithErrors
	case "ConvertUTF16LEToUTF8WithReplacement":
		return activeImplementation.convertUTF16LEToUTF8WithReplacement
	case "ConvertUTF16BEToUTF8WithReplacement":
		return activeImplementation.convertUTF16BEToUTF8WithReplacement
	case "ConvertValidUTF16LEToUTF8":
		return activeImplementation.convertValidUTF16LEToUTF8
	case "ConvertValidUTF16BEToUTF8":
		return activeImplementation.convertValidUTF16BEToUTF8
	case "UTF8LengthFromUTF16LE":
		return activeImplementation.utf8LengthFromUTF16LE
	case "UTF8LengthFromUTF16BE":
		return activeImplementation.utf8LengthFromUTF16BE
	case "ChangeEndiannessUTF16":
		return activeImplementation.changeEndiannessUTF16
	case "CountUTF16LE":
		return activeImplementation.countUTF16LE
	case "CountUTF16BE":
		return activeImplementation.countUTF16BE
	case "UTF8LengthFromUTF16LEWithReplacement":
		return activeImplementation.utf8LengthFromUTF16LEWithReplacement
	case "UTF8LengthFromUTF16BEWithReplacement":
		return activeImplementation.utf8LengthFromUTF16BEWithReplacement
	case "ConvertUTF32ToLatin1":
		return activeImplementation.convertUTF32ToLatin1
	case "ConvertUTF32ToLatin1WithErrors":
		return activeImplementation.convertUTF32ToLatin1WithErrors
	case "ConvertValidUTF32ToLatin1":
		return activeImplementation.convertValidUTF32ToLatin1
	case "ConvertUTF32ToUTF8":
		return activeImplementation.convertUTF32ToUTF8
	case "ConvertUTF32ToUTF8WithErrors":
		return activeImplementation.convertUTF32ToUTF8WithErrors
	case "ConvertValidUTF32ToUTF8":
		return activeImplementation.convertValidUTF32ToUTF8
	case "ConvertUTF32ToUTF16LE":
		return activeImplementation.convertUTF32ToUTF16LE
	case "ConvertUTF32ToUTF16BE":
		return activeImplementation.convertUTF32ToUTF16BE
	case "ConvertUTF32ToUTF16LEWithErrors":
		return activeImplementation.convertUTF32ToUTF16LEWithErrors
	case "ConvertUTF32ToUTF16BEWithErrors":
		return activeImplementation.convertUTF32ToUTF16BEWithErrors
	case "ConvertValidUTF32ToUTF16LE":
		return activeImplementation.convertValidUTF32ToUTF16LE
	case "ConvertValidUTF32ToUTF16BE":
		return activeImplementation.convertValidUTF32ToUTF16BE
	case "UTF8LengthFromUTF32":
		return activeImplementation.utf8LengthFromUTF32
	case "UTF16LengthFromUTF32":
		return activeImplementation.utf16LengthFromUTF32
	case "Find":
		return activeImplementation.find
	case "FindUTF16":
		return activeImplementation.findUTF16
	case "DetectEncodings":
		return activeImplementation.detectEncodings
	case "BinaryLengthFromBase64":
		return activeImplementation.binaryLengthFromBase64
	case "BinaryLengthFromBase64UTF16":
		return activeImplementation.binaryLengthFromBase64UTF16
	case "Base64ToBinary":
		return activeImplementation.base64ToBinary
	case "Base64ToBinaryUTF16":
		return activeImplementation.base64ToBinaryUTF16
	case "Base64ToBinaryDetails":
		return activeImplementation.base64ToBinaryDetails
	case "Base64ToBinaryDetailsUTF16":
		return activeImplementation.base64ToBinaryDetailsUTF16
	case "BinaryToBase64":
		return activeImplementation.binaryToBase64
	case "BinaryToBase64WithLines":
		return activeImplementation.binaryToBase64WithLines
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
			case "ConvertUTF16LEToLatin1":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16LEToLatin1(row.uint16s, dst)
				}
			case "ConvertUTF16BEToLatin1":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16BEToLatin1(row.uint16s, dst)
				}
			case "ConvertUTF16LEToLatin1WithErrors":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF16LEToLatin1WithErrors(row.uint16s, dst)
				}
			case "ConvertUTF16BEToLatin1WithErrors":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF16BEToLatin1WithErrors(row.uint16s, dst)
				}
			case "ConvertValidUTF16LEToLatin1":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF16LEToLatin1(row.uint16s, dst)
				}
			case "ConvertValidUTF16BEToLatin1":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF16BEToLatin1(row.uint16s, dst)
				}
			case "ConvertUTF16LEToUTF32":
				dst := make([]uint32, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16LEToUTF32(row.uint16s, dst)
				}
			case "ConvertUTF16BEToUTF32":
				dst := make([]uint32, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16BEToUTF32(row.uint16s, dst)
				}
			case "ConvertUTF16LEToUTF32WithErrors":
				dst := make([]uint32, len(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF16LEToUTF32WithErrors(row.uint16s, dst)
				}
			case "ConvertUTF16BEToUTF32WithErrors":
				dst := make([]uint32, len(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF16BEToUTF32WithErrors(row.uint16s, dst)
				}
			case "ConvertValidUTF16LEToUTF32":
				dst := make([]uint32, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF16LEToUTF32(row.uint16s, dst)
				}
			case "ConvertValidUTF16BEToUTF32":
				dst := make([]uint32, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF16BEToUTF32(row.uint16s, dst)
				}
			case "UTF32LengthFromUTF16LE":
				for b.Loop() {
					benchmarkIntSink = UTF32LengthFromUTF16LE(row.uint16s)
				}
			case "UTF32LengthFromUTF16BE":
				for b.Loop() {
					benchmarkIntSink = UTF32LengthFromUTF16BE(row.uint16s)
				}
			case "ConvertUTF16LEToUTF8":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16LEToUTF8(row.uint16s, dst)
				}
			case "ConvertUTF16BEToUTF8":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16BEToUTF8(row.uint16s, dst)
				}
			case "ConvertUTF16LEToUTF8WithErrors":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF16LEToUTF8WithErrors(row.uint16s, dst)
				}
			case "ConvertUTF16BEToUTF8WithErrors":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF16BEToUTF8WithErrors(row.uint16s, dst)
				}
			case "ConvertUTF16LEToUTF8WithReplacement":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16LEToUTF8WithReplacement(row.uint16s, dst)
				}
			case "ConvertUTF16BEToUTF8WithReplacement":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF16BEToUTF8WithReplacement(row.uint16s, dst)
				}
			case "ConvertValidUTF16LEToUTF8":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF16LEToUTF8(row.uint16s, dst)
				}
			case "ConvertValidUTF16BEToUTF8":
				dst := make([]byte, len(row.uint16s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF16BEToUTF8(row.uint16s, dst)
				}
			case "UTF8LengthFromUTF16LE":
				for b.Loop() {
					benchmarkIntSink = UTF8LengthFromUTF16LE(row.uint16s)
				}
			case "UTF8LengthFromUTF16BE":
				for b.Loop() {
					benchmarkIntSink = UTF8LengthFromUTF16BE(row.uint16s)
				}
			case "ChangeEndiannessUTF16":
				dst := make([]uint16, len(row.uint16s))
				for b.Loop() {
					ChangeEndiannessUTF16(row.uint16s, dst)
				}
			case "CountUTF16LE":
				for b.Loop() {
					benchmarkIntSink = CountUTF16LE(row.uint16s)
				}
			case "CountUTF16BE":
				for b.Loop() {
					benchmarkIntSink = CountUTF16BE(row.uint16s)
				}
			case "UTF8LengthFromUTF16LEWithReplacement":
				for b.Loop() {
					benchmarkResultSink = UTF8LengthFromUTF16LEWithReplacement(row.uint16s)
				}
			case "UTF8LengthFromUTF16BEWithReplacement":
				for b.Loop() {
					benchmarkResultSink = UTF8LengthFromUTF16BEWithReplacement(row.uint16s)
				}
			case "ConvertUTF32ToLatin1":
				dst := make([]byte, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF32ToLatin1(row.uint32s, dst)
				}
			case "ConvertUTF32ToLatin1WithErrors":
				dst := make([]byte, len(row.uint32s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF32ToLatin1WithErrors(row.uint32s, dst)
				}
			case "ConvertValidUTF32ToLatin1":
				dst := make([]byte, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF32ToLatin1(row.uint32s, dst)
				}
			case "ConvertUTF32ToUTF8":
				dst := make([]byte, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF32ToUTF8(row.uint32s, dst)
				}
			case "ConvertUTF32ToUTF8WithErrors":
				dst := make([]byte, len(row.uint32s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF32ToUTF8WithErrors(row.uint32s, dst)
				}
			case "ConvertValidUTF32ToUTF8":
				dst := make([]byte, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF32ToUTF8(row.uint32s, dst)
				}
			case "ConvertUTF32ToUTF16LE":
				dst := make([]uint16, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF32ToUTF16LE(row.uint32s, dst)
				}
			case "ConvertUTF32ToUTF16BE":
				dst := make([]uint16, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertUTF32ToUTF16BE(row.uint32s, dst)
				}
			case "ConvertUTF32ToUTF16LEWithErrors":
				dst := make([]uint16, len(row.uint32s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF32ToUTF16LEWithErrors(row.uint32s, dst)
				}
			case "ConvertUTF32ToUTF16BEWithErrors":
				dst := make([]uint16, len(row.uint32s))
				for b.Loop() {
					benchmarkResultSink = ConvertUTF32ToUTF16BEWithErrors(row.uint32s, dst)
				}
			case "ConvertValidUTF32ToUTF16LE":
				dst := make([]uint16, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF32ToUTF16LE(row.uint32s, dst)
				}
			case "ConvertValidUTF32ToUTF16BE":
				dst := make([]uint16, len(row.uint32s))
				for b.Loop() {
					benchmarkIntSink = ConvertValidUTF32ToUTF16BE(row.uint32s, dst)
				}
			case "UTF8LengthFromUTF32":
				for b.Loop() {
					benchmarkIntSink = UTF8LengthFromUTF32(row.uint32s)
				}
			case "UTF16LengthFromUTF32":
				for b.Loop() {
					benchmarkIntSink = UTF16LengthFromUTF32(row.uint32s)
				}
			case "Find":
				needle := row.bytes[len(row.bytes)-1]
				for b.Loop() {
					benchmarkIntSink = Find(row.bytes, needle)
				}
			case "FindUTF16":
				needle := row.uint16s[len(row.uint16s)-1]
				for b.Loop() {
					benchmarkIntSink = FindUTF16(row.uint16s, needle)
				}
			case "DetectEncodings":
				for b.Loop() {
					benchmarkEncodingSink = DetectEncodings(row.bytes)
				}
			case "BinaryLengthFromBase64":
				for b.Loop() {
					benchmarkIntSink = BinaryLengthFromBase64(row.bytes)
				}
			case "BinaryLengthFromBase64UTF16":
				for b.Loop() {
					benchmarkIntSink = BinaryLengthFromBase64UTF16(row.uint16s)
				}
			case "Base64ToBinary":
				dst := make([]byte, MaximalBinaryLengthFromBase64(row.bytes))
				for b.Loop() {
					benchmarkResultSink = Base64ToBinary(row.bytes, dst, Base64Default, Loose)
				}
			case "Base64ToBinaryUTF16":
				dst := make([]byte, MaximalBinaryLengthFromBase64UTF16(row.uint16s))
				for b.Loop() {
					benchmarkResultSink = Base64ToBinaryUTF16(row.uint16s, dst, Base64Default, Loose)
				}
			case "Base64ToBinaryDetails":
				dst := make([]byte, MaximalBinaryLengthFromBase64(row.bytes))
				for b.Loop() {
					benchmarkFullResultSink = Base64ToBinaryDetails(row.bytes, dst, Base64Default, Loose)
				}
			case "Base64ToBinaryDetailsUTF16":
				dst := make([]byte, MaximalBinaryLengthFromBase64UTF16(row.uint16s))
				for b.Loop() {
					benchmarkFullResultSink = Base64ToBinaryDetailsUTF16(row.uint16s, dst, Base64Default, Loose)
				}
			case "BinaryToBase64":
				dst := make([]byte, Base64LengthFromBinary(len(row.bytes), Base64Default))
				for b.Loop() {
					benchmarkIntSink = BinaryToBase64(row.bytes, dst, Base64Default)
				}
			case "BinaryToBase64WithLines":
				dst := make([]byte, Base64LengthFromBinaryWithLines(len(row.bytes), Base64Default, DefaultLineLength))
				for b.Loop() {
					benchmarkIntSink = BinaryToBase64WithLines(row.bytes, dst, DefaultLineLength, Base64Default)
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
ConvertUTF16LEToLatin1/Q-u16-zero/short/0001
ConvertUTF16LEToLatin1/Q-u16-zero/short/0007
ConvertUTF16LEToLatin1/Q-u16-zero/short/0008
ConvertUTF16LEToLatin1/Q-u16-zero/short/0009
ConvertUTF16LEToLatin1/Q-u16-zero/short/0015
ConvertUTF16LEToLatin1/Q-u16-zero/short/0016
ConvertUTF16LEToLatin1/Q-u16-zero/short/0017
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0031
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0032
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0033
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0063
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0064
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0065
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0127
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0128
ConvertUTF16LEToLatin1/Q-u16-zero/boundary/0129
ConvertUTF16LEToLatin1/Q-u16-zero/bulk/2048
ConvertUTF16BEToLatin1/Q-u16-zero/short/0001
ConvertUTF16BEToLatin1/Q-u16-zero/short/0007
ConvertUTF16BEToLatin1/Q-u16-zero/short/0008
ConvertUTF16BEToLatin1/Q-u16-zero/short/0009
ConvertUTF16BEToLatin1/Q-u16-zero/short/0015
ConvertUTF16BEToLatin1/Q-u16-zero/short/0016
ConvertUTF16BEToLatin1/Q-u16-zero/short/0017
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0031
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0032
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0033
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0063
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0064
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0065
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0127
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0128
ConvertUTF16BEToLatin1/Q-u16-zero/boundary/0129
ConvertUTF16BEToLatin1/Q-u16-zero/bulk/2048
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0001
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0007
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0008
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0009
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0015
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0016
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/short/0017
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0031
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0032
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0033
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0063
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0064
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0065
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0127
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0128
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/boundary/0129
ConvertUTF16LEToLatin1WithErrors/Q-u16-zero/bulk/2048
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0001
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0007
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0008
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0009
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0015
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0016
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/short/0017
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0031
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0032
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0033
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0063
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0064
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0065
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0127
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0128
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/boundary/0129
ConvertUTF16BEToLatin1WithErrors/Q-u16-zero/bulk/2048
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0001
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0007
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0008
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0009
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0015
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0016
ConvertValidUTF16LEToLatin1/Q-u16-zero/short/0017
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0031
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0032
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0033
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0063
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0064
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0065
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0127
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0128
ConvertValidUTF16LEToLatin1/Q-u16-zero/boundary/0129
ConvertValidUTF16LEToLatin1/Q-u16-zero/bulk/2048
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0001
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0007
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0008
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0009
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0015
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0016
ConvertValidUTF16BEToLatin1/Q-u16-zero/short/0017
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0031
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0032
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0033
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0063
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0064
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0065
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0127
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0128
ConvertValidUTF16BEToLatin1/Q-u16-zero/boundary/0129
ConvertValidUTF16BEToLatin1/Q-u16-zero/bulk/2048
ConvertUTF16LEToUTF32/Q-u16-zero/short/0001
ConvertUTF16LEToUTF32/Q-u16-zero/short/0007
ConvertUTF16LEToUTF32/Q-u16-zero/short/0008
ConvertUTF16LEToUTF32/Q-u16-zero/short/0009
ConvertUTF16LEToUTF32/Q-u16-zero/short/0015
ConvertUTF16LEToUTF32/Q-u16-zero/short/0016
ConvertUTF16LEToUTF32/Q-u16-zero/short/0017
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0031
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0032
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0033
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0063
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0064
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0065
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0127
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0128
ConvertUTF16LEToUTF32/Q-u16-zero/boundary/0129
ConvertUTF16LEToUTF32/Q-u16-zero/bulk/2048
ConvertUTF16BEToUTF32/Q-u16-zero/short/0001
ConvertUTF16BEToUTF32/Q-u16-zero/short/0007
ConvertUTF16BEToUTF32/Q-u16-zero/short/0008
ConvertUTF16BEToUTF32/Q-u16-zero/short/0009
ConvertUTF16BEToUTF32/Q-u16-zero/short/0015
ConvertUTF16BEToUTF32/Q-u16-zero/short/0016
ConvertUTF16BEToUTF32/Q-u16-zero/short/0017
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0031
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0032
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0033
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0063
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0064
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0065
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0127
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0128
ConvertUTF16BEToUTF32/Q-u16-zero/boundary/0129
ConvertUTF16BEToUTF32/Q-u16-zero/bulk/2048
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0001
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0007
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0008
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0009
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0015
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0016
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/short/0017
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0031
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0032
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0033
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0063
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0064
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0065
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0127
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0128
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/boundary/0129
ConvertUTF16LEToUTF32WithErrors/Q-u16-zero/bulk/2048
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0001
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0007
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0008
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0009
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0015
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0016
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/short/0017
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0031
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0032
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0033
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0063
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0064
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0065
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0127
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0128
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/boundary/0129
ConvertUTF16BEToUTF32WithErrors/Q-u16-zero/bulk/2048
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0001
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0007
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0008
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0009
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0015
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0016
ConvertValidUTF16LEToUTF32/Q-u16-zero/short/0017
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0031
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0032
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0033
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0063
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0064
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0065
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0127
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0128
ConvertValidUTF16LEToUTF32/Q-u16-zero/boundary/0129
ConvertValidUTF16LEToUTF32/Q-u16-zero/bulk/2048
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0001
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0007
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0008
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0009
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0015
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0016
ConvertValidUTF16BEToUTF32/Q-u16-zero/short/0017
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0031
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0032
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0033
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0063
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0064
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0065
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0127
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0128
ConvertValidUTF16BEToUTF32/Q-u16-zero/boundary/0129
ConvertValidUTF16BEToUTF32/Q-u16-zero/bulk/2048
UTF32LengthFromUTF16LE/Q-u16-zero/short/0001
UTF32LengthFromUTF16LE/Q-u16-zero/short/0007
UTF32LengthFromUTF16LE/Q-u16-zero/short/0008
UTF32LengthFromUTF16LE/Q-u16-zero/short/0009
UTF32LengthFromUTF16LE/Q-u16-zero/short/0015
UTF32LengthFromUTF16LE/Q-u16-zero/short/0016
UTF32LengthFromUTF16LE/Q-u16-zero/short/0017
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0031
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0032
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0033
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0063
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0064
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0065
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0127
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0128
UTF32LengthFromUTF16LE/Q-u16-zero/boundary/0129
UTF32LengthFromUTF16LE/Q-u16-zero/bulk/2048
UTF32LengthFromUTF16BE/Q-u16-zero/short/0001
UTF32LengthFromUTF16BE/Q-u16-zero/short/0007
UTF32LengthFromUTF16BE/Q-u16-zero/short/0008
UTF32LengthFromUTF16BE/Q-u16-zero/short/0009
UTF32LengthFromUTF16BE/Q-u16-zero/short/0015
UTF32LengthFromUTF16BE/Q-u16-zero/short/0016
UTF32LengthFromUTF16BE/Q-u16-zero/short/0017
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0031
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0032
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0033
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0063
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0064
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0065
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0127
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0128
UTF32LengthFromUTF16BE/Q-u16-zero/boundary/0129
UTF32LengthFromUTF16BE/Q-u16-zero/bulk/2048
ConvertUTF16LEToUTF8/Q-u16-zero/short/0001
ConvertUTF16LEToUTF8/Q-u16-zero/short/0007
ConvertUTF16LEToUTF8/Q-u16-zero/short/0008
ConvertUTF16LEToUTF8/Q-u16-zero/short/0009
ConvertUTF16LEToUTF8/Q-u16-zero/short/0015
ConvertUTF16LEToUTF8/Q-u16-zero/short/0016
ConvertUTF16LEToUTF8/Q-u16-zero/short/0017
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0031
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0032
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0033
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0063
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0064
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0065
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0127
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0128
ConvertUTF16LEToUTF8/Q-u16-zero/boundary/0129
ConvertUTF16LEToUTF8/Q-u16-zero/bulk/2048
ConvertUTF16BEToUTF8/Q-u16-zero/short/0001
ConvertUTF16BEToUTF8/Q-u16-zero/short/0007
ConvertUTF16BEToUTF8/Q-u16-zero/short/0008
ConvertUTF16BEToUTF8/Q-u16-zero/short/0009
ConvertUTF16BEToUTF8/Q-u16-zero/short/0015
ConvertUTF16BEToUTF8/Q-u16-zero/short/0016
ConvertUTF16BEToUTF8/Q-u16-zero/short/0017
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0031
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0032
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0033
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0063
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0064
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0065
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0127
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0128
ConvertUTF16BEToUTF8/Q-u16-zero/boundary/0129
ConvertUTF16BEToUTF8/Q-u16-zero/bulk/2048
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0001
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0007
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0008
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0009
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0015
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0016
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/short/0017
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0031
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0032
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0033
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0063
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0064
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0065
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0127
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0128
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/boundary/0129
ConvertUTF16LEToUTF8WithErrors/Q-u16-zero/bulk/2048
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0001
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0007
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0008
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0009
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0015
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0016
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/short/0017
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0031
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0032
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0033
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0063
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0064
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0065
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0127
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0128
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/boundary/0129
ConvertUTF16BEToUTF8WithErrors/Q-u16-zero/bulk/2048
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0001
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0007
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0008
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0009
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0015
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0016
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/short/0017
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0031
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0032
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0033
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0063
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0064
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0065
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0127
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0128
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/boundary/0129
ConvertUTF16LEToUTF8WithReplacement/Q-u16-zero/bulk/2048
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0001
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0007
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0008
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0009
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0015
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0016
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/short/0017
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0031
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0032
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0033
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0063
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0064
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0065
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0127
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0128
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/boundary/0129
ConvertUTF16BEToUTF8WithReplacement/Q-u16-zero/bulk/2048
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0001
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0007
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0008
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0009
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0015
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0016
ConvertValidUTF16LEToUTF8/Q-u16-zero/short/0017
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0031
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0032
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0033
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0063
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0064
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0065
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0127
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0128
ConvertValidUTF16LEToUTF8/Q-u16-zero/boundary/0129
ConvertValidUTF16LEToUTF8/Q-u16-zero/bulk/2048
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0001
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0007
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0008
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0009
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0015
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0016
ConvertValidUTF16BEToUTF8/Q-u16-zero/short/0017
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0031
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0032
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0033
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0063
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0064
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0065
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0127
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0128
ConvertValidUTF16BEToUTF8/Q-u16-zero/boundary/0129
ConvertValidUTF16BEToUTF8/Q-u16-zero/bulk/2048
UTF8LengthFromUTF16LE/Q-u16-zero/short/0001
UTF8LengthFromUTF16LE/Q-u16-zero/short/0007
UTF8LengthFromUTF16LE/Q-u16-zero/short/0008
UTF8LengthFromUTF16LE/Q-u16-zero/short/0009
UTF8LengthFromUTF16LE/Q-u16-zero/short/0015
UTF8LengthFromUTF16LE/Q-u16-zero/short/0016
UTF8LengthFromUTF16LE/Q-u16-zero/short/0017
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0031
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0032
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0033
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0063
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0064
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0065
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0127
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0128
UTF8LengthFromUTF16LE/Q-u16-zero/boundary/0129
UTF8LengthFromUTF16LE/Q-u16-zero/bulk/2048
UTF8LengthFromUTF16BE/Q-u16-zero/short/0001
UTF8LengthFromUTF16BE/Q-u16-zero/short/0007
UTF8LengthFromUTF16BE/Q-u16-zero/short/0008
UTF8LengthFromUTF16BE/Q-u16-zero/short/0009
UTF8LengthFromUTF16BE/Q-u16-zero/short/0015
UTF8LengthFromUTF16BE/Q-u16-zero/short/0016
UTF8LengthFromUTF16BE/Q-u16-zero/short/0017
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0031
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0032
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0033
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0063
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0064
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0065
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0127
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0128
UTF8LengthFromUTF16BE/Q-u16-zero/boundary/0129
UTF8LengthFromUTF16BE/Q-u16-zero/bulk/2048
ChangeEndiannessUTF16/Q-u16-zero/short/0001
ChangeEndiannessUTF16/Q-u16-zero/short/0007
ChangeEndiannessUTF16/Q-u16-zero/short/0008
ChangeEndiannessUTF16/Q-u16-zero/short/0009
ChangeEndiannessUTF16/Q-u16-zero/short/0015
ChangeEndiannessUTF16/Q-u16-zero/short/0016
ChangeEndiannessUTF16/Q-u16-zero/short/0017
ChangeEndiannessUTF16/Q-u16-zero/boundary/0031
ChangeEndiannessUTF16/Q-u16-zero/boundary/0032
ChangeEndiannessUTF16/Q-u16-zero/boundary/0033
ChangeEndiannessUTF16/Q-u16-zero/boundary/0063
ChangeEndiannessUTF16/Q-u16-zero/boundary/0064
ChangeEndiannessUTF16/Q-u16-zero/boundary/0065
ChangeEndiannessUTF16/Q-u16-zero/boundary/0127
ChangeEndiannessUTF16/Q-u16-zero/boundary/0128
ChangeEndiannessUTF16/Q-u16-zero/boundary/0129
ChangeEndiannessUTF16/Q-u16-zero/bulk/2048
CountUTF16LE/Q-u16-zero/short/0001
CountUTF16LE/Q-u16-zero/short/0007
CountUTF16LE/Q-u16-zero/short/0008
CountUTF16LE/Q-u16-zero/short/0009
CountUTF16LE/Q-u16-zero/short/0015
CountUTF16LE/Q-u16-zero/short/0016
CountUTF16LE/Q-u16-zero/short/0017
CountUTF16LE/Q-u16-zero/boundary/0031
CountUTF16LE/Q-u16-zero/boundary/0032
CountUTF16LE/Q-u16-zero/boundary/0033
CountUTF16LE/Q-u16-zero/boundary/0063
CountUTF16LE/Q-u16-zero/boundary/0064
CountUTF16LE/Q-u16-zero/boundary/0065
CountUTF16LE/Q-u16-zero/boundary/0127
CountUTF16LE/Q-u16-zero/boundary/0128
CountUTF16LE/Q-u16-zero/boundary/0129
CountUTF16LE/Q-u16-zero/bulk/2048
CountUTF16BE/Q-u16-zero/short/0001
CountUTF16BE/Q-u16-zero/short/0007
CountUTF16BE/Q-u16-zero/short/0008
CountUTF16BE/Q-u16-zero/short/0009
CountUTF16BE/Q-u16-zero/short/0015
CountUTF16BE/Q-u16-zero/short/0016
CountUTF16BE/Q-u16-zero/short/0017
CountUTF16BE/Q-u16-zero/boundary/0031
CountUTF16BE/Q-u16-zero/boundary/0032
CountUTF16BE/Q-u16-zero/boundary/0033
CountUTF16BE/Q-u16-zero/boundary/0063
CountUTF16BE/Q-u16-zero/boundary/0064
CountUTF16BE/Q-u16-zero/boundary/0065
CountUTF16BE/Q-u16-zero/boundary/0127
CountUTF16BE/Q-u16-zero/boundary/0128
CountUTF16BE/Q-u16-zero/boundary/0129
CountUTF16BE/Q-u16-zero/bulk/2048
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0001
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0007
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0008
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0009
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0015
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0016
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/short/0017
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0031
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0032
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0033
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0063
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0064
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0065
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0127
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0128
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/boundary/0129
UTF8LengthFromUTF16LEWithReplacement/Q-u16-zero/bulk/2048
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0001
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0007
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0008
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0009
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0015
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0016
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/short/0017
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0031
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0032
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0033
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0063
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0064
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0065
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0127
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0128
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/boundary/0129
UTF8LengthFromUTF16BEWithReplacement/Q-u16-zero/bulk/2048
ConvertUTF32ToLatin1/Q-u32-zero/short/0001
ConvertUTF32ToLatin1/Q-u32-zero/short/0003
ConvertUTF32ToLatin1/Q-u32-zero/short/0004
ConvertUTF32ToLatin1/Q-u32-zero/short/0005
ConvertUTF32ToLatin1/Q-u32-zero/short/0007
ConvertUTF32ToLatin1/Q-u32-zero/short/0008
ConvertUTF32ToLatin1/Q-u32-zero/short/0009
ConvertUTF32ToLatin1/Q-u32-zero/boundary/0015
ConvertUTF32ToLatin1/Q-u32-zero/boundary/0016
ConvertUTF32ToLatin1/Q-u32-zero/boundary/0017
ConvertUTF32ToLatin1/Q-u32-zero/boundary/0031
ConvertUTF32ToLatin1/Q-u32-zero/boundary/0032
ConvertUTF32ToLatin1/Q-u32-zero/boundary/0033
ConvertUTF32ToLatin1/Q-u32-zero/bulk/1024
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0001
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0003
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0004
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0005
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0007
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0008
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/short/0009
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/boundary/0015
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/boundary/0016
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/boundary/0017
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/boundary/0031
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/boundary/0032
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/boundary/0033
ConvertUTF32ToLatin1WithErrors/Q-u32-zero/bulk/1024
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0001
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0003
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0004
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0005
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0007
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0008
ConvertValidUTF32ToLatin1/Q-u32-zero/short/0009
ConvertValidUTF32ToLatin1/Q-u32-zero/boundary/0015
ConvertValidUTF32ToLatin1/Q-u32-zero/boundary/0016
ConvertValidUTF32ToLatin1/Q-u32-zero/boundary/0017
ConvertValidUTF32ToLatin1/Q-u32-zero/boundary/0031
ConvertValidUTF32ToLatin1/Q-u32-zero/boundary/0032
ConvertValidUTF32ToLatin1/Q-u32-zero/boundary/0033
ConvertValidUTF32ToLatin1/Q-u32-zero/bulk/1024
ConvertUTF32ToUTF8/Q-u32-zero/short/0001
ConvertUTF32ToUTF8/Q-u32-zero/short/0003
ConvertUTF32ToUTF8/Q-u32-zero/short/0004
ConvertUTF32ToUTF8/Q-u32-zero/short/0005
ConvertUTF32ToUTF8/Q-u32-zero/short/0007
ConvertUTF32ToUTF8/Q-u32-zero/short/0008
ConvertUTF32ToUTF8/Q-u32-zero/short/0009
ConvertUTF32ToUTF8/Q-u32-zero/boundary/0015
ConvertUTF32ToUTF8/Q-u32-zero/boundary/0016
ConvertUTF32ToUTF8/Q-u32-zero/boundary/0017
ConvertUTF32ToUTF8/Q-u32-zero/boundary/0031
ConvertUTF32ToUTF8/Q-u32-zero/boundary/0032
ConvertUTF32ToUTF8/Q-u32-zero/boundary/0033
ConvertUTF32ToUTF8/Q-u32-zero/bulk/1024
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0001
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0003
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0004
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0005
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0007
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0008
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/short/0009
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/boundary/0015
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/boundary/0016
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/boundary/0017
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/boundary/0031
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/boundary/0032
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/boundary/0033
ConvertUTF32ToUTF8WithErrors/Q-u32-zero/bulk/1024
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0001
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0003
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0004
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0005
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0007
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0008
ConvertValidUTF32ToUTF8/Q-u32-zero/short/0009
ConvertValidUTF32ToUTF8/Q-u32-zero/boundary/0015
ConvertValidUTF32ToUTF8/Q-u32-zero/boundary/0016
ConvertValidUTF32ToUTF8/Q-u32-zero/boundary/0017
ConvertValidUTF32ToUTF8/Q-u32-zero/boundary/0031
ConvertValidUTF32ToUTF8/Q-u32-zero/boundary/0032
ConvertValidUTF32ToUTF8/Q-u32-zero/boundary/0033
ConvertValidUTF32ToUTF8/Q-u32-zero/bulk/1024
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0001
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0003
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0004
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0005
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0007
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0008
ConvertUTF32ToUTF16LE/Q-u32-zero/short/0009
ConvertUTF32ToUTF16LE/Q-u32-zero/boundary/0015
ConvertUTF32ToUTF16LE/Q-u32-zero/boundary/0016
ConvertUTF32ToUTF16LE/Q-u32-zero/boundary/0017
ConvertUTF32ToUTF16LE/Q-u32-zero/boundary/0031
ConvertUTF32ToUTF16LE/Q-u32-zero/boundary/0032
ConvertUTF32ToUTF16LE/Q-u32-zero/boundary/0033
ConvertUTF32ToUTF16LE/Q-u32-zero/bulk/1024
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0001
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0003
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0004
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0005
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0007
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0008
ConvertUTF32ToUTF16BE/Q-u32-zero/short/0009
ConvertUTF32ToUTF16BE/Q-u32-zero/boundary/0015
ConvertUTF32ToUTF16BE/Q-u32-zero/boundary/0016
ConvertUTF32ToUTF16BE/Q-u32-zero/boundary/0017
ConvertUTF32ToUTF16BE/Q-u32-zero/boundary/0031
ConvertUTF32ToUTF16BE/Q-u32-zero/boundary/0032
ConvertUTF32ToUTF16BE/Q-u32-zero/boundary/0033
ConvertUTF32ToUTF16BE/Q-u32-zero/bulk/1024
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0001
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0003
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0004
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0005
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0007
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0008
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/short/0009
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/boundary/0015
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/boundary/0016
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/boundary/0017
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/boundary/0031
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/boundary/0032
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/boundary/0033
ConvertUTF32ToUTF16LEWithErrors/Q-u32-zero/bulk/1024
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0001
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0003
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0004
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0005
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0007
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0008
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/short/0009
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/boundary/0015
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/boundary/0016
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/boundary/0017
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/boundary/0031
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/boundary/0032
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/boundary/0033
ConvertUTF32ToUTF16BEWithErrors/Q-u32-zero/bulk/1024
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0001
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0003
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0004
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0005
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0007
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0008
ConvertValidUTF32ToUTF16LE/Q-u32-zero/short/0009
ConvertValidUTF32ToUTF16LE/Q-u32-zero/boundary/0015
ConvertValidUTF32ToUTF16LE/Q-u32-zero/boundary/0016
ConvertValidUTF32ToUTF16LE/Q-u32-zero/boundary/0017
ConvertValidUTF32ToUTF16LE/Q-u32-zero/boundary/0031
ConvertValidUTF32ToUTF16LE/Q-u32-zero/boundary/0032
ConvertValidUTF32ToUTF16LE/Q-u32-zero/boundary/0033
ConvertValidUTF32ToUTF16LE/Q-u32-zero/bulk/1024
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0001
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0003
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0004
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0005
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0007
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0008
ConvertValidUTF32ToUTF16BE/Q-u32-zero/short/0009
ConvertValidUTF32ToUTF16BE/Q-u32-zero/boundary/0015
ConvertValidUTF32ToUTF16BE/Q-u32-zero/boundary/0016
ConvertValidUTF32ToUTF16BE/Q-u32-zero/boundary/0017
ConvertValidUTF32ToUTF16BE/Q-u32-zero/boundary/0031
ConvertValidUTF32ToUTF16BE/Q-u32-zero/boundary/0032
ConvertValidUTF32ToUTF16BE/Q-u32-zero/boundary/0033
ConvertValidUTF32ToUTF16BE/Q-u32-zero/bulk/1024
UTF8LengthFromUTF32/Q-u32-zero/short/0001
UTF8LengthFromUTF32/Q-u32-zero/short/0003
UTF8LengthFromUTF32/Q-u32-zero/short/0004
UTF8LengthFromUTF32/Q-u32-zero/short/0005
UTF8LengthFromUTF32/Q-u32-zero/short/0007
UTF8LengthFromUTF32/Q-u32-zero/short/0008
UTF8LengthFromUTF32/Q-u32-zero/short/0009
UTF8LengthFromUTF32/Q-u32-zero/boundary/0015
UTF8LengthFromUTF32/Q-u32-zero/boundary/0016
UTF8LengthFromUTF32/Q-u32-zero/boundary/0017
UTF8LengthFromUTF32/Q-u32-zero/boundary/0031
UTF8LengthFromUTF32/Q-u32-zero/boundary/0032
UTF8LengthFromUTF32/Q-u32-zero/boundary/0033
UTF8LengthFromUTF32/Q-u32-zero/bulk/1024
UTF16LengthFromUTF32/Q-u32-zero/short/0001
UTF16LengthFromUTF32/Q-u32-zero/short/0003
UTF16LengthFromUTF32/Q-u32-zero/short/0004
UTF16LengthFromUTF32/Q-u32-zero/short/0005
UTF16LengthFromUTF32/Q-u32-zero/short/0007
UTF16LengthFromUTF32/Q-u32-zero/short/0008
UTF16LengthFromUTF32/Q-u32-zero/short/0009
UTF16LengthFromUTF32/Q-u32-zero/boundary/0015
UTF16LengthFromUTF32/Q-u32-zero/boundary/0016
UTF16LengthFromUTF32/Q-u32-zero/boundary/0017
UTF16LengthFromUTF32/Q-u32-zero/boundary/0031
UTF16LengthFromUTF32/Q-u32-zero/boundary/0032
UTF16LengthFromUTF32/Q-u32-zero/boundary/0033
UTF16LengthFromUTF32/Q-u32-zero/bulk/1024
Find/Q-find-byte/short/0001
Find/Q-find-byte/short/0015
Find/Q-find-byte/short/0016
Find/Q-find-byte/short/0017
Find/Q-find-byte/short/0031
Find/Q-find-byte/short/0032
Find/Q-find-byte/short/0033
Find/Q-find-byte/boundary/0063
Find/Q-find-byte/boundary/0064
Find/Q-find-byte/boundary/0065
Find/Q-find-byte/boundary/0127
Find/Q-find-byte/boundary/0128
Find/Q-find-byte/boundary/0129
Find/Q-find-byte/bulk/4096
FindUTF16/Q-find-u16le/short/0001
FindUTF16/Q-find-u16le/short/0007
FindUTF16/Q-find-u16le/short/0008
FindUTF16/Q-find-u16le/short/0009
FindUTF16/Q-find-u16le/short/0015
FindUTF16/Q-find-u16le/short/0016
FindUTF16/Q-find-u16le/short/0017
FindUTF16/Q-find-u16le/boundary/0031
FindUTF16/Q-find-u16le/boundary/0032
FindUTF16/Q-find-u16le/boundary/0033
FindUTF16/Q-find-u16le/boundary/0063
FindUTF16/Q-find-u16le/boundary/0064
FindUTF16/Q-find-u16le/boundary/0065
FindUTF16/Q-find-u16le/boundary/0127
FindUTF16/Q-find-u16le/boundary/0128
FindUTF16/Q-find-u16le/boundary/0129
FindUTF16/Q-find-u16le/bulk/2048
DetectEncodings/Q-detection-valid/short/0001
DetectEncodings/Q-detection-valid/short/0015
DetectEncodings/Q-detection-valid/short/0016
DetectEncodings/Q-detection-valid/short/0017
DetectEncodings/Q-detection-valid/short/0031
DetectEncodings/Q-detection-valid/short/0032
DetectEncodings/Q-detection-valid/short/0033
DetectEncodings/Q-detection-valid/boundary/0063
DetectEncodings/Q-detection-valid/boundary/0064
DetectEncodings/Q-detection-valid/boundary/0065
DetectEncodings/Q-detection-valid/boundary/0127
DetectEncodings/Q-detection-valid/boundary/0128
DetectEncodings/Q-detection-valid/boundary/0129
DetectEncodings/Q-detection-valid/bulk/4096
BinaryLengthFromBase64/Q-byte-zero/short/0001
BinaryLengthFromBase64/Q-byte-zero/short/0015
BinaryLengthFromBase64/Q-byte-zero/short/0016
BinaryLengthFromBase64/Q-byte-zero/short/0017
BinaryLengthFromBase64/Q-byte-zero/short/0031
BinaryLengthFromBase64/Q-byte-zero/short/0032
BinaryLengthFromBase64/Q-byte-zero/short/0033
BinaryLengthFromBase64/Q-byte-zero/boundary/0063
BinaryLengthFromBase64/Q-byte-zero/boundary/0064
BinaryLengthFromBase64/Q-byte-zero/boundary/0065
BinaryLengthFromBase64/Q-byte-zero/boundary/0127
BinaryLengthFromBase64/Q-byte-zero/boundary/0128
BinaryLengthFromBase64/Q-byte-zero/boundary/0129
BinaryLengthFromBase64/Q-byte-zero/bulk/4096
BinaryLengthFromBase64/Q-emoji/bulk/3150
BinaryToBase64/Q-byte-zero/short/0001
BinaryToBase64/Q-byte-zero/short/0015
BinaryToBase64/Q-byte-zero/short/0016
BinaryToBase64/Q-byte-zero/short/0017
BinaryToBase64/Q-byte-zero/short/0031
BinaryToBase64/Q-byte-zero/short/0032
BinaryToBase64/Q-byte-zero/short/0033
BinaryToBase64/Q-byte-zero/boundary/0063
BinaryToBase64/Q-byte-zero/boundary/0064
BinaryToBase64/Q-byte-zero/boundary/0065
BinaryToBase64/Q-byte-zero/boundary/0127
BinaryToBase64/Q-byte-zero/boundary/0128
BinaryToBase64/Q-byte-zero/boundary/0129
BinaryToBase64/Q-byte-zero/bulk/4096
BinaryToBase64/Q-emoji/bulk/3150
BinaryToBase64WithLines/Q-byte-zero/short/0001
BinaryToBase64WithLines/Q-byte-zero/short/0015
BinaryToBase64WithLines/Q-byte-zero/short/0016
BinaryToBase64WithLines/Q-byte-zero/short/0017
BinaryToBase64WithLines/Q-byte-zero/short/0031
BinaryToBase64WithLines/Q-byte-zero/short/0032
BinaryToBase64WithLines/Q-byte-zero/short/0033
BinaryToBase64WithLines/Q-byte-zero/boundary/0063
BinaryToBase64WithLines/Q-byte-zero/boundary/0064
BinaryToBase64WithLines/Q-byte-zero/boundary/0065
BinaryToBase64WithLines/Q-byte-zero/boundary/0127
BinaryToBase64WithLines/Q-byte-zero/boundary/0128
BinaryToBase64WithLines/Q-byte-zero/boundary/0129
BinaryToBase64WithLines/Q-byte-zero/bulk/4096
BinaryToBase64WithLines/Q-emoji/bulk/3150
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0001
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0015
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0016
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0017
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0031
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0032
BinaryLengthFromBase64UTF16/Q-byte-zero/short/0033
BinaryLengthFromBase64UTF16/Q-byte-zero/boundary/0063
BinaryLengthFromBase64UTF16/Q-byte-zero/boundary/0064
BinaryLengthFromBase64UTF16/Q-byte-zero/boundary/0065
BinaryLengthFromBase64UTF16/Q-byte-zero/boundary/0127
BinaryLengthFromBase64UTF16/Q-byte-zero/boundary/0128
BinaryLengthFromBase64UTF16/Q-byte-zero/boundary/0129
BinaryLengthFromBase64UTF16/Q-byte-zero/bulk/4096
BinaryLengthFromBase64UTF16/Q-emoji/bulk/3150
Base64ToBinary/Q-dns-normalized/bulk/35000000
Base64ToBinaryDetails/Q-dns-normalized/bulk/35000000
Base64ToBinaryUTF16/Q-dns-normalized/bulk/35000000
Base64ToBinaryDetailsUTF16/Q-dns-normalized/bulk/35000000
`

func TestDispatchQualificationSurface(t *testing.T) {
	rows := dispatchQualificationRows()
	wantNames := strings.Fields(dispatchQualificationExpectedNames)
	if len(rows) != 1359 || len(wantNames) != 1359 {
		t.Fatalf("row counts = (%d, %d), want (1359, 1359)", len(rows), len(wantNames))
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
	findByte, findU16LE, findU16LERaw := materializeDispatchQualificationFindCorpora()
	if got := fmt.Sprintf("%x", sha256.Sum256(findByte)); got != dispatchQualificationFindByteSHA256 {
		t.Fatalf("Q-find-byte SHA-256 = %s, want %s", got, dispatchQualificationFindByteSHA256)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(findU16LERaw)); got != dispatchQualificationFindU16LESHA256 {
		t.Fatalf("Q-find-u16le SHA-256 = %s, want %s", got, dispatchQualificationFindU16LESHA256)
	}
	encodedFind := make([]byte, 2*len(findU16LE))
	for i, codeUnit := range findU16LE {
		binary.LittleEndian.PutUint16(encodedFind[2*i:], codeUnit)
	}
	if !reflect.DeepEqual(encodedFind, findU16LERaw) {
		t.Fatal("Q-find-u16le is not the little-endian encoding of its code units")
	}
	dnsNormalized := loadDispatchQualificationDNSNormalized()
	if got := fmt.Sprintf("%x", sha256.Sum256(dnsNormalized)); got != dispatchQualificationDNSNormalizedSHA256 {
		t.Fatalf("Q-dns-normalized SHA-256 = %s, want %s", got, dispatchQualificationDNSNormalizedSHA256)
	}
	if len(dnsNormalized) != dispatchQualificationDNSNormalizedSize {
		t.Fatalf("Q-dns-normalized length = %d, want %d", len(dnsNormalized), dispatchQualificationDNSNormalizedSize)
	}
	for i, row := range dispatchQualificationRows() {
		wantBytes := int64(row.size)
		if row.uint16s != nil {
			wantBytes *= 2
		}
		if row.uint32s != nil {
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
		if row.corpus == "Q-find-byte" && (row.size > len(findByte) || row.size < 1) {
			t.Fatalf("row %d (%s) invalid Q-find-byte size %d", i, row.name(), row.size)
		}
		if row.corpus == "Q-detection-valid" && (row.size > len(findByte) || row.size < 1) {
			t.Fatalf("row %d (%s) invalid Q-detection-valid size %d", i, row.name(), row.size)
		}
		if row.corpus == "Q-find-u16le" && (row.size > len(findU16LE) || row.size < 1) {
			t.Fatalf("row %d (%s) invalid Q-find-u16le size %d", i, row.name(), row.size)
		}
		if row.corpus == "Q-dns-normalized" && row.size != dispatchQualificationDNSNormalizedSize {
			t.Fatalf("row %d (%s) input length = %d, want %d", i, row.name(), row.size, dispatchQualificationDNSNormalizedSize)
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
		"benchmarkIntSink = ConvertUTF16LEToLatin1(row.uint16s, dst)",
		"benchmarkIntSink = ConvertUTF16BEToLatin1(row.uint16s, dst)",
		"benchmarkResultSink = ConvertUTF16LEToLatin1WithErrors(row.uint16s, dst)",
		"benchmarkResultSink = ConvertUTF16BEToLatin1WithErrors(row.uint16s, dst)",
		"benchmarkIntSink = ConvertValidUTF16LEToLatin1(row.uint16s, dst)",
		"benchmarkIntSink = ConvertValidUTF16BEToLatin1(row.uint16s, dst)",
		"benchmarkIntSink = ConvertUTF16LEToUTF32(row.uint16s, dst)",
		"benchmarkIntSink = ConvertUTF16BEToUTF32(row.uint16s, dst)",
		"benchmarkResultSink = ConvertUTF16LEToUTF32WithErrors(row.uint16s, dst)",
		"benchmarkResultSink = ConvertUTF16BEToUTF32WithErrors(row.uint16s, dst)",
		"benchmarkIntSink = ConvertValidUTF16LEToUTF32(row.uint16s, dst)",
		"benchmarkIntSink = ConvertValidUTF16BEToUTF32(row.uint16s, dst)",
		"benchmarkIntSink = UTF32LengthFromUTF16LE(row.uint16s)",
		"benchmarkIntSink = UTF32LengthFromUTF16BE(row.uint16s)",
		"benchmarkIntSink = ConvertUTF16LEToUTF8(row.uint16s, dst)",
		"benchmarkIntSink = ConvertUTF16BEToUTF8(row.uint16s, dst)",
		"benchmarkResultSink = ConvertUTF16LEToUTF8WithErrors(row.uint16s, dst)",
		"benchmarkResultSink = ConvertUTF16BEToUTF8WithErrors(row.uint16s, dst)",
		"benchmarkIntSink = ConvertUTF16LEToUTF8WithReplacement(row.uint16s, dst)",
		"benchmarkIntSink = ConvertUTF16BEToUTF8WithReplacement(row.uint16s, dst)",
		"benchmarkIntSink = ConvertValidUTF16LEToUTF8(row.uint16s, dst)",
		"benchmarkIntSink = ConvertValidUTF16BEToUTF8(row.uint16s, dst)",
		"benchmarkIntSink = UTF8LengthFromUTF16LE(row.uint16s)",
		"benchmarkIntSink = UTF8LengthFromUTF16BE(row.uint16s)",
		"ChangeEndiannessUTF16(row.uint16s, dst)",
		"benchmarkIntSink = CountUTF16LE(row.uint16s)",
		"benchmarkIntSink = CountUTF16BE(row.uint16s)",
		"benchmarkResultSink = UTF8LengthFromUTF16LEWithReplacement(row.uint16s)",
		"benchmarkResultSink = UTF8LengthFromUTF16BEWithReplacement(row.uint16s)",
		"benchmarkIntSink = ConvertUTF32ToLatin1(row.uint32s, dst)",
		"benchmarkResultSink = ConvertUTF32ToLatin1WithErrors(row.uint32s, dst)",
		"benchmarkIntSink = ConvertValidUTF32ToLatin1(row.uint32s, dst)",
		"benchmarkIntSink = ConvertUTF32ToUTF8(row.uint32s, dst)",
		"benchmarkResultSink = ConvertUTF32ToUTF8WithErrors(row.uint32s, dst)",
		"benchmarkIntSink = ConvertValidUTF32ToUTF8(row.uint32s, dst)",
		"benchmarkIntSink = ConvertUTF32ToUTF16LE(row.uint32s, dst)",
		"benchmarkIntSink = ConvertUTF32ToUTF16BE(row.uint32s, dst)",
		"benchmarkResultSink = ConvertUTF32ToUTF16LEWithErrors(row.uint32s, dst)",
		"benchmarkResultSink = ConvertUTF32ToUTF16BEWithErrors(row.uint32s, dst)",
		"benchmarkIntSink = ConvertValidUTF32ToUTF16LE(row.uint32s, dst)",
		"benchmarkIntSink = ConvertValidUTF32ToUTF16BE(row.uint32s, dst)",
		"benchmarkIntSink = UTF8LengthFromUTF32(row.uint32s)",
		"benchmarkIntSink = UTF16LengthFromUTF32(row.uint32s)",
		"benchmarkIntSink = Find(row.bytes, needle)",
		"benchmarkIntSink = FindUTF16(row.uint16s, needle)",
		"benchmarkEncodingSink = DetectEncodings(row.bytes)",
		"benchmarkIntSink = BinaryLengthFromBase64(row.bytes)",
		"benchmarkIntSink = BinaryLengthFromBase64UTF16(row.uint16s)",
		"benchmarkResultSink = Base64ToBinary(row.bytes, dst, Base64Default, Loose)",
		"benchmarkResultSink = Base64ToBinaryUTF16(row.uint16s, dst, Base64Default, Loose)",
		"benchmarkFullResultSink = Base64ToBinaryDetails(row.bytes, dst, Base64Default, Loose)",
		"benchmarkFullResultSink = Base64ToBinaryDetailsUTF16(row.uint16s, dst, Base64Default, Loose)",
		"benchmarkIntSink = BinaryToBase64(row.bytes, dst, Base64Default)",
		"benchmarkIntSink = BinaryToBase64WithLines(row.bytes, dst, DefaultLineLength, Base64Default)",
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
