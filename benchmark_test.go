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
