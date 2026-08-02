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
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type FinalOperationV1 struct {
	ISAOrdinal                                          int
	ISASemanticOperationExact, SemanticOperationID      string
	InitialRowCount, InitialCellCount, ExistingRowCount int
	Reconciliation, EvidenceAnchor                      string
}
type FinalCellV1 struct {
	RowKeyV1, CellStorageID                                                                                                                                                                                                             string
	ManifestOrdinal, CanonicalRowRank                                                                                                                                                                                                   int
	FamilyContractDisplayID, Backend, SemanticOperationID, ISAOrdinalOrScalar, BackendOutcome, BackendNAReason, BackendEvidenceAnchor, DirectSymbol, DirectSymbolStorageID, SharedKernelID, KernelOwnerCellKey, KernelOwnerDependencyID string
	KernelBatchDisplayID, KernelBatchStorageID, EvidenceBatchDisplayID, EvidenceBatchStorageID                                                                                                                                          string
}
type KernelRegistryV1 struct {
	Backend, CanonicalKernelName, SharedKernelID, FamilyContractDisplayID, SemanticOperationID, KernelOwnerCellKey string
	MemberCount                                                                                                    int
}
type MembershipV1 struct {
	Operations []FinalOperationV1
	Cells      []FinalCellV1
	Kernels    []KernelRegistryV1
}

var membershipBackends = [...]string{"westmere", "haswell", "archsimd", "neon"}

