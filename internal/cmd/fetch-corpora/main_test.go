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
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// zeroBackoff removes the retry sleeps so transport-failure tests stay fast.
func zeroBackoff(t *testing.T) {
	t.Helper()
	old := retryBackoff
	retryBackoff = []time.Duration{0, 0, 0}
	t.Cleanup(func() { retryBackoff = old })
}

func TestProvision(t *testing.T) {
	valid := []byte("frozen corpus fixture payload\n")
	validSum := fmt.Sprintf("%x", sha256.Sum256(valid))
	sameLength := []byte("FROZEN CORPUS FIXTURE PAYLOAD\n")

	tests := map[string]struct {
		existing   []byte // pre-placed at the destination; nil means absent
		served     []byte // HTTP response body; nil means 404
		wantErrSub string // "" means provision must succeed
		wantOutSub string
		wantOnDisk []byte // nil means the destination must be absent
		wantHits   int64  // -1 skips the request-count assertion
	}{
		"success: fetches a missing corpus": {
			served:     valid,
			wantOutSub: "ok (fetched)",
			wantOnDisk: valid,
			wantHits:   1,
		},
		"success: idempotent when the destination is already valid": {
			existing:   valid,
			wantOutSub: "ok (cached)",
			wantOnDisk: valid,
			wantHits:   0,
		},
		"success: corrupt destination is repaired, not rejected": {
			existing:   []byte("truncated"),
			served:     valid,
			wantOutSub: "CORPUS-REPAIRED:",
			wantOnDisk: valid,
			wantHits:   1,
		},
		"error: wrong downloaded size is CORPUS-INVALID": {
			served:     valid[:len(valid)-1],
			wantErrSub: "CORPUS-INVALID:",
			wantOnDisk: nil,
			wantHits:   1,
		},
		"error: correct size with wrong digest is CORPUS-INVALID": {
			served:     sameLength,
			wantErrSub: "CORPUS-INVALID:",
			wantOnDisk: nil,
			wantHits:   1,
		},
		"error: transport failure exhausts retries as CORPUS-UNAVAILABLE": {
			served:     nil,
			wantErrSub: "CORPUS-UNAVAILABLE:",
			wantOnDisk: nil,
			wantHits:   3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			zeroBackoff(t)
			var hits atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				if tt.served == nil {
					http.NotFound(w, r)
					return
				}
				w.Write(tt.served)
			}))
			defer server.Close()

			root := t.TempDir()
			entry := corpusEntry{
				relPath: "corpus/fixture.txt",
				url:     server.URL + "/fixture.txt",
				size:    int64(len(valid)),
				sha256:  validSum,
			}
			dst := filepath.Join(root, "corpus", "fixture.txt")
			if tt.existing != nil {
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dst, tt.existing, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			var out bytes.Buffer
			err := provision(root, entry, server.Client(), &out)

			if tt.wantErrSub == "" {
				if err != nil {
					t.Fatalf("provision() error = %v, want success; output:\n%s", err, out.String())
				}
			} else {
				if err == nil {
					t.Fatalf("provision() succeeded, want error containing %q; output:\n%s", tt.wantErrSub, out.String())
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("provision() error = %v, want substring %q", err, tt.wantErrSub)
				}
				if strings.Contains(err.Error(), "CORPUS-REPAIRED") {
					t.Fatalf("provision() error = %v: REPAIRED is a success outcome, never an error", err)
				}
			}
			if tt.wantOutSub != "" && !strings.Contains(out.String(), tt.wantOutSub) {
				t.Fatalf("provision() output = %q, want substring %q", out.String(), tt.wantOutSub)
			}
			if tt.wantErrSub != "" && strings.Contains(out.String(), "CORPUS-REPAIRED") {
				t.Fatalf("provision() output = %q: failed run must not claim CORPUS-REPAIRED", out.String())
			}

			data, readErr := os.ReadFile(dst)
			if tt.wantOnDisk == nil {
				if readErr == nil {
					t.Fatalf("destination %s exists with %d bytes, want absent", dst, len(data))
				}
			} else {
				if readErr != nil {
					t.Fatalf("destination %s: %v, want %d verified bytes", dst, readErr, len(tt.wantOnDisk))
				}
				if !bytes.Equal(data, tt.wantOnDisk) {
					t.Fatalf("destination %s holds %d unexpected bytes", dst, len(data))
				}
			}

			// A failed verification must never leave a stray temp file behind,
			// and an unverified temp must never be promoted to the destination.
			stray, err := filepath.Glob(filepath.Join(filepath.Dir(dst), ".fetch-corpora-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(stray) != 0 {
				t.Fatalf("stray temp files left behind: %v", stray)
			}

			if tt.wantHits >= 0 && hits.Load() != tt.wantHits {
				t.Fatalf("server hits = %d, want %d", hits.Load(), tt.wantHits)
			}
		})
	}
}

func TestProvisionIdempotentDoesNotRewrite(t *testing.T) {
	valid := []byte("frozen corpus fixture payload\n")
	root := t.TempDir()
	dst := filepath.Join(root, "corpus", "fixture.txt")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, valid, 0o644); err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(dst, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	entry := corpusEntry{
		relPath: "corpus/fixture.txt",
		url:     "http://127.0.0.1:0/never-dialed",
		size:    int64(len(valid)),
		sha256:  fmt.Sprintf("%x", sha256.Sum256(valid)),
	}
	var out bytes.Buffer
	if err := provision(root, entry, http.DefaultClient, &out); err != nil {
		t.Fatalf("provision() error = %v, want cached success", err)
	}
	if !strings.Contains(out.String(), "ok (cached)") {
		t.Fatalf("provision() output = %q, want ok (cached)", out.String())
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(stamp) {
		t.Fatalf("destination mtime = %v, want untouched %v", info.ModTime(), stamp)
	}
}

func TestRunWritesChecksumsOnlyIfAbsent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, filepath.FromSlash(checksumsRelPath))

	if err := run(root, nil, http.DefaultClient, &bytes.Buffer{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != checksumsContent {
		t.Fatalf("SHA256SUMS = %q, want the frozen constants verbatim", data)
	}

	sentinel := []byte("sentinel: pre-existing checksums are never rewritten\n")
	if err := os.WriteFile(path, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(root, nil, http.DefaultClient, &bytes.Buffer{}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, sentinel) {
		t.Fatalf("SHA256SUMS = %q, want untouched sentinel", data)
	}
}

func TestFrozenCorporaMatchChecksumsFile(t *testing.T) {
	for _, entry := range frozenCorpora {
		rel, ok := strings.CutPrefix(entry.relPath, ".omx/artifacts/phase0/benchmark-corpora/")
		if !ok {
			t.Fatalf("entry %s is outside the frozen corpus tree", entry.relPath)
		}
		line := entry.sha256 + "  " + rel + "\n"
		if !strings.Contains(checksumsContent, line) {
			t.Fatalf("checksumsContent is missing the frozen line %q", line)
		}
	}
	if got, want := strings.Count(checksumsContent, "\n"), len(frozenCorpora); got != want {
		t.Fatalf("checksumsContent has %d lines, want %d", got, want)
	}
}
