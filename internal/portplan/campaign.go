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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"strings"
)

// CommandOutputV1 identifies one required typed command artifact.
type CommandOutputV1 struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	Required  bool   `json:"required"`
}

// CampaignCommandV1 is one shell-free, typed campaign command.
type CampaignCommandV1 struct {
	Ordinal        int               `json:"ordinal"`
	ID             string            `json:"id"`
	Action         string            `json:"action"`
	Role           string            `json:"role"`
	Argv           []string          `json:"argv"`
	CWD            string            `json:"cwd"`
	Env            map[string]string `json:"env"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	ExpectedExit   int               `json:"expected_exit"`
	Outputs        []CommandOutputV1 `json:"outputs"`
	OperationID    string            `json:"operation_id"`
	BatchID        string            `json:"batch_id"`
	RowID          string            `json:"row_id"`
	CellID         string            `json:"cell_id"`
	Provider       string            `json:"provider"`
}

// CommandRequirementV1 is one exact required command population.
type CommandRequirementV1 struct {
	Action, Role, OperationID, BatchID, RowID, CellID, Provider string
	Count                                                       int
}

// CommandValidationContextV1 binds a manifest to already-validated inputs.
type CommandValidationContextV1 struct {
	Evidence         EvidenceValidationContextV1
	ExpectedCommands []CampaignCommandV1
	Requirements     []CommandRequirementV1
}

var campaignActionsV1 = map[string]bool{
	"source_commit": true, "source_tree": true, "source_parent": true, "source_status": true,
	"host_uname": true, "host_cpu": true, "go_version": true, "file_digest": true,
	"go_test_focused": true, "go_test_full": true, "go_test_race": true,
	"go_fuzz_replay": true, "go_fuzz": true, "go_benchmark": true, "benchstat": true,
	"provider_guard": true, "selector_test": true, "final_selector_test": true, "go_object_build": true, "go_objdump": true,
	"quiet_affinity_recheck": true, "state_transition": true, "not_applicable": true, "return_index": true,
}
var campaignRolesV1 = map[string]bool{"host": true, "old": true, "new": true, "direct": true, "selector": true, "object": true}

// CampaignCommandTopologyKeyV1 returns the exact command coverage key.
func CampaignCommandTopologyKeyV1(command CampaignCommandV1) string {
	key := []string{command.Action, command.Role, command.OperationID, command.BatchID, command.RowID, command.CellID, command.Provider}
	if command.Action == "file_digest" {
		key = append(key, command.Argv[len(command.Argv)-1])
	}
	return strings.Join(key, "\x00")
}

// CampaignCacheRootV1 returns the only cache root permitted for a source role.
func CampaignCacheRootV1(context EvidenceValidationContextV1, sourceRole string) (string, error) {
	if sourceRole != "old" && sourceRole != "new" {
		return "", errors.New("invalid cache source role")
	}
	if !validID(context.CampaignID, "campaign-v1-") {
		return "", errors.New("invalid cache campaign")
	}
	os, _, _ := laneSettingsV1(context.LaneID)
	if os == "" {
		return "", errors.New("invalid cache lane")
	}
	root := "/home/zchee/.cache/gjc/simdutf-port/v1/"
	if os == "darwin" {
		root = "/Users/zchee/.cache/gjc/simdutf-port/v1/"
	}
	return root + context.CampaignID + "/cache/" + sourceRole + "/", nil
}

// RenderCanonicalCampaignCommandsV1 validates and renders canonical manifest JSON.
func RenderCanonicalCampaignCommandsV1(commands []CampaignCommandV1) ([]byte, error) {
	if len(commands) == 0 {
		return nil, errors.New("command array is empty")
	}
	ids := make(map[string]bool, len(commands))
	for i := range commands {
		if ids[commands[i].ID] {
			return nil, errors.New("duplicate command ID")
		}
		ids[commands[i].ID] = true
		if err := validateCampaignCommandV1(commands[i], i+1, EvidenceValidationContextV1{}); err != nil {
			return nil, err
		}
	}
	encoded, err := json.Marshal(commands)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// ValidateCampaignCommandsV1 rejects noncanonical JSON and anything other than
// the trusted, required command topology.
func ValidateCampaignCommandsV1(input []byte, context CommandValidationContextV1) ([]CampaignCommandV1, error) {
	if err := validateContext(context.Evidence); err != nil {
		return nil, err
	}
	var commands []CampaignCommandV1
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&commands); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, errors.New("trailing JSON data")
	}
	for i := range commands {
		if err := validateCampaignCommandV1(commands[i], i+1, context.Evidence); err != nil {
			return nil, err
		}
	}
	canonical, err := RenderCanonicalCampaignCommandsV1(commands)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(input, canonical) {
		return nil, errors.New("noncanonical command JSON")
	}
	expected, err := RenderCanonicalCampaignCommandsV1(context.ExpectedCommands)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(expected)
	if context.Evidence.CommandManifestSHA256 != hex.EncodeToString(sum[:]) {
		return nil, errors.New("trusted command digest does not match context")
	}
	if !bytes.Equal(canonical, expected) {
		return nil, errors.New("command array does not match trusted input")
	}
	if err := validateCommandRequirementsV1(commands, context.Requirements); err != nil {
		return nil, err
	}
	return commands, nil
}

func validateCampaignCommandV1(c CampaignCommandV1, ordinal int, evidence EvidenceValidationContextV1) error {
	if c.Ordinal != ordinal || !safeEvidencePart(c.ID) || !campaignActionsV1[c.Action] || !campaignRolesV1[c.Role] || c.TimeoutSeconds < 1 || c.ExpectedExit != 0 || !canonicalProviderV1(c.Provider) {
		return errors.New("invalid command identity")
	}
	if !allowedActionRoleV1(c.Action, c.Role) {
		return errors.New("invalid command action and role")
	}
	if c.CWD != commandCWDV1(c.Action, c.Role) {
		return errors.New("invalid command cwd")
	}
	if !validID(c.OperationID, "op-v1-") || !validID(c.BatchID, "batch-v1-") || !validID(c.RowID, "rk-v1-") || !validID(c.CellID, "cell-v1-") {
		return errors.New("invalid command bindings")
	}
	if evidence.Provider != "" && c.Provider != evidence.Provider {
		return errors.New("command provider does not match context")
	}
	if err := validateArgvV1(c); err != nil {
		return err
	}
	if err := validateEnvV1(c, evidence); err != nil {
		return err
	}
	return validateOutputsV1(c)
}

func canonicalProviderV1(provider string) bool {
	switch provider {
	case "westmere", "haswell", "archsimd", "neon":
		return true
	}
	return false
}

func allowedActionRoleV1(action, role string) bool {
	switch action {
	case "source_commit", "source_tree", "source_parent", "source_status":
		return role == "old" || role == "new"
	case "host_uname", "host_cpu", "go_version", "file_digest":
		return role == "host"
	case "go_test_focused", "go_test_full", "go_test_race":
		return role == "old" || role == "new" || role == "direct"
	case "go_fuzz_replay", "go_fuzz", "provider_guard", "quiet_affinity_recheck", "state_transition", "not_applicable", "return_index":
		return role == "direct"
	case "go_benchmark":
		return role == "old" || role == "new"
	case "benchstat":
		return role == "new"
	case "selector_test", "final_selector_test":
		return role == "selector"
	case "go_object_build", "go_objdump":
		return role == "object"
	}
	return false
}
func commandCWDV1(action, role string) string {
	if role == "old" {
		return "source/old"
	}
	if role == "host" {
		return "host"
	}
	return "source/new"
}

func validateArgvV1(c CampaignCommandV1) error {
	for _, a := range c.Argv {
		if a == "" || strings.ContainsAny(a, "\x00\r\n|&<>()`*?[]\\") || strings.Contains(a, "${") {
			return errors.New("unsafe command argv")
		}
	}
	goBin := "/home/zchee/sdk/go1.26.5/bin/go"
	if c.Env["GOOS"] == "darwin" {
		goBin = "/Users/zchee/sdk/go1.26.5/bin/go"
	}
	switch c.Action {
	case "source_commit", "source_tree", "source_parent", "source_status":
		return exactArgsV1(c.Argv, []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "source-identity", "--action=" + c.Action, "--role=" + c.Role, "--receipt=staging/identity/" + c.Role + ".json", "--archive=staging/source/" + c.Role + ".tar"})
	case "host_uname":
		return exactArgsV1(c.Argv, []string{"/usr/bin/uname", "-srm"})
	case "host_cpu":
		if c.Env["GOOS"] == "darwin" {
			return exactArgsV1(c.Argv, []string{"/usr/sbin/sysctl", "-n", "machdep.cpu.brand_string", "hw.logicalcpu"})
		}
		return exactArgsV1(c.Argv, []string{"/usr/bin/lscpu", "--json"})
	case "go_version":
		return exactArgsV1(c.Argv, []string{goBin, "version"})
	case "file_digest":
		prefix := []string{"/usr/bin/sha256sum"}
		if c.Env["GOOS"] == "darwin" {
			prefix = []string{"/usr/bin/shasum", "-a", "256"}
		}
		if err := literalDigestArgsV1(c.Argv, prefix); err != nil {
			return err
		}
		switch c.Argv[len(c.Argv)-1] {
		case "staging/identity/go-binary", "staging/source/old.tar", "staging/source/new.tar", "staging/campaign-commands-v1.json", "staging/qualification-contract-v1.tsv", "staging/corpus-contract-v1.tsv", "staging/cache-policy-v1.json":
			return nil
		}
		return errors.New("digest target is not allowlisted")
	case "go_objdump":
		if len(c.Argv) != 5 || c.Argv[0] != goBin || c.Argv[1] != "tool" || c.Argv[2] != "objdump" || !literalObjectPathV1(c.Argv[3]) || !literalSymbolV1(c.Argv[4]) {
			return errors.New("invalid objdump profile")
		}
		return nil
	case "go_test_focused":
		if len(c.Argv) != 5 || c.Argv[0] != goBin || c.Argv[1] != "test" || !literalTestNameV1(c.Argv[2], "-run=^Test") || c.Argv[3] != "-count=1" || c.Argv[4] != "." {
			return errors.New("invalid focused test profile")
		}
		return nil
	case "go_test_full":
		return exactArgsV1(c.Argv, []string{goBin, "test", "."})
	case "go_test_race":
		if c.Env["GOOS"] != "darwin" && c.Env["GOOS"] != "linux" {
			return errors.New("race is unsupported on this platform")
		}
		return exactArgsV1(c.Argv, []string{goBin, "test", "-race", "."})
	case "go_fuzz_replay":
		if len(c.Argv) != 5 || c.Argv[0] != goBin || c.Argv[1] != "test" || !literalTestNameV1(c.Argv[2], "-run=^Fuzz") || c.Argv[3] != "-count=1" || c.Argv[4] != "." {
			return errors.New("invalid fuzz replay profile")
		}
		return nil
	case "go_fuzz":
		if len(c.Argv) != 4 || c.Argv[0] != goBin || c.Argv[1] != "test" || !literalTestNameV1(c.Argv[2], "-fuzz=^Fuzz") || c.Argv[3] != "." {
			return errors.New("invalid fuzz profile")
		}
		return nil
	case "go_benchmark":
		prefix := []string{goBin, "test"}
		if c.Env["GOOS"] == "linux" {
			cpu := c.Env["SIMDUTF_CPU"]
			if !literalCPUSetV1(cpu) {
				return errors.New("invalid Linux benchmark CPU")
			}
			prefix = []string{"/usr/bin/taskset", "-c", cpu, goBin, "test"}
		}
		if len(c.Argv) != len(prefix)+5 || !exactArgsEqualV1(c.Argv[:len(prefix)], prefix) || c.Argv[len(prefix)] != "-run=^$" || !literalTestNameV1(c.Argv[len(prefix)+1], "-bench=^Benchmark") || c.Argv[len(prefix)+2] != "-benchmem" || c.Argv[len(prefix)+3] != "-count=10" || c.Argv[len(prefix)+4] != "." {
			return errors.New("invalid benchmark profile")
		}
		return nil
	case "benchstat":
		return validateEvidenceProducerBenchstatArgsV1(c.Argv, goBin)
	case "go_object_build":
		return exactArgsV1(c.Argv, []string{goBin, "test", "-c", "."})
	case "provider_guard":
		return exactArgsV1(c.Argv, []string{goBin, "test", "-run=^TestProviderGuard$", "."})
	case "selector_test":
		return exactArgsV1(c.Argv, []string{goBin, "test", "-run=^TestSelector$", "."})
	case "final_selector_test":
		return exactArgsV1(c.Argv, []string{goBin, "test", "-run=^TestFinalSelector$", "."})
	case "quiet_affinity_recheck":
		if c.Env["GOOS"] != "linux" {
			return errors.New("quiet affinity recheck is Linux-only")
		}
		if !literalCPUSetV1(c.Env["SIMDUTF_CPU"]) {
			return errors.New("invalid affinity recheck CPU")
		}
		if c.Env["SIMDUTF_AFFINITY"] != "taskset:"+c.Env["SIMDUTF_CPU"] {
			return errors.New("invalid affinity recheck policy")
		}
		return exactArgsV1(c.Argv, []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "quiet-affinity-recheck", "--cpu=" + c.Env["SIMDUTF_CPU"], "--policy=" + c.Env["SIMDUTF_AFFINITY"]})
	case "state_transition", "not_applicable":
		return validateEvidenceProducerStateArgsV1(c.Argv, goBin, c.Action == "not_applicable")
	case "return_index":
		return exactArgsV1(c.Argv, []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "return-index", "--descriptor-dir=staging/descriptors"})
	default:
		return errors.New("unsupported command executable profile")
	}
}
func exactArgsV1(got, want []string) error {
	if len(got) != len(want) {
		return errors.New("incorrect command argv")
	}
	for i := range want {
		if got[i] != want[i] {
			return errors.New("incorrect command argv")
		}
	}
	return nil
}

