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

package portplan

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func TestAuthorityV1Receipts(t *testing.T) {
	upstream := mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/upstream-authority-v1.tsv")
	if _, err := ParseUpstreamAuthorityV1(upstream); err != nil {
		t.Fatal(err)
	}
	goBase := mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/go-base-authority-v1.tsv")
	if _, err := ParseGoBaseAuthorityV1(goBase); err != nil {
		t.Fatal(err)
	}
	locked := map[string][]LockedSetRecordV1{
		"authoritative_host":    {{ValueHex: hex.EncodeToString([]byte("darwin-arm64-apple-m3-max"))}, {ValueHex: hex.EncodeToString([]byte("linux-amd64-debian-13-xeon-platinum-8481c"))}},
		"official_architecture": {{ValueHex: hex.EncodeToString([]byte("amd64"))}, {ValueHex: hex.EncodeToString([]byte("arm64"))}},
	}
	if _, err := ParseHostAuthorityV1(mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/host-authority-v1.tsv"), locked); err != nil {
		t.Fatal(err)
	}
	sources, err := ParseArchsimdSourceFilesV1(mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/archsimd-source-files-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ValidateArchsimdPrimitivesV1(mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/archsimd-primitives-v1.tsv"), sources, nil); err != nil {
		t.Fatal(err)
	}
	if _, err = ParseCorpusContractV1(mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/corpus-contract-v1.tsv")); err != nil {
		t.Fatal(err)
	}
}

func TestFrozenInputsV1RoundTripAndMutations(t *testing.T) {
	files := map[string][]byte{"b.txt": []byte("b"), "a.txt": []byte("a\n")}
	receipt, digest, err := RenderFrozenInputsV1(files)
	if err != nil {
		t.Fatal(err)
	}
	again, againDigest, err := RenderFrozenInputsV1(files)
	if err != nil || string(receipt) != string(again) || digest != againDigest {
		t.Fatalf("render not deterministic: %v", err)
	}
	if _, got, err := ParseFrozenInputsV1(receipt, files); err != nil || got != digest {
		t.Fatalf("round trip: %q %v", got, err)
	}
	tuple, err := CanonicalFrozenAuthorityTupleV1([]FrozenInputV1{{Path: "a.txt", Size: 2, SHA256: SHA256Hex([]byte("a\n"))}, {Path: "b.txt", Size: 1, SHA256: SHA256Hex([]byte("b"))}})
	if err != nil || SHA256Hex(tuple) != digest {
		t.Fatalf("canonical authority tuple: %v", err)
	}
	if _, err := CanonicalFrozenAuthorityTupleV1([]FrozenInputV1{{Path: "b.txt", Size: 1, SHA256: SHA256Hex([]byte("b"))}, {Path: "a.txt", Size: 2, SHA256: SHA256Hex([]byte("a\n"))}}); err == nil {
		t.Fatal("accepted unsorted authority tuple")
	}
	for _, bad := range [][]byte{
		[]byte(strings.Replace(string(receipt), "schema_version", "version", 1)),
		[]byte(strings.Replace(string(receipt), "\tv1\t", "\tv1\t", 1) + "\r"),
		[]byte(strings.Replace(string(receipt), "a.txt", "../a.txt", 1)),
		[]byte(strings.Replace(string(receipt), "\t2\t", "\t3\t", 1)),
	} {
		if _, _, err := ParseFrozenInputsV1(bad, files); err == nil {
			t.Fatal("accepted mutated receipt")
		}
	}
	changed := map[string][]byte{"b.txt": []byte("b"), "a.txt": []byte("changed")}
	if _, _, err := ParseFrozenInputsV1(receipt, changed); err == nil {
		t.Fatal("accepted file byte drift")
	}
	if _, _, err := ParseFrozenInputsV1(receipt, map[string][]byte{"a.txt": []byte("a\n")}); err == nil {
		t.Fatal("accepted missing file")
	}
	if _, _, err := ParseFrozenInputsV1(receipt, map[string][]byte{"a.txt": []byte("a\n"), "b.txt": []byte("b"), "c.txt": []byte("c")}); err == nil {
		t.Fatal("accepted extra file")
	}
}

func TestAuthorityV1RejectsStructuralMutations(t *testing.T) {
	upstream := mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/upstream-authority-v1.tsv")
	for _, bad := range [][]byte{
		[]byte(strings.Replace(string(upstream), "schema_version", "version", 1)),
		[]byte(strings.Replace(string(upstream), "611becc", "711becc", 1)),
		append(append([]byte(nil), upstream[:len(upstream)-1]...), '\r', '\n'),
	} {
		if _, err := ParseUpstreamAuthorityV1(bad); err == nil {
			t.Fatal("accepted mutated upstream authority")
		}
	}
	goBase := mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/go-base-authority-v1.tsv")
	for _, bad := range [][]byte{
		[]byte(strings.Replace(string(goBase), "083f5bc", "183f5bc", 1)),
		[]byte(strings.Replace(string(goBase), "\ttrue\t", "\tfalse\t", 1)),
	} {
		if _, err := ParseGoBaseAuthorityV1(bad); err == nil {
			t.Fatal("accepted mutated Go base authority")
		}
	}
	corpus := mustAuthorityFile(t, "../../docs/porting/simdutf-port-v1/inputs/corpus-contract-v1.tsv")
	for _, bad := range [][]byte{
		[]byte(strings.Replace(string(corpus), "\nv1\t1\t", "\nv1\t2\t", 1)),
		[]byte(strings.Replace(string(corpus), "frozen_deterministic", "deferred_until_scalar_publication", 1)),
		[]byte(strings.Replace(string(corpus), "FC-v1-find", "FC-v1-unknown", 1)),
	} {
		if _, err := ParseCorpusContractV1(bad); err == nil {
			t.Fatal("accepted mutated corpus contract")
		}
	}
}

func mustAuthorityFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
