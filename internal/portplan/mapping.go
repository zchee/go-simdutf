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
	"fmt"
	"strconv"
	"strings"
)

var (
	reviewedMappingHeaderV1 = [...]string{"mapping_version", "manifest_ordinal", "go_symbol", "family_contract_display_id", "isa_ordinal_or_scalar", "canonical_kernel_name", "westmere_outcome", "westmere_direct_symbol", "westmere_na_reason", "westmere_evidence_anchor", "haswell_outcome", "haswell_direct_symbol", "haswell_na_reason", "haswell_evidence_anchor", "archsimd_outcome", "archsimd_direct_symbol", "archsimd_na_reason", "archsimd_evidence_anchor", "neon_outcome", "neon_direct_symbol", "neon_na_reason", "neon_evidence_anchor"}
	existingMemberHeaderV1  = [...]string{"mapping_version", "go_symbol", "isa_ordinal_or_scalar", "evidence_anchor"}
	finalOperationsHeaderV1 = [...]string{"mapping_version", "isa_ordinal", "isa_semantic_operation_exact", "semantic_operation_id", "initial_row_count", "initial_cell_count", "existing_row_count", "reconciliation", "evidence_anchor"}
	finalCellsHeaderV1      = [...]string{"mapping_version", "row_key_v1", "manifest_ordinal", "canonical_row_rank", "family_contract_display_id", "backend", "cell_storage_id", "semantic_operation_id", "isa_ordinal_or_scalar", "backend_outcome", "backend_na_reason", "backend_evidence_anchor", "direct_symbol", "direct_symbol_storage_id", "shared_kernel_id", "kernel_owner_cell_key", "kernel_owner_dependency_id", "kernel_batch_display_id", "kernel_batch_storage_id", "evidence_batch_display_id", "evidence_batch_storage_id"}
	kernelRegistryHeaderV1  = [...]string{"mapping_version", "backend", "canonical_kernel_name", "shared_kernel_id", "family_contract_display_id", "semantic_operation_id", "kernel_owner_cell_key", "member_count"}
)

type (
	BackendMappingV1  struct{ Outcome, DirectSymbol, NAReason, EvidenceAnchor string }
	ReviewedMappingV1 struct {
		ManifestOrdinal                                                            int
		GoSymbol, FamilyContractDisplayID, ISAOrdinalOrScalar, CanonicalKernelName string
		Backends                                                                   [4]BackendMappingV1
	}
)
type ExistingMemberV1 struct{ GoSymbol, ISAOrdinalOrScalar, EvidenceAnchor string }

func ParseReviewedMappingsV1(data []byte, planned []ManifestRowV1, ledger []ISARowV1) ([]ReviewedMappingV1, error) {
	lines, err := parseLines(data, "reviewed mapping")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], reviewedMappingHeaderV1[:], "reviewed mapping"); err != nil {
		return nil, err
	}
	if len(lines)-1 != 125 {
		return nil, fmt.Errorf("reviewed mapping: got %d data rows, want 125", len(lines)-1)
	}
	if len(planned) != 125 {
		return nil, fmt.Errorf("planned rows: got %d rows, want 125", len(planned))
	}
	if len(ledger) != 23 {
		return nil, fmt.Errorf("ISA ledger: got %d rows, want 23", len(ledger))
	}
	out := make([]ReviewedMappingV1, 0, 125)
	for i, line := range lines[1:] {
		f, e := parseRecord(line, len(reviewedMappingHeaderV1), i+2, "reviewed mapping")
		if e != nil {
			return nil, e
		}
		if f[0] != "v1" {
			return nil, fmt.Errorf("reviewed mapping line %d: invalid mapping version", i+2)
		}
		n, e := strconv.Atoi(f[1])
		if e != nil || strconv.Itoa(n) != f[1] || n != i+1 || planned[i].PlannedOrdinal != n || planned[i].Cells[manifestGoSymbolIndex] != f[2] {
			return nil, fmt.Errorf("reviewed mapping line %d: manifest join mismatch", i+2)
		}
		family, e := ClassifyFamilyContractV1(planned[i].Cells[manifestFamilyIndex], f[2])
		if e != nil || f[3] != family {
			return nil, fmt.Errorf("reviewed mapping line %d: family mismatch", i+2)
		}
		m := ReviewedMappingV1{ManifestOrdinal: n, GoSymbol: f[2], FamilyContractDisplayID: f[3], ISAOrdinalOrScalar: f[4], CanonicalKernelName: f[5]}
		if f[4] == "scalar" {
			if f[5] != "" {
				return nil, fmt.Errorf("reviewed mapping line %d: scalar kernel", i+2)
			}
		} else {
			ord, e := strconv.Atoi(f[4])
			if e != nil || ord < 1 || ord > len(ledger) || strconv.Itoa(ord) != f[4] || ledger[ord-1].LedgerOrdinal != ord {
				return nil, fmt.Errorf("reviewed mapping line %d: ISA ordinal", i+2)
			}
		}
		for b := range 4 {
			x := 6 + b*4
			m.Backends[b] = BackendMappingV1{f[x], f[x+1], f[x+2], f[x+3]}
			if e := validateBackendMapping(m, b, planned[i], ledgerForMapping(m, ledger)); e != nil {
				return nil, fmt.Errorf("reviewed mapping line %d: %w", i+2, e)
			}
		}
		if f[4] == "scalar" {
			for _, x := range m.Backends {
				if x.Outcome != "not_applicable" {
					return nil, fmt.Errorf("reviewed mapping line %d: scalar row has eligible backend", i+2)
				}
			}
		}
		out = append(out, m)
	}
	return out, nil
}

