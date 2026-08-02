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
	classificationHeaderV1 = [...]string{"mapping_version", "row_key_v1", "manifest_ordinal", "canonical_row_rank", "canonical_dependency_rank", "go_symbol", "go_signature", "family_contract_display_id", "family_storage_id", "wave", "semantic_operation_id", "dependency_tuple_hex", "scalar_batch_display_id", "scalar_batch_storage_id", "api_transaction_display_id", "api_transaction_storage_id", "scalar_source", "test_source", "fuzz_source", "benchmark_source"}
	batchesHeaderV1        = [...]string{"mapping_version", "batch_kind", "batch_display_id", "batch_storage_id", "family_contract_display_id", "backend", "sequence", "distinct_semantic_operation_count", "member_count", "member_tuple_hex"}
)

// ClassificationRowV1 is the canonical publication row for one planned API.
type ClassificationRowV1 struct {
	RowKeyV1, GoSymbol, GoSignature, FamilyContractDisplayID, FamilyStorageID, SemanticOperationID string
	ManifestOrdinal, CanonicalRowRank, CanonicalDependencyRank, Wave                               int
	DependencyTupleHex, ScalarBatchDisplayID, ScalarBatchStorageID                                 string
	APITransactionDisplayID, APITransactionStorageID                                               string
	ScalarSource, TestSource, FuzzSource, BenchmarkSource                                          string
}

// BatchRecordV1 records one scalar, kernel, or evidence batch.
type BatchRecordV1 struct {
	Kind, DisplayID, StorageID, FamilyContractDisplayID, Backend, MemberTupleHex string
	Sequence, DistinctSemanticOperationCount                                     int
	MemberStorageKeys                                                            []string
}

// ClassificationV1 is the complete deterministic classification receipt.
type ClassificationV1 struct {
	Rows    []ClassificationRowV1
	Batches []BatchRecordV1
	Cells   []FinalCellV1
}

func ClassificationHeaderV1() []string { return append([]string(nil), classificationHeaderV1[:]...) }
func BatchesHeaderV1() []string        { return append([]string(nil), batchesHeaderV1[:]...) }

