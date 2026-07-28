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

// Package portplan parses the frozen Phase 0 porting inputs.
package portplan

var manifestHeaderV1 = [...]string{
	"family", "upstream_symbol", "upstream_signature", "header_path_line",
	"feature_gate", "overload_disposition", "go_symbol", "go_signature",
	"input_unit", "output_unit", "result_shape", "count_meaning",
	"destination_precondition", "alias_contract", "scalar_source", "amd64_source",
	"arm64_source", "test_source", "fuzz_source", "benchmark_source", "status",
	"milestone", "exclusion_reason",
}

var isaLedgerHeaderV1 = [...]string{
	"semantic_operation", "upstream_scalar_source", "upstream_generic_source",
	"westmere_source", "haswell_source", "arm64_source", "westmere_declared_features",
	"haswell_declared_features", "arm64_features", "amd64_asm_phase", "amd64_asm_reason",
	"arm64_asm_phase", "arm64_asm_reason", "archsimd_status", "archsimd_reason",
	"required_opcode_audit", "provenance_notes",
}

// ManifestHeaderV1 returns a copy of the required API manifest header.
func ManifestHeaderV1() []string {
	return append([]string(nil), manifestHeaderV1[:]...)
}

// ISALedgerHeaderV1 returns a copy of the required ISA eligibility header.
func ISALedgerHeaderV1() []string {
	return append([]string(nil), isaLedgerHeaderV1[:]...)
}

// ManifestRowV1 is one exact API manifest record.
type ManifestRowV1 struct {
	Cells          [23]string
	SourceLine     int
	PlannedOrdinal int
	RowKeyV1       string
}

// ISARowV1 is one exact ISA eligibility ledger record.
type ISARowV1 struct {
	Cells             [17]string
	LedgerOrdinal     int
	SemanticOperation string
}
