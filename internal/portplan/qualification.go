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

const qualificationSchemaV1 = "simdutf-qualification-contract-v1"

var qualificationPolicyHeaderV1 = [...]string{
	"schema_version", "ordinal", "family_contract_id", "required_corpus_ids",
	"promotion_sequence", "old_sample_count", "new_sample_count", "sample_order",
	"required_allocs_per_op", "minimum_bulk_win_bps", "maximum_protected_slowdown_bps",
	"maximum_p_value_millionths", "no_statistically_significant_slowdown",
	"inconclusive_outcome", "failure_outcome", "selected_outcome", "required_evidence_kinds",
}

var qualificationBenchmarkSourceHeaderV1 = [...]string{
	"schema_version", "ordinal", "operation_ordinal", "operation_id", "public_wrapper", "family_contract_id",
	"corpus_id", "class", "input_unit", "size_units", "set_bytes_denominator", "options_id", "benchmark_name",
}

var qualificationProviderOrderV1 = [...]string{"westmere", "haswell", "archsimd", "neon"}

var qualificationEvidenceKindsV1 = [...]string{
	"incumbent-benchmark", "candidate-benchmark", "benchstat", "provider-guard", "selector", "final-selector",
}

var qualificationFamilyCorporaV1 = map[string][]string{
	"FC-v1-helper-validation": {"Q-byte-zero", "Q-u16-zero", "Q-u32-zero", "Q-towellformed-cpp-capture"},
	"FC-v1-latin1-source":     {"Q-latin1-ramp"},
	"FC-v1-utf8-source":       {"Q-byte-zero", "Q-emoji", "Q-arabic-lipsum"},
	"FC-v1-utf16-source":      {"Q-u16-zero", "Q-emoji-utf16le", "Q-emoji-utf16be", "Q-emoji-utf16-native"},
	"FC-v1-utf32-source":      {"Q-u32-zero", "Q-emoji-utf32-native"},
	"FC-v1-detection":         {"Q-detection-valid"},
	"FC-v1-find":              {"Q-find-byte", "Q-find-u16le", "Q-find-cpp-capture"},
	"FC-v1-base64":            {"Q-emoji", "Q-dns-normalized"},
}

// QualificationPolicyV1 is one immutable family-wide qualification policy.
type QualificationPolicyV1 struct {
	Ordinal                                              int
	FamilyContractID, PromotionSequence, SampleOrder     string
	RequiredCorpusIDs, RequiredEvidenceKinds             []string
	OldSampleCount, NewSampleCount, RequiredAllocsPerOp  int
	MinimumBulkWinBPS, MaximumProtectedSlowdownBPS       int
	MaximumPValueMillionths                              int
	NoStatisticallySignificantSlowdown                   bool
	InconclusiveOutcome, FailureOutcome, SelectedOutcome string
}

// QualificationTierV1 binds a tier to the only runtime symbols accepted by the
// pre-loop provider guard. Scalar is always first.
type QualificationTierV1 struct {
	Tier               string   `json:"tier"`
	RuntimeIdentifiers []string `json:"runtime_identifiers"`
}

// QualificationBenchmarkRowV1 is one immutable public-dispatch benchmark row.
type QualificationBenchmarkRowV1 struct {
	Ordinal               int                   `json:"ordinal"`
	OperationOrdinal      int                   `json:"operation_ordinal"`
	OperationID           string                `json:"operation_id"`
	PublicWrapper         string                `json:"public_wrapper"`
	BenchmarkName         string                `json:"benchmark_name"`
	CorpusOrdinal         int                   `json:"corpus_ordinal"`
	CorpusID              string                `json:"corpus_id"`
	CorpusByteLength      int                   `json:"corpus_byte_length"`
	CorpusSHA256          string                `json:"corpus_sha256"`
	CorpusSourceIdentity  string                `json:"corpus_source_identity"`
	CorpusMaterialization string                `json:"corpus_materialization"`
	ValidityClass         string                `json:"validity_class"`
	Class                 string                `json:"class"`
	InputUnit             string                `json:"input_unit"`
	SizeUnits             int                   `json:"size_units"`
	SetBytesDenominator   int                   `json:"set_bytes_denominator"`
	OptionsID             string                `json:"options_id"`
	OptionsTupleHex       string                `json:"options_tuple_hex"`
	DestinationPolicy     string                `json:"destination_policy"`
	ResultSink            string                `json:"result_sink"`
	SetupBoundary         string                `json:"setup_boundary"`
	TimedBoundary         string                `json:"timed_boundary"`
	Protected             bool                  `json:"protected"`
	IntendedBulk          bool                  `json:"intended_bulk"`
	Tiers                 []QualificationTierV1 `json:"tiers"`
	GoDispatchStatus      string                `json:"go_dispatch_status"`
	PinnedCPPStatus       string                `json:"pinned_cpp_status"`
	PinnedCPPReason       string                `json:"pinned_cpp_reason"`
	PinnedCPPProcedure    string                `json:"pinned_cpp_procedure"`
	PinnedCPPOptions      string                `json:"pinned_cpp_options"`
	RowDigest             string                `json:"row_digest"`
}