// BuildClassificationV1 derives the scalar, kernel, and evidence batches from
// the reviewed membership. It validates all identities again at this boundary.
func BuildClassificationV1(planned []ManifestRowV1, reviewed []ReviewedMappingV1, dependencies []DependencyRecordV1, ranks map[string]int, membership MembershipV1) (ClassificationV1, error) {
	if len(planned) != 125 || len(reviewed) != 125 || len(ranks) != 125 || len(membership.Cells) != 500 {
		return ClassificationV1{}, fmt.Errorf("classification requires 125 planned/reviewed/ranked rows and 500 cells")
	}
	rowByKey := make(map[string]int, len(planned))
	cellByKey := make(map[string]int, len(membership.Cells))
	for i, p := range planned {
		if p.PlannedOrdinal != i+1 || p.RowKeyV1 == "" || reviewed[i].ManifestOrdinal != i+1 || reviewed[i].GoSymbol != p.Cells[manifestGoSymbolIndex] {
			return ClassificationV1{}, fmt.Errorf("classification planned/reviewed join %d is invalid", i+1)
		}
		if rank, ok := ranks[p.RowKeyV1]; !ok || rank < 0 || rank >= len(planned) {
			return ClassificationV1{}, fmt.Errorf("invalid rank for row %q", p.RowKeyV1)
		}
		if _, duplicate := rowByKey[p.RowKeyV1]; duplicate {
			return ClassificationV1{}, fmt.Errorf("duplicate planned row key %q", p.RowKeyV1)
		}
		rowByKey[p.RowKeyV1] = i
	}
	seenRanks := make([]bool, len(planned))
	lastWave := 0
	for rank := range planned {
		var row string
		for key, value := range ranks {
			if value == rank {
				row = key
				break
			}
		}
		if row == "" || seenRanks[rank] {
			return ClassificationV1{}, fmt.Errorf("canonical ranks are not total")
		}
		seenRanks[rank] = true
		index, ok := rowByKey[row]
		if !ok {
			return ClassificationV1{}, fmt.Errorf("rank has unknown row %q", row)
		}
		wave, ok := familyWave(reviewed[index].FamilyContractDisplayID)
		if !ok || wave < lastWave {
			return ClassificationV1{}, fmt.Errorf("canonical ranks do not preserve wave separation")
		}
		lastWave = wave
	}
	cells := append([]FinalCellV1(nil), membership.Cells...)
	symbols := map[string]bool{}
	for i := range cells {
		c := &cells[i]
		key, err := CellKeyV1(c.RowKeyV1, c.Backend)
		rowIndex, exists := rowByKey[c.RowKeyV1]
		if err != nil || !exists || c.CellStorageID != key.StorageID || c.ManifestOrdinal != rowIndex+1 || c.CanonicalRowRank != ranks[c.RowKeyV1] {
			return ClassificationV1{}, fmt.Errorf("invalid emitted cell %d", i)
		}
		if _, exists := cellByKey[key.StorageID]; exists {
			return ClassificationV1{}, fmt.Errorf("duplicate cell key %q", key.StorageID)
		}
		cellByKey[key.StorageID] = i
		if c.BackendOutcome != "eligible" {
			if c.DirectSymbolStorageID != "" || c.KernelBatchDisplayID != "" || c.KernelBatchStorageID != "" || c.EvidenceBatchDisplayID != "" || c.EvidenceBatchStorageID != "" {
				return ClassificationV1{}, fmt.Errorf("N/A cell %q has classification IDs", key.StorageID)
			}
			continue
		}
		symbol, err := SymbolKeyV1(c.Backend, c.DirectSymbol)
		if err != nil || c.DirectSymbolStorageID != symbol.StorageID || symbols[c.Backend+"\x00"+c.DirectSymbol] {
			return ClassificationV1{}, fmt.Errorf("invalid or duplicate direct symbol %q", c.DirectSymbol)
		}
		symbols[c.Backend+"\x00"+c.DirectSymbol] = true
		if c.SharedKernelID == "" || c.KernelOwnerCellKey == "" {
			return ClassificationV1{}, fmt.Errorf("eligible cell %q lacks kernel identity", key.StorageID)
		}
		kernel, err := SharedKernelIDV1(c.Backend, reviewed[rowIndex].CanonicalKernelName)
		if err != nil || c.SharedKernelID != kernel {
			return ClassificationV1{}, fmt.Errorf("cell %q has noncanonical shared kernel ID", key.StorageID)
		}
		if c.KernelOwnerCellKey != key.StorageID && c.KernelOwnerDependencyID != c.KernelOwnerCellKey {
			return ClassificationV1{}, fmt.Errorf("cell %q has inconsistent owner dependency", key.StorageID)
		}
	}
	if len(cellByKey) != 500 {
		return ClassificationV1{}, fmt.Errorf("missing or extra membership cell")
	}

	depsByRow := make(map[string][]DependencyRecordV1, len(planned))
	for _, d := range dependencies {
		if _, exists := rowByKey[d.ConsumerRowKeyV1]; !exists {
			return ClassificationV1{}, fmt.Errorf("dependency has unknown consumer %q", d.ConsumerRowKeyV1)
		}
		depsByRow[d.ConsumerRowKeyV1] = append(depsByRow[d.ConsumerRowKeyV1], d)
	}
	for key := range rowByKey {
		if len(depsByRow[key]) == 0 {
			return ClassificationV1{}, fmt.Errorf("row %q has no dependencies", key)
		}
	}

	out := ClassificationV1{Cells: cells}
	familyRows := map[string][]int{}
	for i := range planned {
		familyRows[reviewed[i].FamilyContractDisplayID] = append(familyRows[reviewed[i].FamilyContractDisplayID], i)
	}
	families := make([]string, 0, len(familyRows))
	for family := range familyRows {
		families = append(families, family)
	}
	sort.Strings(families)
	for _, family := range families {
		indexes := familyRows[family]
		sort.Slice(indexes, func(a, b int) bool { return ranks[planned[indexes[a]].RowKeyV1] < ranks[planned[indexes[b]].RowKeyV1] })
		familyKey, err := FamilyKeyV1(family)
		if err != nil {
			return ClassificationV1{}, err
		}
		transactionRows := make([]string, len(indexes))
		for i, index := range indexes {
			transactionRows[i] = planned[index].RowKeyV1
		}
		transactionDisplay := "AP-v1-" + family
		transaction, err := TransactionKeyV1(transactionDisplay, transactionRows)
		if err != nil {
			return ClassificationV1{}, err
		}
		for start, sequence := 0, 1; start < len(indexes); start, sequence = start+12, sequence+1 {
			end := min(start+12, len(indexes))
			members := make([]string, end-start)
			for i, index := range indexes[start:end] {
				members[i] = planned[index].RowKeyV1
			}
			display := fmt.Sprintf("SB-v1-%s-%02d", family, sequence)
			batch, err := BatchKeyV1("scalar", display, members)
			if err != nil {
				return ClassificationV1{}, err
			}
			out.Batches = append(out.Batches, BatchRecordV1{Kind: "scalar", DisplayID: display, StorageID: batch.StorageID, FamilyContractDisplayID: family, Sequence: sequence, MemberTupleHex: hex.EncodeToString(EncodeTupleV1(members...)), MemberStorageKeys: members})
			for _, index := range indexes[start:end] {
				p := planned[index]
				deps := depsByRow[p.RowKeyV1]
				depHex, _, err := classificationDependencies(deps, ranks, cellByKey, cells)
				if err != nil {
					return ClassificationV1{}, err
				}
				westmere, err := CellKeyV1(p.RowKeyV1, membershipBackends[0])
				if err != nil {
					return ClassificationV1{}, err
				}
				cellIndex, ok := cellByKey[westmere.StorageID]
				if !ok {
					return ClassificationV1{}, fmt.Errorf("missing westmere cell for %q", p.RowKeyV1)
				}
				out.Rows = append(out.Rows, ClassificationRowV1{RowKeyV1: p.RowKeyV1, ManifestOrdinal: index + 1, CanonicalRowRank: ranks[p.RowKeyV1], CanonicalDependencyRank: ranks[p.RowKeyV1], GoSymbol: p.Cells[manifestGoSymbolIndex], GoSignature: p.Cells[manifestGoSignatureIndex], FamilyContractDisplayID: family, FamilyStorageID: familyKey.StorageID, Wave: mustFamilyWave(family), SemanticOperationID: cells[cellIndex].SemanticOperationID, DependencyTupleHex: depHex, ScalarBatchDisplayID: display, ScalarBatchStorageID: batch.StorageID, APITransactionDisplayID: transactionDisplay, APITransactionStorageID: transaction.StorageID, ScalarSource: p.Cells[14], TestSource: p.Cells[17], FuzzSource: p.Cells[18], BenchmarkSource: p.Cells[19]})
			}
		}
	}
	if err := assignKernelAndEvidenceBatches(&out, cellByKey); err != nil {
		return ClassificationV1{}, err
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		a, b := out.Rows[i], out.Rows[j]
		if a.Wave != b.Wave {
			return a.Wave < b.Wave
		}
		if a.FamilyContractDisplayID != b.FamilyContractDisplayID {
			return a.FamilyContractDisplayID < b.FamilyContractDisplayID
		}
		if a.CanonicalDependencyRank != b.CanonicalDependencyRank {
			return a.CanonicalDependencyRank < b.CanonicalDependencyRank
		}
		if a.SemanticOperationID != b.SemanticOperationID {
			return a.SemanticOperationID < b.SemanticOperationID
		}
		if a.ManifestOrdinal != b.ManifestOrdinal {
			return a.ManifestOrdinal < b.ManifestOrdinal
		}
		return a.RowKeyV1 < b.RowKeyV1
	})
	if err := validateClassificationEmission(out, cellByKey); err != nil {
		return ClassificationV1{}, err
	}
	return out, nil
}

