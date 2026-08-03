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
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zchee/go-simdutf/internal/portplan"
)

var generatedNames = []string{
	"operations-v1.tsv",
	"cells-v1.tsv",
	"kernel-registry-v1.tsv",
	"classification-v1.tsv",
	"batches-v1.tsv",
	"frozen-inputs-v1.tsv",
	"authority-v1.tsv",
}

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "docs/porting/simdutf-port-v1/generated", "generated output directory")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected arguments: %s", strings.Join(flag.Args(), " ")))
	}
	if err := generate(*root, *out); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "simdutf-portplan:", err)
	os.Exit(1)
}

func generate(root, out string) error {
	if !filepath.IsAbs(out) {
		out = filepath.Join(root, out)
	}
	artifacts, err := render(root)
	if err != nil {
		return err
	}
	again, err := render(root)
	if err != nil {
		return err
	}
	if err := equalArtifacts(artifacts, again); err != nil {
		return fmt.Errorf("non-deterministic render: %w", err)
	}
	return publish(out, artifacts)
}

func render(root string) (map[string][]byte, error) {
	inputs := map[string][]byte{}
	read := func(name string) ([]byte, error) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		inputs[name] = data
		return data, nil
	}

	manifestData, err := read("docs/porting/api-manifest.tsv")
	if err != nil {
		return nil, err
	}
	manifest, err := portplan.ParseManifestV1(manifestData)
	if err != nil {
		return nil, err
	}
	plannedData, err := read("docs/porting/simdutf-port-v1/inputs/planned-rows-v1.tsv")
	if err != nil {
		return nil, err
	}
	planned, err := portplan.ParseManifestV1(plannedData)
	if err != nil {
		return nil, err
	}
	if len(planned) != 125 {
		return nil, errors.New("planned snapshot drift")
	}
	manifestKeys := make(map[string]bool, len(manifest))
	for _, row := range manifest {
		manifestKeys[row.RowKeyV1] = true
	}
	for _, row := range planned {
		if !manifestKeys[row.RowKeyV1] {
			return nil, errors.New("planned snapshot row is absent from live manifest")
		}
	}
	ledgerData, err := read("docs/porting/isa-eligibility.tsv")
	if err != nil {
		return nil, err
	}
	ledger, err := portplan.ParseISALedgerV1(ledgerData)
	if err != nil {
		return nil, err
	}

	fragmentNames := []string{
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-001-020.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-021-055.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-056-105.tsv",
		"docs/porting/simdutf-port-v1/inputs/review-fragments/rows-106-125.tsv",
	}
	fragments := make([][]byte, len(fragmentNames))
	for i, name := range fragmentNames {
		fragments[i], err = read(name)
		if err != nil {
			return nil, err
		}
	}
	reviewedData, err := joinReviewFragments(fragments)
	if err != nil {
		return nil, err
	}
	reviewed, err := portplan.ParseReviewedMappingsV1(reviewedData, planned, ledger)
	if err != nil {
		return nil, err
	}
	existingData, err := read("docs/porting/simdutf-port-v1/inputs/review-fragments/existing-members-v1.tsv")
	if err != nil {
		return nil, err
	}
	existing, err := portplan.ParseReviewedExistingMembersV1(existingData, manifest, ledger)
	if err != nil {
		return nil, err
	}
	dependencyData, err := read("docs/porting/simdutf-port-v1/inputs/dependency-map-v1.tsv")
	if err != nil {
		return nil, err
	}
	dependencies, err := portplan.ParseDependencyMapV1(dependencyData, planned, reviewed)
	if err != nil {
		return nil, err
	}
	golden, err := read("testdata/public-api.golden")
	if err != nil {
		return nil, err
	}
	for _, name := range []string{
		"docs/porting/provenance.md",
		"docs/porting/benchmark-contract.md",
		"docs/porting/corpus-freeze.md",
	} {
		data, readErr := read(name)
		if readErr != nil {
			return nil, readErr
		}
		if len(data) == 0 || data[len(data)-1] != '\n' || bytes.ContainsRune(data, '\r') {
			return nil, fmt.Errorf("%s is not canonical LF text", name)
		}
	}
	lockedData, err := read("docs/porting/simdutf-port-v1/inputs/locked-sets-v1.tsv")
	if err != nil {
		return nil, err
	}
	locked, err := portplan.ParseLockedSetsV1(lockedData, golden, planned)
	if err != nil {
		return nil, err
	}
	upstreamData, err := read("docs/porting/simdutf-port-v1/inputs/upstream-authority-v1.tsv")
	if err != nil {
		return nil, err
	}
	if _, err = portplan.ParseUpstreamAuthorityV1(upstreamData); err != nil {
		return nil, err
	}
	goBaseData, err := read("docs/porting/simdutf-port-v1/inputs/go-base-authority-v1.tsv")
	if err != nil {
		return nil, err
	}
	if _, err = portplan.ParseGoBaseAuthorityV1(goBaseData); err != nil {
		return nil, err
	}
	hostData, err := read("docs/porting/simdutf-port-v1/inputs/host-authority-v1.tsv")
	if err != nil {
		return nil, err
	}
	if _, err = portplan.ParseHostAuthorityV1(hostData, locked); err != nil {
		return nil, err
	}
	sourceData, err := read("docs/porting/simdutf-port-v1/inputs/archsimd-source-files-v1.tsv")
	if err != nil {
		return nil, err
	}
	sources, err := portplan.ParseArchsimdSourceFilesV1(sourceData)
	if err != nil {
		return nil, err
	}
	primitivesData, err := read("docs/porting/simdutf-port-v1/inputs/archsimd-primitives-v1.tsv")
	if err != nil {
		return nil, err
	}
	if _, err = portplan.ValidateArchsimdPrimitivesV1(primitivesData, sources, reviewed); err != nil {
		return nil, err
	}
	corpusData, err := read("docs/porting/simdutf-port-v1/inputs/corpus-contract-v1.tsv")
	if err != nil {
		return nil, err
	}
	corpora, err := portplan.ParseCorpusContractV1(corpusData)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{
		"docs/porting/simdutf-port-v1/inputs/evidence-schema-v1.json",
		"docs/porting/simdutf-port-v1/inputs/campaign-command-schema-v1.json",
		"docs/porting/simdutf-port-v1/inputs/qualification-contract-schema-v1.json",
		"docs/porting/simdutf-port-v1/inputs/completion-schema-v1.json",
	} {
		data, readErr := read(name)
		if readErr != nil {
			return nil, readErr
		}
		if err := validateSchema(data, name); err != nil {
			return nil, err
		}
	}
	qualificationPolicyData, err := read("docs/porting/simdutf-port-v1/inputs/qualification-policy-v1.tsv")
	if err != nil {
		return nil, err
	}
	qualificationPolicy, err := portplan.ParseQualificationPolicyV1(qualificationPolicyData, corpora)
	if err != nil {
		return nil, err
	}
	renderedQualificationPolicy, err := portplan.RenderQualificationPolicyV1(qualificationPolicy, corpora)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(qualificationPolicyData, renderedQualificationPolicy) {
		return nil, errors.New("qualification policy does not round-trip canonically")
	}

	ranks, err := portplan.BuildCanonicalRowRanksV1(planned, reviewed, ledger, dependencies)
	if err != nil {
		return nil, err
	}
	membership, err := portplan.BuildMembershipV1(reviewed, manifest, planned, ledger, existing, ranks)
	if err != nil {
		return nil, err
	}
	classification, err := portplan.BuildClassificationV1(planned, reviewed, dependencies, ranks, membership)
	if err != nil {
		return nil, err
	}
	for _, backend := range []string{"westmere", "haswell", "archsimd", "neon"} {
		eligible := 0
		for _, cell := range classification.Cells {
			if cell.Backend == backend && cell.BackendOutcome == "eligible" {
				eligible++
			}
		}
		if eligible != 79 {
			return nil, fmt.Errorf("%s eligible cell count is %d, want 79", backend, eligible)
		}
	}

	frozenInputs, _, err := portplan.RenderFrozenInputsV1(inputs)
	if err != nil {
		return nil, err
	}
	records, _, err := portplan.ParseFrozenInputsV1(frozenInputs, inputs)
	if err != nil {
		return nil, err
	}
	authority, err := renderAuthority(records)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		"operations-v1.tsv":      portplan.RenderOperationsV1(membership.Operations),
		"cells-v1.tsv":           portplan.RenderCellsV1(classification.Cells),
		"kernel-registry-v1.tsv": portplan.RenderKernelsV1(membership.Kernels),
		"classification-v1.tsv":  portplan.RenderClassificationV1(classification.Rows),
		"batches-v1.tsv":         portplan.RenderBatchesV1(classification.Batches),
		"frozen-inputs-v1.tsv":   frozenInputs,
		"authority-v1.tsv":       authority,
	}, nil
}

