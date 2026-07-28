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
	"strings"
	"testing"
)

func TestParseManifestV1RejectsMalformedInput(t *testing.T) {
	valid := manifestFixture(
		manifestRecord("implemented", "611becc-current-api", "one"),
		manifestRecord("planned", "future-upstream-api", "two"),
	)
	tests := []struct {
		name string
		data string
	}{
		{"missing trailing LF", strings.TrimSuffix(valid, "\n")},
		{"CRLF", strings.ReplaceAll(valid, "\n", "\r\n")},
		{"wrong header", strings.Replace(valid, "family", "Family", 1)},
		{"wrong width", manifestFixture("too\tfew")},
		{"invalid status pair", manifestFixture(manifestRecord("planned", "611becc-current-api", "bad"))},
		{"duplicate planned composite", manifestFixture(manifestRecord("planned", "future-upstream-api", "same"), manifestRecord("planned", "future-upstream-api", "same"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseManifestV1([]byte(test.data)); err == nil {
				t.Fatal("ParseManifestV1 succeeded")
			}
		})
	}
}

func TestFreezePlannedRowsV1PreservesBytesAndOrder(t *testing.T) {
	implemented := manifestRecord("implemented", "611becc-current-api", "first")
	plannedOne := manifestRecord("planned", "future-upstream-api", "second")
	plannedTwo := manifestRecord("planned", "future-upstream-api", "third")
	input := []byte(manifestFixture(implemented, plannedOne, plannedTwo))

	rendered, rows, err := FreezePlannedRowsV1(input)
	if err != nil {
		t.Fatalf("FreezePlannedRowsV1: %v", err)
	}
	want := []byte(strings.Join([]string{strings.Join(ManifestHeaderV1(), "\t"), plannedOne, plannedTwo}, "\n") + "\n")
	if string(rendered) != string(want) {
		t.Fatalf("rendered = %q, want %q", rendered, want)
	}
	if len(rows) != 2 || rows[0].PlannedOrdinal != 1 || rows[1].PlannedOrdinal != 2 {
		t.Fatalf("planned rows = %#v", rows)
	}
	if rows[0].RowKeyV1 == "" || rows[1].RowKeyV1 == "" {
		t.Fatalf("planned row keys = %q, %q", rows[0].RowKeyV1, rows[1].RowKeyV1)
	}
}

func TestParseISALedgerV1RejectsMalformedInput(t *testing.T) {
	valid := isaFixture()
	tests := []struct {
		name string
		data string
	}{
		{"wrong width", strings.Replace(valid, "operation-01", "operation-01\textra", 1)},
		{"duplicate semantic operation", strings.Replace(valid, "operation-02", "operation-01", 1)},
		{"empty semantic operation", strings.Replace(valid, "operation-01", "", 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseISALedgerV1([]byte(test.data)); err == nil {
				t.Fatal("ParseISALedgerV1 succeeded")
			}
		})
	}
}

func manifestFixture(records ...string) string {
	return strings.Join(append([]string{strings.Join(ManifestHeaderV1(), "\t")}, records...), "\n") + "\n"
}

func manifestRecord(status, milestone, symbol string) string {
	fields := make([]string, len(ManifestHeaderV1()))
	for index := range fields {
		fields[index] = "value"
	}
	fields[0] = "family"
	fields[1] = "upstream-" + symbol
	fields[2] = "signature-" + symbol
	fields[3] = "header:" + symbol
	fields[6] = "Go" + symbol
	fields[7] = "func " + symbol + "()"
	fields[20] = status
	fields[21] = milestone
	return strings.Join(fields, "\t")
}

func isaFixture() string {
	lines := []string{strings.Join(ISALedgerHeaderV1(), "\t")}
	for row := 1; row <= 23; row++ {
		fields := make([]string, len(ISALedgerHeaderV1()))
		for index := range fields {
			fields[index] = "value"
		}
		fields[0] = "operation-" + twoDigits(row)
		lines = append(lines, strings.Join(fields, "\t"))
	}
	return strings.Join(lines, "\n") + "\n"
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}
