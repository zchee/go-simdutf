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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func validEvidenceRecordV1(t *testing.T, contents []byte) (EvidenceRecordV1, EvidenceValidationContextV1) {
	t.Helper()

	rowFields := [6]string{"Unicode", "fixture", "fixture()", "include/simdutf/implementation.h:1", "Fixture", "func Fixture()"}
	rowID := RowKeyV1(rowFields)
	operationID, err := ScalarOperationIDV1(rowID)
	if err != nil {
		t.Fatal(err)
	}
	family, err := FamilyKeyV1("FC-v1-helper-validation")
	if err != nil {
		t.Fatal(err)
	}
	campaign, err := CampaignKeyV1("fixture-campaign", []byte("fixture manifest"))
	if err != nil {
		t.Fatal(err)
	}
	batch, err := BatchKeyV1("evidence", "fixture-batch", []string{rowID})
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := TransactionKeyV1("fixture-transaction", []string{rowID})
	if err != nil {
		t.Fatal(err)
	}
	cell, err := CellKeyV1(rowID, "westmere")
	if err != nil {
		t.Fatal(err)
	}

	authority := []byte("canonical fixture authority")
	authoritySum := sha256.Sum256(authority)
	authorityDigest := hex.EncodeToString(authoritySum[:])
	provider := "westmere"
	producer := "producer-fixture"
	oldCommit, oldTree, oldParent := strings.Repeat("1", 40), strings.Repeat("2", 40), strings.Repeat("3", 40)
	newCommit, newTree, newParent := strings.Repeat("4", 40), strings.Repeat("5", 40), strings.Repeat("6", 40)
	hostDigest := SHA256Hex(EncodeTupleV1(
		"evidence-host-identity-v1",
		SHA256Hex([]byte("Linux fixture\n")), SHA256Hex([]byte("{\"lscpu\":[]}\n")),
		SHA256Hex([]byte("go version go1.26.5 linux/amd64\n")), strings.Repeat("d", 64),
	))

	contract := fixtureQualificationContractV1(operationID)
	contractBytes, err := json.Marshal(contract)
	if err != nil {
		t.Fatal(err)
	}
	contractBytes = append(contractBytes, '\n')

	context := EvidenceValidationContextV1{
		AuthorityBytes: authority, AuthoritySHA256: authorityDigest, FrozenInputSHA256: authorityDigest,
		FamilyID: family.StorageID, CampaignID: campaign.StorageID, LaneID: "linux-amd64-none",
		ProducerID: producer, Provider: provider, SourceRole: "old",
		SourceCommit: oldCommit, SourceTree: oldTree, SourceParent: oldParent,
		OldSourceCommit: oldCommit, OldSourceTree: oldTree, OldSourceParent: oldParent,
		NewSourceCommit: newCommit, NewSourceTree: newTree, NewSourceParent: newParent,
		QualificationContractID:     contract.QualificationContractID,
		QualificationContractDigest: SHA256Hex(contractBytes), QualificationContractBytes: contractBytes,
		CorpusID: "corpus-v1", CorpusDigest: strings.Repeat("7", 64),
		HostReceiptID: hostReceiptIDV1(hostDigest), HostReceiptDigest: hostDigest, HostIdentityDigest: hostDigest,
	}
	context.IdentitySetDigest = identitySetDigestV1(context)

	goBin := "/home/zchee/sdk/go1.26.5/bin/go"
	sourceIdentityArgs := func(role string) []string {
		return []string{
			goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "source-identity",
			"--role=" + role, "--receipt=staging/identity/" + role + ".json",
			"--archive=staging/source/" + role + ".tar",
		}
	}
	actions := []struct {
		action string
		role   string
		argv   []string
	}{
		{"source_commit", "old", sourceIdentityArgs("old")},
		{"source_tree", "old", sourceIdentityArgs("old")},
		{"source_parent", "old", sourceIdentityArgs("old")},
		{"source_status", "old", sourceIdentityArgs("old")},
		{"source_commit", "new", sourceIdentityArgs("new")},
		{"source_tree", "new", sourceIdentityArgs("new")},
		{"source_parent", "new", sourceIdentityArgs("new")},
		{"source_status", "new", sourceIdentityArgs("new")},
		{"host_uname", "host", []string{"/usr/bin/uname", "-srm"}},
		{"host_cpu", "host", []string{"/usr/bin/lscpu", "--json"}},
		{"go_version", "host", []string{goBin, "version"}},
		{"file_digest", "host", []string{"/usr/bin/sha256sum", "staging/identity/go-binary"}},
		{"go_test_focused", "old", []string{goBin, "test", "-run=^TestFixture$", "-count=1", "."}},
	}
	for i, action := range actions {
		command := evidenceFixtureCommandV1(i+1, action.action, action.role, action.argv, context, operationID, batch.StorageID, rowID, cell.StorageID)
		context.ExpectedCommands = append(context.ExpectedCommands, command)
		for _, output := range command.Outputs {
			context.ExpectedBindings = append(context.ExpectedBindings, EvidenceBindingV1{
				KeyKind: "row", StorageID: rowID, TupleHex: hex.EncodeToString(EncodeTupleV1(rowFields[:]...)),
				RowID: rowID, CellID: cell.StorageID, BatchID: batch.StorageID, TransactionID: transaction.StorageID,
				OperationID: operationID, OrderedMembers: []string{}, CommandID: command.ID, OutputID: output.ID,
				Kind: output.Kind, Provider: provider, InitialState: "snapshot_planned",
			})
		}
	}
	commandBytes, err := RenderCanonicalCampaignCommandsV1(context.ExpectedCommands)
	if err != nil {
		t.Fatal(err)
	}
	context.CommandManifestSHA256 = SHA256Hex(commandBytes)

	command := context.ExpectedCommands[len(context.ExpectedCommands)-1]
	contentDigest := SHA256Hex(contents)
	record := EvidenceRecordV1{
		Schema: evidenceSchemaV1, Version: 1, KeyKind: "row", StorageID: rowID,
		TupleHex: hex.EncodeToString(EncodeTupleV1(rowFields[:]...)),
		FamilyID: context.FamilyID, CampaignID: context.CampaignID, LaneID: context.LaneID,
		ProducerID: producer, Provider: provider, Kind: "stdout", MediaType: "text/plain",
		Size: int64(len(contents)), Digest: contentDigest,
		SourceCommit: oldCommit, SourceTree: oldTree, SourceParent: oldParent, SourceRole: "old",
		OriginCampaignID: context.CampaignID, OriginProducerID: producer,
		OriginSourceCommit: oldCommit, OriginSourceTree: oldTree, OriginSourceParent: oldParent,
		RowID: rowID, CellID: cell.StorageID, BatchID: batch.StorageID, TransactionID: transaction.StorageID, OperationID: operationID,
		OrderedMembers: []string{}, StateSubject: "none", InitialState: "snapshot_planned", ProofReceiptIDs: []string{},
		IdentitySetDigest: context.IdentitySetDigest, CommandDigest: context.CommandManifestSHA256,
		QualificationContractID: context.QualificationContractID, QualificationContractDigest: context.QualificationContractDigest,
		CorpusID: context.CorpusID, CorpusDigest: context.CorpusDigest,
		HostReceiptID: context.HostReceiptID, HostReceiptDigest: context.HostReceiptDigest,
		AuthoritySHA256: authorityDigest,
		CommandID:       command.ID, CommandOrdinal: command.Ordinal, CommandAction: command.Action, CommandRole: command.Role, OutputID: "stdout",
	}
	record.Path = "raw/" + authorityDigest[:12] + "/" + record.FamilyID + "/" + record.CampaignID + "/" + record.LaneID + "/" + record.ProducerID + "/" + record.Kind + "/" + record.StorageID + "/" + record.Digest + ".txt"
	record.OriginPath = command.Outputs[0].Path
	record.ReceiptID = ReceiptIDV1(record)
	return record, context
}

