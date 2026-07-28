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

// Package portplan provides deterministic identifiers for Phase 0 artifacts.
package portplan

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
)

const (
	keyDomain          = "simdutf-port-key-v1"
	ledgerDomain       = "isa-ledger-v1"
	scalarDomain       = "scalar-v1"
	sharedKernelDomain = "simdutf-port-shared-kernel-v1"
)

// KeyRecord is the serialized identity carried by Phase 0 receipts and indexes.
type KeyRecord struct {
	Kind, StorageID, DisplayID, TupleHex string
}

// EncodeTupleV1 encodes fields as canonical decimal byte-length-prefixed values.
func EncodeTupleV1(fields ...string) []byte {
	length := 0
	for _, field := range fields {
		length += len(strconv.Itoa(len(field))) + 1 + len(field)
	}
	encoded := make([]byte, 0, length)
	for _, field := range fields {
		encoded = strconv.AppendInt(encoded, int64(len(field)), 10)
		encoded = append(encoded, ':')
		encoded = append(encoded, field...)
	}
	return encoded
}

// DecodeTupleV1 decodes exactly fieldCount canonical length-prefixed fields.
func DecodeTupleV1(tuple []byte, fieldCount int) ([]string, error) {
	if fieldCount < 0 {
		return nil, errors.New("negative tuple field count")
	}
	if fieldCount > len(tuple)/2 {
		return nil, errors.New("tuple has too many fields")
	}

	fields := make([]string, 0, fieldCount)
	at := 0
	for len(fields) < fieldCount {
		if at == len(tuple) {
			return nil, errors.New("truncated tuple length")
		}
		start := at
		if tuple[at] < '0' || tuple[at] > '9' {
			return nil, errors.New("tuple length is not decimal")
		}
		if tuple[at] == '0' && at+1 < len(tuple) && tuple[at+1] != ':' {
			return nil, errors.New("tuple length has leading zero")
		}

		length := 0
		for at < len(tuple) && tuple[at] != ':' {
			c := tuple[at]
			if c < '0' || c > '9' {
				return nil, errors.New("tuple length is not decimal")
			}
			digit := int(c - '0')
			if length > (int(^uint(0)>>1)-digit)/10 {
				return nil, errors.New("tuple length overflows int")
			}
			length = length*10 + digit
			at++
		}
		if at == start || at == len(tuple) {
			return nil, errors.New("tuple length is missing separator")
		}
		at++
		if length > len(tuple)-at {
			return nil, errors.New("tuple field is truncated")
		}
		fields = append(fields, string(tuple[at:at+length]))
		at += length
	}
	if at != len(tuple) {
		return nil, errors.New("tuple has trailing data")
	}
	return fields, nil
}

// DecodeTupleHexV1 decodes lowercase hexadecimal tuple bytes.
func DecodeTupleHexV1(encoded string, fieldCount int) ([]string, error) {
	if len(encoded)%2 != 0 {
		return nil, errors.New("odd-length tuple hex")
	}
	for i := range encoded {
		c := encoded[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, fmt.Errorf("tuple hex contains non-lowercase-hex byte %q", c)
		}
	}
	tuple, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode tuple hex: %w", err)
	}
	return DecodeTupleV1(tuple, fieldCount)
}

// FamilyKeyV1 returns the identifier for a manifest family.
func FamilyKeyV1(displayID string) (KeyRecord, error) {
	if displayID == "" {
		return KeyRecord{}, errors.New("family display ID is empty")
	}
	return logicalKeyV1("family", displayID, displayID)
}

// CellKeyV1 returns the identifier for one backend-specific manifest cell.
func CellKeyV1(rowKey, backend string) (KeyRecord, error) {
	if !validID(rowKey, "rk-v1-") {
		return KeyRecord{}, fmt.Errorf("invalid row key %q", rowKey)
	}
	if !validBackend(backend) {
		return KeyRecord{}, fmt.Errorf("invalid backend %q", backend)
	}
	return logicalKeyV1("cell", "", rowKey, backend)
}

// SymbolKeyV1 returns the identifier for one direct backend symbol.
func SymbolKeyV1(backend, directSymbol string) (KeyRecord, error) {
	if !validBackend(backend) {
		return KeyRecord{}, fmt.Errorf("invalid backend %q", backend)
	}
	if directSymbol == "" {
		return KeyRecord{}, errors.New("direct symbol is empty")
	}
	return logicalKeyV1("symbol", directSymbol, backend, directSymbol)
}

// BatchKeyV1 returns the identifier for an ordered batch of storage keys.
func BatchKeyV1(batchKind, displayID string, orderedMemberStorageKeys []string) (KeyRecord, error) {
	switch batchKind {
	case "scalar", "kernel", "evidence":
	default:
		return KeyRecord{}, fmt.Errorf("invalid batch kind %q", batchKind)
	}
	if displayID == "" {
		return KeyRecord{}, errors.New("batch display ID is empty")
	}
	if err := validateUniqueStorageKeys(orderedMemberStorageKeys); err != nil {
		return KeyRecord{}, err
	}
	return logicalKeyV1("batch", displayID, batchKind, displayID, string(EncodeTupleV1(orderedMemberStorageKeys...)))
}

