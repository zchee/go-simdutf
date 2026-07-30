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

//go:build amd64 && !goexperiment.simd

package simdutf

func archsimdChangeEndiannessUTF16() func([]uint16, []uint16) { return nil }
func archsimdCountUTF16LE() func([]uint16) int                { return nil }
func archsimdCountUTF16BE() func([]uint16) int                { return nil }
func archsimdUTF32LengthFromUTF16LE() func([]uint16) int      { return nil }
func archsimdUTF32LengthFromUTF16BE() func([]uint16) int      { return nil }

func archsimdConvertUTF16LEToLatin1() func([]uint16, []byte) int              { return nil }
func archsimdConvertUTF16BEToLatin1() func([]uint16, []byte) int              { return nil }
func archsimdConvertUTF16LEToLatin1WithErrors() func([]uint16, []byte) Result { return nil }
func archsimdConvertUTF16BEToLatin1WithErrors() func([]uint16, []byte) Result { return nil }
func archsimdConvertValidUTF16LEToLatin1() func([]uint16, []byte) int         { return nil }
func archsimdConvertValidUTF16BEToLatin1() func([]uint16, []byte) int         { return nil }

func archsimdConvertUTF16LEToUTF32() func([]uint16, []uint32) int              { return nil }
func archsimdConvertUTF16BEToUTF32() func([]uint16, []uint32) int              { return nil }
func archsimdConvertUTF16LEToUTF32WithErrors() func([]uint16, []uint32) Result { return nil }
func archsimdConvertUTF16BEToUTF32WithErrors() func([]uint16, []uint32) Result { return nil }
func archsimdConvertValidUTF16LEToUTF32() func([]uint16, []uint32) int         { return nil }
func archsimdConvertValidUTF16BEToUTF32() func([]uint16, []uint32) int         { return nil }
