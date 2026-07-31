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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Phase 0 intentionally validates the closed schema rather than inventing a
// completion artifact or evidence registry before the port is complete.
func TestCompletionSchemaV1(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "docs", "porting", "simdutf-port-v1", "inputs", "completion-schema-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	if schema["additionalProperties"] != false || schema["$id"] != "completion-schema-v1.json" {
		t.Fatal("completion schema is not closed or versioned")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || properties["schema"].(map[string]any)["const"] != completionSchemaV1 {
		t.Fatal("completion schema runtime identity is not frozen")
	}
	properties, ok = schema["properties"].(map[string]any)
	if !ok || properties["implemented"].(map[string]any)["const"].(float64) != 155 || properties["planned"].(map[string]any)["const"].(float64) != 0 || properties["excluded"].(map[string]any)["const"].(float64) != 9 {
		t.Fatal("completion totals are not frozen")
	}
	for _, forbidden := range []string{"timestamp", "time", "path", "workspace"} {
		if _, found := properties[forbidden]; found {
			t.Fatalf("mutable field %q", forbidden)
		}
	}
}

func TestCompletionClassificationV1RejectsSemanticMutations(t *testing.T) {
	classification, membership, frozen := completionClassificationFixtureV1(t)
	if err := validateCompletionClassificationV1(classification, membership, frozen); err != nil {
		t.Fatalf("valid classification rejected: %v", err)
	}

	mutatedClassification := classification
	mutatedClassification.Cells = append([]FinalCellV1(nil), classification.Cells...)
	mutatedMembership := membership
	mutatedMembership.Cells = append([]FinalCellV1(nil), membership.Cells...)
	for index := range mutatedClassification.Cells {
		if mutatedClassification.Cells[index].BackendOutcome != "eligible" {
			continue
		}
		mutatedClassification.Cells[index].DirectSymbol += "Mutated"
		symbol, err := SymbolKeyV1(mutatedClassification.Cells[index].Backend, mutatedClassification.Cells[index].DirectSymbol)
		if err != nil {
			t.Fatal(err)
		}
		mutatedClassification.Cells[index].DirectSymbolStorageID = symbol.StorageID
		mutatedMembership.Cells[index].DirectSymbol = mutatedClassification.Cells[index].DirectSymbol
		mutatedMembership.Cells[index].DirectSymbolStorageID = symbol.StorageID
		break
	}
	if err := validateCompletionClassificationV1(mutatedClassification, mutatedMembership, frozen); err == nil {
		t.Fatal("classification accepted a lockstep direct-symbol substitution")
	}

	mutatedMembership = membership
	mutatedMembership.Operations = append([]FinalOperationV1(nil), membership.Operations...)
	mutatedMembership.Operations[0].InitialCellCount++
	if err := validateCompletionClassificationV1(classification, mutatedMembership, frozen); err == nil {
		t.Fatal("classification accepted a false operation count")
	}

	mutatedClassification = classification
	mutatedClassification.Batches = append([]BatchRecordV1(nil), classification.Batches...)
	mutatedClassification.Batches = append(mutatedClassification.Batches, classification.Batches[0])
	if err := validateCompletionClassificationV1(mutatedClassification, membership, frozen); err == nil {
		t.Fatal("classification accepted a duplicate batch")
	}
}

