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
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSourceIdentityEmitsActionSpecificStdoutAndReceipt(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "old.tar")
	receipt := filepath.Join(dir, "old.json")
	if err := os.WriteFile(archive, []byte("archive-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	seed := sourceIdentitySeedV1{
		Schema: "simdutf-source-identity-seed-v1", Version: 1,
		Commit: strings.Repeat("a", 40), Tree: strings.Repeat("b", 40), Parent: strings.Repeat("c", 40), Status: "clean",
	}
	seedBytes, err := json.Marshal(seed)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive+".seed.json", append(seedBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout := captureStdout(t, func() {
		if err := run([]string{
			"source-identity",
			"--action=source_commit",
			"--role=old",
			"--receipt=" + receipt,
			"--archive=" + archive,
		}); err != nil {
			t.Fatal(err)
		}
	})
	if stdout != strings.Repeat("a", 40)+"\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	raw, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	var got sourceIdentityReceiptV1
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != "simdutf-source-identity-receipt-v1" || got.Role != "old" || got.Status != "clean" || got.ArchivePath != archive || got.ArchiveDigest == "" {
		t.Fatalf("receipt = %#v", got)
	}
}

func TestStateTransitionStdoutMatchesFlags(t *testing.T) {
	stdout := captureStdout(t, func() {
		if err := run([]string{
			"state-transition",
			"--state-subject=row",
			"--prerequisite-state=snapshot_planned",
			"--current-state=scalar_private",
			"--disposition=",
			"--go-qualification=",
			"--proof-receipt-id=",
		}); err != nil {
			t.Fatal(err)
		}
	})
	var got stateTransitionArtifactV1
	dec := json.NewDecoder(strings.NewReader(stdout))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != "simdutf-state-transition-v1" || got.StateSubject != "row" || got.CurrentState != "scalar_private" {
		t.Fatalf("artifact = %#v", got)
	}
}

func TestQuietAffinityLinuxOnly(t *testing.T) {
	err := run([]string{"quiet-affinity-recheck", "--cpu=1", "--policy=taskset:1"})
	if runtime.GOOS == "linux" {
		return
	}
	if err == nil || !strings.Contains(err.Error(), "Linux-only") {
		t.Fatalf("err = %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = original }()
	fn()
	w.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
