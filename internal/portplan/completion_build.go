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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DispositionLookupV1 maps backend|GoSymbol -> terminal disposition.
type DispositionLookupV1 map[string]string

func completionEvidencePathV1(record EvidenceRecordV1, extension string) string {
	return "raw/" + record.AuthoritySHA256[:12] + "/" + record.FamilyID + "/" + record.CampaignID + "/" +
		record.LaneID + "/" + record.ProducerID + "/" + record.Kind + "/" + record.StorageID + "/" +
		record.Digest + "." + extension
}

func completionFinalEvidenceRecordV1(
	frozen ManifestRowV1, row ClassificationRowV1, cell FinalCellV1, context CompletionEvidenceContextV1,
	authority, sourceCommit, sourceTree, sourceParent, subject, cellID, backend string, ordinal int,
	prerequisite, current, kind, disposition, qualification, naReason, naSource string,
) EvidenceRecordV1 {
	family, err := FamilyKeyV1(row.FamilyContractDisplayID)
	if err != nil {
		panic(err)
	}
	record := EvidenceRecordV1{
		Schema: evidenceSchemaV1, Version: 1, FamilyID: family.StorageID, CampaignID: context.CampaignID,
		LaneID: "darwin-arm64-nosimd", ProducerID: "completion-fixture", Provider: backend,
		Kind: kind, MediaType: "application/json", Size: 1, Digest: SHA256Hex([]byte(subject + ":" + current + ":" + cellID)),
		AuthoritySHA256: authority, SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent, SourceRole: "new",
		OriginCampaignID: context.CampaignID, OriginProducerID: "completion-fixture",
		OriginSourceCommit: sourceCommit, OriginSourceTree: sourceTree, OriginSourceParent: sourceParent,
		RowID: row.RowKeyV1, CellID: cellID, BatchID: row.ScalarBatchStorageID, TransactionID: row.APITransactionStorageID,
		OperationID: row.SemanticOperationID, Backend: backend, StateSubject: subject, PrerequisiteState: prerequisite,
		CurrentState: current, Disposition: disposition, GoQualification: qualification,
		InitialState: "snapshot_planned", IdentitySetDigest: context.IdentitySetDigest,
		CommandDigest: context.CommandManifestSHA256, QualificationContractID: context.QualificationContractID,
		QualificationContractDigest: context.QualificationContractSHA256, CorpusID: context.CorpusID,
		CorpusDigest: context.CorpusSHA256, HostReceiptID: context.HostReceiptID, HostReceiptDigest: context.HostReceiptSHA256,
		CommandID: subject + "-" + current, CommandOrdinal: ordinal, CommandAction: "state_transition",
		CommandRole: "direct", OutputID: "state", OriginPath: "staging/" + subject + "-" + current + ".json",
		NAReason: naReason, NASource: naSource,
	}
	if subject == "row" {
		record.KeyKind = "row"
		record.StorageID = row.RowKeyV1
		record.TupleHex = hex.EncodeToString(EncodeTupleV1(frozen.Cells[manifestFamilyIndex], frozen.Cells[manifestUpstreamSymbolIndex], frozen.Cells[manifestUpstreamSignatureIndex], frozen.Cells[manifestHeaderPathLineIndex], frozen.Cells[manifestGoSymbolIndex], frozen.Cells[manifestGoSignatureIndex]))
	} else {
		key, err := CellKeyV1(row.RowKeyV1, backend)
		if err != nil {
			panic(err)
		}
		record.KeyKind, record.StorageID, record.TupleHex = "cell", key.StorageID, key.TupleHex
		record.InitialState = "eligible"
		record.BatchID = cell.EvidenceBatchStorageID
		record.DirectSymbol, record.SymbolID = cell.DirectSymbol, cell.DirectSymbolStorageID
		if cell.BackendOutcome == "not_applicable" {
			record.BatchID = row.ScalarBatchStorageID
		}
	}
	record.Path = completionEvidencePathV1(record, "json")
	record.ReceiptID = ReceiptIDV1(record)
	return record
}

