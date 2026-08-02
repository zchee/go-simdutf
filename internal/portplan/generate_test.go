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

func TestRenderOperationsV1(t *testing.T) {
	got := string(RenderOperationsV1(nil))
	want := "mapping_version\tisa_ordinal\tisa_semantic_operation_exact\tsemantic_operation_id\tinitial_row_count\tinitial_cell_count\texisting_row_count\treconciliation\tevidence_anchor\n"
	if got != want {
		t.Fatalf("rendered header = %q", got)
	}
}

func TestOwnerLessV1UsesWaveBeforeFamily(t *testing.T) {
	utf16 := FinalCellV1{FamilyContractDisplayID: "FC-v1-utf16-source", SemanticOperationID: "a", ManifestOrdinal: 1, RowKeyV1: "a"}
	detection := FinalCellV1{FamilyContractDisplayID: "FC-v1-detection", SemanticOperationID: "a", ManifestOrdinal: 1, RowKeyV1: "a"}
	find := FinalCellV1{FamilyContractDisplayID: "FC-v1-find", SemanticOperationID: "a", ManifestOrdinal: 1, RowKeyV1: "a"}
	if !ownerLess(utf16, detection) {
		t.Fatal("W04 must sort before W06")
	}
	if !ownerLess(detection, find) {
		t.Fatal("W06 family IDs must break wave ties")
	}
	if _, ok := familyWave("FC-v1-unknown"); ok {
		t.Fatal("unknown family accepted")
	}
}

func TestValidateMembershipEmissionV1RejectsUnpinnedZeroInitial(t *testing.T) {
	ledger := []ISARowV1{{LedgerOrdinal: 1, Cells: [17]string{16: "unfrozen"}}}
	membership := MembershipV1{Operations: []FinalOperationV1{{
		ISAOrdinal:     1,
		Reconciliation: "zero_initial_explained",
		EvidenceAnchor: "unfrozen",
	}}}
	if err := validateMembershipEmission(membership, ledger, nil); err == nil {
		t.Fatal("unfrozen zero-initial operation accepted")
	}
	ledger[0].Cells[16] = "Pin: ledger provenance"
	membership.Operations[0].EvidenceAnchor = ledger[0].Cells[16]
	if err := validateMembershipEmission(membership, ledger, nil); err != nil {
		t.Fatalf("pinned zero-initial operation rejected: %v", err)
	}
}