func mustFamilyWave(family string) int { wave, _ := familyWave(family); return wave }

func validateClassificationEmission(classification ClassificationV1, cellByKey map[string]int) error {
	if len(classification.Rows) != 125 || len(classification.Cells) != 500 {
		return fmt.Errorf("classification emission has incomplete rows or cells")
	}
	batches := make(map[string]BatchRecordV1, len(classification.Batches))
	usedBatches := make(map[string]bool, len(classification.Batches))
	for _, batch := range classification.Batches {
		if _, duplicate := batches[batch.StorageID]; duplicate || len(batch.MemberStorageKeys) == 0 {
			return fmt.Errorf("duplicate or empty batch %q", batch.StorageID)
		}
		key, err := BatchKeyV1(batch.Kind, batch.DisplayID, batch.MemberStorageKeys)
		if err != nil || key.StorageID != batch.StorageID ||
			hex.EncodeToString(EncodeTupleV1(batch.MemberStorageKeys...)) != batch.MemberTupleHex {
			return fmt.Errorf("batch %q does not recompute", batch.StorageID)
		}
		if _, ok := familyWave(batch.FamilyContractDisplayID); !ok || batch.Sequence < 1 {
			return fmt.Errorf("batch %q has invalid family or sequence", batch.StorageID)
		}
		switch batch.Kind {
		case "scalar":
			if batch.Backend != "" || batch.DistinctSemanticOperationCount != 0 ||
				batch.DisplayID != fmt.Sprintf("SB-v1-%s-%02d", batch.FamilyContractDisplayID, batch.Sequence) ||
				len(batch.MemberStorageKeys) > 12 {
				return fmt.Errorf("scalar batch %q has invalid metadata or size", batch.StorageID)
			}
		case "evidence":
			operations := map[string]bool{}
			for _, member := range batch.MemberStorageKeys {
				index, ok := cellByKey[member]
				if !ok || classification.Cells[index].Backend != batch.Backend ||
					classification.Cells[index].FamilyContractDisplayID != batch.FamilyContractDisplayID {
					return fmt.Errorf("evidence batch %q has invalid cell member", batch.StorageID)
				}
				operations[classification.Cells[index].SemanticOperationID] = true
			}
			if !validMembershipBackend(batch.Backend) ||
				!strings.HasPrefix(batch.DisplayID, "EB-v1-KB-v1-"+batch.FamilyContractDisplayID+"-"+batch.Backend+"-") ||
				len(batch.MemberStorageKeys) > 12 || len(operations) != batch.DistinctSemanticOperationCount {
				return fmt.Errorf("evidence batch %q has invalid metadata or size", batch.StorageID)
			}
		case "kernel":
			operations := map[string]bool{}
			for _, member := range batch.MemberStorageKeys {
				index, ok := cellByKey[member]
				if !ok || classification.Cells[index].KernelOwnerCellKey != member ||
					classification.Cells[index].Backend != batch.Backend ||
					classification.Cells[index].FamilyContractDisplayID != batch.FamilyContractDisplayID {
					return fmt.Errorf("kernel batch %q has invalid owner member", batch.StorageID)
				}
				operations[classification.Cells[index].SemanticOperationID] = true
			}
			if !validMembershipBackend(batch.Backend) ||
				batch.DisplayID != fmt.Sprintf("KB-v1-%s-%s-%02d", batch.FamilyContractDisplayID, batch.Backend, batch.Sequence) ||
				len(operations) > 6 || len(operations) != batch.DistinctSemanticOperationCount {
				return fmt.Errorf("kernel batch %q has invalid operation metadata", batch.StorageID)
			}
		default:
			return fmt.Errorf("unknown batch kind %q", batch.Kind)
		}
		batches[batch.StorageID] = batch
	}

	rows := map[string]bool{}
	ranks := map[int]bool{}
	transactionRows := map[string][]ClassificationRowV1{}
	for _, row := range classification.Rows {
		if rows[row.RowKeyV1] || ranks[row.CanonicalRowRank] ||
			row.CanonicalRowRank != row.CanonicalDependencyRank {
			return fmt.Errorf("duplicate or inconsistent classification row %q", row.RowKeyV1)
		}
		rows[row.RowKeyV1] = true
		ranks[row.CanonicalRowRank] = true
		family, err := FamilyKeyV1(row.FamilyContractDisplayID)
		if err != nil || family.StorageID != row.FamilyStorageID ||
			row.APITransactionDisplayID != "AP-v1-"+row.FamilyContractDisplayID {
			return fmt.Errorf("row %q has invalid family or transaction identity", row.RowKeyV1)
		}
		scalar, ok := batches[row.ScalarBatchStorageID]
		if !ok || scalar.Kind != "scalar" || scalar.DisplayID != row.ScalarBatchDisplayID ||
			!containsStorageKey(scalar.MemberStorageKeys, row.RowKeyV1) {
			return fmt.Errorf("row %q has invalid scalar batch", row.RowKeyV1)
		}
		usedBatches[scalar.StorageID] = true
		transactionRows[row.APITransactionStorageID] = append(transactionRows[row.APITransactionStorageID], row)
	}
	for transactionID, members := range transactionRows {
		sort.Slice(members, func(i, j int) bool {
			return members[i].CanonicalRowRank < members[j].CanonicalRowRank
		})
		keys := make([]string, len(members))
		for i := range members {
			keys[i] = members[i].RowKeyV1
		}
		transaction, err := TransactionKeyV1(members[0].APITransactionDisplayID, keys)
		if err != nil || transaction.StorageID != transactionID {
			return fmt.Errorf("API transaction %q does not recompute", transactionID)
		}
	}

	kernelBatchBySharedKernel := map[string]string{}
	for i := range classification.Cells {
		cell := classification.Cells[i]
		key, err := CellKeyV1(cell.RowKeyV1, cell.Backend)
		if err != nil {
			return err
		}
		if cell.BackendOutcome != "eligible" {
			if cell.DirectSymbolStorageID != "" || cell.KernelBatchStorageID != "" ||
				cell.EvidenceBatchStorageID != "" {
				return fmt.Errorf("N/A cell %q has eligible-only classification", key.StorageID)
			}
			continue
		}
		kernel, ok := batches[cell.KernelBatchStorageID]
		evidence, evidenceOK := batches[cell.EvidenceBatchStorageID]
		if !ok || kernel.Kind != "kernel" || !containsStorageKey(kernel.MemberStorageKeys, cell.KernelOwnerCellKey) ||
			!evidenceOK || evidence.Kind != "evidence" || !containsStorageKey(evidence.MemberStorageKeys, key.StorageID) {
			return fmt.Errorf("eligible cell %q has invalid kernel or evidence batch", key.StorageID)
		}
		group := cell.Backend + "\x00" + cell.SharedKernelID
		if prior := kernelBatchBySharedKernel[group]; prior != "" && prior != kernel.StorageID {
			return fmt.Errorf("shared kernel %q is split across batches", cell.SharedKernelID)
		}
		kernelBatchBySharedKernel[group] = kernel.StorageID
		usedBatches[kernel.StorageID] = true
		usedBatches[evidence.StorageID] = true
	}
	if len(rows) != 125 || len(ranks) != 125 || len(usedBatches) != len(batches) {
		return fmt.Errorf("classification emission has orphaned or incomplete membership")
	}
	return nil
}

