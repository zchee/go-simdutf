# Phase 0 integration gate

**Decision: PASS as inherited Phase 0 readiness evidence.** The exact Phase 0
work and exit gate in `.omx/plans/port-simdutf-dec3aad192f4-go.md` was
satisfied. G004-A remains historical source-control refresh evidence. Phase A
froze the live 30-row API, and Phase B repinned the live upstream authority and
checksum record. Together these records prove pinning, manifest, licensing,
environment, corpus-method, preservation, and smoke readiness only; they are
not Phase 13 performance acceptance.

## Canonical source-controlled record

The canonical `docs/porting/` inventory is exactly:

- `api-manifest.md` and `api-manifest.tsv`
- `benchmark-contract.md` and `corpus-freeze.md`
- `environment.md`
- `isa-eligibility.tsv`
- `preservation-manifest.tsv` and `preservation.md`
- `provenance.md`
- `phase0-gate.md`
- `phase0-authoritative.sha256`

`phase0-authoritative.sha256` lists exactly the other ten files. No replay
document, root validator, script, or ignored artifact is source-controlled
authority.

## Work and exit-gate map

| Plan requirement | Evidence and outcome | Status |
| --- | --- | --- |
| Exact upstream identity and clean detached archive | `provenance.md`; current commit `611becc2a08c27a4edc77d9a45ff74c97130129b`, tree `c8292790d793212ca0a1faf6ae42e7f8e7b70d4f`, detached clean checkout, and direct parent `c7bef0ff14a13fd6ea52e3347da2c659383392de` (tree `4cbac4c5d1ce0d7f98cc35360d53725433f12811`). The exact delta is eight ARM64 files, +59/-37, with no public-header, fallback, generic, Westmere, Haswell, test, fuzz, or benchmark change. Archive SHA-256 `041f66dc08c1cbec7207aa01e2b70f3581d9cd991993c9863c2126b7ad032dbd` remains inherited historical Phase 0 evidence. | **PASS** |
| Complete API manifest | `api-manifest.tsv`: 164 rows × 23 columns, `planned=125`, `implemented=30`, `excluded=9`; milestones are 30 `611becc-current-api`, 125 `future-upstream-api`, and 9 `upstream-excluded`; 133/133 candidate coverage, exact Find index/absent semantics, UTF-16 Find/Base64 benchmark non-applicability, 126 reproducible `SIMDUTF_SPAN` ranges, zero competition mappings | **PASS** |
| Per-operation semantics and sources | All rows classify result/count units, destination and alias contracts, feature gates, source/test/fuzz/benchmark correspondence or exact non-applicability | **PASS** |
| Frozen API rules | Raw-storage UTF-16, internal native-endian delegation, ordinary bounds panic, bounded written-count-only behavior, and Base64-safe `(Result, written int)` are explicit | **PASS** |
| Apache-2.0 file policy | `provenance.md` contains file-level header, full-SHA/path, notice, Fuchsia, PyTorch-detector, and competition-tree rules | **PASS** |
| Both Go matrices and remote environment | Fresh local explicit `nosimd,runtimesecret` and `simd,runtimesecret` tests/builds pass; checked remote `none`/`simd` evidence passes; fresh read-only SSH verifies remote identity, Go 1.26.5, Podman, and Git | **PASS** |
| Exact both-host C++ smoke | All five required targets and all 22 exact-zero exit files pass on both hosts; the regenerated 337-entry artifact manifest verifies 337/337 after documenting the late CTest-log mutation | **PASS** |
| Frozen corpora and random-input method | External corpus commits/trees/paths/blobs/sizes/SHA-256 values pass. Phase 0 freezes only the exact capture/replay method for unseeded inputs; the companion harness, fixtures, materialization, and timing remain Phase 13 | **PASS** |
| Preservation | The historical Phase 0 fenced checker passed its captured 133-row baseline. G004-A recorded 134 rows, 13 immutable and 121 mutable: `.claude/scheduled_tasks.lock` is the sole immutable-absent exception and `.codex/config.toml` is the sole exact `.codex` asset. Its targeted checks passed and rejected lock creation or config drift; unrelated later baseline drift is not silently rebaselined or promoted to current full-gate PASS. | **PASS (historical); targeted refresh PASS** |
| ISA/opcode policy | `isa-eligibility.tsv` is 23 rows × 17 columns with valid assembly/archsimd states and per-symbol opcode policy | **PASS** |

All five explicit exit bullets pass: zero unclassified or ambiguous API
declarations; actionable license policy; both explicit Go matrices build;
exact C++ test/benchmark targets smoke on both hosts; and preservation has no
unclassified entry or failed invariant.

## Replay scope boundary

No ignored benchmark-replay artifact is authoritative or a completed Phase 0
requirement. Any existing replay material under `.omx` is nonauthoritative
mutable evidence only. Phase 13 owns the companion harness and captured
fixtures, on-host hash rechecks, committed identical Go benchmark surfaces,
serialized 10+10 sampling, `benchstat`, affinity/load capture, and final
Go/C++ comparative reports.

## Inherited Phase 0 final verification

The standard-library-only validator and its captured execution record are under
`.omx/artifacts/phase0/final-verification/`:

- `verify_phase0.py`
- `command.txt`
- `stdout.log`
- `stderr.log`
- `exit.txt`
- `SHA256SUMS`

The recorded command was `python3
.omx/artifacts/phase0/final-verification/verify_phase0.py`; `exit.txt` is
exactly `0` for the historical Phase 0 snapshot. That immutable artifact is
retained as inherited evidence; it is not relabelled as current `611becc`
acceptance. G004-A remains predecessor-refresh evidence. Phase A validates the
current API and milestone totals; Phase B validates the new commit/tree,
direct-parent delta, and live `phase0-authoritative.sha256` only.
