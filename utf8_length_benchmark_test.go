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
	"fmt"
	"testing"
)

// Benchmarks mapped from
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de (tree
// 4cbac4c5d1ce0d7f98cc35360d53725433f12811):
// benchmarks/shortbench.cpp:29-65,419-422,493-497,520-526 and
// benchmarks/src/benchmark.cpp:167-169,999-1011. Pinned shortbench registers
// UTF16LengthFromUTF8 and UTF32LengthFromUTF8; the main benchmark registers
// only UTF16LengthFromUTF8 and processes input_data.size()/4 input bytes.
// Public, direct-dispatch, and scalar rows share identical setup and input-byte
// denominators. Latin1LengthFromUTF8 and TrimPartialUTF8 have no registered
// standalone pinned benchmark procedure.
//
// The dispatch-boundary and UTF-32 emoji comparisons are Go-only diagnostics
// over the same frozen corpora.

func BenchmarkUTF16LengthFromUTF8(b *testing.B) {
	corpus := materializeShortbenchZero128()
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err != nil {
		b.Fatal(err)
	}
	selection := detectSelectionInput()
	b.Run("shortbench-zero-128", func(b *testing.B) {
		for _, prefix := range shortbenchZero128Spec.prefixes {
			input := corpus[:prefix]
			b.Run(fmt.Sprintf("%03dB", len(input)), func(b *testing.B) {
				b.Run("public", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = UTF16LengthFromUTF8(input)
					}
				})
				b.Run("direct", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = activeImplementation.utf16LengthFromUTF8(input)
					}
				})
				b.Run("scalar", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = utf16LengthFromUTF8Scalar(input)
					}
				})
				for _, candidate := range utf8LengthDirectVariants {
					if !candidate.utf16.supportedBy(selection) {
						continue
					}
					b.Run(candidate.name, func(b *testing.B) {
						b.ReportAllocs()
						b.SetBytes(int64(len(input)))
						for b.Loop() {
							benchmarkIntSink = candidate.utf16.value(input)
						}
					})
				}
			})
		}
	})

	b.Run("go-only-dispatch-boundary-zero-128", func(b *testing.B) {
		for _, length := range [...]int{0, 1, 11, 15, 16, 17, 21} {
			input := corpus[:length]
			b.Run(fmt.Sprintf("%03dB", len(input)), func(b *testing.B) {
				b.Run("public", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = UTF16LengthFromUTF8(input)
					}
				})
				b.Run("scalar", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = utf16LengthFromUTF8Scalar(input)
					}
				})
			})
		}
	})

	if err := checkBenchmarkCorpus(upstreamEmojiUTF8Spec, upstreamEmojiUTF8); err != nil {
		b.Fatal(err)
	}
	input := upstreamEmojiUTF8[:len(upstreamEmojiUTF8)/4]
	b.Run("main-upstream-emoji-utf8", func(b *testing.B) {
		b.Run(fmt.Sprintf("%04dB", len(input)), func(b *testing.B) {
			b.Run("public", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkIntSink = UTF16LengthFromUTF8(input)
				}
			})
			b.Run("direct", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkIntSink = activeImplementation.utf16LengthFromUTF8(input)
				}
			})
			b.Run("scalar", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkIntSink = utf16LengthFromUTF8Scalar(input)
				}
			})
			for _, candidate := range utf8LengthDirectVariants {
				if !candidate.utf16.supportedBy(selection) {
					continue
				}
				b.Run(candidate.name, func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = candidate.utf16.value(input)
					}
				})
			}
		})
	})
}

func BenchmarkUTF32LengthFromUTF8(b *testing.B) {
	corpus := materializeShortbenchZero128()
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err != nil {
		b.Fatal(err)
	}
	selection := detectSelectionInput()
	b.Run("shortbench-zero-128", func(b *testing.B) {
		for _, prefix := range shortbenchZero128Spec.prefixes {
			input := corpus[:prefix]
			b.Run(fmt.Sprintf("%03dB", len(input)), func(b *testing.B) {
				b.Run("public", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = UTF32LengthFromUTF8(input)
					}
				})
				b.Run("direct", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = activeImplementation.utf32LengthFromUTF8(input)
					}
				})
				b.Run("scalar", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = utf32LengthFromUTF8Scalar(input)
					}
				})
				for _, candidate := range utf8LengthDirectVariants {
					if !candidate.utf32.supportedBy(selection) {
						continue
					}
					b.Run(candidate.name, func(b *testing.B) {
						b.ReportAllocs()
						b.SetBytes(int64(len(input)))
						for b.Loop() {
							benchmarkIntSink = candidate.utf32.value(input)
						}
					})
				}
			})
		}
	})

	b.Run("go-only-dispatch-boundary-zero-128", func(b *testing.B) {
		for _, length := range [...]int{0, 1, 11, 61, 63, 64, 65, 71} {
			input := corpus[:length]
			b.Run(fmt.Sprintf("%03dB", len(input)), func(b *testing.B) {
				b.Run("public", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = UTF32LengthFromUTF8(input)
					}
				})
				b.Run("scalar", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = utf32LengthFromUTF8Scalar(input)
					}
				})
			})
		}
	})

	if err := checkBenchmarkCorpus(upstreamEmojiUTF8Spec, upstreamEmojiUTF8); err != nil {
		b.Fatal(err)
	}
	input := upstreamEmojiUTF8[:len(upstreamEmojiUTF8)/4]
	b.Run("go-only-upstream-emoji-utf8", func(b *testing.B) {
		b.Run(fmt.Sprintf("%04dB", len(input)), func(b *testing.B) {
			b.Run("public", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkIntSink = UTF32LengthFromUTF8(input)
				}
			})
			b.Run("scalar", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkIntSink = utf32LengthFromUTF8Scalar(input)
				}
			})
		})
	})
}