func containsStorageKey(keys []string, want string) bool {
	return slices.Contains(keys, want)
}

func validMembershipBackend(backend string) bool {
	for _, allowed := range membershipBackends {
		if backend == allowed {
			return true
		}
	}
	return false
}

func classificationDependencies(deps []DependencyRecordV1, ranks, cellByKey map[string]int, cells []FinalCellV1) (string, int, error) {
	sort.Slice(deps, func(i, j int) bool { return dependencyEdgeKey(deps[i]) < dependencyEdgeKey(deps[j]) })
	values := make([]string, len(deps))
	rank := 125
	for i, d := range deps {
		values[i] = dependencyEdgeKey(d)
		if d.OwnerKind == "row" {
			value, ok := ranks[d.OwnerLogicalID]
			if !ok {
				return "", 0, fmt.Errorf("unknown row dependency %q", d.OwnerLogicalID)
			}
			if value < rank {
				rank = value
			}
		}
		if d.OwnerKind == "cell" {
			cell, ok := cellByKey[d.OwnerLogicalID]
			if !ok || cells[cell].KernelOwnerCellKey != d.OwnerLogicalID {
				return "", 0, fmt.Errorf("inconsistent cell dependency %q", d.OwnerLogicalID)
			}
			if cells[cell].CanonicalRowRank < rank {
				rank = cells[cell].CanonicalRowRank
			}
		}
	}
	if rank == 125 {
		rank = 0
	}
	return hex.EncodeToString(EncodeTupleV1(values...)), rank, nil
}