func joinReviewFragments(fragments [][]byte) ([]byte, error) {
	if len(fragments) != 4 {
		return nil, errors.New("review mapping requires four fragments")
	}
	var header string
	var rows []string
	for i, data := range fragments {
		if len(data) == 0 || data[len(data)-1] != '\n' || bytes.ContainsRune(data, '\r') {
			return nil, fmt.Errorf("review fragment %d is not canonical LF text", i+1)
		}
		lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
		if len(lines) < 2 || lines[0] == "" {
			return nil, fmt.Errorf("review fragment %d is empty", i+1)
		}
		if i == 0 {
			header = lines[0]
		} else if lines[0] != header {
			return nil, fmt.Errorf("review fragment %d header drift", i+1)
		}
		rows = append(rows, lines[1:]...)
	}
	return []byte(header + "\n" + strings.Join(rows, "\n") + "\n"), nil
}

func validateSchema(data []byte, name string) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var schema map[string]json.RawMessage
	if err := decoder.Decode(&schema); err != nil {
		return fmt.Errorf("%s: invalid JSON: %w", name, err)
	}
	if len(schema) == 0 {
		return fmt.Errorf("%s: empty schema", name)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: multiple JSON values", name)
	}
	var identifier, dialect, schemaType string
	var additional bool
	if err := json.Unmarshal(schema["$id"], &identifier); err != nil || identifier != filepath.Base(name) {
		return fmt.Errorf("%s: invalid schema identifier", name)
	}
	if err := json.Unmarshal(schema["$schema"], &dialect); err != nil || dialect != "https://json-schema.org/draft/2020-12/schema" {
		return fmt.Errorf("%s: invalid schema dialect", name)
	}
	if err := json.Unmarshal(schema["type"], &schemaType); err != nil {
		return fmt.Errorf("%s: invalid root type", name)
	}
	if schemaType == "array" {
		if filepath.Base(name) != "campaign-command-schema-v1.json" {
			return fmt.Errorf("%s: invalid root type", name)
		}
		if _, ok := schema["items"]; !ok {
			return fmt.Errorf("%s: array schema lacks items", name)
		}
		return nil
	}
	if schemaType != "object" {
		return fmt.Errorf("%s: invalid root type", name)
	}
	if err := json.Unmarshal(schema["additionalProperties"], &additional); err != nil || additional {
		return fmt.Errorf("%s: root must reject additional properties", name)
	}
	return nil
}

