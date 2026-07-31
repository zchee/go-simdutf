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
	"encoding/json"
	"maps"
	"reflect"
	"strings"
	"testing"
)

func TestReceiptIDV1BindsEveryLogicalSubjectPart(t *testing.T) {
	record := EvidenceRecordV1{
		AuthoritySHA256: strings.Repeat("a", 64), CampaignID: "campaign-v1-" + strings.Repeat("b", 64),
		LaneID: "linux-amd64-simd", Provider: "provider", ProducerID: "producer", Kind: "stdout",
		CommandID: "command", OutputID: "output", KeyKind: "row", StorageID: "rk-v1-" + strings.Repeat("c", 64), Digest: strings.Repeat("d", 64),
	}
	want := ReceiptIDV1(record)
	if !strings.HasPrefix(want, "receipt-v1-") || len(want) != len("receipt-v1-")+64 {
		t.Fatalf("invalid receipt format %q", want)
	}
	for _, mutate := range []func(*EvidenceRecordV1){
		func(r *EvidenceRecordV1) { r.AuthoritySHA256 = strings.Repeat("e", 64) },
		func(r *EvidenceRecordV1) { r.CampaignID += "x" },
		func(r *EvidenceRecordV1) { r.LaneID = "linux-amd64-none" },
		func(r *EvidenceRecordV1) { r.Provider = "other" },
		func(r *EvidenceRecordV1) { r.ProducerID = "other" },
		func(r *EvidenceRecordV1) { r.Kind = "stderr" },
		func(r *EvidenceRecordV1) { r.CommandID = "other" },
		func(r *EvidenceRecordV1) { r.OutputID = "other" },
		func(r *EvidenceRecordV1) { r.KeyKind = "cell" },
		func(r *EvidenceRecordV1) { r.StorageID = "cell-v1-" + strings.Repeat("c", 64) },
		func(r *EvidenceRecordV1) { r.Digest = strings.Repeat("e", 64) },
	} {
		changed := record
		mutate(&changed)
		if ReceiptIDV1(changed) == want {
			t.Fatal("receipt omitted a logical subject component")
		}
	}
}

func TestEvidenceRegistryV1RejectsDuplicateLogicalReceipt(t *testing.T) {
	r := &EvidenceRegistryV1{}
	record := EvidenceRecordV1{AuthoritySHA256: strings.Repeat("a", 64), FamilyID: "family", CampaignID: "campaign", LaneID: "lane", Provider: "provider", ProducerID: "producer", Kind: "stdout", CommandID: "command", OutputID: "output", KeyKind: "row", StorageID: "row", ReceiptID: "receipt-one", StateSubject: "none"}
	if err := r.addValidated(record); err != nil {
		t.Fatal(err)
	}
	record.ReceiptID = "receipt-two"
	if err := r.addValidated(record); err == nil {
		t.Fatal("receipt value bypassed logical uniqueness")
	}
}

