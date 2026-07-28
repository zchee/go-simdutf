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
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	manifestFamilyIndex            = 0
	manifestUpstreamSymbolIndex    = 1
	manifestUpstreamSignatureIndex = 2
	manifestHeaderPathLineIndex    = 3
	manifestGoSymbolIndex          = 6
	manifestGoSignatureIndex       = 7
	manifestStatusIndex            = 20
	manifestMilestoneIndex         = 21
)

// SHA256Hex returns the lowercase SHA-256 digest of data.
func SHA256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// ParseManifestV1 strictly parses an API manifest V1 input.
func ParseManifestV1(data []byte) ([]ManifestRowV1, error) {
	lines, err := parseLines(data, "API manifest")
	if err != nil {
		return nil, err
	}
	if err := validateHeader(lines[0], manifestHeaderV1[:], "API manifest"); err != nil {
		return nil, err
	}

	rows := make([]ManifestRowV1, 0, len(lines)-1)
	rowComposites := make(map[[6]string]struct{})
	rowKeys := make(map[string]struct{})
	plannedOrdinal := 0
	for index, line := range lines[1:] {
		fields, err := parseRecord(line, len(manifestHeaderV1), index+2, "API manifest")
		if err != nil {
			return nil, err
		}
		if !validManifestStatusMilestone(fields[manifestStatusIndex], fields[manifestMilestoneIndex]) {
			return nil, fmt.Errorf("API manifest line %d: invalid status/milestone pair %q/%q", index+2, fields[manifestStatusIndex], fields[manifestMilestoneIndex])
		}
		var row ManifestRowV1
		copy(row.Cells[:], fields)
		row.SourceLine = index + 2
		composite := manifestComposite(row.Cells)
		row.RowKeyV1 = RowKeyV1(composite)
		if _, duplicate := rowComposites[composite]; duplicate {
			return nil, fmt.Errorf("API manifest line %d: duplicate row composite", row.SourceLine)
		}
		if _, duplicate := rowKeys[row.RowKeyV1]; duplicate {
			return nil, fmt.Errorf("API manifest line %d: duplicate row key", row.SourceLine)
		}
		rowComposites[composite] = struct{}{}
		rowKeys[row.RowKeyV1] = struct{}{}
		if fields[manifestStatusIndex] == "planned" && fields[manifestMilestoneIndex] == "future-upstream-api" {
			plannedOrdinal++
			row.PlannedOrdinal = plannedOrdinal
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// ParseISALedgerV1 strictly parses an ISA eligibility ledger V1 input.
func ParseISALedgerV1(data []byte) ([]ISARowV1, error) {
	lines, err := parseLines(data, "ISA eligibility ledger")
	if err != nil {
		return nil, err
	}
	if err := validateHeader(lines[0], isaLedgerHeaderV1[:], "ISA eligibility ledger"); err != nil {
		return nil, err
	}
	if len(lines)-1 != 23 {
		return nil, fmt.Errorf("ISA eligibility ledger: got %d data rows, want 23", len(lines)-1)
	}

	rows := make([]ISARowV1, 0, 23)
	operations := make(map[string]struct{}, 23)
	for index, line := range lines[1:] {
		fields, err := parseRecord(line, len(isaLedgerHeaderV1), index+2, "ISA eligibility ledger")
		if err != nil {
			return nil, err
		}
		operation := fields[0]
		if operation == "" {
			return nil, fmt.Errorf("ISA eligibility ledger line %d: empty semantic operation", index+2)
		}
		if _, duplicate := operations[operation]; duplicate {
			return nil, fmt.Errorf("ISA eligibility ledger line %d: duplicate semantic operation %q", index+2, operation)
		}
		operations[operation] = struct{}{}
		var row ISARowV1
		copy(row.Cells[:], fields)
		row.LedgerOrdinal = index + 1
		row.SemanticOperation = operation
		rows = append(rows, row)
	}
	return rows, nil
}

// FreezePlannedRowsV1 renders the planned API rows without changing their bytes.
func FreezePlannedRowsV1(manifestData []byte) (rendered []byte, rows []ManifestRowV1, err error) {
	allRows, err := ParseManifestV1(manifestData)
	if err != nil {
		return nil, nil, err
	}
	selected := make([]ManifestRowV1, 0)
	for _, row := range allRows {
		if row.Cells[manifestStatusIndex] == "planned" && row.Cells[manifestMilestoneIndex] == "future-upstream-api" {
			selected = append(selected, row)
		}
	}
	lines := make([]string, 0, len(selected)+1)
	lines = append(lines, strings.Join(manifestHeaderV1[:], "\t"))
	for _, row := range selected {
		lines = append(lines, strings.Join(row.Cells[:], "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n"), selected, nil
}

func parseLines(data []byte, name string) ([]string, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s: invalid UTF-8", name)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("%s: require trailing LF", name)
	}
	if strings.ContainsRune(string(data), '\r') {
		return nil, fmt.Errorf("%s: CR is not permitted", name)
	}
	if strings.IndexByte(string(data), 0) >= 0 {
		return nil, fmt.Errorf("%s: NUL is not permitted", name)
	}
	lines := strings.Split(string(data[:len(data)-1]), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return nil, fmt.Errorf("%s: require exactly one header", name)
	}
	for index, line := range lines {
		if line == "" {
			return nil, fmt.Errorf("%s line %d: blank record", name, index+1)
		}
	}
	return lines, nil
}

func validateHeader(line string, expected []string, name string) error {
	fields := strings.Split(line, "\t")
	if len(fields) != len(expected) {
		return fmt.Errorf("%s: header has %d columns, want %d", name, len(fields), len(expected))
	}
	seen := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		if _, duplicate := seen[field]; duplicate {
			return fmt.Errorf("%s: duplicate header name %q", name, field)
		}
		seen[field] = struct{}{}
		if field != expected[index] {
			return fmt.Errorf("%s: header column %d is %q, want %q", name, index+1, field, expected[index])
		}
	}
	return nil
}

func parseRecord(line string, width, lineNumber int, name string) ([]string, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != width {
		return nil, fmt.Errorf("%s line %d: got %d columns, want %d", name, lineNumber, len(fields), width)
	}
	return fields, nil
}

func validManifestStatusMilestone(status, milestone string) bool {
	switch status {
	case "implemented":
		return milestone == "611becc-current-api"
	case "planned":
		return milestone == "future-upstream-api"
	case "excluded":
		return milestone == "upstream-excluded"
	default:
		return false
	}
}

func manifestComposite(cells [23]string) [6]string {
	return [6]string{
		cells[manifestFamilyIndex],
		cells[manifestUpstreamSymbolIndex],
		cells[manifestUpstreamSignatureIndex],
		cells[manifestHeaderPathLineIndex],
		cells[manifestGoSymbolIndex],
		cells[manifestGoSignatureIndex],
	}
}
