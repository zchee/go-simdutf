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

// Hand-authored Go-only benchmarks pinned to
// simdutf/simdutf@c7bef0ff14a13fd6ea52e3347da2c659383392de:
// benchmarks/shortbench.cpp:29-35,419-422,493-497,520-526;
// benchmarks/src/benchmark.cpp:120-127,697-715; and
// docs/porting/benchmark-contract.md. The ValidateASCII benchmark maps the
// shortbench validate_ascii registration and zero-buffer prefix loop. The
// ValidateASCIIWithErrors benchmark maps only the main benchmark registration
// and runner; main-zero-128 is a procedure label, not a corpus ID.
// Public rows call the exported functions literally so their wrappers remain
// inlineable; direct diagnostic rows intentionally use registry indirection.

func BenchmarkValidateASCII(b *testing.B) {
	corpus := materializeShortbenchZero128()
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err != nil {
		b.Fatal(err)
	}
	selection := asciiBenchmarkSelectionInput

	b.Run("shortbench-zero-128", func(b *testing.B) {
		for _, prefix := range shortbenchZero128Spec.prefixes {
			input := corpus[:prefix]
			b.Run(fmt.Sprintf("%03dB", len(input)), func(b *testing.B) {
				for _, candidate := range validateASCIIBenchmarkVariants {
					b.Run(candidate.name, func(b *testing.B) {
						b.ReportAllocs()
						b.SetBytes(int64(len(input)))
						if candidate.name == "public" {
							for b.Loop() {
								benchmarkBoolSink = ValidateASCII(input)
							}
							return
						}
						fn := candidate.value
						if fn == nil || !candidate.variant.supportedBy(selection) {
							b.Skip("direct variant is unavailable or unsupported")
						}
						for b.Loop() {
							benchmarkBoolSink = fn(input)
						}
					})
				}
			})
		}
	})
}

func BenchmarkValidateASCIIWithErrors(b *testing.B) {
	corpus := materializeShortbenchZero128()
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err != nil {
		b.Fatal(err)
	}
	selection := asciiBenchmarkSelectionInput
	input := corpus

	b.Run("main-zero-128", func(b *testing.B) {
		b.Run("128B", func(b *testing.B) {
			for _, candidate := range validateASCIIWithErrorsBenchmarkVariants {
				b.Run(candidate.name, func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					if candidate.name == "public" {
						for b.Loop() {
							benchmarkResultSink = ValidateASCIIWithErrors(input)
						}
						return
					}
					fn := candidate.value
					if fn == nil || !candidate.variant.supportedBy(selection) {
						b.Skip("direct variant is unavailable or unsupported")
					}
					for b.Loop() {
						benchmarkResultSink = fn(input)
					}
				})
			}
		})
	})
}
