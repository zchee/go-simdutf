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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ingestFixtureV1(t *testing.T, contents []byte) (string, EvidenceRecordV1, EvidenceValidationContextV1) {
	t.Helper()
	record, context := validEvidenceRecordV1(t, contents)
	record.Kind = "exit"
	record.OutputID = "exit"
	record.MediaType = "application/json"
	for _, output := range context.ExpectedCommands[len(context.ExpectedCommands)-1].Outputs {
		if output.ID == record.OutputID {
			record.OriginPath = output.Path
			break
		}
	}
	record.Path = "raw/" + record.AuthoritySHA256[:12] + "/" + record.FamilyID + "/" + record.CampaignID + "/" +
		record.LaneID + "/" + record.ProducerID + "/" + record.Kind + "/" + record.StorageID + "/" +
		record.Digest + ".json"
	record.ReceiptID = ReceiptIDV1(record)
	root := t.TempDir()
	filename := filepath.Join(append([]string{root}, strings.Split(record.Path, "/")...)...)
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	return root, record, context
}

func TestIngestEvidenceFileV1SafeNestedFile(t *testing.T) {
	contents := []byte("{\"exit_code\":0}\n")
	root, record, context := ingestFixtureV1(t, contents)
	if err := IngestEvidenceFileV1(root, record, context, &EvidenceRegistryV1{}); err != nil {
		t.Fatal(err)
	}
}

func TestIngestEvidenceFileV1RejectsLimitAndMismatch(t *testing.T) {
	emptyRecord, emptyContext := validEvidenceRecordV1(t, nil)
	if _, err := validateEvidenceEnvelopeV1(emptyRecord, 0, emptyRecord.Digest, emptyContext); err != nil {
		t.Fatalf("zero-sized artifact metadata: %v", err)
	}

	contents := []byte("{\"exit_code\":0}\n")
	root, record, context := ingestFixtureV1(t, contents)
	if err := IngestEvidenceFileV1(root, record, context, &EvidenceRegistryV1{}); err != nil {
		t.Fatalf("valid bounded file: %v", err)
	}
	tooLarge := record
	tooLarge.Size = MaxRawEvidenceSizeV1 + 1
	if err := IngestEvidenceFileV1(root, tooLarge, context, &EvidenceRegistryV1{}); err == nil {
		t.Fatal("accepted one-over maximum metadata")
	}
	mismatch := record
	mismatch.Size++
	if err := IngestEvidenceFileV1(root, mismatch, context, &EvidenceRegistryV1{}); err == nil {
		t.Fatal("accepted size mismatch")
	}
}

func TestIngestEvidenceFileV1RejectsSymlinks(t *testing.T) {
	contents := []byte("{\"exit_code\":0}\n")
	root, record, context := ingestFixtureV1(t, contents)
	link := root + "-link"
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := IngestEvidenceFileV1(link, record, context, &EvidenceRegistryV1{}); err == nil {
		t.Fatal("accepted symlink root")
	}
	filename := filepath.Join(append([]string{root}, strings.Split(record.Path, "/")...)...)
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/null", filename); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := IngestEvidenceFileV1(root, record, context, &EvidenceRegistryV1{}); err == nil {
		t.Fatal("accepted symlink file")
	}
}

func TestIngestEvidenceFileV1RejectsNilRegistry(t *testing.T) {
	contents := []byte("{\"exit_code\":0}\n")
	root, record, context := ingestFixtureV1(t, contents)
	if err := IngestEvidenceFileV1(root, record, context, nil); err == nil {
		t.Fatal("accepted full evidence ingestion without a semantic registry")
	}
}

func TestRenderEvidenceRegistryIndexV1BindsTypedLogicalKeys(t *testing.T) {
	first := EvidenceRecordV1{
		Path: "raw/a", MediaType: "text/plain", Size: 1, Digest: strings.Repeat("1", 64),
		KeyKind: "row", StorageID: "rk-v1-" + strings.Repeat("2", 64),
		TupleHex: "00", CommandID: "command-a", OutputID: "stdout", ProducerID: "producer",
		CampaignID: "campaign-v1-" + strings.Repeat("3", 64), RowID: "rk-v1-" + strings.Repeat("2", 64),
	}
	first.ReceiptID = ReceiptIDV1(first)
	second := first
	second.Path = "raw/b"
	second.CommandID = "command-b"
	second.ReceiptID = ReceiptIDV1(second)
	registry := &EvidenceRegistryV1{receipts: map[string]EvidenceRecordV1{
		second.ReceiptID: second,
		first.ReceiptID:  first,
	}}
	index, err := RenderEvidenceRegistryIndexV1(registry)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(index), "\n"), "\n")
	if len(lines) != 3 || lines[0] != strings.TrimSuffix(returnIndexHeaderV1, "\n") {
		t.Fatalf("unexpected global index population: %q", index)
	}
	if !strings.HasPrefix(lines[1], first.Path+"\t") || !strings.HasPrefix(lines[2], second.Path+"\t") {
		t.Fatalf("global index is not path-canonical: %q", index)
	}
	fields := strings.Split(lines[1], "\t")
	if len(fields) != 16 || fields[5] != first.KeyKind || fields[6] != first.StorageID ||
		fields[8] != first.TupleHex || fields[9] != first.CommandID {
		t.Fatalf("global index omitted typed logical-key fields: %#v", fields)
	}
	digest, err := EvidenceRegistryDigestV1(registry)
	if err != nil {
		t.Fatal(err)
	}
	if digest != SHA256Hex(index) {
		t.Fatal("registry digest is not the canonical global index digest")
	}
}
