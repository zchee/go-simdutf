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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"path"
	"sort"
	"strconv"
	"strings"
)

const (
	evidenceSchemaV1                = "simdutf-port-evidence-v1"
	returnIndexHeaderV1             = "path\tmedia_type\tsize\tdigest\treceipt_id\tkey_kind\tstorage_id\tdisplay_id\ttuple_hex\tcommand_id\toutput_id\tproducer\tcampaign\trow\tcell\tdisposition\n"
	MaxRawEvidenceSizeV1      int64 = 256 << 20
	MaxSemanticEvidenceSizeV1 int64 = 32 << 20
)

var evidenceLanesV1 = map[string]bool{
	"darwin-arm64-nosimd": true, "darwin-arm64-simd-negative": true,
	"linux-amd64-none": true, "linux-amd64-simd": true,
	"linux-riscv64-compile": true, "linux-s390x-compile": true,
}

var evidenceKindsV1 = map[string]bool{
	"identity": true, "stdout": true, "stderr": true, "exit": true, "argv-env": true,
	"test": true, "race": true, "fuzz": true, "object": true, "disassembly": true,
	"benchmark": true, "incumbent-benchmark": true, "candidate-benchmark": true, "benchstat": true,
	"provider-guard": true, "selector": true, "final-selector": true, "state-transition": true,
	"not-applicable": true, "index": true, "quiet-affinity": true,
}

// EvidenceValidationContextV1 contains the complete frozen evidence matrix.
type EvidenceValidationContextV1 struct {
	AuthorityBytes                                       []byte
	AuthoritySHA256, FrozenInputSHA256                   string
	FamilyID, CampaignID, LaneID, ProducerID, Provider   string
	SourceRole, SourceCommit, SourceTree, SourceParent   string
	CommandManifestSHA256                                string
	QualificationContractID, QualificationContractDigest string
	QualificationContractBytes                           []byte
	CorpusID, CorpusDigest                               string
	HostReceiptID, HostReceiptDigest                     string
	ExpectedBindings                                     []EvidenceBindingV1
	OldSourceCommit, OldSourceTree, OldSourceParent      string
	NewSourceCommit, NewSourceTree, NewSourceParent      string
	HostIdentityDigest, IdentitySetDigest                string
	ExpectedCommands                                     []CampaignCommandV1
}

// EvidenceBindingV1 is one exact frozen subject, command, and output relation.
type EvidenceBindingV1 struct {
	KeyKind, StorageID, DisplayID, TupleHex                      string
	RowID, CellID, SymbolID, BatchID, TransactionID, OperationID string
	Backend, DirectSymbol                                        string
	OrderedMembers                                               []string
	CommandID, OutputID, Kind, Provider                          string
	InitialState, NAReason, NASource                             string
}

// EvidenceRecordV1 is a content-addressed command output receipt.
type EvidenceRecordV1 struct {
	Schema                      string   `json:"schema"`
	Version                     int      `json:"version"`
	KeyKind                     string   `json:"key_kind"`
	StorageID                   string   `json:"storage_id"`
	DisplayID                   string   `json:"display_id"`
	TupleHex                    string   `json:"tuple_hex"`
	FamilyID                    string   `json:"family_id"`
	CampaignID                  string   `json:"campaign_id"`
	LaneID                      string   `json:"lane_id"`
	ProducerID                  string   `json:"producer_id"`
	Provider                    string   `json:"provider"`
	Kind                        string   `json:"kind"`
	Path                        string   `json:"path"`
	OriginPath                  string   `json:"origin_path"`
	MediaType                   string   `json:"media_type"`
	Size                        int64    `json:"size"`
	Digest                      string   `json:"digest"`
	SourceCommit                string   `json:"source_commit"`
	SourceTree                  string   `json:"source_tree"`
	SourceParent                string   `json:"source_parent"`
	SourceRole                  string   `json:"source_role"`
	SourceDirty                 bool     `json:"source_dirty"`
	OriginCampaignID            string   `json:"origin_campaign_id"`
	OriginProducerID            string   `json:"origin_producer_id"`
	OriginSourceCommit          string   `json:"origin_source_commit"`
	OriginSourceTree            string   `json:"origin_source_tree"`
	OriginSourceParent          string   `json:"origin_source_parent"`
	RowID                       string   `json:"row_id"`
	CellID                      string   `json:"cell_id"`
	SymbolID                    string   `json:"symbol_id"`
	BatchID                     string   `json:"batch_id"`
	TransactionID               string   `json:"transaction_id"`
	OperationID                 string   `json:"operation_id"`
	Backend                     string   `json:"backend"`
	DirectSymbol                string   `json:"direct_symbol"`
	OrderedMembers              []string `json:"ordered_members"`
	StateSubject                string   `json:"state_subject"`
	PrerequisiteState           string   `json:"prerequisite_state"`
	CurrentState                string   `json:"current_state"`
	Disposition                 string   `json:"disposition"`
	GoQualification             string   `json:"go_qualification"`
	InitialState                string   `json:"initial_state"`
	ProofReceiptIDs             []string `json:"proof_receipt_ids"`
	NAReason                    string   `json:"na_reason"`
	NASource                    string   `json:"na_source"`
	IdentityValue               string   `json:"identity_value"`
	IdentitySetDigest           string   `json:"identity_set_digest"`
	CommandDigest               string   `json:"command_digest"`
	QualificationContractID     string   `json:"qualification_contract_id"`
	QualificationContractDigest string   `json:"qualification_contract_digest"`
	CorpusID                    string   `json:"corpus_id"`
	CorpusDigest                string   `json:"corpus_digest"`
	HostReceiptID               string   `json:"host_receipt_id"`
	AuthoritySHA256             string   `json:"authority_sha256"`
	HostReceiptDigest           string   `json:"host_receipt_digest"`
	CommandID                   string   `json:"command_id"`
	CommandOrdinal              int      `json:"command_ordinal"`
	CommandAction               string   `json:"command_action"`
	CommandRole                 string   `json:"command_role"`
	OutputID                    string   `json:"output_id"`
	ReceiptID                   string   `json:"receipt_id"`
}

// EvidenceRegistryV1 holds validated receipts and frozen state progress.
type EvidenceRegistryV1 struct {
	keys           map[string]struct{}
	receipts       map[string]EvidenceRecordV1
	contents       map[string][]byte
	qualifications map[string]QualificationContractV1
	states         map[string]string
}

func (r *EvidenceRegistryV1) Receipt(id string) (EvidenceRecordV1, bool) {
	if r == nil {
		return EvidenceRecordV1{}, false
	}
	v, ok := r.receipts[id]
	if !ok {
		return EvidenceRecordV1{}, false
	}
	return cloneEvidenceRecordV1(v), true
}

func (r *EvidenceRegistryV1) Snapshot() []EvidenceRecordV1 {
	if r == nil {
		return nil
	}
	ids := make([]string, 0, len(r.receipts))
	for id := range r.receipts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]EvidenceRecordV1, 0, len(ids))
	for _, id := range ids {
		out = append(out, cloneEvidenceRecordV1(r.receipts[id]))
	}
	return out
}

func cloneEvidenceRecordV1(record EvidenceRecordV1) EvidenceRecordV1 {
	record.OrderedMembers = append([]string(nil), record.OrderedMembers...)
	record.ProofReceiptIDs = append([]string(nil), record.ProofReceiptIDs...)
	return record
}

// EvidenceRegistryDigestV1 returns the SHA-256 of the canonical complete return
// index for the semantically validated receipt population.
func EvidenceRegistryDigestV1(registry *EvidenceRegistryV1) (string, error) {
	index, err := RenderEvidenceRegistryIndexV1(registry)
	if err != nil {
		return "", err
	}
	return SHA256Hex(index), nil
}
func (r *EvidenceRegistryV1) addValidated(record EvidenceRecordV1) error {
	key, err := r.validateAdd(record)
	if err != nil {
		return err
	}
	r.commitAdd(record, key)
	return nil
}

func (r *EvidenceRegistryV1) validateAdd(record EvidenceRecordV1) (string, error) {
	if r == nil {
		return "", errors.New("nil evidence registry")
	}
	key := strings.Join([]string{record.AuthoritySHA256, record.FamilyID, record.CampaignID, record.LaneID, record.Provider, record.ProducerID, record.Kind, record.CommandID, record.OutputID, record.KeyKind, record.StorageID}, "\x00")
	if _, ok := r.keys[key]; ok {
		return "", errors.New("duplicate evidence logical key")
	}
	if _, exists := r.receipts[record.ReceiptID]; exists {
		return "", errors.New("duplicate evidence receipt identity")
	}
	if record.StateSubject != "none" {
		stateKey := evidenceStateKeyV1(record)
		prior, exists := r.states[stateKey]
		if exists && prior != record.PrerequisiteState {
			return "", errors.New("state transition does not follow its campaign state")
		}
		if !exists && initialState(record.StateSubject, record) != record.PrerequisiteState {
			return "", errors.New("state transition skips its campaign initial state")
		}
	}
	return key, nil
}

func (r *EvidenceRegistryV1) commitAdd(record EvidenceRecordV1, key string) {
	if r.keys == nil {
		r.keys = map[string]struct{}{}
		r.receipts = map[string]EvidenceRecordV1{}
		r.states = map[string]string{}
	}
	record = cloneEvidenceRecordV1(record)
	r.keys[key] = struct{}{}
	r.receipts[record.ReceiptID] = record
	if record.StateSubject != "none" {
		r.states[evidenceStateKeyV1(record)] = record.CurrentState
	}
}

func evidenceStateKeyV1(record EvidenceRecordV1) string {
	return strings.Join([]string{
		record.StateSubject, record.StorageID, record.CampaignID, record.IdentitySetDigest,
	}, "\x00")
}
func initialState(subject string, r EvidenceRecordV1) string { return r.InitialState }