func BuildMembershipV1(reviewed []ReviewedMappingV1, manifest, planned []ManifestRowV1, ledger []ISARowV1, existing []ExistingMemberV1, canonicalRowRanks map[string]int) (MembershipV1, error) {
	if err := validateMembershipInputs(reviewed, manifest, planned, ledger, existing, canonicalRowRanks); err != nil {
		return MembershipV1{}, err
	}
	m := MembershipV1{Operations: make([]FinalOperationV1, len(ledger)), Cells: make([]FinalCellV1, 0, 500)}
	for i, l := range ledger {
		id, err := LedgerOperationIDV1(l.LedgerOrdinal, l.SemanticOperation)
		if err != nil {
			return MembershipV1{}, err
		}
		m.Operations[i] = FinalOperationV1{ISAOrdinal: l.LedgerOrdinal, ISASemanticOperationExact: l.SemanticOperation, SemanticOperationID: id, Reconciliation: "unreconciled", EvidenceAnchor: l.Cells[16]}
	}
	cellKeys := make(map[string]struct{}, 500)
	for i, r := range reviewed {
		var id string
		ord := 0
		if r.ISAOrdinalOrScalar == "scalar" {
			var err error
			id, err = ScalarOperationIDV1(planned[i].RowKeyV1)
			if err != nil {
				return MembershipV1{}, err
			}
		} else {
			ord, _ = strconv.Atoi(r.ISAOrdinalOrScalar)
			id = m.Operations[ord-1].SemanticOperationID
			m.Operations[ord-1].InitialRowCount++
		}
		for b, name := range membershipBackends {
			x := r.Backends[b]
			key, err := CellKeyV1(planned[i].RowKeyV1, name)
			if err != nil {
				return MembershipV1{}, err
			}
			if _, duplicate := cellKeys[key.StorageID]; duplicate {
				return MembershipV1{}, fmt.Errorf("duplicate computed cell key %q", key.StorageID)
			}
			cellKeys[key.StorageID] = struct{}{}
			c := FinalCellV1{RowKeyV1: planned[i].RowKeyV1, CellStorageID: key.StorageID, ManifestOrdinal: i + 1, CanonicalRowRank: canonicalRowRanks[planned[i].RowKeyV1], FamilyContractDisplayID: r.FamilyContractDisplayID, Backend: name, SemanticOperationID: id, ISAOrdinalOrScalar: r.ISAOrdinalOrScalar, BackendOutcome: x.Outcome, BackendNAReason: x.NAReason, BackendEvidenceAnchor: x.EvidenceAnchor, DirectSymbol: x.DirectSymbol}
			if ord != 0 {
				m.Operations[ord-1].InitialCellCount++
			}
			if x.Outcome == "eligible" {
				sid, err := SharedKernelIDV1(name, r.CanonicalKernelName)
				if err != nil {
					return MembershipV1{}, err
				}
				c.SharedKernelID = sid
				symbol, err := SymbolKeyV1(name, x.DirectSymbol)
				if err != nil {
					return MembershipV1{}, err
				}
				c.DirectSymbolStorageID = symbol.StorageID
			}
			m.Cells = append(m.Cells, c)
		}
	}
	for _, e := range existing {
		if e.ISAOrdinalOrScalar != "scalar" {
			n, _ := strconv.Atoi(e.ISAOrdinalOrScalar)
			m.Operations[n-1].ExistingRowCount++
		}
	}
	for i := range m.Operations {
		o := &m.Operations[i]
		switch {
		case o.InitialRowCount > 0:
			o.Reconciliation = "initial_members"
		case o.ExistingRowCount > 0:
			o.Reconciliation = "existing_only"
		case o.EvidenceAnchor != "":
			o.Reconciliation = "zero_initial_explained"
		default:
			return MembershipV1{}, fmt.Errorf("operation %d has unexplained zero initial membership", o.ISAOrdinal)
		}
	}
	type group struct {
		indexes []int
		name    string
	}
	groups := map[string]*group{}
	for i := range m.Cells {
		c := &m.Cells[i]
		if c.BackendOutcome != "eligible" {
			continue
		}
		k := c.Backend + "\x00" + c.SharedKernelID
		g := groups[k]
		if g == nil {
			g = &group{name: reviewed[i/4].CanonicalKernelName}
			groups[k] = g
		} else if g.name != reviewed[i/4].CanonicalKernelName {
			return MembershipV1{}, fmt.Errorf("shared kernel ID %q has unequal canonical names", c.SharedKernelID)
		}
		g.indexes = append(g.indexes, i)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		g := groups[k]
		sort.SliceStable(g.indexes, func(a, b int) bool { return ownerLess(m.Cells[g.indexes[a]], m.Cells[g.indexes[b]]) })
		owner := &m.Cells[g.indexes[0]]
		key, err := CellKeyV1(owner.RowKeyV1, owner.Backend)
		if err != nil {
			return MembershipV1{}, err
		}
		owner.KernelOwnerCellKey = key.StorageID
		for _, idx := range g.indexes[1:] {
			m.Cells[idx].KernelOwnerCellKey = owner.KernelOwnerCellKey
			m.Cells[idx].KernelOwnerDependencyID = owner.KernelOwnerCellKey
		}
		m.Kernels = append(m.Kernels, KernelRegistryV1{Backend: owner.Backend, CanonicalKernelName: g.name, SharedKernelID: owner.SharedKernelID, FamilyContractDisplayID: owner.FamilyContractDisplayID, SemanticOperationID: owner.SemanticOperationID, KernelOwnerCellKey: owner.KernelOwnerCellKey, MemberCount: len(g.indexes)})
	}
	if err := validateMembershipEmission(m, ledger, existing); err != nil {
		return MembershipV1{}, err
	}
	return m, nil
}