func renderAuthority(records []portplan.FrozenInputV1) ([]byte, error) {
	tuple, err := portplan.CanonicalFrozenAuthorityTupleV1(records)
	if err != nil {
		return nil, err
	}
	return []byte("schema_version\tauthority_tuple_hex\tauthority_sha256\n" + "v1\t" + hex.EncodeToString(tuple) + "\t" + portplan.SHA256Hex(tuple) + "\n"), nil
}

func publish(out string, artifacts map[string][]byte) error {
	if err := checkArtifacts(artifacts); err != nil {
		return err
	}
	parent := filepath.Dir(out)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(parent, ".simdutf-portplan-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	for _, name := range generatedNames {
		file := filepath.Join(temporary, name)
		if err := os.WriteFile(file, artifacts[name], 0o644); err != nil {
			return err
		}
		if err := syncFile(file); err != nil {
			return err
		}
	}
	if err := verifyDirectory(temporary, artifacts); err != nil {
		return err
	}
	backup := out + ".previous"
	if _, err := os.Lstat(backup); err == nil {
		return fmt.Errorf("refusing to replace existing backup %s", backup)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Lstat(out); err == nil {
		if err := os.Rename(out, backup); err != nil {
			return err
		}
		if err := os.Rename(temporary, out); err != nil {
			if rollbackErr := os.Rename(backup, out); rollbackErr != nil {
				return fmt.Errorf("publish failed: %w; rollback failed: %v", err, rollbackErr)
			}
			return err
		}
		if err := os.RemoveAll(backup); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(temporary, out); err != nil {
			return err
		}
	} else {
		return err
	}
	return verifyDirectory(out, artifacts)
}

func checkArtifacts(artifacts map[string][]byte) error {
	if len(artifacts) != len(generatedNames) {
		return errors.New("generated artifact set is incomplete")
	}
	for _, name := range generatedNames {
		if len(artifacts[name]) == 0 {
			return fmt.Errorf("generated artifact %s is empty", name)
		}
	}
	return nil
}

func verifyDirectory(directory string, artifacts map[string][]byte) error {
	for _, name := range generatedNames {
		data, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		if !bytes.Equal(data, artifacts[name]) {
			return fmt.Errorf("written artifact %s differs from rendered bytes", name)
		}
	}
	return nil
}

func syncFile(name string) error {
	// os.Open yields a read-only handle. On Windows, (*File).Sync calls
	// FlushFileBuffers, which requires GENERIC_WRITE access and fails with
	// "Access is denied." otherwise. POSIX fsync(2) has no such requirement,
	// which is why this only ever failed on windows-latest.
	file, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func equalArtifacts(a, b map[string][]byte) error {
	if err := checkArtifacts(a); err != nil {
		return err
	}
	if err := checkArtifacts(b); err != nil {
		return err
	}
	for _, name := range generatedNames {
		if !bytes.Equal(a[name], b[name]) {
			return fmt.Errorf("%s differs", name)
		}
	}
	return nil
}
