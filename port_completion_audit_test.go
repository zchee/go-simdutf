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
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zchee/go-simdutf/internal/portplan"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	return filepath.Clean(filepath.Dir(file))
}

func TestPortCompletionAudit(t *testing.T) {
	root := repositoryRoot(t)

	// Fail closed on host/SSH authority drift against locked inputs.
	hostBytes, err := os.ReadFile(filepath.Join(root, "docs/porting/simdutf-port-v1/inputs/host-authority-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	hostText := string(hostBytes)
	for _, must := range []string{
		"darwin-arm64-apple-m3-max",
		"linux-amd64-debian-13-xeon-platinum-8481c",
		"ssh:debian-13-trixie.gaudiy-platform",
	} {
		if !strings.Contains(hostText, must) {
			t.Fatalf("host authority missing %q", must)
		}
	}

	lockedBytes, err := os.ReadFile(filepath.Join(root, "docs/porting/simdutf-port-v1/inputs/locked-sets-v1.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	for _, lane := range []string{
		"darwin-arm64-nosimd",
		"darwin-arm64-simd-negative",
		"linux-amd64-none",
		"linux-amd64-simd",
		"linux-riscv64-compile",
		"linux-s390x-compile",
	} {
		if !lockedSetContainsEvidenceLane(lockedBytes, lane) {
			t.Fatalf("locked-sets missing evidence lane %q", lane)
		}
	}

	// Six literal D3 lane raw proofs must exist, bind authority, and report exit_ok.
	laneIndexPath := filepath.Join(root, "docs/porting/simdutf-port-v1/evidence/d3-lanes-index-v1.json")
	laneIndexRaw, err := os.ReadFile(laneIndexPath)
	if err != nil {
		t.Fatalf("missing D3 lanes index: %v", err)
	}
	var laneIndex struct {
		Schema string `json:"schema"`
		Lanes  []struct {
			LaneID    string `json:"lane_id"`
			HostID    string `json:"host_id"`
			Transport string `json:"transport"`
			ExitOK    bool   `json:"exit_ok"`
			RawSHA256 string `json:"raw_sha256"`
		} `json:"lanes"`
	}
	if err := json.Unmarshal(laneIndexRaw, &laneIndex); err != nil {
		t.Fatal(err)
	}
	if laneIndex.Schema != "simdutf-d3-lanes-index-v1" || len(laneIndex.Lanes) != 6 {
		t.Fatalf("D3 lanes index invalid: schema=%s lanes=%d", laneIndex.Schema, len(laneIndex.Lanes))
	}
	seen := map[string]bool{}
	for _, lane := range laneIndex.Lanes {
		seen[lane.LaneID] = true
		if !lane.ExitOK {
			t.Fatalf("D3 lane %s not exit_ok", lane.LaneID)
		}
		proof := filepath.Join(root, "docs/porting/simdutf-port-v1/evidence", "d3-"+lane.LaneID+"-lane-v1.txt")
		raw, err := os.ReadFile(proof)
		if err != nil {
			t.Fatalf("missing lane proof %s: %v", lane.LaneID, err)
		}
		if !bytes.Contains(raw, []byte("exit_ok: true")) {
			t.Fatalf("lane proof %s missing exit_ok", lane.LaneID)
		}
		switch lane.LaneID {
		case "darwin-arm64-nosimd", "darwin-arm64-simd-negative":
			if lane.HostID != "darwin-arm64-apple-m3-max" || lane.Transport != "local-physical" {
				t.Fatalf("darwin lane authority drift: %#v", lane)
			}
		case "linux-amd64-none", "linux-amd64-simd", "linux-riscv64-compile", "linux-s390x-compile":
			if lane.HostID != "linux-amd64-debian-13-xeon-platinum-8481c" || lane.Transport != "ssh:debian-13-trixie.gaudiy-platform" {
				t.Fatalf("linux lane authority drift: %#v", lane)
			}
		default:
			t.Fatalf("unexpected lane id %q", lane.LaneID)
		}
	}
	for _, want := range []string{
		"darwin-arm64-nosimd", "darwin-arm64-simd-negative",
		"linux-amd64-none", "linux-amd64-simd",
		"linux-riscv64-compile", "linux-s390x-compile",
	} {
		if !seen[want] {
			t.Fatalf("missing D3 lane %s", want)
		}
	}

	// Qualification summaries for W01-W07 families must exist.
	for _, name := range []string{
		"w01-qualification-summary-v1.json",
		"latin1-qualification-summary-v1.json",
		"utf8-qualification-summary-v1.json",
		"utf16-qualification-summary-v1.json",
		"utf32-qualification-summary-v1.json",
		"find-qualification-summary-v1.json",
		"detect-qualification-summary-v1.json",
		"base64-qualification-summary-v1.json",
	} {
		if _, err := os.Stat(filepath.Join(root, "docs/porting/simdutf-port-v1/evidence", name)); err != nil {
			t.Fatalf("missing qualification summary %s: %v", name, err)
		}
	}

	// Dispositions must be legal.
	dispositions, err := portplan.LoadQualificationDispositionsV1(filepath.Join(root, "docs/porting/simdutf-port-v1/evidence"))
	if err != nil {
		t.Fatal(err)
	}
	for key, disp := range dispositions {
		switch disp {
		case "selected", "direct_only":
		default:
			t.Fatalf("illegal disposition %q for %s", disp, key)
		}
	}

	// Manifest totals must be exactly 155/0/9.
	manifest, err := portplan.ParseManifestV1(mustRead(t, filepath.Join(root, "docs/porting/api-manifest.tsv")))
	if err != nil {
		t.Fatal(err)
	}
	implemented, planned, excluded := countManifestStatuses(manifest)
	if implemented != 155 || planned != 0 || excluded != 9 {
		t.Fatalf("manifest counts %d/%d/%d want 155/0/9", implemented, planned, excluded)
	}

	// Working tree must be clean for a durable completion claim.
	status, err := exec.Command("git", "-C", root, "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(bytes.TrimSpace(status)) != 0 {
		t.Fatalf("working tree is dirty; completion audit requires SourceClean:\n%s", status)
	}

	completion, context, err := portplan.BuildRepositoryCompletionV1(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := portplan.ValidateCompletionV1(completion, context); err != nil {
		t.Fatalf("ValidateCompletionV1: %v", err)
	}
	rendered, err := portplan.RenderCompletionV1(completion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := portplan.ValidateCompletionJSONV1(rendered, context); err != nil {
		t.Fatalf("ValidateCompletionJSONV1: %v", err)
	}
	if len(completion.Rows) != 125 {
		t.Fatalf("rows=%d want 125", len(completion.Rows))
	}
	cells := 0
	for _, row := range completion.Rows {
		cells += len(row.Cells)
	}
	if cells != 500 {
		t.Fatalf("cells=%d want 500", cells)
	}
	if completion.Implemented != 155 || completion.Planned != 0 || completion.Excluded != 9 {
		t.Fatalf("completion totals %d/%d/%d", completion.Implemented, completion.Planned, completion.Excluded)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func countManifestStatuses(rows []portplan.ManifestRowV1) (implemented, planned, excluded int) {
	for _, row := range rows {
		if len(row.Cells) < 3 {
			continue
		}
		// api-manifest.tsv: status is the third-from-last column (status, milestone, exclusion_reason).
		switch row.Cells[len(row.Cells)-3] {
		case "implemented":
			implemented++
		case "planned":
			planned++
		case "excluded":
			excluded++
		}
	}
	return implemented, planned, excluded
}

func lockedSetContainsEvidenceLane(locked []byte, lane string) bool {
	// locked-sets stores hex of the lane string in the value column.
	hex := make([]byte, 0, len(lane)*2)
	for i := 0; i < len(lane); i++ {
		hex = append(hex, "0123456789abcdef"[lane[i]>>4], "0123456789abcdef"[lane[i]&0xf])
	}
	return bytes.Contains(locked, hex) || bytes.Contains(locked, []byte(lane))
}