func validateMembershipInputs(reviewed []ReviewedMappingV1, manifest, planned []ManifestRowV1, ledger []ISARowV1, existing []ExistingMemberV1, ranks map[string]int) error {
	if len(manifest) != 164 || len(reviewed) != 125 || len(planned) != 125 || len(ledger) != 23 || len(existing) != 30 || len(ranks) != 125 {
		return fmt.Errorf("membership inputs require 164 manifest rows, 125 reviewed/planned rows, 23 ledger rows, 30 existing members, and 125 canonical ranks")
	}
	manifestRows, implemented := map[string]ManifestRowV1{}, map[string]bool{}
	implementedCount, plannedCount, excludedCount := 0, 0, 0
	for _, row := range manifest {
		fields := manifestComposite(row.Cells)
		if row.RowKeyV1 == "" || row.RowKeyV1 != RowKeyV1(fields) {
			return fmt.Errorf("manifest row %q has noncanonical row key", row.Cells[manifestGoSymbolIndex])
		}
		if manifestRows[row.RowKeyV1].RowKeyV1 != "" {
			return fmt.Errorf("duplicate manifest row key %q", row.RowKeyV1)
		}
		manifestRows[row.RowKeyV1] = row
		if row.Cells[manifestStatusIndex] == "implemented" {
			implementedCount++
			if implemented[row.Cells[manifestGoSymbolIndex]] {
				return fmt.Errorf("duplicate implemented member %q", row.Cells[manifestGoSymbolIndex])
			}
			implemented[row.Cells[manifestGoSymbolIndex]] = true
		}
		if row.Cells[manifestStatusIndex] == "planned" {
			plannedCount++
		}
		if row.Cells[manifestStatusIndex] == "excluded" {
			excludedCount++
		}
	}
	if implementedCount+plannedCount != 155 || implementedCount < len(existing) || excludedCount != 9 || len(implemented) != implementedCount {
		return fmt.Errorf("manifest requires 155 published or planned members, including 30 initial implemented members, and 9 excluded members")
	}
	seenRows, seenRanks, seenExisting := map[string]bool{}, map[int]bool{}, map[string]bool{}
	kernelByBackendName := map[string]string{}
	idByBackendName := map[string]string{}
	directID := map[string]string{}
	for i := range planned {
		p, r := planned[i], reviewed[i]
		if p.PlannedOrdinal != i+1 || r.ManifestOrdinal != i+1 || r.GoSymbol != p.Cells[manifestGoSymbolIndex] || p.RowKeyV1 == "" {
			return fmt.Errorf("reviewed/planned join %d is invalid", i+1)
		}
		if p.RowKeyV1 != RowKeyV1(manifestComposite(p.Cells)) {
			return fmt.Errorf("planned row %d has noncanonical row key", i+1)
		}
		manifestRow, ok := manifestRows[p.RowKeyV1]
		if !ok || !sameManifestContractV1(manifestRow, p) {
			return fmt.Errorf("planned row %d does not join the evolving manifest", i+1)
		}
		if _, ok := familyWave(r.FamilyContractDisplayID); !ok {
			return fmt.Errorf("reviewed row %d has unknown family %q", i+1, r.FamilyContractDisplayID)
		}
		if seenRows[p.RowKeyV1] {
			return fmt.Errorf("duplicate row key %q", p.RowKeyV1)
		}
		seenRows[p.RowKeyV1] = true
		rank, ok := ranks[p.RowKeyV1]
		if !ok || rank < 0 || rank >= 125 || seenRanks[rank] {
			return fmt.Errorf("invalid canonical rank for row key %q", p.RowKeyV1)
		}
		seenRanks[rank] = true
		if r.ISAOrdinalOrScalar != "scalar" {
			n, err := strconv.Atoi(r.ISAOrdinalOrScalar)
			if err != nil || n < 1 || n > 23 || strconv.Itoa(n) != r.ISAOrdinalOrScalar {
				return fmt.Errorf("noncanonical ISA ordinal %q", r.ISAOrdinalOrScalar)
			}
		}
		eligible := false
		for b := range r.Backends {
			if err := validateBackendMapping(r, b, p, ledgerForMapping(r, ledger)); err != nil {
				return fmt.Errorf("reviewed row %d: %w", i+1, err)
			}
		}
		for b, x := range r.Backends {
			if x.Outcome != "eligible" {
				continue
			}
			eligible = true
			id, err := SharedKernelIDV1(membershipBackends[b], r.CanonicalKernelName)
			if err != nil {
				return err
			}
			nameKey := membershipBackends[b] + "\x00" + r.CanonicalKernelName
			if prior, ok := idByBackendName[nameKey]; ok && prior != id {
				return fmt.Errorf("kernel name %q has unequal identity", r.CanonicalKernelName)
			}
			idByBackendName[nameKey] = id
			if prior, ok := kernelByBackendName[membershipBackends[b]+"\x00"+id]; ok && prior != r.CanonicalKernelName {
				return fmt.Errorf("kernel identity %q has unequal name", id)
			}
			kernelByBackendName[membershipBackends[b]+"\x00"+id] = r.CanonicalKernelName
			directKey := membershipBackends[b] + "\x00" + x.DirectSymbol
			if prior, ok := directID[directKey]; ok && prior != id {
				return fmt.Errorf("direct symbol %q has unequal kernel identity", x.DirectSymbol)
			}
			directID[directKey] = id
			suffix := [...]string{"Westmere", "Haswell", "Archsimd", "NEON"}[b]
			if x.DirectSymbol != lowerFirst(r.GoSymbol)+suffix {
				return fmt.Errorf("direct symbol %q does not match Go symbol", x.DirectSymbol)
			}
		}
		if !eligible && r.CanonicalKernelName != "" {
			return fmt.Errorf("row %d has canonical kernel without eligible backend", i+1)
		}
	}
	if len(seenRows) != 125 || len(seenRanks) != 125 {
		return fmt.Errorf("planned rows or canonical ranks are incomplete")
	}
	for i, l := range ledger {
		if l.LedgerOrdinal != i+1 || l.SemanticOperation == "" {
			return fmt.Errorf("invalid ISA ledger row %d", i+1)
		}
	}
	for _, e := range existing {
		if e.GoSymbol == "" || e.EvidenceAnchor == "" || seenExisting[e.GoSymbol] || !implemented[e.GoSymbol] {
			return fmt.Errorf("invalid existing member %q", e.GoSymbol)
		}
		seenExisting[e.GoSymbol] = true
		if e.ISAOrdinalOrScalar != "scalar" {
			n, err := strconv.Atoi(e.ISAOrdinalOrScalar)
			if err != nil || n < 1 || n > 23 || strconv.Itoa(n) != e.ISAOrdinalOrScalar {
				return fmt.Errorf("existing member %q has invalid ISA ordinal", e.GoSymbol)
			}
		}
	}
	if len(seenExisting) != len(existing) {
		return fmt.Errorf("existing members do not join the initial implemented rows")
	}
	return nil
}