// CampaignKeyV1 returns the identifier for a campaign manifest.
func CampaignKeyV1(displayID string, canonicalManifest []byte) (KeyRecord, error) {
	if displayID == "" {
		return KeyRecord{}, errors.New("campaign display ID is empty")
	}
	sum := sha256.Sum256(canonicalManifest)
	return logicalKeyV1("campaign", displayID, hex.EncodeToString(sum[:]))
}

// TransactionKeyV1 returns the identifier for an ordered set of manifest rows.
func TransactionKeyV1(displayID string, orderedRowKeys []string) (KeyRecord, error) {
	if displayID == "" {
		return KeyRecord{}, errors.New("transaction display ID is empty")
	}
	if err := validateUniqueRowKeys(orderedRowKeys); err != nil {
		return KeyRecord{}, err
	}
	return logicalKeyV1("transaction", displayID, displayID, string(EncodeTupleV1(orderedRowKeys...)))
}

func logicalKeyV1(tag, displayID string, fields ...string) (KeyRecord, error) {
	tuple := EncodeTupleV1(fields...)
	storageID := hashID(tag+"-v1-", []byte(keyDomain), []byte{0}, EncodeTupleV1(tag), tuple)
	if !validID(storageID, tag+"-v1-") {
		return KeyRecord{}, errors.New("invalid logical key digest")
	}
	return KeyRecord{Kind: tag, StorageID: storageID, DisplayID: displayID, TupleHex: hex.EncodeToString(tuple)}, nil
}

// RowKeyV1 returns the identifier for the six exact manifest fields.
func RowKeyV1(fields [6]string) string {
	return hashID("rk-v1-", []byte(fields[0]), []byte{0x1f}, []byte(fields[1]), []byte{0x1f}, []byte(fields[2]), []byte{0x1f}, []byte(fields[3]), []byte{0x1f}, []byte(fields[4]), []byte{0x1f}, []byte(fields[5]))
}

// LedgerOperationIDV1 returns the identifier for one positive ISA ledger ordinal.
func LedgerOperationIDV1(ordinal int, semantic string) (string, error) {
	if ordinal <= 0 {
		return "", errors.New("ledger ordinal must be positive")
	}
	id := hashID("op-v1-", []byte(ledgerDomain), []byte{0x1f}, []byte(strconv.Itoa(ordinal)), []byte{0x1f}, []byte(semantic))
	if !validID(id, "op-v1-") {
		return "", errors.New("invalid ledger operation digest")
	}
	return id, nil
}

// ScalarOperationIDV1 returns the scalar operation identifier for a row key.
func ScalarOperationIDV1(rowKey string) (string, error) {
	if !validID(rowKey, "rk-v1-") {
		return "", fmt.Errorf("invalid row key %q", rowKey)
	}
	id := hashID("op-v1-", []byte(scalarDomain), []byte{0x1f}, []byte(rowKey))
	if !validID(id, "op-v1-") {
		return "", errors.New("invalid scalar operation digest")
	}
	return id, nil
}

// SharedKernelIDV1 returns the identifier for a reviewed backend kernel pair.
func SharedKernelIDV1(backend, canonicalKernelName string) (string, error) {
	if !validBackend(backend) {
		return "", fmt.Errorf("invalid backend %q", backend)
	}
	if canonicalKernelName == "" {
		return "", errors.New("canonical kernel name is empty")
	}
	id := hashID("shared-kernel-v1-", []byte(sharedKernelDomain), []byte{0}, EncodeTupleV1(backend), EncodeTupleV1(canonicalKernelName))
	if !validID(id, "shared-kernel-v1-") {
		return "", errors.New("invalid shared kernel digest")
	}
	return id, nil
}

func hashID(prefix string, parts ...[]byte) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write(part)
	}
	return prefix + hex.EncodeToString(h.Sum(nil))
}

func validBackend(backend string) bool {
	switch backend {
	case "westmere", "haswell", "archsimd", "neon":
		return true
	default:
		return false
	}
}

func validStorageID(id string) bool {
	for _, tag := range []string{"family", "cell", "symbol", "batch", "campaign", "transaction"} {
		if validID(id, tag+"-v1-") {
			return true
		}
	}
	return false
}

func validateUniqueStorageKeys(keys []string) error {
	if len(keys) == 0 {
		return errors.New("batch members are empty")
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if !validStorageID(key) {
			return fmt.Errorf("invalid storage key %q", key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate storage key %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateUniqueRowKeys(keys []string) error {
	if len(keys) == 0 {
		return errors.New("transaction row keys are empty")
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if !validID(key, "rk-v1-") {
			return fmt.Errorf("invalid row key %q", key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate row key %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validID(id, prefix string) bool {
	if len(id) != len(prefix)+sha256.Size*2 || id[:len(prefix)] != prefix {
		return false
	}
	for _, c := range id[len(prefix):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
