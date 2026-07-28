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
	"strings"
	"testing"
)

func TestRenderCanonicalCampaignCommandsV1Profiles(t *testing.T) {
	goBin := "/home/zchee/sdk/go1.26.5/bin/go"
	profiles := []struct {
		action, role string
		argv         []string
	}{
		{"source_commit", "old", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "source-identity", "--role=old", "--receipt=staging/identity/old.json", "--archive=staging/source/old.tar"}},
		{"source_tree", "new", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "source-identity", "--role=new", "--receipt=staging/identity/new.json", "--archive=staging/source/new.tar"}},
		{"source_parent", "old", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "source-identity", "--role=old", "--receipt=staging/identity/old.json", "--archive=staging/source/old.tar"}},
		{"source_status", "new", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "source-identity", "--role=new", "--receipt=staging/identity/new.json", "--archive=staging/source/new.tar"}},
		{"host_uname", "host", []string{"/usr/bin/uname", "-srm"}},
		{"host_cpu", "host", []string{"/usr/bin/lscpu", "--json"}},
		{"go_version", "host", []string{goBin, "version"}},
		{"file_digest", "host", []string{"/usr/bin/sha256sum", "staging/identity/go-binary"}},
		{"go_test_focused", "old", []string{goBin, "test", "-run=^TestUTF8$", "-count=1", "."}},
		{"go_test_race", "direct", []string{goBin, "test", "-race", "."}},
		{"go_fuzz", "direct", []string{goBin, "test", "-fuzz=^FuzzUTF8$", "."}},
		{"provider_guard", "direct", []string{goBin, "test", "-run=^TestProviderGuard$", "."}},
		{"selector_test", "selector", []string{goBin, "test", "-run=^TestSelector$", "."}},
		{"final_selector_test", "selector", []string{goBin, "test", "-run=^TestFinalSelector$", "."}},
		{"go_object_build", "object", []string{goBin, "test", "-c", "."}},
		{"go_objdump", "object", []string{goBin, "tool", "objdump", "staging/object.test", "example.Symbol"}},
		{"state_transition", "direct", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "state-transition"}},
		{"not_applicable", "direct", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "not-applicable"}},
		{"return_index", "direct", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "return-index", "--descriptor-dir=staging/descriptors"}},
	}
	commands := make([]CampaignCommandV1, 0, len(profiles))
	for i, p := range profiles {
		commands = append(commands, campaignTestCommand(i+1, p.action, p.role, p.argv))
	}
	if _, err := RenderCanonicalCampaignCommandsV1(commands); err != nil {
		t.Fatalf("valid profiles rejected: %v", err)
	}
}

func TestCampaignCommandV1RejectsContractMutations(t *testing.T) {
	goBin := "/home/zchee/sdk/go1.26.5/bin/go"
	command := campaignTestCommand(1, "go_benchmark", "old", []string{"/usr/bin/taskset", "-c", "1", goBin, "test", "-run=^$", "-bench=^BenchmarkUTF8$", "-benchmem", "-count=10", "."})
	mutations := []struct {
		name string
		edit func(*CampaignCommandV1)
	}{
		{"wrong-tier", func(c *CampaignCommandV1) { c.Env["SIMDUTF_BENCH_EXPECT_TIER"] = "invalid" }},
		{"missing-typed-output", func(c *CampaignCommandV1) { c.Outputs = c.Outputs[:len(c.Outputs)-1] }},
		{"wrong-source-cwd", func(c *CampaignCommandV1) { c.CWD = "source/new" }},
		{"noncanonical-provider", func(c *CampaignCommandV1) { c.Provider = "provider" }},
		{"missing-linux-affinity", func(c *CampaignCommandV1) { delete(c.Env, "SIMDUTF_AFFINITY") }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			bad := command
			bad.Env = cloneEnv(command.Env)
			bad.Outputs = append([]CommandOutputV1(nil), command.Outputs...)
			mutation.edit(&bad)
			if _, err := RenderCanonicalCampaignCommandsV1([]CampaignCommandV1{bad}); err == nil {
				t.Fatal("mutation accepted")
			}
		})
	}
	darwin := campaignTestCommand(1, "host_cpu", "host", []string{"/usr/sbin/sysctl", "-n", "machdep.cpu.brand_string"})
	darwin.Env["GOOS"], darwin.Env["GOARCH"], darwin.Env["GOEXPERIMENT"] = "darwin", "arm64", "nosimd,runtimesecret"
	delete(darwin.Env, "GOAMD64")
	if _, err := RenderCanonicalCampaignCommandsV1([]CampaignCommandV1{darwin}); err == nil {
		t.Fatal("incomplete Darwin CPU argv accepted")
	}
}

