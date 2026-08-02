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

//go:build amd64

package simdutf

import (
	"testing"

	"golang.org/x/sys/cpu"
)

// These are hand-authored Go-only dispatch-safety tests. They exercise the
// golang.org/x/sys/cpu mapping and the independent LZCNT probe implementing
// the feature contract pinned to
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/simdutf/westmere.h and src/simdutf/haswell.h. They are not upstream test
// vectors and do not reproduce include/simdutf/internal/isadetection.h.

type cpuidInput struct {
	eax uint32
	ecx uint32
}

type cpuidOutput struct {
	eax uint32
	ebx uint32
	ecx uint32
	edx uint32
}

func fakeCPUID(t *testing.T, outputs map[cpuidInput]cpuidOutput) cpuidFunc {
	t.Helper()
	return func(eax, ecx uint32) (uint32, uint32, uint32, uint32) {
		t.Helper()
		out, ok := outputs[cpuidInput{eax: eax, ecx: ecx}]
		if !ok {
			t.Fatalf("unexpected CPUID leaf %#x subleaf %#x", eax, ecx)
		}
		return out.eax, out.ebx, out.ecx, out.edx
	}
}

func TestHasLZCNT(t *testing.T) {
	tests := map[string]struct {
		outputs map[cpuidInput]cpuidOutput
		want    bool
	}{
		"extended leaf 1 unavailable": {
			outputs: map[cpuidInput]cpuidOutput{
				{eax: 0x80000000}: {eax: 0x80000000},
			},
		},
		"LZCNT bit clear": {
			outputs: map[cpuidInput]cpuidOutput{
				{eax: 0x80000000}: {eax: 0x80000001},
				{eax: 0x80000001}: {},
			},
		},
		"LZCNT bit set": {
			outputs: map[cpuidInput]cpuidOutput{
				{eax: 0x80000000}: {eax: 0x80000001},
				{eax: 0x80000001}: {ecx: 1 << 5},
			},
			want: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := hasLZCNT(fakeCPUID(t, test.outputs)); got != test.want {
				t.Fatalf("hasLZCNT = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDetectAMD64FeaturesLiveHost(t *testing.T) {
	known := cpuSSSE3 | cpuSSE42 | cpuPOPCNT | cpuAVX2 | cpuBMI1 | cpuBMI2 | cpuLZCNT
	got := detectAMD64Features()
	if got & ^known != 0 {
		t.Fatalf("features = %#x, contains unknown amd64 feature bits %#x", got, got & ^known)
	}
	for _, bit := range []struct {
		name    string
		feature cpuFeatures
		oracle  bool
	}{
		{name: "SSSE3", feature: cpuSSSE3, oracle: cpu.X86.HasSSSE3},
		{name: "SSE4.2", feature: cpuSSE42, oracle: cpu.X86.HasSSE42},
		{name: "POPCNT", feature: cpuPOPCNT, oracle: cpu.X86.HasPOPCNT},
		{name: "BMI1", feature: cpuBMI1, oracle: cpu.X86.HasBMI1},
		{name: "BMI2", feature: cpuBMI2, oracle: cpu.X86.HasBMI2},
		{name: "AVX2", feature: cpuAVX2, oracle: cpu.X86.HasAVX2},
		{name: "LZCNT", feature: cpuLZCNT, oracle: hasLZCNT(cpuid)},
	} {
		if has := got&bit.feature != 0; has != bit.oracle {
			t.Fatalf("feature %s = %v, want %v per golang.org/x/sys/cpu", bit.name, has, bit.oracle)
		}
	}
}
