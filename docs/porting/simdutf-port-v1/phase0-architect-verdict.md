VERDICT: CLEAR

Architecture audit of D0/Phase 0 evidence-producer freeze at HEAD 6d611b17c01fbfa2e48d86fa9e939f8ea3853142 (gajae/comprehensive-simdutf-port). Scope limited to the listed producer/campaign/schema/dispatch paths. W01 hard gates, Latin-1 host qualification dispositions, remote artifact 876, and D3 are out of scope.

## 1. source-identity --action= + receipt/archive argv — CLEAR
- internal/portplan/cmd/simdutf-evidence/main.go runSourceIdentity requires exact ordered flags --action=, --role=, --receipt=, --archive=; actions are source_commit|source_tree|source_parent|source_status; roles old|new; writes simdutf-source-identity-receipt-v1 to --receipt= after reading archive + .seed.json.
- internal/portplan/campaign.go validateArgvV1 fail-closes those actions via exactArgsV1 to: go run ./internal/portplan/cmd/simdutf-evidence source-identity --action=<action> --role=<role> --receipt=staging/identity/<role>.json --archive=staging/source/<role>.tar.
- docs/porting/simdutf-port-v1/inputs/evidence-contract-schema-v1.json $defs/sourceIdentityReceipt + $defs/sourceIdentitySeed match producer schemas.

## 2. state_transition / not_applicable fixed-order structural flags — CLEAR
- internal/portplan/cmd/simdutf-evidence/main.go runStateTransition + parseExactFlagsWithProofs require fixed order --state-subject=, --prerequisite-state=, --current-state=, --disposition=, --go-qualification= (plus --na-reason=, --na-source= for not-applicable); trailing args must be --proof-receipt-id= only.
- internal/portplan/campaign.go validateEvidenceProducerStateArgsV1 mirrors that prefix/flag order against go run … simdutf-evidence state-transition|not-applicable, with NA structural constraints (backend_cell, not_applicable disposition/state) and fail-closed proof-receipt tails.
- Producer emits simdutf-state-transition-v1 JSON on stdout; contract $defs/stateTransition covers the shape.

## 3. quiet-affinity kind/media JSON + Linux affinity semantics — CLEAR
- Kind/media: internal/portplan/evidence_core.go evidenceKindsV1 includes quiet-affinity; internal/portplan/campaign.go validateOutputsV1/outputExtensionV1 bind action quiet_affinity_recheck → kind quiet-affinity, media_type=application/json, .json path; docs/porting/simdutf-port-v1/inputs/evidence-schema-v1.json and campaign-command-schema-v1.json enumerate the same kind.
- Linux semantics: producer runQuietAffinity is Linux-only, requires policy == taskset:+cpu, and checks /proc/self/status Cpus_allowed_list via readLinuxCpusAllowedList; campaign argv/SIMDUTF_* env enforce the same Linux-only + taskset: policy before exactArgsV1.
- Artifact schema simdutf-quiet-affinity-v1 is validated by canonicalQuietAffinityArtifactV1 and $defs/quietAffinity.

## 4. benchstat via simdutf-evidence → simdutf-benchstat-v1 — CLEAR
- Campaign validateEvidenceProducerBenchstatArgsV1 requires go run ./internal/portplan/cmd/simdutf-evidence benchstat with fixed --incumbent=, --candidate=, receipt IDs, --qualification-contract=, --operation-id= (no external benchstat binary).
- Producer runBenchstat calls portplan.RenderCanonicalBenchstatArtifactV1, which emits schema simdutf-benchstat-v1 (newline-terminated canonical JSON) to stdout.
- validateBenchstatArtifactV1 / $defs/benchstatArtifact require simdutf-benchstat-v1.

## 5. return-index --descriptor-dir= + proof JSON stdout — CLEAR
- Producer runReturnIndex accepts only --descriptor-dir=, loads context.json + registry under that dir, gates via ValidateEvidenceRecordV1 + RenderReturnIndexV1, then writes simdutf-return-index-proof-v1 JSON to stdout.
- Campaign exactArgsV1 freezes argv to … return-index --descriptor-dir=staging/descriptors.
- Contract $defs/returnIndexProof + $defs/returnIndexContext match producer proof/context shapes; campaign output kind for return_index is index with application/json.

## 6. evidenceKindsV1 + evidence-contract-schema oneOf — CLEAR
- internal/portplan/evidence_core.go evidenceKindsV1 includes producer-facing kinds benchstat, quiet-affinity, state-transition, not-applicable, index (plus identity/benchmark pair kinds).
- docs/porting/simdutf-port-v1/inputs/evidence-contract-schema-v1.json oneOf refs #/$defs/quietAffinity, stateTransition, returnIndexProof, sourceIdentityReceipt, sourceIdentitySeed, returnIndexContext, benchstatArtifact.
- Runtime receipt kinds remain governed by docs/porting/simdutf-port-v1/inputs/evidence-schema-v1.json (kind enum includes the same producer kinds).

## 7. campaign exactArgsV1 fail-closed to producer contract — CLEAR
- internal/portplan/campaign.go validateArgvV1 routes every producer action through exactArgsV1 or a dedicated exact-prefix validator (validateEvidenceProducerStateArgsV1, validateEvidenceProducerBenchstatArgsV1); default returns unsupported command executable profile.
- Producer CLI surface is only source-identity, quiet-affinity-recheck, state-transition, not-applicable, benchstat, return-index (main.go run); campaign action enum in campaign-command-schema-v1.json / campaignActionsV1 aligns, with role/cwd allOf fail-closed.
- Output topology (validateOutputsV1) requires typed proof artifact + stdout/stderr/exit/argv-env with exact staging paths and media types for those actions.

## 8. Latin-1 public dispatch scalar-first by default — CLEAR
- In-scope Latin-1-source public ops (UTF8LengthFromLatin1, ConvertLatin1ToUTF8, ConvertLatin1ToUTF16LE, ConvertLatin1ToUTF16BE, ConvertLatin1ToUTF32) select via dispatch.go selectVariant first-supported order.
- dispatch_amd64.go and dispatch_arm64.go list implementationScalar first for each of those fields; without SIMDUTF_FORCE_PROVIDER / SIMDUTF_BENCH_EXPECT_TIER, default selection is scalar.
- Latin1LengthFromUTF8 (SIMD-first countUTF8 alias) is out of this theme Latin-1-source freeze scope.