func TestEvidenceRegistryV1RejectedInsertionsAreAtomic(t *testing.T) {
	contents := []byte("evidence")
	record, context := validEvidenceRecordV1(t, contents)
	record.StateSubject = "none"
	registry := &EvidenceRegistryV1{
		contents: map[string][]byte{"sentinel": bytes.Clone(contents)},
		states:   map[string]string{"sentinel": "snapshot_planned"},
	}
	contract, err := evidenceQualificationContractV1(context)
	if err != nil {
		t.Fatal(err)
	}
	registry.commitQualification(context.QualificationContractDigest, contract)
	if err := registry.addValidated(record); err != nil {
		t.Fatal(err)
	}

	type registryState struct {
		keys           map[string]struct{}
		receipts       map[string]EvidenceRecordV1
		contents       map[string][]byte
		qualifications map[string]QualificationContractV1
		states         map[string]string
	}
	snapshot := func() registryState {
		state := registryState{
			keys:           make(map[string]struct{}, len(registry.keys)),
			receipts:       make(map[string]EvidenceRecordV1, len(registry.receipts)),
			contents:       make(map[string][]byte, len(registry.contents)),
			qualifications: make(map[string]QualificationContractV1, len(registry.qualifications)),
			states:         make(map[string]string, len(registry.states)),
		}
		for key := range registry.keys {
			state.keys[key] = struct{}{}
		}
		for id, value := range registry.receipts {
			state.receipts[id] = cloneEvidenceRecordV1(value)
		}
		for id, value := range registry.contents {
			state.contents[id] = bytes.Clone(value)
		}
		maps.Copy(state.qualifications, registry.qualifications)
		maps.Copy(state.states, registry.states)
		return state
	}
	assertUnchanged := func(t *testing.T, before registryState) {
		t.Helper()
		after := snapshot()
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("registry changed after rejected insertion\nbefore: %#v\nafter:  %#v", before, after)
		}
	}

	t.Run("duplicate logical key", func(t *testing.T) {
		before := snapshot()
		duplicate := record
		duplicate.ReceiptID = "receipt-v1-" + strings.Repeat("f", 64)
		if err := registry.addValidated(duplicate); err == nil {
			t.Fatal("accepted duplicate logical key")
		}
		assertUnchanged(t, before)
	})

	t.Run("unequal qualification contract", func(t *testing.T) {
		unequal := contract
		unequal.MinimumBulkWinBPS++
		registry.qualifications[context.QualificationContractDigest] = unequal
		before := snapshot()
		if _, _, err := prepareEvidenceQualificationV1(context, registry); err == nil {
			t.Fatal("accepted unequal contract under an existing qualification digest")
		}
		assertUnchanged(t, before)
	})
}
func TestEvidenceRecordV1RejectsOriginPathAndContextTopologyMutations(t *testing.T) {
	contents := []byte("evidence")
	record, context := validEvidenceRecordV1(t, contents)

	record.OriginPath = "staging/other.txt"
	if _, err := validateEvidenceEnvelopeV1(record, int64(len(contents)), record.Digest, context); err == nil {
		t.Fatal("accepted origin path not bound to frozen output")
	}

	record, context = validEvidenceRecordV1(t, contents)
	context.ExpectedBindings = nil
	if _, err := validateEvidenceEnvelopeV1(record, int64(len(contents)), record.Digest, context); err == nil {
		t.Fatal("accepted empty frozen binding topology")
	}

	record, context = validEvidenceRecordV1(t, contents)
	context.ExpectedCommands = append(context.ExpectedCommands, context.ExpectedCommands[0])
	if _, err := validateEvidenceEnvelopeV1(record, int64(len(contents)), record.Digest, context); err == nil {
		t.Fatal("accepted duplicate frozen command topology")
	}
}

func TestEvidenceRecordV1RejectsSemanticTupleSubstitution(t *testing.T) {
	contents := []byte("evidence")
	record, context := validEvidenceRecordV1(t, contents)
	record.RowID = "rk-v1-" + strings.Repeat("a", 64)
	context.ExpectedBindings[0].RowID = record.RowID
	context.ExpectedCommands[0].RowID = record.RowID
	record.ReceiptID = ReceiptIDV1(record)
	if _, err := validateEvidenceEnvelopeV1(record, int64(len(contents)), record.Digest, context); err == nil {
		t.Fatal("accepted row tuple with substituted semantic row")
	}
}

