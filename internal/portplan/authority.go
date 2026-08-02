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
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

const archsimdSourceAggregateV1 = "68b4d22b88b577530e288ac976782e0bed772dae279d5e4e4d9ed6d73f8af419"

var (
	upstreamAuthorityHeaderV1 = [...]string{"schema_version", "remote_url", "commit", "tree", "parent", "clean", "archive_format", "archive_sha256", "archive_recipe", "evidence_anchor"}
	goBaseAuthorityHeaderV1   = [...]string{"schema_version", "repository", "commit", "tree", "parent", "clean", "archive_format", "archive_sha256", "archive_recipe", "evidence_anchor"}
	hostAuthorityHeaderV1     = [...]string{"schema_version", "host_id", "transport", "os", "kernel", "arch", "cpu_model", "logical_cpu_count", "cpu_feature_digest_or_na", "go_version", "go_binary_sha256", "identity_fact_sha256", "evidence_anchor"}
	archsimdSourceHeaderV1    = [...]string{"schema_version", "go_toolchain", "goos", "goarch", "ordinal", "relative_path", "sha256"}
	archsimdPrimitiveHeaderV1 = [...]string{"mapping_version", "family_contract_display_id", "go_toolchain", "archsimd_source_digest", "required_primitives", "available_primitives", "audit_outcome", "evidence_anchor"}
	corpusContractHeaderV1    = [...]string{"schema_version", "ordinal", "corpus_id", "state", "element_type", "size_units", "byte_length_or_pending", "sha256_or_pending", "source_identity", "recipe", "family_contracts"}
	frozenInputsHeaderV1      = [...]string{"schema_version", "path", "size", "sha256"}
)

type (
	UpstreamAuthorityV1 struct{ RemoteURL, Commit, Tree, Parent, ArchiveSHA256, ArchiveRecipe, EvidenceAnchor string }
	GoBaseAuthorityV1   struct {
		Repository, Commit, Tree, Parent, ArchiveSHA256, ArchiveRecipe, EvidenceAnchor string
	}
)

type HostAuthorityV1 struct {
	HostID, Transport, OS, Kernel, Arch, CPUModel                                       string
	LogicalCPUCount                                                                     int
	CPUFeatureDigestOrNA, GoVersion, GoBinarySHA256, IdentityFactSHA256, EvidenceAnchor string
}
type ArchsimdSourceFileV1 struct {
	Ordinal              int
	RelativePath, SHA256 string
}
type (
	ArchsimdPrimitiveV1    struct{ FamilyContractDisplayID, RequiredPrimitives, AvailablePrimitives, EvidenceAnchor string }
	CorpusContractRecordV1 struct {
		Ordinal                                                                                                                int
		CorpusID, State, ElementType, SizeUnits, ByteLengthOrPending, SHA256OrPending, SourceIdentity, Recipe, FamilyContracts string
	}
)

type FrozenInputV1 struct {
	Path   string
	Size   int
	SHA256 string
}

// ParseUpstreamAuthorityV1 parses the single immutable upstream receipt.
func ParseUpstreamAuthorityV1(data []byte) (UpstreamAuthorityV1, error) {
	lines, err := parseLines(data, "upstream authority")
	if err != nil {
		return UpstreamAuthorityV1{}, err
	}
	if len(lines) != 2 {
		return UpstreamAuthorityV1{}, fmt.Errorf("upstream authority: got %d data rows, want 1", len(lines)-1)
	}
	if err := validateHeader(lines[0], upstreamAuthorityHeaderV1[:], "upstream authority"); err != nil {
		return UpstreamAuthorityV1{}, err
	}
	f, err := parseRecord(lines[1], len(upstreamAuthorityHeaderV1), 2, "upstream authority")
	if err != nil {
		return UpstreamAuthorityV1{}, err
	}
	if f[0] != "v1" || f[1] != "https://github.com/simdutf/simdutf.git" || f[2] != "611becc2a08c27a4edc77d9a45ff74c97130129b" || f[3] != "c8292790d793212ca0a1faf6ae42e7f8e7b70d4f" || f[4] != "c7bef0ff14a13fd6ea52e3347da2c659383392de" || f[5] != "true" || f[6] != "git-tar" || f[7] != "aab5aabfbe2da400047b06626210853c1396a989bc412a5a58d95c4df68a4e7d" || f[8] != "git archive --format=tar 611becc2a08c27a4edc77d9a45ff74c97130129b" || f[9] == "" {
		return UpstreamAuthorityV1{}, fmt.Errorf("upstream authority: noncanonical receipt")
	}
	return UpstreamAuthorityV1{f[1], f[2], f[3], f[4], f[7], f[8], f[9]}, nil
}