func sameManifestContractV1(current, frozen ManifestRowV1) bool {
	for index := range current.Cells {
		if index == manifestStatusIndex || index == manifestMilestoneIndex {
			continue
		}
		if current.Cells[index] != frozen.Cells[index] {
			return false
		}
	}
	status, milestone := current.Cells[manifestStatusIndex], current.Cells[manifestMilestoneIndex]
	return status == "planned" && milestone == "future-upstream-api" ||
		status == "implemented" && milestone == "611becc-current-api"
}

func validateMembershipEmission(m MembershipV1, ledger []ISARowV1, existing []ExistingMemberV1) error {
	if len(m.Operations) != len(ledger) {
		return fmt.Errorf("operation emission length mismatch")
	}
	rows := make([]map[string]bool, len(ledger))
	cells, existingRows := make([]int, len(ledger)), make([]int, len(ledger))
	for i := range rows {
		rows[i] = map[string]bool{}
	}
	for _, c := range m.Cells {
		if c.ISAOrdinalOrScalar == "scalar" {
			continue
		}
		n, err := strconv.Atoi(c.ISAOrdinalOrScalar)
		if err != nil || n < 1 || n > len(ledger) {
			return fmt.Errorf("emitted cell has invalid ISA ordinal %q", c.ISAOrdinalOrScalar)
		}
		rows[n-1][c.RowKeyV1] = true
		cells[n-1]++
	}
	for _, e := range existing {
		if e.ISAOrdinalOrScalar == "scalar" {
			continue
		}
		n, err := strconv.Atoi(e.ISAOrdinalOrScalar)
		if err != nil || n < 1 || n > len(ledger) || e.EvidenceAnchor == "" {
			return fmt.Errorf("existing member has invalid operation reconciliation")
		}
		existingRows[n-1]++
	}
	for i, o := range m.Operations {
		if o.InitialRowCount != len(rows[i]) || o.InitialCellCount != cells[i] || o.ExistingRowCount != existingRows[i] {
			return fmt.Errorf("operation %d emitted counts disagree", o.ISAOrdinal)
		}
		switch o.Reconciliation {
		case "initial_members":
			if o.InitialRowCount == 0 {
				return fmt.Errorf("operation %d has invalid initial reconciliation", o.ISAOrdinal)
			}
		case "existing_only":
			if o.InitialRowCount != 0 || existingRows[i] == 0 {
				return fmt.Errorf("operation %d lacks existing anchor", o.ISAOrdinal)
			}
		case "zero_initial_explained":
			if o.InitialRowCount != 0 || o.ExistingRowCount != 0 || !strings.HasPrefix(ledger[i].Cells[16], "Pin:") {
				return fmt.Errorf("operation %d lacks frozen Pin provenance", o.ISAOrdinal)
			}
		default:
			return fmt.Errorf("operation %d has invalid reconciliation", o.ISAOrdinal)
		}
	}
	return nil
}