func ValidateEvidenceJSONV1(input, contents []byte, context EvidenceValidationContextV1, registry *EvidenceRegistryV1) (EvidenceRecordV1, error) {
	var r EvidenceRecordV1
	if registry == nil {
		return r, errors.New("evidence validation requires a semantic evidence registry")
	}
	d := json.NewDecoder(bytes.NewReader(input))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, err
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return r, errors.New("trailing JSON")
	}
	canonical, err := json.Marshal(r)
	if err != nil {
		return r, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(input, canonical) {
		return r, errors.New("noncanonical evidence JSON")
	}
	return r, ValidateEvidenceRecordV1(r, contents, context, registry)
}
func ValidateEvidenceRecordV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	if registry == nil {
		return errors.New("evidence validation requires a semantic evidence registry")
	}
	if int64(len(contents)) > MaxRawEvidenceSizeV1 {
		return errors.New("evidence exceeds maximum raw artifact size")
	}
	sum := sha256.Sum256(contents)
	if _, err := validateEvidenceEnvelopeV1(r, int64(len(contents)), hex.EncodeToString(sum[:]), c); err != nil {
		return err
	}
	return validateEvidenceContentsV1(r, contents, c, registry)
}

// ValidateEvidenceMetadataV1 revalidates streaming-observed metadata against a
// receipt that already crossed the semantic, content-bearing registry boundary.
func ValidateEvidenceMetadataV1(r EvidenceRecordV1, size int64, digest string, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	if _, err := validateEvidenceEnvelopeV1(r, size, digest, c); err != nil {
		return err
	}
	if registry == nil {
		return errors.New("metadata validation requires a semantic evidence registry")
	}
	stored, ok := registry.Receipt(r.ReceiptID)
	if !ok {
		return errors.New("metadata receipt was not semantically registered")
	}
	got, gotErr := json.Marshal(r)
	want, wantErr := json.Marshal(stored)
	if gotErr != nil || wantErr != nil || !bytes.Equal(got, want) {
		return errors.New("metadata differs from semantically registered receipt")
	}
	return nil
}
func validateEvidenceMetadataV1(r EvidenceRecordV1, c EvidenceValidationContextV1) error {
	_, err := validateEvidenceEnvelopeV1(r, r.Size, r.Digest, c)
	return err
}

func validateEvidenceContentsV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	return validateEvidenceStreamedV1(r, int64(len(contents)), r.Digest, contents, true, c, registry)
}

func validateEvidenceStreamedV1(r EvidenceRecordV1, size int64, digest string, contents []byte, hasContents bool, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	binding, err := validateEvidenceEnvelopeV1(r, size, digest, c)
	if err != nil {
		return err
	}
	if requiresEvidenceContentsV1(r.Kind) && !hasContents {
		return errors.New("semantic evidence bytes were not retained")
	}
	if hasContents && int64(len(contents)) > MaxSemanticEvidenceSizeV1 && requiresEvidenceContentsV1(r.Kind) {
		return errors.New("semantic evidence exceeds its bounded parsing limit")
	}
	if err := validateArtifactSemanticsV1(r, contents, c, registry); err != nil {
		return err
	}
	if err := validateState(r, binding, registry); err != nil {
		return err
	}
	if registry != nil {
		contract, addQualification, err := prepareEvidenceQualificationV1(c, registry)
		if err != nil {
			return err
		}
		key, err := registry.validateAdd(r)
		if err != nil {
			return err
		}
		registry.commitAdd(r, key)
		if addQualification {
			registry.commitQualification(c.QualificationContractDigest, contract)
		}
		if registry.contents == nil {
			registry.contents = map[string][]byte{}
		}
		if retainsEvidenceContentsV1(r.Kind) {
			registry.contents[r.ReceiptID] = append([]byte(nil), contents...)
		}
	}
	return nil
}

func requiresEvidenceContentsV1(kind string) bool {
	if retainsEvidenceContentsV1(kind) {
		return true
	}
	switch kind {
	case "state-transition", "not-applicable", "index":
		return true
	}
	return false
}

func retainsEvidenceContentsV1(kind string) bool {
	switch kind {
	case "identity", "exit", "argv-env", "incumbent-benchmark", "candidate-benchmark", "benchstat":
		return true
	}
	return false
}

func validateEvidenceEnvelopeV1(r EvidenceRecordV1, size int64, digest string, c EvidenceValidationContextV1) (EvidenceBindingV1, error) {
	if err := validateContext(c); err != nil {
		return EvidenceBindingV1{}, err
	}
	if r.Schema != evidenceSchemaV1 || r.Version != 1 || !evidenceKindsV1[r.Kind] {
		return EvidenceBindingV1{}, errors.New("unsupported evidence schema or kind")
	}
	if size < 0 || size > MaxRawEvidenceSizeV1 || r.Size != size || r.Digest != digest || !lowerHex(digest, 64) {
		return EvidenceBindingV1{}, errors.New("evidence bytes do not match metadata")
	}
	if r.AuthoritySHA256 != c.AuthoritySHA256 || r.FamilyID != c.FamilyID || r.CampaignID != c.CampaignID || r.LaneID != c.LaneID || r.ProducerID != c.ProducerID || r.Provider != c.Provider || r.SourceDirty || r.OriginCampaignID != c.CampaignID || r.OriginProducerID != c.ProducerID || r.IdentitySetDigest != c.IdentitySetDigest || r.CommandDigest != c.CommandManifestSHA256 || r.QualificationContractID != c.QualificationContractID || r.QualificationContractDigest != c.QualificationContractDigest || r.CorpusID != c.CorpusID || r.CorpusDigest != c.CorpusDigest || r.HostReceiptID != c.HostReceiptID || r.HostReceiptDigest != c.HostReceiptDigest {
		return EvidenceBindingV1{}, errors.New("record context mismatch")
	}
	if err := validateRecordSourceIdentityV1(r, c); err != nil {
		return EvidenceBindingV1{}, err
	}
	if !validRawEvidencePath(r, c) || r.MediaType == "" {
		return EvidenceBindingV1{}, errors.New("invalid evidence path or media type")
	}
	binding, err := bindingFor(r, c)
	if err != nil {
		return EvidenceBindingV1{}, err
	}
	if err := validateKeyAndBinding(r, binding); err != nil {
		return EvidenceBindingV1{}, err
	}
	if err := validateCommand(r, binding, c); err != nil {
		return EvidenceBindingV1{}, err
	}
	if r.ReceiptID != ReceiptIDV1(r) {
		return EvidenceBindingV1{}, errors.New("invalid receipt identity")
	}
	return binding, nil
}
func identitySetDigestV1(c EvidenceValidationContextV1) string {
	return SHA256Hex(EncodeTupleV1(
		"evidence-identity-set-v1",
		c.OldSourceCommit, c.OldSourceTree, c.OldSourceParent,
		c.NewSourceCommit, c.NewSourceTree, c.NewSourceParent,
		c.HostIdentityDigest,
	))
}

func hostReceiptIDV1(digest string) string {
	return "host-v1-" + digest
}

