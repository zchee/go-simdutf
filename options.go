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

// Ported from simdutf commit c7bef0ff14a13fd6ea52e3347da2c659383392de,
// include/simdutf/implementation.h:187-188,4094-4138,4194-4228.

const DefaultLineLength int = 76

const Base64ReversePadding uint64 = 2

type Base64Options uint64

const (
	Base64Default                   Base64Options = 0
	Base64URL                       Base64Options = 1
	Base64DefaultNoPadding          Base64Options = 2
	Base64URLWithPadding            Base64Options = 3
	Base64DefaultAcceptGarbage      Base64Options = 4
	Base64URLAcceptGarbage          Base64Options = 5
	Base64DefaultOrURL              Base64Options = 8
	Base64DefaultOrURLAcceptGarbage Base64Options = 12
)

func Base64OptionsString(options Base64Options) string {
	switch options {
	case Base64Default:
		return "base64_default"
	case Base64URL:
		return "base64_url"
	case Base64DefaultNoPadding:
		return "base64_reverse_padding"
	case Base64URLWithPadding:
		return "base64_url_with_padding"
	case Base64DefaultAcceptGarbage:
		return "base64_default_accept_garbage"
	case Base64URLAcceptGarbage:
		return "base64_url_accept_garbage"
	case Base64DefaultOrURL:
		return "base64_default_or_url"
	case Base64DefaultOrURLAcceptGarbage:
		return "base64_default_or_url_accept_garbage"
	default:
		return "<unknown>"
	}
}

type LastChunkHandlingOptions uint64

const (
	Loose LastChunkHandlingOptions = iota
	Strict
	StopBeforePartial
	OnlyFullChunks
)

func IsPartial(options LastChunkHandlingOptions) bool {
	return options == StopBeforePartial || options == OnlyFullChunks
}

func LastChunkHandlingOptionsString(options LastChunkHandlingOptions) string {
	switch options {
	case Loose:
		return "loose"
	case Strict:
		return "strict"
	case StopBeforePartial:
		return "stop_before_partial"
	case OnlyFullChunks:
		return "only_full_chunks"
	default:
		return "<unknown>"
	}
}
