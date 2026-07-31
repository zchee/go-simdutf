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
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// IngestEvidenceFileV1 validates and registers an evidence file from a
// descriptor-stable path, retaining bytes only through semantic validation.
func IngestEvidenceFileV1(rootPath string, record EvidenceRecordV1, context EvidenceValidationContextV1, registry *EvidenceRegistryV1) error {
	if registry == nil {
		return errors.New("evidence ingestion requires a semantic evidence registry")
	}
	// Context validation must precede all authority-derived path processing.
	if err := validateContext(context); err != nil {
		return err
	}
	if record.Size < 0 || record.Size > MaxRawEvidenceSizeV1 {
		return errors.New("evidence size is outside the raw artifact limit")
	}
	if err := validateEvidenceMetadataV1(record, context); err != nil {
		return err
	}

	rootInfo, err := os.Lstat(rootPath)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return errors.New("evidence root is not a real directory")
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return err
	}
	defer root.Close()
	openedRoot, err := root.Stat(".")
	if err != nil {
		return err
	}
	if !openedRoot.IsDir() || !os.SameFile(rootInfo, openedRoot) {
		return errors.New("evidence root changed while opening")
	}

	parts := strings.Split(record.Path, "/")
	if len(parts) < 2 {
		return errors.New("invalid evidence path")
	}
	current := root
	for _, component := range parts[:len(parts)-1] {
		if !safeDescriptorComponentV1(component) {
			return errors.New("invalid evidence directory component")
		}
		before, err := current.Lstat(component)
		if err != nil {
			return err
		}
		if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
			return errors.New("evidence path contains a non-directory or symlink")
		}
		child, err := current.OpenRoot(component)
		if err != nil {
			return err
		}
		after, statErr := child.Stat(".")
		if statErr != nil {
			child.Close()
			return statErr
		}
		if !after.IsDir() || !os.SameFile(before, after) {
			child.Close()
			return errors.New("evidence directory changed while opening")
		}
		if current != root {
			current.Close()
		}
		current = child
	}
	if current != root {
		defer current.Close()
	}

	base := parts[len(parts)-1]
	if !safeDescriptorComponentV1(base) {
		return errors.New("invalid evidence filename")
	}
	before, err := current.Lstat(base)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return errors.New("evidence path is not a regular file")
	}
	file, err := current.Open(base)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || opened.Size() != record.Size {
		return errors.New("evidence file changed while opening")
	}

	h := sha256.New()
	var contents bytes.Buffer
	writer := io.Writer(h)
	retainContents := requiresEvidenceContentsV1(record.Kind)
	if retainContents {
		if record.Size > MaxSemanticEvidenceSizeV1 {
			return errors.New("semantic evidence exceeds its bounded parsing limit")
		}
		writer = io.MultiWriter(h, &contents)
	}
	written, err := io.Copy(writer, io.LimitReader(file, record.Size))
	if err != nil {
		return err
	}
	var extra [1]byte
	n, err := file.Read(extra[:])
	if err != nil && err != io.EOF {
		return err
	}
	if written != record.Size || n != 0 {
		return errors.New("evidence file size does not match metadata")
	}
	final, err := file.Stat()
	if err != nil {
		return err
	}
	if !final.Mode().IsRegular() || !os.SameFile(opened, final) || final.Size() != record.Size {
		return errors.New("evidence file changed during read")
	}
	digest := fmt.Sprintf("%x", h.Sum(nil))
	if record.Digest != digest {
		return errors.New("evidence file digest does not match metadata")
	}
	return validateEvidenceStreamedV1(record, record.Size, digest, contents.Bytes(), retainContents, context, registry)
}

func safeDescriptorComponentV1(part string) bool {
	return part != "" && part != "." && part != ".." && !strings.ContainsAny(part, "/\\")
}

// RenderEvidenceRegistryIndexV1 renders the canonical global return index for
// every semantically registered receipt, including retained historical campaigns.
func RenderEvidenceRegistryIndexV1(registry *EvidenceRegistryV1) ([]byte, error) {
	if registry == nil || len(registry.receipts) == 0 {
		return nil, errors.New("evidence registry is empty")
	}
	records := make([]EvidenceRecordV1, 0, len(registry.receipts))
	for id, record := range registry.receipts {
		if id != record.ReceiptID || id != ReceiptIDV1(record) {
			return nil, errors.New("evidence registry contains an invalid receipt identity")
		}
		records = append(records, record)
	}
	return renderReturnIndexRecordsV1(records)
}

