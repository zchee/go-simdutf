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
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

const completionSchemaV1 = "simdutf-port-completion-v1"

// CompletionValidationContextV1 is the complete immutable input boundary for a
// final completion audit. All digests are lowercase SHA-256 values unless the
// field is a Git identity, which is a lowercase 40-character SHA-1 value.
type CompletionValidationContextV1 struct {
	InitialManifest, FinalManifest, FrozenPlanned                                        []ManifestRowV1
	ReviewedMappings                                                                     []ReviewedMappingV1
	ISALedger                                                                            []ISARowV1
	ExistingMembers                                                                      []ExistingMemberV1
	Dependencies                                                                         []DependencyRecordV1
	Classification                                                                       ClassificationV1
	Membership                                                                           MembershipV1
	Registry                                                                             *EvidenceRegistryV1
	AuthorityBytes, QualificationBytes, CorpusBytes                                      []byte
	OfficialBackendsBytes, OfficialHostsBytes, OfficialAPIsBytes                         []byte
	AuthoritySHA256, FrozenInputSHA256                                                   string
	SourceCommit, SourceTree, SourceParent                                               string
	SourceClean                                                                          bool
	ClassificationSHA256, QualificationSHA256, ManifestSHA256, CorpusSHA256, IndexSHA256 string
	OfficialBackendsSHA256, OfficialHostsSHA256, OfficialAPIsSHA256                      string
}

// CompletionV1 is the canonical, evidence-bound final audit record. It has no
// timestamps or workspace paths so its bytes are portable and reproducible.
type CompletionV1 struct {
	Schema                 string            `json:"schema"`
	Version                int               `json:"version"`
	SourceCommit           string            `json:"source_commit"`
	SourceTree             string            `json:"source_tree"`
	SourceParent           string            `json:"source_parent"`
	SourceClean            bool              `json:"source_clean"`
	AuthoritySHA256        string            `json:"authority_sha256"`
	FrozenInputSHA256      string            `json:"frozen_input_sha256"`
	ClassificationSHA256   string            `json:"classification_sha256"`
	QualificationSHA256    string            `json:"qualification_sha256"`
	ManifestSHA256         string            `json:"manifest_sha256"`
	CorpusSHA256           string            `json:"corpus_sha256"`
	IndexSHA256            string            `json:"raw_return_index_sha256"`
	OfficialBackendsSHA256 string            `json:"official_backends_sha256"`
	OfficialHostsSHA256    string            `json:"official_hosts_sha256"`
	OfficialAPIsSHA256     string            `json:"official_apis_sha256"`
	Implemented            int               `json:"implemented"`
	Planned                int               `json:"planned"`
	Excluded               int               `json:"excluded"`
	Rows                   []CompletionRowV1 `json:"rows"`
}

// CompletionEvidenceContextV1 selects one exact final evidence chain while the
// global index retains every historical campaign.
type CompletionEvidenceContextV1 struct {
	CampaignID                  string `json:"campaign_id"`
	IdentitySetDigest           string `json:"identity_set_digest"`
	CommandManifestSHA256       string `json:"command_manifest_sha256"`
	QualificationContractID     string `json:"qualification_contract_id"`
	QualificationContractSHA256 string `json:"qualification_contract_sha256"`
	CorpusID                    string `json:"corpus_id"`
	CorpusSHA256                string `json:"corpus_sha256"`
	HostReceiptID               string `json:"host_receipt_id"`
	HostReceiptSHA256           string `json:"host_receipt_sha256"`
}

// CompletionRowV1 closes one original planned row.
type CompletionRowV1 struct {
	RowKeyV1                string                      `json:"row_key_v1"`
	ManifestOrdinal         int                         `json:"manifest_ordinal"`
	GoSymbol                string                      `json:"go_symbol"`
	ScalarBatchStorageID    string                      `json:"scalar_batch_storage_id"`
	APITransactionStorageID string                      `json:"api_transaction_storage_id"`
	State                   string                      `json:"state"`
	EvidenceContext         CompletionEvidenceContextV1 `json:"evidence_context"`
	ReceiptIDs              []string                    `json:"receipt_ids"`
	Cells                   []CompletionCellV1          `json:"cells"`
}

// CompletionCellV1 closes one exact original row/backend cell. State is the
// terminal evidence state: not_applicable, selected, or direct_only.
type CompletionCellV1 struct {
	Backend                string                      `json:"backend"`
	CellStorageID          string                      `json:"cell_storage_id"`
	Outcome                string                      `json:"outcome"`
	State                  string                      `json:"state"`
	DirectSymbol           string                      `json:"direct_symbol"`
	DirectSymbolStorageID  string                      `json:"direct_symbol_storage_id"`
	KernelBatchStorageID   string                      `json:"kernel_batch_storage_id"`
	EvidenceBatchStorageID string                      `json:"evidence_batch_storage_id"`
	Qualification          string                      `json:"qualification"`
	Disposition            string                      `json:"disposition"`
	NAReason               string                      `json:"na_reason"`
	NASource               string                      `json:"na_source"`
	EvidenceContext        CompletionEvidenceContextV1 `json:"evidence_context"`
	ReceiptIDs             []string                    `json:"receipt_ids"`
}

// ValidateCompletionJSONV1 rejects every completion document that is not both
// canonical JSON and an exact closure of the supplied frozen input context.
func ValidateCompletionJSONV1(input []byte, context CompletionValidationContextV1) (CompletionV1, error) {
	var completion CompletionV1
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&completion); err != nil {
		return completion, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return completion, errors.New("completion: trailing JSON")
	}
	canonical, err := RenderCompletionV1(completion)
	if err != nil || !bytes.Equal(input, canonical) {
		return completion, errors.New("completion: noncanonical JSON")
	}
	return completion, ValidateCompletionV1(completion, context)
}