func TestEvidenceRecordV1RejectsNilRegistry(t *testing.T) {
	contents := []byte("evidence")
	record, context := validEvidenceRecordV1(t, contents)
	if err := ValidateEvidenceRecordV1(record, contents, context, nil); err == nil {
		t.Fatal("accepted full evidence validation without a semantic registry")
	}
}
func TestEvidenceStateV1RequiresEveryHardGateProof(t *testing.T) {
	record := EvidenceRecordV1{
		Kind: "state-transition", StateSubject: "backend_cell",
		PrerequisiteState: "direct_built", CurrentState: "hard_gates_green",
		StorageID: "cell", CellID: "cell", KeyKind: "cell",
	}
	registry := &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{}}
	for _, kind := range []string{"test", "race", "object", "disassembly"} {
		id := "receipt-" + kind
		registry.receipts[id] = EvidenceRecordV1{Kind: kind, StateSubject: "none", StorageID: record.StorageID, CellID: record.CellID, KeyKind: record.KeyKind}
		record.ProofReceiptIDs = append(record.ProofReceiptIDs, id)
	}
	if err := validateState(record, EvidenceBindingV1{}, registry); err == nil {
		t.Fatal("accepted hard gates without fuzz proof")
	}
	registry.receipts["receipt-fuzz"] = EvidenceRecordV1{Kind: "fuzz", StateSubject: "none", StorageID: record.StorageID, CellID: record.CellID, KeyKind: record.KeyKind}
	record.ProofReceiptIDs = append(record.ProofReceiptIDs, "receipt-fuzz")
	if err := validateState(record, EvidenceBindingV1{}, registry); err != nil {
		t.Fatalf("rejected complete hard-gate proof set: %v", err)
	}
}
func TestEvidenceSemanticArtifactsV1FailClosed(t *testing.T) {
	state := EvidenceRecordV1{StateSubject: "backend_cell", PrerequisiteState: "dispatch_candidate", CurrentState: "direct_only", Disposition: "direct_only", GoQualification: "fail"}
	if canonicalStateArtifactV1(state, []byte("arbitrary")) {
		t.Fatal("arbitrary transition content was accepted")
	}
	exit := EvidenceRecordV1{Kind: "exit"}
	if err := validateArtifactSemanticsV1(exit, []byte("{\"exit_code\":1}\n"), EvidenceValidationContextV1{}, nil); err == nil {
		t.Fatal("nonzero exit artifact was accepted")
	}
	proof := EvidenceRecordV1{Kind: "test"}
	if err := validateArtifactSemanticsV1(proof, []byte("ok\n"), EvidenceValidationContextV1{}, nil); err == nil {
		t.Fatal("proof was accepted without a registered campaign identity set")
	}
	source := EvidenceRecordV1{CommandAction: "source_commit", SourceCommit: strings.Repeat("a", 40), IdentityValue: strings.Repeat("a", 40)}
	if identityArtifactV1(source, []byte(strings.Repeat("b", 40)+"\n"), EvidenceValidationContextV1{}) {
		t.Fatal("fabricated source identity was accepted")
	}
}

func TestQualificationSamplesV1RejectsWrongPopulation(t *testing.T) {
	raw := []byte("BenchmarkA-8 1 1 ns/op 1 B/op 0 allocs/op\n")
	rows := []QualificationBenchmarkRowV1{{BenchmarkName: "BenchmarkA"}}
	samples, err := parseBenchmarkSamplesV1(raw, rows)
	if err != nil {
		t.Fatal(err)
	}
	if _, complete := benchmarkMedianX2V1(samples["BenchmarkA"]); complete {
		t.Fatal("single benchmark sample was accepted")
	}
}