func evidenceQualificationContractV1(c EvidenceValidationContextV1) (QualificationContractV1, error) {
	var contract QualificationContractV1
	if len(c.QualificationContractBytes) == 0 {
		return contract, errors.New("evidence context lacks qualification contract bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(c.QualificationContractBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return contract, fmt.Errorf("evidence qualification contract: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return contract, errors.New("evidence qualification contract has trailing JSON")
	}
	canonical, err := json.Marshal(contract)
	if err != nil {
		return contract, err
	}
	canonical = append(canonical, '\n')
	family, familyErr := FamilyKeyV1(contract.FamilyContractID)
	if familyErr != nil || family.StorageID != c.FamilyID ||
		!bytes.Equal(c.QualificationContractBytes, canonical) ||
		c.QualificationContractDigest != SHA256Hex(canonical) ||
		c.QualificationContractID != contract.QualificationContractID ||
		contract.Schema != qualificationSchemaV1 || contract.Version != 1 ||
		contract.QualificationContractID != QualificationContractIDV1(contract) ||
		!sameQualificationStrings(contract.Providers, qualificationProviderOrderV1[:]) ||
		contract.OldSampleCount != 10 || contract.NewSampleCount != 10 ||
		contract.SampleOrder != "old_then_new" || contract.RequiredAllocsPerOp != 0 ||
		contract.MinimumBulkWinBPS != 300 || contract.MaximumProtectedSlowdownBPS != 200 ||
		contract.MaximumPValueMillionths != 50000 ||
		!contract.NoStatisticallySignificantSlowdown ||
		contract.InconclusiveOutcome != "direct_only" || contract.FailureOutcome != "direct_only" ||
		contract.SelectedOutcome != "selected" ||
		!sameQualificationStrings(contract.RequiredEvidenceKinds, qualificationEvidenceKindsV1[:]) ||
		len(contract.Rows) == 0 {
		return contract, errors.New("evidence qualification contract is not canonical or policy-bound")
	}
	names := make(map[string]bool, len(contract.Rows))
	operations := make(map[string]struct{ protected, bulk bool })
	for i, row := range contract.Rows {
		if row.Ordinal != i+1 || row.BenchmarkName == "" || names[row.BenchmarkName] ||
			row.RowDigest != QualificationBenchmarkRowDigestV1(row) ||
			row.GoDispatchStatus != "required_pending" {
			return contract, errors.New("evidence qualification contract has invalid benchmark rows")
		}
		names[row.BenchmarkName] = true
		coverage := operations[row.OperationID]
		coverage.protected = coverage.protected || row.Protected
		coverage.bulk = coverage.bulk || row.IntendedBulk
		operations[row.OperationID] = coverage
	}
	for _, coverage := range operations {
		if !coverage.protected || !coverage.bulk {
			return contract, errors.New("evidence qualification contract lacks protected or bulk coverage")
		}
	}
	return contract, nil
}

func prepareEvidenceQualificationV1(c EvidenceValidationContextV1, registry *EvidenceRegistryV1) (QualificationContractV1, bool, error) {
	if registry == nil {
		return QualificationContractV1{}, false, nil
	}
	contract, err := evidenceQualificationContractV1(c)
	if err != nil {
		return QualificationContractV1{}, false, err
	}
	if prior, ok := registry.qualifications[c.QualificationContractDigest]; ok {
		priorBytes, marshalErr := json.Marshal(prior)
		if marshalErr != nil || !bytes.Equal(append(priorBytes, '\n'), c.QualificationContractBytes) {
			return QualificationContractV1{}, false, errors.New("qualification digest maps to unequal contracts")
		}
		return contract, false, nil
	}
	return contract, true, nil
}

func (r *EvidenceRegistryV1) commitQualification(digest string, contract QualificationContractV1) {
	if r.qualifications == nil {
		r.qualifications = make(map[string]QualificationContractV1)
	}
	r.qualifications[digest] = contract
}

func validateContext(c EvidenceValidationContextV1) error {
	sum := sha256.Sum256(c.AuthorityBytes)
	d := hex.EncodeToString(sum[:])
	oldRole := c.SourceRole == "old"
	if len(c.AuthorityBytes) == 0 || c.AuthoritySHA256 != d || c.FrozenInputSHA256 != d ||
		!validID(c.FamilyID, "family-v1-") || !validID(c.CampaignID, "campaign-v1-") ||
		!evidenceLanesV1[c.LaneID] || !safeEvidencePart(c.ProducerID) || !canonicalProviderV1(c.Provider) ||
		(c.SourceRole != "old" && c.SourceRole != "new") ||
		!lowerHex(c.OldSourceCommit, 40) || !lowerHex(c.OldSourceTree, 40) || !lowerHex(c.OldSourceParent, 40) ||
		!lowerHex(c.NewSourceCommit, 40) || !lowerHex(c.NewSourceTree, 40) || !lowerHex(c.NewSourceParent, 40) ||
		c.SourceCommit != map[bool]string{true: c.OldSourceCommit, false: c.NewSourceCommit}[oldRole] ||
		c.SourceTree != map[bool]string{true: c.OldSourceTree, false: c.NewSourceTree}[oldRole] ||
		c.SourceParent != map[bool]string{true: c.OldSourceParent, false: c.NewSourceParent}[oldRole] ||
		!lowerHex(c.CommandManifestSHA256, 64) || !lowerHex(c.QualificationContractDigest, 64) ||
		!lowerHex(c.CorpusDigest, 64) || !safeEvidencePart(c.QualificationContractID) ||
		!safeEvidencePart(c.CorpusID) || !lowerHex(c.HostIdentityDigest, 64) ||
		c.HostReceiptDigest != c.HostIdentityDigest || c.HostReceiptID != hostReceiptIDV1(c.HostIdentityDigest) ||
		c.IdentitySetDigest != identitySetDigestV1(c) {
		return errors.New("invalid evidence context")
	}
	if _, err := evidenceQualificationContractV1(c); err != nil {
		return err
	}
	if len(c.ExpectedBindings) == 0 || len(c.ExpectedCommands) == 0 {
		return errors.New("empty frozen evidence topology")
	}
	bindings := make(map[string]struct{}, len(c.ExpectedBindings))
	for _, b := range c.ExpectedBindings {
		if err := validateBindingTopology(b); err != nil {
			return err
		}
		k := strings.Join([]string{b.KeyKind, b.StorageID, b.CommandID, b.OutputID, b.Kind}, "\x00")
		if _, ok := bindings[k]; ok {
			return errors.New("duplicate frozen evidence binding")
		}
		bindings[k] = struct{}{}
	}
	commands := make(map[string]CampaignCommandV1, len(c.ExpectedCommands))
	outputs := make(map[string]CommandOutputV1)
	for i, cmd := range c.ExpectedCommands {
		if err := validateCampaignCommandV1(cmd, i+1, c); err != nil {
			return fmt.Errorf("malformed frozen command: %w", err)
		}
		if _, ok := commands[cmd.ID]; ok {
			return errors.New("duplicate frozen command")
		}
		commands[cmd.ID] = cmd
		for _, out := range cmd.Outputs {
			key := cmd.ID + "\x00" + out.ID
			if _, ok := outputs[key]; ok {
				return errors.New("duplicate frozen command output")
			}
			outputs[key] = out
		}
	}
	canonicalCommands, err := RenderCanonicalCampaignCommandsV1(c.ExpectedCommands)
	if err != nil {
		return err
	}
	commandSum := sha256.Sum256(canonicalCommands)
	if c.CommandManifestSHA256 != hex.EncodeToString(commandSum[:]) {
		return errors.New("frozen command manifest digest mismatch")
	}
	boundOutputs := make(map[string]struct{}, len(c.ExpectedBindings))
	for _, b := range c.ExpectedBindings {
		cmd, cmdFound := commands[b.CommandID]
		output, outputFound := outputs[b.CommandID+"\x00"+b.OutputID]
		if !cmdFound || !outputFound || output.Kind != b.Kind {
			return errors.New("binding references inconsistent frozen command output")
		}
		if cmd.RowID != b.RowID || cmd.CellID != b.CellID || cmd.BatchID != b.BatchID || cmd.OperationID != b.OperationID || cmd.Provider != b.Provider {
			return errors.New("binding differs from frozen command subject")
		}
		key := b.CommandID + "\x00" + b.OutputID
		if _, duplicate := boundOutputs[key]; duplicate {
			return errors.New("command output is bound to multiple evidence subjects")
		}
		boundOutputs[key] = struct{}{}
	}
	if len(boundOutputs) != len(outputs) {
		return errors.New("frozen evidence bindings do not exactly cover command outputs")
	}
	identityRequired := map[string]bool{}
	for _, role := range []string{"old", "new"} {
		for _, action := range []string{"source_commit", "source_tree", "source_parent", "source_status"} {
			identityRequired[action+"\x00"+role] = false
		}
	}
	for _, action := range []string{"host_uname", "host_cpu", "go_version", "file_digest"} {
		identityRequired[action+"\x00host"] = false
	}
	for _, command := range c.ExpectedCommands {
		key := command.Action + "\x00" + command.Role
		if _, required := identityRequired[key]; required {
			if identityRequired[key] {
				return errors.New("duplicate frozen identity command")
			}
			identityRequired[key] = true
		}
	}
	for _, present := range identityRequired {
		if !present {
			return errors.New("frozen campaign omits source or host identity command")
		}
	}
	if err := validateBenchmarkTopologyV1(c.ExpectedCommands); err != nil {
		return err
	}
	return nil
}
func bindingFor(r EvidenceRecordV1, c EvidenceValidationContextV1) (EvidenceBindingV1, error) {
	var found *EvidenceBindingV1
	for i := range c.ExpectedBindings {
		b := &c.ExpectedBindings[i]
		if b.KeyKind == r.KeyKind && b.StorageID == r.StorageID && b.CommandID == r.CommandID && b.OutputID == r.OutputID && b.Kind == r.Kind {
			if found != nil {
				return EvidenceBindingV1{}, errors.New("ambiguous frozen evidence binding")
			}
			found = b
		}
	}
	if found == nil {
		return EvidenceBindingV1{}, errors.New("record absent from frozen evidence matrix")
	}
	return *found, nil
}
func validateKeyAndBinding(r EvidenceRecordV1, b EvidenceBindingV1) error {
	if r.DisplayID != b.DisplayID || r.TupleHex != b.TupleHex || r.RowID != b.RowID || r.CellID != b.CellID || r.SymbolID != b.SymbolID || r.BatchID != b.BatchID || r.TransactionID != b.TransactionID || r.OperationID != b.OperationID || r.Backend != b.Backend || r.DirectSymbol != b.DirectSymbol || !sameStrings(r.OrderedMembers, b.OrderedMembers) || r.InitialState != b.InitialState || r.Provider != b.Provider {
		return errors.New("record differs from frozen binding")
	}
	return validateTypedEvidenceKey(r)
}
func validateCommand(r EvidenceRecordV1, b EvidenceBindingV1, c EvidenceValidationContextV1) error {
	for _, cmd := range c.ExpectedCommands {
		if cmd.ID != r.CommandID {
			continue
		}
		if cmd.Ordinal != r.CommandOrdinal || cmd.Action != r.CommandAction || cmd.Role != r.CommandRole || cmd.RowID != r.RowID || cmd.CellID != r.CellID || cmd.BatchID != r.BatchID || cmd.OperationID != r.OperationID || cmd.Provider != r.Provider {
			return errors.New("record command differs from frozen command")
		}
		for _, out := range cmd.Outputs {
			if out.ID == r.OutputID && out.Kind == r.Kind && out.Path == r.OriginPath && out.MediaType == r.MediaType {
				return nil
			}
		}
		return errors.New("record output differs from frozen command")
	}
	return errors.New("unknown frozen command")
}
func validateRecordSourceIdentityV1(r EvidenceRecordV1, c EvidenceValidationContextV1) error {
	role := r.CommandRole
	sourceRole := "new"
	commit, tree, parent := c.NewSourceCommit, c.NewSourceTree, c.NewSourceParent
	if role == "old" {
		sourceRole = "old"
		commit, tree, parent = c.OldSourceCommit, c.OldSourceTree, c.OldSourceParent
	} else if role == "host" {
		sourceRole = "host"
	}
	if r.SourceRole != sourceRole {
		return errors.New("record source role does not match command role")
	}
	if r.SourceCommit != commit || r.SourceTree != tree || r.SourceParent != parent ||
		r.OriginSourceCommit != commit || r.OriginSourceTree != tree || r.OriginSourceParent != parent {
		return errors.New("record source identity differs from command role")
	}
	return nil
}

func validateArtifactSemanticsV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	if r.Kind != "identity" && r.IdentityValue != "" {
		return errors.New("non-identity evidence carries an identity value")
	}
	switch r.Kind {
	case "exit":
		if !bytes.Equal(contents, []byte("{\"exit_code\":0}\n")) {
			return errors.New("noncanonical or nonzero exit artifact")
		}
		return nil
	case "argv-env":
		if !canonicalArgvEnvArtifactV1(r, contents, c) {
			return errors.New("argv/environment artifact differs from frozen command")
		}
		return nil
	case "identity":
		if !identityArtifactV1(r, contents, c) {
			return errors.New("identity output is not exact command identity")
		}
		if registry == nil || !registeredSuccessfulExitV1(r, registry) {
			return errors.New("identity output lacks matching successful command execution")
		}
		return nil
	}
	if registry == nil {
		if proofArtifactV1(r.Kind) {
			return errors.New("proof validation requires the registered campaign identity set")
		}
		return nil
	}
	if !identityBoundV1(r, c, registry) {
		return errors.New("record lacks the exact registered campaign identity set")
	}
	switch r.Kind {
	case "state-transition", "not-applicable":
		if !canonicalStateArtifactV1(r, contents) {
			return errors.New("state artifact does not equal receipt state")
		}
	case "quiet-affinity":
		if !canonicalQuietAffinityArtifactV1(r, contents, c) {
			return errors.New("quiet-affinity artifact does not equal commanded affinity")
		}
	case "index":
		if !canonicalIndexArtifactV1(r, contents) {
			return errors.New("index proof does not equal its frozen campaign identity")
		}
	case "incumbent-benchmark", "candidate-benchmark":
		if bytes.ContainsAny(contents, "\x00\r") {
			return errors.New("benchmark output is not canonical text")
		}
	case "benchstat":
		if err := validateBenchstatArtifactV1(r, contents, c, registry); err != nil {
			return err
		}
	}
	if proofArtifactV1(r.Kind) && !registeredSuccessfulExitV1(r, registry) {
		return errors.New("proof lacks matching successful command execution")
	}
	return nil
}