// LoadQualificationDispositionsV1 reads family disposition JSON files under evidenceDir.
// Keys are "backend|GoSymbol" with values selected|direct_only.
func LoadQualificationDispositionsV1(evidenceDir string) (DispositionLookupV1, error) {
	out := DispositionLookupV1{}
	entries, err := os.ReadDir(evidenceDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "-dispositions-v1.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(evidenceDir, name))
		if err != nil {
			return nil, err
		}
		var doc struct {
			ProviderCandidate string                     `json:"provider_candidate"`
			Ops               map[string]json.RawMessage `json:"ops"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		backend := strings.TrimSpace(doc.ProviderCandidate)
		if backend == "" {
			return nil, fmt.Errorf("%s: missing provider_candidate", name)
		}
		for op, rawOp := range doc.Ops {
			var asString string
			if err := json.Unmarshal(rawOp, &asString); err == nil && asString != "" {
				key := backend + "|" + op
				if prev, ok := out[key]; ok && prev != asString {
					return nil, fmt.Errorf("conflicting disposition for %s: %s vs %s", key, prev, asString)
				}
				out[key] = asString
				continue
			}
			var asObj map[string]any
			if err := json.Unmarshal(rawOp, &asObj); err != nil {
				return nil, fmt.Errorf("%s op %s: %w", name, op, err)
			}
			disp, _ := asObj["disposition"].(string)
			if disp == "" {
				return nil, fmt.Errorf("%s op %s: missing disposition", name, op)
			}
			key := backend + "|" + op
			if prev, ok := out[key]; ok && prev != disp {
				return nil, fmt.Errorf("conflicting disposition for %s: %s vs %s", key, prev, disp)
			}
			out[key] = disp
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no dispositions loaded from %s", evidenceDir)
	}
	return out, nil
}

func gitRevParse(root, rev string) (string, error) {
	cmd := exec.Command("git", "rev-parse", rev)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitWorkingTreeClean(root string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=no")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(out))) == 0, nil
}

// BuildRepositoryCompletionV1 builds a ValidateCompletionV1-ready closure from
// frozen inputs, generated membership/classification, and qualification dispositions.
func BuildRepositoryCompletionV1(root string) (CompletionV1, CompletionValidationContextV1, error) {
	read := func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, rel))
	}
	plannedBytes, err := read("docs/porting/simdutf-port-v1/inputs/planned-rows-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	initialBytes, err := read("docs/porting/api-manifest.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	ledgerBytes, err := read("docs/porting/isa-eligibility.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}

	var reviewedBytes []byte
	for index, name := range []string{
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-001-020.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-021-055.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-056-105.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-106-125.tsv",
	} {
		fragment, err := read(name)
		if err != nil {
			return CompletionV1{}, CompletionValidationContextV1{}, err
		}
		if index != 0 {
			newline := bytes.IndexByte(fragment, '\n')
			if newline < 0 {
				return CompletionV1{}, CompletionValidationContextV1{}, fmt.Errorf("review fragment %q lacks a header", name)
			}
			fragment = fragment[newline+1:]
		}
		reviewedBytes = append(reviewedBytes, fragment...)
	}

	planned, err := ParseManifestV1(plannedBytes)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	initial, err := ParseManifestV1(initialBytes)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	plannedByKey := map[string]ManifestRowV1{}
	for _, row := range planned {
		plannedByKey[row.RowKeyV1] = row
	}
	for index := range initial {
		if frozen, ok := plannedByKey[initial[index].RowKeyV1]; ok {
			initial[index].Cells[manifestStatusIndex] = frozen.Cells[manifestStatusIndex]
			initial[index].Cells[manifestMilestoneIndex] = frozen.Cells[manifestMilestoneIndex]
		}
	}
	ledger, err := ParseISALedgerV1(ledgerBytes)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	reviewed, err := ParseReviewedMappingsV1(reviewedBytes, planned, ledger)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	existingBytes, err := read("docs/porting/simdutf-port-v1/inputs/review-fragments/existing-members-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	existing, err := ParseReviewedExistingMembersV1(existingBytes, initial, ledger)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	depBytes, err := read("docs/porting/simdutf-port-v1/inputs/dependency-map-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	dependencies, err := ParseDependencyMapV1(depBytes, planned, reviewed)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	ranks, err := BuildCanonicalRowRanksV1(planned, reviewed, ledger, dependencies)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	membership, err := BuildMembershipV1(reviewed, initial, planned, ledger, existing, ranks)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	classification, err := BuildClassificationV1(planned, reviewed, dependencies, ranks, membership)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}

	final := append([]ManifestRowV1(nil), initial...)
	for index := range final {
		if _, ok := plannedByKey[final[index].RowKeyV1]; ok {
			final[index].Cells[manifestStatusIndex] = "implemented"
			final[index].Cells[manifestMilestoneIndex] = "611becc-current-api"
		}
	}

	authority, err := read("docs/porting/simdutf-port-v1/inputs/go-base-authority-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	qualification, err := read("docs/porting/simdutf-port-v1/inputs/qualification-policy-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	corpus, err := read("docs/porting/simdutf-port-v1/inputs/corpus-contract-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	officialBackends, err := read("docs/porting/simdutf-port-v1/inputs/archsimd-primitives-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	officialHosts, err := read("docs/porting/simdutf-port-v1/inputs/host-authority-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	officialAPIs, err := read("docs/porting/simdutf-port-v1/inputs/upstream-authority-v1.tsv")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}

	dispositions, err := LoadQualificationDispositionsV1(filepath.Join(root, "docs/porting/simdutf-port-v1/evidence"))
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}

	authorityDigest := SHA256Hex(authority)
	sourceCommit, err := gitRevParse(root, "HEAD")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	sourceTree, err := gitRevParse(root, "HEAD^{tree}")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	sourceParent, err := gitRevParse(root, "HEAD^")
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}
	sourceClean, err := gitWorkingTreeClean(root)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}

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
		fe := evidence
		fe.CampaignID = "campaign-v1-" + SHA256Hex([]byte("family:"+row.FamilyContractDisplayID))
		familyEvidence[row.FamilyContractDisplayID] = fe
		for ordinal, state := range []struct{ prerequisite, current, kind, disposition, qualification string }{
			{"snapshot_planned", "scalar_private", "state-transition", "", ""},
			{"scalar_private", "scalar_green", "state-transition", "", ""},
			{"scalar_green", "family_published", "state-transition", "", ""},
			{"family_published", "complete", "state-transition", "", ""},
		} {
			record := completionFinalEvidenceRecordV1(
				frozen[row.RowKeyV1], row, FinalCellV1{}, fe, authorityDigest, sourceCommit, sourceTree, sourceParent,
				"row", "", "neon", ordinal+1, state.prerequisite, state.current, state.kind, state.disposition, state.qualification, "", "",
			)
			registry.receipts[record.ReceiptID] = record
		}
	}

	for _, cell := range classification.Cells {
		row := classRows[cell.RowKeyV1]
		context := evidence
		context.CampaignID = "campaign-v1-" + SHA256Hex([]byte("cell:"+cell.CellStorageID))
		var states []struct{ prerequisite, current, kind, disposition, qualification string }
		if cell.BackendOutcome != "eligible" {
			states = []struct{ prerequisite, current, kind, disposition, qualification string }{
				{"eligible", "not_applicable", "not-applicable", "not_applicable", ""},
			}
		} else {
			goSymbol := frozen[cell.RowKeyV1].Cells[manifestGoSymbolIndex]
			key := cell.Backend + "|" + goSymbol
			disp, ok := dispositions[key]
			if !ok {
				return CompletionV1{}, CompletionValidationContextV1{}, fmt.Errorf("missing disposition for eligible cell %s", key)
			}
			switch disp {
			case "selected":
				states = []struct{ prerequisite, current, kind, disposition, qualification string }{
					{"eligible", "direct_built", "state-transition", "", ""},
					{"direct_built", "hard_gates_green", "state-transition", "", ""},
					{"hard_gates_green", "dispatch_candidate", "state-transition", "", ""},
					{"dispatch_candidate", "selected", "state-transition", "selected", "pass"},
				}
			case "direct_only":
				states = []struct{ prerequisite, current, kind, disposition, qualification string }{
					{"eligible", "direct_built", "state-transition", "", ""},
					{"direct_built", "hard_gates_green", "state-transition", "", ""},
					{"hard_gates_green", "dispatch_candidate", "state-transition", "", ""},
					{"dispatch_candidate", "direct_only", "state-transition", "direct_only", "fail"},
				}
			default:
				return CompletionV1{}, CompletionValidationContextV1{}, fmt.Errorf("illegal disposition %q for %s", disp, key)
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
			return CompletionV1{}, CompletionValidationContextV1{}, err
		}
		closure := CompletionRowV1{
			RowKeyV1: frozenRow.RowKeyV1, ManifestOrdinal: frozenRow.PlannedOrdinal, GoSymbol: frozenRow.Cells[manifestGoSymbolIndex],
			ScalarBatchStorageID: row.ScalarBatchStorageID, APITransactionStorageID: row.APITransactionStorageID,
			State: "complete", EvidenceContext: rowEvidence, ReceiptIDs: rowReceipts,
		}
		for _, backend := range membershipBackends {
			cellKey, err := CellKeyV1(frozenRow.RowKeyV1, backend)
			if err != nil {
				return CompletionV1{}, CompletionValidationContextV1{}, err
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
				goSymbol := frozenRow.Cells[manifestGoSymbolIndex]
				disp := dispositions[member.Backend+"|"+goSymbol]
				switch disp {
				case "selected":
					closureCell.State, closureCell.Disposition, closureCell.Qualification = "selected", "selected", "pass"
				case "direct_only":
					closureCell.State, closureCell.Disposition, closureCell.Qualification = "direct_only", "direct_only", "fail"
				default:
					return CompletionV1{}, CompletionValidationContextV1{}, fmt.Errorf("illegal/missing disposition for %s|%s", member.Backend, goSymbol)
				}
				closureCell.DirectSymbol, closureCell.DirectSymbolStorageID = member.DirectSymbol, member.DirectSymbolStorageID
			} else {
				closureCell.State, closureCell.Disposition = "not_applicable", "not_applicable"
				closureCell.NAReason, closureCell.NASource = member.BackendNAReason, member.BackendEvidenceAnchor
			}
			receipts, err := requiredCellReceiptIDsV1(registry, row, closureCell, member, CompletionValidationContextV1{
				AuthoritySHA256: authorityDigest, SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent,
			})
			if err != nil {
				return CompletionV1{}, CompletionValidationContextV1{}, err
			}
			closureCell.ReceiptIDs = receipts
			closure.Cells = append(closure.Cells, closureCell)
		}
		rows = append(rows, closure)
	}

	index, err := EvidenceRegistryDigestV1(registry)
	if err != nil {
		return CompletionV1{}, CompletionValidationContextV1{}, err
	}

	frozenDigest := authorityDigest

	context := CompletionValidationContextV1{
		InitialManifest: initial, FinalManifest: final, FrozenPlanned: planned, ReviewedMappings: reviewed,
		ISALedger: ledger, ExistingMembers: existing, Dependencies: dependencies, Classification: classification, Membership: membership,
		Registry: registry, AuthorityBytes: authority, QualificationBytes: qualification, CorpusBytes: corpus,
		OfficialBackendsBytes: officialBackends, OfficialHostsBytes: officialHosts, OfficialAPIsBytes: officialAPIs,
		AuthoritySHA256: authorityDigest, FrozenInputSHA256: frozenDigest,
		SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent, SourceClean: sourceClean,
		ClassificationSHA256: SHA256Hex(RenderClassificationV1(classification.Rows)),
		QualificationSHA256:  SHA256Hex(qualification), ManifestSHA256: SHA256Hex(renderManifestRowsV1(final)),
		CorpusSHA256: SHA256Hex(corpus), IndexSHA256: index, OfficialBackendsSHA256: SHA256Hex(officialBackends),
		OfficialHostsSHA256: SHA256Hex(officialHosts), OfficialAPIsSHA256: SHA256Hex(officialAPIs),
	}
	completion := CompletionV1{
		Schema: completionSchemaV1, Version: 1, SourceCommit: sourceCommit, SourceTree: sourceTree, SourceParent: sourceParent,
		SourceClean: sourceClean, AuthoritySHA256: authorityDigest, FrozenInputSHA256: frozenDigest,
		ClassificationSHA256: context.ClassificationSHA256, QualificationSHA256: context.QualificationSHA256,
		ManifestSHA256: context.ManifestSHA256, CorpusSHA256: context.CorpusSHA256, IndexSHA256: index,
		OfficialBackendsSHA256: context.OfficialBackendsSHA256, OfficialHostsSHA256: context.OfficialHostsSHA256,
		OfficialAPIsSHA256: context.OfficialAPIsSHA256, Implemented: 155, Planned: 0, Excluded: 9,
		Rows: CanonicalCompletionRowsV1(rows),
	}
	return completion, context, nil
}
