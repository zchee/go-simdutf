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

// Command fetch-corpora provisions the two frozen Phase 0 benchmark corpus
// blobs by fetching them from their pinned upstream commits and verifying
// every byte against the frozen sizes and SHA-256 digests recorded in
// docs/porting/corpus-freeze.md. The blobs are never redistributed by this
// project — the corpus freeze defers a redistribution review — so the only
// provisioning path is fetch-from-origin with identity verification: the URL
// is a transport, the hash is the contract.
//
// Outcomes are typed and greppable:
//
//   - "CORPUS-INVALID:" — downloaded bytes failed the size or SHA-256 check.
//     A supply-chain signal, never retried; the process exits non-zero.
//   - "CORPUS-UNAVAILABLE:" — the transport failed after 3 attempts with
//     backoff. An infrastructure failure, not a code regression; exits
//     non-zero.
//   - "CORPUS-REPAIRED:" — a local destination file existed with the wrong
//     size or digest, was deleted, re-fetched, and verified. A normal,
//     successful outcome (exit 0): a truncated cache entry must self-heal or
//     a bad cache would wedge CI permanently.
package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// corpusEntry describes one frozen corpus blob. The sizes and SHA-256 values
// are the frozen identities from docs/porting/corpus-freeze.md; the URLs pin
// immutable upstream commit SHAs, never branch names.
type corpusEntry struct {
	relPath string
	url     string
	size    int64
	sha256  string
}

var frozenCorpora = []corpusEntry{
	{
		relPath: ".omx/artifacts/phase0/benchmark-corpora/corpus/unicode_lipsum/lipsum/Arabic-Lipsum.utf8.txt",
		url:     "https://raw.githubusercontent.com/lemire/unicode_lipsum/b0f1d0c1cb0cb168fc08dbf0e3b7100cdec517dc/lipsum/Arabic-Lipsum.utf8.txt",
		size:    81685,
		sha256:  "b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb",
	},
	{
		relPath: ".omx/artifacts/phase0/benchmark-corpora/corpus/base64data/dns/swedenzonebase.txt",
		url:     "https://raw.githubusercontent.com/lemire/base64data/3e89f846f15b04e7ee16713984dbdd9d8b5928d4/dns/swedenzonebase.txt",
		size:    35100000,
		sha256:  "d837553e61f96476a75350142085983b366beb9754ed6f21ff9299878d2a1de0",
	},
}

// checksumsRelPath sits one level above the cached corpus/ directory, so it is
// absent on a fresh runner whether or not the cache hit.
const checksumsRelPath = ".omx/artifacts/phase0/benchmark-corpora/SHA256SUMS"

// checksumsContent reproduces the frozen SHA256SUMS verbatim. The constants
// are deliberately not recomputed from the downloaded bytes — recomputation
// would make the checksum file self-confirming and worthless.
const checksumsContent = "d837553e61f96476a75350142085983b366beb9754ed6f21ff9299878d2a1de0  corpus/base64data/dns/swedenzonebase.txt\n" +
	"b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb  corpus/unicode_lipsum/lipsum/Arabic-Lipsum.utf8.txt\n"

// retryBackoff is the wait before each fetch attempt; its length is the
// attempt budget. Transport failures (connection errors, non-200 statuses)
// walk through it; verification failures never do.
var retryBackoff = []time.Duration{0, 1 * time.Second, 2 * time.Second}

func main() {
	dest := flag.String("dest", "", "destination root; defaults to the repository root containing go.mod")
	flag.Parse()

	root := *dest
	if root == "" {
		var err error
		root, err = findRepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch-corpora: %v\n", err)
			os.Exit(1)
		}
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	if err := run(root, frozenCorpora, client, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

// findRepoRoot walks up from the working directory to the directory
// containing go.mod, so the tool runs from anywhere inside the module.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in any parent directory; pass -dest")
		}
		dir = parent
	}
}

// run provisions every entry under root and writes the frozen SHA256SUMS if
// absent. It stops at the first CORPUS-INVALID or CORPUS-UNAVAILABLE failure.
func run(root string, entries []corpusEntry, client *http.Client, out io.Writer) error {
	for _, entry := range entries {
		if err := provision(root, entry, client, out); err != nil {
			return err
		}
	}
	return writeChecksumsIfAbsent(root)
}

