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
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

var (
	dependencyHeaderV1 = [...]string{"mapping_version", "consumer_row_key_v1", "consumer_manifest_ordinal", "consumer_go_symbol", "dependency_kind", "owner_kind", "owner_logical_id", "owner_go_symbol_or_empty", "evidence_anchor"}
	lockedSetHeaderV1  = [...]string{"schema_version", "set_name", "ordinal", "value_hex", "evidence_anchor"}
)

// DependencyRecordV1 describes one reviewed implementation dependency.
type DependencyRecordV1 struct {
	ConsumerRowKeyV1        string
	ConsumerManifestOrdinal int
	ConsumerGoSymbol        string
	DependencyKind          string
	OwnerKind               string
	OwnerLogicalID          string
	OwnerGoSymbolOrEmpty    string
	EvidenceAnchor          string
}

// LockedSetRecordV1 is one ordered, hex-encoded locked value.
type LockedSetRecordV1 struct {
	SetName        string
	Ordinal        int
	ValueHex       string
	EvidenceAnchor string
}

func DependencyHeaderV1() []string { return append([]string(nil), dependencyHeaderV1[:]...) }
func LockedSetHeaderV1() []string  { return append([]string(nil), lockedSetHeaderV1[:]...) }

// ParseDependencyMapV1 parses the frozen dependency map and independently
// recomputes its wrapper and shared-kernel edges from the reviewed mapping.
func ParseDependencyMapV1(data []byte, planned []ManifestRowV1, reviewed []ReviewedMappingV1) ([]DependencyRecordV1, error) {
	if err := validateDependencyInputs(planned, reviewed); err != nil {
		return nil, err
	}
	lines, err := parseLines(data, "dependency map")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], dependencyHeaderV1[:], "dependency map"); err != nil {
		return nil, err
	}
	if len(lines)-1 != 185 {
		return nil, fmt.Errorf("dependency map: got %d records, want 185", len(lines)-1)
	}
	index := rowIndex(planned)
	expectedWrapper := expectedWrapperEdges(planned, reviewed)
	expectedCell, err := expectedSharedKernelEdges(planned, reviewed)
	if err != nil {
		return nil, err
	}
	out := make([]DependencyRecordV1, 0, 185)
	perRow := make([]int, len(planned))
	noneCount, wrapperCount, cellCount := 0, 0, 0
	seen := make(map[string]bool, 185)
	gotWrapper := make(map[string]bool, len(expectedWrapper))
	gotCell := make(map[string]bool, len(expectedCell))
	rowEdges := make([][]int, len(planned))
	for n, line := range lines[1:] {
		f, err := parseRecord(line, len(dependencyHeaderV1), n+2, "dependency map")
		if err != nil {
			return nil, err
		}
		ordinal, err := canonicalPositive(f[2])
		if err != nil || ordinal > len(planned) || f[0] != "v1" || f[1] != planned[ordinal-1].RowKeyV1 || f[3] != planned[ordinal-1].Cells[manifestGoSymbolIndex] {
			return nil, fmt.Errorf("dependency map line %d: invalid consumer join", n+2)
		}
		if !validDependencyEvidence(f[8]) {
			return nil, fmt.Errorf("dependency map line %d: invalid pinned evidence anchor", n+2)
		}
		r := DependencyRecordV1{ConsumerRowKeyV1: f[1], ConsumerManifestOrdinal: ordinal, ConsumerGoSymbol: f[3], DependencyKind: f[4], OwnerKind: f[5], OwnerLogicalID: f[6], OwnerGoSymbolOrEmpty: f[7], EvidenceAnchor: f[8]}
		key := dependencyEdgeKey(r)
		if seen[key] {
			return nil, fmt.Errorf("dependency map line %d: duplicate record", n+2)
		}
		seen[key] = true
		consumer := ordinal - 1
		switch r.DependencyKind {
		case "none":
			if r.OwnerKind != "none" || r.OwnerLogicalID != "" || r.OwnerGoSymbolOrEmpty != "" {
				return nil, fmt.Errorf("dependency map line %d: invalid none dependency", n+2)
			}
			noneCount++
		case "wrapper_delegate":
			owner, ok := index[r.OwnerLogicalID]
			if r.OwnerKind != "row" || !ok || owner == consumer || r.OwnerGoSymbolOrEmpty != planned[owner].Cells[manifestGoSymbolIndex] {
				return nil, fmt.Errorf("dependency map line %d: invalid wrapper owner", n+2)
			}
			if !expectedWrapper[key] {
				return nil, fmt.Errorf("dependency map line %d: unexpected wrapper edge", n+2)
			}
			gotWrapper[key] = true
			rowEdges[owner] = append(rowEdges[owner], consumer)
			wrapperCount++
		case "shared_kernel_delegate":
			if r.OwnerKind != "cell" || !expectedCell[key] {
				return nil, fmt.Errorf("dependency map line %d: unexpected shared-kernel edge", n+2)
			}
			cellCount++
			gotCell[key] = true
		default:
			return nil, fmt.Errorf("dependency map line %d: unknown dependency kind", n+2)
		}
		perRow[consumer]++
		out = append(out, r)
	}
	if noneCount != 84 || wrapperCount != 53 || cellCount != 48 || len(gotWrapper) != len(expectedWrapper) || len(gotCell) != len(expectedCell) {
		return nil, fmt.Errorf("dependency map: recomputed edge set or record counts differ")
	}
	consumers := 0
	for i, count := range perRow {
		if count == 0 || (count > 1 && hasNoneDependency(out, planned[i].RowKeyV1)) {
			return nil, fmt.Errorf("dependency map: invalid dependencies for consumer %d", i+1)
		}
		consumers++
	}
	if consumers != 125 || hasCycle(rowEdges) {
		return nil, fmt.Errorf("dependency map: row dependency cycle or consumer count")
	}
	return out, nil
}