// QualificationContractV1 freezes a complete family benchmark contract before
// a provider candidate is created.
type QualificationContractV1 struct {
	Schema                             string                           `json:"schema"`
	Version                            int                              `json:"version"`
	QualificationContractID            string                           `json:"qualification_contract_id"`
	FamilyContractID                   string                           `json:"family_contract_id"`
	ScalarPublicationCommit            string                           `json:"scalar_publication_commit"`
	ScalarPublicationTree              string                           `json:"scalar_publication_tree"`
	ScalarPublicationParent            string                           `json:"scalar_publication_parent"`
	PolicySHA256                       string                           `json:"policy_sha256"`
	ClassificationSHA256               string                           `json:"classification_sha256"`
	BenchmarkSourceSHA256              string                           `json:"benchmark_source_sha256"`
	BenchmarkOrderSHA256               string                           `json:"benchmark_order_sha256"`
	CorpusContractSHA256               string                           `json:"corpus_contract_sha256"`
	Providers                          []string                         `json:"providers"`
	OldSampleCount                     int                              `json:"old_sample_count"`
	NewSampleCount                     int                              `json:"new_sample_count"`
	SampleOrder                        string                           `json:"sample_order"`
	RequiredAllocsPerOp                int                              `json:"required_allocs_per_op"`
	MinimumBulkWinBPS                  int                              `json:"minimum_bulk_win_bps"`
	MaximumProtectedSlowdownBPS        int                              `json:"maximum_protected_slowdown_bps"`
	MaximumPValueMillionths            int                              `json:"maximum_p_value_millionths"`
	NoStatisticallySignificantSlowdown bool                             `json:"no_statistically_significant_slowdown"`
	InconclusiveOutcome                string                           `json:"inconclusive_outcome"`
	FailureOutcome                     string                           `json:"failure_outcome"`
	SelectedOutcome                    string                           `json:"selected_outcome"`
	RequiredEvidenceKinds              []string                         `json:"required_evidence_kinds"`
	Rows                               []QualificationBenchmarkRowV1    `json:"rows"`
	NoAcceleratedOperations            []QualificationOperationStatusV1 `json:"no_accelerated_operations"`
}

// QualificationOperationStatusV1 closes a semantic operation that has no
// eligible accelerated classification cells.
type QualificationOperationStatusV1 struct {
	OperationOrdinal int    `json:"operation_ordinal"`
	OperationID      string `json:"operation_id"`
	PublicWrapper    string `json:"public_wrapper"`
	Status           string `json:"status"`
}

// QualificationPrimaryKeyV1 identifies one frozen benchmark row.
type QualificationPrimaryKeyV1 struct {
	OperationID string
	CorpusID    string
	Class       string
	SizeUnits   int
	OptionsID   string
}

// QualificationValidationContextV1 binds qualification validation to its frozen
// classification, corpus, benchmark source, and benchmark ordering inputs.
type QualificationValidationContextV1 struct {
	ClassificationRows   []ClassificationRowV1
	ClassificationCells  []FinalCellV1
	Corpora              []CorpusContractRecordV1
	ClassificationBytes  []byte
	BenchmarkSourceBytes []byte
	BenchmarkOrderBytes  []byte
	CorpusContractBytes  []byte
}