func ParseReviewedExistingMembersV1(data []byte, manifest []ManifestRowV1, ledger []ISARowV1) ([]ExistingMemberV1, error) {
	lines, e := parseLines(data, "reviewed existing members")
	if e != nil {
		return nil, e
	}
	if e = validateHeader(lines[0], existingMemberHeaderV1[:], "reviewed existing members"); e != nil {
		return nil, e
	}
	if len(lines)-1 != 30 {
		return nil, fmt.Errorf("reviewed existing members: got %d data rows, want 30", len(lines)-1)
	}
	implementedBySymbol := make(map[string]ManifestRowV1, len(manifest))
	for _, row := range manifest {
		if row.Cells[manifestStatusIndex] == "implemented" {
			implementedBySymbol[row.Cells[manifestGoSymbolIndex]] = row
		}
	}
	out := make([]ExistingMemberV1, 0, 30)
	seen := map[string]bool{}
	for i, l := range lines[1:] {
		f, e := parseRecord(l, 4, i+2, "reviewed existing members")
		if e != nil {
			return nil, e
		}
		row, ok := implementedBySymbol[f[1]]
		if f[0] != "v1" || f[1] == "" || f[2] == "" || f[3] == "" || !ok ||
			row.Cells[manifestGoSymbolIndex] != f[1] {
			return nil, fmt.Errorf("reviewed existing members line %d: manifest join mismatch", i+2)
		}
		if seen[f[1]] {
			return nil, fmt.Errorf("reviewed existing members line %d: duplicate symbol", i+2)
		}
		if f[2] != "scalar" {
			n, e := strconv.Atoi(f[2])
			if e != nil || n < 1 || n > len(ledger) || strconv.Itoa(n) != f[2] || ledger[n-1].LedgerOrdinal != n {
				return nil, fmt.Errorf("reviewed existing members line %d: ISA ordinal", i+2)
			}
		}
		seen[f[1]] = true
		out = append(out, ExistingMemberV1{f[1], f[2], f[3]})
	}
	return out, nil
}

func ledgerForMapping(m ReviewedMappingV1, ledger []ISARowV1) *ISARowV1 {
	if m.ISAOrdinalOrScalar == "scalar" {
		return nil
	}
	n, err := strconv.Atoi(m.ISAOrdinalOrScalar)
	if err != nil || n < 1 || n > len(ledger) {
		return nil
	}
	return &ledger[n-1]
}

func validateBackendMapping(m ReviewedMappingV1, index int, manifest ManifestRowV1, selected ...*ISARowV1) error {
	x := m.Backends[index]
	if x.Outcome != "eligible" && x.Outcome != "not_applicable" {
		return fmt.Errorf("invalid outcome %q", x.Outcome)
	}
	if x.EvidenceAnchor == "" {
		return fmt.Errorf("empty evidence anchor")
	}
	source := manifest.Cells[15]
	if index == 3 {
		source = manifest.Cells[16]
	}
	sourceNA := strings.HasPrefix(sourcePathToken(source), "not_applicable:")
	requiresPinnedSource := index != 2 || x.Outcome == "not_applicable" && (x.NAReason != "primitive_gap" || m.ISAOrdinalOrScalar == "scalar")
	if requiresPinnedSource && (!strings.Contains(x.EvidenceAnchor, "611becc2a08c27a4edc77d9a45ff74c97130129b") || !strings.Contains(x.EvidenceAnchor, manifest.Cells[3])) {
		return fmt.Errorf("source evidence must pin commit and manifest header_path_line")
	}
	if x.Outcome == "eligible" {
		if m.ISAOrdinalOrScalar == "scalar" || m.CanonicalKernelName == "" || x.NAReason != "" || sourceNA {
			return fmt.Errorf("invalid eligible cell")
		}
		suffix := [...]string{"Westmere", "Haswell", "Archsimd", "NEON"}[index]
		if x.DirectSymbol != lowerFirst(m.GoSymbol)+suffix {
			return fmt.Errorf("direct symbol %q does not match Go symbol", x.DirectSymbol)
		}
		if index == 2 {
			if !strings.Contains(x.EvidenceAnchor, "simd/archsimd Go 1.26.5:") {
				return fmt.Errorf("archsimd evidence anchor must name primitives")
			}
		} else if len(selected) != 1 || selected[0] == nil {
			return fmt.Errorf("backend evidence lacks pinned ISA ledger source")
		} else {
			pinnedSource := sourcePathToken(selected[0].Cells[[4]int{3, 4, 0, 5}[index]])
			if !strings.HasPrefix(pinnedSource, "src/") || !strings.Contains(x.EvidenceAnchor, pinnedSource) {
				return fmt.Errorf("backend evidence must name pinned ISA ledger source")
			}
		}
		return nil
	}
	if x.DirectSymbol != "" || !validNAReason(x.NAReason) {
		return fmt.Errorf("invalid not-applicable cell")
	}
	if x.NAReason == "native_wrapper_delegates_explicit_endian" && (!sourceNA || !strings.Contains(source, "native wrapper delegates")) {
		return fmt.Errorf("native delegation reason lacks source-declared N/A")
	}
	if x.NAReason == "composite_wrapper_delegates_accelerated_core" && (!sourceNA || !strings.Contains(x.EvidenceAnchor, "dependency:")) {
		return fmt.Errorf("composite wrapper requires source N/A and dependency evidence")
	}
	if x.NAReason == "primitive_gap" {
		if index != 2 && m.ISAOrdinalOrScalar != "scalar" && !sourceNA {
			return fmt.Errorf("primitive gap requires source-declared N/A")
		}
		if index == 2 && m.ISAOrdinalOrScalar != "scalar" && !strings.Contains(x.EvidenceAnchor, "simd/archsimd Go 1.26.5:") {
			return fmt.Errorf("archsimd primitive gap must name primitives")
		}
	}
	return nil
}

