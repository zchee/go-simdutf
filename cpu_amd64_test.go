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

import "testing"

// These are hand-authored Go-only dispatch-safety tests. They exercise an
// independent probe implementing the feature contract pinned to
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
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

func panicXGETBV() (uint32, uint32) {
	panic("unexpected XGETBV")
}

func TestDetectAMD64FeaturesLeafBounds(t *testing.T) {
	t.Run("basic leaf 1 unavailable", func(t *testing.T) {
		got := detectAMD64FeaturesWith(fakeCPUID(t, map[cpuidInput]cpuidOutput{
			{eax: 0}:          {eax: 0},
			{eax: 0x80000000}: {eax: 0x80000000},
		}), panicXGETBV)
		if got != 0 {
			t.Fatalf("features = %#x, want 0", got)
		}
	})

	t.Run("basic leaf 7 unavailable", func(t *testing.T) {
		got := detectAMD64FeaturesWith(fakeCPUID(t, map[cpuidInput]cpuidOutput{
			{eax: 0}:          {eax: 6},
			{eax: 1}:          {},
			{eax: 0x80000000}: {eax: 0x80000000},
		}), panicXGETBV)
		if got != 0 {
			t.Fatalf("features = %#x, want 0", got)
		}
	})

	t.Run("extended leaf 1 unavailable", func(t *testing.T) {
		got := detectAMD64FeaturesWith(fakeCPUID(t, map[cpuidInput]cpuidOutput{
			{eax: 0}:          {eax: 1},
			{eax: 1}:          {},
			{eax: 0x80000000}: {eax: 0x80000000},
		}), panicXGETBV)
		if got != 0 {
			t.Fatalf("features = %#x, want 0", got)
		}
	})
}

func TestDetectAMD64FeaturesXGETBVGuard(t *testing.T) {
	for _, test := range []struct {
		name     string
		leaf1ECX uint32
	}{
		{name: "AVX missing", leaf1ECX: 1 << 27},
		{name: "OSXSAVE missing", leaf1ECX: 1 << 28},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := detectAMD64FeaturesWith(fakeCPUID(t, completeCPUID(test.leaf1ECX, 1<<5, 0)), panicXGETBV)
			if got&cpuAVX2 != 0 {
				t.Fatalf("features = %#x, unexpectedly includes AVX2", got)
			}
		})
	}
}

func TestDetectAMD64FeaturesAVX2RequiresCPUIDLeaf7(t *testing.T) {
	const leaf1 = 1<<27 | 1<<28
	xgetbvCalled := false
	got := detectAMD64FeaturesWith(fakeCPUID(t, completeCPUID(leaf1, 0, 0)), func() (uint32, uint32) {
		xgetbvCalled = true
		return 0x6, 0
	})
	if got&cpuAVX2 != 0 {
		t.Fatalf("features = %#x, unexpectedly includes AVX2", got)
	}
	if xgetbvCalled {
		t.Fatal("XGETBV called without raw CPUID AVX2 support")
	}
}

func TestDetectAMD64FeaturesAVX2OSState(t *testing.T) {
	for _, test := range []struct {
		name     string
		leaf1ECX uint32
		xcr0     uint32
		want     bool
	}{
		{name: "AVX missing", leaf1ECX: 1 << 27, xcr0: 0x6},
		{name: "OSXSAVE missing", leaf1ECX: 1 << 28, xcr0: 0x6},
		{name: "XMM state missing", leaf1ECX: 1<<27 | 1<<28, xcr0: 0x4},
		{name: "YMM state missing", leaf1ECX: 1<<27 | 1<<28, xcr0: 0x2},
		{name: "XMM and YMM state", leaf1ECX: 1<<27 | 1<<28, xcr0: 0x6, want: true},
		{name: "XMM and YMM state with superset", leaf1ECX: 1<<27 | 1<<28, xcr0: 0xe7, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			xgetbv := func() (uint32, uint32) {
				if test.leaf1ECX&(1<<27|1<<28) != 1<<27|1<<28 {
					panic("unexpected XGETBV")
				}
				return test.xcr0, 0
			}
			got := detectAMD64FeaturesWith(fakeCPUID(t, completeCPUID(test.leaf1ECX, 1<<5, 0)), xgetbv)
			if has := got&cpuAVX2 != 0; has != test.want {
				t.Fatalf("features = %#x, AVX2 = %v, want %v", got, has, test.want)
			}
		})
	}
}

func TestDetectAMD64FeaturesIndependentBits(t *testing.T) {
	for _, test := range []struct {
		name        string
		leaf1ECX    uint32
		leaf7EBX    uint32
		extendedECX uint32
		want        cpuFeatures
	}{
		{name: "SSSE3", leaf1ECX: 1 << 9, want: cpuSSSE3},
		{name: "SSE4.2", leaf1ECX: 1 << 20, want: cpuSSE42},
		{name: "POPCNT", leaf1ECX: 1 << 23, want: cpuPOPCNT},
		{name: "BMI1", leaf7EBX: 1 << 3, want: cpuBMI1},
		{name: "BMI2", leaf7EBX: 1 << 8, want: cpuBMI2},
		{name: "LZCNT", extendedECX: 1 << 5, want: cpuLZCNT},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := detectAMD64FeaturesWith(fakeCPUID(t, completeCPUID(test.leaf1ECX, test.leaf7EBX, test.extendedECX)), panicXGETBV)
			if got != test.want {
				t.Fatalf("features = %#x, want %#x", got, test.want)
			}
		})
	}
}

func TestDetectAMD64FeaturesFullySupported(t *testing.T) {
	const leaf1 = 1<<9 | 1<<20 | 1<<23 | 1<<27 | 1<<28
	const leaf7 = 1<<3 | 1<<5 | 1<<8
	want := cpuSSSE3 | cpuSSE42 | cpuPOPCNT | cpuAVX2 | cpuBMI1 | cpuBMI2 | cpuLZCNT
	got := detectAMD64FeaturesWith(fakeCPUID(t, completeCPUID(leaf1, leaf7, 1<<5)), func() (uint32, uint32) {
		return 0x6, 0
	})
	if got != want {
		t.Fatalf("features = %#x, want %#x", got, want)
	}
}

func TestDetectAMD64FeaturesLiveHost(t *testing.T) {
	known := cpuSSSE3 | cpuSSE42 | cpuPOPCNT | cpuAVX2 | cpuBMI1 | cpuBMI2 | cpuLZCNT
	if got := detectAMD64Features(); got & ^known != 0 {
		t.Fatalf("features = %#x, contains unknown amd64 feature bits %#x", got, got & ^known)
	}
}

func completeCPUID(leaf1ECX, leaf7EBX, extendedECX uint32) map[cpuidInput]cpuidOutput {
	return map[cpuidInput]cpuidOutput{
		{eax: 0}:          {eax: 7},
		{eax: 1}:          {ecx: leaf1ECX},
		{eax: 7}:          {ebx: leaf7EBX},
		{eax: 0x80000000}: {eax: 0x80000001},
		{eax: 0x80000001}: {ecx: extendedECX},
	}
}