func TestValidateBenchmarkTopologyV1(t *testing.T) {
	goBin := "/home/zchee/sdk/go1.26.5/bin/go"
	old := campaignTestCommand(1, "go_benchmark", "old", []string{"/usr/bin/taskset", "-c", "1", goBin, "test", "-run=^$", "-bench=^BenchmarkUTF8$", "-benchmem", "-count=10", "."})
	quiet := campaignTestCommand(2, "quiet_affinity_recheck", "direct", []string{goBin, "run", "./internal/portplan/cmd/simdutf-evidence", "quiet-affinity-recheck", "--cpu=1", "--policy=taskset:1"})
	quiet.Env["SIMDUTF_CPU"], quiet.Env["SIMDUTF_AFFINITY"] = "1", "taskset:1"
	new := campaignTestCommand(3, "go_benchmark", "new", []string{"/usr/bin/taskset", "-c", "1", goBin, "test", "-run=^$", "-bench=^BenchmarkUTF8$", "-benchmem", "-count=10", "."})
	guard := campaignTestCommand(4, "provider_guard", "direct", []string{goBin, "test", "-run=^TestProviderGuard$", "."})
	benchstat := campaignTestCommand(5, "benchstat", "new", []string{"/home/zchee/go/bin/benchstat", "-alpha=0.05", commandOutputPathV1(old, "incumbent-benchmark"), commandOutputPathV1(new, "candidate-benchmark")})
	selector := campaignTestCommand(6, "selector_test", "selector", []string{goBin, "test", "-run=^TestSelector$", "."})
	if err := validateBenchmarkTopologyV1([]CampaignCommandV1{old, quiet, new, guard, benchstat, selector}); err != nil {
		t.Fatalf("canonical topology rejected: %v", err)
	}
	old.Ordinal, new.Ordinal = 2, 1
	if err := validateBenchmarkTopologyV1([]CampaignCommandV1{old, quiet, new, guard, benchstat, selector}); err == nil {
		t.Fatal("new-before-old accepted")
	}
}
func TestValidateCachesV1RequiresFullAuthorityNamespace(t *testing.T) {
	command := campaignTestCommand(1, "go_test_full", "old", []string{"/home/zchee/sdk/go1.26.5/bin/go", "test", "."})
	authority := strings.Repeat("a", 64)
	context := EvidenceValidationContextV1{AuthorityBytes: []byte{1}, AuthoritySHA256: authority, CampaignID: campaignTestID("campaign-v1-"), LaneID: "linux-amd64-none"}
	command.Env["GOCACHE"] = "/home/zchee/.cache/gjc/simdutf-port/v1/" + context.CampaignID + "/cache/old/go-build"
	command.Env["GOMODCACHE"] = "/home/zchee/.cache/gjc/simdutf-port/v1/" + context.CampaignID + "/cache/old/gomod"
	if err := validateCachesV1(command, context); err != nil {
		t.Fatalf("full authority namespace rejected: %v", err)
	}
	command.Env["GOCACHE"] = strings.Replace(command.Env["GOCACHE"], context.CampaignID, "campaign-v1-"+strings.Repeat("1", 64), 1)
	if err := validateCachesV1(command, context); err == nil {
		t.Fatal("truncated authority namespace accepted")
	}
}

func campaignTestCommand(ordinal int, action, role string, argv []string) CampaignCommandV1 {
	cwd := commandCWDV1(action, role)
	id := "command-test-" + string(rune('a'+ordinal))
	env := map[string]string{"LC_ALL": "C", "GOMAXPROCS": "1", "GOEXPERIMENT": "nosimd,runtimesecret", "GOAMD64": "v1", "GOCACHE": "/var/tmp/simdutf-port-v1/cache/" + strings.TrimPrefix(cwd, "source/") + "/go-build", "GOMODCACHE": "/var/tmp/simdutf-port-v1/cache/" + strings.TrimPrefix(cwd, "source/") + "/gomod", "CGO_ENABLED": "0", "GOOS": "linux", "GOARCH": "amd64"}
	if action == "provider_guard" {
		env["SIMDUTF_PROVIDER"], env["SIMDUTF_OPERATION"] = "westmere", campaignTestID("op-v1-")
	}
	if action == "go_benchmark" {
		env["SIMDUTF_BENCH_EXPECT_OPERATION"] = campaignTestID("op-v1-")
		env["SIMDUTF_BENCH_EXPECT_TIER"] = "scalar"
		env["SIMDUTF_CPU"] = "1"
		env["SIMDUTF_AFFINITY"] = "taskset:1"
		if role == "new" {
			env["SIMDUTF_BENCH_EXPECT_TIER"] = "westmere"
		}
	}
	if action == "go_test_race" {
		env["CGO_ENABLED"] = "1"
	}
	kind := campaignArtifactKindV1(action, role)
	outputs := []CommandOutputV1{{"stdout", "stdout", "staging/" + id + "/stdout.txt", "text/plain", true}, {"stderr", "stderr", "staging/" + id + "/stderr.txt", "text/plain", true}, {"exit", "exit", "staging/" + id + "/exit.json", "application/json", true}, {"argv-env", "argv-env", "staging/" + id + "/argv-env.json", "application/json", true}}
	outputs = append(outputs, CommandOutputV1{kind, kind, "staging/" + id + "/" + kind + outputExtensionV1(kind), map[bool]string{true: "application/json", false: "text/plain"}[action == "state_transition" || action == "not_applicable" || action == "return_index"], true})
	return CampaignCommandV1{ordinal, id, action, role, argv, cwd, env, 60, 0, outputs, campaignTestID("op-v1-"), campaignTestID("batch-v1-"), campaignTestID("rk-v1-"), campaignTestID("cell-v1-"), "westmere"}
}

func campaignArtifactKindV1(action, role string) string {
	if action == "go_benchmark" {
		if role == "old" {
			return "incumbent-benchmark"
		}
		return "candidate-benchmark"
	}
	return map[string]string{"source_commit": "identity", "source_tree": "identity", "source_parent": "identity", "source_status": "identity", "host_uname": "identity", "host_cpu": "identity", "go_version": "identity", "file_digest": "identity", "go_test_focused": "test", "go_test_full": "test", "go_test_race": "race", "go_fuzz_replay": "fuzz", "go_fuzz": "fuzz", "benchstat": "benchstat", "provider_guard": "provider-guard", "selector_test": "selector", "final_selector_test": "final-selector", "go_object_build": "object", "go_objdump": "disassembly", "quiet_affinity_recheck": "quiet-affinity", "state_transition": "state-transition", "not_applicable": "not-applicable", "return_index": "index"}[action]
}
func cloneEnv(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
func campaignTestID(prefix string) string { return prefix + strings.Repeat("0", 64) }
