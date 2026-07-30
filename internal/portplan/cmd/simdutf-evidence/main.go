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

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/zchee/go-simdutf/internal/portplan"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "simdutf-evidence:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("missing subcommand")
	}
	switch args[0] {
	case "source-identity":
		return runSourceIdentity(args[1:])
	case "quiet-affinity-recheck":
		return runQuietAffinity(args[1:])
	case "state-transition":
		return runStateTransition(args[1:], false)
	case "not-applicable":
		return runStateTransition(args[1:], true)
	case "benchstat":
		return runBenchstat(args[1:])
	case "return-index":
		return runReturnIndex(args[1:])
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

type sourceIdentitySeedV1 struct {
	Schema  string `json:"schema"`
	Version int    `json:"version"`
	Commit  string `json:"commit"`
	Tree    string `json:"tree"`
	Parent  string `json:"parent"`
	Status  string `json:"status"`
}

type sourceIdentityReceiptV1 struct {
	Schema        string `json:"schema"`
	Version       int    `json:"version"`
	Role          string `json:"role"`
	Commit        string `json:"commit"`
	Tree          string `json:"tree"`
	Parent        string `json:"parent"`
	Status        string `json:"status"`
	ArchivePath   string `json:"archive_path"`
	ArchiveDigest string `json:"archive_digest"`
}

func runSourceIdentity(args []string) error {
	flags, err := parseExactFlags(args, []string{"--action=", "--role=", "--receipt=", "--archive="})
	if err != nil {
		return err
	}
	action := flags["--action="]
	role := flags["--role="]
	receiptPath := flags["--receipt="]
	archivePath := flags["--archive="]
	switch action {
	case "source_commit", "source_tree", "source_parent", "source_status":
	default:
		return fmt.Errorf("unsupported source-identity action %q", action)
	}
	if role != "old" && role != "new" {
		return fmt.Errorf("unsupported source-identity role %q", role)
	}
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		return err
	}
	seedBytes, err := os.ReadFile(archivePath + ".seed.json")
	if err != nil {
		return fmt.Errorf("source-identity seed: %w", err)
	}
	var seed sourceIdentitySeedV1
	dec := json.NewDecoder(bytes.NewReader(seedBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&seed); err != nil {
		return fmt.Errorf("source-identity seed: %w", err)
	}
	if seed.Schema != "simdutf-source-identity-seed-v1" || seed.Version != 1 || seed.Status != "clean" {
		return errors.New("source-identity seed is invalid")
	}
	if !gitSHA(seed.Commit) || !gitSHA(seed.Tree) || !gitSHA(seed.Parent) {
		return errors.New("source-identity seed contains invalid git object ids")
	}
	sum := sha256.Sum256(archive)
	receipt := sourceIdentityReceiptV1{
		Schema:        "simdutf-source-identity-receipt-v1",
		Version:       1,
		Role:          role,
		Commit:        seed.Commit,
		Tree:          seed.Tree,
		Parent:        seed.Parent,
		Status:        "clean",
		ArchivePath:   archivePath,
		ArchiveDigest: hex.EncodeToString(sum[:]),
	}
	if err := writeCanonicalJSON(receiptPath, receipt); err != nil {
		return err
	}
	switch action {
	case "source_commit":
		_, err = io.WriteString(os.Stdout, seed.Commit+"\n")
	case "source_tree":
		_, err = io.WriteString(os.Stdout, seed.Tree+"\n")
	case "source_parent":
		_, err = io.WriteString(os.Stdout, seed.Parent+"\n")
	case "source_status":
		err = nil
	}
	return err
}

type quietAffinityArtifactV1 struct {
	Schema  string `json:"schema"`
	Version int    `json:"version"`
	CPU     string `json:"cpu"`
	Policy  string `json:"policy"`
	Status  string `json:"status"`
}

func runQuietAffinity(args []string) error {
	flags, err := parseExactFlags(args, []string{"--cpu=", "--policy="})
	if err != nil {
		return err
	}
	cpu := flags["--cpu="]
	policy := flags["--policy="]
	if cpu == "" || !literalCPUSet(cpu) {
		return errors.New("invalid --cpu")
	}
	if policy != "taskset:"+cpu {
		return errors.New("invalid --policy")
	}
	if runtime.GOOS != "linux" {
		return errors.New("quiet affinity recheck is Linux-only")
	}
	allowed, err := readLinuxCpusAllowedList()
	if err != nil {
		return err
	}
	if allowed != cpu {
		return fmt.Errorf("process affinity %q is not quiet for cpu %q", allowed, cpu)
	}
	return writeCanonicalJSONWriter(os.Stdout, quietAffinityArtifactV1{
		Schema: "simdutf-quiet-affinity-v1", Version: 1, CPU: cpu, Policy: policy, Status: "quiet",
	})
}

type stateTransitionArtifactV1 struct {
	Schema            string `json:"schema"`
	Version           int    `json:"version"`
	StateSubject      string `json:"state_subject"`
	PrerequisiteState string `json:"prerequisite_state"`
	CurrentState      string `json:"current_state"`
	Disposition       string `json:"disposition"`
	GoQualification   string `json:"go_qualification"`
	NAReason          string `json:"na_reason"`
	NASource          string `json:"na_source"`
}

func runStateTransition(args []string, notApplicable bool) error {
	required := []string{"--state-subject=", "--prerequisite-state=", "--current-state=", "--disposition=", "--go-qualification="}
	if notApplicable {
		required = append(required, "--na-reason=", "--na-source=")
	}
	flags, _, err := parseExactFlagsWithProofs(args, required)
	if err != nil {
		return err
	}
	artifact := stateTransitionArtifactV1{
		Schema: "simdutf-state-transition-v1", Version: 1,
		StateSubject: flags["--state-subject="], PrerequisiteState: flags["--prerequisite-state="],
		CurrentState: flags["--current-state="], Disposition: flags["--disposition="],
		GoQualification: flags["--go-qualification="], NAReason: flags["--na-reason="], NASource: flags["--na-source="],
	}
	if notApplicable {
		if artifact.StateSubject != "backend_cell" || artifact.CurrentState != "not_applicable" || artifact.Disposition != "not_applicable" {
			return errors.New("invalid not-applicable state flags")
		}
		if artifact.NAReason == "" || artifact.NASource == "" {
			return errors.New("not-applicable requires --na-reason and --na-source")
		}
	} else if artifact.NAReason != "" || artifact.NASource != "" {
		return errors.New("state-transition forbids NA flags")
	}
	return writeCanonicalJSONWriter(os.Stdout, artifact)
}

type returnIndexProofV1 struct {
	Schema                string `json:"schema"`
	Version               int    `json:"version"`
	AuthoritySHA256       string `json:"authority_sha256"`
	CampaignID            string `json:"campaign_id"`
	CommandManifestSHA256 string `json:"command_manifest_sha256"`
	IdentitySetDigest     string `json:"identity_set_digest"`
}

type returnIndexContextFileV1 struct {
	Schema                      string                       `json:"schema"`
	Version                     int                          `json:"version"`
	AuthoritySHA256             string                       `json:"authority_sha256"`
	FrozenInputSHA256           string                       `json:"frozen_input_sha256"`
	FamilyID                    string                       `json:"family_id"`
	CampaignID                  string                       `json:"campaign_id"`
	LaneID                      string                       `json:"lane_id"`
	ProducerID                  string                       `json:"producer_id"`
	Provider                    string                       `json:"provider"`
	SourceRole                  string                       `json:"source_role"`
	SourceCommit                string                       `json:"source_commit"`
	SourceTree                  string                       `json:"source_tree"`
	SourceParent                string                       `json:"source_parent"`
	CommandManifestSHA256       string                       `json:"command_manifest_sha256"`
	QualificationContractID     string                       `json:"qualification_contract_id"`
	QualificationContractDigest string                       `json:"qualification_contract_digest"`
	CorpusID                    string                       `json:"corpus_id"`
	CorpusDigest                string                       `json:"corpus_digest"`
	HostReceiptID               string                       `json:"host_receipt_id"`
	HostReceiptDigest           string                       `json:"host_receipt_digest"`
	OldSourceCommit             string                       `json:"old_source_commit"`
	OldSourceTree               string                       `json:"old_source_tree"`
	OldSourceParent             string                       `json:"old_source_parent"`
	NewSourceCommit             string                       `json:"new_source_commit"`
	NewSourceTree               string                       `json:"new_source_tree"`
	NewSourceParent             string                       `json:"new_source_parent"`
	HostIdentityDigest          string                       `json:"host_identity_digest"`
	IdentitySetDigest           string                       `json:"identity_set_digest"`
	ExpectedCommands            []portplan.CampaignCommandV1 `json:"expected_commands"`
	ExpectedBindings            []returnIndexBindingFileV1   `json:"expected_bindings"`
	AuthorityPath               string                       `json:"authority_path"`
	QualificationContractPath   string                       `json:"qualification_contract_path"`
}

type returnIndexBindingFileV1 struct {
	KeyKind        string   `json:"key_kind"`
	StorageID      string   `json:"storage_id"`
	DisplayID      string   `json:"display_id"`
	TupleHex       string   `json:"tuple_hex"`
	RowID          string   `json:"row_id"`
	CellID         string   `json:"cell_id"`
	SymbolID       string   `json:"symbol_id"`
	BatchID        string   `json:"batch_id"`
	TransactionID  string   `json:"transaction_id"`
	OperationID    string   `json:"operation_id"`
	Backend        string   `json:"backend"`
	DirectSymbol   string   `json:"direct_symbol"`
	OrderedMembers []string `json:"ordered_members"`
	CommandID      string   `json:"command_id"`
	OutputID       string   `json:"output_id"`
	Kind           string   `json:"kind"`
	Provider       string   `json:"provider"`
	InitialState   string   `json:"initial_state"`
	NAReason       string   `json:"na_reason"`
	NASource       string   `json:"na_source"`
}

func runReturnIndex(args []string) error {
	flags, err := parseExactFlags(args, []string{"--descriptor-dir="})
	if err != nil {
		return err
	}
	dir := flags["--descriptor-dir="]
	contextBytes, err := os.ReadFile(filepath.Join(dir, "context.json"))
	if err != nil {
		return err
	}
	var file returnIndexContextFileV1
	dec := json.NewDecoder(bytes.NewReader(contextBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return fmt.Errorf("return-index context: %w", err)
	}
	if file.Schema != "simdutf-return-index-context-v1" || file.Version != 1 {
		return errors.New("return-index context schema is invalid")
	}
	authority, err := os.ReadFile(filepath.Join(dir, file.AuthorityPath))
	if err != nil {
		return err
	}
	contractBytes, err := os.ReadFile(filepath.Join(dir, file.QualificationContractPath))
	if err != nil {
		return err
	}
	context := portplan.EvidenceValidationContextV1{
		AuthorityBytes: authority, AuthoritySHA256: file.AuthoritySHA256, FrozenInputSHA256: file.FrozenInputSHA256,
		FamilyID: file.FamilyID, CampaignID: file.CampaignID, LaneID: file.LaneID, ProducerID: file.ProducerID, Provider: file.Provider,
		SourceRole: file.SourceRole, SourceCommit: file.SourceCommit, SourceTree: file.SourceTree, SourceParent: file.SourceParent,
		CommandManifestSHA256: file.CommandManifestSHA256, QualificationContractID: file.QualificationContractID,
		QualificationContractDigest: file.QualificationContractDigest, QualificationContractBytes: contractBytes,
		CorpusID: file.CorpusID, CorpusDigest: file.CorpusDigest, HostReceiptID: file.HostReceiptID, HostReceiptDigest: file.HostReceiptDigest,
		OldSourceCommit: file.OldSourceCommit, OldSourceTree: file.OldSourceTree, OldSourceParent: file.OldSourceParent,
		NewSourceCommit: file.NewSourceCommit, NewSourceTree: file.NewSourceTree, NewSourceParent: file.NewSourceParent,
		HostIdentityDigest: file.HostIdentityDigest, IdentitySetDigest: file.IdentitySetDigest, ExpectedCommands: file.ExpectedCommands,
	}
	for _, binding := range file.ExpectedBindings {
		context.ExpectedBindings = append(context.ExpectedBindings, portplan.EvidenceBindingV1{
			KeyKind: binding.KeyKind, StorageID: binding.StorageID, DisplayID: binding.DisplayID, TupleHex: binding.TupleHex,
			RowID: binding.RowID, CellID: binding.CellID, SymbolID: binding.SymbolID, BatchID: binding.BatchID,
			TransactionID: binding.TransactionID, OperationID: binding.OperationID, Backend: binding.Backend,
			DirectSymbol: binding.DirectSymbol, OrderedMembers: binding.OrderedMembers, CommandID: binding.CommandID,
			OutputID: binding.OutputID, Kind: binding.Kind, Provider: binding.Provider, InitialState: binding.InitialState,
			NAReason: binding.NAReason, NASource: binding.NASource,
		})
	}
	registry := &portplan.EvidenceRegistryV1{}
	registryDir := filepath.Join(dir, "registry")
	entries, err := os.ReadDir(registryDir)
	if err != nil {
		return err
	}
	type pending struct {
		record   portplan.EvidenceRecordV1
		contents []byte
		done     bool
	}
	var records []pending
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		recordBytes, err := os.ReadFile(filepath.Join(registryDir, entry.Name()))
		if err != nil {
			return err
		}
		var record portplan.EvidenceRecordV1
		dec := json.NewDecoder(bytes.NewReader(recordBytes))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&record); err != nil {
			return fmt.Errorf("registry receipt %s: %w", entry.Name(), err)
		}
		contents, err := os.ReadFile(filepath.Join(registryDir, strings.TrimSuffix(entry.Name(), ".json")+".content"))
		if err != nil {
			return fmt.Errorf("registry content %s: %w", entry.Name(), err)
		}
		records = append(records, pending{record: record, contents: contents})
	}
	if len(records) == 0 {
		return errors.New("return-index registry is empty")
	}
	for pass := 0; pass < len(records); pass++ {
		progress := false
		for i := range records {
			if records[i].done {
				continue
			}
			if err := portplan.ValidateEvidenceRecordV1(records[i].record, records[i].contents, context, registry); err != nil {
				continue
			}
			records[i].done = true
			progress = true
		}
		if !progress {
			break
		}
	}
	for _, item := range records {
		if !item.done {
			if err := portplan.ValidateEvidenceRecordV1(item.record, item.contents, context, registry); err != nil {
				return fmt.Errorf("registry receipt %s: %w", item.record.ReceiptID, err)
			}
		}
	}
	if _, err := portplan.RenderReturnIndexV1(context, registry); err != nil {
		return fmt.Errorf("return-index completeness gate: %w", err)
	}
	return writeCanonicalJSONWriter(os.Stdout, returnIndexProofV1{
		Schema: "simdutf-return-index-proof-v1", Version: 1,
		AuthoritySHA256: context.AuthoritySHA256, CampaignID: context.CampaignID,
		CommandManifestSHA256: context.CommandManifestSHA256, IdentitySetDigest: context.IdentitySetDigest,
	})
}