func TestCompletionEvidenceRecordV1BindsProvenanceAndCampaign(t *testing.T) {
	record, rowKey := completionEvidenceRecordFixtureV1(t)
	if !completionEvidenceRecordValidV1(record, rowKey, "", record.AuthoritySHA256) {
		t.Fatal("valid completion evidence record rejected")
	}

	mutated := record
	mutated.OriginSourceCommit = strings.Repeat("9", 40)
	mutated.ReceiptID = ReceiptIDV1(mutated)
	if completionEvidenceRecordValidV1(mutated, rowKey, "", record.AuthoritySHA256) {
		t.Fatal("origin/source substitution accepted")
	}

	context := CompletionValidationContextV1{
		FrozenPlanned: []ManifestRowV1{{RowKeyV1: rowKey}},
		Registry: &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{
			record.ReceiptID: record,
		}},
		AuthoritySHA256: record.AuthoritySHA256,
	}
	if err := validateCompletionRegistryV1(context); err != nil {
		t.Fatalf("valid registry rejected: %v", err)
	}

	raw := record
	raw.Kind = "stdout"
	raw.CommandID = "command-other"
	raw.CommandAction = "go_test_full"
	raw.OutputID = "stdout"
	raw.StateSubject = "none"
	raw.PrerequisiteState = ""
	raw.CurrentState = ""
	raw.CommandDigest = strings.Repeat("8", 64)
	raw.OriginPath = "staging/stdout.txt"
	raw.Path = completionEvidencePathV1(raw, "txt")
	raw.ReceiptID = ReceiptIDV1(raw)
	if !completionEvidenceRecordValidV1(raw, rowKey, "", raw.AuthoritySHA256) {
		t.Fatal("valid raw completion evidence rejected")
	}
	context.Registry.receipts[raw.ReceiptID] = raw
	if err := validateCompletionRegistryV1(context); err == nil {
		t.Fatal("one campaign ID accepted unequal command manifests")
	}
}

func TestRequiredCellReceiptIDsV1RejectsCrossCampaignChain(t *testing.T) {
	base, rowKey := completionEvidenceRecordFixtureV1(t)
	cellKey, err := CellKeyV1(rowKey, "neon")
	if err != nil {
		t.Fatal(err)
	}
	base.KeyKind = "cell"
	base.StorageID = cellKey.StorageID
	base.TupleHex = cellKey.TupleHex
	base.CellID = cellKey.StorageID
	base.Backend = "neon"
	base.DirectSymbol = "validateNEON"
	symbol, err := SymbolKeyV1("neon", base.DirectSymbol)
	if err != nil {
		t.Fatal(err)
	}
	base.SymbolID = symbol.StorageID
	base.StateSubject = "backend_cell"
	base.InitialState = "eligible"

	states := []struct {
		prerequisite, current, disposition, qualification string
	}{
		{"eligible", "direct_built", "", ""},
		{"direct_built", "hard_gates_green", "", ""},
		{"hard_gates_green", "dispatch_candidate", "", ""},
		{"dispatch_candidate", "selected", "selected", "pass"},
	}
	registry := &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{}}
	for index, state := range states {
		record := base
		record.CommandOrdinal = index + 1
		record.CommandID = "state-" + strings.ReplaceAll(state.current, "_", "-")
		record.PrerequisiteState = state.prerequisite
		record.CurrentState = state.current
		record.Disposition = state.disposition
		record.GoQualification = state.qualification
		record.Path = completionEvidencePathV1(record, "json")
		record.ReceiptID = ReceiptIDV1(record)
		registry.receipts[record.ReceiptID] = record
	}
	cell := CompletionCellV1{
		Backend: "neon", CellStorageID: cellKey.StorageID, Outcome: "eligible", State: "selected",
		DirectSymbol: base.DirectSymbol, DirectSymbolStorageID: base.SymbolID,
		EvidenceBatchStorageID: base.BatchID, Disposition: "selected", Qualification: "pass",
		EvidenceContext: completionEvidenceContextFromRecordV1(base),
	}
	classification := ClassificationRowV1{
		RowKeyV1: rowKey, FamilyContractDisplayID: "FC-v1-helper-validation",
		SemanticOperationID: base.OperationID, APITransactionStorageID: base.TransactionID,
	}
	member := FinalCellV1{
		RowKeyV1: rowKey, CellStorageID: cellKey.StorageID,
		FamilyContractDisplayID: "FC-v1-helper-validation", Backend: "neon",
		SemanticOperationID: base.OperationID, DirectSymbol: base.DirectSymbol,
		DirectSymbolStorageID: base.SymbolID, EvidenceBatchStorageID: base.BatchID,
	}
	context := CompletionValidationContextV1{
		Registry: registry, AuthoritySHA256: base.AuthoritySHA256,
		SourceCommit: base.SourceCommit, SourceTree: base.SourceTree, SourceParent: base.SourceParent,
	}
	receipts, err := requiredCellReceiptIDsV1(registry, classification, cell, member, context)
	if err != nil {
		t.Fatalf("valid cell chain rejected: %v", err)
	}
	if len(receipts) != len(states) {
		t.Fatalf("receipt count = %d, want %d", len(receipts), len(states))
	}
	historical := make([]EvidenceRecordV1, 0, len(states))
	for _, record := range registry.receipts {
		record.CampaignID = "campaign-v1-" + strings.Repeat("8", 64)
		record.OriginCampaignID = record.CampaignID
		record.IdentitySetDigest = strings.Repeat("8", 64)
		record.CommandDigest = strings.Repeat("8", 64)
		record.Path = completionEvidencePathV1(record, "json")
		record.ReceiptID = ReceiptIDV1(record)
		historical = append(historical, record)
	}
	for _, record := range historical {
		registry.receipts[record.ReceiptID] = record
	}
	if receipts, err := requiredCellReceiptIDsV1(registry, classification, cell, member, context); err != nil || len(receipts) != len(states) {
		t.Fatalf("selected chain did not tolerate retained history: receipts=%d err=%v", len(receipts), err)
	}

	for id, record := range registry.receipts {
		if record.CurrentState != "hard_gates_green" || record.CampaignID != base.CampaignID {
			continue
		}
		delete(registry.receipts, id)
		record.CampaignID = "campaign-v1-" + strings.Repeat("9", 64)
		record.OriginCampaignID = record.CampaignID
		record.Path = completionEvidencePathV1(record, "json")
		record.ReceiptID = ReceiptIDV1(record)
		registry.receipts[record.ReceiptID] = record
		break
	}
	if _, err := requiredCellReceiptIDsV1(registry, classification, cell, member, context); err == nil {
		t.Fatal("cell chain accepted a transition from another campaign")
	}
}