func ownerLess(a, b FinalCellV1) bool {
	if a.CanonicalRowRank != b.CanonicalRowRank {
		return a.CanonicalRowRank < b.CanonicalRowRank
	}
	aw, aok := familyWave(a.FamilyContractDisplayID)
	bw, bok := familyWave(b.FamilyContractDisplayID)
	if aok && bok && aw != bw {
		return aw < bw
	}
	if a.FamilyContractDisplayID != b.FamilyContractDisplayID {
		return a.FamilyContractDisplayID < b.FamilyContractDisplayID
	}
	if a.ManifestOrdinal != b.ManifestOrdinal {
		return a.ManifestOrdinal < b.ManifestOrdinal
	}
	return a.RowKeyV1 < b.RowKeyV1
}

func familyWave(family string) (int, bool) {
	switch family {
	case "FC-v1-helper-validation":
		return 1, true
	case "FC-v1-latin1-source":
		return 2, true
	case "FC-v1-utf8-source":
		return 3, true
	case "FC-v1-utf16-source":
		return 4, true
	case "FC-v1-utf32-source":
		return 5, true
	case "FC-v1-detection", "FC-v1-find":
		return 6, true
	case "FC-v1-base64":
		return 7, true
	default:
		return 0, false
	}
}

func RenderOperationsV1(rows []FinalOperationV1) []byte {
	lines := []string{strings.Join(finalOperationsHeaderV1[:], "\t")}
	for _, r := range rows {
		lines = append(lines, strings.Join([]string{"v1", strconv.Itoa(r.ISAOrdinal), r.ISASemanticOperationExact, r.SemanticOperationID, strconv.Itoa(r.InitialRowCount), strconv.Itoa(r.InitialCellCount), strconv.Itoa(r.ExistingRowCount), r.Reconciliation, r.EvidenceAnchor}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func RenderCellsV1(rows []FinalCellV1) []byte {
	lines := []string{strings.Join(finalCellsHeaderV1[:], "\t")}
	for _, r := range rows {
		lines = append(lines, strings.Join([]string{"v1", r.RowKeyV1, strconv.Itoa(r.ManifestOrdinal), strconv.Itoa(r.CanonicalRowRank), r.FamilyContractDisplayID, r.Backend, r.CellStorageID, r.SemanticOperationID, r.ISAOrdinalOrScalar, r.BackendOutcome, r.BackendNAReason, r.BackendEvidenceAnchor, r.DirectSymbol, r.DirectSymbolStorageID, r.SharedKernelID, r.KernelOwnerCellKey, r.KernelOwnerDependencyID, r.KernelBatchDisplayID, r.KernelBatchStorageID, r.EvidenceBatchDisplayID, r.EvidenceBatchStorageID}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func RenderKernelsV1(rows []KernelRegistryV1) []byte {
	lines := []string{strings.Join(kernelRegistryHeaderV1[:], "\t")}
	for _, r := range rows {
		lines = append(lines, strings.Join([]string{"v1", r.Backend, r.CanonicalKernelName, r.SharedKernelID, r.FamilyContractDisplayID, r.SemanticOperationID, r.KernelOwnerCellKey, strconv.Itoa(r.MemberCount)}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}