// RenderReturnIndexV1 renders the canonical complete receipt index for frozen
// required outputs. It is also the authoritative coverage check for validation.
func RenderReturnIndexV1(context EvidenceValidationContextV1, registry *EvidenceRegistryV1) ([]byte, error) {
	if err := validateContext(context); err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, errors.New("return index requires an evidence registry")
	}
	type requiredOutput struct {
		command CampaignCommandV1
		output  CommandOutputV1
	}
	required := make(map[string]requiredOutput)
	for _, command := range context.ExpectedCommands {
		for _, output := range command.Outputs {
			if !output.Required {
				continue
			}
			key := command.ID + "\x00" + output.ID
			if output.ID == "" || output.Kind == "" || output.Path == "" || output.MediaType == "" {
				return nil, errors.New("invalid required command output")
			}
			if _, duplicate := required[key]; duplicate {
				return nil, errors.New("duplicate required command output")
			}
			required[key] = requiredOutput{command, output}
		}
	}
	if len(required) == 0 {
		return nil, errors.New("return index requires at least one required output")
	}

	records := make([]EvidenceRecordV1, 0, len(required))
	for key, requiredOutput := range required {
		var (
			record EvidenceRecordV1
			found  bool
		)
		for _, candidate := range registry.receipts {
			if !recordMatchesCampaignContextV1(candidate, context) ||
				candidate.CommandID != requiredOutput.command.ID ||
				candidate.OutputID != requiredOutput.output.ID {
				continue
			}
			if found {
				return nil, fmt.Errorf("multiple registered receipts for required output %q", key)
			}
			record, found = candidate, true
		}
		if !found {
			return nil, fmt.Errorf("missing registered receipt for required output %q", key)
		}
		if err := ValidateEvidenceMetadataV1(record, record.Size, record.Digest, context, registry); err != nil {
			return nil, fmt.Errorf("invalid registered receipt: %w", err)
		}
		if record.CommandID != requiredOutput.command.ID || record.OutputID != requiredOutput.output.ID || record.OriginPath != requiredOutput.output.Path || record.Kind != requiredOutput.output.Kind || record.MediaType != requiredOutput.output.MediaType {
			return nil, fmt.Errorf("registered receipt does not match required output %q", key)
		}
		records = append(records, record)
	}
	// No registered receipt in this campaign may silently supply an optional or
	// unrequested output. Historical campaigns remain independently indexable.
	for _, record := range registry.receipts {
		if !recordMatchesCampaignContextV1(record, context) {
			continue
		}
		key := record.CommandID + "\x00" + record.OutputID
		if _, ok := required[key]; !ok {
			return nil, fmt.Errorf("registered receipt is not a required output %q", key)
		}
	}
	return renderReturnIndexRecordsV1(records)
}

func recordMatchesCampaignContextV1(record EvidenceRecordV1, context EvidenceValidationContextV1) bool {
	return record.AuthoritySHA256 == context.AuthoritySHA256 &&
		record.FamilyID == context.FamilyID &&
		record.CampaignID == context.CampaignID &&
		record.LaneID == context.LaneID &&
		record.ProducerID == context.ProducerID &&
		record.Provider == context.Provider &&
		record.CommandDigest == context.CommandManifestSHA256 &&
		record.IdentitySetDigest == context.IdentitySetDigest &&
		record.QualificationContractID == context.QualificationContractID &&
		record.QualificationContractDigest == context.QualificationContractDigest &&
		record.CorpusID == context.CorpusID &&
		record.CorpusDigest == context.CorpusDigest &&
		record.HostReceiptID == context.HostReceiptID &&
		record.HostReceiptDigest == context.HostReceiptDigest
}

func renderReturnIndexRecordsV1(records []EvidenceRecordV1) ([]byte, error) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].Path != records[j].Path {
			return records[i].Path < records[j].Path
		}
		return records[i].ReceiptID < records[j].ReceiptID
	})
	var out strings.Builder
	out.WriteString(returnIndexHeaderV1)
	for _, record := range records {
		fields := []string{
			record.Path,
			record.MediaType,
			strconv.FormatInt(record.Size, 10),
			record.Digest,
			record.ReceiptID,
			record.KeyKind,
			record.StorageID,
			record.DisplayID,
			record.TupleHex,
			record.CommandID,
			record.OutputID,
			record.ProducerID,
			record.CampaignID,
			record.RowID,
			record.CellID,
			record.Disposition,
		}
		for _, field := range fields {
			if strings.ContainsAny(field, "\t\r\n") {
				return nil, errors.New("return index field contains a delimiter")
			}
		}
		out.WriteString(strings.Join(fields, "\t"))
		out.WriteByte('\n')
	}
	return []byte(out.String()), nil
}