func validateEvidenceProducerStateArgsV1(argv []string, goBin string, notApplicable bool) error {
	subcommand := "state-transition"
	if notApplicable {
		subcommand = "not-applicable"
	}
	prefix := []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", subcommand}
	required := []string{"--state-subject=", "--prerequisite-state=", "--current-state=", "--disposition=", "--go-qualification="}
	if notApplicable {
		required = append(required, "--na-reason=", "--na-source=")
	}
	if len(argv) < len(prefix)+len(required)+1 {
		return errors.New("incorrect command argv")
	}
	if !exactArgsEqualV1(argv[:len(prefix)], prefix) {
		return errors.New("incorrect command argv")
	}
	var naReason string
	for i, flagPrefix := range required {
		arg := argv[len(prefix)+i]
		if !strings.HasPrefix(arg, flagPrefix) {
			return errors.New("incorrect command argv")
		}
		value := strings.TrimPrefix(arg, flagPrefix)
		if strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("incorrect command argv")
		}
		switch flagPrefix {
		case "--state-subject=":
			if notApplicable {
				if value != "backend_cell" {
					return errors.New("incorrect command argv")
				}
			} else if value != "row" && value != "backend_cell" {
				return errors.New("incorrect command argv")
			}
		case "--prerequisite-state=", "--current-state=":
			if value == "" {
				return errors.New("incorrect command argv")
			}
			if notApplicable && flagPrefix == "--current-state=" && value != "not_applicable" {
				return errors.New("incorrect command argv")
			}
		case "--disposition=":
			if value != "" && value != "selected" && value != "direct_only" && value != "not_applicable" {
				return errors.New("incorrect command argv")
			}
			if notApplicable && value != "not_applicable" {
				return errors.New("incorrect command argv")
			}
		case "--go-qualification=":
			if value != "" && value != "pass" && value != "fail" && value != "inconclusive" {
				return errors.New("incorrect command argv")
			}
		case "--na-reason=":
			if !validNAReason(value) {
				return errors.New("incorrect command argv")
			}
			naReason = value
		case "--na-source=":
			if !validNAEvidenceSourceV1(naReason, value) {
				return errors.New("incorrect command argv")
			}
		}
	}
	proofs := argv[len(prefix)+len(required):]
	if len(proofs) == 0 {
		return errors.New("incorrect command argv")
	}
	if len(proofs) == 1 {
		if proofs[0] != "--proof-receipt-id=" && !literalProofReceiptFlagV1(proofs[0]) {
			return errors.New("incorrect command argv")
		}
		return nil
	}
	for _, proof := range proofs {
		if !literalProofReceiptFlagV1(proof) {
			return errors.New("incorrect command argv")
		}
	}
	return nil
}

