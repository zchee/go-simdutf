# Pinned upstream provenance and adaptation policy

## Immutable authority and validation evidence

The sole upstream semantic, algorithm, table, vector, test, fuzz, benchmark,
and ISA authority is the following **post-release development snapshot**. Its
headers report `9.0.0`, but it must never be called the released `v9.0.0`.

- repository: [`simdutf/simdutf`](https://github.com/simdutf/simdutf)
- commit: [`c7bef0ff14a13fd6ea52e3347da2c659383392de`](https://github.com/simdutf/simdutf/tree/c7bef0ff14a13fd6ea52e3347da2c659383392de)
- tree: `4cbac4c5d1ce0d7f98cc35360d53725433f12811`
- checkout: a task-owned, detached, clean temporary checkout

Before using an upstream source, run and record these exact commands:

```sh
UP=/absolute/path/to/task-owned/simdutf-checkout
git -C "$UP" rev-parse HEAD^{commit} HEAD^{tree}
git -C "$UP" diff --quiet --ignore-submodules --exit-code
git -C "$UP" diff --cached --quiet --ignore-submodules --exit-code
test -z "$(git -C "$UP" status --porcelain=v1 --untracked-files=all)"
git -C "$UP" remote get-url origin
```

G004-A evidence recorded on 2026-07-28 UTC: the identity commands printed
`c7bef0ff14a13fd6ea52e3347da2c659383392de` and
`4cbac4c5d1ce0d7f98cc35360d53725433f12811`; every cleanliness check exited
zero; the sole direct-parent delta was
`src/icelake/icelake_base64.inl.cpp`; and the approved source, test, fuzz, and
benchmark paths were byte-identical to the direct parent. Phase 0 archive and
smoke artifacts remain inherited historical evidence only. A failure is a hard
blocker: recreate or repair the pinned checkout and repeat the commands. No
moving branch, later tag, later commit, blog post, competing implementation, or
generated output can fill a semantic or algorithm gap.

The initial acceleration inventory is in
[`isa-eligibility.tsv`](isa-eligibility.tsv). It is an evidence
and work-planning matrix that may also record later audited implementations.
Only a row explicitly marked `implemented` makes that narrow implementation
claim; eligibility or a `required` phase value alone does not.

## Apache-2.0 selection and required source treatment

Select the Apache-2.0 side of upstream’s dual Apache/MIT license, matching this
repository. Evidence: pinned
[`LICENSE-APACHE`](https://github.com/simdutf/simdutf/blob/c7bef0ff14a13fd6ea52e3347da2c659383392de/LICENSE-APACHE),
[`LICENSE-MIT`](https://github.com/simdutf/simdutf/blob/c7bef0ff14a13fd6ea52e3347da2c659383392de/LICENSE-MIT),
and [README license statement](https://github.com/simdutf/simdutf/blob/c7bef0ff14a13fd6ea52e3347da2c659383392de/README.md#L3020-L3024).
This choice does not erase any embedded third-party notice or Apache-2.0
attribution obligation.

| Material | Required file-level action |
| --- | --- |
| New `.go` source | Begin with the Apache-2.0 header copied exactly from [`doc.go`](../../doc.go). For a substantive translation add `Portions Copyright 2021 The simdutf Authors.` and an immediate precise comment: `Translated and adapted from simdutf/simdutf@<full SHA>:<path>[, <path>...]`. For independently written Go glue, retain the header and cite the pinned declaration/contract without falsely calling it translated. |
| New hand-written `.s` source | Use `//` comments containing the equivalent Apache-2.0 header. State the Go ABI wrapper, exact upstream kernel/helper path(s), and full SHA. A source-family name such as “Haswell” is insufficient provenance. |
| Copied/adapted algorithm, table, declaration, vector, or corpus | Preserve source copyright/attribution; record all material input paths and the full SHA; state material Go changes (slice bounds, capacity/panic, raw-storage endian representation, ABI boundary, canaries). A copied table/corpus has a deterministic checksum or vector test. |
| Upstream test/fuzz/benchmark source | Comment with exact source path(s), full SHA, and Go-only scaffolding. Corpus metadata additionally includes origin/retrieval date, license/notice, byte length, and SHA-256. |
| Hand-authored Go-only test | Use the Apache header and identify the narrow Go-only concern (slice bounds, alias, dispatch, build tag, object safety, or scalar differential fuzzing); do not label it an upstream vector. |

Every material input is named: scalar translations normally cite both
`src/fallback/implementation.cpp` and the relevant
`include/simdutf/scalar/**` file; vector translations cite the target kernel,
any `src/generic/**` helper/table, contract, and test source used. Never cite a
moving branch or shortened SHA. A generator is allowed only if pinned upstream
already has one or manual transcription is not auditable; it then requires the
input paths, generator provenance, output checksum, and deterministic
regeneration test.

## Third-party and excluded-source boundaries

| Source/risk | Mandatory policy |
| --- | --- |
| Fuchsia scalar/reference | `include/simdutf/scalar/utf8.h`, `tests/reference/validate_utf8.cpp`, and `tests/reference/validate_utf8_to_latin1.cpp` expressly credit Google Fuchsia under Apache. If their control flow or vectors are adapted, retain that credit in the target file/test in addition to simdutf provenance. |
| PyTorch ISA detector | `include/simdutf/internal/isadetection.h` says it is highly modified from PyTorch and carries a separate BSD-style notice. **Do not copy, translate, or structurally reproduce it.** Independently write the minimum CPUID/XGETBV probe from the CPU contract. Any exception requires the complete PyTorch notice and a separate license review before merge. |
| Competition tree | **Never port `benchmarks/competition/**`** into product code, tests, fuzz seeds, tables, or algorithm sources. Pinned [`benchmarks/competition/README.md`](https://github.com/simdutf/simdutf/blob/c7bef0ff14a13fd6ea52e3347da2c659383392de/benchmarks/competition/README.md) says it is research/benchmark-only, separately licensed, and not part of simdutf. It includes mixed licensing (for example LLVM-with-exception and OSL-3.0). `competition/inoue2008/inoue_utf8_to_utf16.h`’s `pclmul` target is specifically not ISA evidence. |
| Other separately noticed material | Do not import `fuzz/helpers/nameof.hpp` or another third-party helper merely because the parent repository is dual licensed. Review and retain its own notice first. |
| External corpus / later source | Pinned tracking alone does not prove corpus redistribution rights. Only use tracked data or explicitly approved upstream-recommended data with complete metadata. Later simdutf, other bindings, production cgo, Go dependencies, compiler-generated assembly, and unreviewed snippets are forbidden semantic sources. |

## Per-symbol ISA and object-proof policy

The upstream family names and target compilation regions are input evidence,
not executable Go predicates. The selector for every Go symbol is a tested
superset of the non-baseline opcodes actually present in its source and final
object code.

1. Pinned [`src/simdutf/westmere.h`](https://github.com/simdutf/simdutf/blob/c7bef0ff14a13fd6ea52e3347da2c659383392de/src/simdutf/westmere.h) declares `sse4.2,popcnt`. A Westmere-derived Go symbol must separately prove SSE4.2 and POPCNT when emitted; it may omit either only after source plus object audit prove absence.
2. Pinned [`src/simdutf/haswell.h`](https://github.com/simdutf/simdutf/blob/c7bef0ff14a13fd6ea52e3347da2c659383392de/src/simdutf/haswell.h) declares `avx2,bmi,lzcnt,popcnt`. `bmi` is not one final predicate: audit BMI1 and BMI2 independently, as well as LZCNT and POPCNT. Do not dispatch merely because the source family is named Haswell.
3. Every AVX/YMM symbol requires CPUID AVX, OSXSAVE, `XCR0[1]` and `XCR0[2]` (XMM/YMM state), and CPUID AVX2 before execution. The direct upstream detector is semantic evidence only; it is not approved source for copy because of the PyTorch notice.
4. PCLMULQDQ is detected by upstream but does not occur in the pinned product `src/westmere/**`, `src/haswell/**`, or `src/arm64/**` source trees. Do not make it a blanket predicate. Probe PCLMUL only if a specific Go assembly object audit proves its instruction is emitted.
5. Audit POPCNT, LZCNT/TZCNT, BMI1, BMI2, PCLMULQDQ, and every other non-baseline instruction per symbol. `_popcnt64` occurs in pinned Westmere/Haswell bit-manipulation headers; that is a trigger to audit the affected symbol, not evidence that every family symbol needs POPCNT.
6. Compile amd64 audit artifacts with `GOAMD64=v1`. Save `go tool objdump` (or equivalent) per direct symbol. A selector must cover every emitted non-baseline opcode, and must not require a feature solely because the C++ family region permitted it.
7. Initial arm64 acceleration is NEON hand-written `.s` only. Audit emitted NEON instructions and Go ABI/bounds safety. No arm64 file may import `simd/archsimd`.
8. `archsimd_status` has exactly three values: `pending-audit` for eligible or partially implemented work that has not passed every gate; `implemented` only when the implementation is tagged `amd64 && goexperiment.simd`, runtime-gated by `archsimd.X86.AVX2()`, covered by direct scalar-differential fuzzing, and backed by per-symbol `GOAMD64=v1` object proof; and `not_applicable` when no direct archsimd implementation is appropriate. A successful tagged build is not eligibility proof, and ordinary Go loops are not an archsimd implementation.

## Per-change merge gate

- [ ] Pinned checkout revalidated to the exact commit/tree and cleanliness evidence recorded.
- [ ] New Go/assembly files have required Apache headers and exact source comments.
- [ ] Every source/table/vector/corpus material input has full SHA/path; Fuchsia credit is retained if used.
- [ ] No PyTorch-derived detector or `benchmarks/competition/**` material entered the diff.
- [ ] Tables/corpora have deterministic verification and metadata.
- [ ] Each accelerated symbol has direct scalar-differential fuzz coverage and its `GOAMD64=v1`/NEON object audit.
- [ ] Each selector accepts its full audited feature set and rejects every missing feature, including AVX2 without OSXSAVE/XCR0 vector state.
- [ ] Tagged source is amd64-only, real vector code, and cannot leak `simd/archsimd` into default/arm64 builds.
