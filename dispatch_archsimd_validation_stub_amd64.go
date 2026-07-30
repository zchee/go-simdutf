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

func archsimdValidateUTF16LE() func([]uint16) bool             { return nil }
func archsimdValidateUTF16BE() func([]uint16) bool             { return nil }
func archsimdValidateUTF16LEWithErrors() func([]uint16) Result { return nil }
func archsimdValidateUTF16BEWithErrors() func([]uint16) Result { return nil }
func archsimdToWellFormedUTF16LE() func([]uint16, []uint16)    { return nil }
func archsimdToWellFormedUTF16BE() func([]uint16, []uint16)    { return nil }
func archsimdValidateUTF32() func([]uint32) bool               { return nil }
func archsimdValidateUTF32WithErrors() func([]uint32) Result   { return nil }