// provision brings one destination to its frozen identity: verified files are
// left alone, corrupt files are repaired, missing files are fetched. Every
// installed byte has passed the size and SHA-256 checks.
func provision(root string, entry corpusEntry, client *http.Client, out io.Writer) error {
	dst := filepath.Join(root, filepath.FromSlash(entry.relPath))
	verifyErr := verifyFile(dst, entry.size, entry.sha256)
	switch {
	case verifyErr == nil:
		fmt.Fprintf(out, "ok (cached): %s\n", entry.relPath)
		return nil
	case errors.Is(verifyErr, os.ErrNotExist):
		if err := fetchInto(dst, entry, client); err != nil {
			return err
		}
		fmt.Fprintf(out, "ok (fetched): %s\n", entry.relPath)
		return nil
	default:
		// A wrong-sized or wrong-digest *local* file self-heals: delete and
		// re-fetch. This is CORPUS-REPAIRED, not CORPUS-INVALID — the invalid
		// classification is reserved for freshly downloaded bytes.
		if err := os.Remove(dst); err != nil {
			return fmt.Errorf("fetch-corpora: removing corrupt %s: %w", dst, err)
		}
		if err := fetchInto(dst, entry, client); err != nil {
			return err
		}
		fmt.Fprintf(out, "CORPUS-REPAIRED: %s: %v; re-fetched and verified\n", entry.relPath, verifyErr)
		return nil
	}
}

// verifyFile checks that the file at path has exactly size bytes and the
// frozen SHA-256. The returned error wraps os.ErrNotExist when the file is
// absent and otherwise states expected-vs-actual.
func verifyFile(path string, size int64, wantSum string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() != size {
		return fmt.Errorf("size = %d, want %d", info.Size(), size)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != wantSum {
		return fmt.Errorf("SHA-256 = %s, want %s", got, wantSum)
	}
	return nil
}

// fetchInto downloads entry.url to a temp file next to dst, verifies size then
// SHA-256, and renames it into place. Nothing unverified is ever left at dst.
func fetchInto(dst string, entry corpusEntry, client *http.Client) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("fetch-corpora: %w", err)
	}

	var lastErr error
	for _, backoff := range retryBackoff {
		time.Sleep(backoff)
		invalid, err := fetchAttempt(dst, entry, client)
		if err == nil {
			return nil
		}
		if invalid {
			// A post-download digest or size mismatch is a supply-chain
			// signal, not a flake: never retried.
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("CORPUS-UNAVAILABLE: %s: %d attempts failed, last error: %v",
		entry.url, len(retryBackoff), lastErr)
}

// fetchAttempt performs one download-verify-install cycle. The bool reports
// whether the failure is a verification failure (CORPUS-INVALID, not
// retryable) as opposed to a transport failure (retryable).
func fetchAttempt(dst string, entry corpusEntry, client *http.Client) (invalid bool, err error) {
	resp, err := client.Get(entry.url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("GET %s: %s", entry.url, resp.Status)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), ".fetch-corpora-*")
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hash), resp.Body)
	if err != nil {
		return false, err
	}
	if written != entry.size {
		return true, fmt.Errorf("CORPUS-INVALID: %s: downloaded size = %d, want %d",
			entry.relPath, written, entry.size)
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != entry.sha256 {
		return true, fmt.Errorf("CORPUS-INVALID: %s: downloaded SHA-256 = %s, want %s",
			entry.relPath, got, entry.sha256)
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return false, err
	}
	return false, nil
}

// writeChecksumsIfAbsent emits the frozen SHA256SUMS next to the corpus so
// `shasum -a 256 -c SHA256SUMS` works on a fresh runner. An existing file —
// the canonical one on a developer clone — is never touched.
func writeChecksumsIfAbsent(root string) error {
	path := filepath.Join(root, filepath.FromSlash(checksumsRelPath))
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("fetch-corpora: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("fetch-corpora: %w", err)
	}
	if err := os.WriteFile(path, []byte(checksumsContent), 0o644); err != nil {
		return fmt.Errorf("fetch-corpora: %w", err)
	}
	return nil
}