func completionClassificationFixtureV1(t *testing.T) (ClassificationV1, MembershipV1, map[string]ManifestRowV1) {
	t.Helper()
	root := completionRepositoryRootV1(t)
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	manifestData := read("docs/porting/api-manifest.tsv")
	manifest, err := ParseManifestV1(manifestData)
	if err != nil {
		t.Fatal(err)
	}
	planned, err := ParseManifestV1(read("docs/porting/simdutf-port-v1/inputs/planned-rows-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(planned) != 125 {
		t.Fatalf("planned snapshot: got %d rows, want 125", len(planned))
	}
	ledger, err := ParseISALedgerV1(read("docs/porting/isa-eligibility.tsv"))
	if err != nil {
		t.Fatal(err)
	}

	var reviewedData []byte
	for index, name := range []string{
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-001-020.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-021-055.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-056-105.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-106-125.tsv",
	} {
		fragment := read(name)
		if index != 0 {
			newline := bytes.IndexByte(fragment, '\n')
			if newline < 0 {
				t.Fatalf("review fragment %q lacks a header", name)
			}
			fragment = fragment[newline+1:]
		}
		reviewedData = append(reviewedData, fragment...)
	}
	reviewed, err := ParseReviewedMappingsV1(reviewedData, planned, ledger)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := ParseReviewedExistingMembersV1(
		read("docs/porting/simdutf-port-v1/inputs/review-fragments/existing-members-v1.tsv"),
		manifest,
		ledger,
	)
	if err != nil {
		t.Fatal(err)
	}
	dependencies, err := ParseDependencyMapV1(
		read("docs/porting/simdutf-port-v1/inputs/dependency-map-v1.tsv"),
		planned,
		reviewed,
	)
	if err != nil {
		t.Fatal(err)
	}
	ranks, err := BuildCanonicalRowRanksV1(planned, reviewed, ledger, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	membership, err := BuildMembershipV1(reviewed, manifest, planned, ledger, existing, ranks)
	if err != nil {
		t.Fatal(err)
	}
	classification, err := BuildClassificationV1(planned, reviewed, dependencies, ranks, membership)
	if err != nil {
		t.Fatal(err)
	}
	frozen := make(map[string]ManifestRowV1, len(planned))
	for _, row := range planned {
		frozen[row.RowKeyV1] = row
	}
	return classification, membership, frozen
}

func completionEvidenceRecordFixtureV1(t *testing.T) (EvidenceRecordV1, string) {
	t.Helper()
	fields := [6]string{"ValidateFixture", "func([]byte) bool", "planned", "future-upstream-api", "fixture", "fixture"}
	rowKey := RowKeyV1(fields)
	family, err := FamilyKeyV1("FC-v1-helper-validation")
	if err != nil {
		t.Fatal(err)
	}
	hostDigest := strings.Repeat("7", 64)
	record := EvidenceRecordV1{
		Schema: evidenceSchemaV1, Version: 1,
		KeyKind: "row", StorageID: rowKey, TupleHex: hex.EncodeToString(EncodeTupleV1(fields[:]...)),
		FamilyID:   family.StorageID,
		CampaignID: "campaign-v1-" + strings.Repeat("2", 64),
		LaneID:     "darwin-arm64-nosimd", ProducerID: "producer", Provider: "neon",
		Kind: "state-transition", MediaType: "application/json", Size: 1, Digest: strings.Repeat("3", 64),
		SourceCommit: strings.Repeat("4", 40), SourceTree: strings.Repeat("5", 40),
		SourceParent: strings.Repeat("6", 40), SourceRole: "new",
		OriginCampaignID: "campaign-v1-" + strings.Repeat("2", 64), OriginProducerID: "producer",
		OriginSourceCommit: strings.Repeat("4", 40), OriginSourceTree: strings.Repeat("5", 40),
		OriginSourceParent: strings.Repeat("6", 40),
		RowID:              rowKey, OperationID: "op-v1-" + strings.Repeat("8", 64),
		BatchID: "batch-v1-" + strings.Repeat("9", 64), TransactionID: "transaction-v1-" + strings.Repeat("0", 64),
		StateSubject: "row", PrerequisiteState: "snapshot_planned", CurrentState: "scalar_private",
		InitialState: "snapshot_planned", IdentitySetDigest: strings.Repeat("a", 64),
		CommandDigest: strings.Repeat("b", 64), QualificationContractID: "qualification-v1",
		QualificationContractDigest: strings.Repeat("c", 64), CorpusID: "corpus-v1",
		CorpusDigest: strings.Repeat("d", 64), HostReceiptID: hostReceiptIDV1(hostDigest),
		AuthoritySHA256: strings.Repeat("e", 64), HostReceiptDigest: hostDigest,
		CommandID: "state-scalar-private", CommandOrdinal: 1, CommandAction: "state_transition",
		CommandRole: "direct", OutputID: "state", OriginPath: "staging/state.json",
	}
	record.Path = completionEvidencePathV1(record, "json")
	record.ReceiptID = ReceiptIDV1(record)
	return record, rowKey
}

func completionEvidenceContextFromRecordV1(record EvidenceRecordV1) CompletionEvidenceContextV1 {
	return CompletionEvidenceContextV1{
		CampaignID: record.CampaignID, IdentitySetDigest: record.IdentitySetDigest,
		CommandManifestSHA256:       record.CommandDigest,
		QualificationContractID:     record.QualificationContractID,
		QualificationContractSHA256: record.QualificationContractDigest,
		CorpusID:                    record.CorpusID, CorpusSHA256: record.CorpusDigest,
		HostReceiptID: record.HostReceiptID, HostReceiptSHA256: record.HostReceiptDigest,
	}
}
func TestCompletionV1EndToEndFixture(t *testing.T) {
	completion, context := completionEndToEndFixtureV1(t)
	if err := ValidateCompletionV1(completion, context); err != nil {
		t.Fatalf("valid completion fixture rejected: %v", err)
	}
}

func TestCompletionV1EndToEndFixtureRetainsHistoricalEvidence(t *testing.T) {
	completion, context := completionEndToEndFixtureV1(t)
	var selected EvidenceRecordV1
	for _, record := range context.Registry.receipts {
		selected = record
		break
	}
	historical := selected
	historical.CampaignID = "campaign-v1-" + SHA256Hex([]byte("retained-history"))
	historical.OriginCampaignID = historical.CampaignID
	historical.Kind = "stdout"
	historical.StateSubject = "none"
	historical.PrerequisiteState = ""
	historical.CurrentState = ""
	historical.Disposition = ""
	historical.GoQualification = ""
	historical.ProofReceiptIDs = nil
	historical.NAReason = ""
	historical.NASource = ""
	historical.CommandID = "retained-history"
	historical.CommandAction = "go_test_full"
	historical.CommandRole = "direct"
	historical.OriginPath = "staging/retained-history.txt"
	historical.Digest = SHA256Hex([]byte("retained-history-output"))
	historical.Path = completionEvidencePathV1(historical, "txt")
	historical.ReceiptID = ReceiptIDV1(historical)
	context.Registry.receipts[historical.ReceiptID] = historical
	index, err := EvidenceRegistryDigestV1(context.Registry)
	if err != nil {
		t.Fatal(err)
	}
	context.IndexSHA256 = index
	completion.IndexSHA256 = index
	if err := ValidateCompletionV1(completion, context); err != nil {
		t.Fatalf("completion rejected retained historical evidence: %v", err)
	}
}

func TestCompletionV1EndToEndFixtureRejectsLockstepAndSelectedEvidenceMutations(t *testing.T) {
	completion, context := completionEndToEndFixtureV1(t)
	mutated := context
	mutated.Membership.Cells = append([]FinalCellV1(nil), context.Membership.Cells...)
	mutated.Classification.Cells = append([]FinalCellV1(nil), context.Classification.Cells...)
	for index := range mutated.Membership.Cells {
		if mutated.Membership.Cells[index].BackendOutcome != "eligible" {
			continue
		}
		mutated.Membership.Cells[index].DirectSymbol += "Mutated"
		symbol, err := SymbolKeyV1(mutated.Membership.Cells[index].Backend, mutated.Membership.Cells[index].DirectSymbol)
		if err != nil {
			t.Fatal(err)
		}
		mutated.Membership.Cells[index].DirectSymbolStorageID = symbol.StorageID
		for cellIndex := range mutated.Classification.Cells {
			if mutated.Classification.Cells[cellIndex].CellStorageID == mutated.Membership.Cells[index].CellStorageID {
				mutated.Classification.Cells[cellIndex].DirectSymbol = mutated.Membership.Cells[index].DirectSymbol
				mutated.Classification.Cells[cellIndex].DirectSymbolStorageID = symbol.StorageID
				break
			}
		}
		break
	}
	if err := ValidateCompletionV1(completion, mutated); err == nil {
		t.Fatal("completion accepted lockstep-mutated classification authority")
	}

	mutated = context
	mutated.Registry = &EvidenceRegistryV1{receipts: make(map[string]EvidenceRecordV1, len(context.Registry.receipts))}
	maps.Copy(mutated.Registry.receipts, context.Registry.receipts)
	for id, record := range mutated.Registry.receipts {
		if record.StateSubject != "backend_cell" {
			continue
		}
		delete(mutated.Registry.receipts, id)
		record.CampaignID = "campaign-v1-" + SHA256Hex([]byte("mutated-selected-campaign"))
		record.OriginCampaignID = record.CampaignID
		record.Path = completionEvidencePathV1(record, "json")
		record.ReceiptID = ReceiptIDV1(record)
		mutated.Registry.receipts[record.ReceiptID] = record
		break
	}
	index, err := EvidenceRegistryDigestV1(mutated.Registry)
	if err != nil {
		t.Fatal(err)
	}
	mutated.IndexSHA256 = index
	completion.IndexSHA256 = index
	if err := ValidateCompletionV1(completion, mutated); err == nil {
		t.Fatal("completion accepted a selected evidence campaign mutation")
	}
	completion, context = completionEndToEndFixtureV1(t)
	mutated = context
	mutated.Registry = &EvidenceRegistryV1{receipts: make(map[string]EvidenceRecordV1, len(context.Registry.receipts))}
	maps.Copy(mutated.Registry.receipts, context.Registry.receipts)
	for id, record := range mutated.Registry.receipts {
		if record.StateSubject != "backend_cell" {
			continue
		}
		delete(mutated.Registry.receipts, id)
		record.SourceCommit = strings.Repeat("4", 40)
		record.OriginSourceCommit = record.SourceCommit
		record.Path = completionEvidencePathV1(record, "json")
		record.ReceiptID = ReceiptIDV1(record)
		mutated.Registry.receipts[record.ReceiptID] = record
		break
	}
	index, err = EvidenceRegistryDigestV1(mutated.Registry)
	if err != nil {
		t.Fatal(err)
	}
	mutated.IndexSHA256 = index
	completion.IndexSHA256 = index
	if err := ValidateCompletionV1(completion, mutated); err == nil {
		t.Fatal("completion accepted a selected evidence source mutation")
	}
}

func completionEndToEndFixtureV1(t *testing.T) (CompletionV1, CompletionValidationContextV1) {
	t.Helper()
	root := completionRepositoryRootV1(t)
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	initial, err := ParseManifestV1(read("docs/porting/api-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	planned, err := ParseManifestV1(read("docs/porting/simdutf-port-v1/inputs/planned-rows-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	plannedByKey := make(map[string]ManifestRowV1, len(planned))
	for _, row := range planned {
		plannedByKey[row.RowKeyV1] = row
	}
	for index := range initial {
		if frozen, ok := plannedByKey[initial[index].RowKeyV1]; ok {
			initial[index].Cells[manifestStatusIndex] = frozen.Cells[manifestStatusIndex]
			initial[index].Cells[manifestMilestoneIndex] = frozen.Cells[manifestMilestoneIndex]
		}
	}
	ledger, err := ParseISALedgerV1(read("docs/porting/isa-eligibility.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	var reviewedBytes []byte
	for index, name := range []string{
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-001-020.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-021-055.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-056-105.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-106-125.tsv",
	} {
		fragment := read(name)
		if index != 0 {
			newline := bytes.IndexByte(fragment, '\n')
			if newline < 0 {
				t.Fatalf("review fragment %q lacks a header", name)
			}
			fragment = fragment[newline+1:]
		}
		reviewedBytes = append(reviewedBytes, fragment...)
	}
	reviewed, err := ParseReviewedMappingsV1(reviewedBytes, planned, ledger)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := ParseReviewedExistingMembersV1(read("docs/porting/simdutf-port-v1/inputs/review-fragments/existing-members-v1.tsv"), initial, ledger)
	if err != nil {
		t.Fatal(err)
	}
	dependencies, err := ParseDependencyMapV1(read("docs/porting/simdutf-port-v1/inputs/dependency-map-v1.tsv"), planned, reviewed)
	if err != nil {
		t.Fatal(err)
	}
	ranks, err := BuildCanonicalRowRanksV1(planned, reviewed, ledger, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	membership, err := BuildMembershipV1(reviewed, initial, planned, ledger, existing, ranks)
	if err != nil {
		t.Fatal(err)
	}
	classification, err := BuildClassificationV1(planned, reviewed, dependencies, ranks, membership)
	if err != nil {
		t.Fatal(err)
	}

	final := append([]ManifestRowV1(nil), initial...)
	for index := range final {
		if _, ok := plannedByKey[final[index].RowKeyV1]; ok {
			final[index].Cells[manifestStatusIndex] = "implemented"
			final[index].Cells[manifestMilestoneIndex] = "611becc-current-api"
		}
	}

	authority := read("docs/porting/simdutf-port-v1/inputs/go-base-authority-v1.tsv")
	qualification := read("docs/porting/simdutf-port-v1/inputs/qualification-policy-v1.tsv")
	corpus := read("docs/porting/simdutf-port-v1/inputs/corpus-contract-v1.tsv")
	officialBackends := read("docs/porting/simdutf-port-v1/inputs/archsimd-primitives-v1.tsv")
	officialHosts := read("docs/porting/simdutf-port-v1/inputs/host-authority-v1.tsv")
	officialAPIs := read("docs/porting/simdutf-port-v1/inputs/upstream-authority-v1.tsv")
	authorityDigest := SHA256Hex(authority)
	sourceCommit := strings.Repeat("1", 40)
	sourceTree := strings.Repeat("2", 40)
	sourceParent := strings.Repeat("3", 40)
	evidence := CompletionEvidenceContextV1{
		IdentitySetDigest:           SHA256Hex([]byte("phase-0-final-identity-set")),
		CommandManifestSHA256:       SHA256Hex([]byte("phase-0-final-command-manifest")),
		QualificationContractID:     "qualification-v1-final",
		QualificationContractSHA256: SHA256Hex(qualification),
		CorpusID:                    "corpus-v1-final",
		CorpusSHA256:                SHA256Hex(corpus),
		HostReceiptSHA256:           SHA256Hex(officialHosts),
	}
	evidence.HostReceiptID = hostReceiptIDV1(evidence.HostReceiptSHA256)
	registry := &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{}}
	classRows := make(map[string]ClassificationRowV1, len(classification.Rows))
	classCells := make(map[string]FinalCellV1, len(classification.Cells))
	frozen := make(map[string]ManifestRowV1, len(planned))
	for _, row := range classification.Rows {
		classRows[row.RowKeyV1] = row
	}
	for _, cell := range classification.Cells {
		classCells[cell.CellStorageID] = cell
	}
	for _, row := range planned {
		frozen[row.RowKeyV1] = row
	}
	familyEvidence := map[string]CompletionEvidenceContextV1{}
	for _, row := range classification.Rows {
		context, ok := familyEvidence[row.FamilyContractDisplayID]
		if !ok {
			context = evidence
			context.CampaignID = "campaign-v1-" + SHA256Hex([]byte("family:"+row.FamilyContractDisplayID))
			familyEvidence[row.FamilyContractDisplayID] = context
		}
		for ordinal, state := range []struct{ prerequisite, current string }{
			{"snapshot_planned", "scalar_private"},
			{"scalar_private", "scalar_green"},
			{"scalar_green", "family_published"},
			{"family_published", "complete"},
		} {
			record := completionFinalEvidenceRecordV1(
				frozen[row.RowKeyV1], row, FinalCellV1{}, context, authorityDigest, sourceCommit, sourceTree, sourceParent,
				"row", "", "neon", ordinal+1, state.prerequisite, state.current, "state-transition", "", "", "", "",
			)
			registry.receipts[record.ReceiptID] = record
		}
	}
	for _, cell := range classification.Cells {
		row := classRows[cell.RowKeyV1]
		context := evidence
		context.CampaignID = "campaign-v1-" + SHA256Hex([]byte("cell:"+cell.CellStorageID))
		states := []struct{ prerequisite, current, kind, disposition, qualification string }{
			{"eligible", "not_applicable", "not-applicable", "not_applicable", ""},
		}
		if cell.BackendOutcome == "eligible" {
			states = []struct{ prerequisite, current, kind, disposition, qualification string }{
				{"eligible", "direct_built", "state-transition", "", ""},
				{"direct_built", "hard_gates_green", "state-transition", "", ""},
				{"hard_gates_green", "dispatch_candidate", "state-transition", "", ""},
				{"dispatch_candidate", "selected", "state-transition", "selected", "pass"},
			}
		}
		for ordinal, state := range states {
			record := completionFinalEvidenceRecordV1(
				frozen[cell.RowKeyV1], row, cell, context, authorityDigest, sourceCommit, sourceTree, sourceParent,
				"backend_cell", cell.CellStorageID, cell.Backend, ordinal+1, state.prerequisite, state.current, state.kind,
				state.disposition, state.qualification, cell.BackendNAReason, cell.BackendEvidenceAnchor,
			)
			registry.receipts[record.ReceiptID] = record
		}
	}

	rows := make([]CompletionRowV1, 0, len(planned))
	for _, frozenRow := range planned {
		row := classRows[frozenRow.RowKeyV1]
		rowEvidence := familyEvidence[row.FamilyContractDisplayID]
		rowReceipts, err := requiredRowReceiptIDsV1(registry, row, rowEvidence, CompletionValidationContextV1{
			AuthoritySHA256: authorityDigest, SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent,
		})
		if err != nil {
			t.Fatal(err)
		}
		closure := CompletionRowV1{
			RowKeyV1: frozenRow.RowKeyV1, ManifestOrdinal: frozenRow.PlannedOrdinal, GoSymbol: frozenRow.Cells[manifestGoSymbolIndex],
			ScalarBatchStorageID: row.ScalarBatchStorageID, APITransactionStorageID: row.APITransactionStorageID,
			State: "complete", EvidenceContext: rowEvidence, ReceiptIDs: rowReceipts,
		}
		for _, backend := range membershipBackends {
			cellKey, err := CellKeyV1(frozenRow.RowKeyV1, backend)
			if err != nil {
				t.Fatal(err)
			}
			member := classCells[cellKey.StorageID]
			cellEvidence := evidence
			cellEvidence.CampaignID = "campaign-v1-" + SHA256Hex([]byte("cell:"+member.CellStorageID))
			closureCell := CompletionCellV1{
				Backend: member.Backend, CellStorageID: member.CellStorageID, Outcome: member.BackendOutcome,
				KernelBatchStorageID: member.KernelBatchStorageID, EvidenceBatchStorageID: member.EvidenceBatchStorageID,
				EvidenceContext: cellEvidence,
			}
			if member.BackendOutcome == "eligible" {
				closureCell.State, closureCell.Disposition, closureCell.Qualification = "selected", "selected", "pass"
				closureCell.DirectSymbol, closureCell.DirectSymbolStorageID = member.DirectSymbol, member.DirectSymbolStorageID
			} else {
				closureCell.State, closureCell.Disposition = "not_applicable", "not_applicable"
				closureCell.NAReason, closureCell.NASource = member.BackendNAReason, member.BackendEvidenceAnchor
			}
			receipts, err := requiredCellReceiptIDsV1(registry, row, closureCell, member, CompletionValidationContextV1{
				AuthoritySHA256: authorityDigest, SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent,
			})
			if err != nil {
				t.Fatal(err)
			}
			closureCell.ReceiptIDs = receipts
			closure.Cells = append(closure.Cells, closureCell)
		}
		rows = append(rows, closure)
	}
	index, err := EvidenceRegistryDigestV1(registry)
	if err != nil {
		t.Fatal(err)
	}
	context := CompletionValidationContextV1{
		InitialManifest: initial, FinalManifest: final, FrozenPlanned: planned, ReviewedMappings: reviewed,
		ISALedger: ledger, ExistingMembers: existing, Dependencies: dependencies, Classification: classification, Membership: membership,
		Registry: registry, AuthorityBytes: authority, QualificationBytes: qualification, CorpusBytes: corpus,
		OfficialBackendsBytes: officialBackends, OfficialHostsBytes: officialHosts, OfficialAPIsBytes: officialAPIs,
		AuthoritySHA256: authorityDigest, FrozenInputSHA256: authorityDigest,
		SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent, SourceClean: true,
		ClassificationSHA256: SHA256Hex(RenderClassificationV1(classification.Rows)),
		QualificationSHA256:  SHA256Hex(qualification), ManifestSHA256: SHA256Hex(renderManifestRowsV1(final)),
		CorpusSHA256: SHA256Hex(corpus), IndexSHA256: index, OfficialBackendsSHA256: SHA256Hex(officialBackends),
		OfficialHostsSHA256: SHA256Hex(officialHosts), OfficialAPIsSHA256: SHA256Hex(officialAPIs),
	}
	completion := CompletionV1{
		Schema: completionSchemaV1, Version: 1, SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent,
		SourceClean: true, AuthoritySHA256: authorityDigest, FrozenInputSHA256: authorityDigest,
		ClassificationSHA256: context.ClassificationSHA256, QualificationSHA256: context.QualificationSHA256,
		ManifestSHA256: context.ManifestSHA256, CorpusSHA256: context.CorpusSHA256, IndexSHA256: index,
		OfficialBackendsSHA256: context.OfficialBackendsSHA256, OfficialHostsSHA256: context.OfficialHostsSHA256,
		OfficialAPIsSHA256: context.OfficialAPIsSHA256, Implemented: 155, Planned: 0, Excluded: 9,
		Rows: CanonicalCompletionRowsV1(rows),
	}
	return completion, context
}

func completionRepositoryRootV1(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
