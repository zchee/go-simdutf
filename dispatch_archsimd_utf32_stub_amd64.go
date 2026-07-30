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

func archsimdConvertUTF32ToLatin1() func([]uint32, []byte) int              { return nil }
func archsimdConvertUTF32ToLatin1WithErrors() func([]uint32, []byte) Result { return nil }
func archsimdConvertValidUTF32ToLatin1() func([]uint32, []byte) int         { return nil }

func archsimdConvertUTF32ToUTF8() func([]uint32, []byte) int              { return nil }
func archsimdConvertUTF32ToUTF8WithErrors() func([]uint32, []byte) Result { return nil }
func archsimdConvertValidUTF32ToUTF8() func([]uint32, []byte) int         { return nil }

func archsimdConvertUTF32ToUTF16LE() func([]uint32, []uint16) int              { return nil }
func archsimdConvertUTF32ToUTF16BE() func([]uint32, []uint16) int              { return nil }
func archsimdConvertUTF32ToUTF16LEWithErrors() func([]uint32, []uint16) Result { return nil }
func archsimdConvertUTF32ToUTF16BEWithErrors() func([]uint32, []uint16) Result { return nil }
func archsimdConvertValidUTF32ToUTF16LE() func([]uint32, []uint16) int         { return nil }
func archsimdConvertValidUTF32ToUTF16BE() func([]uint32, []uint16) int         { return nil }

func archsimdUTF8LengthFromUTF32() func([]uint32) int  { return nil }
func archsimdUTF16LengthFromUTF32() func([]uint32) int { return nil }
