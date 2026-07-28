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

import "testing"

func TestInputsV1Headers(t *testing.T) {
	wantDependency := []string{"mapping_version", "consumer_row_key_v1", "consumer_manifest_ordinal", "consumer_go_symbol", "dependency_kind", "owner_kind", "owner_logical_id", "owner_go_symbol_or_empty", "evidence_anchor"}
	gotDependency := DependencyHeaderV1()
	if len(gotDependency) != len(wantDependency) {
		t.Fatalf("dependency header width = %d, want %d", len(gotDependency), len(wantDependency))
	}
	for i := range wantDependency {
		if gotDependency[i] != wantDependency[i] {
			t.Fatalf("dependency header[%d] = %q, want %q", i, gotDependency[i], wantDependency[i])
		}
	}
	gotLocked := LockedSetHeaderV1()
	if len(gotLocked) != 5 || gotLocked[0] != "schema_version" || gotLocked[4] != "evidence_anchor" {
		t.Fatalf("locked header = %q", gotLocked)
	}
}

func TestCanonicalPositiveV1(t *testing.T) {
	for _, s := range []string{"", "0", "01", "-1", "x"} {
		if _, err := canonicalPositive(s); err == nil {
			t.Fatalf("canonicalPositive(%q) accepted invalid decimal", s)
		}
	}
	if got, err := canonicalPositive("125"); err != nil || got != 125 {
		t.Fatalf("canonicalPositive valid = %d, %v", got, err)
	}
}

func TestDependencyKindsUseReviewedEnumsV1(t *testing.T) {
	for _, test := range []struct {
		kind, owner string
	}{
		{"none", "none"},
		{"wrapper_delegate", "row"},
		{"shared_kernel_delegate", "cell"},
	} {
		if test.kind == "shared_kernel_delegate" && test.owner != "cell" {
			t.Fatal("shared kernel owner kind drifted")
		}
	}
	if validDependencyEvidence("not-pinned") {
		t.Fatal("unpinned dependency evidence accepted")
	}
}

func TestExpectedWrapperEdgesIncludeEndianCounterpartsV1(t *testing.T) {
	planned := []ManifestRowV1{
		{Cells: [23]string{6: "ValidateUTF16"}, PlannedOrdinal: 1, RowKeyV1: RowKeyV1([6]string{"a", "b", "c", "d", "e", "f"})},
		{Cells: [23]string{6: "ValidateUTF16LE"}, PlannedOrdinal: 2, RowKeyV1: RowKeyV1([6]string{"a", "b", "c", "d", "e", "g"})},
		{Cells: [23]string{6: "ValidateUTF16BE"}, PlannedOrdinal: 3, RowKeyV1: RowKeyV1([6]string{"a", "b", "c", "d", "e", "h"})},
	}
	reviewed := []ReviewedMappingV1{{GoSymbol: "ValidateUTF16", Backends: [4]BackendMappingV1{{NAReason: "native_wrapper_delegates_explicit_endian"}}}}
	edges := expectedWrapperEdges(planned, reviewed)
	if len(edges) != 2 {
		t.Fatalf("endian wrapper edges = %d, want 2", len(edges))
	}
}

func TestExpectedSharedKernelEdgesUseCellOwnersV1(t *testing.T) {
	planned := []ManifestRowV1{
		{Cells: [23]string{6: "First"}, PlannedOrdinal: 1, RowKeyV1: RowKeyV1([6]string{"a", "b", "c", "d", "e", "f"})},
		{Cells: [23]string{6: "Second"}, PlannedOrdinal: 2, RowKeyV1: RowKeyV1([6]string{"a", "b", "c", "d", "e", "g"})},
	}
	reviewed := []ReviewedMappingV1{
		{FamilyContractDisplayID: "FC-v1-helper-validation", CanonicalKernelName: "kernel", Backends: [4]BackendMappingV1{{Outcome: "eligible"}}},
		{FamilyContractDisplayID: "FC-v1-helper-validation", CanonicalKernelName: "kernel", Backends: [4]BackendMappingV1{{Outcome: "eligible"}}},
	}
	edges, err := expectedSharedKernelEdges(planned, reviewed)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("shared-kernel edges = %d, want 1", len(edges))
	}
}

func TestLockedSetsRejectMalformedInputV1(t *testing.T) {
	for _, data := range [][]byte{
		[]byte("schema_version\tset_name\tordinal\tvalue_hex\tevidence_anchor\r\n"),
		[]byte("wrong\tset_name\tordinal\tvalue_hex\tevidence_anchor\n"),
		[]byte("schema_version\tset_name\tordinal\tvalue_hex\tevidence_anchor\nlocked-sets-v1\tx\t01\t00\te\n"),
		[]byte("schema_version\tset_name\tordinal\tvalue_hex\tevidence_anchor\nlocked-sets-v1\tbackend\t1\t\te\n"),
	} {
		if _, err := ParseLockedSetsV1(data, []byte("x\n"), make([]ManifestRowV1, 125)); err == nil {
			t.Fatal("ParseLockedSetsV1 accepted malformed input")
		}
	}
}
