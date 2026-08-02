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
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

func TestTupleV1RoundTripExactBytes(t *testing.T) {
	fields := []string{"", "a:1f", "é\x00\x1f", "Case", "case"}
	want := "0:4:a:1f4:é\x00\x1f4:Case4:case"
	encoded := EncodeTupleV1(fields...)
	if string(encoded) != want {
		t.Fatalf("EncodeTupleV1 = %q, want %q", encoded, want)
	}
	decoded, err := DecodeTupleV1(encoded, len(fields))
	if err != nil || strings.Join(decoded, "\x00") != strings.Join(fields, "\x00") {
		t.Fatalf("DecodeTupleV1 = %#v, %v; want %#v", decoded, err, fields)
	}
	hexEncoded := hex.EncodeToString(encoded)
	decoded, err = DecodeTupleHexV1(hexEncoded, len(fields))
	if err != nil || strings.Join(decoded, "\x00") != strings.Join(fields, "\x00") {
		t.Fatalf("DecodeTupleHexV1 = %#v, %v; want %#v", decoded, err, fields)
	}
}

func TestTupleV1RejectsMalformedInputs(t *testing.T) {
	for _, tuple := range []string{"", "x:field", ":field", "01:a", "1a:x", "1", "2:x", "1:xextra"} {
		if _, err := DecodeTupleV1([]byte(tuple), 1); err == nil {
			t.Errorf("DecodeTupleV1(%q) accepted malformed tuple", tuple)
		}
	}
	for _, encoded := range []string{"0", "0G", "0A"} {
		if _, err := DecodeTupleHexV1(encoded, 0); err == nil {
			t.Errorf("DecodeTupleHexV1(%q) accepted malformed hex", encoded)
		}
	}
	if _, err := DecodeTupleV1([]byte("999999999999999999999999999999999999999999999999999999999999999999:"), 1); err == nil {
		t.Error("DecodeTupleV1 accepted overflowing length")
	}
	if _, err := DecodeTupleV1(nil, -1); err == nil {
		t.Error("DecodeTupleV1 accepted negative field count")
	}
}

func TestLogicalKeyV1WrappersExactPreimages(t *testing.T) {
	row := RowKeyV1([6]string{"backend", "isa", "operation", "source", "file", "symbol"})
	family, err := FamilyKeyV1("Family-ID")
	if err != nil {
		t.Fatal(err)
	}
	cell, err := CellKeyV1(row, "haswell")
	if err != nil {
		t.Fatal(err)
	}
	symbol, err := SymbolKeyV1("haswell", "direct_symbol")
	if err != nil {
		t.Fatal(err)
	}
	batch, err := BatchKeyV1("kernel", "batch-1", []string{family.StorageID, symbol.StorageID})
	if err != nil {
		t.Fatal(err)
	}
	manifest := []byte("canonical\nmanifest\n")
	campaign, err := CampaignKeyV1("campaign-a", manifest)
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := TransactionKeyV1("transaction-1", []string{row})
	if err != nil {
		t.Fatal(err)
	}

	manifestSum := sha256.Sum256(manifest)
	cases := []struct {
		name, tag, display string
		record             KeyRecord
		fields             []string
	}{
		{"family", "family", "Family-ID", family, []string{"Family-ID"}},
		{"cell", "cell", "", cell, []string{row, "haswell"}},
		{"symbol", "symbol", "direct_symbol", symbol, []string{"haswell", "direct_symbol"}},
		{"batch", "batch", "batch-1", batch, []string{"kernel", "batch-1", independentTuple(family.StorageID, symbol.StorageID)}},
		{"campaign", "campaign", "campaign-a", campaign, []string{hex.EncodeToString(manifestSum[:])}},
		{"transaction", "transaction", "transaction-1", transaction, []string{"transaction-1", independentTuple(row)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tuple := independentTuple(tc.fields...)
			wantID := independentID(tc.tag+"-v1-", "simdutf-port-key-v1\x00"+independentTuple(tc.tag)+tuple)
			if tc.record.Kind != tc.tag || tc.record.DisplayID != tc.display || tc.record.TupleHex != hex.EncodeToString([]byte(tuple)) || tc.record.StorageID != wantID {
				t.Fatalf("record = %#v, want exact %s preimage", tc.record, tc.name)
			}
		})
	}
}

func TestLogicalKeyV1MetadataAndOrder(t *testing.T) {
	rowA := RowKeyV1([6]string{"a", "b", "c", "d", "e", "f"})
	rowB := RowKeyV1([6]string{"b", "b", "c", "d", "e", "f"})
	cell, err := CellKeyV1(rowA, "neon")
	if err != nil {
		t.Fatal(err)
	}
	if cell.DisplayID != "" {
		t.Fatalf("CellKeyV1 DisplayID = %q, want empty metadata", cell.DisplayID)
	}
	campaignA, err := CampaignKeyV1("campaign-a", []byte("manifest"))
	if err != nil {
		t.Fatal(err)
	}
	campaignB, err := CampaignKeyV1("campaign-b", []byte("manifest"))
	if err != nil {
		t.Fatal(err)
	}
	if campaignA.StorageID != campaignB.StorageID {
		t.Fatal("campaign display metadata changed storage hash")
	}
	campaignC, err := CampaignKeyV1("campaign-c", []byte("different manifest"))
	if err != nil {
		t.Fatal(err)
	}
	first, err := TransactionKeyV1("transaction", []string{rowA, rowB})
	if err != nil {
		t.Fatal(err)
	}
	second, err := TransactionKeyV1("transaction", []string{rowB, rowA})
	if err != nil {
		t.Fatal(err)
	}
	if first.StorageID == second.StorageID {
		t.Fatal("transaction member order did not change storage hash")
	}
	firstBatch, err := BatchKeyV1("kernel", "batch", []string{campaignA.StorageID, campaignC.StorageID})
	if err != nil {
		t.Fatal(err)
	}
	secondBatch, err := BatchKeyV1("kernel", "batch", []string{campaignC.StorageID, campaignA.StorageID})
	if err != nil {
		t.Fatal(err)
	}
	if firstBatch.StorageID == secondBatch.StorageID {
		t.Fatal("batch member order did not change storage hash")
	}
}

