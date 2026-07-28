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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func qualificationInputsV1(t *testing.T) ([]CorpusContractRecordV1, []QualificationPolicyV1, []byte) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller is unavailable")
	}
	inputRoot := filepath.Join(filepath.Dir(file), "..", "..", "docs", "porting", "simdutf-port-v1", "inputs")
	corpusBytes, err := os.ReadFile(filepath.Join(inputRoot, "corpus-contract-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	corpora, err := ParseCorpusContractV1(corpusBytes)
	if err != nil {
		t.Fatal(err)
	}
	policyBytes, err := os.ReadFile(filepath.Join(inputRoot, "qualification-policy-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := ParseQualificationPolicyV1(policyBytes, corpora)
	if err != nil {
		t.Fatal(err)
	}
	return corpora, policy, policyBytes
}

func qualificationContractFixtureV1(t *testing.T, corpora []CorpusContractRecordV1, policyBytes []byte) (QualificationContractV1, QualificationValidationContextV1) {
	t.Helper()
	corpora = append([]CorpusContractRecordV1(nil), corpora...)
	for i := range corpora {
		if semicolonContains(corpora[i].FamilyContracts, "FC-v1-helper-validation") && !frozenQualificationCorpus(corpora[i]) {
			corpora[i].State = "frozen_deterministic"
			corpora[i].SizeUnits = "2048"
			corpora[i].ByteLengthOrPending = "4096"
			corpora[i].SHA256OrPending = strings.Repeat("a", 64)
			corpora[i].SourceIdentity = "generated"
			corpora[i].Recipe = "4096 zero bytes"
		}
	}
	operationID := "op-v1-" + strings.Repeat("a", 64)
	context := QualificationValidationContextV1{
		ClassificationRows: []ClassificationRowV1{{
			FamilyContractDisplayID: "FC-v1-helper-validation",
			SemanticOperationID:     operationID,
			GoSymbol:                "ValidateUTF16LE",
			CanonicalRowRank:        0,
		}},
		ClassificationCells: []FinalCellV1{{
			FamilyContractDisplayID: "FC-v1-helper-validation",
			SemanticOperationID:     operationID,
			Backend:                 "westmere",
			BackendOutcome:          "eligible",
		}},
		Corpora:             corpora,
		ClassificationBytes: []byte("frozen classification\n"),
	}
	context.ClassificationBytes = RenderClassificationV1(context.ClassificationRows)
	var err error
	context.CorpusContractBytes, err = renderQualificationCorpusSnapshotV1(context.Corpora)
	if err != nil {
		t.Fatal(err)
	}
	makeRow := func(ordinal int, corpus CorpusContractRecordV1, class string) QualificationBenchmarkRowV1 {
		row := QualificationBenchmarkRowV1{
			Ordinal: ordinal, OperationOrdinal: 1, OperationID: operationID,
			PublicWrapper: "ValidateUTF16LE", CorpusOrdinal: corpus.Ordinal, CorpusID: corpus.CorpusID,
			CorpusByteLength: mustCanonicalPositive(corpus.ByteLengthOrPending), CorpusSHA256: corpus.SHA256OrPending,
			CorpusSourceIdentity: corpus.SourceIdentity, CorpusMaterialization: corpus.Recipe,
			ValidityClass: "valid", Class: class, InputUnit: corpus.ElementType, SizeUnits: func() int {
				if class == "bulk" {
					return mustCanonicalPositive(corpus.SizeUnits)
				}
				return 1
			}(),
			SetBytesDenominator: 0, OptionsID: "default",
			OptionsTupleHex: hex.EncodeToString(EncodeTupleV1("default")), DestinationPolicy: "none",
			ResultSink: "benchmarkBoolSink", SetupBoundary: "outside_timed_loop",
			TimedBoundary: "public_wrapper_only", Protected: class != "bulk", IntendedBulk: class == "bulk",
			Tiers: []QualificationTierV1{
				{Tier: "scalar", RuntimeIdentifiers: []string{"validateUTF16LEScalar"}},
				{Tier: "westmere", RuntimeIdentifiers: []string{"validateUTF16LEWestmere"}},
			},
			GoDispatchStatus: "required_pending", PinnedCPPStatus: "not_applicable",
			PinnedCPPReason: "no_upstream_procedure",
		}
		row.SetBytesDenominator = map[string]int{"byte": 1, "uint16": 2, "uint32": 4}[corpus.ElementType] * row.SizeUnits
		row.BenchmarkName = "BenchmarkDispatchQualification/ValidateUTF16LE/" + corpus.CorpusID + "/" + class + "/" + fourDigitsV1(row.SizeUnits)
		row.RowDigest = QualificationBenchmarkRowDigestV1(row)
		return row
	}
	contract := QualificationContractV1{
		Schema: qualificationSchemaV1, Version: 1, FamilyContractID: "FC-v1-helper-validation",
		ScalarPublicationCommit: strings.Repeat("1", 40), ScalarPublicationTree: strings.Repeat("2", 40),
		ScalarPublicationParent: strings.Repeat("3", 40), PolicySHA256: SHA256Hex(policyBytes),
		ClassificationSHA256:  SHA256Hex(context.ClassificationBytes),
		BenchmarkSourceSHA256: SHA256Hex(context.BenchmarkSourceBytes),
		BenchmarkOrderSHA256:  "",
		CorpusContractSHA256:  SHA256Hex(context.CorpusContractBytes),
		Providers:             []string{"westmere"}, OldSampleCount: 10, NewSampleCount: 10,
		SampleOrder: "old_then_new", RequiredAllocsPerOp: 0, MinimumBulkWinBPS: 300,
		MaximumProtectedSlowdownBPS: 200, MaximumPValueMillionths: 50000,
		NoStatisticallySignificantSlowdown: true, InconclusiveOutcome: "direct_only",
		FailureOutcome: "direct_only", SelectedOutcome: "selected",
		RequiredEvidenceKinds: append([]string(nil), qualificationEvidenceKindsV1[:]...),
	}
	for _, corpus := range corpora {
		if semicolonContains(corpus.FamilyContracts, contract.FamilyContractID) {
			for _, class := range []string{"short", "bulk"} {
				contract.Rows = append(contract.Rows, makeRow(len(contract.Rows)+1, corpus, class))
			}
		}
	}
	order := make([]QualificationPrimaryKeyV1, len(contract.Rows))
	for i, row := range contract.Rows {
		order[i] = QualificationPrimaryKeyV1{
			OperationID: row.OperationID, CorpusID: row.CorpusID, Class: row.Class,
			SizeUnits: row.SizeUnits, OptionsID: row.OptionsID,
		}
	}
	context.BenchmarkSourceBytes, err = renderQualificationBenchmarkSourceV1(contract.FamilyContractID, contract.Rows)
	if err != nil {
		t.Fatal(err)
	}
	contract.BenchmarkSourceSHA256 = SHA256Hex(context.BenchmarkSourceBytes)
	context.BenchmarkOrderBytes, err = RenderQualificationBenchmarkOrderV1(order)
	if err != nil {
		t.Fatal(err)
	}
	contract.BenchmarkOrderSHA256 = SHA256Hex(context.BenchmarkOrderBytes)
	contract.QualificationContractID = QualificationContractIDV1(contract)
	return contract, context
}

func fourDigitsV1(value int) string {
	result := "0000" + strconvItoaV1(value)
	if len(result) > 4 {
		return result[len(result)-maxIntV1(4, len(strconvItoaV1(value))):]
	}
	return result[len(result)-4:]
}

func strconvItoaV1(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	at := len(digits)
	for value > 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}

func maxIntV1(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func TestQualificationPolicyV1ExactAndDeterministic(t *testing.T) {
	corpora, policy, policyBytes := qualificationInputsV1(t)
	if len(policy) != 8 {
		t.Fatalf("policy rows = %d, want 8", len(policy))
	}
	rendered, err := RenderQualificationPolicyV1(policy, corpora)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rendered, policyBytes) {
		t.Fatal("qualification policy does not round-trip byte-identically")
	}
	for _, mutation := range []string{
		strings.Replace(string(policyBytes), "\t300\t200\t", "\t301\t200\t", 1),
		strings.Replace(string(policyBytes), "old_then_new", "new_then_old", 1),
		strings.Replace(string(policyBytes), ";final-selector", "", 1),
		strings.Replace(string(policyBytes), "Q-byte-zero;Q-u16-zero", "Q-u16-zero;Q-byte-zero", 1),
	} {
		if _, err := ParseQualificationPolicyV1([]byte(mutation), corpora); err == nil {
			t.Fatal("accepted qualification policy mutation")
		}
	}
}

func TestQualificationContractV1CanonicalAndFailClosed(t *testing.T) {
	corpora, policy, policyBytes := qualificationInputsV1(t)
	contract, context := qualificationContractFixtureV1(t, corpora, policyBytes)
	encoded, err := RenderQualificationContractV1(contract, policy, context)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseQualificationContractV1(encoded, policy, context)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.QualificationContractID != contract.QualificationContractID {
		t.Fatal("qualification contract identity changed")
	}
	if err := ValidateQualificationContractApplicabilityV1(contract, []string{"westmere"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateQualificationContractApplicabilityV1(contract, []string{"westmere", "haswell"}); err == nil {
		t.Fatal("accepted incomplete provider applicability")
	}

	mutations := []QualificationContractV1{}
	wrongThreshold := contract
	wrongThreshold.MinimumBulkWinBPS = 299
	mutations = append(mutations, wrongThreshold)
	missingCorpus := contract
	missingCorpus.Rows = missingCorpus.Rows[:len(missingCorpus.Rows)-1]
	missingCorpus.QualificationContractID = QualificationContractIDV1(missingCorpus)
	mutations = append(mutations, missingCorpus)
	wrongDigest := contract
	wrongDigest.Rows = append([]QualificationBenchmarkRowV1(nil), contract.Rows...)
	wrongDigest.Rows[0].SizeUnits = 2
	mutations = append(mutations, wrongDigest)
	duplicatePrimary := contract
	duplicatePrimary.Rows = append([]QualificationBenchmarkRowV1(nil), contract.Rows...)
	duplicatePrimary.Rows[1] = duplicatePrimary.Rows[0]
	duplicatePrimary.Rows[1].Ordinal = 2
	duplicatePrimary.Rows[1].RowDigest = QualificationBenchmarkRowDigestV1(duplicatePrimary.Rows[1])
	duplicatePrimary.QualificationContractID = QualificationContractIDV1(duplicatePrimary)
	mutations = append(mutations, duplicatePrimary)
	staleDispatch := contract
	staleDispatch.Rows = append([]QualificationBenchmarkRowV1(nil), contract.Rows...)
	staleDispatch.Rows[0].GoDispatchStatus = "pass"
	staleDispatch.Rows[0].RowDigest = QualificationBenchmarkRowDigestV1(staleDispatch.Rows[0])
	staleDispatch.QualificationContractID = QualificationContractIDV1(staleDispatch)
	mutations = append(mutations, staleDispatch)
	for _, mutation := range mutations {
		if _, err := RenderQualificationContractV1(mutation, policy, context); err == nil {
			t.Fatal("accepted invalid qualification contract")
		}
	}
	missingOperation := context
	missingOperation.ClassificationRows = append([]ClassificationRowV1(nil), context.ClassificationRows...)
	missingOperation.ClassificationRows = append(missingOperation.ClassificationRows, ClassificationRowV1{
		FamilyContractDisplayID: contract.FamilyContractID,
		SemanticOperationID:     "op-v1-" + strings.Repeat("b", 64),
	})
	missingOperation.BenchmarkOrderBytes = append([]byte(nil), context.BenchmarkOrderBytes...)
	missingOperation.ClassificationBytes = RenderClassificationV1(missingOperation.ClassificationRows)
	order, err := ParseQualificationBenchmarkOrderV1(missingOperation.BenchmarkOrderBytes)
	if err != nil {
		t.Fatal(err)
	}
	for _, corpusID := range qualificationFamilyCorporaV1[contract.FamilyContractID] {
		for _, class := range []string{"short", "bulk"} {
			order = append(order, QualificationPrimaryKeyV1{
				OperationID: "op-v1-" + strings.Repeat("b", 64), CorpusID: corpusID,
				Class: class, SizeUnits: 1, OptionsID: "default",
			})
		}
	}
	missingOperation.BenchmarkOrderBytes, err = RenderQualificationBenchmarkOrderV1(order)
	if err != nil {
		t.Fatal(err)
	}
	missingOperationContract := contract
	missingOperationContract.ClassificationSHA256 = SHA256Hex(missingOperation.ClassificationBytes)
	missingOperationContract.QualificationContractID = QualificationContractIDV1(missingOperationContract)
	if _, err := RenderQualificationContractV1(missingOperationContract, policy, missingOperation); err == nil {
		t.Fatal("accepted contract missing classified operation")
	}
	for _, mutate := range []func(*QualificationValidationContextV1){
		func(c *QualificationValidationContextV1) { c.ClassificationBytes = append(c.ClassificationBytes, 'x') },
		func(c *QualificationValidationContextV1) {
			c.BenchmarkSourceBytes = append(c.BenchmarkSourceBytes, 'x')
		},
		func(c *QualificationValidationContextV1) { c.BenchmarkOrderBytes = append(c.BenchmarkOrderBytes, 'x') },
		func(c *QualificationValidationContextV1) { c.CorpusContractBytes = append(c.CorpusContractBytes, 'x') },
	} {
		mutated := context
		mutate(&mutated)
		if _, err := RenderQualificationContractV1(contract, policy, mutated); err == nil {
			t.Fatal("accepted changed frozen validation context")
		}
	}

	withUnknown := bytes.Replace(encoded, []byte("{\"schema\":"), []byte("{\"unknown\":1,\"schema\":"), 1)
	if _, err := ParseQualificationContractV1(withUnknown, policy, context); err == nil {
		t.Fatal("accepted unknown qualification field")
	}
	if _, err := ParseQualificationContractV1(append(encoded, '\n'), policy, context); err == nil {
		t.Fatal("accepted noncanonical qualification JSON")
	}
}

func TestQualificationContractV1RejectsDeferredCorpus(t *testing.T) {
	corpora, policy, policyBytes := qualificationInputsV1(t)
	contract, context := qualificationContractFixtureV1(t, corpora, policyBytes)
	for i := range corpora {
		if corpora[i].CorpusID == "Q-byte-zero" {
			corpora[i].State = "deferred_until_scalar_publication"
			corpora[i].SizeUnits = "pending"
			corpora[i].ByteLengthOrPending = "pending"
			corpora[i].SHA256OrPending = "pending"
		}
	}
	context.Corpora = corpora
	if _, err := RenderQualificationContractV1(contract, policy, context); err == nil {
		t.Fatal("accepted a provider contract with deferred corpus")
	}
}

func TestQualificationSchemaV1Parity(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "porting", "simdutf-port-v1", "inputs", "qualification-contract-schema-v1.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, literal := range []string{
		qualificationSchemaV1, "\"minimum_bulk_win_bps\": {\"const\": 300}",
		"\"maximum_protected_slowdown_bps\": {\"const\": 200}",
		"\"maximum_p_value_millionths\": {\"const\": 50000}",
		"incumbent-benchmark", "candidate-benchmark", "final-selector", "row_digest", "\"go_dispatch_status\": {\"const\": \"required_pending\"}",
	} {
		if !strings.Contains(text, literal) {
			t.Fatalf("qualification schema lacks %q", literal)
		}
	}
}