func literalProofReceiptFlagV1(arg string) bool {
	const prefix = "--proof-receipt-id="
	if !strings.HasPrefix(arg, prefix) {
		return false
	}
	value := strings.TrimPrefix(arg, prefix)
	return strings.HasPrefix(value, "receipt-v1-") && lowerHex(strings.TrimPrefix(value, "receipt-v1-"), 64)
}

func validateEvidenceProducerBenchstatArgsV1(argv []string, goBin string) error {
	prefix := []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "benchstat"}
	required := []string{"--incumbent=", "--candidate=", "--incumbent-receipt-id=", "--candidate-receipt-id=", "--qualification-contract=", "--operation-id="}
	if len(argv) != len(prefix)+len(required) || !exactArgsEqualV1(argv[:len(prefix)], prefix) {
		return errors.New("invalid benchstat profile")
	}
	values := map[string]string{}
	for i, flagPrefix := range required {
		arg := argv[len(prefix)+i]
		if !strings.HasPrefix(arg, flagPrefix) {
			return errors.New("invalid benchstat profile")
		}
		value := strings.TrimPrefix(arg, flagPrefix)
		if value == "" || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("invalid benchstat profile")
		}
		values[flagPrefix] = value
	}
	if !literalObjectPathV1(values["--incumbent="]) || !literalObjectPathV1(values["--candidate="]) || values["--incumbent="] == values["--candidate="] {
		return errors.New("invalid benchstat profile")
	}
	if !literalObjectPathV1(values["--qualification-contract="]) {
		return errors.New("invalid benchstat profile")
	}
	if !strings.HasPrefix(values["--incumbent-receipt-id="], "receipt-v1-") || !lowerHex(strings.TrimPrefix(values["--incumbent-receipt-id="], "receipt-v1-"), 64) {
		return errors.New("invalid benchstat profile")
	}
	if !strings.HasPrefix(values["--candidate-receipt-id="], "receipt-v1-") || !lowerHex(strings.TrimPrefix(values["--candidate-receipt-id="], "receipt-v1-"), 64) {
		return errors.New("invalid benchstat profile")
	}
	if !strings.HasPrefix(values["--operation-id="], "op-v1-") || !lowerHex(strings.TrimPrefix(values["--operation-id="], "op-v1-"), 64) {
		return errors.New("invalid benchstat profile")
	}
	return nil
}