func fixtureQualificationContractV1(operationID string) QualificationContractV1 {
	rows := make([]QualificationBenchmarkRowV1, 2)
	for i, class := range []string{"short", "bulk"} {
		rows[i] = QualificationBenchmarkRowV1{
			Ordinal: i + 1, OperationOrdinal: 1, OperationID: operationID,
			PublicWrapper: "Fixture", BenchmarkName: "BenchmarkDispatchQualification/Fixture/corpus-v1/" + class + "/0001",
			Class: class, SizeUnits: 1, OptionsID: "default", Protected: class == "short", IntendedBulk: class == "bulk",
			GoDispatchStatus: "required_pending",
		}
		rows[i].RowDigest = QualificationBenchmarkRowDigestV1(rows[i])
	}
	contract := QualificationContractV1{
		Schema: qualificationSchemaV1, Version: 1, FamilyContractID: "FC-v1-helper-validation",
		ScalarPublicationCommit: strings.Repeat("1", 40), ScalarPublicationTree: strings.Repeat("2", 40), ScalarPublicationParent: strings.Repeat("3", 40),
		PolicySHA256: strings.Repeat("8", 64), ClassificationSHA256: strings.Repeat("9", 64),
		BenchmarkSourceSHA256: strings.Repeat("a", 64), BenchmarkOrderSHA256: strings.Repeat("b", 64), CorpusContractSHA256: strings.Repeat("c", 64),
		Providers: append([]string(nil), qualificationProviderOrderV1[:]...), OldSampleCount: 10, NewSampleCount: 10, SampleOrder: "old_then_new",
		RequiredAllocsPerOp: 0, MinimumBulkWinBPS: 300, MaximumProtectedSlowdownBPS: 200,
		MaximumPValueMillionths: 50000, NoStatisticallySignificantSlowdown: true,
		InconclusiveOutcome: "direct_only", FailureOutcome: "direct_only", SelectedOutcome: "selected",
		RequiredEvidenceKinds: append([]string(nil), qualificationEvidenceKindsV1[:]...), Rows: rows,
	}
	contract.QualificationContractID = QualificationContractIDV1(contract)
	return contract
}

