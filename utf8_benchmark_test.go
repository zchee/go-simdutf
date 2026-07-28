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

// Benchmarks mapped from simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// benchmarks/shortbench.cpp:29-40,419-422,493-497,520-526 and
// benchmarks/src/benchmark.cpp:611-645. ValidateUTF8 keeps shortbench's frozen
// zero prefixes; ValidateUTF8WithErrors uses the exact pinned
// benchmarks/dataset/emoji.txt corpus. Public and scalar rows deliberately
// share identical corpus setup and benchmark names for a scalar baseline.

func BenchmarkValidateUTF8(b *testing.B) {
	corpus := materializeShortbenchZero128()
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err != nil {
		b.Fatal(err)
	}
	b.Run("shortbench-zero-128", func(b *testing.B) {
		for _, prefix := range shortbenchZero128Spec.prefixes {
			input := corpus[:prefix]
			b.Run(fmt.Sprintf("%03dB", len(input)), func(b *testing.B) {
				b.Run("public", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkBoolSink = ValidateUTF8(input)
					}
				})
				b.Run("scalar", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkBoolSink = validateUTF8Scalar(input)
					}
				})
				for _, candidate := range utf8DirectVariants {
					if !candidate.validate.supportedBy(detectSelectionInput()) {
						continue
					}
					b.Run(candidate.name, func(b *testing.B) {
						b.ReportAllocs()
						b.SetBytes(int64(len(input)))
						for b.Loop() {
							benchmarkBoolSink = candidate.validate.value(input)
						}
					})
				}
			})
		}
	})
}

func BenchmarkValidateUTF8WithErrors(b *testing.B) {
	input := upstreamEmojiUTF8
	if err := checkBenchmarkCorpus(upstreamEmojiUTF8Spec, input); err != nil {
		b.Fatal(err)
	}
	b.Run("main-upstream-emoji-utf8", func(b *testing.B) {
		b.Run("3150B", func(b *testing.B) {
			b.Run("public", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkResultSink = ValidateUTF8WithErrors(input)
				}
			})
			b.Run("scalar", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkResultSink = validateUTF8WithErrorsScalar(input)
				}
			})
			for _, candidate := range utf8DirectVariants {
				if !candidate.withErrors.supportedBy(detectSelectionInput()) {
					continue
				}
				b.Run(candidate.name, func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkResultSink = candidate.withErrors.value(input)
					}
				})
			}
		})
	})
}