// RenderQualificationBenchmarkOrderV1 renders the exact frozen primary-key
// population in execution order.
func RenderQualificationBenchmarkOrderV1(keys []QualificationPrimaryKeyV1) ([]byte, error) {
	if len(keys) == 0 {
		return nil, errors.New("qualification benchmark order: empty key population")
	}
	lines := make([]string, 1, len(keys)+1)
	lines[0] = strings.Join(qualificationBenchmarkSourceHeaderV1[:], "\t")
	for i, key := range keys {
		if !validID(key.OperationID, "op-v1-") || !validQualificationClass(key.Class) || key.SizeUnits < 1 ||
			!safeQualificationTSVFieldV1(key.CorpusID) || !safeEvidencePart(key.OptionsID) {
			return nil, fmt.Errorf("qualification benchmark order: invalid key %d", i+1)
		}
		lines = append(lines, strings.Join([]string{
			"v1", strconv.Itoa(i + 1), "1", key.OperationID, "Unspecified", "FC-v1-unspecified",
			key.CorpusID, key.Class, "byte", strconv.Itoa(key.SizeUnits), strconv.Itoa(key.SizeUnits),
			key.OptionsID, "BenchmarkDispatchQualification/Unspecified/" + key.CorpusID + "/" + key.Class + "/" + fmt.Sprintf("%04d", key.SizeUnits),
		}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

// ParseQualificationBenchmarkOrderV1 parses canonical primary keys from the
// reviewed benchmark row-spec TSV.
func ParseQualificationBenchmarkOrderV1(data []byte) ([]QualificationPrimaryKeyV1, error) {
	lines, err := parseLines(data, "qualification benchmark order")
	if err != nil {
		return nil, err
	}
	if err := validateHeader(lines[0], qualificationBenchmarkSourceHeaderV1[:], "qualification benchmark order"); err != nil {
		return nil, err
	}
	keys := make([]QualificationPrimaryKeyV1, 0, len(lines)-1)
	for i, line := range lines[1:] {
		fields, err := parseRecord(line, len(qualificationBenchmarkSourceHeaderV1), i+2, "qualification benchmark order")
		if err != nil {
			return nil, err
		}
		ordinal, ordinalErr := canonicalPositive(fields[1])
		size, sizeErr := canonicalPositive(fields[9])
		if ordinalErr != nil || sizeErr != nil || fields[0] != "v1" || ordinal != i+1 ||
			!validID(fields[3], "op-v1-") || !safeQualificationTSVFieldV1(fields[6]) ||
			!validQualificationClass(fields[7]) || !safeEvidencePart(fields[11]) {
			return nil, fmt.Errorf("qualification benchmark order line %d: invalid key", i+2)
		}
		keys = append(keys, QualificationPrimaryKeyV1{
			OperationID: fields[3], CorpusID: fields[6], Class: fields[7], SizeUnits: size, OptionsID: fields[11],
		})
	}
	return keys, nil
}

func renderQualificationBenchmarkSourceV1(family string, rows []QualificationBenchmarkRowV1) ([]byte, error) {
	lines := make([]string, 1, len(rows)+1)
	lines[0] = strings.Join(qualificationBenchmarkSourceHeaderV1[:], "\t")
	for i, row := range rows {
		if row.Ordinal != i+1 {
			return nil, errors.New("qualification benchmark source: noncanonical row ordinal")
		}
		lines = append(lines, strings.Join([]string{
			"v1", strconv.Itoa(row.Ordinal), strconv.Itoa(row.OperationOrdinal), row.OperationID, row.PublicWrapper,
			family, row.CorpusID, row.Class, row.InputUnit, strconv.Itoa(row.SizeUnits),
			strconv.Itoa(row.SetBytesDenominator), row.OptionsID, row.BenchmarkName,
		}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func renderQualificationCorpusSnapshotV1(corpora []CorpusContractRecordV1) ([]byte, error) {
	if len(corpora) == 0 {
		return nil, errors.New("qualification corpus snapshot: empty corpus population")
	}
	lines := make([]string, 1, len(corpora)+1)
	lines[0] = strings.Join(corpusContractHeaderV1[:], "\t")
	seen := make(map[string]bool, len(corpora))
	for i, corpus := range corpora {
		fields := []string{
			"v1", strconv.Itoa(corpus.Ordinal), corpus.CorpusID, corpus.State, corpus.ElementType,
			corpus.SizeUnits, corpus.ByteLengthOrPending, corpus.SHA256OrPending, corpus.SourceIdentity,
			corpus.Recipe, corpus.FamilyContracts,
		}
		if corpus.Ordinal != i+1 || seen[corpus.CorpusID] {
			return nil, errors.New("qualification corpus snapshot: invalid identity or order")
		}
		for _, field := range fields {
			if !safeQualificationTSVFieldV1(field) {
				return nil, errors.New("qualification corpus snapshot: unsafe field")
			}
		}
		seen[corpus.CorpusID] = true
		lines = append(lines, strings.Join(fields, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

func safeQualificationTSVFieldV1(value string) bool {
	return value != "" && !strings.ContainsAny(value, "\t\r\n\x00")
}

// ParseQualificationPolicyV1 parses the exact eight-family policy. Deferred
// corpora are permitted here because the policy names future prerequisites;
// they are rejected from a provider-ready contract below.
func ParseQualificationPolicyV1(data []byte, corpora []CorpusContractRecordV1) ([]QualificationPolicyV1, error) {
	lines, err := parseLines(data, "qualification policy")
	if err != nil {
		return nil, err
	}
	if err := validateHeader(lines[0], qualificationPolicyHeaderV1[:], "qualification policy"); err != nil {
		return nil, err
	}
	if len(lines) != 9 {
		return nil, fmt.Errorf("qualification policy: got %d rows, want 8", len(lines)-1)
	}
	corpusByID := make(map[string]CorpusContractRecordV1, len(corpora))
	for _, corpus := range corpora {
		if corpus.CorpusID == "" || corpusByID[corpus.CorpusID].CorpusID != "" {
			return nil, errors.New("qualification policy: invalid corpus input")
		}
		corpusByID[corpus.CorpusID] = corpus
	}
	families := []string{
		"FC-v1-helper-validation", "FC-v1-latin1-source", "FC-v1-utf8-source", "FC-v1-utf16-source",
		"FC-v1-utf32-source", "FC-v1-detection", "FC-v1-find", "FC-v1-base64",
	}
	out := make([]QualificationPolicyV1, 0, 8)
	for i, line := range lines[1:] {
		f, err := parseRecord(line, len(qualificationPolicyHeaderV1), i+2, "qualification policy")
		if err != nil {
			return nil, err
		}
		ordinal, e1 := canonicalPositive(f[1])
		oldCount, e2 := canonicalPositive(f[5])
		newCount, e3 := canonicalPositive(f[6])
		allocs, e4 := canonicalNonNegativeV1(f[8])
		bulk, e5 := canonicalPositive(f[9])
		protected, e6 := canonicalPositive(f[10])
		pValue, e7 := canonicalPositive(f[11])
		corpusIDs := strings.Split(f[3], ";")
		evidenceKinds := strings.Split(f[16], ";")
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil || e5 != nil || e6 != nil || e7 != nil ||
			f[0] != "v1" || ordinal != i+1 || f[2] != families[i] ||
			!sameQualificationStrings(corpusIDs, qualificationFamilyCorporaV1[f[2]]) ||
			f[4] != "linux-amd64-none:westmere>haswell;linux-amd64-simd:archsimd;darwin-arm64-nosimd:neon" ||
			oldCount != 10 || newCount != 10 || f[7] != "old_then_new" || allocs != 0 ||
			bulk != 300 || protected != 200 || pValue != 50000 || f[12] != "required" ||
			f[13] != "direct_only" || f[14] != "direct_only" || f[15] != "selected" ||
			!sameQualificationStrings(evidenceKinds, qualificationEvidenceKindsV1[:]) {
			return nil, fmt.Errorf("qualification policy line %d: noncanonical policy", i+2)
		}
		for _, id := range corpusIDs {
			corpus, ok := corpusByID[id]
			if !ok || !semicolonContains(corpus.FamilyContracts, f[2]) {
				return nil, fmt.Errorf("qualification policy line %d: corpus %q is not bound to family", i+2, id)
			}
		}
		out = append(out, QualificationPolicyV1{
			Ordinal: ordinal, FamilyContractID: f[2], RequiredCorpusIDs: corpusIDs,
			PromotionSequence: f[4], OldSampleCount: oldCount, NewSampleCount: newCount,
			SampleOrder: f[7], RequiredAllocsPerOp: allocs, MinimumBulkWinBPS: bulk,
			MaximumProtectedSlowdownBPS: protected, MaximumPValueMillionths: pValue,
			NoStatisticallySignificantSlowdown: true, InconclusiveOutcome: f[13],
			FailureOutcome: f[14], SelectedOutcome: f[15], RequiredEvidenceKinds: evidenceKinds,
		})
	}
	return out, nil
}

// RenderQualificationPolicyV1 renders only the exact canonical policy.
func RenderQualificationPolicyV1(rows []QualificationPolicyV1, corpora []CorpusContractRecordV1) ([]byte, error) {
	if len(rows) != 8 {
		return nil, errors.New("qualification policy: need exactly eight family rows")
	}
	lines := []string{strings.Join(qualificationPolicyHeaderV1[:], "\t")}
	for _, row := range rows {
		noSlowdown := "optional"
		if row.NoStatisticallySignificantSlowdown {
			noSlowdown = "required"
		}
		lines = append(lines, strings.Join([]string{
			"v1", strconv.Itoa(row.Ordinal), row.FamilyContractID, strings.Join(row.RequiredCorpusIDs, ";"),
			row.PromotionSequence, strconv.Itoa(row.OldSampleCount), strconv.Itoa(row.NewSampleCount), row.SampleOrder,
			strconv.Itoa(row.RequiredAllocsPerOp), strconv.Itoa(row.MinimumBulkWinBPS),
			strconv.Itoa(row.MaximumProtectedSlowdownBPS), strconv.Itoa(row.MaximumPValueMillionths), noSlowdown,
			row.InconclusiveOutcome, row.FailureOutcome, row.SelectedOutcome, strings.Join(row.RequiredEvidenceKinds, ";"),
		}, "\t"))
	}
	data := []byte(strings.Join(lines, "\n") + "\n")
	parsed, err := ParseQualificationPolicyV1(data, corpora)
	if err != nil || len(parsed) != len(rows) {
		return nil, errors.New("qualification policy: cannot render invalid policy")
	}
	return data, nil
}

// ParseQualificationContractV1 rejects unknown fields, noncanonical JSON, and
// any contract that is not ready to start a provider campaign.
func ParseQualificationContractV1(data []byte, policy []QualificationPolicyV1, context QualificationValidationContextV1) (QualificationContractV1, error) {
	var contract QualificationContractV1
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return QualificationContractV1{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return QualificationContractV1{}, errors.New("qualification contract: trailing JSON data")
	}
	if err := validateQualificationContractV1(contract, policy, context); err != nil {
		return QualificationContractV1{}, err
	}
	canonical, err := json.Marshal(contract)
	if err != nil {
		return QualificationContractV1{}, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(data, canonical) {
		return QualificationContractV1{}, errors.New("qualification contract: noncanonical JSON")
	}
	return contract, nil
}

// RenderQualificationContractV1 validates and renders canonical JSON.
func RenderQualificationContractV1(contract QualificationContractV1, policy []QualificationPolicyV1, context QualificationValidationContextV1) ([]byte, error) {
	if err := validateQualificationContractV1(contract, policy, context); err != nil {
		return nil, err
	}
	data, err := json.Marshal(contract)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// QualificationBenchmarkRowDigestV1 computes the immutable digest of one row.
func QualificationBenchmarkRowDigestV1(row QualificationBenchmarkRowV1) string {
	fields := []string{
		"qualification-row-v1", strconv.Itoa(row.Ordinal), strconv.Itoa(row.OperationOrdinal), row.OperationID,
		row.PublicWrapper, row.BenchmarkName, strconv.Itoa(row.CorpusOrdinal), row.CorpusID,
		strconv.Itoa(row.CorpusByteLength), row.CorpusSHA256, row.CorpusSourceIdentity,
		row.CorpusMaterialization, row.ValidityClass, row.Class, row.InputUnit, strconv.Itoa(row.SizeUnits),
		strconv.Itoa(row.SetBytesDenominator), row.OptionsID, row.OptionsTupleHex, row.DestinationPolicy,
		row.ResultSink, row.SetupBoundary, row.TimedBoundary, strconv.FormatBool(row.Protected),
		strconv.FormatBool(row.IntendedBulk), row.GoDispatchStatus, row.PinnedCPPStatus,
		row.PinnedCPPReason, row.PinnedCPPProcedure, row.PinnedCPPOptions,
	}
	for _, tier := range row.Tiers {
		fields = append(fields, tier.Tier, strings.Join(tier.RuntimeIdentifiers, ";"))
	}
	return SHA256Hex(EncodeTupleV1(fields...))
}

// QualificationContractIDV1 computes the content identity excluding the ID field.
func QualificationContractIDV1(contract QualificationContractV1) string {
	fields := []string{
		"qualification-contract-v1", contract.FamilyContractID, contract.ScalarPublicationCommit,
		contract.ScalarPublicationTree, contract.ScalarPublicationParent, contract.PolicySHA256,
		contract.ClassificationSHA256, contract.BenchmarkSourceSHA256, contract.BenchmarkOrderSHA256,
		contract.CorpusContractSHA256, strings.Join(contract.Providers, ";"),
	}
	for _, row := range contract.Rows {
		fields = append(fields, row.RowDigest)
	}
	return "qualification-v1-" + SHA256Hex(EncodeTupleV1(fields...))
}

// ValidateQualificationContractApplicabilityV1 verifies exact canonical equality
// between contract providers and the applicable backend set.
func ValidateQualificationContractApplicabilityV1(contract QualificationContractV1, applicable []string) error {
	if !canonicalProviderSubset(applicable) || !sameQualificationStrings(contract.Providers, applicable) {
		return errors.New("qualification contract: providers differ from applicability")
	}
	return nil
}

func validateQualificationContractV1(contract QualificationContractV1, policy []QualificationPolicyV1, context QualificationValidationContextV1) error {
	var selected *QualificationPolicyV1
	for i := range policy {
		if policy[i].FamilyContractID == contract.FamilyContractID {
			selected = &policy[i]
			break
		}
	}
	if selected == nil || contract.Schema != qualificationSchemaV1 || contract.Version != 1 ||
		!lowerHex(contract.ScalarPublicationCommit, 40) || !lowerHex(contract.ScalarPublicationTree, 40) ||
		!lowerHex(contract.ScalarPublicationParent, 40) || !lowerHex(contract.PolicySHA256, 64) ||
		!lowerHex(contract.ClassificationSHA256, 64) || !lowerHex(contract.BenchmarkSourceSHA256, 64) ||
		!lowerHex(contract.BenchmarkOrderSHA256, 64) || !lowerHex(contract.CorpusContractSHA256, 64) ||
		contract.OldSampleCount != selected.OldSampleCount || contract.NewSampleCount != selected.NewSampleCount ||
		contract.SampleOrder != selected.SampleOrder || contract.RequiredAllocsPerOp != selected.RequiredAllocsPerOp ||
		contract.MinimumBulkWinBPS != selected.MinimumBulkWinBPS ||
		contract.MaximumProtectedSlowdownBPS != selected.MaximumProtectedSlowdownBPS ||
		contract.MaximumPValueMillionths != selected.MaximumPValueMillionths ||
		contract.NoStatisticallySignificantSlowdown != selected.NoStatisticallySignificantSlowdown ||
		contract.InconclusiveOutcome != selected.InconclusiveOutcome || contract.FailureOutcome != selected.FailureOutcome ||
		contract.SelectedOutcome != selected.SelectedOutcome ||
		!sameQualificationStrings(contract.RequiredEvidenceKinds, selected.RequiredEvidenceKinds) ||
		!canonicalProviderSubset(contract.Providers) || (len(contract.Rows) == 0 && len(contract.NoAcceleratedOperations) == 0) {
		return errors.New("qualification contract: identity or policy mismatch")
	}
	policyBytes, err := RenderQualificationPolicyV1(policy, context.Corpora)
	corpusBytes, corpusErr := renderQualificationCorpusSnapshotV1(context.Corpora)
	requiredOrder, orderErr := ParseQualificationBenchmarkOrderV1(context.BenchmarkOrderBytes)
	sourceBytes, sourceErr := renderQualificationBenchmarkSourceV1(contract.FamilyContractID, contract.Rows)
	if err != nil || corpusErr != nil || orderErr != nil || sourceErr != nil ||
		!bytes.Equal(context.BenchmarkSourceBytes, sourceBytes) ||
		!bytes.Equal(context.ClassificationBytes, RenderClassificationV1(context.ClassificationRows)) ||
		!bytes.Equal(context.CorpusContractBytes, corpusBytes) ||
		contract.PolicySHA256 != SHA256Hex(policyBytes) ||
		contract.ClassificationSHA256 != SHA256Hex(context.ClassificationBytes) ||
		contract.BenchmarkSourceSHA256 != SHA256Hex(context.BenchmarkSourceBytes) ||
		contract.BenchmarkOrderSHA256 != SHA256Hex(context.BenchmarkOrderBytes) ||
		contract.CorpusContractSHA256 != SHA256Hex(context.CorpusContractBytes) {
		return errors.New("qualification contract: frozen input digest mismatch")
	}
	derivedProviders, noAccelerated, applicabilityErr := qualificationApplicabilityV1(contract.FamilyContractID, context)
	if applicabilityErr != nil || !sameQualificationStrings(contract.Providers, derivedProviders) ||
		!sameQualificationOperationStatuses(contract.NoAcceleratedOperations, noAccelerated) {
		return errors.New("qualification contract: providers or no-accelerated operations differ from classification")
	}
	if contract.QualificationContractID != QualificationContractIDV1(contract) {
		return errors.New("qualification contract: invalid content identity")
	}
	corpusByID := make(map[string]CorpusContractRecordV1, len(context.Corpora))
	for _, corpus := range context.Corpora {
		if corpus.CorpusID == "" || corpusByID[corpus.CorpusID].CorpusID != "" {
			return errors.New("qualification contract: invalid corpus context")
		}
		corpusByID[corpus.CorpusID] = corpus
	}
	operationByID := make(map[string]ClassificationRowV1)
	for _, classified := range context.ClassificationRows {
		if classified.FamilyContractDisplayID == contract.FamilyContractID {
			operationByID[classified.SemanticOperationID] = classified
		}
	}
	required, err := qualificationRequiredPrimaryKeys(contract.FamilyContractID, selected.RequiredCorpusIDs, requiredOrder, context)
	if err != nil {
		return err
	}
	if len(contract.Rows) != len(required) {
		return errors.New("qualification contract: benchmark row population differs from frozen order")
	}
	seenPrimary := make(map[string]bool, len(contract.Rows))
	var previous QualificationBenchmarkRowV1
	for i, row := range contract.Rows {
		if row.Ordinal != i+1 || (i > 0 && !qualificationRowLess(previous, row)) {
			return fmt.Errorf("qualification contract: row %d is not canonically ordered", i+1)
		}
		corpus, ok := corpusByID[row.CorpusID]
		if !ok || !frozenQualificationCorpus(corpus) || !semicolonContains(corpus.FamilyContracts, contract.FamilyContractID) ||
			row.CorpusByteLength != mustCanonicalPositive(corpus.ByteLengthOrPending) ||
			row.CorpusSHA256 != corpus.SHA256OrPending || row.CorpusSourceIdentity != corpus.SourceIdentity ||
			row.CorpusMaterialization != corpus.Recipe {
			return fmt.Errorf("qualification contract: row %d uses deferred, pending, or drifted corpus", i+1)
		}
		classified, classifiedOK := operationByID[row.OperationID]
		if !classifiedOK || row.OperationOrdinal != classified.CanonicalRowRank+1 || row.PublicWrapper != classified.GoSymbol {
			return fmt.Errorf("qualification contract: row %d operation authority differs from classification", i+1)
		}
		if err := validateQualificationRowV1(row, contract.Providers, corpus); err != nil {
			return fmt.Errorf("qualification contract row %d: %w", i+1, err)
		}
		primary := qualificationPrimaryKeyV1(row.OperationID, row.CorpusID, row.Class, row.SizeUnits, row.OptionsID)
		want := qualificationPrimaryKeyV1(required[i].OperationID, required[i].CorpusID, required[i].Class, required[i].SizeUnits, required[i].OptionsID)
		if seenPrimary[primary] || primary != want {
			return fmt.Errorf("qualification contract: row %d differs from frozen primary-key order", i+1)
		}
		seenPrimary[primary] = true
		previous = row
	}
	return nil
}

func qualificationApplicabilityV1(family string, context QualificationValidationContextV1) ([]string, []QualificationOperationStatusV1, error) {
	operations := make(map[string]ClassificationRowV1)
	for _, row := range context.ClassificationRows {
		if row.FamilyContractDisplayID != family {
			continue
		}
		if !validID(row.SemanticOperationID, "op-v1-") || !validGoIdentifierV1(row.GoSymbol, true) || row.CanonicalRowRank < 0 {
			return nil, nil, errors.New("qualification contract: invalid classified operation")
		}
		operations[row.SemanticOperationID] = row
	}
	if len(operations) == 0 {
		return nil, nil, errors.New("qualification contract: no classified operations for family")
	}
	eligible := make(map[string]bool)
	providers := make(map[string]bool)
	for _, cell := range context.ClassificationCells {
		_, known := operations[cell.SemanticOperationID]
		if cell.FamilyContractDisplayID != family || !known || cell.BackendOutcome != "eligible" {
			continue
		}
		if !validQualificationProvider(cell.Backend) {
			return nil, nil, errors.New("qualification contract: eligible classification has invalid provider")
		}
		eligible[cell.SemanticOperationID] = true
		providers[cell.Backend] = true
	}
	outProviders := make([]string, 0, len(providers))
	for _, provider := range qualificationProviderOrderV1 {
		if providers[provider] {
			outProviders = append(outProviders, provider)
		}
	}
	if len(outProviders) == 0 {
		return nil, nil, errors.New("qualification contract: no eligible provider")
	}
	var noAccelerated []QualificationOperationStatusV1
	for _, row := range operations {
		if !eligible[row.SemanticOperationID] {
			noAccelerated = append(noAccelerated, QualificationOperationStatusV1{
				OperationOrdinal: row.CanonicalRowRank + 1, OperationID: row.SemanticOperationID,
				PublicWrapper: row.GoSymbol, Status: "not_applicable_no_accelerated_cell",
			})
		}
	}
	sort.Slice(noAccelerated, func(i, j int) bool { return noAccelerated[i].OperationOrdinal < noAccelerated[j].OperationOrdinal })
	return outProviders, noAccelerated, nil
}

func sameQualificationOperationStatuses(a, b []QualificationOperationStatusV1) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func qualificationRequiredPrimaryKeys(family string, corpusIDs []string, frozen []QualificationPrimaryKeyV1, context QualificationValidationContextV1) ([]QualificationPrimaryKeyV1, error) {
	operations := map[string]bool{}
	for _, row := range context.ClassificationRows {
		if row.FamilyContractDisplayID != family {
			continue
		}
		if !validID(row.SemanticOperationID, "op-v1-") {
			return nil, errors.New("qualification contract: invalid classified operation population")
		}
		operations[row.SemanticOperationID] = true
	}
	if len(operations) == 0 {
		return nil, errors.New("qualification contract: no classified operations for family")
	}
	corpora := make(map[string]bool, len(corpusIDs))
	for _, id := range corpusIDs {
		if id == "Q-find-cpp-capture" {
			continue
		}
		if corpora[id] {
			return nil, errors.New("qualification contract: duplicate required corpus")
		}
		corpora[id] = true
	}
	corpusByID := make(map[string]CorpusContractRecordV1, len(context.Corpora))
	for _, corpus := range context.Corpora {
		corpusByID[corpus.CorpusID] = corpus
	}
	operationCoverage := make(map[string]struct{ protected, bulk bool }, len(operations))
	corpusCoverage := make(map[string]bool, len(corpora))
	seen := make(map[string]bool, len(frozen))
	for _, key := range frozen {
		if !operations[key.OperationID] || !corpora[key.CorpusID] || !validQualificationClass(key.Class) ||
			key.SizeUnits < 1 || !safeEvidencePart(key.OptionsID) {
			return nil, errors.New("qualification contract: frozen benchmark key is outside the classified family")
		}
		corpus := corpusByID[key.CorpusID]
		if key.OptionsID != "default" || (key.Class == "short" && key.SizeUnits != 1) ||
			(key.Class == "bulk" && key.SizeUnits != mustCanonicalPositive(corpus.SizeUnits)) {
			return nil, errors.New("qualification contract: benchmark key differs from immutable matrix")
		}
		primary := qualificationPrimaryKeyV1(key.OperationID, key.CorpusID, key.Class, key.SizeUnits, key.OptionsID)
		if seen[primary] {
			return nil, errors.New("qualification contract: duplicate frozen benchmark key")
		}
		seen[primary] = true
		coverage := operationCoverage[key.OperationID]
		coverage.protected = coverage.protected || key.Class == "short" || key.Class == "boundary"
		coverage.bulk = coverage.bulk || key.Class == "bulk"
		operationCoverage[key.OperationID] = coverage
		corpusCoverage[key.CorpusID] = true
	}
	for operation := range operations {
		coverage := operationCoverage[operation]
		if !coverage.protected || !coverage.bulk {
			return nil, errors.New("qualification contract: frozen benchmark order omits protected or bulk operation coverage")
		}
	}
	for corpus := range corpora {
		if !corpusCoverage[corpus] {
			return nil, errors.New("qualification contract: frozen benchmark order omits required corpus")
		}
	}
	return frozen, nil
}

func qualificationPrimaryKeyV1(operationID, corpusID, class string, sizeUnits int, optionsID string) string {
	return strings.Join([]string{operationID, corpusID, class, strconv.Itoa(sizeUnits), optionsID}, "\x00")
}

func validateQualificationRowV1(row QualificationBenchmarkRowV1, providers []string, corpus CorpusContractRecordV1) error {
	if row.OperationOrdinal < 1 || !validID(row.OperationID, "op-v1-") || !validGoIdentifierV1(row.PublicWrapper, true) ||
		row.CorpusOrdinal != corpus.Ordinal || row.SizeUnits < 1 || row.CorpusByteLength < 1 ||
		!validQualificationClass(row.Class) || !validQualificationValidity(row.ValidityClass) ||
		!validQualificationUnit(row.InputUnit) || !safeEvidencePart(row.OptionsID) ||
		!validOptionsTupleV1(row.OptionsTupleHex) || !validQualificationDestination(row.DestinationPolicy) ||
		!validGoIdentifierV1(row.ResultSink, false) || row.SetupBoundary != "outside_timed_loop" ||
		row.TimedBoundary != "public_wrapper_only" || row.RowDigest != QualificationBenchmarkRowDigestV1(row) {
		return errors.New("invalid row identity or recipe")
	}
	wantName := fmt.Sprintf("BenchmarkDispatchQualification/%s/%s/%s/%04d", row.PublicWrapper, row.CorpusID, row.Class, row.SizeUnits)
	if row.BenchmarkName != wantName {
		return errors.New("benchmark name is not canonical")
	}
	units := mustCanonicalPositive(corpus.SizeUnits)
	if row.SizeUnits > units {
		return errors.New("row size exceeds corpus")
	}
	width := map[string]int{"byte": 1, "uint16": 2, "uint32": 4}[row.InputUnit]
	if width == 0 || row.InputUnit != corpus.ElementType || row.SetBytesDenominator != width*row.SizeUnits || row.SetBytesDenominator > row.CorpusByteLength {
		return errors.New("invalid input unit or SetBytes denominator")
	}
	if row.Class == "bulk" {
		if row.Protected || !row.IntendedBulk {
			return errors.New("bulk row has invalid protection flags")
		}
	} else if !row.Protected || row.IntendedBulk {
		return errors.New("protected row has invalid protection flags")
	}
	if row.GoDispatchStatus != "required_pending" {
		return errors.New("pre-candidate row must have required_pending Go dispatch status")
	}
	if err := validateQualificationTiers(row.Tiers, providers); err != nil {
		return err
	}
	return validatePinnedCPPV1(row)
}

func validateQualificationTiers(tiers []QualificationTierV1, providers []string) error {
	if len(tiers) != len(providers)+1 || tiers[0].Tier != "scalar" {
		return errors.New("tier allowlist does not cover scalar and providers")
	}
	seenIdentifiers := map[string]bool{}
	for i, tier := range tiers {
		if i > 0 && tier.Tier != providers[i-1] {
			return errors.New("tier allowlist order differs from providers")
		}
		if len(tier.RuntimeIdentifiers) == 0 {
			return errors.New("tier has no runtime identifier")
		}
		for _, identifier := range tier.RuntimeIdentifiers {
			if !validGoIdentifierV1(identifier, false) || seenIdentifiers[identifier] {
				return errors.New("invalid or duplicate runtime identifier")
			}
			seenIdentifiers[identifier] = true
		}
	}
	return nil
}

func validatePinnedCPPV1(row QualificationBenchmarkRowV1) error {
	reasons := map[string]bool{
		"no_upstream_procedure": true, "cpp_width_mismatch": true, "semantic_or_option_mismatch": true,
		"unseeded_capture_missing": true, "out_of_scope_isa": true,
	}
	switch row.PinnedCPPStatus {
	case "comparable_pass", "comparable_result_only":
		if row.PinnedCPPReason != "" || row.PinnedCPPProcedure == "" || row.PinnedCPPOptions == "" {
			return errors.New("comparable C++ row lacks exact procedure/options")
		}
	case "not_comparable", "not_applicable":
		if !reasons[row.PinnedCPPReason] || row.PinnedCPPProcedure != "" || row.PinnedCPPOptions != "" {
			return errors.New("C++ N/A row has invalid independent reason")
		}
	default:
		return errors.New("invalid pinned C++ status")
	}
	return nil
}

func qualificationRowLess(a, b QualificationBenchmarkRowV1) bool {
	if a.OperationOrdinal != b.OperationOrdinal {
		return a.OperationOrdinal < b.OperationOrdinal
	}
	if a.OperationID != b.OperationID {
		return a.OperationID < b.OperationID
	}
	if a.CorpusOrdinal != b.CorpusOrdinal {
		return a.CorpusOrdinal < b.CorpusOrdinal
	}
	classRank := map[string]int{"short": 0, "boundary": 1, "bulk": 2}
	if classRank[a.Class] != classRank[b.Class] {
		return classRank[a.Class] < classRank[b.Class]
	}
	if a.SizeUnits != b.SizeUnits {
		return a.SizeUnits < b.SizeUnits
	}
	return a.OptionsID < b.OptionsID
}

func validQualificationProvider(provider string) bool {
	for _, candidate := range qualificationProviderOrderV1 {
		if provider == candidate {
			return true
		}
	}
	return false
}

func canonicalProviderSubset(providers []string) bool {
	if len(providers) == 0 {
		return false
	}
	at := 0
	for _, provider := range providers {
		for at < len(qualificationProviderOrderV1) && qualificationProviderOrderV1[at] != provider {
			at++
		}
		if at == len(qualificationProviderOrderV1) {
			return false
		}
		at++
	}
	return true
}

func sameQualificationStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func frozenQualificationCorpus(corpus CorpusContractRecordV1) bool {
	return strings.HasPrefix(corpus.State, "frozen_") && corpus.ByteLengthOrPending != "pending" &&
		corpus.SizeUnits != "pending" && lowerDigest(corpus.SHA256OrPending)
}

func semicolonContains(value, want string) bool {
	for _, field := range strings.Split(value, ";") {
		if field == want {
			return true
		}
	}
	return false
}

func canonicalNonNegativeV1(value string) (int, error) {
	if value == "0" {
		return 0, nil
	}
	return canonicalPositive(value)
}

func mustCanonicalPositive(value string) int {
	result, err := canonicalPositive(value)
	if err != nil {
		return -1
	}
	return result
}

func validQualificationClass(value string) bool {
	return value == "short" || value == "boundary" || value == "bulk"
}

func validQualificationValidity(value string) bool {
	return value == "valid" || value == "invalid" || value == "mixed" || value == "option_dependent"
}

func validQualificationUnit(value string) bool {
	return value == "byte" || value == "uint16" || value == "uint32"
}

func validQualificationDestination(value string) bool {
	return value == "none" || value == "preallocated_exact" || value == "preallocated_safe" || value == "reset_each_iteration"
}

func validOptionsTupleV1(value string) bool {
	if value == "" || len(value)%2 != 0 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return false
	}
	for count := 1; count <= len(decoded)/2+1; count++ {
		fields, err := DecodeTupleV1(decoded, count)
		if err == nil && bytes.Equal(decoded, EncodeTupleV1(fields...)) {
			return true
		}
	}
	return false
}

func validGoIdentifierV1(value string, exported bool) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if exported {
				if r < 'A' || r > 'Z' {
					return false
				}
			} else if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
			continue
		}
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}