func assignKernelAndEvidenceBatches(out *ClassificationV1, cellByKey map[string]int) error {
	type group struct {
		family, backend, operation, kernel, owner string
		members                                   []int
	}
	groupsByID := map[string]*group{}
	sharedGroups := map[string]string{}
	for i := range out.Cells {
		c := out.Cells[i]
		if c.BackendOutcome != "eligible" {
			continue
		}
		id := strings.Join([]string{c.FamilyContractDisplayID, c.Backend, c.SemanticOperationID, c.SharedKernelID}, "\x00")
		sharedID := c.Backend + "\x00" + c.SharedKernelID
		if prior, exists := sharedGroups[sharedID]; exists && prior != id {
			return fmt.Errorf("shared kernel %q would be split", c.SharedKernelID)
		}
		sharedGroups[sharedID] = id
		g := groupsByID[id]
		if g == nil {
			g = &group{family: c.FamilyContractDisplayID, backend: c.Backend, operation: c.SemanticOperationID, kernel: c.SharedKernelID, owner: c.KernelOwnerCellKey}
			groupsByID[id] = g
		}
		if g.owner != c.KernelOwnerCellKey {
			return fmt.Errorf("inconsistent kernel owner")
		}
		g.members = append(g.members, i)
	}
	byFamilyBackend := map[string][]*group{}
	for _, g := range groupsByID {
		owner, ok := cellByKey[g.owner]
		if !ok || out.Cells[owner].FamilyContractDisplayID != g.family || out.Cells[owner].Backend != g.backend || out.Cells[owner].SemanticOperationID != g.operation || out.Cells[owner].SharedKernelID != g.kernel {
			return fmt.Errorf("kernel group has invalid owner %q", g.owner)
		}
		byFamilyBackend[g.family+"\x00"+g.backend] = append(byFamilyBackend[g.family+"\x00"+g.backend], g)
	}
	keys := make([]string, 0, len(byFamilyBackend))
	for key := range byFamilyBackend {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		groups := byFamilyBackend[key]
		sort.Slice(groups, func(i, j int) bool {
			a, b := groups[i], groups[j]
			if a.operation != b.operation {
				return a.operation < b.operation
			}
			if a.kernel != b.kernel {
				return a.kernel < b.kernel
			}
			return a.owner < b.owner
		})
		for at, sequence := 0, 1; at < len(groups); sequence++ {
			end, operations := at, map[string]bool{}
			for end < len(groups) {
				if !operations[groups[end].operation] && len(operations) == 6 {
					break
				}
				operations[groups[end].operation] = true
				end++
			}
			if end == at {
				return fmt.Errorf("kernel batch made no progress")
			}
			owners := make([]string, end-at)
			members := []int{}
			for i, g := range groups[at:end] {
				owners[i] = g.owner
				for _, member := range g.members {
					out.Cells[member].KernelBatchDisplayID = fmt.Sprintf("KB-v1-%s-%s-%02d", g.family, g.backend, sequence)
					members = append(members, member)
				}
			}
			display := out.Cells[members[0]].KernelBatchDisplayID
			batch, err := BatchKeyV1("kernel", display, owners)
			if err != nil {
				return err
			}
			for _, member := range members {
				out.Cells[member].KernelBatchStorageID = batch.StorageID
			}
			out.Batches = append(out.Batches, BatchRecordV1{Kind: "kernel", DisplayID: display, StorageID: batch.StorageID, FamilyContractDisplayID: groups[at].family, Backend: groups[at].backend, Sequence: sequence, DistinctSemanticOperationCount: len(operations), MemberTupleHex: hex.EncodeToString(EncodeTupleV1(owners...)), MemberStorageKeys: owners})
			sort.Slice(members, func(i, j int) bool {
				a, b := out.Cells[members[i]], out.Cells[members[j]]
				if a.CanonicalRowRank != b.CanonicalRowRank {
					return a.CanonicalRowRank < b.CanonicalRowRank
				}
				if a.SemanticOperationID != b.SemanticOperationID {
					return a.SemanticOperationID < b.SemanticOperationID
				}
				if a.SharedKernelID != b.SharedKernelID {
					return a.SharedKernelID < b.SharedKernelID
				}
				aKey, _ := CellKeyV1(a.RowKeyV1, a.Backend)
				bKey, _ := CellKeyV1(b.RowKeyV1, b.Backend)
				return aKey.StorageID < bKey.StorageID
			})
			for start, evidenceSequence := 0, 1; start < len(members); start, evidenceSequence = start+12, evidenceSequence+1 {
				end := min(start+12, len(members))
				ids := make([]string, end-start)
				evidenceOperations := map[string]bool{}
				for i, member := range members[start:end] {
					id, _ := CellKeyV1(out.Cells[member].RowKeyV1, out.Cells[member].Backend)
					ids[i] = id.StorageID
					evidenceOperations[out.Cells[member].SemanticOperationID] = true
				}
				evidenceDisplay := fmt.Sprintf("EB-v1-%s-%02d", display, evidenceSequence)
				evidence, err := BatchKeyV1("evidence", evidenceDisplay, ids)
				if err != nil {
					return err
				}
				out.Batches = append(out.Batches, BatchRecordV1{Kind: "evidence", DisplayID: evidenceDisplay, StorageID: evidence.StorageID, FamilyContractDisplayID: groups[at].family, Backend: groups[at].backend, Sequence: evidenceSequence, DistinctSemanticOperationCount: len(evidenceOperations), MemberTupleHex: hex.EncodeToString(EncodeTupleV1(ids...)), MemberStorageKeys: ids})
				for _, member := range members[start:end] {
					out.Cells[member].EvidenceBatchDisplayID = evidenceDisplay
					out.Cells[member].EvidenceBatchStorageID = evidence.StorageID
				}
			}
			at = end
		}
	}
	return nil
}