func TestEvaluateQualificationOutcomeV1Thresholds(t *testing.T) {
	contract := QualificationContractV1{
		MinimumBulkWinBPS:                  300,
		MaximumProtectedSlowdownBPS:        200,
		MaximumPValueMillionths:            50000,
		NoStatisticallySignificantSlowdown: true,
	}
	tests := []struct {
		name        string
		row         QualificationBenchmarkRowV1
		old         benchmarkSamplesV1
		new         benchmarkSamplesV1
		pValue      int
		want        string
		wantErr     bool
		staleMedian bool
	}{
		{
			name: "bulk pass", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(96_000_000, true),
			pValue: 50000, want: "pass",
		},
		{
			name: "bulk minimum miss", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(98_000_000, true),
			pValue: 50000, want: "fail",
		},
		{
			name: "bulk significance miss", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(96_000_000, true),
			pValue: 50001, want: "fail",
		},
		{
			name: "protected slowdown limit", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkProtected", Protected: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(103_000_000, true),
			pValue: 900000, want: "fail",
		},
		{
			name: "significant slowdown", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkProtected", Protected: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(101_000_000, true),
			pValue: 50000, want: "fail",
		},
		{
			name: "nonsignificant protected noise", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkProtected", Protected: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(101_000_000, true),
			pValue: 50001, want: "pass",
		},
		{
			name: "allocation regression", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(96_000_000, false),
			pValue: 50000, want: "fail",
		},
		{
			name: "missing population", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: benchmarkSamplesV1{NanosMillionths: []int64{96_000_000}, ZeroAllocs: true},
			pValue: -1, want: "inconclusive",
		},
		{
			name: "benchstat inconclusive", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(96_000_000, true),
			pValue: -1, want: "inconclusive",
		},
		{
			name: "stale comparison", row: QualificationBenchmarkRowV1{BenchmarkName: "BenchmarkBulk", IntendedBulk: true},
			old: completeBenchmarkSamplesV1(100_000_000, true), new: completeBenchmarkSamplesV1(96_000_000, true),
			pValue: 50000, wantErr: true, staleMedian: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldMedian, _ := benchmarkMedianX2V1(test.old)
			newMedian, _ := benchmarkMedianX2V1(test.new)
			if test.staleMedian {
				newMedian++
			}
			comparison := benchstatRowV1{
				BenchmarkName: test.row.BenchmarkName, OldMedianNanosMillionthsX2: oldMedian,
				NewMedianNanosMillionthsX2: newMedian, PValueMillionths: test.pValue,
			}
			got, err := evaluateQualificationOutcomeV1(
				contract,
				[]QualificationBenchmarkRowV1{test.row},
				map[string]benchmarkSamplesV1{test.row.BenchmarkName: test.old},
				map[string]benchmarkSamplesV1{test.row.BenchmarkName: test.new},
				[]benchstatRowV1{comparison},
			)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Fatalf("outcome = %q, want %q", got, test.want)
			}
		})
	}
}

func completeBenchmarkSamplesV1(value int64, zeroAllocs bool) benchmarkSamplesV1 {
	values := make([]int64, 10)
	for i := range values {
		values[i] = value
	}
	return benchmarkSamplesV1{NanosMillionths: values, ZeroAllocs: zeroAllocs}
}

func TestEvidenceRegistryDigestV1BindsCanonicalReceiptPopulation(t *testing.T) {
	first := EvidenceRecordV1{ReceiptID: "receipt-v1-" + strings.Repeat("0", 64)}
	first.ReceiptID = ReceiptIDV1(first)
	second := EvidenceRecordV1{Kind: "stderr"}
	second.ReceiptID = ReceiptIDV1(second)

	left := &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{
		first.ReceiptID: first, second.ReceiptID: second,
	}}
	right := &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{
		second.ReceiptID: second, first.ReceiptID: first,
	}}
	leftDigest, err := EvidenceRegistryDigestV1(left)
	if err != nil {
		t.Fatal(err)
	}
	rightDigest, err := EvidenceRegistryDigestV1(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftDigest != rightDigest {
		t.Fatal("registry digest depends on map iteration order")
	}
	delete(right.receipts, second.ReceiptID)
	smallerDigest, err := EvidenceRegistryDigestV1(right)
	if err != nil {
		t.Fatal(err)
	}
	if smallerDigest == leftDigest {
		t.Fatal("registry digest omitted a receipt")
	}
	right.receipts["wrong"] = first
	if _, err := EvidenceRegistryDigestV1(right); err == nil {
		t.Fatal("registry digest accepted a receipt under the wrong identity")
	}
}

