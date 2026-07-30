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

package simdutf

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/zchee/go-simdutf/internal/portplan"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPublicAPIContract(t *testing.T) {
	got := publicAPIRecords(t)

	counts := make(map[string]int)
	for _, record := range got {
		kind, _, ok := strings.Cut(record, "\t")
		if !ok {
			t.Fatalf("API record has no kind separator: %q", record)
		}
		counts[kind]++
	}
	wantCounts := map[string]int{
		"const":      2,
		"enum-const": 31,
		"field":      6,
		"func":       39,
		"method":     3,
		"type":       6,
	}
	if !maps.Equal(counts, wantCounts) {
		t.Fatalf("API leaf counts = %v, want %v", counts, wantCounts)
	}
	if len(got) != 87 {
		t.Fatalf("API leaf record count = %d, want 87", len(got))
	}

	want, err := os.ReadFile(filepath.Join("testdata", "public-api.golden"))
	if err != nil {
		t.Fatal(err)
	}
	gotBytes := []byte(strings.Join(got, "\n") + "\n")
	if !bytes.Equal(gotBytes, want) {
		t.Fatalf("public API changed (-want +got):\n%s", lineDiff(string(want), string(gotBytes)))
	}
}

func publicAPIRecords(t *testing.T) []string {
	t.Helper()

	buildPackage, err := build.Default.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	sourceNames := append(slices.Clone(buildPackage.GoFiles), buildPackage.CgoFiles...)
	slices.Sort(sourceNames)

	fset := token.NewFileSet()
	files := make([]*ast.File, 0, len(sourceNames))
	for _, name := range sourceNames {
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatal("package has no buildable Go source files")
	}

	config := types.Config{Importer: importer.Default()}
	checked, err := config.Check(buildPackage.ImportPath, fset, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	qualifier := func(other *types.Package) string {
		if other == checked {
			return ""
		}
		return other.Name()
	}

	var records []string
	for _, name := range checked.Scope().Names() {
		object := checked.Scope().Lookup(name)
		if !object.Exported() {
			continue
		}
		switch object := object.(type) {
		case *types.Const:
			typeName := types.TypeString(object.Type(), qualifier)
			kind := "const"
			if _, ok := object.Type().(*types.Named); ok {
				kind = "enum-const"
			}
			records = append(records, fmt.Sprintf("%s\t%s\t%s\t%s", kind, name, typeName, object.Val().ExactString()))
		case *types.Func:
			records = append(records, fmt.Sprintf("func\t%s\t%s", name, canonicalSignature(object.Type().(*types.Signature), qualifier)))
		case *types.TypeName:
			named, ok := object.Type().(*types.Named)
			if !ok {
				records = append(records, fmt.Sprintf("type\t%s\talias %s", name, types.TypeString(object.Type(), qualifier)))
				continue
			}
			underlying := named.Underlying()
			typeDescription := types.TypeString(underlying, qualifier)
			if structure, ok := underlying.(*types.Struct); ok {
				typeDescription = "struct"
				for fieldIndex := range structure.NumFields() {
					field := structure.Field(fieldIndex)
					if !field.Exported() {
						continue
					}
					records = append(records, fmt.Sprintf(
						"field\t%s.%s\t%d\t%s\tembedded=%t\ttag=%q",
						name,
						field.Name(),
						fieldIndex,
						types.TypeString(field.Type(), qualifier),
						field.Embedded(),
						structure.Tag(fieldIndex),
					))
				}
			}
			records = append(records, fmt.Sprintf("type\t%s\t%s", name, typeDescription))
			for methodIndex := range named.NumMethods() {
				method := named.Method(methodIndex)
				if !method.Exported() {
					continue
				}
				signature := method.Type().(*types.Signature)
				records = append(records, fmt.Sprintf(
					"method\t%s.%s\t%s\t%s",
					name,
					method.Name(),
					types.TypeString(signature.Recv().Type(), qualifier),
					canonicalSignature(signature, qualifier),
				))
			}
		case *types.Var:
			records = append(records, fmt.Sprintf("var\t%s\t%s", name, types.TypeString(object.Type(), qualifier)))
		default:
			t.Fatalf("unsupported exported object %s (%T)", name, object)
		}
	}
	slices.Sort(records)
	return records
}

func canonicalSignature(signature *types.Signature, qualifier types.Qualifier) string {
	withoutNames := func(tuple *types.Tuple) *types.Tuple {
		variables := make([]*types.Var, tuple.Len())
		for index := range tuple.Len() {
			variables[index] = types.NewVar(token.NoPos, nil, "", tuple.At(index).Type())
		}
		return types.NewTuple(variables...)
	}
	typeParameters := func(list *types.TypeParamList) []*types.TypeParam {
		parameters := make([]*types.TypeParam, list.Len())
		for index := range list.Len() {
			parameters[index] = list.At(index)
		}
		return parameters
	}
	canonical := types.NewSignatureType(
		signature.Recv(),
		typeParameters(signature.RecvTypeParams()),
		typeParameters(signature.TypeParams()),
		withoutNames(signature.Params()),
		withoutNames(signature.Results()),
		signature.Variadic(),
	)
	return types.TypeString(canonical, qualifier)
}

func TestAPIManifestMilestones(t *testing.T) {
	file, err := os.Open(filepath.Join("docs", "porting", "api-manifest.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 165 {
		t.Fatalf("manifest row count = %d, want 165 including header", len(rows))
	}

	columns := make(map[string]int, len(rows[0]))
	for index, name := range rows[0] {
		columns[name] = index
	}
	for _, name := range []string{"family", "go_symbol", "status", "milestone"} {
		if _, ok := columns[name]; !ok {
			t.Fatalf("manifest has no %q column", name)
		}
	}
	if columns["milestone"] != columns["status"]+1 {
		t.Fatalf("milestone column index = %d, want immediately after status at %d", columns["milestone"], columns["status"]+1)
	}

	statusCounts := make(map[string]int)
	milestoneCounts := make(map[string]int)
	familyCounts := make(map[string]int)
	var currentSymbols []string
	for rowIndex, row := range rows[1:] {
		if len(row) != len(rows[0]) {
			t.Fatalf("manifest row %d has %d columns, want %d", rowIndex+2, len(row), len(rows[0]))
		}
		status := row[columns["status"]]
		milestone := row[columns["milestone"]]
		wantMilestone := map[string]string{
			"implemented": "611becc-current-api",
			"planned":     "future-upstream-api",
			"excluded":    "upstream-excluded",
		}[status]
		if wantMilestone == "" {
			t.Fatalf("manifest row %d has unknown status %q", rowIndex+2, status)
		}
		if milestone != wantMilestone {
			t.Fatalf("manifest row %d milestone = %q, want %q for status %q", rowIndex+2, milestone, wantMilestone, status)
		}
		statusCounts[status]++
		milestoneCounts[milestone]++
		familyCounts[row[columns["family"]]]++
		if milestone == "611becc-current-api" {
			currentSymbols = append(currentSymbols, row[columns["go_symbol"]])
		}
	}

	assertStringIntMap(t, "status counts", statusCounts, map[string]int{
		"excluded":    9,
		"implemented": 50,
		"planned":     105,
	})
	assertStringIntMap(t, "milestone counts", milestoneCounts, map[string]int{
		"611becc-current-api": 50,
		"future-upstream-api": 105,
		"upstream-excluded":   9,
	})
	assertStringIntMap(t, "family counts", familyCounts, map[string]int{
		"ASCII":              5,
		"Base64":             29,
		"C++ mechanics":      3,
		"Encoding detection": 9,
		"Find":               2,
		"Result/error":       7,
		"Shared helper":      1,
		"Transcoding/length": 86,
		"UTF-16":             16,
		"UTF-32":             2,
		"UTF-8":              4,
	})

	slices.Sort(currentSymbols)
	wantCurrentSymbols := []string{
		"BOMByteSize",
		"Base64Options",
		"Base64OptionsString",
		"Base64ReversePadding",
		"CheckBOM",
		"ConvertLatin1ToUTF16",
		"ConvertLatin1ToUTF16BE",
		"ConvertLatin1ToUTF16LE",
		"ConvertLatin1ToUTF32",
		"ConvertLatin1ToUTF8",
		"ConvertLatin1ToUTF8Safe",
		"CountUTF8",
		"DefaultLineLength",
		"Encoding",
		"EncodingString",
		"ErrorCode",
		"ErrorToString",
		"FullResult",
		"FullResult.Result",
		"IsPartial",
		"LastChunkHandlingOptions",
		"LastChunkHandlingOptionsString",
		"Latin1LengthFromUTF8",
		"Result",
		"Result.IsErr",
		"Result.IsOK",
		"ToWellFormedUTF16",
		"ToWellFormedUTF16BE",
		"ToWellFormedUTF16LE",
		"TrimPartialUTF8",
		"UTF16LengthFromLatin1",
		"UTF16LengthFromUTF8",
		"UTF32LengthFromLatin1",
		"UTF32LengthFromUTF8",
		"UTF8LengthFromLatin1",
		"ValidateASCII",
		"ValidateASCIIWithErrors",
		"ValidateUTF16",
		"ValidateUTF16AsASCII",
		"ValidateUTF16BE",
		"ValidateUTF16BEAsASCII",
		"ValidateUTF16BEWithErrors",
		"ValidateUTF16LE",
		"ValidateUTF16LEAsASCII",
		"ValidateUTF16LEWithErrors",
		"ValidateUTF16WithErrors",
		"ValidateUTF32",
		"ValidateUTF32WithErrors",
		"ValidateUTF8",
		"ValidateUTF8WithErrors",
	}
	if !slices.Equal(currentSymbols, wantCurrentSymbols) {
		t.Fatalf("current manifest symbols = %v, want %v", currentSymbols, wantCurrentSymbols)
	}
}

func TestPortPhase0FrozenInputs(t *testing.T) {
	manifestPath := filepath.Join("docs", "porting", "api-manifest.tsv")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := portplan.SHA256Hex(manifest), "7379c49a42309fc3b695bdc93ec3f6fda5f20b3f10b6f034828377ac8dc25ef2"; got != want {
		t.Fatalf("%s SHA-256 = %s, want %s", manifestPath, got, want)
	}
	allRows, err := portplan.ParseManifestV1(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(allRows), 164; got != want {
		t.Fatalf("manifest data rows = %d, want %d", got, want)
	}
	_, livePlannedRows, err := portplan.FreezePlannedRowsV1(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(livePlannedRows), 105; got != want {
		t.Fatalf("remaining planned rows = %d, want %d", got, want)
	}
	frozenPath := filepath.Join("docs", "porting", "simdutf-port-v1", "inputs", "planned-rows-v1.tsv")
	wantFrozen, err := os.ReadFile(frozenPath)
	if err != nil {
		t.Fatal(err)
	}
	frozenRows, err := portplan.ParseManifestV1(wantFrozen)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(frozenRows), 125; got != want {
		t.Fatalf("frozen initial planned rows = %d, want %d", got, want)
	}
	frozenKeys := make(map[string]bool, len(frozenRows))
	for _, row := range frozenRows {
		frozenKeys[row.RowKeyV1] = true
	}
	for _, row := range livePlannedRows {
		if !frozenKeys[row.RowKeyV1] {
			t.Fatalf("remaining planned row %s is absent from the frozen snapshot", row.RowKeyV1)
		}
	}

	isaPath := filepath.Join("docs", "porting", "isa-eligibility.tsv")
	isaLedger, err := os.ReadFile(isaPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := portplan.SHA256Hex(isaLedger), "d73b6bc48466dba23d28c2d06c877ccbeaa10ab5049d8fb6b314d651a46549cb"; got != want {
		t.Fatalf("%s SHA-256 = %s, want %s", isaPath, got, want)
	}
	isaRows, err := portplan.ParseISALedgerV1(isaLedger)
	if err != nil {
		t.Fatal(err)
	}
	operationIDs := make(map[string]struct{}, len(isaRows))
	for _, row := range isaRows {
		id, err := portplan.LedgerOperationIDV1(row.LedgerOrdinal, row.SemanticOperation)
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := operationIDs[id]; exists {
			t.Fatalf("duplicate ISA semantic operation ID %s", id)
		}
		operationIDs[id] = struct{}{}
	}
}

func TestPortPhase0ReviewedMembership(t *testing.T) {
	read := func(parts ...string) []byte {
		t.Helper()
		path := filepath.Join(parts...)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}

	manifest := read("docs", "porting", "api-manifest.tsv")
	allRows, err := portplan.ParseManifestV1(manifest)
	if err != nil {
		t.Fatal(err)
	}
	plannedRows, err := portplan.ParseManifestV1(
		read("docs", "porting", "simdutf-port-v1", "inputs", "planned-rows-v1.tsv"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := portplan.ParseISALedgerV1(read("docs", "porting", "isa-eligibility.tsv"))
	if err != nil {
		t.Fatal(err)
	}

	var reviewed bytes.Buffer
	for index, name := range []string{
		"rows-001-020.tsv",
		"rows-021-055.tsv",
		"rows-056-105.tsv",
		"rows-106-125.tsv",
	} {
		data := read("docs", "porting", "simdutf-port-v1", "inputs", "review-fragments", name)
		if index != 0 {
			newline := bytes.IndexByte(data, '\n')
			if newline < 0 {
				t.Fatalf("%s has no header terminator", name)
			}
			data = data[newline+1:]
		}
		reviewed.Write(data)
	}
	mappings, err := portplan.ParseReviewedMappingsV1(reviewed.Bytes(), plannedRows, ledger)
	if err != nil {
		t.Fatal(err)
	}
	existing, err := portplan.ParseReviewedExistingMembersV1(
		read("docs", "porting", "simdutf-port-v1", "inputs", "review-fragments", "existing-members-v1.tsv"),
		allRows,
		ledger,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(existing), 30; got != want {
		t.Fatalf("existing membership rows = %d, want %d", got, want)
	}

	wantFamilies := map[string]int{
		"FC-v1-helper-validation": 11,
		"FC-v1-latin1-source":     9,
		"FC-v1-utf8-source":       15,
		"FC-v1-utf16-source":      48,
		"FC-v1-utf32-source":      18,
		"FC-v1-detection":         2,
		"FC-v1-find":              2,
		"FC-v1-base64":            20,
	}
	families := make(map[string]int, len(wantFamilies))
	eligible := map[string]int{"westmere": 0, "haswell": 0, "archsimd": 0, "neon": 0}
	directSymbols := map[string]map[string]struct{}{
		"westmere": {}, "haswell": {}, "archsimd": {}, "neon": {},
	}
	scalarRows := 0
	for _, mapping := range mappings {
		families[mapping.FamilyContractDisplayID]++
		if mapping.ISAOrdinalOrScalar == "scalar" {
			scalarRows++
		}
		for index, backend := range []string{"westmere", "haswell", "archsimd", "neon"} {
			cell := mapping.Backends[index]
			if cell.Outcome != "eligible" {
				continue
			}
			eligible[backend]++
			if _, duplicate := directSymbols[backend][cell.DirectSymbol]; duplicate {
				t.Fatalf("duplicate %s direct symbol %q", backend, cell.DirectSymbol)
			}
			directSymbols[backend][cell.DirectSymbol] = struct{}{}
		}
	}
	assertStringIntMap(t, "reviewed family counts", families, wantFamilies)
	assertStringIntMap(t, "reviewed eligible backend counts", eligible, map[string]int{
		"westmere": 79,
		"haswell":  79,
		"archsimd": 79,
		"neon":     79,
	})
	if scalarRows != 18 {
		t.Fatalf("reviewed scalar rows = %d, want 18", scalarRows)
	}
	dependencies, err := portplan.ParseDependencyMapV1(
		read("docs", "porting", "simdutf-port-v1", "inputs", "dependency-map-v1.tsv"),
		plannedRows,
		mappings,
	)
	if err != nil {
		t.Fatal(err)
	}
	lockedSets, err := portplan.ParseLockedSetsV1(
		read("docs", "porting", "simdutf-port-v1", "inputs", "locked-sets-v1.tsv"),
		read("testdata", "public-api.golden"),
		plannedRows,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(lockedSets), 8; got != want {
		t.Fatalf("locked sets = %d, want %d", got, want)
	}
	ranks, err := portplan.BuildCanonicalRowRanksV1(plannedRows, mappings, ledger, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	membership, err := portplan.BuildMembershipV1(mappings, allRows, plannedRows, ledger, existing, ranks)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(membership.Operations), 23; got != want {
		t.Fatalf("membership operations = %d, want %d", got, want)
	}
	if got, want := len(membership.Cells), 500; got != want {
		t.Fatalf("membership cells = %d, want %d", got, want)
	}
	classification, err := portplan.BuildClassificationV1(plannedRows, mappings, dependencies, ranks, membership)
	if err != nil {
		t.Fatal(err)
	}
	secondMembership, err := portplan.BuildMembershipV1(mappings, allRows, plannedRows, ledger, existing, ranks)
	if err != nil {
		t.Fatal(err)
	}
	secondClassification, err := portplan.BuildClassificationV1(plannedRows, mappings, dependencies, ranks, secondMembership)
	if err != nil {
		t.Fatal(err)
	}
	for name, pair := range map[string][2][]byte{
		"operations":     {portplan.RenderOperationsV1(membership.Operations), portplan.RenderOperationsV1(secondMembership.Operations)},
		"cells":          {portplan.RenderCellsV1(classification.Cells), portplan.RenderCellsV1(secondClassification.Cells)},
		"kernels":        {portplan.RenderKernelsV1(membership.Kernels), portplan.RenderKernelsV1(secondMembership.Kernels)},
		"classification": {portplan.RenderClassificationV1(classification.Rows), portplan.RenderClassificationV1(secondClassification.Rows)},
		"batches":        {portplan.RenderBatchesV1(classification.Batches), portplan.RenderBatchesV1(secondClassification.Batches)},
	} {
		if !bytes.Equal(pair[0], pair[1]) {
			t.Fatalf("%s generation is not byte-identical", name)
		}
	}
}

func assertStringIntMap(t *testing.T, name string, got, want map[string]int) {
	t.Helper()
	if !maps.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func lineDiff(want, got string) string {
	wantLines := strings.Split(strings.TrimSuffix(want, "\n"), "\n")
	gotLines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	var diff strings.Builder
	for index := range max(len(wantLines), len(gotLines)) {
		if index < len(wantLines) && index < len(gotLines) && wantLines[index] == gotLines[index] {
			continue
		}
		if index < len(wantLines) {
			fmt.Fprintf(&diff, "-%s\n", wantLines[index])
		}
		if index < len(gotLines) {
			fmt.Fprintf(&diff, "+%s\n", gotLines[index])
		}
	}
	return diff.String()
}