// ParseGoBaseAuthorityV1 parses the immutable clean Go integration-base receipt.
func ParseGoBaseAuthorityV1(data []byte) (GoBaseAuthorityV1, error) {
	lines, err := parseLines(data, "go base authority")
	if err != nil {
		return GoBaseAuthorityV1{}, err
	}
	if len(lines) != 2 {
		return GoBaseAuthorityV1{}, fmt.Errorf("go base authority: got %d data rows, want 1", len(lines)-1)
	}
	if err := validateHeader(lines[0], goBaseAuthorityHeaderV1[:], "go base authority"); err != nil {
		return GoBaseAuthorityV1{}, err
	}
	f, err := parseRecord(lines[1], len(goBaseAuthorityHeaderV1), 2, "go base authority")
	if err != nil {
		return GoBaseAuthorityV1{}, err
	}
	if f[0] != "v1" ||
		f[1] != "https://github.com/zchee/go-simdutf.git" ||
		f[2] != "083f5bc47a010626612e05378ae31a3359f904ad" ||
		f[3] != "b142a6d7eb3da840a805b2b316e7f085fba062ba" ||
		f[4] != "b813f7ab4c4974b8188afa691c1d81d14f3ae8a1" ||
		f[5] != "true" ||
		f[6] != "git-tar" ||
		f[7] != "deab8255169241cac1fe3f589faa5c20f4f238c742320f4f8c0e8096a8f8f965" ||
		f[8] != "git archive --format=tar 083f5bc47a010626612e05378ae31a3359f904ad" ||
		f[9] != "git:083f5bc47a010626612e05378ae31a3359f904ad" {
		return GoBaseAuthorityV1{}, errors.New("go base authority: noncanonical receipt")
	}
	return GoBaseAuthorityV1{
		Repository: f[1], Commit: f[2], Tree: f[3], Parent: f[4],
		ArchiveSHA256: f[7], ArchiveRecipe: f[8], EvidenceAnchor: f[9],
	}, nil
}