func sourcePathToken(source string) string {
	source = strings.TrimSpace(source)
	if i := strings.IndexAny(source, ";("); i >= 0 {
		source = strings.TrimSpace(source[:i])
	}
	return source
}

func FamilyContractDisplayIDV1(family string) string {
	switch family {
	case "Encoding detection":
		return "FC-v1-detection"
	case "UTF-16":
		return "FC-v1-utf16-source"
	case "UTF-32":
		return "FC-v1-utf32-source"
	case "UTF-8":
		return "FC-v1-utf8-source"
	case "Latin-1":
		return "FC-v1-latin1-source"
	case "Base64":
		return "FC-v1-base64"
	default:
		return ""
	}
}

func validNAReason(reason string) bool {
	switch reason {
	case "native_wrapper_delegates_explicit_endian", "primitive_gap", "composite_wrapper_delegates_accelerated_core":
		return true
	default:
		return false
	}
}

// ClassifyFamilyContractV1 assigns every reviewed Go symbol to exactly one family.
func ClassifyFamilyContractV1(manifestFamily, symbol string) (string, error) {
	if manifestFamily == "Base64" {
		return "FC-v1-base64", nil
	}
	if strings.HasPrefix(symbol, "Validate") || strings.HasPrefix(symbol, "ToWellFormed") {
		return "FC-v1-helper-validation", nil
	}
	switch symbol {
	case "AutodetectEncoding", "DetectEncodings":
		return "FC-v1-detection", nil
	case "Find", "FindUTF16":
		return "FC-v1-find", nil
	}
	var token string
	switch {
	case strings.HasPrefix(symbol, "ConvertValid"):
		token = exactConvertSource(symbol[len("ConvertValid"):])
	case strings.HasPrefix(symbol, "Convert"):
		token = exactConvertSource(symbol[len("Convert"):])
	case strings.Contains(symbol, "LengthFrom"):
		parts := strings.Split(symbol, "LengthFrom")
		if len(parts) != 2 || parts[0] == "" {
			return "", fmt.Errorf("unclassified symbol %q", symbol)
		}
		token = strings.TrimSuffix(parts[1], "WithReplacement")
	case strings.HasPrefix(symbol, "Count"):
		token = symbol[len("Count"):]
	case strings.HasPrefix(symbol, "TrimPartial"):
		token = symbol[len("TrimPartial"):]
	case strings.HasPrefix(symbol, "ChangeEndianness"):
		token = symbol[len("ChangeEndianness"):]
	default:
		return "", fmt.Errorf("unclassified symbol %q", symbol)
	}
	family, ok := sourceContract(token)
	if !ok {
		return "", fmt.Errorf("unclassified source token %q in %q", token, symbol)
	}
	return family, nil
}

func exactConvertSource(s string) string {
	parts := strings.Split(s, "To")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ""
	}
	return parts[0]
}

func sourceContract(s string) (string, bool) {
	switch s {
	case "UTF16", "UTF16LE", "UTF16BE":
		return "FC-v1-utf16-source", true
	case "UTF32":
		return "FC-v1-utf32-source", true
	case "UTF8":
		return "FC-v1-utf8-source", true
	case "Latin1":
		return "FC-v1-latin1-source", true
	default:
		return "", false
	}
}

func lowerFirst(s string) string {
	for _, prefix := range []string{"UTF16", "UTF32", "UTF8", "Base64"} {
		if strings.HasPrefix(s, prefix) {
			return strings.ToLower(prefix) + s[len(prefix):]
		}
	}
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
