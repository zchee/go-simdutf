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
	_ "embed"
	"errors"
	"fmt"
	"testing"
)

// Hand-authored Go-only benchmark scaffolding pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b,
// benchmarks/shortbench.cpp:419-422,493-497,520-526, and
// docs/porting/benchmark-contract.md. This file defines corpus and accounting
// helpers only; it adds no product behavior, upstream algorithm vectors,
// Benchmark function, or benchmark result. Phase 2 benchmarks must keep their
// b.Loop and per-iteration operation visibly local rather than use a generic
// closure helper.

const (
	shortbenchZero128SHA256 = "38723a2e5e8a17aa7950dc008209944e898f69a7bd10a23c839d341e935fd5ca"
	upstreamEmojiUTF8SHA256 = "d6484d359bff183e4d6a4d20b3cc7056c55f372011f28b21b06462ba4d643523"
)

//go:embed testdata/upstream/emoji.txt
var upstreamEmojiUTF8 []byte

var shortbenchPrefixSizes = [...]int{1, 11, 21, 31, 41, 51, 61, 71, 81, 91, 101, 111, 121}

type benchmarkCorpusSpec struct {
	name     string
	size     int
	sha256   string
	prefixes []int
}

var shortbenchZero128Spec = benchmarkCorpusSpec{
	name:     "shortbench-zero-128",
	size:     128,
	sha256:   shortbenchZero128SHA256,
	prefixes: shortbenchPrefixSizes[:],
}

// Frozen exact bytes from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// benchmarks/dataset/emoji.txt. The embed and specification keep file loading
// and integrity verification outside benchmark b.Loop bodies.
var upstreamEmojiUTF8Spec = benchmarkCorpusSpec{
	name:   "upstream-emoji-utf8",
	size:   3150,
	sha256: upstreamEmojiUTF8SHA256,
}

func materializeShortbenchZero128() []byte {
	return make([]byte, shortbenchZero128Spec.size)
}

func checkBenchmarkCorpus(spec benchmarkCorpusSpec, corpus []byte) error {
	if spec.name == "" || spec.size < 0 || spec.sha256 == "" {
		return errors.New("invalid benchmark corpus specification")
	}
	if len(corpus) != spec.size {
		return fmt.Errorf("benchmark corpus %q length = %d, want %d", spec.name, len(corpus), spec.size)
	}
	gotHash := fmt.Sprintf("%x", sha256.Sum256(corpus))
	if gotHash != spec.sha256 {
		return fmt.Errorf("benchmark corpus %q SHA-256 = %s, want %s", spec.name, gotHash, spec.sha256)
	}
	previous := 0
	for _, prefix := range spec.prefixes {
		if prefix <= previous || prefix > spec.size {
			return fmt.Errorf("benchmark corpus %q has invalid prefix size %d", spec.name, prefix)
		}
		previous = prefix
	}
	return nil
}

type benchmarkInput interface {
	inputBytes() int
}

type benchmarkBytes []byte

func (input benchmarkBytes) inputBytes() int { return len(input) }

type benchmarkUint16s []uint16

func (input benchmarkUint16s) inputBytes() int { return 2 * len(input) }

type benchmarkUint32s []uint32

func (input benchmarkUint32s) inputBytes() int { return 4 * len(input) }

func benchmarkInputBytes(input benchmarkInput) int { return input.inputBytes() }

var benchmarkBoolSink bool

var benchmarkResultSink Result

var benchmarkIntSink int

func TestShortbenchZero128Corpus(t *testing.T) {
	corpus := materializeShortbenchZero128()
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus) != 128 {
		t.Fatalf("corpus length = %d, want 128", len(corpus))
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(corpus)); got != shortbenchZero128SHA256 {
		t.Fatalf("corpus SHA-256 = %s, want %s", got, shortbenchZero128SHA256)
	}
}

func TestShortbenchPrefixSizes(t *testing.T) {
	want := [...]int{1, 11, 21, 31, 41, 51, 61, 71, 81, 91, 101, 111, 121}
	if shortbenchPrefixSizes != want {
		t.Fatalf("shortbench prefix sizes = %v, want %v", shortbenchPrefixSizes, want)
	}
}

func TestUpstreamEmojiUTF8Corpus(t *testing.T) {
	if err := checkBenchmarkCorpus(upstreamEmojiUTF8Spec, upstreamEmojiUTF8); err != nil {
		t.Fatal(err)
	}
}

func TestBenchmarkInputBytes(t *testing.T) {
	if got := benchmarkInputBytes(make(benchmarkBytes, 7)); got != 7 {
		t.Errorf("byte denominator = %d, want 7", got)
	}
	if got := benchmarkInputBytes(make(benchmarkUint16s, 7)); got != 14 {
		t.Errorf("uint16 denominator = %d, want 14", got)
	}
	if got := benchmarkInputBytes(make(benchmarkUint32s, 7)); got != 28 {
		t.Errorf("uint32 denominator = %d, want 28", got)
	}
}

func TestCheckBenchmarkCorpusRejectsInvalidCorpus(t *testing.T) {
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, make([]byte, 127)); err == nil {
		t.Error("short corpus was accepted")
	}
	corpus := materializeShortbenchZero128()
	corpus[0] = 1
	if err := checkBenchmarkCorpus(shortbenchZero128Spec, corpus); err == nil {
		t.Error("corpus with wrong digest was accepted")
	}
}

// Hand-authored Go-only benchmarks pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
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

// Benchmark mapped from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// benchmarks/shortbench.cpp:29-40,66-72,419-422,493-497,520-526 and
// benchmarks/src/benchmark.cpp:3428-3443. CountUTF8 keeps shortbench's frozen
// zero prefixes and the pinned benchmarks/dataset/emoji.txt main corpus.
// Public and scalar rows deliberately share identical corpus setup and names;
// later direct variants register through test-only scaffolding.

func BenchmarkCountUTF8(b *testing.B) {
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
						benchmarkIntSink = CountUTF8(input)
					}
				})
				b.Run("scalar", func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = countUTF8Scalar(input)
					}
				})
				for _, candidate := range countUTF8DirectVariants {
					if !candidate.supportedBy(selection) {
						continue
					}
					b.Run(candidate.name, func(b *testing.B) {
						b.ReportAllocs()
						b.SetBytes(int64(len(input)))
						for b.Loop() {
							benchmarkIntSink = candidate.value(input)
						}
					})
				}
			})
		}
	})

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
					benchmarkIntSink = CountUTF8(input)
				}
			})
			b.Run("scalar", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input)))
				for b.Loop() {
					benchmarkIntSink = countUTF8Scalar(input)
				}
			})
			for _, candidate := range countUTF8DirectVariants {
				if !candidate.supportedBy(selection) {
					continue
				}
				b.Run(candidate.name, func(b *testing.B) {
					b.ReportAllocs()
					b.SetBytes(int64(len(input)))
					for b.Loop() {
						benchmarkIntSink = candidate.value(input)
					}
				})
			}
		})
	})
}

// Benchmarks mapped from simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
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

// Benchmarks mapped from
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b (tree
// c8292790d793212ca0a1faf6ae42e7f8e7b70d4f):
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
