# Phase 0 integration gate

**Decision: PASS.** The exact Phase 0 work and exit gate in
`.omx/plans/port-simdutf-dec3aad192f4-go.md` is satisfied and fresh final
verification exits zero. This proves pinning, manifest, licensing,
environment, corpus-method, preservation, and smoke readiness only; it is not
product implementation or Phase 13 performance acceptance.

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
| Exact upstream identity and clean detached archive | `provenance.md`; live commit `dec3aad192f47081110d9c766d4917bad243906f`, tree `eb5429bb160dfdf1a7d208f0184d3379940e69ee`, detached clean checkout, archive SHA-256 `041f66dc08c1cbec7207aa01e2b70f3581d9cd991993c9863c2126b7ad032dbd` | **PASS** |
| Complete API manifest | `api-manifest.tsv`: 164 rows × 22 columns, `planned=155`, `excluded=9`, 133/133 candidate coverage, exact Find index/absent semantics, UTF-16 Find/Base64 benchmark non-applicability, 126 reproducible `SIMDUTF_SPAN` ranges, zero competition mappings | **PASS** |
| Per-operation semantics and sources | All rows classify result/count units, destination and alias contracts, feature gates, source/test/fuzz/benchmark correspondence or exact non-applicability | **PASS** |
| Frozen API rules | Raw-storage UTF-16, internal native-endian delegation, ordinary bounds panic, bounded written-count-only behavior, and Base64-safe `(Result, written int)` are explicit | **PASS** |
| Apache-2.0 file policy | `provenance.md` contains file-level header, full-SHA/path, notice, Fuchsia, PyTorch-detector, and competition-tree rules | **PASS** |
| Both Go matrices and remote environment | Fresh local explicit `nosimd,runtimesecret` and `simd,runtimesecret` tests/builds pass; checked remote `none`/`simd` evidence passes; fresh read-only SSH verifies remote identity, Go 1.26.5, Podman, and Git | **PASS** |
| Exact both-host C++ smoke | All five required targets and all 22 exact-zero exit files pass on both hosts; the regenerated 337-entry artifact manifest verifies 337/337 after documenting the late CTest-log mutation | **PASS** |
| Frozen corpora and random-input method | External corpus commits/trees/paths/blobs/sizes/SHA-256 values pass. Phase 0 freezes only the exact capture/replay method for unseeded inputs; the companion harness, fixtures, materialization, and timing remain Phase 13 | **PASS** |
| Preservation | The manifest remains outside mutable paths; the sole missing mutable directory was restored at its recorded `0755`. The exact fenced block is fail-closed: its live positive reports 133 rows, 12 immutable, 121 mutable, `failures=0`, and exit 0; isolated `AGENTS.md` missing, regular-to-directory type, and mode-change negatives each emit the precise failure and exit nonzero | **PASS** |
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

## Fresh final verification

The standard-library-only validator and its captured execution record are under
`.omx/artifacts/phase0/final-verification/`:

- `verify_phase0.py`
- `command.txt`
- `stdout.log`
- `stderr.log`
- `exit.txt`
- `SHA256SUMS`

The recorded command is `python3
.omx/artifacts/phase0/final-verification/verify_phase0.py`; `exit.txt` is
exactly `0`. The validator checks the canonical inventory and hashes, upstream
identity/archive, API and 126-range evidence, ISA/provenance/preservation,
fresh local Go matrices, checked plus live remote evidence, all C++ targets and
exits, artifact checksums, corpora, links/whitespace, and the Phase 13-only
replay boundary. It executes the exact preservation fenced block live and in
isolated missing/type/mode negative fixtures, requiring the live run to exit 0
with `failures=0` and every negative to emit its precise failure and exit
nonzero.