func runBenchstat(args []string) error {
	flags, err := parseExactFlags(args, []string{"--incumbent=", "--candidate=", "--incumbent-receipt-id=", "--candidate-receipt-id=", "--qualification-contract=", "--operation-id="})
	if err != nil {
		return err
	}
	incumbentRaw, err := os.ReadFile(flags["--incumbent="])
	if err != nil {
		return err
	}
	candidateRaw, err := os.ReadFile(flags["--candidate="])
	if err != nil {
		return err
	}
	contractBytes, err := os.ReadFile(flags["--qualification-contract="])
	if err != nil {
		return err
	}
	var contract portplan.QualificationContractV1
	dec := json.NewDecoder(bytes.NewReader(contractBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&contract); err != nil {
		return fmt.Errorf("qualification contract: %w", err)
	}
	artifact, err := portplan.RenderCanonicalBenchstatArtifactV1(
		contract,
		flags["--operation-id="],
		flags["--incumbent-receipt-id="],
		flags["--candidate-receipt-id="],
		incumbentRaw,
		candidateRaw,
	)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(artifact)
	return err
}

func parseExactFlags(args, required []string) (map[string]string, error) {
	flags, proofs, err := parseExactFlagsWithProofs(args, required)
	if err != nil {
		return nil, err
	}
	if len(proofs) != 0 {
		return nil, errors.New("unexpected proof flags")
	}
	return flags, nil
}

func parseExactFlagsWithProofs(args, required []string) (map[string]string, []string, error) {
	if len(args) < len(required) {
		return nil, nil, errors.New("incorrect command argv")
	}
	out := make(map[string]string, len(required))
	for i, prefix := range required {
		if !strings.HasPrefix(args[i], prefix) {
			return nil, nil, fmt.Errorf("missing %s", prefix)
		}
		out[prefix] = strings.TrimPrefix(args[i], prefix)
	}
	var proofs []string
	for _, arg := range args[len(required):] {
		if !strings.HasPrefix(arg, "--proof-receipt-id=") {
			return nil, nil, fmt.Errorf("unexpected flag %q", arg)
		}
		proofs = append(proofs, strings.TrimPrefix(arg, "--proof-receipt-id="))
	}
	return out, proofs, nil
}

func writeCanonicalJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	return writeCanonicalJSONWriter(file, value)
}

func writeCanonicalJSONWriter(w io.Writer, value any) error {
	canonical, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = w.Write(append(canonical, '\n'))
	return err
}

func gitSHA(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func literalCPUSet(value string) bool {
	if value == "" || value[0] < '0' || value[0] > '9' {
		return false
	}
	for _, r := range value {
		if r == ',' || r == '-' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func readLinuxCpusAllowedList() (string, error) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Cpus_allowed_list:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed_list:")), nil
		}
	}
	return "", errors.New("Cpus_allowed_list missing")
}