// ParseHostAuthorityV1 parses host identities and binds them to locked sets.
func ParseHostAuthorityV1(data []byte, locked map[string][]LockedSetRecordV1) ([]HostAuthorityV1, error) {
	lines, err := parseLines(data, "host authority")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], hostAuthorityHeaderV1[:], "host authority"); err != nil {
		return nil, err
	}
	if len(lines) != 3 {
		return nil, fmt.Errorf("host authority: got %d data rows, want 2", len(lines)-1)
	}
	expected := [][]string{{"darwin-arm64-apple-m3-max", "local-physical", "darwin", "27.0.0", "arm64", "Apple M3 Max", "16", "not_applicable:apple_arm64_feature_selection_is_by_hw_optional_sysctl", "go1.26.5 darwin/arm64", "3925fc3221ac440ebf7c35361ff663bed0c7bdb2e0a157b75fe993607ffe0a19", "efdd8f59951f85885536e93900158f98a5184abccde69df71799e4b2a92a1c4e"}, {"linux-amd64-debian-13-xeon-platinum-8481c", "ssh:debian-13-trixie.gaudiy-platform", "linux", "6.12.90+deb13.1-cloud-amd64", "amd64", "Intel(R) Xeon(R) Platinum 8481C CPU @ 2.70GHz", "44", "9251032637a94bda0db0430f71022e0c43a5188f103e9c48831ba58a4a4c68fa", "go1.26.5 linux/amd64", "8da5fd321795754b994c64e3eb8a5a14ff47bd285559a7e876f3c79abafc67f9", "a19d706a49acc40b78beb9b22ea5a858102a6e2730133815843140be656d62da"}}
	out := make([]HostAuthorityV1, 0, 2)
	for i, line := range lines[1:] {
		f, e := parseRecord(line, len(hostAuthorityHeaderV1), i+2, "host authority")
		if e != nil {
			return nil, e
		}
		want := expected[i]
		if f[0] != "v1" || f[1] != want[0] || f[2] != want[1] || f[3] != want[2] || f[4] != want[3] || f[5] != want[4] || f[6] != want[5] || f[7] != want[6] || f[8] != want[7] || f[9] != want[8] || f[10] != want[9] || f[11] != want[10] || f[12] == "" {
			return nil, fmt.Errorf("host authority line %d: identity drift", i+2)
		}
		cpu, e := canonicalPositive(f[7])
		if e != nil {
			return nil, fmt.Errorf("host authority line %d: CPU count: %w", i+2, e)
		}
		out = append(out, HostAuthorityV1{f[1], f[2], f[3], f[4], f[5], f[6], cpu, f[8], f[9], f[10], f[11], f[12]})
	}
	for set, values := range map[string][]string{"authoritative_host": {expected[0][0], expected[1][0]}, "official_architecture": {"amd64", "arm64"}} {
		got := locked[set]
		if len(got) != len(values) {
			return nil, fmt.Errorf("host authority: locked %s count", set)
		}
		for i, want := range values {
			raw, e := hex.DecodeString(got[i].ValueHex)
			if e != nil || string(raw) != want {
				return nil, fmt.Errorf("host authority: locked %s drift", set)
			}
		}
	}
	return out, nil
}

// ParseArchsimdSourceFilesV1 parses and recomputes the reviewed source aggregate.
func ParseArchsimdSourceFilesV1(data []byte) ([]ArchsimdSourceFileV1, error) {
	lines, err := parseLines(data, "archsimd source files")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], archsimdSourceHeaderV1[:], "archsimd source files"); err != nil {
		return nil, err
	}
	if len(lines) != 17 {
		return nil, fmt.Errorf("archsimd source files: got %d data rows, want 16", len(lines)-1)
	}
	out := make([]ArchsimdSourceFileV1, 0, 16)
	fields := []string{"archsimd-source-v1", "go1.26.5", "linux", "amd64"}
	for i, line := range lines[1:] {
		f, e := parseRecord(line, 7, i+2, "archsimd source files")
		if e != nil {
			return nil, e
		}
		n, e := canonicalPositive(f[4])
		if e != nil || n != i+1 || f[0] != "v1" || f[1] != "go1.26.5" || f[2] != "linux" || f[3] != "amd64" || !safeTopFile(f[5]) || !lowerDigest(f[6]) {
			return nil, fmt.Errorf("archsimd source files line %d: invalid record", i+2)
		}
		out = append(out, ArchsimdSourceFileV1{n, f[5], f[6]})
		fields = append(fields, f[5], f[6])
	}
	if SHA256Hex(EncodeTupleV1(fields...)) != archsimdSourceAggregateV1 {
		return nil, fmt.Errorf("archsimd source files: aggregate mismatch")
	}
	return out, nil
}