func TestEvidenceRegistryV1DeepCopiesReceiptSlices(t *testing.T) {
	record := EvidenceRecordV1{
		AuthoritySHA256: strings.Repeat("a", 64), FamilyID: "family", CampaignID: "campaign",
		LaneID: "lane", Provider: "provider", ProducerID: "producer", Kind: "stdout",
		CommandID: "command", OutputID: "output", KeyKind: "row", StorageID: "row",
		ReceiptID: "receipt", StateSubject: "none",
		OrderedMembers: []string{"member"}, ProofReceiptIDs: []string{"proof"},
	}
	registry := &EvidenceRegistryV1{}
	if err := registry.addValidated(record); err != nil {
		t.Fatal(err)
	}
	record.OrderedMembers[0], record.ProofReceiptIDs[0] = "mutated", "mutated"
	stored, ok := registry.Receipt("receipt")
	if !ok || stored.OrderedMembers[0] != "member" || stored.ProofReceiptIDs[0] != "proof" {
		t.Fatal("registry insertion retained caller-owned slices")
	}
	stored.OrderedMembers[0], stored.ProofReceiptIDs[0] = "mutated", "mutated"
	snapshot := registry.Snapshot()
	snapshot[0].OrderedMembers[0], snapshot[0].ProofReceiptIDs[0] = "mutated", "mutated"
	stored, _ = registry.Receipt("receipt")
	if stored.OrderedMembers[0] != "member" || stored.ProofReceiptIDs[0] != "proof" {
		t.Fatal("registry accessor exposed mutable internal slices")
	}
}

func TestValidRawEvidenceRecordPathV1UsesAuthorityPrefix(t *testing.T) {
	record := EvidenceRecordV1{
		AuthoritySHA256: strings.Repeat("a", 64), FamilyID: "family-v1-" + strings.Repeat("b", 64),
		CampaignID: "campaign-v1-" + strings.Repeat("c", 64), LaneID: "linux-amd64-none",
		ProducerID: "producer", Kind: "test", StorageID: "rk-v1-" + strings.Repeat("d", 64),
		Digest: strings.Repeat("e", 64),
	}
	suffix := "/" + record.FamilyID + "/" + record.CampaignID + "/" + record.LaneID + "/" +
		record.ProducerID + "/" + record.Kind + "/" + record.StorageID + "/" + record.Digest + ".txt"
	record.Path = "raw/" + record.AuthoritySHA256[:12] + suffix
	if !validRawEvidenceRecordPathV1(record) {
		t.Fatal("canonical authority prefix path rejected")
	}
	record.Path = "raw/" + record.AuthoritySHA256 + suffix
	if validRawEvidenceRecordPathV1(record) {
		t.Fatal("full authority digest path accepted in authority-prefix slot")
	}
}

func TestBenchmarkMedianAndThresholdArithmeticV1(t *testing.T) {
	const maxInt64 = int64(^uint64(0) >> 1)
	overflow := completeBenchmarkSamplesV1(maxInt64, true)
	if _, ok := benchmarkMedianX2V1(overflow); ok {
		t.Fatal("overflowing doubled median accepted")
	}
	if comparison, err := compareBenchmarkChangeV1(10000, 9700, 300); err != nil || comparison != 0 {
		t.Fatalf("exact bulk boundary = %d, %v", comparison, err)
	}
	if comparison, err := compareBenchmarkChangeV1(10000, 9701, 300); err != nil || comparison >= 0 {
		t.Fatalf("one-under bulk boundary = %d, %v", comparison, err)
	}
}

func TestMannWhitneyPValueMillionthsV1IsDerived(t *testing.T) {
	oldSamples := completeBenchmarkSamplesV1(100, true).NanosMillionths
	newSamples := completeBenchmarkSamplesV1(90, true).NanosMillionths
	got, err := mannWhitneyPValueMillionthsV1(oldSamples, newSamples)
	if err != nil {
		t.Fatal(err)
	}
	if got != 11 {
		t.Fatalf("complete-separation p-value = %d, want 11 millionths", got)
	}
	got, err = mannWhitneyPValueMillionthsV1(oldSamples, oldSamples)
	if err != nil || got != 1000000 {
		t.Fatalf("equal-sample p-value = %d, %v", got, err)
	}
}

