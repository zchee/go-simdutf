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

// The feature contract follows the target declarations at
// simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f:
// src/simdutf/westmere.h and src/simdutf/haswell.h. This probe is an
// independent implementation of the architectural CPUID/XGETBV contract; it
// is not translated or structurally copied from
// include/simdutf/internal/isadetection.h, which has separate provenance.

type (
	cpuidFunc  func(eax, ecx uint32) (a, b, c, d uint32)
	xgetbvFunc func() (eax, edx uint32)
)

//go:noescape
func cpuid(eaxArg, ecxArg uint32) (eax, ebx, ecx, edx uint32)

//go:noescape
func xgetbv0() (eax, edx uint32)

func detectAMD64Features() cpuFeatures {
	return detectAMD64FeaturesWith(cpuid, xgetbv0)
}

func detectAMD64FeaturesWith(cpuid cpuidFunc, xgetbv xgetbvFunc) cpuFeatures {
	maxBasic, _, _, _ := cpuid(0, 0)

	var features cpuFeatures
	var avx, osxsave bool
	if maxBasic >= 1 {
		_, _, ecx, _ := cpuid(1, 0)
		if ecx&(1<<20) != 0 {
			features |= cpuSSE42
		}
		if ecx&(1<<23) != 0 {
			features |= cpuPOPCNT
		}
		osxsave = ecx&(1<<27) != 0
		avx = ecx&(1<<28) != 0
	}

	var rawAVX2 bool
	if maxBasic >= 7 {
		_, ebx, _, _ := cpuid(7, 0)
		if ebx&(1<<3) != 0 {
			features |= cpuBMI1
		}
		rawAVX2 = ebx&(1<<5) != 0
		if ebx&(1<<8) != 0 {
			features |= cpuBMI2
		}
	}

	if rawAVX2 && avx && osxsave {
		eax, _ := xgetbv()
		if eax&0x6 == 0x6 {
			features |= cpuAVX2
		}
	}

	maxExtended, _, _, _ := cpuid(0x80000000, 0)
	if maxExtended >= 0x80000001 {
		_, _, ecx, _ := cpuid(0x80000001, 0)
		if ecx&(1<<5) != 0 {
			features |= cpuLZCNT
		}
	}

	return features
}