// RenderCompletionV1 renders the stable JSON representation. Semantic input
// checks belong to ValidateCompletionV1 because rendering never fabricates an audit.
func RenderCompletionV1(completion CompletionV1) ([]byte, error) {
	if completion.Schema != completionSchemaV1 || completion.Version != 1 {
		return nil, errors.New("completion: unsupported schema")
	}
	data, err := json.Marshal(completion)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// ValidateCompletionV1 validates exact row, cell, state, and proof closure.
func ValidateCompletionV1(completion CompletionV1, context CompletionValidationContextV1) error {
	if err := validateCompletionContext(context); err != nil {
		return err
	}
	if completion.Schema != completionSchemaV1 || completion.Version != 1 || !completion.SourceClean ||
		completion.SourceCommit != context.SourceCommit || completion.SourceTree != context.SourceTree ||
		completion.SourceParent != context.SourceParent || completion.AuthoritySHA256 != context.AuthoritySHA256 ||
		completion.FrozenInputSHA256 != context.FrozenInputSHA256 ||
		completion.ClassificationSHA256 != context.ClassificationSHA256 ||
		completion.QualificationSHA256 != context.QualificationSHA256 ||
		completion.ManifestSHA256 != context.ManifestSHA256 || completion.CorpusSHA256 != context.CorpusSHA256 ||
		completion.IndexSHA256 != context.IndexSHA256 ||
		completion.OfficialBackendsSHA256 != context.OfficialBackendsSHA256 ||
		completion.OfficialHostsSHA256 != context.OfficialHostsSHA256 ||
		completion.OfficialAPIsSHA256 != context.OfficialAPIsSHA256 {
		return errors.New("completion: identity binding mismatch")
	}
	implemented, planned, excluded := completionStatusCounts(context.FinalManifest)
	if implemented != 155 || planned != 0 || excluded != 9 || completion.Implemented != implemented ||
		completion.Planned != planned || completion.Excluded != excluded {
		return errors.New("completion: status counts do not close")
	}

	initialByKey, err := manifestRowsByKeyV1(context.InitialManifest)
	if err != nil {
		return err
	}
	finalByKey, err := manifestRowsByKeyV1(context.FinalManifest)
	if err != nil {
		return err
	}
	frozenByKey := make(map[string]ManifestRowV1, len(context.FrozenPlanned))
	for _, row := range context.FrozenPlanned {
		if row.RowKeyV1 == "" || row.PlannedOrdinal < 1 || frozenByKey[row.RowKeyV1].RowKeyV1 != "" {
			return errors.New("completion: invalid frozen planned rows")
		}
		frozenByKey[row.RowKeyV1] = row
	}
	if err := validateFinalManifestTransitionV1(initialByKey, finalByKey, frozenByKey); err != nil {
		return err
	}
	if err := validateCompletionClassificationV1(context.Classification, context.Membership, frozenByKey); err != nil {
		return err
	}

	classRows := make(map[string]ClassificationRowV1, len(context.Classification.Rows))
	for _, row := range context.Classification.Rows {
		if classRows[row.RowKeyV1].RowKeyV1 != "" {
			return errors.New("completion: duplicate classification row")
		}
		classRows[row.RowKeyV1] = row
	}
	members := make(map[string]FinalCellV1, len(context.Classification.Cells))
	for _, cell := range context.Classification.Cells {
		if cell.CellStorageID == "" || members[cell.CellStorageID].CellStorageID != "" {
			return errors.New("completion: duplicate classification cell")
		}
		members[cell.CellStorageID] = cell
	}
	if len(completion.Rows) != len(frozenByKey) {
		return errors.New("completion: missing or extra row")
	}

	seenRows := map[string]bool{}
	seenCells := map[string]bool{}
	seenReceipts := map[string]string{}
	for index, row := range completion.Rows {
		if index > 0 && completionRowLess(row, completion.Rows[index-1]) {
			return errors.New("completion: rows are not canonical")
		}
		frozen, frozenOK := frozenByKey[row.RowKeyV1]
		final, finalOK := finalByKey[row.RowKeyV1]
		if !frozenOK || !finalOK || seenRows[row.RowKeyV1] || row.ManifestOrdinal != frozen.PlannedOrdinal ||
			row.GoSymbol != frozen.Cells[manifestGoSymbolIndex] || row.GoSymbol != final.Cells[manifestGoSymbolIndex] ||
			row.State != "complete" || !validCompletionEvidenceContextV1(row.EvidenceContext) {
			return errors.New("completion: invalid row")
		}
		classification, ok := classRows[row.RowKeyV1]
		if !ok || row.ScalarBatchStorageID != classification.ScalarBatchStorageID ||
			row.APITransactionStorageID != classification.APITransactionStorageID {
			return errors.New("completion: scalar or transaction join mismatch")
		}
		requiredRowReceipts, err := requiredRowReceiptIDsV1(context.Registry, classification, row.EvidenceContext, context)
		if err != nil {
			return err
		}
		if !sameStrings(row.ReceiptIDs, requiredRowReceipts) {
			return errors.New("completion: row receipt closure is not canonical or complete")
		}
		rowOwner := "row:" + row.RowKeyV1
		for _, id := range row.ReceiptIDs {
			if owner, reused := seenReceipts[id]; reused && owner != rowOwner {
				return fmt.Errorf("completion: proof %q is reused", id)
			}
			seenReceipts[id] = rowOwner
		}
		seenRows[row.RowKeyV1] = true
		if len(row.Cells) != len(membershipBackends) {
			return errors.New("completion: missing or extra cell")
		}
		for cellIndex, cell := range row.Cells {
			if cell.Backend != membershipBackends[cellIndex] {
				return errors.New("completion: cells are not canonical")
			}
			if err := validateCompletionCellV1(classification, cell, members, context, seenCells, seenReceipts); err != nil {
				return err
			}
		}
	}
	if len(seenRows) != 125 || len(seenCells) != 500 {
		return errors.New("completion: row or cell closure mismatch")
	}
	if err := validateFamilyPublicationEvidenceV1(context.Classification, completion.Rows, context.Registry, context); err != nil {
		return err
	}
	return nil
}

func validateCompletionContext(context CompletionValidationContextV1) error {
	if len(context.InitialManifest) != 164 || len(context.FinalManifest) != 164 || len(context.FrozenPlanned) != 125 ||
		len(context.ReviewedMappings) != 125 || len(context.ISALedger) != 23 ||
		len(context.ExistingMembers) != 30 || len(context.Dependencies) != 185 ||
		len(context.Classification.Rows) != 125 || len(context.Classification.Cells) != 500 ||
		len(context.Membership.Cells) != 500 || len(context.Membership.Operations) != 23 || context.Registry == nil ||
		len(context.AuthorityBytes) == 0 || len(context.QualificationBytes) == 0 || len(context.CorpusBytes) == 0 ||
		len(context.OfficialBackendsBytes) == 0 || len(context.OfficialHostsBytes) == 0 ||
		len(context.OfficialAPIsBytes) == 0 ||
		!context.SourceClean || !lowerHex(context.SourceCommit, 40) || !lowerHex(context.SourceTree, 40) ||
		!lowerHex(context.SourceParent, 40) {
		return errors.New("completion: incomplete final context")
	}
	for _, digest := range []string{
		context.AuthoritySHA256, context.FrozenInputSHA256, context.ClassificationSHA256,
		context.QualificationSHA256, context.ManifestSHA256, context.CorpusSHA256, context.IndexSHA256,
		context.OfficialBackendsSHA256, context.OfficialHostsSHA256, context.OfficialAPIsSHA256,
	} {
		if !lowerHex(digest, 64) {
			return errors.New("completion: invalid context digest")
		}
	}
	if authorityDigest := SHA256Hex(context.AuthorityBytes); context.AuthoritySHA256 != authorityDigest ||
		context.FrozenInputSHA256 != authorityDigest ||
		context.QualificationSHA256 != SHA256Hex(context.QualificationBytes) ||
		context.CorpusSHA256 != SHA256Hex(context.CorpusBytes) ||
		context.OfficialBackendsSHA256 != SHA256Hex(context.OfficialBackendsBytes) ||
		context.OfficialHostsSHA256 != SHA256Hex(context.OfficialHostsBytes) ||
		context.OfficialAPIsSHA256 != SHA256Hex(context.OfficialAPIsBytes) ||
		context.ManifestSHA256 != SHA256Hex(renderManifestRowsV1(context.FinalManifest)) {
		return errors.New("completion: context bytes do not match their advertised digests")
	}

	ranks, err := BuildCanonicalRowRanksV1(
		context.FrozenPlanned, context.ReviewedMappings, context.ISALedger, context.Dependencies,
	)
	if err != nil {
		return fmt.Errorf("completion: rebuild canonical ranks: %w", err)
	}
	membership, err := BuildMembershipV1(
		context.ReviewedMappings,
		context.InitialManifest,
		context.FrozenPlanned,
		context.ISALedger,
		context.ExistingMembers,
		ranks,
	)
	if err != nil {
		return fmt.Errorf("completion: rebuild membership: %w", err)
	}
	classification, err := BuildClassificationV1(
		context.FrozenPlanned, context.ReviewedMappings, context.Dependencies, ranks, membership,
	)
	if err != nil {
		return fmt.Errorf("completion: rebuild classification: %w", err)
	}
	if !bytes.Equal(RenderOperationsV1(membership.Operations), RenderOperationsV1(context.Membership.Operations)) ||
		!bytes.Equal(RenderCellsV1(membership.Cells), RenderCellsV1(context.Membership.Cells)) ||
		!bytes.Equal(RenderKernelsV1(membership.Kernels), RenderKernelsV1(context.Membership.Kernels)) ||
		!bytes.Equal(RenderClassificationV1(classification.Rows), RenderClassificationV1(context.Classification.Rows)) ||
		!bytes.Equal(RenderBatchesV1(classification.Batches), RenderBatchesV1(context.Classification.Batches)) ||
		!bytes.Equal(RenderCellsV1(classification.Cells), RenderCellsV1(context.Classification.Cells)) {
		return errors.New("completion: membership or classification differs from canonical regeneration")
	}
	classificationBytes := RenderClassificationV1(classification.Rows)
	if context.ClassificationSHA256 != SHA256Hex(classificationBytes) {
		return errors.New("completion: classification digest does not match canonical regeneration")
	}

	registryDigest, err := EvidenceRegistryDigestV1(context.Registry)
	if err != nil || registryDigest != context.IndexSHA256 {
		return errors.New("completion: evidence registry does not match the frozen return index")
	}
	return validateCompletionRegistryV1(context)
}

func renderManifestRowsV1(rows []ManifestRowV1) []byte {
	var out strings.Builder
	out.WriteString(strings.Join(manifestHeaderV1[:], "\t"))
	out.WriteByte('\n')
	for _, row := range rows {
		out.WriteString(strings.Join(row.Cells[:], "\t"))
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func validateCompletionRegistryV1(context CompletionValidationContextV1) error {
	rows := make(map[string]bool, len(context.FrozenPlanned))
	for _, row := range context.FrozenPlanned {
		if !validID(row.RowKeyV1, "rk-v1-") || rows[row.RowKeyV1] {
			return errors.New("completion: invalid registry row authority")
		}
		rows[row.RowKeyV1] = true
	}
	cells := make(map[string]string, len(context.Classification.Cells))
	for _, cell := range context.Classification.Cells {
		if !validID(cell.CellStorageID, "cell-v1-") || cells[cell.CellStorageID] != "" {
			return errors.New("completion: invalid registry cell authority")
		}
		cells[cell.CellStorageID] = cell.RowKeyV1
	}

	campaigns := make(map[string]EvidenceRecordV1)
	for _, record := range context.Registry.receipts {
		if !rows[record.RowID] || !completionEvidenceRecordValidV1(record, record.RowID, record.CellID, context.AuthoritySHA256) {
			return errors.New("completion: registry contains an invalid or unplanned record")
		}
		if record.CellID != "" && cells[record.CellID] != record.RowID {
			return errors.New("completion: registry record has an invalid row/cell join")
		}
		switch record.StateSubject {
		case "none":
		case "row":
			if record.StorageID != record.RowID {
				return errors.New("completion: row transition has an invalid subject")
			}
		case "backend_cell":
			if record.CellID == "" || record.StorageID != record.CellID {
				return errors.New("completion: cell transition has an invalid subject")
			}
		default:
			return errors.New("completion: registry record has an invalid state subject")
		}
		if prior, ok := campaigns[record.CampaignID]; ok {
			if !sameCampaignEvidenceV1(prior, record) {
				return errors.New("completion: campaign identity maps to unequal evidence contexts")
			}
		} else {
			campaigns[record.CampaignID] = record
		}
	}
	return nil
}

func validCompletionEvidenceContextV1(context CompletionEvidenceContextV1) bool {
	return validID(context.CampaignID, "campaign-v1-") &&
		lowerHex(context.IdentitySetDigest, 64) &&
		lowerHex(context.CommandManifestSHA256, 64) &&
		safeEvidencePart(context.QualificationContractID) &&
		lowerHex(context.QualificationContractSHA256, 64) &&
		safeEvidencePart(context.CorpusID) &&
		lowerHex(context.CorpusSHA256, 64) &&
		context.HostReceiptID == hostReceiptIDV1(context.HostReceiptSHA256) &&
		lowerHex(context.HostReceiptSHA256, 64)
}

func completionEvidenceContextMatchesV1(record EvidenceRecordV1, context CompletionEvidenceContextV1) bool {
	return record.CampaignID == context.CampaignID &&
		record.IdentitySetDigest == context.IdentitySetDigest &&
		record.CommandDigest == context.CommandManifestSHA256 &&
		record.QualificationContractID == context.QualificationContractID &&
		record.QualificationContractDigest == context.QualificationContractSHA256 &&
		record.CorpusID == context.CorpusID &&
		record.CorpusDigest == context.CorpusSHA256 &&
		record.HostReceiptID == context.HostReceiptID &&
		record.HostReceiptDigest == context.HostReceiptSHA256
}
func completionEvidenceRecordValidV1(record EvidenceRecordV1, rowKey, cellID, authority string) bool {
	sourceRole := "new"
	if record.CommandRole == "old" {
		sourceRole = "old"
	} else if record.CommandRole == "host" {
		sourceRole = "host"
	}
	if record.Schema != evidenceSchemaV1 || record.Version != 1 ||
		record.AuthoritySHA256 != authority || record.RowID != rowKey || record.CellID != cellID ||
		record.SourceDirty || !evidenceKindsV1[record.Kind] || !evidenceLanesV1[record.LaneID] ||
		!validID(record.FamilyID, "family-v1-") || !validID(record.CampaignID, "campaign-v1-") ||
		!validID(record.OperationID, "op-v1-") || !validID(record.BatchID, "batch-v1-") ||
		!validID(record.TransactionID, "transaction-v1-") ||
		!canonicalProviderV1(record.Provider) || !safeEvidencePart(record.ProducerID) ||
		!safeEvidencePart(record.CommandID) || !safeEvidencePart(record.OutputID) ||
		record.CommandOrdinal < 1 || !campaignActionsV1[record.CommandAction] ||
		!campaignRolesV1[record.CommandRole] || !allowedActionRoleV1(record.CommandAction, record.CommandRole) ||
		record.SourceRole != sourceRole ||
		!lowerHex(record.SourceCommit, 40) || !lowerHex(record.SourceTree, 40) ||
		!lowerHex(record.SourceParent, 40) || !lowerHex(record.OriginSourceCommit, 40) ||
		!lowerHex(record.OriginSourceTree, 40) || !lowerHex(record.OriginSourceParent, 40) ||
		record.OriginCampaignID != record.CampaignID || record.OriginProducerID != record.ProducerID ||
		record.OriginSourceCommit != record.SourceCommit || record.OriginSourceTree != record.SourceTree ||
		record.OriginSourceParent != record.SourceParent ||
		!lowerHex(record.Digest, 64) || record.Size < 0 || record.Size > MaxRawEvidenceSizeV1 ||
		record.MediaType == "" || !validOriginPath(record.OriginPath) || !validRawEvidenceRecordPathV1(record) ||
		!validID(record.ReceiptID, "receipt-v1-") || record.ReceiptID != ReceiptIDV1(record) ||
		!lowerHex(record.IdentitySetDigest, 64) || !lowerHex(record.CommandDigest, 64) ||
		!safeEvidencePart(record.QualificationContractID) ||
		!lowerHex(record.QualificationContractDigest, 64) || !safeEvidencePart(record.CorpusID) ||
		!lowerHex(record.CorpusDigest, 64) || record.HostReceiptID != hostReceiptIDV1(record.HostReceiptDigest) ||
		!lowerHex(record.HostReceiptDigest, 64) ||
		(record.InitialState != "snapshot_planned" && record.InitialState != "eligible") {
		return false
	}
	if record.StateSubject == "none" &&
		(record.PrerequisiteState != "" || record.CurrentState != "" || record.Disposition != "" ||
			record.GoQualification != "" || len(record.ProofReceiptIDs) != 0 ||
			record.NAReason != "" || record.NASource != "") {
		return false
	}
	return validateTypedEvidenceKey(record) == nil
}

func completionStatusCounts(rows []ManifestRowV1) (implemented, planned, excluded int) {
	for _, row := range rows {
		switch row.Cells[manifestStatusIndex] {
		case "implemented":
			implemented++
		case "planned":
			planned++
		case "excluded":
			excluded++
		}
	}
	return implemented, planned, excluded
}

func manifestRowsByKeyV1(rows []ManifestRowV1) (map[string]ManifestRowV1, error) {
	result := make(map[string]ManifestRowV1, len(rows))
	for _, row := range rows {
		if row.RowKeyV1 == "" || row.RowKeyV1 != RowKeyV1(manifestComposite(row.Cells)) || result[row.RowKeyV1].RowKeyV1 != "" {
			return nil, errors.New("completion: invalid manifest row identity")
		}
		result[row.RowKeyV1] = row
	}
	return result, nil
}

func validateFinalManifestTransitionV1(initial, final, frozen map[string]ManifestRowV1) error {
	if len(initial) != 164 || len(final) != 164 || len(frozen) != 125 {
		return errors.New("completion: manifest sets are incomplete")
	}
	for key, before := range initial {
		after, ok := final[key]
		if !ok {
			return errors.New("completion: final manifest changed row identity")
		}
		if _, planned := frozen[key]; !planned {
			if before.Cells != after.Cells {
				return errors.New("completion: current or excluded manifest row changed")
			}
			continue
		}
		for index := range before.Cells {
			if index == manifestStatusIndex || index == manifestMilestoneIndex {
				continue
			}
			if before.Cells[index] != after.Cells[index] {
				return errors.New("completion: planned manifest contract changed during publication")
			}
		}
		if before.Cells[manifestStatusIndex] != "planned" || before.Cells[manifestMilestoneIndex] != "future-upstream-api" ||
			after.Cells[manifestStatusIndex] != "implemented" || after.Cells[manifestMilestoneIndex] != "611becc-current-api" {
			return errors.New("completion: planned manifest state did not publish exactly once")
		}
	}
	return nil
}

func completionRowLess(left, right CompletionRowV1) bool {
	return left.ManifestOrdinal < right.ManifestOrdinal ||
		left.ManifestOrdinal == right.ManifestOrdinal && left.RowKeyV1 < right.RowKeyV1
}

func validateCompletionMembershipV1(membership MembershipV1, frozen map[string]ManifestRowV1) error {
	if len(membership.Cells) != 500 || len(membership.Operations) != 23 {
		return errors.New("completion: incomplete frozen membership")
	}
	operations := make(map[int]FinalOperationV1, len(membership.Operations))
	for index, operation := range membership.Operations {
		expectedID, err := LedgerOperationIDV1(operation.ISAOrdinal, operation.ISASemanticOperationExact)
		if err != nil || operation.ISAOrdinal != index+1 || operation.SemanticOperationID != expectedID ||
			operation.InitialRowCount < 0 || operation.InitialCellCount < 0 ||
			operation.ExistingRowCount < 0 || operation.EvidenceAnchor == "" {
			return errors.New("completion: invalid frozen operation identity")
		}
		operations[operation.ISAOrdinal] = operation
	}

	operationRows := make(map[int]map[string]bool, len(operations))
	operationCells := make(map[int]int, len(operations))
	kernelGroups := make(map[string][]FinalCellV1)
	seenCells := make(map[string]bool, len(membership.Cells))
	for _, cell := range membership.Cells {
		frozenRow, ok := frozen[cell.RowKeyV1]
		key, keyErr := CellKeyV1(cell.RowKeyV1, cell.Backend)
		if !ok || keyErr != nil || key.StorageID != cell.CellStorageID || seenCells[cell.CellStorageID] ||
			cell.ManifestOrdinal != frozenRow.PlannedOrdinal || cell.CanonicalRowRank < 0 ||
			cell.CanonicalRowRank >= len(frozen) || cell.FamilyContractDisplayID == "" ||
			cell.KernelBatchDisplayID != "" || cell.KernelBatchStorageID != "" ||
			cell.EvidenceBatchDisplayID != "" || cell.EvidenceBatchStorageID != "" {
			return errors.New("completion: invalid frozen membership cell")
		}
		seenCells[cell.CellStorageID] = true

		if cell.ISAOrdinalOrScalar == "scalar" {
			operationID, err := ScalarOperationIDV1(cell.RowKeyV1)
			if err != nil || cell.SemanticOperationID != operationID {
				return errors.New("completion: invalid scalar operation join")
			}
		} else {
			ordinal, err := strconv.Atoi(cell.ISAOrdinalOrScalar)
			operation, found := operations[ordinal]
			if err != nil || !found || strconv.Itoa(ordinal) != cell.ISAOrdinalOrScalar ||
				cell.SemanticOperationID != operation.SemanticOperationID {
				return errors.New("completion: invalid ISA operation join")
			}
			if operationRows[ordinal] == nil {
				operationRows[ordinal] = map[string]bool{}
			}
			operationRows[ordinal][cell.RowKeyV1] = true
			operationCells[ordinal]++
		}

		switch cell.BackendOutcome {
		case "eligible":
			symbol, err := SymbolKeyV1(cell.Backend, cell.DirectSymbol)
			suffix := ""
			switch cell.Backend {
			case "westmere":
				suffix = "Westmere"
			case "haswell":
				suffix = "Haswell"
			case "archsimd":
				suffix = "Archsimd"
			case "neon":
				suffix = "NEON"
			}
			expectedDirect := lowerFirst(frozenRow.Cells[manifestGoSymbolIndex]) + suffix
			if err != nil || suffix == "" || cell.DirectSymbol != expectedDirect ||
				symbol.StorageID != cell.DirectSymbolStorageID ||
				!validID(cell.SharedKernelID, "shared-kernel-v1-") ||
				!validID(cell.KernelOwnerCellKey, "cell-v1-") ||
				cell.BackendNAReason != "" || cell.BackendEvidenceAnchor == "" {
				return errors.New("completion: invalid eligible membership cell")
			}
			group := cell.Backend + "\x00" + cell.SharedKernelID
			kernelGroups[group] = append(kernelGroups[group], cell)
		case "not_applicable":
			if cell.BackendNAReason == "" || cell.BackendEvidenceAnchor == "" ||
				cell.DirectSymbol != "" || cell.DirectSymbolStorageID != "" ||
				cell.SharedKernelID != "" || cell.KernelOwnerCellKey != "" ||
				cell.KernelOwnerDependencyID != "" {
				return errors.New("completion: invalid not-applicable membership cell")
			}
		default:
			return errors.New("completion: invalid membership outcome")
		}
	}
	if len(seenCells) != 500 {
		return errors.New("completion: incomplete frozen membership cells")
	}

	for ordinal, operation := range operations {
		if operation.InitialRowCount != len(operationRows[ordinal]) ||
			operation.InitialCellCount != operationCells[ordinal] {
			return errors.New("completion: frozen operation counts do not close")
		}
		switch operation.Reconciliation {
		case "initial_members":
			if operation.InitialRowCount == 0 {
				return errors.New("completion: invalid initial operation reconciliation")
			}
		case "existing_only":
			if operation.InitialRowCount != 0 || operation.ExistingRowCount == 0 {
				return errors.New("completion: invalid existing operation reconciliation")
			}
		case "zero_initial_explained":
			if operation.InitialRowCount != 0 || operation.ExistingRowCount != 0 ||
				!strings.HasPrefix(operation.EvidenceAnchor, "Pin:") {
				return errors.New("completion: invalid zero-initial operation reconciliation")
			}
		default:
			return errors.New("completion: unknown operation reconciliation")
		}
	}

	kernels := make(map[string]KernelRegistryV1, len(membership.Kernels))
	for _, kernel := range membership.Kernels {
		group := kernel.Backend + "\x00" + kernel.SharedKernelID
		computedID, err := SharedKernelIDV1(kernel.Backend, kernel.CanonicalKernelName)
		if err != nil || computedID != kernel.SharedKernelID || kernels[group].SharedKernelID != "" {
			return errors.New("completion: invalid frozen kernel identity")
		}
		kernels[group] = kernel
	}
	if len(kernels) != len(kernelGroups) {
		return errors.New("completion: frozen kernel registry is incomplete")
	}
	for group, members := range kernelGroups {
		sort.Slice(members, func(i, j int) bool { return ownerLess(members[i], members[j]) })
		owner := members[0]
		kernel, ok := kernels[group]
		if !ok || kernel.FamilyContractDisplayID != owner.FamilyContractDisplayID ||
			kernel.SemanticOperationID != owner.SemanticOperationID ||
			kernel.KernelOwnerCellKey != owner.CellStorageID || kernel.MemberCount != len(members) {
			return errors.New("completion: frozen kernel registry does not match its owner")
		}
		for index, member := range members {
			expectedDependency := owner.CellStorageID
			if index == 0 {
				expectedDependency = ""
			}
			if member.KernelOwnerCellKey != owner.CellStorageID ||
				member.KernelOwnerDependencyID != expectedDependency {
				return errors.New("completion: frozen kernel ownership is not canonical")
			}
		}
	}
	return nil
}
func validateCompletionClassificationV1(classification ClassificationV1, membership MembershipV1, frozen map[string]ManifestRowV1) error {
	if err := validateCompletionMembershipV1(membership, frozen); err != nil {
		return err
	}
	baseCells := make(map[string]int, len(membership.Cells))
	for index, cell := range membership.Cells {
		if cell.CellStorageID == "" || baseCells[cell.CellStorageID] != 0 {
			return errors.New("completion: invalid base membership cells")
		}
		baseCells[cell.CellStorageID] = index + 1
	}
	if len(baseCells) != 500 {
		return errors.New("completion: invalid base membership cells")
	}
	cellByKey := make(map[string]int, len(classification.Cells))
	seenCells := make(map[string]bool, len(classification.Cells))
	for index, cell := range classification.Cells {
		key, err := CellKeyV1(cell.RowKeyV1, cell.Backend)
		baseIndex, ok := baseCells[cell.CellStorageID]
		if err != nil || !ok || seenCells[cell.CellStorageID] || key.StorageID != cell.CellStorageID {
			return errors.New("completion: invalid classification cell identity")
		}
		base := membership.Cells[baseIndex-1]
		classificationBase := cell
		classificationBase.KernelBatchDisplayID = ""
		classificationBase.KernelBatchStorageID = ""
		classificationBase.EvidenceBatchDisplayID = ""
		classificationBase.EvidenceBatchStorageID = ""
		if classificationBase != base {
			return errors.New("completion: classification cell differs from frozen membership")
		}
		seenCells[cell.CellStorageID] = true
		cellByKey[cell.CellStorageID] = index
	}
	if len(cellByKey) != 500 {
		return errors.New("completion: incomplete classification cells")
	}
	if err := validateClassificationEmission(classification, cellByKey); err != nil {
		return fmt.Errorf("completion: invalid classification emission: %w", err)
	}

	rows := make(map[string]ClassificationRowV1, len(classification.Rows))
	families := map[string][]ClassificationRowV1{}
	ranks := map[int]bool{}
	for _, row := range classification.Rows {
		frozenRow, ok := frozen[row.RowKeyV1]
		if !ok || rows[row.RowKeyV1].RowKeyV1 != "" || ranks[row.CanonicalRowRank] ||
			row.CanonicalRowRank < 0 || row.CanonicalRowRank != row.CanonicalDependencyRank ||
			row.ManifestOrdinal != frozenRow.PlannedOrdinal || row.GoSymbol != frozenRow.Cells[manifestGoSymbolIndex] {
			return errors.New("completion: classification row differs from frozen manifest")
		}
		rows[row.RowKeyV1] = row
		ranks[row.CanonicalRowRank] = true
		families[row.FamilyContractDisplayID] = append(families[row.FamilyContractDisplayID], row)
	}
	if len(rows) != 125 || len(ranks) != 125 {
		return errors.New("completion: incomplete classification rows")
	}
	for rank := range 125 {
		if !ranks[rank] {
			return errors.New("completion: noncanonical classification rank")
		}
	}
	batches := make(map[string]BatchRecordV1, len(classification.Batches))
	for _, batch := range classification.Batches {
		key, err := BatchKeyV1(batch.Kind, batch.DisplayID, batch.MemberStorageKeys)
		if err != nil || key.StorageID != batch.StorageID ||
			hex.EncodeToString(EncodeTupleV1(batch.MemberStorageKeys...)) != batch.MemberTupleHex ||
			batches[batch.StorageID].StorageID != "" {
			return errors.New("completion: invalid batch typed key")
		}
		batches[batch.StorageID] = batch
	}
	for _, batch := range batches {
		switch batch.Kind {
		case "kernel":
			owners := map[string]FinalCellV1{}
			for _, cell := range classification.Cells {
				if cell.BackendOutcome == "eligible" && cell.KernelBatchStorageID == batch.StorageID {
					ownerIndex, ok := cellByKey[cell.KernelOwnerCellKey]
					if !ok {
						return errors.New("completion: kernel batch references an unknown owner")
					}
					owners[cell.KernelOwnerCellKey] = classification.Cells[ownerIndex]
				}
			}
			expected := make([]FinalCellV1, 0, len(owners))
			for _, owner := range owners {
				expected = append(expected, owner)
			}
			sort.Slice(expected, func(i, j int) bool {
				if expected[i].SemanticOperationID != expected[j].SemanticOperationID {
					return expected[i].SemanticOperationID < expected[j].SemanticOperationID
				}
				if expected[i].SharedKernelID != expected[j].SharedKernelID {
					return expected[i].SharedKernelID < expected[j].SharedKernelID
				}
				return expected[i].KernelOwnerCellKey < expected[j].KernelOwnerCellKey
			})
			keys := make([]string, len(expected))
			for i := range expected {
				keys[i] = expected[i].KernelOwnerCellKey
			}
			if !sameStrings(batch.MemberStorageKeys, keys) {
				return errors.New("completion: kernel batch membership is not exact")
			}
		case "evidence":
			expected := []FinalCellV1{}
			for _, cell := range classification.Cells {
				if cell.BackendOutcome == "eligible" && cell.EvidenceBatchStorageID == batch.StorageID {
					expected = append(expected, cell)
				}
			}
			sort.Slice(expected, func(i, j int) bool {
				if expected[i].CanonicalRowRank != expected[j].CanonicalRowRank {
					return expected[i].CanonicalRowRank < expected[j].CanonicalRowRank
				}
				if expected[i].SemanticOperationID != expected[j].SemanticOperationID {
					return expected[i].SemanticOperationID < expected[j].SemanticOperationID
				}
				if expected[i].SharedKernelID != expected[j].SharedKernelID {
					return expected[i].SharedKernelID < expected[j].SharedKernelID
				}
				return expected[i].CellStorageID < expected[j].CellStorageID
			})
			keys := make([]string, len(expected))
			for i := range expected {
				keys[i] = expected[i].CellStorageID
			}
			if !sameStrings(batch.MemberStorageKeys, keys) {
				return errors.New("completion: evidence batch membership is not exact")
			}
		}
	}
	for family, members := range families {
		sort.Slice(members, func(i, j int) bool { return members[i].CanonicalRowRank < members[j].CanonicalRowRank })
		transactionMembers := make([]string, len(members))
		for i := range members {
			transactionMembers[i] = members[i].RowKeyV1
		}
		for start, sequence := 0, 1; start < len(members); start, sequence = start+12, sequence+1 {
			end := min(start+12, len(members))
			row := members[start]
			batch, ok := batches[row.ScalarBatchStorageID]
			if !ok || batch.Kind != "scalar" || batch.FamilyContractDisplayID != family ||
				batch.Sequence != sequence || batch.DisplayID != row.ScalarBatchDisplayID ||
				!sameStrings(batch.MemberStorageKeys, transactionMembers[start:end]) {
				return errors.New("completion: scalar batch does not close canonical family group")
			}
			for _, member := range members[start:end] {
				if member.ScalarBatchStorageID != batch.StorageID || member.ScalarBatchDisplayID != batch.DisplayID {
					return errors.New("completion: scalar batch membership is not exact")
				}
			}
		}
		transactionDisplay := "AP-v1-" + family
		transaction, err := TransactionKeyV1(transactionDisplay, transactionMembers)
		if err != nil {
			return err
		}
		for _, member := range members {
			if member.APITransactionDisplayID != transactionDisplay || member.APITransactionStorageID != transaction.StorageID {
				return errors.New("completion: transaction does not close canonical family")
			}
		}
	}
	return nil
}

func validateCompletionCellV1(classification ClassificationRowV1, cell CompletionCellV1, members map[string]FinalCellV1, context CompletionValidationContextV1, seenCells map[string]bool, seenReceipts map[string]string) error {
	rowKey := classification.RowKeyV1
	key, err := CellKeyV1(rowKey, cell.Backend)
	member, ok := members[cell.CellStorageID]
	if err != nil || !ok || seenCells[cell.CellStorageID] || cell.CellStorageID != key.StorageID ||
		member.Backend != cell.Backend || member.RowKeyV1 != rowKey || member.BackendOutcome != cell.Outcome ||
		!validCompletionEvidenceContextV1(cell.EvidenceContext) {
		return errors.New("completion: invalid cell")
	}
	seenCells[cell.CellStorageID] = true

	if cell.Outcome == "not_applicable" {
		if cell.State != "not_applicable" || cell.NAReason != member.BackendNAReason ||
			cell.NASource != member.BackendEvidenceAnchor || cell.DirectSymbol != "" ||
			cell.DirectSymbolStorageID != "" || cell.KernelBatchStorageID != "" ||
			cell.EvidenceBatchStorageID != "" || cell.Qualification != "" ||
			cell.Disposition != "not_applicable" {
			return errors.New("completion: invalid not-applicable cell")
		}
	} else if cell.Outcome == "eligible" {
		symbol, symbolErr := SymbolKeyV1(cell.Backend, member.DirectSymbol)
		qualified := cell.State == "selected" && cell.Qualification == "pass" ||
			cell.State == "direct_only" && (cell.Qualification == "fail" || cell.Qualification == "inconclusive")
		if symbolErr != nil || cell.State != cell.Disposition || !qualified ||
			cell.DirectSymbol != member.DirectSymbol || cell.DirectSymbolStorageID != symbol.StorageID ||
			cell.KernelBatchStorageID != member.KernelBatchStorageID ||
			cell.EvidenceBatchStorageID != member.EvidenceBatchStorageID || cell.NAReason != "" || cell.NASource != "" {
			return errors.New("completion: invalid eligible cell")
		}
	} else {
		return errors.New("completion: unknown cell outcome")
	}

	required, err := requiredCellReceiptIDsV1(context.Registry, classification, cell, member, context)
	if err != nil {
		return err
	}
	if !sameStrings(cell.ReceiptIDs, required) {
		return errors.New("completion: cell receipt closure is not canonical or complete")
	}
	for _, id := range cell.ReceiptIDs {
		if owner, reused := seenReceipts[id]; reused && owner != cell.CellStorageID {
			return fmt.Errorf("completion: proof %q is reused", id)
		}
		seenReceipts[id] = cell.CellStorageID
	}
	return nil
}

func completionEvidenceUsesFinalSourceV1(record EvidenceRecordV1, context CompletionValidationContextV1) bool {
	return record.SourceRole == "new" &&
		record.SourceCommit == context.SourceCommit &&
		record.SourceTree == context.SourceTree &&
		record.SourceParent == context.SourceParent
}

func completionCellEvidenceBindingMatchesV1(record EvidenceRecordV1, classification ClassificationRowV1, cell CompletionCellV1, member FinalCellV1, familyID string) bool {
	if record.FamilyID != familyID || record.Provider != cell.Backend ||
		record.Backend != cell.Backend || record.OperationID != member.SemanticOperationID ||
		record.TransactionID != classification.APITransactionStorageID {
		return false
	}
	if cell.Outcome == "not_applicable" {
		return record.SymbolID == "" && record.DirectSymbol == ""
	}
	return record.SymbolID == cell.DirectSymbolStorageID &&
		record.DirectSymbol == cell.DirectSymbol &&
		record.BatchID == cell.EvidenceBatchStorageID
}
func requiredCellReceiptIDsV1(registry *EvidenceRegistryV1, classification ClassificationRowV1, cell CompletionCellV1, member FinalCellV1, context CompletionValidationContextV1) ([]string, error) {
	rowKey := classification.RowKeyV1
	family, err := FamilyKeyV1(member.FamilyContractDisplayID)
	if err != nil {
		return nil, err
	}
	transitions := map[string]EvidenceRecordV1{}
	for _, record := range registry.receipts {
		if record.StateSubject != "backend_cell" || record.StorageID != cell.CellStorageID ||
			!completionEvidenceContextMatchesV1(record, cell.EvidenceContext) {
			continue
		}
		if !completionEvidenceRecordValidV1(record, rowKey, cell.CellStorageID, context.AuthoritySHA256) ||
			!completionCellEvidenceBindingMatchesV1(record, classification, cell, member, family.StorageID) ||
			!completionEvidenceUsesFinalSourceV1(record, context) ||
			transitions[record.CurrentState].ReceiptID != "" {
			return nil, errors.New("completion: stale, duplicate, or unbound cell transition")
		}
		transitions[record.CurrentState] = record
	}

	states := []string{"not_applicable"}
	prerequisites := map[string]string{"not_applicable": "eligible"}
	if cell.Outcome == "eligible" {
		states = []string{"direct_built", "hard_gates_green", "dispatch_candidate", cell.State}
		prerequisites = map[string]string{
			"direct_built":       "eligible",
			"hard_gates_green":   "direct_built",
			"dispatch_candidate": "hard_gates_green",
			cell.State:           "dispatch_candidate",
		}
	}
	if len(transitions) != len(states) {
		return nil, fmt.Errorf("completion: cell state chain is incomplete or has extras for %s: got %d transitions, want %d", cell.CellStorageID, len(transitions), len(states))
	}
	terminal, ok := transitions[cell.State]
	if !ok {
		return nil, errors.New("completion: cell terminal transition is missing")
	}

	required := map[string]bool{}
	for _, state := range states {
		transition, ok := transitions[state]
		if !ok || transition.PrerequisiteState != prerequisites[state] {
			return nil, errors.New("completion: cell state chain is incomplete or out of order")
		}
		if state == "not_applicable" {
			if transition.Kind != "not-applicable" || transition.Disposition != cell.Disposition ||
				transition.GoQualification != "" || transition.NAReason != cell.NAReason ||
				transition.NASource != cell.NASource {
				return nil, errors.New("completion: not-applicable transition does not close")
			}
		} else if transition.Kind != "state-transition" {
			return nil, errors.New("completion: cell transition has an invalid proof kind")
		} else if state == cell.State {
			if transition.Disposition != cell.Disposition || transition.GoQualification != cell.Qualification {
				return nil, errors.New("completion: terminal disposition or qualification does not close")
			}
		} else if transition.Disposition != "" || transition.GoQualification != "" {
			return nil, errors.New("completion: nonterminal cell transition carries a terminal result")
		}
		if !sameStateCampaignEvidenceV1(terminal, transition) {
			return nil, errors.New("completion: cell state chain crosses campaign or source identity")
		}
		required[transition.ReceiptID] = true
		for _, proofID := range transition.ProofReceiptIDs {
			proof, found := registry.Receipt(proofID)
			if !found || proof.ReceiptID == transition.ReceiptID ||
				!completionEvidenceRecordValidV1(proof, rowKey, cell.CellStorageID, context.AuthoritySHA256) ||
				!completionCellEvidenceBindingMatchesV1(proof, classification, cell, member, family.StorageID) ||
				!sameProofContext(transition, proof) ||
				proof.SourceRole == "new" && !completionEvidenceUsesFinalSourceV1(proof, context) {
				return nil, errors.New("completion: state proof is missing, stale, or inherited")
			}
			required[proofID] = true
		}
	}
	return canonicalReceiptSetV1(required), nil
}

func validateFamilyPublicationEvidenceV1(classification ClassificationV1, completionRows []CompletionRowV1, registry *EvidenceRegistryV1, context CompletionValidationContextV1) error {
	families := make(map[string][]ClassificationRowV1)
	for _, row := range classification.Rows {
		families[row.FamilyContractDisplayID] = append(families[row.FamilyContractDisplayID], row)
	}
	completionByRow := make(map[string]CompletionRowV1, len(completionRows))
	for _, row := range completionRows {
		completionByRow[row.RowKeyV1] = row
	}
	for family, rows := range families {
		var publication EvidenceRecordV1
		for _, row := range rows {
			closure, ok := completionByRow[row.RowKeyV1]
			if !ok {
				return fmt.Errorf("completion: family %q is missing a completion row", family)
			}
			var found EvidenceRecordV1
			for _, record := range registry.receipts {
				if record.StateSubject != "row" || record.StorageID != row.RowKeyV1 ||
					record.CurrentState != "family_published" ||
					!completionEvidenceContextMatchesV1(record, closure.EvidenceContext) {
					continue
				}
				if found.ReceiptID != "" {
					return fmt.Errorf("completion: family %q has duplicate selected row publication evidence", family)
				}
				found = record
			}
			if found.ReceiptID == "" ||
				!completionEvidenceRecordValidV1(found, row.RowKeyV1, found.CellID, context.AuthoritySHA256) ||
				!completionEvidenceUsesFinalSourceV1(found, context) ||
				found.PrerequisiteState != "scalar_green" ||
				found.OperationID != row.SemanticOperationID ||
				found.TransactionID != row.APITransactionStorageID {
				return fmt.Errorf("completion: family %q has invalid row publication evidence", family)
			}
			if publication.ReceiptID == "" {
				publication = found
			} else if !sameStateCampaignEvidenceV1(publication, found) ||
				publication.TransactionID != found.TransactionID {
				return fmt.Errorf("completion: family %q was not published by one campaign transaction", family)
			}
		}
	}
	return nil
}
func requiredRowReceiptIDsV1(registry *EvidenceRegistryV1, classification ClassificationRowV1, evidence CompletionEvidenceContextV1, context CompletionValidationContextV1) ([]string, error) {
	rowKey := classification.RowKeyV1
	family, err := FamilyKeyV1(classification.FamilyContractDisplayID)
	if err != nil {
		return nil, err
	}
	transitions := map[string]EvidenceRecordV1{}
	for _, record := range registry.receipts {
		if record.StateSubject != "row" || record.StorageID != rowKey ||
			!completionEvidenceContextMatchesV1(record, evidence) {
			continue
		}
		if !completionEvidenceRecordValidV1(record, rowKey, record.CellID, context.AuthoritySHA256) ||
			record.FamilyID != family.StorageID ||
			record.OperationID != classification.SemanticOperationID ||
			record.TransactionID != classification.APITransactionStorageID ||
			!completionEvidenceUsesFinalSourceV1(record, context) ||
			transitions[record.CurrentState].ReceiptID != "" {
			return nil, errors.New("completion: stale, duplicate, or unbound row transition")
		}
		transitions[record.CurrentState] = record
	}

	states := []string{"scalar_private", "scalar_green", "family_published", "complete"}
	prerequisites := map[string]string{
		"scalar_private":   "snapshot_planned",
		"scalar_green":     "scalar_private",
		"family_published": "scalar_green",
		"complete":         "family_published",
	}
	if len(transitions) != len(states) {
		return nil, errors.New("completion: row state chain is incomplete or has extras")
	}

	required := map[string]bool{}
	for _, state := range states {
		transition, ok := transitions[state]
		if !ok || transition.Kind != "state-transition" ||
			transition.PrerequisiteState != prerequisites[state] ||
			transition.Disposition != "" || transition.GoQualification != "" {
			return nil, errors.New("completion: row state chain is incomplete or invalid")
		}
		if (state == "scalar_private" || state == "scalar_green") &&
			transition.BatchID != classification.ScalarBatchStorageID {
			return nil, errors.New("completion: scalar row transition has the wrong batch")
		}
		required[transition.ReceiptID] = true
		for _, proofID := range transition.ProofReceiptIDs {
			proof, found := registry.Receipt(proofID)
			if !found || proof.ReceiptID == transition.ReceiptID ||
				!completionEvidenceRecordValidV1(proof, rowKey, transition.CellID, context.AuthoritySHA256) ||
				proof.FamilyID != family.StorageID ||
				proof.OperationID != classification.SemanticOperationID ||
				proof.TransactionID != classification.APITransactionStorageID ||
				!completionEvidenceUsesFinalSourceV1(proof, context) ||
				!sameProofContext(transition, proof) {
				return nil, errors.New("completion: row proof is missing, stale, or inherited")
			}
			required[proofID] = true
		}
	}
	return canonicalReceiptSetV1(required), nil
}

func sameStateCampaignEvidenceV1(left, right EvidenceRecordV1) bool {
	return sameCampaignEvidenceV1(left, right) &&
		left.SourceRole == right.SourceRole && left.SourceCommit == right.SourceCommit &&
		left.SourceTree == right.SourceTree && left.SourceParent == right.SourceParent &&
		left.OriginSourceCommit == right.OriginSourceCommit &&
		left.OriginSourceTree == right.OriginSourceTree &&
		left.OriginSourceParent == right.OriginSourceParent
}

func canonicalReceiptSetV1(receipts map[string]bool) []string {
	ids := make([]string, 0, len(receipts))
	for id := range receipts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// CanonicalCompletionRowsV1 orders rows by original planned manifest ordinal.
func CanonicalCompletionRowsV1(rows []CompletionRowV1) []CompletionRowV1 {
	result := append([]CompletionRowV1(nil), rows...)
	sort.Slice(result, func(i, j int) bool { return completionRowLess(result[i], result[j]) })
	return result
}