type commandInvocationArtifactV1 struct {
	Schema         string            `json:"schema"`
	Version        int               `json:"version"`
	CommandID      string            `json:"command_id"`
	Argv           []string          `json:"argv"`
	CWD            string            `json:"cwd"`
	Env            map[string]string `json:"env"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	ExpectedExit   int               `json:"expected_exit"`
}

func canonicalArgvEnvArtifactV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1) bool {
	for _, command := range c.ExpectedCommands {
		if command.ID != r.CommandID {
			continue
		}
		artifact := commandInvocationArtifactV1{
			Schema: "simdutf-command-invocation-v1", Version: 1, CommandID: command.ID,
			Argv: command.Argv, CWD: command.CWD, Env: command.Env,
			TimeoutSeconds: command.TimeoutSeconds, ExpectedExit: command.ExpectedExit,
		}
		canonical, err := json.Marshal(artifact)
		return err == nil && bytes.Equal(contents, append(canonical, '\n'))
	}
	return false
}

func identityArtifactV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1) bool {
	switch r.CommandAction {
	case "source_status":
		return r.CommandRole == r.SourceRole && len(contents) == 0 && r.IdentityValue == "clean"
	case "source_commit":
		return r.CommandRole == r.SourceRole && r.IdentityValue == r.SourceCommit && bytes.Equal(contents, []byte(r.SourceCommit+"\n"))
	case "source_tree":
		return r.CommandRole == r.SourceRole && r.IdentityValue == r.SourceTree && bytes.Equal(contents, []byte(r.SourceTree+"\n"))
	case "source_parent":
		return r.CommandRole == r.SourceRole && r.IdentityValue == r.SourceParent && bytes.Equal(contents, []byte(r.SourceParent+"\n"))
	case "host_uname", "host_cpu", "go_version":
		if r.CommandRole != "host" || len(contents) == 0 || contents[len(contents)-1] != '\n' ||
			bytes.ContainsAny(contents, "\x00\r") {
			return false
		}
		return r.IdentityValue == SHA256Hex(contents)
	case "file_digest":
		return fileDigestArtifactV1(r, contents, c)
	default:
		return false
	}
}

func fileDigestArtifactV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1) bool {
	if r.CommandRole != "host" || !lowerHex(r.IdentityValue, 64) {
		return false
	}
	for _, command := range c.ExpectedCommands {
		if command.ID != r.CommandID {
			continue
		}
		if command.Action != "file_digest" || command.Role != "host" || len(command.Argv) < 2 {
			return false
		}
		commandedPath := command.Argv[len(command.Argv)-1]
		return validOriginPath(commandedPath) &&
			bytes.Equal(contents, []byte(r.IdentityValue+"  "+commandedPath+"\n"))
	}
	return false
}

func canonicalStateArtifactV1(r EvidenceRecordV1, contents []byte) bool {
	artifact := struct {
		Schema            string `json:"schema"`
		Version           int    `json:"version"`
		StateSubject      string `json:"state_subject"`
		PrerequisiteState string `json:"prerequisite_state"`
		CurrentState      string `json:"current_state"`
		Disposition       string `json:"disposition"`
		GoQualification   string `json:"go_qualification"`
		NAReason          string `json:"na_reason"`
		NASource          string `json:"na_source"`
	}{
		Schema: "simdutf-state-transition-v1", Version: 1,
		StateSubject: r.StateSubject, PrerequisiteState: r.PrerequisiteState,
		CurrentState: r.CurrentState, Disposition: r.Disposition,
		GoQualification: r.GoQualification, NAReason: r.NAReason, NASource: r.NASource,
	}
	canonical, err := json.Marshal(artifact)
	return err == nil && bytes.Equal(contents, append(canonical, '\n'))
}

func canonicalQuietAffinityArtifactV1(r EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1) bool {
	if r.CommandAction != "quiet_affinity_recheck" {
		return false
	}
	for _, command := range c.ExpectedCommands {
		if command.ID != r.CommandID {
			continue
		}
		if command.Action != "quiet_affinity_recheck" || command.Env["SIMDUTF_CPU"] == "" || command.Env["SIMDUTF_AFFINITY"] == "" {
			return false
		}
		cpu := command.Env["SIMDUTF_CPU"]
		policy := command.Env["SIMDUTF_AFFINITY"]
		for _, arg := range command.Argv {
			switch {
			case strings.HasPrefix(arg, "--cpu="):
				if strings.TrimPrefix(arg, "--cpu=") != cpu {
					return false
				}
			case strings.HasPrefix(arg, "--policy="):
				if strings.TrimPrefix(arg, "--policy=") != policy {
					return false
				}
			}
		}
		artifact := struct {
			Schema  string `json:"schema"`
			Version int    `json:"version"`
			CPU     string `json:"cpu"`
			Policy  string `json:"policy"`
			Status  string `json:"status"`
		}{
			Schema: "simdutf-quiet-affinity-v1", Version: 1,
			CPU: cpu, Policy: policy, Status: "quiet",
		}
		canonical, err := json.Marshal(artifact)
		return err == nil && bytes.Equal(contents, append(canonical, '\n'))
	}
	return false
}

func canonicalIndexArtifactV1(r EvidenceRecordV1, contents []byte) bool {
	artifact := struct {
		Schema                string `json:"schema"`
		Version               int    `json:"version"`
		AuthoritySHA256       string `json:"authority_sha256"`
		CampaignID            string `json:"campaign_id"`
		CommandManifestSHA256 string `json:"command_manifest_sha256"`
		IdentitySetDigest     string `json:"identity_set_digest"`
	}{
		Schema: "simdutf-return-index-proof-v1", Version: 1,
		AuthoritySHA256: r.AuthoritySHA256, CampaignID: r.CampaignID,
		CommandManifestSHA256: r.CommandDigest, IdentitySetDigest: r.IdentitySetDigest,
	}
	canonical, err := json.Marshal(artifact)
	return err == nil && bytes.Equal(contents, append(canonical, '\n'))
}

func proofArtifactV1(kind string) bool {
	switch kind {
	case "test", "race", "fuzz", "object", "disassembly",
		"incumbent-benchmark", "candidate-benchmark", "benchstat",
		"provider-guard", "selector", "final-selector",
		"quiet-affinity", "state-transition", "not-applicable", "index":
		return true
	}
	return false
}

func sameCampaignEvidenceV1(a, b EvidenceRecordV1) bool {
	return a.AuthoritySHA256 == b.AuthoritySHA256 && a.FamilyID == b.FamilyID &&
		a.CampaignID == b.CampaignID && a.LaneID == b.LaneID &&
		a.ProducerID == b.ProducerID && a.Provider == b.Provider &&
		a.CommandDigest == b.CommandDigest && a.IdentitySetDigest == b.IdentitySetDigest &&
		a.QualificationContractID == b.QualificationContractID &&
		a.QualificationContractDigest == b.QualificationContractDigest &&
		a.CorpusID == b.CorpusID && a.CorpusDigest == b.CorpusDigest &&
		a.HostReceiptID == b.HostReceiptID && a.HostReceiptDigest == b.HostReceiptDigest
}

func sameCommandExecutionV1(a, b EvidenceRecordV1) bool {
	return sameCampaignEvidenceV1(a, b) && a.CommandID == b.CommandID &&
		a.CommandOrdinal == b.CommandOrdinal && a.CommandAction == b.CommandAction &&
		a.CommandRole == b.CommandRole && a.RowID == b.RowID && a.CellID == b.CellID &&
		a.BatchID == b.BatchID && a.OperationID == b.OperationID &&
		a.SourceRole == b.SourceRole && a.SourceCommit == b.SourceCommit &&
		a.SourceTree == b.SourceTree && a.SourceParent == b.SourceParent
}

func registeredCommandOutputV1(r EvidenceRecordV1, outputID, kind string, registry *EvidenceRegistryV1) (EvidenceRecordV1, bool) {
	if registry == nil {
		return EvidenceRecordV1{}, false
	}
	var found EvidenceRecordV1
	for _, candidate := range registry.receipts {
		if candidate.OutputID != outputID || candidate.Kind != kind || !sameCommandExecutionV1(r, candidate) {
			continue
		}
		if found.ReceiptID != "" {
			return EvidenceRecordV1{}, false
		}
		found = candidate
	}
	return found, found.ReceiptID != ""
}

func registeredSuccessfulExitV1(r EvidenceRecordV1, registry *EvidenceRegistryV1) bool {
	exit, ok := registeredCommandOutputV1(r, "exit", "exit", registry)
	if !ok || !bytes.Equal(registry.contents[exit.ReceiptID], []byte("{\"exit_code\":0}\n")) {
		return false
	}
	invocation, ok := registeredCommandOutputV1(r, "argv-env", "argv-env", registry)
	return ok && len(registry.contents[invocation.ReceiptID]) != 0
}

func identityBoundV1(r EvidenceRecordV1, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) bool {
	sourceNeeded := map[string]map[string]string{
		"old": {
			"source_commit": c.OldSourceCommit, "source_tree": c.OldSourceTree,
			"source_parent": c.OldSourceParent, "source_status": "clean",
		},
		"new": {
			"source_commit": c.NewSourceCommit, "source_tree": c.NewSourceTree,
			"source_parent": c.NewSourceParent, "source_status": "clean",
		},
	}
	sourceSeen := map[string]map[string]bool{"old": {}, "new": {}}
	hostValues := map[string]string{}
	for _, record := range registry.receipts {
		if record.Kind != "identity" || !sameCampaignEvidenceV1(r, record) ||
			!identityArtifactV1(record, registry.contents[record.ReceiptID], c) ||
			!registeredSuccessfulExitV1(record, registry) {
			continue
		}
		if expected, ok := sourceNeeded[record.CommandRole][record.CommandAction]; ok {
			if record.IdentityValue != expected || sourceSeen[record.CommandRole][record.CommandAction] {
				return false
			}
			sourceSeen[record.CommandRole][record.CommandAction] = true
			continue
		}
		if record.CommandRole == "host" {
			switch record.CommandAction {
			case "host_uname", "host_cpu", "go_version", "file_digest":
				if hostValues[record.CommandAction] != "" {
					return false
				}
				hostValues[record.CommandAction] = record.IdentityValue
			}
		}
	}
	for role, actions := range sourceNeeded {
		for action := range actions {
			if !sourceSeen[role][action] {
				return false
			}
		}
	}
	hostDigest := SHA256Hex(EncodeTupleV1(
		"evidence-host-identity-v1",
		hostValues["host_uname"], hostValues["host_cpu"],
		hostValues["go_version"], hostValues["file_digest"],
	))
	return hostValues["host_uname"] != "" && hostValues["host_cpu"] != "" &&
		hostValues["go_version"] != "" && hostValues["file_digest"] != "" &&
		hostDigest == c.HostIdentityDigest && c.IdentitySetDigest == identitySetDigestV1(c)
}

func validateState(r EvidenceRecordV1, b EvidenceBindingV1, registry *EvidenceRegistryV1) error {
	if r.StateSubject == "none" {
		if r.PrerequisiteState != "" || r.CurrentState != "" || r.Disposition != "" || r.GoQualification != "" || len(r.ProofReceiptIDs) != 0 || r.NAReason != "" || r.NASource != "" {
			return errors.New("raw evidence carries state fields")
		}
		return nil
	}
	if r.StateSubject != "row" && r.StateSubject != "backend_cell" {
		return errors.New("invalid state subject")
	}
	if r.StateSubject == "row" {
		if r.KeyKind != "row" || r.StorageID != r.RowID {
			return errors.New("row state subject does not match its typed key")
		}
	} else if r.KeyKind != "cell" || r.StorageID != r.CellID {
		return errors.New("backend-cell state subject does not match its typed key")
	}
	if r.StateSubject == "row" {
		if r.Kind != "state-transition" || !legalRowTransition(r.PrerequisiteState, r.CurrentState) || r.Disposition != "" || r.GoQualification != "" || r.NAReason != "" || r.NASource != "" {
			return errors.New("invalid row transition")
		}
	} else if r.CurrentState == "not_applicable" {
		if r.Kind != "not-applicable" || r.PrerequisiteState != "eligible" || r.Disposition != "not_applicable" || r.NAReason != b.NAReason || r.NASource != b.NASource || r.SymbolID != "" || len(r.ProofReceiptIDs) != 0 {
			return errors.New("invalid not-applicable transition")
		}
		return nil
	} else if r.Kind != "state-transition" || !legalCellTransition(r.PrerequisiteState, r.CurrentState) || r.NAReason != "" || r.NASource != "" {
		return errors.New("invalid cell transition")
	}
	if registry == nil {
		return errors.New("state transition requires registered proofs")
	}
	kinds := map[string]bool{}
	for _, id := range r.ProofReceiptIDs {
		p, ok := registry.Receipt(id)
		if !ok {
			return errors.New("unknown proof receipt")
		}
		if !sameProofContext(r, p) {
			return errors.New("proof context mismatch")
		}
		if p.Kind == "state-transition" {
			if r.CurrentState != "dispatch_candidate" || p.StateSubject != "backend_cell" || p.PrerequisiteState != "direct_built" || p.CurrentState != "hard_gates_green" {
				return errors.New("unexpected transition proof")
			}
		} else if p.StateSubject != "none" {
			return errors.New("proof is not raw evidence")
		}
		if kinds[p.Kind] {
			return errors.New("duplicate proof kind")
		}
		kinds[p.Kind] = true
	}
	require := func(kindsNeeded ...string) bool {
		for _, kind := range kindsNeeded {
			if !kinds[kind] {
				return false
			}
		}
		return true
	}
	if r.CurrentState == "selected" || r.CurrentState == "direct_only" {
		outcome, err := qualificationOutcomeV1(r, registry)
		if err != nil || r.GoQualification != outcome {
			return errors.New("terminal qualification is not derived from benchmark evidence")
		}
		if r.CurrentState == "selected" && outcome != "pass" {
			return errors.New("selected terminal outcome is not qualified")
		}
		if r.CurrentState == "direct_only" && outcome == "pass" {
			return errors.New("direct-only terminal outcome is qualified")
		}
	}
	switch r.CurrentState {
	case "family_published", "complete":
		if !require("index") {
			return errors.New("row publication lacks typed proof")
		}
	case "direct_built":
		if !require("object", "disassembly", "test") {
			return errors.New("direct build lacks direct proof")
		}
	case "hard_gates_green":
		if !require("test", "race", "fuzz", "object", "disassembly") {
			return errors.New("hard gates lack required proof")
		}
	case "dispatch_candidate":
		if !require("state-transition") || r.GoQualification != "" || r.Disposition != "" {
			return errors.New("dispatch candidate lacks hard-gate transition")
		}
	case "selected":
		if r.Disposition != "selected" || r.GoQualification != "pass" || !requireTerminalSelectionProof(kinds) {
			return errors.New("selected lacks qualification proof")
		}
	case "direct_only":
		if r.Disposition != "direct_only" || (r.GoQualification != "fail" && r.GoQualification != "inconclusive") || !requireDirectOnlyProofV1(kinds) {
			return errors.New("direct-only lacks qualification proof")
		}
	default:
		if r.Disposition != "" || r.GoQualification != "" {
			return errors.New("nonterminal state has terminal disposition")
		}
	}
	return nil
}

type benchmarkSamplesV1 struct {
	NanosMillionths []int64
	ZeroAllocs      bool
	Malformed       bool
}

type benchstatRowV1 struct {
	BenchmarkName              string `json:"benchmark_name"`
	OldMedianNanosMillionthsX2 int64  `json:"old_median_nanos_millionths_x2"`
	NewMedianNanosMillionthsX2 int64  `json:"new_median_nanos_millionths_x2"`
	PValueMillionths           int    `json:"p_value_millionths"`
}

type benchstatArtifactV1 struct {
	Schema                  string           `json:"schema"`
	Version                 int              `json:"version"`
	QualificationContractID string           `json:"qualification_contract_id"`
	IncumbentReceiptID      string           `json:"incumbent_receipt_id"`
	CandidateReceiptID      string           `json:"candidate_receipt_id"`
	Rows                    []benchstatRowV1 `json:"rows"`
}

func qualificationRowsForRecordV1(contract QualificationContractV1, record EvidenceRecordV1) []QualificationBenchmarkRowV1 {
	rows := make([]QualificationBenchmarkRowV1, 0)
	for _, row := range contract.Rows {
		if row.OperationID == record.OperationID {
			rows = append(rows, row)
		}
	}
	return rows
}

func parseDecimalMillionthsV1(value string) (int64, bool) {
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, false
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts) == 2 && (parts[1] == "" || len(parts[1]) > 6) {
		return 0, false
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > (int64(^uint64(0)>>1)-999999)/1000000 {
		return 0, false
	}
	fraction := int64(0)
	if len(parts) == 2 {
		padded := parts[1] + strings.Repeat("0", 6-len(parts[1]))
		fraction, err = strconv.ParseInt(padded, 10, 64)
		if err != nil {
			return 0, false
		}
	}
	return whole*1000000 + fraction, true
}

func benchmarkNameWithoutCPUCountV1(name string) string {
	at := strings.LastIndexByte(name, '-')
	if at < 0 || at == len(name)-1 {
		return name
	}
	for _, digit := range name[at+1:] {
		if digit < '0' || digit > '9' {
			return name
		}
	}
	return name[:at]
}

func parseBenchmarkSamplesV1(raw []byte, rows []QualificationBenchmarkRowV1) (map[string]benchmarkSamplesV1, error) {
	if bytes.ContainsAny(raw, "\x00\r") {
		return nil, errors.New("benchmark output is not canonical text")
	}
	result := make(map[string]benchmarkSamplesV1, len(rows))
	for _, row := range rows {
		if result[row.BenchmarkName].NanosMillionths != nil {
			return nil, errors.New("duplicate qualification benchmark name")
		}
		result[row.BenchmarkName] = benchmarkSamplesV1{NanosMillionths: []int64{}, ZeroAllocs: true}
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		name := benchmarkNameWithoutCPUCountV1(fields[0])
		samples, expected := result[name]
		if !expected {
			return nil, fmt.Errorf("unexpected benchmark row %q", name)
		}
		nsAt, allocAt := -1, -1
		for i, field := range fields {
			if field == "ns/op" {
				nsAt = i
			}
			if field == "allocs/op" {
				allocAt = i
			}
		}
		if nsAt < 1 || allocAt < 1 {
			samples.Malformed = true
			result[name] = samples
			continue
		}
		nanos, nanosOK := parseDecimalMillionthsV1(fields[nsAt-1])
		allocs, allocsOK := parseDecimalMillionthsV1(fields[allocAt-1])
		if !nanosOK || nanos <= 0 || !allocsOK {
			samples.Malformed = true
			result[name] = samples
			continue
		}
		samples.NanosMillionths = append(samples.NanosMillionths, nanos)
		samples.ZeroAllocs = samples.ZeroAllocs && allocs == 0
		result[name] = samples
	}
	return result, nil
}

func benchmarkMedianX2V1(samples benchmarkSamplesV1) (int64, bool) {
	if samples.Malformed || len(samples.NanosMillionths) != 10 {
		return 0, false
	}
	values := append([]int64(nil), samples.NanosMillionths...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	const maxInt64 = int64(^uint64(0) >> 1)
	if values[4] <= 0 || values[5] > maxInt64-values[4] {
		return 0, false
	}
	return values[4] + values[5], true
}

func RenderCanonicalBenchstatArtifactV1(contract QualificationContractV1, operationID, incumbentReceiptID, candidateReceiptID string, incumbentRaw, candidateRaw []byte) ([]byte, error) {
	if operationID == "" || incumbentReceiptID == "" || candidateReceiptID == "" {
		return nil, errors.New("benchstat artifact identity is incomplete")
	}
	record := EvidenceRecordV1{OperationID: operationID}
	rows := qualificationRowsForRecordV1(contract, record)
	if len(rows) == 0 {
		return nil, errors.New("benchstat artifact has no frozen operation rows")
	}
	oldSamples, err := parseBenchmarkSamplesV1(incumbentRaw, rows)
	if err != nil {
		return nil, err
	}
	newSamples, err := parseBenchmarkSamplesV1(candidateRaw, rows)
	if err != nil {
		return nil, err
	}
	artifact := benchstatArtifactV1{
		Schema: "simdutf-benchstat-v1", Version: 1,
		QualificationContractID: contract.QualificationContractID,
		IncumbentReceiptID:      incumbentReceiptID, CandidateReceiptID: candidateReceiptID,
		Rows: make([]benchstatRowV1, 0, len(rows)),
	}
	for _, row := range rows {
		oldRow := oldSamples[row.BenchmarkName]
		newRow := newSamples[row.BenchmarkName]
		oldMedian, oldComplete := benchmarkMedianX2V1(oldRow)
		newMedian, newComplete := benchmarkMedianX2V1(newRow)
		pValue := -1
		if oldComplete && newComplete {
			value, pErr := mannWhitneyPValueMillionthsV1(oldRow.NanosMillionths, newRow.NanosMillionths)
			if pErr != nil {
				return nil, pErr
			}
			pValue = value
		}
		artifact.Rows = append(artifact.Rows, benchstatRowV1{
			BenchmarkName:              row.BenchmarkName,
			OldMedianNanosMillionthsX2: oldMedian,
			NewMedianNanosMillionthsX2: newMedian,
			PValueMillionths:           pValue,
		})
	}
	canonical, err := json.Marshal(artifact)
	if err != nil {
		return nil, err
	}
	return append(canonical, '\n'), nil
}

func decodeBenchstatArtifactV1(contents []byte) (benchstatArtifactV1, error) {
	var artifact benchstatArtifactV1
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&artifact); err != nil {
		return artifact, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return artifact, errors.New("benchstat artifact has trailing JSON")
	}
	canonical, err := json.Marshal(artifact)
	if err != nil || !bytes.Equal(contents, append(canonical, '\n')) {
		return artifact, errors.New("benchstat artifact is not canonical")
	}
	return artifact, nil
}

func benchmarkProofsV1(record EvidenceRecordV1, registry *EvidenceRegistryV1) (EvidenceRecordV1, EvidenceRecordV1, error) {
	var old, candidate EvidenceRecordV1
	for _, proof := range registry.receipts {
		if !sameCampaignEvidenceV1(record, proof) || proof.RowID != record.RowID ||
			proof.CellID != record.CellID || proof.OperationID != record.OperationID {
			continue
		}
		switch proof.Kind {
		case "incumbent-benchmark":
			if old.ReceiptID != "" {
				return old, candidate, errors.New("duplicate incumbent benchmark proof")
			}
			old = proof
		case "candidate-benchmark":
			if candidate.ReceiptID != "" {
				return old, candidate, errors.New("duplicate candidate benchmark proof")
			}
			candidate = proof
		}
	}
	if old.ReceiptID == "" || candidate.ReceiptID == "" ||
		!registeredSuccessfulExitV1(old, registry) || !registeredSuccessfulExitV1(candidate, registry) {
		return old, candidate, errors.New("benchmark pair is incomplete or unsuccessful")
	}
	return old, candidate, nil
}

func validateBenchstatArtifactV1(record EvidenceRecordV1, contents []byte, c EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	contract, err := evidenceQualificationContractV1(c)
	if err != nil {
		return err
	}
	rows := qualificationRowsForRecordV1(contract, record)
	if len(rows) == 0 {
		return errors.New("benchstat artifact has no frozen operation rows")
	}
	old, candidate, err := benchmarkProofsV1(record, registry)
	if err != nil {
		return err
	}
	oldSamples, err := parseBenchmarkSamplesV1(registry.contents[old.ReceiptID], rows)
	if err != nil {
		return err
	}
	newSamples, err := parseBenchmarkSamplesV1(registry.contents[candidate.ReceiptID], rows)
	if err != nil {
		return err
	}
	artifact, err := decodeBenchstatArtifactV1(contents)
	if err != nil {
		return err
	}
	if artifact.Schema != "simdutf-benchstat-v1" || artifact.Version != 1 ||
		artifact.QualificationContractID != contract.QualificationContractID ||
		artifact.IncumbentReceiptID != old.ReceiptID || artifact.CandidateReceiptID != candidate.ReceiptID ||
		len(artifact.Rows) != len(rows) {
		return errors.New("benchstat artifact identity or population mismatch")
	}
	for i, row := range rows {
		comparison := artifact.Rows[i]
		oldRow := oldSamples[row.BenchmarkName]
		newRow := newSamples[row.BenchmarkName]
		oldMedian, oldComplete := benchmarkMedianX2V1(oldRow)
		newMedian, newComplete := benchmarkMedianX2V1(newRow)
		if comparison.BenchmarkName != row.BenchmarkName ||
			comparison.OldMedianNanosMillionthsX2 != oldMedian ||
			comparison.NewMedianNanosMillionthsX2 != newMedian ||
			comparison.PValueMillionths < -1 || comparison.PValueMillionths > 1000000 ||
			(!oldComplete || !newComplete) && comparison.PValueMillionths != -1 {
			return errors.New("benchstat row differs from raw benchmark evidence")
		}
		if oldComplete && newComplete {
			pValue, pErr := mannWhitneyPValueMillionthsV1(
				oldRow.NanosMillionths,
				newRow.NanosMillionths,
			)
			if pErr != nil || comparison.PValueMillionths != pValue {
				return errors.New("benchstat p-value differs from raw benchmark evidence")
			}
		}
	}
	return nil
}

func changeBPSV1(oldMedianX2, newMedianX2 int64) (int, error) {
	if oldMedianX2 <= 0 || newMedianX2 <= 0 {
		return 0, errors.New("invalid benchmark median")
	}
	numerator := new(big.Int).Sub(big.NewInt(oldMedianX2), big.NewInt(newMedianX2))
	numerator.Mul(numerator, big.NewInt(10000))
	numerator.Quo(numerator, big.NewInt(oldMedianX2))
	if !numerator.IsInt64() {
		return 0, errors.New("benchmark change exceeds integer range")
	}
	value := numerator.Int64()
	if int64(int(value)) != value {
		return 0, errors.New("benchmark change exceeds platform integer range")
	}
	return int(value), nil
}

func compareBenchmarkChangeV1(oldMedianX2, newMedianX2 int64, thresholdBPS int) (int, error) {
	if oldMedianX2 <= 0 || newMedianX2 <= 0 {
		return 0, errors.New("invalid benchmark median")
	}
	left := new(big.Int).Sub(big.NewInt(oldMedianX2), big.NewInt(newMedianX2))
	left.Mul(left, big.NewInt(10000))
	right := new(big.Int).Mul(big.NewInt(int64(thresholdBPS)), big.NewInt(oldMedianX2))
	return left.Cmp(right), nil
}

func mannWhitneyPValueMillionthsV1(oldSamples, newSamples []int64) (int, error) {
	n1, n2 := len(oldSamples), len(newSamples)
	if n1 == 0 || n2 == 0 || n1+n2 > 50 {
		return 0, errors.New("invalid Mann-Whitney sample population")
	}
	type rankedSample struct {
		value int64
		old   bool
	}
	combined := make([]rankedSample, 0, n1+n2)
	for _, value := range oldSamples {
		if value <= 0 {
			return 0, errors.New("invalid Mann-Whitney sample")
		}
		combined = append(combined, rankedSample{value: value, old: true})
	}
	for _, value := range newSamples {
		if value <= 0 {
			return 0, errors.New("invalid Mann-Whitney sample")
		}
		combined = append(combined, rankedSample{value: value})
	}
	sort.SliceStable(combined, func(i, j int) bool { return combined[i].value < combined[j].value })

	ranksX2 := make([]int, len(combined))
	observedRankSumX2 := 0
	for first := 0; first < len(combined); {
		last := first + 1
		for last < len(combined) && combined[last].value == combined[first].value {
			last++
		}
		rankX2 := first + 1 + last
		for i := first; i < last; i++ {
			ranksX2[i] = rankX2
			if combined[i].old {
				observedRankSumX2 += rankX2
			}
		}
		first = last
	}
	observedUX2 := observedRankSumX2 - n1*(n1+1)
	mirrorUX2 := 2*n1*n2 - observedUX2
	if observedUX2 == mirrorUX2 {
		return 1000000, nil
	}
	limitUX2 := observedUX2
	if mirrorUX2 < limitUX2 {
		limitUX2 = mirrorUX2
	}

	counts := make([]map[int]uint64, n1+1)
	for i := range counts {
		counts[i] = make(map[int]uint64)
	}
	counts[0][0] = 1
	seen := 0
	for _, rankX2 := range ranksX2 {
		seen++
		maxSelected := n1
		if seen < maxSelected {
			maxSelected = seen
		}
		for selected := maxSelected; selected > 0; selected-- {
			for sum, count := range counts[selected-1] {
				counts[selected][sum+rankX2] += count
			}
		}
	}
	var lowerTail, total uint64
	for rankSumX2, count := range counts[n1] {
		total += count
		if rankSumX2-n1*(n1+1) <= limitUX2 {
			lowerTail += count
		}
	}
	if total == 0 || lowerTail > total {
		return 0, errors.New("invalid Mann-Whitney distribution")
	}
	numerator := lowerTail * 2 * 1000000
	pValue := (numerator + total/2) / total
	if pValue > 1000000 {
		pValue = 1000000
	}
	return int(pValue), nil
}

func qualificationOutcomeV1(record EvidenceRecordV1, registry *EvidenceRegistryV1) (string, error) {
	contract, ok := registry.qualifications[record.QualificationContractDigest]
	if !ok || contract.QualificationContractID != record.QualificationContractID {
		return "", errors.New("terminal transition lacks its validated qualification contract")
	}
	var old, candidate, stat EvidenceRecordV1
	proofSet := make(map[string]bool, len(record.ProofReceiptIDs))
	for _, id := range record.ProofReceiptIDs {
		proof, found := registry.Receipt(id)
		if !found || !sameProofContext(record, proof) || !registeredSuccessfulExitV1(proof, registry) {
			return "", errors.New("terminal proof is missing, stale, or unsuccessful")
		}
		proofSet[id] = true
		switch proof.Kind {
		case "incumbent-benchmark":
			old = proof
		case "candidate-benchmark":
			candidate = proof
		case "benchstat":
			stat = proof
		}
	}
	if old.ReceiptID == "" || candidate.ReceiptID == "" || stat.ReceiptID == "" {
		return "", errors.New("terminal qualification proof population is incomplete")
	}
	artifact, err := decodeBenchstatArtifactV1(registry.contents[stat.ReceiptID])
	if err != nil || artifact.IncumbentReceiptID != old.ReceiptID ||
		artifact.CandidateReceiptID != candidate.ReceiptID ||
		!proofSet[artifact.IncumbentReceiptID] || !proofSet[artifact.CandidateReceiptID] {
		return "", errors.New("terminal benchstat proof does not bind its benchmark pair")
	}
	rows := qualificationRowsForRecordV1(contract, record)
	if len(rows) == 0 || len(artifact.Rows) != len(rows) {
		return "", errors.New("terminal benchmark rows differ from qualification contract")
	}
	oldSamples, err := parseBenchmarkSamplesV1(registry.contents[old.ReceiptID], rows)
	if err != nil {
		return "", err
	}
	newSamples, err := parseBenchmarkSamplesV1(registry.contents[candidate.ReceiptID], rows)
	if err != nil {
		return "", err
	}
	return evaluateQualificationOutcomeV1(contract, rows, oldSamples, newSamples, artifact.Rows)
}

func evaluateQualificationOutcomeV1(contract QualificationContractV1, rows []QualificationBenchmarkRowV1, oldSamples, newSamples map[string]benchmarkSamplesV1, comparisons []benchstatRowV1) (string, error) {
	if len(rows) == 0 || len(comparisons) != len(rows) {
		return "", errors.New("qualification comparison population mismatch")
	}
	inconclusive := false
	for i, row := range rows {
		oldRow, oldFound := oldSamples[row.BenchmarkName]
		newRow, newFound := newSamples[row.BenchmarkName]
		oldMedian, oldComplete := benchmarkMedianX2V1(oldRow)
		newMedian, newComplete := benchmarkMedianX2V1(newRow)
		comparison := comparisons[i]
		if !oldFound || !newFound || comparison.BenchmarkName != row.BenchmarkName ||
			comparison.OldMedianNanosMillionthsX2 != oldMedian ||
			comparison.NewMedianNanosMillionthsX2 != newMedian ||
			comparison.PValueMillionths < -1 || comparison.PValueMillionths > 1000000 {
			return "", errors.New("terminal benchmark comparison is stale")
		}
		if !oldRow.ZeroAllocs || !newRow.ZeroAllocs {
			return "fail", nil
		}
		if !oldComplete || !newComplete || comparison.PValueMillionths < 0 {
			inconclusive = true
			continue
		}
		protectedComparison, err := compareBenchmarkChangeV1(oldMedian, newMedian, -contract.MaximumProtectedSlowdownBPS)
		if err != nil {
			return "", err
		}
		if row.Protected && protectedComparison < 0 {
			return "fail", nil
		}
		slowdownComparison, err := compareBenchmarkChangeV1(oldMedian, newMedian, 0)
		if err != nil {
			return "", err
		}
		if contract.NoStatisticallySignificantSlowdown && slowdownComparison < 0 &&
			comparison.PValueMillionths <= contract.MaximumPValueMillionths {
			return "fail", nil
		}
		bulkComparison, err := compareBenchmarkChangeV1(oldMedian, newMedian, contract.MinimumBulkWinBPS)
		if err != nil {
			return "", err
		}
		if row.IntendedBulk && (bulkComparison < 0 ||
			comparison.PValueMillionths > contract.MaximumPValueMillionths) {
			return "fail", nil
		}
	}
	if inconclusive {
		return "inconclusive", nil
	}
	return "pass", nil
}
func legalRowTransition(a, b string) bool {
	return map[string]string{"snapshot_planned": "scalar_private", "scalar_private": "scalar_green", "scalar_green": "family_published", "family_published": "complete"}[a] == b
}
func legalCellTransition(a, b string) bool {
	return map[string]map[string]bool{"eligible": {"direct_built": true, "not_applicable": true}, "direct_built": {"hard_gates_green": true}, "hard_gates_green": {"dispatch_candidate": true}, "dispatch_candidate": {"selected": true, "direct_only": true}}[a][b]
}
func validateBindingTopology(b EvidenceBindingV1) error {
	if !safeEvidencePart(b.CommandID) || !safeEvidencePart(b.OutputID) || !evidenceKindsV1[b.Kind] || !safeEvidencePart(b.Provider) || (b.InitialState != "snapshot_planned" && b.InitialState != "eligible") {
		return errors.New("malformed frozen evidence binding")
	}
	if (b.NAReason == "") != (b.NASource == "") ||
		b.NAReason != "" && (!validNAReason(b.NAReason) || !validNAEvidenceSourceV1(b.NAReason, b.NASource)) {
		return errors.New("malformed frozen not-applicable binding")
	}
	return validateTypedEvidenceKey(EvidenceRecordV1{
		KeyKind: b.KeyKind, StorageID: b.StorageID, DisplayID: b.DisplayID, TupleHex: b.TupleHex,
		RowID: b.RowID, CellID: b.CellID, SymbolID: b.SymbolID, BatchID: b.BatchID,
		TransactionID: b.TransactionID, OperationID: b.OperationID, Backend: b.Backend,
		DirectSymbol: b.DirectSymbol, OrderedMembers: b.OrderedMembers,
	})
}

func validNAEvidenceSourceV1(reason, source string) bool {
	const prefix = "611becc2a08c27a4edc77d9a45ff74c97130129b:include/simdutf/implementation.h:"
	if !strings.HasPrefix(source, prefix) {
		return false
	}
	location := strings.TrimPrefix(source, prefix)
	dependency := ""
	if at := strings.IndexByte(location, ';'); at >= 0 {
		dependency = location[at+1:]
		location = location[:at]
	}
	line, err := strconv.Atoi(location)
	if err != nil || line <= 0 || strconv.Itoa(line) != location {
		return false
	}
	if reason == "composite_wrapper_delegates_accelerated_core" {
		if !strings.HasPrefix(dependency, "dependency:") {
			return false
		}
		dependency = strings.TrimPrefix(dependency, "dependency:")
		if dependency == "" {
			return false
		}
		for i, character := range dependency {
			if !(character == '_' || character >= 'a' && character <= 'z' ||
				character >= 'A' && character <= 'Z' ||
				i > 0 && character >= '0' && character <= '9') {
				return false
			}
		}
		return true
	}
	return dependency == ""
}

func validateTypedEvidenceKey(r EvidenceRecordV1) error {
	var k KeyRecord
	var err error
	switch r.KeyKind {
	case "row":
		f, e := DecodeTupleHexV1(r.TupleHex, 6)
		if e != nil {
			return e
		}
		k = KeyRecord{Kind: "row", StorageID: RowKeyV1([6]string{f[0], f[1], f[2], f[3], f[4], f[5]}), TupleHex: r.TupleHex}
		if r.RowID != k.StorageID || r.DisplayID != "" {
			return errors.New("row tuple differs from semantic fields")
		}
	case "cell":
		f, e := DecodeTupleHexV1(r.TupleHex, 2)
		if e != nil {
			return e
		}
		k, err = CellKeyV1(f[0], f[1])
		if r.RowID != f[0] || r.Backend != f[1] || r.CellID != k.StorageID {
			return errors.New("cell tuple differs from semantic fields")
		}
	case "symbol":
		f, e := DecodeTupleHexV1(r.TupleHex, 2)
		if e != nil {
			return e
		}
		k, err = SymbolKeyV1(f[0], f[1])
		if r.Backend != f[0] || r.DirectSymbol != f[1] || r.SymbolID != k.StorageID {
			return errors.New("symbol tuple differs from semantic fields")
		}
	case "batch":
		f, e := DecodeTupleHexV1(r.TupleHex, 3)
		if e != nil {
			return e
		}
		m, e := decodeNestedTuple(f[2])
		if e != nil {
			return e
		}
		k, err = BatchKeyV1(f[0], f[1], m)
		if r.DisplayID != f[1] || !sameStrings(r.OrderedMembers, m) || r.BatchID != k.StorageID {
			return errors.New("batch tuple differs from semantic fields")
		}
	case "transaction":
		f, e := DecodeTupleHexV1(r.TupleHex, 2)
		if e != nil {
			return e
		}
		m, e := decodeNestedTuple(f[1])
		if e != nil {
			return e
		}
		k, err = TransactionKeyV1(f[0], m)
		if r.DisplayID != f[0] || !sameStrings(r.OrderedMembers, m) || r.TransactionID != k.StorageID {
			return errors.New("transaction tuple differs from semantic fields")
		}
	default:
		return errors.New("invalid evidence key kind")
	}
	if err != nil || k.StorageID != r.StorageID || k.DisplayID != r.DisplayID || k.TupleHex != r.TupleHex {
		return errors.New("typed key does not recompute")
	}
	return nil
}

func sameProofContext(transition, proof EvidenceRecordV1) bool {
	if proof.RowID != transition.RowID || proof.CellID != transition.CellID ||
		proof.OperationID != transition.OperationID || proof.Backend != transition.Backend ||
		proof.DirectSymbol != transition.DirectSymbol || proof.AuthoritySHA256 != transition.AuthoritySHA256 ||
		proof.FamilyID != transition.FamilyID || proof.CampaignID != transition.CampaignID ||
		proof.LaneID != transition.LaneID || proof.Provider != transition.Provider ||
		proof.ProducerID != transition.ProducerID || proof.CommandDigest != transition.CommandDigest ||
		proof.IdentitySetDigest != transition.IdentitySetDigest ||
		proof.QualificationContractID != transition.QualificationContractID ||
		proof.QualificationContractDigest != transition.QualificationContractDigest ||
		proof.CorpusID != transition.CorpusID || proof.CorpusDigest != transition.CorpusDigest ||
		proof.HostReceiptID != transition.HostReceiptID || proof.HostReceiptDigest != transition.HostReceiptDigest {
		return false
	}
	if proof.Kind == "state-transition" {
		return proof.StorageID == transition.StorageID && proof.KeyKind == transition.KeyKind &&
			proof.SourceRole == transition.SourceRole && proof.SourceCommit == transition.SourceCommit &&
			proof.SourceTree == transition.SourceTree && proof.SourceParent == transition.SourceParent
	}
	if terminalProofKindV1(proof.Kind) {
		return validTerminalProofCommandV1(proof)
	}
	return proof.SourceRole == transition.SourceRole && proof.SourceCommit == transition.SourceCommit &&
		proof.SourceTree == transition.SourceTree && proof.SourceParent == transition.SourceParent &&
		proof.OriginCampaignID == transition.OriginCampaignID &&
		proof.OriginProducerID == transition.OriginProducerID &&
		proof.OriginSourceCommit == transition.OriginSourceCommit &&
		proof.OriginSourceTree == transition.OriginSourceTree &&
		proof.OriginSourceParent == transition.OriginSourceParent
}

func terminalProofKindV1(kind string) bool {
	switch kind {
	case "incumbent-benchmark", "candidate-benchmark", "benchstat", "provider-guard", "selector", "final-selector":
		return true
	}
	return false
}

func validTerminalProofCommandV1(proof EvidenceRecordV1) bool {
	switch proof.Kind {
	case "incumbent-benchmark":
		return proof.CommandAction == "go_benchmark" && proof.CommandRole == "old" && proof.SourceRole == "old"
	case "candidate-benchmark":
		return proof.CommandAction == "go_benchmark" && proof.CommandRole == "new" && proof.SourceRole == "new"
	case "benchstat":
		return proof.CommandAction == "benchstat" && proof.CommandRole == "new"
	case "provider-guard":
		return proof.CommandAction == "provider_guard" && proof.CommandRole == "direct"
	case "selector":
		return proof.CommandAction == "selector_test" && proof.CommandRole == "selector"
	case "final-selector":
		return proof.CommandAction == "final_selector_test" && proof.CommandRole == "selector"
	}
	return false
}

func requireTerminalSelectionProof(kinds map[string]bool) bool {
	for _, kind := range []string{"incumbent-benchmark", "candidate-benchmark", "benchstat", "provider-guard", "selector", "final-selector"} {
		if !kinds[kind] {
			return false
		}
	}
	return true
}

func requireDirectOnlyProofV1(kinds map[string]bool) bool {
	for _, kind := range []string{"incumbent-benchmark", "candidate-benchmark", "benchstat", "provider-guard", "selector"} {
		if !kinds[kind] {
			return false
		}
	}
	return true
}

func validOriginPath(v string) bool {
	return v != "" && v != "." && !strings.HasPrefix(v, "/") && !strings.ContainsAny(v, "%\\\r\n") && !strings.Contains(v, "..") && path.Clean(v) == v
}
func ReceiptIDV1(r EvidenceRecordV1) string {
	fields := []string{
		"evidence-receipt-v1", r.AuthoritySHA256, r.FamilyID, r.CampaignID, r.LaneID,
		r.Provider, r.ProducerID, r.SourceRole, r.SourceCommit, r.SourceTree, r.SourceParent,
		r.OriginCampaignID, r.OriginProducerID, r.OriginSourceCommit, r.OriginSourceTree,
		r.OriginSourceParent, r.Kind, r.CommandID, r.OutputID, r.CommandAction, r.CommandRole,
		r.KeyKind, r.StorageID, r.DisplayID, r.TupleHex, r.RowID, r.CellID, r.SymbolID,
		r.BatchID, r.TransactionID, r.OperationID, r.Backend, r.DirectSymbol, r.Path,
		r.OriginPath, r.MediaType, strconv.FormatInt(r.Size, 10), r.Digest, r.StateSubject,
		r.PrerequisiteState, r.CurrentState, r.Disposition, r.GoQualification, r.InitialState,
		r.NAReason, r.NASource, r.IdentityValue, r.IdentitySetDigest, r.CommandDigest,
		r.QualificationContractID, r.QualificationContractDigest, r.CorpusID, r.CorpusDigest,
		r.HostReceiptID, r.HostReceiptDigest, strconv.Itoa(r.CommandOrdinal),
	}
	fields = append(fields, "ordered-members", strconv.Itoa(len(r.OrderedMembers)))
	fields = append(fields, r.OrderedMembers...)
	fields = append(fields, "proof-receipts", strconv.Itoa(len(r.ProofReceiptIDs)))
	fields = append(fields, r.ProofReceiptIDs...)
	sum := sha256.Sum256(EncodeTupleV1(fields...))
	return "receipt-v1-" + hex.EncodeToString(sum[:])
}
func validRawEvidencePath(r EvidenceRecordV1, c EvidenceValidationContextV1) bool {
	if err := validateContext(c); err != nil || r.AuthoritySHA256 != c.AuthoritySHA256 {
		return false
	}
	return validRawEvidenceRecordPathV1(r)
}

func validRawEvidenceRecordPathV1(r EvidenceRecordV1) bool {
	if !lowerHex(r.AuthoritySHA256, 64) {
		return false
	}
	prefix := "raw/" + r.AuthoritySHA256[:12] + "/" + r.FamilyID + "/" + r.CampaignID + "/" + r.LaneID + "/" + r.ProducerID + "/" + r.Kind + "/" + r.StorageID + "/"
	name := strings.TrimPrefix(r.Path, prefix)
	dot := strings.LastIndexByte(name, '.')
	return strings.HasPrefix(r.Path, prefix) && dot > 0 && name[:dot] == r.Digest && safeEvidencePart(name[dot+1:]) && !strings.Contains(name, "/") && !strings.ContainsAny(r.Path, "%\\") && !strings.Contains(r.Path, "..") && path.Clean(r.Path) == r.Path
}
func safeEvidencePart(v string) bool {
	if v == "" || timestampLike(v) || strings.Contains(v, "latest") {
		return false
	}
	for i, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || i > 0 && (c == '.' || c == '_' || c == '-')) {
			return false
		}
	}
	return true
}
func timestampLike(v string) bool {
	if len(v) != 8 && len(v) != 10 && len(v) != 13 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func lowerHex(v string, n int) bool {
	if len(v) != n {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func decodeNestedTuple(v string) ([]string, error) {
	for n := 0; n <= len(v)/2+1; n++ {
		if f, e := DecodeTupleV1([]byte(v), n); e == nil {
			return f, nil
		}
	}
	return nil, errors.New("invalid nested tuple")
}
func sameStrings(a, b []string) bool {
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
func validateFrozenBinding(r EvidenceRecordV1, c EvidenceValidationContextV1) error {
	b, e := bindingFor(r, c)
	if e != nil {
		return e
	}
	return validateKeyAndBinding(r, b)
}
func validateEvidenceKey(r EvidenceRecordV1) error {
	return validateTypedEvidenceKey(r)
}
func validateBindingsAndState(r EvidenceRecordV1) error {
	return validateState(r, EvidenceBindingV1{NAReason: r.NAReason, NASource: r.NASource}, nil)
}