func validateDependencyInputs(planned []ManifestRowV1, reviewed []ReviewedMappingV1) error {
	if len(planned) != 125 || len(reviewed) != 125 {
		return fmt.Errorf("dependency map requires 125 planned and reviewed rows")
	}
	seen := make(map[string]bool, len(planned))
	for i, p := range planned {
		if p.PlannedOrdinal != i+1 || p.RowKeyV1 == "" || p.RowKeyV1 != RowKeyV1(manifestComposite(p.Cells)) || seen[p.RowKeyV1] || reviewed[i].ManifestOrdinal != i+1 || reviewed[i].GoSymbol != p.Cells[manifestGoSymbolIndex] {
			return fmt.Errorf("dependency map: invalid planned/reviewed join %d", i+1)
		}
		seen[p.RowKeyV1] = true
	}
	return nil
}

func expectedWrapperEdges(planned []ManifestRowV1, reviewed []ReviewedMappingV1) map[string]bool {
	symbolIndex := indexBySymbol(planned)
	out := map[string]bool{}
	add := func(consumer int, symbol string) {
		if owner, ok := symbolIndex[symbol]; ok {
			r := DependencyRecordV1{ConsumerRowKeyV1: planned[consumer].RowKeyV1, ConsumerManifestOrdinal: consumer + 1, ConsumerGoSymbol: planned[consumer].Cells[manifestGoSymbolIndex], DependencyKind: "wrapper_delegate", OwnerKind: "row", OwnerLogicalID: planned[owner].RowKeyV1, OwnerGoSymbolOrEmpty: symbol}
			out[dependencyEdgeKey(r)] = true
		}
	}
	for i, m := range reviewed {
		for _, backend := range m.Backends {
			for _, symbol := range dependencySymbols(backend.EvidenceAnchor) {
				add(i, symbol)
			}
			if backend.NAReason == "native_wrapper_delegates_explicit_endian" {
				for _, endian := range []string{"LE", "BE"} {
					add(i, strings.Replace(m.GoSymbol, "UTF16", "UTF16"+endian, 1))
				}
			}
		}
	}
	return out
}