func evidenceFixtureCommandV1(ordinal int, action, role string, argv []string, context EvidenceValidationContextV1, operationID, batchID, rowID, cellID string) CampaignCommandV1 {
	id := "command-" + action + "-" + role
	cwd := commandCWDV1(action, role)
	cacheRole := strings.TrimPrefix(cwd, "source/")
	cacheRoot := "/home/zchee/.cache/gjc/simdutf-port/v1/" + context.CampaignID + "/cache/" + cacheRole + "/"
	env := map[string]string{
		"LC_ALL": "C", "GOMAXPROCS": "1", "GOEXPERIMENT": "none",
		"GOCACHE": cacheRoot + "go-build", "GOMODCACHE": cacheRoot + "gomod", "CGO_ENABLED": "0",
		"GOOS": "linux", "GOARCH": "amd64", "GOAMD64": "v1",
	}
	artifact := campaignArtifactKindV1(action, role)
	outputs := []CommandOutputV1{
		{ID: "stdout", Kind: "stdout", Path: "staging/" + id + "/stdout.txt", MediaType: "text/plain", Required: true},
		{ID: "stderr", Kind: "stderr", Path: "staging/" + id + "/stderr.txt", MediaType: "text/plain", Required: true},
		{ID: "exit", Kind: "exit", Path: "staging/" + id + "/exit.json", MediaType: "application/json", Required: true},
		{ID: "argv-env", Kind: "argv-env", Path: "staging/" + id + "/argv-env.json", MediaType: "application/json", Required: true},
		{ID: artifact, Kind: artifact, Path: "staging/" + id + "/" + artifact + ".txt", MediaType: "text/plain", Required: true},
	}
	return CampaignCommandV1{
		Ordinal: ordinal, ID: id, Action: action, Role: role, Argv: argv, CWD: cwd, Env: env,
		TimeoutSeconds: 60, ExpectedExit: 0, Outputs: outputs,
		OperationID: operationID, BatchID: batchID, RowID: rowID, CellID: cellID, Provider: context.Provider,
	}
}
