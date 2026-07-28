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

import "testing"

func TestClassificationV1HeadersAndRenderers(t *testing.T) {
	classification := ClassificationHeaderV1()
	if len(classification) != 20 || classification[0] != "mapping_version" || classification[19] != "benchmark_source" {
		t.Fatalf("classification header = %#v", classification)
	}
	batches := BatchesHeaderV1()
	if len(batches) != 10 || batches[1] != "batch_kind" || batches[9] != "member_tuple_hex" {
		t.Fatalf("batches header = %#v", batches)
	}
	if got := string(RenderClassificationV1(nil)); got == "" || got[len(got)-1] != '\n' {
		t.Fatalf("classification renderer did not produce LF-terminated output: %q", got)
	}
	if got := string(RenderBatchesV1(nil)); got == "" || got[len(got)-1] != '\n' {
		t.Fatalf("batch renderer did not produce LF-terminated output: %q", got)
	}
}

func TestClassificationDependenciesV1RejectsUnknownOwner(t *testing.T) {
	_, _, err := classificationDependencies([]DependencyRecordV1{{
		DependencyKind: "wrapper_delegate",
		OwnerKind:      "row",
		OwnerLogicalID: "rk-v1-unknown",
	}}, map[string]int{}, map[string]int{}, nil)
	if err == nil {
		t.Fatal("unknown dependency owner accepted")
	}
}