// ValidateArchsimdPrimitivesV1 validates family capability receipts against sources and mappings.
func ValidateArchsimdPrimitivesV1(data []byte, sources []ArchsimdSourceFileV1, reviewed []ReviewedMappingV1) ([]ArchsimdPrimitiveV1, error) {
	if len(sources) != 16 || sourceAggregate(sources) != archsimdSourceAggregateV1 {
		return nil, fmt.Errorf("archsimd primitives: invalid source receipt")
	}
	lines, err := parseLines(data, "archsimd primitives")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], archsimdPrimitiveHeaderV1[:], "archsimd primitives"); err != nil {
		return nil, err
	}
	if len(lines) != 9 {
		return nil, fmt.Errorf("archsimd primitives: got %d data rows, want 8", len(lines)-1)
	}
	families := []string{"FC-v1-helper-validation", "FC-v1-latin1-source", "FC-v1-utf8-source", "FC-v1-utf16-source", "FC-v1-utf32-source", "FC-v1-detection", "FC-v1-find", "FC-v1-base64"}
	out := make([]ArchsimdPrimitiveV1, 0, 8)
	for i, line := range lines[1:] {
		f, e := parseRecord(line, 8, i+2, "archsimd primitives")
		if e != nil {
			return nil, e
		}
		if f[0] != "v1" || f[1] != families[i] || f[2] != "Go 1.26.5" || f[3] != archsimdSourceAggregateV1 || f[4] == "" || f[5] == "" || f[6] != "eligible" || !strings.Contains(f[7], "archsimd-source-files-v1.tsv") || !strings.Contains(f[7], "aggregate=sha256(EncodeTupleV1(domain,toolchain,goos,goarch,path,digest...))") {
			return nil, fmt.Errorf("archsimd primitives line %d: invalid receipt", i+2)
		}
		out = append(out, ArchsimdPrimitiveV1{f[1], f[4], f[5], f[7]})
	}
	for _, m := range reviewed {
		if m.ISAOrdinalOrScalar == "scalar" {
			continue
		}
		x := m.Backends[2]
		if x.Outcome == "eligible" || x.NAReason == "primitive_gap" {
			found := false
			for _, p := range out {
				if p.FamilyContractDisplayID == m.FamilyContractDisplayID && strings.Contains(p.EvidenceAnchor, m.FamilyContractDisplayID) {
					found = true
					break
				}
			}
			if !found || (x.Outcome == "eligible" && x.EvidenceAnchor == "") || (x.NAReason == "primitive_gap" && x.EvidenceAnchor == "") {
				return nil, fmt.Errorf("archsimd primitives: mapping audit join failed")
			}
		}
	}
	return out, nil
}

// ParseCorpusContractV1 parses and verifies the frozen/deferred corpus boundary.
func ParseCorpusContractV1(data []byte) ([]CorpusContractRecordV1, error) {
	lines, err := parseLines(data, "corpus contract")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], corpusContractHeaderV1[:], "corpus contract"); err != nil {
		return nil, err
	}
	if len(lines) != 18 {
		return nil, fmt.Errorf("corpus contract: got %d data rows, want 17", len(lines)-1)
	}
	ids := []string{"Q-byte-zero", "Q-u16-zero", "Q-u32-zero", "Q-emoji", "Q-arabic-lipsum", "Q-latin1-ramp", "Q-find-byte", "Q-find-u16le", "Q-detection-valid", "Q-dns-source", "Q-dns-normalized", "Q-emoji-utf16le", "Q-emoji-utf16be", "Q-emoji-utf16-native", "Q-emoji-utf32-native", "Q-find-cpp-capture", "Q-towellformed-cpp-capture"}
	states := []string{"frozen_deterministic", "frozen_deterministic", "frozen_deterministic", "frozen_literal", "frozen_external", "frozen_deterministic", "frozen_deterministic", "frozen_deterministic", "frozen_deterministic", "frozen_external", "frozen_derived", "deferred_until_scalar_publication", "deferred_until_scalar_publication", "deferred_until_scalar_publication", "deferred_until_scalar_publication", "deferred_phase13_companion", "deferred_phase13_companion"}
	frozenFamilies := map[string]bool{"FC-v1-helper-validation": true, "FC-v1-latin1-source": true, "FC-v1-utf8-source": true, "FC-v1-utf16-source": true, "FC-v1-utf32-source": true, "FC-v1-detection": true, "FC-v1-find": true, "FC-v1-base64": true}
	out := make([]CorpusContractRecordV1, 0, 17)
	for i, line := range lines[1:] {
		f, e := parseRecord(line, 11, i+2, "corpus contract")
		if e != nil {
			return nil, e
		}
		n, e := canonicalPositive(f[1])
		if e != nil || n != i+1 || f[0] != "v1" || f[2] != ids[i] || f[3] != states[i] || f[4] == "" || f[8] == "" || f[9] == "" || f[10] == "" {
			return nil, fmt.Errorf("corpus contract line %d: invalid identity", i+2)
		}
		deferred := strings.HasPrefix(f[3], "deferred_")
		if deferred {
			validUnits := f[5] == "pending"
			if i == 15 {
				validUnits = f[5] == "10001"
			}
			if i < 11 || i > 16 || !validUnits || f[6] != "pending" || f[7] != "pending" {
				return nil, fmt.Errorf("corpus contract line %d: invalid deferred matrix", i+2)
			}
		} else {
			units, e1 := canonicalPositive(f[5])
			bytes, e2 := canonicalPositive(f[6])
			if e1 != nil || e2 != nil || !lowerDigest(f[7]) || units < 1 || bytes < 1 {
				return nil, fmt.Errorf("corpus contract line %d: invalid frozen values", i+2)
			}
		}
		for family := range strings.SplitSeq(f[10], ";") {
			if !frozenFamilies[family] {
				return nil, fmt.Errorf("corpus contract line %d: unknown family", i+2)
			}
		}
		out = append(out, CorpusContractRecordV1{n, f[2], f[3], f[4], f[5], f[6], f[7], f[8], f[9], f[10]})
	}
	if err := validateCorpusFixtures(out); err != nil {
		return nil, err
	}
	return out, nil
}