func expectedSharedKernelEdges(planned []ManifestRowV1, reviewed []ReviewedMappingV1) (map[string]bool, error) {
	out := map[string]bool{}
	for backendIndex, backend := range membershipBackends {
		groups := map[string][]int{}
		for i, m := range reviewed {
			if m.Backends[backendIndex].Outcome == "eligible" {
				groups[m.CanonicalKernelName] = append(groups[m.CanonicalKernelName], i)
			}
		}
		for _, members := range groups {
			if len(members) < 2 {
				continue
			}
			sort.Slice(members, func(i, j int) bool { return sharedOwnerLess(members[i], members[j], planned, reviewed) })
			owner := members[0]
			ownerCell, err := CellKeyV1(planned[owner].RowKeyV1, backend)
			if err != nil {
				return nil, err
			}
			for _, consumer := range members[1:] {
				r := DependencyRecordV1{ConsumerRowKeyV1: planned[consumer].RowKeyV1, ConsumerManifestOrdinal: consumer + 1, ConsumerGoSymbol: planned[consumer].Cells[manifestGoSymbolIndex], DependencyKind: "shared_kernel_delegate", OwnerKind: "cell", OwnerLogicalID: ownerCell.StorageID, OwnerGoSymbolOrEmpty: planned[owner].Cells[manifestGoSymbolIndex]}
				out[dependencyEdgeKey(r)] = true
			}
		}
	}
	return out, nil
}

func sharedOwnerLess(a, b int, planned []ManifestRowV1, reviewed []ReviewedMappingV1) bool {
	aw, _ := familyWave(reviewed[a].FamilyContractDisplayID)
	bw, _ := familyWave(reviewed[b].FamilyContractDisplayID)
	if aw != bw {
		return aw < bw
	}
	if reviewed[a].FamilyContractDisplayID != reviewed[b].FamilyContractDisplayID {
		return reviewed[a].FamilyContractDisplayID < reviewed[b].FamilyContractDisplayID
	}
	if planned[a].PlannedOrdinal != planned[b].PlannedOrdinal {
		return planned[a].PlannedOrdinal < planned[b].PlannedOrdinal
	}
	return planned[a].RowKeyV1 < planned[b].RowKeyV1
}

func validDependencyEvidence(anchor string) bool {
	if strings.HasPrefix(anchor, "simd/archsimd Go 1.26.5:") {
		return strings.Contains(anchor, "audit row") && strings.Contains(anchor, "available:")
	}
	return strings.Contains(anchor, "611becc2a08c27a4edc77d9a45ff74c97130129b:include/simdutf/implementation.h:")
}

func dependencySymbols(anchor string) []string {
	var out []string
	for part := range strings.SplitSeq(anchor, ";") {
		if strings.HasPrefix(part, "dependency:") && len(part) > len("dependency:") {
			out = append(out, strings.TrimPrefix(part, "dependency:"))
		}
	}
	return out
}

func rowIndex(planned []ManifestRowV1) map[string]int {
	out := make(map[string]int, len(planned))
	for i, p := range planned {
		out[p.RowKeyV1] = i
	}
	return out
}

func indexBySymbol(planned []ManifestRowV1) map[string]int {
	out := make(map[string]int, len(planned))
	for i, p := range planned {
		out[p.Cells[manifestGoSymbolIndex]] = i
	}
	return out
}

func dependencyEdgeKey(r DependencyRecordV1) string {
	return strings.Join([]string{r.ConsumerRowKeyV1, r.DependencyKind, r.OwnerKind, r.OwnerLogicalID, r.OwnerGoSymbolOrEmpty}, "\x00")
}

func hasNoneDependency(records []DependencyRecordV1, rowKey string) bool {
	for _, record := range records {
		if record.ConsumerRowKeyV1 == rowKey && record.DependencyKind == "none" {
			return true
		}
	}
	return false
}