func TestLogicalKeyV1RejectsInvalidInputs(t *testing.T) {
	validRow := RowKeyV1([6]string{"a", "b", "c", "d", "e", "f"})
	validFamily, err := FamilyKeyV1("family")
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range []func() error{
		func() error { _, err := FamilyKeyV1(""); return err },
		func() error { _, err := CellKeyV1("rk-v1-not-a-digest", "neon"); return err },
		func() error { _, err := CellKeyV1(validRow, "avx2"); return err },
		func() error { _, err := SymbolKeyV1("avx2", "symbol"); return err },
		func() error { _, err := SymbolKeyV1("neon", ""); return err },
		func() error { _, err := BatchKeyV1("other", "batch", []string{validFamily.StorageID}); return err },
		func() error { _, err := BatchKeyV1("scalar", "", []string{validFamily.StorageID}); return err },
		func() error { _, err := BatchKeyV1("scalar", "batch", nil); return err },
		func() error {
			_, err := BatchKeyV1("scalar", "batch", []string{validFamily.StorageID, validFamily.StorageID})
			return err
		},
		func() error { _, err := BatchKeyV1("scalar", "batch", []string{"family-v1-not-a-digest"}); return err },
		func() error { _, err := CampaignKeyV1("", nil); return err },
		func() error { _, err := TransactionKeyV1("", []string{validRow}); return err },
		func() error { _, err := TransactionKeyV1("transaction", nil); return err },
		func() error { _, err := TransactionKeyV1("transaction", []string{validRow, validRow}); return err },
		func() error { _, err := TransactionKeyV1("transaction", []string{"rk-v1-not-a-digest"}); return err },
	} {
		if err := call(); err == nil {
			t.Error("logical key wrapper accepted invalid input")
		}
	}
}

func TestSpecializedIDsV1ExactVectors(t *testing.T) {
	fields := [6]string{"backend", "isa", "operation", "source", "file", "symbol"}
	row := RowKeyV1(fields)
	wantRow := independentID("rk-v1-", "backend\x1fisa\x1foperation\x1fsource\x1ffile\x1fsymbol")
	if row != wantRow || !validID(row, "rk-v1-") {
		t.Fatalf("RowKeyV1 = %q, want %q", row, wantRow)
	}
	ledger, err := LedgerOperationIDV1(7, "validate UTF-8\x1fexact")
	if err != nil {
		t.Fatal(err)
	}
	if want := independentID("op-v1-", "isa-ledger-v1\x1f7\x1fvalidate UTF-8\x1fexact"); ledger != want {
		t.Fatalf("LedgerOperationIDV1 = %q, want %q", ledger, want)
	}
	scalar, err := ScalarOperationIDV1(row)
	if err != nil {
		t.Fatal(err)
	}
	if want := independentID("op-v1-", "scalar-v1\x1f"+row); scalar != want {
		t.Fatalf("ScalarOperationIDV1 = %q, want %q", scalar, want)
	}
	shared, err := SharedKernelIDV1("avx2", "validate_utf8")
	if err == nil || shared != "" {
		t.Fatal("SharedKernelIDV1 accepted invalid backend")
	}
	shared, err = SharedKernelIDV1("haswell", "validate_utf8")
	if err != nil {
		t.Fatal(err)
	}
	if want := independentID("shared-kernel-v1-", "simdutf-port-shared-kernel-v1\x007:haswell13:validate_utf8"); shared != want {
		t.Fatalf("SharedKernelIDV1 = %q, want %q", shared, want)
	}
}

func TestSpecializedIDsV1RejectInvalidInputs(t *testing.T) {
	for _, ordinal := range []int{0, -1} {
		if _, err := LedgerOperationIDV1(ordinal, "semantic"); err == nil {
			t.Errorf("LedgerOperationIDV1 accepted ordinal %d", ordinal)
		}
	}
	for _, row := range []string{"", "rk-v1-not-a-digest", "rk-v1-" + strings.Repeat("A", 64), "op-v1-" + strings.Repeat("0", 64)} {
		if _, err := ScalarOperationIDV1(row); err == nil {
			t.Errorf("ScalarOperationIDV1 accepted malformed row key %q", row)
		}
	}
	for _, backend := range []string{"", "Haswell", "avx2", "scalar", "neon "} {
		if _, err := SharedKernelIDV1(backend, "kernel"); err == nil {
			t.Errorf("SharedKernelIDV1 accepted backend %q", backend)
		}
	}
	if _, err := SharedKernelIDV1("neon", ""); err == nil {
		t.Error("SharedKernelIDV1 accepted empty kernel name")
	}
}

func independentID(prefix, raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return prefix + hex.EncodeToString(sum[:])
}

func independentTuple(fields ...string) string {
	var tuple strings.Builder
	for _, field := range fields {
		tuple.WriteString(strconv.Itoa(len(field)))
		tuple.WriteByte(':')
		tuple.WriteString(field)
	}
	return tuple.String()
}

func TestIndependentLPVector(t *testing.T) {
	// Keep the vector construction independent of EncodeTupleV1.
	raw := strconv.Itoa(len("é")) + ":é"
	if got := string(EncodeTupleV1("é")); got != raw {
		t.Fatalf("EncodeTupleV1 non-ASCII = %q, want %q", got, raw)
	}
}
