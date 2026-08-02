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
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zchee/go-simdutf/internal/portplan"
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
		"func":       144,
		"method":     3,
		"type":       6,
	}
	if !maps.Equal(counts, wantCounts) {
		t.Fatalf("API leaf counts = %v, want %v", counts, wantCounts)
	}
	if len(got) != 192 {
		t.Fatalf("API leaf record count = %d, want 192", len(got))
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
			for method := range named.Methods() {
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
		"implemented": 155,
	})
	assertStringIntMap(t, "milestone counts", milestoneCounts, map[string]int{
		"611becc-current-api": 155,
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
		"AutodetectEncoding",
		"BOMByteSize",
		"Base64Ignorable",
		"Base64IgnorableUTF16",
		"Base64LengthFromBinary",
		"Base64LengthFromBinaryWithLines",
		"Base64Options",
		"Base64OptionsString",
		"Base64ReversePadding",
		"Base64ToBinary",
		"Base64ToBinaryDetails",
		"Base64ToBinaryDetailsUTF16",
		"Base64ToBinarySafe",
		"Base64ToBinarySafeUTF16",
		"Base64ToBinaryUTF16",
		"Base64Valid",
		"Base64ValidOrPadding",
		"Base64ValidOrPaddingUTF16",
		"Base64ValidUTF16",
		"BinaryLengthFromBase64",
		"BinaryLengthFromBase64UTF16",
		"BinaryToBase64",
		"BinaryToBase64WithLines",
		"ChangeEndiannessUTF16",
		"CheckBOM",
		"ConvertLatin1ToUTF16",
		"ConvertLatin1ToUTF16BE",
		"ConvertLatin1ToUTF16LE",
		"ConvertLatin1ToUTF32",
		"ConvertLatin1ToUTF8",
		"ConvertLatin1ToUTF8Safe",
		"ConvertUTF16BEToLatin1",
		"ConvertUTF16BEToLatin1WithErrors",
		"ConvertUTF16BEToUTF32",
		"ConvertUTF16BEToUTF32WithErrors",
		"ConvertUTF16BEToUTF8",
		"ConvertUTF16BEToUTF8WithErrors",
		"ConvertUTF16BEToUTF8WithReplacement",
		"ConvertUTF16LEToLatin1",
		"ConvertUTF16LEToLatin1WithErrors",
		"ConvertUTF16LEToUTF32",
		"ConvertUTF16LEToUTF32WithErrors",
		"ConvertUTF16LEToUTF8",
		"ConvertUTF16LEToUTF8WithErrors",
		"ConvertUTF16LEToUTF8WithReplacement",
		"ConvertUTF16ToLatin1",
		"ConvertUTF16ToLatin1WithErrors",
		"ConvertUTF16ToUTF32",
		"ConvertUTF16ToUTF32WithErrors",
		"ConvertUTF16ToUTF8",
		"ConvertUTF16ToUTF8Safe",
		"ConvertUTF16ToUTF8WithErrors",
		"ConvertUTF16ToUTF8WithReplacement",
		"ConvertUTF32ToLatin1",
		"ConvertUTF32ToLatin1WithErrors",
		"ConvertUTF32ToUTF16",
		"ConvertUTF32ToUTF16BE",
		"ConvertUTF32ToUTF16BEWithErrors",
		"ConvertUTF32ToUTF16LE",
		"ConvertUTF32ToUTF16LEWithErrors",
		"ConvertUTF32ToUTF16WithErrors",
		"ConvertUTF32ToUTF8",
		"ConvertUTF32ToUTF8WithErrors",
		"ConvertUTF8ToLatin1",
		"ConvertUTF8ToLatin1WithErrors",
		"ConvertUTF8ToUTF16",
		"ConvertUTF8ToUTF16BE",
		"ConvertUTF8ToUTF16BEWithErrors",
		"ConvertUTF8ToUTF16LE",
		"ConvertUTF8ToUTF16LEWithErrors",
		"ConvertUTF8ToUTF16WithErrors",
		"ConvertUTF8ToUTF32",
		"ConvertUTF8ToUTF32WithErrors",
		"ConvertValidUTF16BEToLatin1",
		"ConvertValidUTF16BEToUTF32",
		"ConvertValidUTF16BEToUTF8",
		"ConvertValidUTF16LEToLatin1",
		"ConvertValidUTF16LEToUTF32",
		"ConvertValidUTF16LEToUTF8",
		"ConvertValidUTF16ToLatin1",
		"ConvertValidUTF16ToUTF32",
		"ConvertValidUTF16ToUTF8",
		"ConvertValidUTF32ToLatin1",
		"ConvertValidUTF32ToUTF16",
		"ConvertValidUTF32ToUTF16BE",
		"ConvertValidUTF32ToUTF16LE",
		"ConvertValidUTF32ToUTF8",
		"ConvertValidUTF8ToLatin1",
		"ConvertValidUTF8ToUTF16",
		"ConvertValidUTF8ToUTF16BE",
		"ConvertValidUTF8ToUTF16LE",
		"ConvertValidUTF8ToUTF32",
		"CountUTF16",
		"CountUTF16BE",
		"CountUTF16LE",
		"CountUTF8",
		"DefaultLineLength",
		"DetectEncodings",
		"Encoding",
		"EncodingString",
		"ErrorCode",
		"ErrorToString",
		"Find",
		"FindUTF16",
		"FullResult",
		"FullResult.Result",
		"IsPartial",
		"LastChunkHandlingOptions",
		"LastChunkHandlingOptionsString",
		"Latin1LengthFromUTF16",
		"Latin1LengthFromUTF32",
		"Latin1LengthFromUTF8",
		"MaximalBinaryLengthFromBase64",
		"MaximalBinaryLengthFromBase64UTF16",
		"Result",
		"Result.IsErr",
		"Result.IsOK",
		"ToWellFormedUTF16",
		"ToWellFormedUTF16BE",
		"ToWellFormedUTF16LE",
		"TrimPartialUTF16",
		"TrimPartialUTF16BE",
		"TrimPartialUTF16LE",
		"TrimPartialUTF8",
		"UTF16LengthFromLatin1",
		"UTF16LengthFromUTF32",
		"UTF16LengthFromUTF8",
		"UTF32LengthFromLatin1",
		"UTF32LengthFromUTF16",
		"UTF32LengthFromUTF16BE",
		"UTF32LengthFromUTF16LE",
		"UTF32LengthFromUTF8",
		"UTF8LengthFromLatin1",
		"UTF8LengthFromUTF16",
		"UTF8LengthFromUTF16BE",
		"UTF8LengthFromUTF16BEWithReplacement",
		"UTF8LengthFromUTF16LE",
		"UTF8LengthFromUTF16LEWithReplacement",
		"UTF8LengthFromUTF16WithReplacement",
		"UTF8LengthFromUTF32",
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
	if got, want := portplan.SHA256Hex(manifest), "eda31c7e6865c5c5ae34dee526d5f1bb5247e2f3b0a1017b9f26120038a271a0"; got != want {
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
	if got, want := len(livePlannedRows), 0; got != want {
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

// Contract cases derived from simdutf commit 611becc2a08c27a4edc77d9a45ff74c97130129b,
// include/simdutf/error.h:7-124.
// Narrow Go-only scaffolding covers the underlying type, unknown values, and
// Go zero values; these are not upstream test vectors.

func TestErrorCode(t *testing.T) {
	if kind := reflect.TypeOf(Success).Kind(); kind != reflect.Uint8 {
		t.Fatalf("ErrorCode underlying kind = %v, want uint8", kind)
	}

	tests := []struct {
		name string
		code ErrorCode
		want uint8
		text string
	}{
		{"success", Success, 0, "SUCCESS"},
		{"header bits", HeaderBits, 1, "HEADER_BITS"},
		{"too short", TooShort, 2, "TOO_SHORT"},
		{"too long", TooLong, 3, "TOO_LONG"},
		{"overlong", Overlong, 4, "OVERLONG"},
		{"too large", TooLarge, 5, "TOO_LARGE"},
		{"surrogate", Surrogate, 6, "SURROGATE"},
		{"invalid base64 character", InvalidBase64Character, 7, "INVALID_BASE64_CHARACTER"},
		{"base64 input remainder", Base64InputRemainder, 8, "BASE64_INPUT_REMAINDER"},
		{"base64 extra bits", Base64ExtraBits, 9, "BASE64_EXTRA_BITS"},
		{"output buffer too small", OutputBufferTooSmall, 10, "OUTPUT_BUFFER_TOO_SMALL"},
		{"other", Other, 11, "OTHER"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint8(test.code); got != test.want {
				t.Errorf("value = %d, want %d", got, test.want)
			}
			if got := ErrorToString(test.code); got != test.text {
				t.Errorf("ErrorToString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestErrorToStringUnknown(t *testing.T) {
	for _, code := range []ErrorCode{12, 255} {
		if got := ErrorToString(code); got != "OTHER" {
			t.Errorf("ErrorToString(%d) = %q, want OTHER", code, got)
		}
	}
}

func TestResultStatus(t *testing.T) {
	tests := []struct {
		name    string
		result  Result
		wantOK  bool
		wantErr bool
	}{
		{"zero value", Result{}, true, false},
		{"success", Result{Error: Success, Count: 9}, true, false},
		{"error", Result{Error: TooShort, Count: 3}, false, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.result.IsOK(); got != test.wantOK {
				t.Errorf("IsOK() = %v, want %v", got, test.wantOK)
			}
			if got := test.result.IsErr(); got != test.wantErr {
				t.Errorf("IsErr() = %v, want %v", got, test.wantErr)
			}
		})
	}
}

func TestFullResultResult(t *testing.T) {
	tests := []struct {
		name string
		full FullResult
		want Result
	}{
		{"zero value", FullResult{}, Result{}},
		{
			"success uses output count",
			FullResult{Error: Success, InputCount: 12, OutputCount: 7, PaddingError: true},
			Result{Error: Success, Count: 7},
		},
		{
			"error uses input count",
			FullResult{Error: InvalidBase64Character, InputCount: 5, OutputCount: 3, PaddingError: true},
			Result{Error: InvalidBase64Character, Count: 5},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.full.Result(); got != test.want {
				t.Errorf("Result() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestFullResultZeroValue(t *testing.T) {
	var result FullResult
	if result.Error != Success {
		t.Errorf("Error = %d, want Success", result.Error)
	}
	if result.InputCount != 0 {
		t.Errorf("InputCount = %d, want 0", result.InputCount)
	}
	if result.OutputCount != 0 {
		t.Errorf("OutputCount = %d, want 0", result.OutputCount)
	}
	if result.PaddingError {
		t.Error("PaddingError = true, want false")
	}
}

// Contract cases derived from simdutf commit 611becc2a08c27a4edc77d9a45ff74c97130129b,
// include/simdutf/implementation.h:187-188,4094-4138,4194-4228.
// Narrow Go-only scaffolding covers underlying types, unknown values, and
// typed bit composition; these are not upstream test vectors.

func TestBase64Options(t *testing.T) {
	if kind := reflect.TypeOf(Base64Default).Kind(); kind != reflect.Uint64 {
		t.Fatalf("Base64Options underlying kind = %v, want uint64", kind)
	}

	tests := []struct {
		name    string
		options Base64Options
		value   uint64
		text    string
	}{
		{"default", Base64Default, 0, "base64_default"},
		{"URL", Base64URL, 1, "base64_url"},
		{"default no padding", Base64DefaultNoPadding, 2, "base64_reverse_padding"},
		{"URL with padding", Base64URLWithPadding, 3, "base64_url_with_padding"},
		{"default accept garbage", Base64DefaultAcceptGarbage, 4, "base64_default_accept_garbage"},
		{"URL accept garbage", Base64URLAcceptGarbage, 5, "base64_url_accept_garbage"},
		{"default or URL", Base64DefaultOrURL, 8, "base64_default_or_url"},
		{"default or URL accept garbage", Base64DefaultOrURLAcceptGarbage, 12, "base64_default_or_url_accept_garbage"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint64(test.options); got != test.value {
				t.Errorf("value = %d, want %d", got, test.value)
			}
			if got := Base64OptionsString(test.options); got != test.text {
				t.Errorf("Base64OptionsString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestBase64OptionConstants(t *testing.T) {
	if kind := reflect.TypeOf(Base64ReversePadding).Kind(); kind != reflect.Uint64 {
		t.Fatalf("Base64ReversePadding kind = %v, want uint64", kind)
	}
	if Base64ReversePadding != 2 {
		t.Errorf("Base64ReversePadding = %d, want 2", Base64ReversePadding)
	}
	if got := Base64Default | Base64Options(Base64ReversePadding); got != Base64DefaultNoPadding {
		t.Errorf("Base64Default | Base64ReversePadding = %d, want %d", got, Base64DefaultNoPadding)
	}
	if got := Base64URL | Base64Options(Base64ReversePadding); got != Base64URLWithPadding {
		t.Errorf("Base64URL | Base64ReversePadding = %d, want %d", got, Base64URLWithPadding)
	}
	if kind := reflect.TypeOf(DefaultLineLength).Kind(); kind != reflect.Int {
		t.Fatalf("DefaultLineLength kind = %v, want int", kind)
	}
	if DefaultLineLength != 76 {
		t.Errorf("DefaultLineLength = %d, want 76", DefaultLineLength)
	}
}

func TestBase64OptionsStringUnknown(t *testing.T) {
	for _, options := range []Base64Options{6, 7, 255} {
		if got := Base64OptionsString(options); got != "<unknown>" {
			t.Errorf("Base64OptionsString(%d) = %q, want <unknown>", options, got)
		}
	}
}

func TestLastChunkHandlingOptions(t *testing.T) {
	if kind := reflect.TypeOf(Loose).Kind(); kind != reflect.Uint64 {
		t.Fatalf("LastChunkHandlingOptions underlying kind = %v, want uint64", kind)
	}

	tests := []struct {
		name    string
		options LastChunkHandlingOptions
		value   uint64
		text    string
	}{
		{"loose", Loose, 0, "loose"},
		{"strict", Strict, 1, "strict"},
		{"stop before partial", StopBeforePartial, 2, "stop_before_partial"},
		{"only full chunks", OnlyFullChunks, 3, "only_full_chunks"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint64(test.options); got != test.value {
				t.Errorf("value = %d, want %d", got, test.value)
			}
			if got := LastChunkHandlingOptionsString(test.options); got != test.text {
				t.Errorf("LastChunkHandlingOptionsString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestIsPartial(t *testing.T) {
	tests := []struct {
		name    string
		options LastChunkHandlingOptions
		want    bool
	}{
		{"loose", Loose, false},
		{"strict", Strict, false},
		{"stop before partial", StopBeforePartial, true},
		{"only full chunks", OnlyFullChunks, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsPartial(test.options); got != test.want {
				t.Errorf("IsPartial() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLastChunkHandlingOptionsStringUnknown(t *testing.T) {
	for _, options := range []LastChunkHandlingOptions{4, 255} {
		if got := LastChunkHandlingOptionsString(options); got != "<unknown>" {
			t.Errorf("LastChunkHandlingOptionsString(%d) = %q, want <unknown>", options, got)
		}
	}
}

// Contract cases derived from simdutf commit 611becc2a08c27a4edc77d9a45ff74c97130129b,
// include/simdutf/encoding_types.h:15-24 and src/encoding_types.cpp:3-64.
// Narrow Go-only scaffolding covers the underlying type, unknown values,
// truncated inputs, and non-prefix BOMs; these are not upstream test vectors.

func TestEncoding(t *testing.T) {
	if kind := reflect.TypeOf(Unspecified).Kind(); kind != reflect.Uint8 {
		t.Fatalf("Encoding underlying kind = %v, want uint8", kind)
	}

	tests := []struct {
		name     string
		encoding Encoding
		value    uint8
		text     string
	}{
		{"unspecified", Unspecified, 0, "unknown"},
		{"UTF-8", UTF8, 1, "UTF8"},
		{"UTF-16LE", UTF16LE, 2, "UTF16 little-endian"},
		{"UTF-16BE", UTF16BE, 4, "UTF16 big-endian"},
		{"UTF-32LE", UTF32LE, 8, "UTF32 little-endian"},
		{"UTF-32BE", UTF32BE, 16, "UTF32 big-endian"},
		{"Latin-1", Latin1, 32, "error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := uint8(test.encoding); got != test.value {
				t.Errorf("value = %d, want %d", got, test.value)
			}
			if got := EncodingString(test.encoding); got != test.text {
				t.Errorf("EncodingString() = %q, want %q", got, test.text)
			}
		})
	}
}

func TestEncodingStringUnknown(t *testing.T) {
	for _, encoding := range []Encoding{3, 255} {
		if got := EncodingString(encoding); got != "error" {
			t.Errorf("EncodingString(%d) = %q, want error", encoding, got)
		}
	}
}

func TestCheckBOM(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  Encoding
	}{
		{"nil", nil, Unspecified},
		{"empty", []byte{}, Unspecified},
		{"one byte UTF-16LE prefix", []byte{0xff}, Unspecified},
		{"one byte UTF-16BE prefix", []byte{0xfe}, Unspecified},
		{"two byte UTF-8 prefix", []byte{0xef, 0xbb}, Unspecified},
		{"three byte UTF-32BE prefix", []byte{0x00, 0x00, 0xfe}, Unspecified},
		{"UTF-8", []byte{0xef, 0xbb, 0xbf}, UTF8},
		{"UTF-8 with payload", []byte{0xef, 0xbb, 0xbf, 'x'}, UTF8},
		{"UTF-16LE", []byte{0xff, 0xfe}, UTF16LE},
		{"UTF-16LE three byte truncation", []byte{0xff, 0xfe, 0x00}, UTF16LE},
		{"UTF-16LE non-UTF-32 suffix", []byte{0xff, 0xfe, 0x00, 0x01}, UTF16LE},
		{"UTF-16BE", []byte{0xfe, 0xff}, UTF16BE},
		{"UTF-32LE precedence", []byte{0xff, 0xfe, 0x00, 0x00}, UTF32LE},
		{"UTF-32LE with payload", []byte{0xff, 0xfe, 0x00, 0x00, 'x'}, UTF32LE},
		{"UTF-32BE", []byte{0x00, 0x00, 0xfe, 0xff}, UTF32BE},
		{"non-prefix UTF-8", []byte{'x', 0xef, 0xbb, 0xbf}, Unspecified},
		{"non-prefix UTF-16LE", []byte{'x', 0xff, 0xfe}, Unspecified},
		{"no BOM", []byte("plain text"), Unspecified},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CheckBOM(test.input); got != test.want {
				t.Errorf("CheckBOM(% x) = %d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestBOMByteSize(t *testing.T) {
	tests := []struct {
		name     string
		encoding Encoding
		want     int
	}{
		{"unspecified", Unspecified, 0},
		{"UTF-8", UTF8, 3},
		{"UTF-16LE", UTF16LE, 2},
		{"UTF-16BE", UTF16BE, 2},
		{"UTF-32LE", UTF32LE, 4},
		{"UTF-32BE", UTF32BE, 4},
		{"Latin-1", Latin1, 0},
		{"unknown", Encoding(255), 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := BOMByteSize(test.encoding); got != test.want {
				t.Errorf("BOMByteSize() = %d, want %d", got, test.want)
			}
		})
	}
}