func benchstatInputPathsV1(argv []string) (string, string, bool) {
	var incumbent, candidate string
	for _, arg := range argv {
		switch {
		case strings.HasPrefix(arg, "--incumbent="):
			incumbent = strings.TrimPrefix(arg, "--incumbent=")
		case strings.HasPrefix(arg, "--candidate="):
			candidate = strings.TrimPrefix(arg, "--candidate=")
		}
	}
	return incumbent, candidate, incumbent != "" && candidate != "" && incumbent != candidate
}
func literalTestNameV1(arg, prefix string) bool {
	name := strings.TrimSuffix(strings.TrimPrefix(arg, prefix), "$")
	if !strings.HasPrefix(arg, prefix) || !strings.HasSuffix(arg, "$") || name == "" {
		return false
	}
	for _, r := range name {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_') {
			return false
		}
	}
	return true
}

func literalObjectPathV1(value string) bool {
	return strings.HasPrefix(value, "staging/") && path.Clean(value) == value && !strings.Contains(value, "..")
}

func literalSymbolV1(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("._/", r)) {
			return false
		}
	}
	return true
}
func literalCPUSetV1(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r == ',' || r == '-') {
			return false
		}
	}
	return true
}

func exactArgsEqualV1(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func literalDigestArgsV1(args, prefix []string) error {
	if len(args) != len(prefix)+1 || !literalObjectPathV1(args[len(args)-1]) {
		return errors.New("digest command requires one literal file path")
	}
	return exactArgsV1(args[:len(prefix)], prefix)
}

func commandOutputPathV1(c CampaignCommandV1, kind string) string {
	for _, output := range c.Outputs {
		if output.Kind == kind {
			return output.Path
		}
	}
	return ""
}

func validateEnvV1(c CampaignCommandV1, evidence EvidenceValidationContextV1) error {
	allowed := map[string]bool{"LC_ALL": true, "GOMAXPROCS": true, "GOEXPERIMENT": true, "GOCACHE": true, "GOMODCACHE": true, "CGO_ENABLED": true, "GOOS": true, "GOARCH": true, "GOAMD64": true, "SIMDUTF_PROVIDER": true, "SIMDUTF_OPERATION": true, "SIMDUTF_BENCH_EXPECT_OPERATION": true, "SIMDUTF_BENCH_EXPECT_TIER": true, "SIMDUTF_CPU": true, "SIMDUTF_AFFINITY": true}
	for k, v := range c.Env {
		if !allowed[k] || v == "" || strings.ContainsAny(v, "\x00\r\n") {
			return errors.New("invalid command environment")
		}
	}
	for _, k := range []string{"LC_ALL", "GOMAXPROCS", "GOEXPERIMENT", "GOCACHE", "GOMODCACHE", "CGO_ENABLED", "GOOS", "GOARCH"} {
		if c.Env[k] == "" {
			return errors.New("missing required command environment")
		}
	}
	if c.Env["LC_ALL"] != "C" || c.Env["GOMAXPROCS"] != "1" {
		return errors.New("invalid fixed command environment")
	}
	if c.Action == "go_test_race" {
		if c.Env["CGO_ENABLED"] != "1" {
			return errors.New("race requires CGO_ENABLED=1")
		}
	} else if c.Env["CGO_ENABLED"] != "0" {
		return errors.New("non-race command requires CGO_ENABLED=0")
	}
	if c.Env["GOEXPERIMENT"] != "nosimd,runtimesecret" && c.Env["GOEXPERIMENT"] != "simd,runtimesecret" && c.Env["GOEXPERIMENT"] != "none" && c.Env["GOEXPERIMENT"] != "simd" {
		return errors.New("invalid GOEXPERIMENT")
	}
	os, arch, experiment := laneSettingsV1(evidence.LaneID)
	if evidence.LaneID != "" && (c.Env["GOOS"] != os || c.Env["GOARCH"] != arch || c.Env["GOEXPERIMENT"] != experiment) {
		return errors.New("lane environment mismatch")
	}
	if arch == "amd64" || (evidence.LaneID == "" && c.Env["GOARCH"] == "amd64") {
		if c.Env["GOAMD64"] != "v1" {
			return errors.New("GOAMD64 must be v1")
		}
	} else if _, ok := c.Env["GOAMD64"]; ok {
		return errors.New("GOAMD64 is not valid for this architecture")
	}
	if err := validateCachesV1(c, evidence); err != nil {
		return err
	}
	if c.Action == "provider_guard" {
		if c.Env["SIMDUTF_PROVIDER"] != c.Provider || c.Env["SIMDUTF_OPERATION"] != c.OperationID {
			return errors.New("provider guard environment mismatch")
		}
	} else if c.Env["SIMDUTF_PROVIDER"] != "" || c.Env["SIMDUTF_OPERATION"] != "" {
		return errors.New("unexpected provider guard environment")
	}
	if c.Action == "go_benchmark" {
		tier := c.Provider
		if c.Role == "old" {
			tier = c.Env["SIMDUTF_BENCH_EXPECT_TIER"]
			if tier != "scalar" && !canonicalProviderV1(tier) {
				return errors.New("invalid incumbent benchmark tier")
			}
		}
		if c.Env["SIMDUTF_BENCH_EXPECT_OPERATION"] != c.OperationID || c.Env["SIMDUTF_BENCH_EXPECT_TIER"] != tier {
			return errors.New("benchmark environment mismatch")
		}
		if c.Env["GOOS"] == "linux" {
			if c.Env["SIMDUTF_AFFINITY"] != "taskset:"+c.Env["SIMDUTF_CPU"] || !literalCPUSetV1(c.Env["SIMDUTF_CPU"]) {
				return errors.New("invalid Linux benchmark affinity")
			}
		} else if c.Env["GOOS"] == "darwin" && (c.Env["SIMDUTF_AFFINITY"] != "none" || c.Env["SIMDUTF_CPU"] != "") {
			return errors.New("invalid Darwin benchmark affinity")
		}
	} else if c.Action == "quiet_affinity_recheck" {
		if c.Env["SIMDUTF_BENCH_EXPECT_OPERATION"] != "" || c.Env["SIMDUTF_BENCH_EXPECT_TIER"] != "" {
			return errors.New("unexpected benchmark environment")
		}
		if c.Env["GOOS"] != "linux" {
			return errors.New("quiet affinity recheck is Linux-only")
		}
		if c.Env["SIMDUTF_AFFINITY"] != "taskset:"+c.Env["SIMDUTF_CPU"] || !literalCPUSetV1(c.Env["SIMDUTF_CPU"]) {
			return errors.New("invalid quiet affinity environment")
		}
	} else if c.Env["SIMDUTF_BENCH_EXPECT_OPERATION"] != "" || c.Env["SIMDUTF_BENCH_EXPECT_TIER"] != "" || c.Env["SIMDUTF_CPU"] != "" || c.Env["SIMDUTF_AFFINITY"] != "" {
		return errors.New("unexpected benchmark environment")
	}
	return nil
}
func laneSettingsV1(lane string) (string, string, string) {
	switch lane {
	case "darwin-arm64-nosimd":
		return "darwin", "arm64", "nosimd,runtimesecret"
	case "darwin-arm64-simd-negative":
		return "darwin", "arm64", "simd,runtimesecret"
	case "linux-amd64-none":
		return "linux", "amd64", "none"
	case "linux-amd64-simd":
		return "linux", "amd64", "simd"
	case "linux-riscv64-compile":
		return "linux", "riscv64", "nosimd,runtimesecret"
	case "linux-s390x-compile":
		return "linux", "s390x", "nosimd,runtimesecret"
	}
	return "", "", ""
}
func validateCachesV1(c CampaignCommandV1, e EvidenceValidationContextV1) error {
	if c.Role != "old" && c.Role != "new" {
		if c.Env["GOCACHE"] == "" || c.Env["GOMODCACHE"] == "" || c.Env["GOCACHE"] == c.Env["GOMODCACHE"] || path.Clean(c.Env["GOCACHE"]) != c.Env["GOCACHE"] || path.Clean(c.Env["GOMODCACHE"]) != c.Env["GOMODCACHE"] {
			return errors.New("invalid non-source cache paths")
		}
		return nil
	}
	if len(e.AuthorityBytes) == 0 {
		buildCache, moduleCache := c.Env["GOCACHE"], c.Env["GOMODCACHE"]
		root := strings.TrimSuffix(buildCache, "go-build")
		if !path.IsAbs(root) || root == buildCache || moduleCache != root+"gomod" ||
			!strings.HasSuffix(root, "/cache/"+c.Role+"/") ||
			path.Clean(buildCache) != buildCache || path.Clean(moduleCache) != moduleCache {
			return errors.New("invalid cache path")
		}
		return nil
	}
	root, err := CampaignCacheRootV1(e, c.Role)
	if err != nil {
		return err
	}
	if c.Env["GOCACHE"] != root+"go-build" || c.Env["GOMODCACHE"] != root+"gomod" {
		return errors.New("invalid cache path")
	}
	return nil
}

func validateOutputsV1(c CampaignCommandV1) error {
	artifact := map[string]string{
		"source_commit": "identity", "source_tree": "identity", "source_parent": "identity", "source_status": "identity",
		"host_uname": "identity", "host_cpu": "identity", "go_version": "identity", "file_digest": "identity",
		"go_test_focused": "test", "go_test_full": "test", "go_test_race": "race",
		"go_fuzz_replay": "fuzz", "go_fuzz": "fuzz",
		"provider_guard": "provider-guard", "selector_test": "selector", "final_selector_test": "final-selector",
		"go_object_build": "object", "go_objdump": "disassembly", "quiet_affinity_recheck": "quiet-affinity",
		"state_transition": "state-transition", "not_applicable": "not-applicable", "return_index": "index",
	}[c.Action]
	if c.Action == "go_benchmark" {
		if c.Role == "old" {
			artifact = "incumbent-benchmark"
		} else {
			artifact = "candidate-benchmark"
		}
	}
	if c.Action == "benchstat" {
		artifact = "benchstat"
	}
	expected := map[string]struct{ kind, media string }{
		"stdout": {"stdout", "text/plain"}, "stderr": {"stderr", "text/plain"}, "exit": {"exit", "application/json"}, "argv-env": {"argv-env", "application/json"},
	}
	if artifact != "" {
		media := "text/plain"
		if c.Action == "state_transition" || c.Action == "not_applicable" || c.Action == "return_index" || c.Action == "quiet_affinity_recheck" || c.Action == "benchstat" {
			media = "application/json"
		}
		expected[artifact] = struct{ kind, media string }{artifact, media}
	}
	if len(c.Outputs) != len(expected) {
		return errors.New("incorrect command outputs")
	}
	for _, o := range c.Outputs {
		want, ok := expected[o.ID]
		if !ok || o.Kind != want.kind || o.MediaType != want.media || !o.Required || o.Path != "staging/"+c.ID+"/"+o.ID+outputExtensionV1(o.ID) {
			return errors.New("invalid command output")
		}
		delete(expected, o.ID)
	}
	if len(expected) != 0 {
		return errors.New("missing command output")
	}
	return nil
}

func outputExtensionV1(id string) string {
	if id == "exit" || id == "argv-env" || id == "state-transition" || id == "not-applicable" || id == "index" || id == "quiet-affinity" || id == "benchstat" {
		return ".json"
	}
	return ".txt"
}
func validateCommandRequirementsV1(commands []CampaignCommandV1, requirements []CommandRequirementV1) error {
	if len(requirements) == 0 {
		return errors.New("command requirements are empty")
	}
	required := make(map[string]int, len(requirements))
	for _, r := range requirements {
		if r.Count < 1 || !campaignActionsV1[r.Action] || !campaignRolesV1[r.Role] || !allowedActionRoleV1(r.Action, r.Role) || !validID(r.OperationID, "op-v1-") || !validID(r.BatchID, "batch-v1-") || !validID(r.RowID, "rk-v1-") || !validID(r.CellID, "cell-v1-") || !canonicalProviderV1(r.Provider) {
			return errors.New("invalid command requirement")
		}
		key := strings.Join([]string{r.Action, r.Role, r.OperationID, r.BatchID, r.RowID, r.CellID, r.Provider}, "\x00")
		if _, exists := required[key]; exists {
			return errors.New("duplicate command requirement")
		}
		required[key] = r.Count
	}
	actual := make(map[string]int, len(commands))
	for _, c := range commands {
		actual[CampaignCommandTopologyKeyV1(c)]++
	}
	if len(actual) != len(required) {
		return errors.New("command topology does not match requirements")
	}
	for key, count := range required {
		if actual[key] != count {
			return errors.New("command requirement does not match topology")
		}
	}
	return validateBenchmarkTopologyV1(commands)
}

func validateBenchmarkTopologyV1(commands []CampaignCommandV1) error {
	type benchmarkPair struct {
		oldOrdinal, quietOrdinal, newOrdinal, guardOrdinal, benchstatOrdinal, selectorOrdinal int
		oldPath, newPath                                                                      string
	}
	pairs := make(map[string]benchmarkPair)
	for _, c := range commands {
		key := strings.Join([]string{c.OperationID, c.BatchID, c.RowID, c.CellID, c.Provider}, "\x00")
		pair := pairs[key]
		switch c.Action {
		case "go_benchmark":
			if c.Role == "old" {
				if pair.oldOrdinal != 0 {
					return errors.New("duplicate old benchmark command")
				}
				pair.oldOrdinal, pair.oldPath = c.Ordinal, commandOutputPathV1(c, "incumbent-benchmark")
			} else {
				if pair.newOrdinal != 0 {
					return errors.New("duplicate new benchmark command")
				}
				pair.newOrdinal, pair.newPath = c.Ordinal, commandOutputPathV1(c, "candidate-benchmark")
			}
		case "quiet_affinity_recheck":
			pair.quietOrdinal = c.Ordinal
		case "provider_guard":
			pair.guardOrdinal = c.Ordinal
		case "benchstat":
			incumbent, candidate, ok := benchstatInputPathsV1(c.Argv)
			if !ok || incumbent != pair.oldPath || candidate != pair.newPath {
				return errors.New("invalid benchstat inputs")
			}
			pair.benchstatOrdinal = c.Ordinal
		case "selector_test", "final_selector_test":
			pair.selectorOrdinal = c.Ordinal
		default:
			continue
		}
		pairs[key] = pair
	}
	for _, pair := range pairs {
		if pair.oldOrdinal != 0 || pair.newOrdinal != 0 || pair.benchstatOrdinal != 0 {
			if pair.oldOrdinal == 0 || pair.quietOrdinal == 0 || pair.newOrdinal == 0 || pair.guardOrdinal == 0 || pair.benchstatOrdinal == 0 || pair.selectorOrdinal == 0 ||
				!(pair.oldOrdinal < pair.quietOrdinal && pair.quietOrdinal < pair.newOrdinal && pair.newOrdinal < pair.guardOrdinal && pair.guardOrdinal < pair.benchstatOrdinal && pair.benchstatOrdinal < pair.selectorOrdinal) {
				return errors.New("invalid benchmark topology")
			}
		}
	}
	return nil
}
