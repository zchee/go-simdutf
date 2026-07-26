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
	"reflect"
	"testing"
)

func TestBase64Options(t *testing.T) {
	if kind := reflect.TypeOf(Base64Default).Kind(); kind != reflect.Uint64 {
		t.Fatalf("Base64Options underlying kind = %v, want uint64", kind)
	}

	tests := []struct {
		name    string
		options Base64Options
		value   uint64
		text    string
	}{
		{"default", Base64Default, 0, "base64_default"},
		{"URL", Base64URL, 1, "base64_url"},
		{"default no padding", Base64DefaultNoPadding, 2, "base64_reverse_padding"},
		{"URL with padding", Base64URLWithPadding, 3, "base64_url_with_padding"},
		{"default accept garbage", Base64DefaultAcceptGarbage, 4, "base64_default_accept_garbage"},
		{"URL accept garbage", Base64URLAcceptGarbage, 5, "base64_url_accept_garbage"},
		{"default or URL", Base64DefaultOrURL, 8, "base64_default_or_url"},
		{"default or URL accept garbage", Base64DefaultOrURLAcceptGarbage, 12, "base64_default_or_url_accept_garbage"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint64(test.options); got != test.value {
				t.Errorf("value = %d, want %d", got, test.value)
			}
			if got := Base64OptionsString(test.options); got != test.text {
				t.Errorf("Base64OptionsString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestBase64OptionConstants(t *testing.T) {
	if kind := reflect.TypeOf(Base64ReversePadding).Kind(); kind != reflect.Uint64 {
		t.Fatalf("Base64ReversePadding kind = %v, want uint64", kind)
	}
	if Base64ReversePadding != 2 {
		t.Errorf("Base64ReversePadding = %d, want 2", Base64ReversePadding)
	}
	if got := Base64Default | Base64Options(Base64ReversePadding); got != Base64DefaultNoPadding {
		t.Errorf("Base64Default | Base64ReversePadding = %d, want %d", got, Base64DefaultNoPadding)
	}
	if got := Base64URL | Base64Options(Base64ReversePadding); got != Base64URLWithPadding {
		t.Errorf("Base64URL | Base64ReversePadding = %d, want %d", got, Base64URLWithPadding)
	}
	if kind := reflect.TypeOf(DefaultLineLength).Kind(); kind != reflect.Int {
		t.Fatalf("DefaultLineLength kind = %v, want int", kind)
	}
	if DefaultLineLength != 76 {
		t.Errorf("DefaultLineLength = %d, want 76", DefaultLineLength)
	}
}

func TestBase64OptionsStringUnknown(t *testing.T) {
	for _, options := range []Base64Options{6, 7, 255} {
		if got := Base64OptionsString(options); got != "<unknown>" {
			t.Errorf("Base64OptionsString(%d) = %q, want <unknown>", options, got)
		}
	}
}

func TestLastChunkHandlingOptions(t *testing.T) {
	if kind := reflect.TypeOf(Loose).Kind(); kind != reflect.Uint64 {
		t.Fatalf("LastChunkHandlingOptions underlying kind = %v, want uint64", kind)
	}

	tests := []struct {
		name    string
		options LastChunkHandlingOptions
		value   uint64
		text    string
	}{
		{"loose", Loose, 0, "loose"},
		{"strict", Strict, 1, "strict"},
		{"stop before partial", StopBeforePartial, 2, "stop_before_partial"},
		{"only full chunks", OnlyFullChunks, 3, "only_full_chunks"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint64(test.options); got != test.value {
				t.Errorf("value = %d, want %d", got, test.value)
			}
			if got := LastChunkHandlingOptionsString(test.options); got != test.text {
				t.Errorf("LastChunkHandlingOptionsString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestLastChunkHandlingOptionsStringUnknown(t *testing.T) {
	for _, options := range []LastChunkHandlingOptions{4, 255} {
		if got := LastChunkHandlingOptionsString(options); got != "<unknown>" {
			t.Errorf("LastChunkHandlingOptionsString(%d) = %q, want <unknown>", options, got)
		}
	}
}