func TestCanonicalQuietAffinityArtifactV1(t *testing.T) {
	record := EvidenceRecordV1{CommandID: "quiet-command", CommandAction: "quiet_affinity_recheck"}
	context := EvidenceValidationContextV1{ExpectedCommands: []CampaignCommandV1{{
		ID: "quiet-command", Action: "quiet_affinity_recheck",
		Argv: []string{"/home/zchee/sdk/go1.26.5/bin/go", "run", "./internal/portplan/cmd/simdutf-evidence", "quiet-affinity-recheck", "--cpu=1", "--policy=taskset:1"},
		Env:  map[string]string{"SIMDUTF_CPU": "1", "SIMDUTF_AFFINITY": "taskset:1"},
	}}}
	canonical, err := json.Marshal(struct {
		Schema  string `json:"schema"`
		Version int    `json:"version"`
		CPU     string `json:"cpu"`
		Policy  string `json:"policy"`
		Status  string `json:"status"`
	}{"simdutf-quiet-affinity-v1", 1, "1", "taskset:1", "quiet"})
	if err != nil {
		t.Fatal(err)
	}
	if !canonicalQuietAffinityArtifactV1(record, append(canonical, '\n'), context) {
		t.Fatal("exact quiet-affinity artifact rejected")
	}
	if canonicalQuietAffinityArtifactV1(record, []byte(`{"schema":"simdutf-quiet-affinity-v1","version":1,"cpu":"2","policy":"taskset:2","status":"quiet"}`+"\n"), context) {
		t.Fatal("mismatched quiet-affinity artifact accepted")
	}
}

func TestFileDigestArtifactV1BindsCommandedPath(t *testing.T) {
	digest := strings.Repeat("a", 64)
	record := EvidenceRecordV1{
		CommandID: "digest-command", CommandAction: "file_digest", CommandRole: "host",
		IdentityValue: digest,
	}
	context := EvidenceValidationContextV1{ExpectedCommands: []CampaignCommandV1{{
		ID: "digest-command", Action: "file_digest", Role: "host",
		Argv: []string{"/usr/bin/sha256sum", "staging/identity/go-binary"},
	}}}
	if !fileDigestArtifactV1(record, []byte(digest+"  staging/identity/go-binary\n"), context) {
		t.Fatal("exact digest output rejected")
	}
	if fileDigestArtifactV1(record, []byte(digest+"  staging/source/old.tar\n"), context) {
		t.Fatal("digest output for an uncommanded path accepted")
	}
}

func TestNotApplicableEvidenceSourceV1IsAllowlisted(t *testing.T) {
	prefix := "611becc2a08c27a4edc77d9a45ff74c97130129b:include/simdutf/implementation.h:202"
	if !validNAEvidenceSourceV1("native_wrapper_delegates_explicit_endian", prefix) {
		t.Fatal("canonical native-wrapper source rejected")
	}
	if !validNAEvidenceSourceV1("composite_wrapper_delegates_accelerated_core", prefix+";dependency:DetectEncodings") {
		t.Fatal("canonical composite-wrapper source rejected")
	}
	if validNAEvidenceSourceV1("primitive_gap", strings.Replace(prefix, "611becc", "711becc", 1)) ||
		validNAEvidenceSourceV1("composite_wrapper_delegates_accelerated_core", prefix+";arbitrary") {
		t.Fatal("untrusted not-applicable source accepted")
	}
}

func TestDirectOnlyProofV1RequiresSelector(t *testing.T) {
	kinds := map[string]bool{
		"incumbent-benchmark": true, "candidate-benchmark": true, "benchstat": true,
		"provider-guard": true,
	}
	if requireDirectOnlyProofV1(kinds) {
		t.Fatal("direct-only proof accepted without selector evidence")
	}
	kinds["selector"] = true
	if !requireDirectOnlyProofV1(kinds) {
		t.Fatal("complete direct-only proof rejected")
	}
}
