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

import "golang.org/x/sys/cpu"

// The feature contract follows the target declarations at
// simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b:
// src/simdutf/westmere.h and src/simdutf/haswell.h. golang.org/x/sys/cpu
// supplies the CPUID/XGETBV probe for every required feature except LZCNT,
// which x/sys/cpu does not expose and instead comes from an
// independent implementation of the architectural CPUID extended-leaf
// query below; it is not translated or structurally copied from
// include/simdutf/internal/isadetection.h, which has separate provenance.

type cpuidFunc func(eax, ecx uint32) (a, b, c, d uint32)

// cpuSSSE3 is amd64-local because no other architecture selector consumes it.
// It represents CPUID leaf 1 ECX bit 9 exactly; the UTF-8 Westmere kernel uses
// PALIGNR and PSHUFB but no SSE4, POPCNT, or later integer feature.
const cpuSSSE3 cpuFeatures = 1 << 7

//go:noescape
func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)

func detectAMD64Features() cpuFeatures {
	var features cpuFeatures
	if cpu.X86.HasSSSE3 {
		features |= cpuSSSE3
	}
	if cpu.X86.HasSSE42 {
		features |= cpuSSE42
	}
	if cpu.X86.HasPOPCNT {
		features |= cpuPOPCNT
	}
	if cpu.X86.HasBMI1 {
		features |= cpuBMI1
	}
	if cpu.X86.HasBMI2 {
		features |= cpuBMI2
	}
	if cpu.X86.HasAVX2 {
		features |= cpuAVX2
	}
	if hasLZCNT(cpuid) {
		features |= cpuLZCNT
	}
	return features
}

// hasLZCNT reports CPUID extended leaf 0x80000001 ECX bit 5 (LZCNT/ABM).
func hasLZCNT(cpuid cpuidFunc) bool {
	maxExtended, _, _, _ := cpuid(0x80000000, 0)
	if maxExtended < 0x80000001 {
		return false
	}
	_, _, ecx, _ := cpuid(0x80000001, 0)
	return ecx&(1<<5) != 0
}