func RenderClassificationV1(rows []ClassificationRowV1) []byte {
	lines := []string{strings.Join(classificationHeaderV1[:], "\t")}
	for _, r := range rows {
		lines = append(lines, strings.Join([]string{"v1", r.RowKeyV1, strconv.Itoa(r.ManifestOrdinal), strconv.Itoa(r.CanonicalRowRank), strconv.Itoa(r.CanonicalDependencyRank), r.GoSymbol, r.GoSignature, r.FamilyContractDisplayID, r.FamilyStorageID, strconv.Itoa(r.Wave), r.SemanticOperationID, r.DependencyTupleHex, r.ScalarBatchDisplayID, r.ScalarBatchStorageID, r.APITransactionDisplayID, r.APITransactionStorageID, r.ScalarSource, r.TestSource, r.FuzzSource, r.BenchmarkSource}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func RenderBatchesV1(rows []BatchRecordV1) []byte {
	lines := []string{strings.Join(batchesHeaderV1[:], "\t")}
	for _, r := range rows {
		lines = append(lines, strings.Join([]string{"v1", r.Kind, r.DisplayID, r.StorageID, r.FamilyContractDisplayID, r.Backend, strconv.Itoa(r.Sequence), strconv.Itoa(r.DistinctSemanticOperationCount), strconv.Itoa(len(r.MemberStorageKeys)), r.MemberTupleHex}, "\t"))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}