// BuildCanonicalRowRanksV1 returns the deterministic topological rank of each row.
func BuildCanonicalRowRanksV1(planned []ManifestRowV1, reviewed []ReviewedMappingV1, ledger []ISARowV1, dependencies []DependencyRecordV1) (map[string]int, error) {
	if err := validateDependencyInputs(planned, reviewed); err != nil || len(ledger) != 23 || len(dependencies) != 185 {
		return nil, fmt.Errorf("canonical ranks: invalid input counts")
	}
	index := rowIndex(planned)
	wave := make([]int, 125)
	semantic := make([]string, 125)
	for i, entry := range ledger {
		if entry.LedgerOrdinal != i+1 || entry.SemanticOperation == "" {
			return nil, fmt.Errorf("canonical ranks: invalid ISA ledger")
		}
		if _, err := LedgerOperationIDV1(entry.LedgerOrdinal, entry.SemanticOperation); err != nil {
			return nil, err
		}
	}
	for i := range planned {
		w, ok := familyWave(reviewed[i].FamilyContractDisplayID)
		if !ok {
			return nil, fmt.Errorf("canonical ranks: unknown family")
		}
		wave[i] = w
		if reviewed[i].ISAOrdinalOrScalar == "scalar" {
			var err error
			semantic[i], err = ScalarOperationIDV1(planned[i].RowKeyV1)
			if err != nil {
				return nil, err
			}
			continue
		}
		n, err := canonicalPositive(reviewed[i].ISAOrdinalOrScalar)
		if err != nil || n > len(ledger) {
			return nil, fmt.Errorf("canonical ranks: invalid ISA ledger")
		}
		semantic[i], err = LedgerOperationIDV1(n, ledger[n-1].SemanticOperation)
		if err != nil {
			return nil, err
		}
	}
	adj := make([][]int, 125)
	indegree := make([]int, 125)
	owners := make([][]string, 125)
	seen := map[string]bool{}
	for _, d := range dependencies {
		if d.ConsumerManifestOrdinal < 1 || d.ConsumerManifestOrdinal > 125 || d.ConsumerRowKeyV1 != planned[d.ConsumerManifestOrdinal-1].RowKeyV1 || d.ConsumerGoSymbol != planned[d.ConsumerManifestOrdinal-1].Cells[manifestGoSymbolIndex] {
			return nil, fmt.Errorf("canonical ranks: invalid dependency consumer")
		}
		if d.DependencyKind == "none" {
			if d.OwnerKind != "none" || d.OwnerLogicalID != "" || d.OwnerGoSymbolOrEmpty != "" {
				return nil, fmt.Errorf("canonical ranks: invalid none dependency")
			}
			continue
		}
		if d.DependencyKind == "shared_kernel_delegate" {
			if d.OwnerKind != "cell" {
				return nil, fmt.Errorf("canonical ranks: invalid cell dependency")
			}
			continue
		}
		if d.DependencyKind != "wrapper_delegate" || d.OwnerKind != "row" {
			return nil, fmt.Errorf("canonical ranks: invalid dependency kind")
		}
		from, ok := index[d.OwnerLogicalID]
		to := d.ConsumerManifestOrdinal - 1
		if !ok || from == to || d.OwnerGoSymbolOrEmpty != planned[from].Cells[manifestGoSymbolIndex] || wave[from] > wave[to] {
			return nil, fmt.Errorf("canonical ranks: invalid wrapper dependency")
		}
		key := d.OwnerLogicalID + "\x00" + d.ConsumerRowKeyV1
		if seen[key] {
			return nil, fmt.Errorf("canonical ranks: duplicate indegree edge")
		}
		seen[key] = true
		adj[from] = append(adj[from], to)
		indegree[to]++
		owners[to] = append(owners[to], d.OwnerLogicalID)
	}
	expectedWrapper := expectedWrapperEdges(planned, reviewed)
	expectedCell, err := expectedSharedKernelEdges(planned, reviewed)
	if err != nil {
		return nil, err
	}
	wrapperCount, cellCount, noneCount := 0, 0, 0
	for _, d := range dependencies {
		switch d.DependencyKind {
		case "none":
			noneCount++
		case "wrapper_delegate":
			wrapperCount++
			if !expectedWrapper[dependencyEdgeKey(d)] {
				return nil, fmt.Errorf("canonical ranks: unexpected wrapper dependency")
			}
		case "shared_kernel_delegate":
			cellCount++
			if !expectedCell[dependencyEdgeKey(d)] {
				return nil, fmt.Errorf("canonical ranks: unexpected shared-kernel dependency")
			}
		}
	}
	if noneCount != 84 || wrapperCount != 53 || cellCount != 48 || len(expectedWrapper) != wrapperCount || len(expectedCell) != cellCount {
		return nil, fmt.Errorf("canonical ranks: dependency set differs from reviewed mapping")
	}
	for i := range owners {
		sort.Strings(owners[i])
	}
	ranks := map[string]int{}
	ready := make([]int, 0, 125)
	for i := range planned {
		if indegree[i] == 0 {
			ready = append(ready, i)
		}
	}
	less := func(a, b int) bool {
		if wave[a] != wave[b] {
			return wave[a] < wave[b]
		}
		if reviewed[a].FamilyContractDisplayID != reviewed[b].FamilyContractDisplayID {
			return reviewed[a].FamilyContractDisplayID < reviewed[b].FamilyContractDisplayID
		}
		oa, ob := string(EncodeTupleV1(owners[a]...)), string(EncodeTupleV1(owners[b]...))
		if oa != ob {
			return oa < ob
		}
		if semantic[a] != semantic[b] {
			return semantic[a] < semantic[b]
		}
		if planned[a].PlannedOrdinal != planned[b].PlannedOrdinal {
			return planned[a].PlannedOrdinal < planned[b].PlannedOrdinal
		}
		return planned[a].RowKeyV1 < planned[b].RowKeyV1
	}
	for rank := 0; len(ready) > 0; rank++ {
		sort.Slice(ready, func(i, j int) bool { return less(ready[i], ready[j]) })
		x := ready[0]
		ready = ready[1:]
		ranks[planned[x].RowKeyV1] = rank
		for _, y := range adj[x] {
			indegree[y]--
			if indegree[y] == 0 {
				ready = append(ready, y)
			}
		}
	}
	if len(ranks) != 125 {
		return nil, fmt.Errorf("canonical ranks: dependency cycle")
	}
	for a := range planned {
		for b := range planned {
			if wave[a] > wave[b] && ranks[planned[a].RowKeyV1] <= ranks[planned[b].RowKeyV1] {
				return nil, fmt.Errorf("canonical ranks: wave order violated")
			}
		}
	}
	return ranks, nil
}