// RenderFrozenInputsV1 renders a deterministic receipt over raw immutable files.
func RenderFrozenInputsV1(files map[string][]byte) ([]byte, string, error) {
	if len(files) == 0 {
		return nil, "", fmt.Errorf("frozen inputs: empty file set")
	}
	paths := make([]string, 0, len(files))
	for p := range files {
		if !safeReceiptPath(p) {
			return nil, "", fmt.Errorf("frozen inputs: invalid path %q", p)
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	records := make([]FrozenInputV1, 0, len(paths))
	lines := []string{strings.Join(frozenInputsHeaderV1[:], "\t")}
	for _, p := range paths {
		record := FrozenInputV1{Path: p, Size: len(files[p]), SHA256: SHA256Hex(files[p])}
		records = append(records, record)
		lines = append(lines, "v1\t"+record.Path+"\t"+strconv.Itoa(record.Size)+"\t"+record.SHA256)
	}
	tuple, err := CanonicalFrozenAuthorityTupleV1(records)
	if err != nil {
		return nil, "", err
	}
	return []byte(strings.Join(lines, "\n") + "\n"), SHA256Hex(tuple), nil
}

// CanonicalFrozenAuthorityTupleV1 returns the exact bytes that identify a
// canonical frozen-input authority receipt.
func CanonicalFrozenAuthorityTupleV1(records []FrozenInputV1) ([]byte, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("frozen inputs: empty file set")
	}
	fields := make([]string, 1, 1+len(records)*3)
	fields[0] = "frozen-inputs-v1"
	previous := ""
	for _, record := range records {
		if !safeReceiptPath(record.Path) || record.Path <= previous || record.Size < 0 || !lowerDigest(record.SHA256) {
			return nil, fmt.Errorf("frozen inputs: noncanonical authority tuple")
		}
		fields = append(fields, record.Path, strconv.Itoa(record.Size), record.SHA256)
		previous = record.Path
	}
	return EncodeTupleV1(fields...), nil
}

// ParseFrozenInputsV1 validates a receipt against the raw file map and returns its aggregate.
func ParseFrozenInputsV1(receipt []byte, files map[string][]byte) ([]FrozenInputV1, string, error) {
	lines, err := parseLines(receipt, "frozen inputs")
	if err != nil {
		return nil, "", err
	}
	if err = validateHeader(lines[0], frozenInputsHeaderV1[:], "frozen inputs"); err != nil {
		return nil, "", err
	}
	if len(lines)-1 != len(files) || len(files) == 0 {
		return nil, "", fmt.Errorf("frozen inputs: file count mismatch")
	}
	out := make([]FrozenInputV1, 0, len(files))
	seen := map[string]bool{}
	for i, line := range lines[1:] {
		f, e := parseRecord(line, 4, i+2, "frozen inputs")
		if e != nil {
			return nil, "", e
		}
		raw, exists := files[f[1]]
		n, e := canonicalPositive(f[2])
		if e != nil || f[0] != "v1" || !safeReceiptPath(f[1]) || seen[f[1]] || !exists || n != len(raw) || f[3] != SHA256Hex(raw) {
			return nil, "", fmt.Errorf("frozen inputs line %d: mismatch", i+2)
		}
		seen[f[1]] = true
		out = append(out, FrozenInputV1{f[1], n, f[3]})
	}
	tuple, err := CanonicalFrozenAuthorityTupleV1(out)
	if err != nil {
		return nil, "", err
	}
	return out, SHA256Hex(tuple), nil
}

func lowerDigest(s string) bool {
	if len(s) != 64 || strings.ToLower(s) != s {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func safeTopFile(p string) bool {
	return safeReceiptPath(p) && !strings.Contains(p, "/") && (strings.HasSuffix(p, ".go") || strings.HasSuffix(p, ".s"))
}

func safeReceiptPath(p string) bool {
	return p != "" && !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, ".") && path.Clean(p) == p && !strings.Contains(p, "\\") && !strings.Contains(p, "\t") && !strings.Contains(p, "\n") && !strings.HasPrefix(p, "internal/portplan/") && !strings.Contains(p, "generated")
}

func sourceAggregate(rows []ArchsimdSourceFileV1) string {
	f := []string{"archsimd-source-v1", "go1.26.5", "linux", "amd64"}
	for i, r := range rows {
		if r.Ordinal != i+1 || !safeTopFile(r.RelativePath) || !lowerDigest(r.SHA256) {
			return ""
		}
		f = append(f, r.RelativePath, r.SHA256)
	}
	return SHA256Hex(EncodeTupleV1(f...))
}

func validateCorpusFixtures(rows []CorpusContractRecordV1) error {
	zero := make([]byte, 4096)
	ramp := make([]byte, 4096)
	alpha := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	find := make([]byte, 4096)
	for i := range ramp {
		ramp[i] = byte(i)
		find[i] = alpha[i%len(alpha)]
	}
	checks := map[string]string{"Q-byte-zero": SHA256Hex(zero), "Q-u16-zero": SHA256Hex(zero), "Q-u32-zero": SHA256Hex(zero), "Q-emoji": "d6484d359bff183e4d6a4d20b3cc7056c55f372011f28b21b06462ba4d643523", "Q-arabic-lipsum": "b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb", "Q-latin1-ramp": SHA256Hex(ramp), "Q-find-byte": SHA256Hex(find), "Q-detection-valid": SHA256Hex(find), "Q-dns-source": "d837553e61f96476a75350142085983b366beb9754ed6f21ff9299878d2a1de0", "Q-dns-normalized": "79f1eba2fe0c187f1086f7534b74cd1dd4ef795a515d7db13d613eebafdb1d6f"}
	sizes := map[string]string{"Q-byte-zero": "4096", "Q-u16-zero": "4096", "Q-u32-zero": "4096", "Q-emoji": "3150", "Q-arabic-lipsum": "81685", "Q-latin1-ramp": "4096", "Q-find-byte": "4096", "Q-detection-valid": "4096", "Q-dns-source": "35100000", "Q-dns-normalized": "35000000"}
	u16 := make([]byte, 4096)
	for i, b := range find[:2048] {
		u16[2*i] = b
	}
	checks["Q-find-u16le"] = SHA256Hex(u16)
	sizes["Q-find-u16le"] = "4096"
	for _, r := range rows {
		if want, ok := checks[r.CorpusID]; ok && (r.SHA256OrPending != want || r.ByteLengthOrPending != sizes[r.CorpusID]) {
			return fmt.Errorf("corpus contract: fixture %s mismatch", r.CorpusID)
		}
	}
	return nil
}