// ParseLockedSetsV1 parses the frozen locked-set contract.
func ParseLockedSetsV1(data, publicGolden []byte, planned []ManifestRowV1) (map[string][]LockedSetRecordV1, error) {
	lines, err := parseLines(data, "locked sets")
	if err != nil {
		return nil, err
	}
	if err = validateHeader(lines[0], lockedSetHeaderV1[:], "locked sets"); err != nil {
		return nil, err
	}
	if len(lines)-1 != 212 || len(planned) != 125 {
		return nil, fmt.Errorf("locked sets: invalid record or planned count")
	}
	counts := map[string]int{"current_public_api": 67, "planned_api_contract": 125, "backend": 4, "official_architecture": 2, "authoritative_host": 2, "evidence_lane": 6, "compile_only_target": 2, "excluded_acceleration": 4}
	out := map[string][]LockedSetRecordV1{}
	seen := map[string]bool{}
	for i, line := range lines[1:] {
		f, e := parseRecord(line, 5, i+2, "locked sets")
		if e != nil {
			return nil, e
		}
		n, e := canonicalPositive(f[2])
		if e != nil || f[0] != "locked-sets-v1" || counts[f[1]] == 0 || f[3] == "" || f[4] == "" || len(f[3])%2 != 0 || strings.ToLower(f[3]) != f[3] {
			return nil, fmt.Errorf("locked sets line %d: invalid record", i+2)
		}
		value, e := hex.DecodeString(f[3])
		if e != nil || len(value) == 0 {
			return nil, fmt.Errorf("locked sets line %d: invalid value", i+2)
		}
		k := f[1] + "\x00" + f[2]
		if seen[k] {
			return nil, fmt.Errorf("locked sets line %d: duplicate ordinal", i+2)
		}
		seen[k] = true
		out[f[1]] = append(out[f[1]], LockedSetRecordV1{f[1], n, f[3], f[4]})
	}
	for name, count := range counts {
		rows := out[name]
		if len(rows) != count {
			return nil, fmt.Errorf("locked sets: wrong count for %s", name)
		}
		values := map[string]bool{}
		for i, r := range rows {
			if r.Ordinal != i+1 || values[r.ValueHex] {
				return nil, fmt.Errorf("locked sets: noncanonical %s", name)
			}
			values[r.ValueHex] = true
		}
	}
	golden, e := parseLines(publicGolden, "public golden")
	if e != nil || len(golden) < 67 {
		return nil, fmt.Errorf("locked sets: public golden lost frozen API entries")
	}
	goldenValues := make(map[string]bool, len(golden))
	for _, line := range golden {
		goldenValues[hex.EncodeToString([]byte(line))] = true
	}
	for i, record := range out["current_public_api"] {
		if !goldenValues[record.ValueHex] {
			return nil, fmt.Errorf("locked sets: frozen public API entry %d is absent", i+1)
		}
	}
	for i, p := range planned {
		if p.PlannedOrdinal != i+1 || out["planned_api_contract"][i].ValueHex != hex.EncodeToString(EncodeTupleV1(p.Cells[manifestGoSymbolIndex], p.Cells[manifestGoSignatureIndex])) {
			return nil, fmt.Errorf("locked sets: planned contract drift at %d", i+1)
		}
	}
	fixed := map[string][]string{"backend": {"westmere", "haswell", "archsimd", "neon"}, "official_architecture": {"amd64", "arm64"}, "authoritative_host": {"darwin-arm64-apple-m3-max", "linux-amd64-debian-13-xeon-platinum-8481c"}, "evidence_lane": {"darwin-arm64-nosimd", "darwin-arm64-simd-negative", "linux-amd64-none", "linux-amd64-simd", "linux-riscv64-compile", "linux-s390x-compile"}, "compile_only_target": {"riscv64", "s390x"}, "excluded_acceleration": {"avx512", "riscv-vector", "loongarch64", "power"}}
	for name, values := range fixed {
		for i, value := range values {
			if out[name][i].ValueHex != hex.EncodeToString([]byte(value)) {
				return nil, fmt.Errorf("locked sets: fixed set drift for %s", name)
			}
		}
	}
	return out, nil
}

func canonicalPositive(s string) (int, error) {
	n, e := strconv.Atoi(s)
	if e != nil || n < 1 || strconv.Itoa(n) != s {
		return 0, fmt.Errorf("noncanonical decimal")
	}
	return n, nil
}

func hasCycle(edges [][]int) bool {
	state := make([]uint8, len(edges))
	var visit func(int) bool
	visit = func(n int) bool {
		if state[n] == 1 {
			return true
		}
		if state[n] == 2 {
			return false
		}
		state[n] = 1
		if slices.ContainsFunc(edges[n], visit) {
			return true
		}
		state[n] = 2
		return false
	}
	for n := range edges {
		if visit(n) {
			return true
		}
	}
	return false
}
