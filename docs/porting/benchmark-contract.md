# Benchmark and corpus contract

This conflict-safe contract is pinned to
`simdutf/simdutf@611becc2a08c27a4edc77d9a45ff74c97130129b`. Phase B proved
that the direct-parent delta changes no benchmark file. The inherited skeleton
Go checks, independent one-iteration C++ smoke procedures captured against the
predecessor, and frozen method for later unseeded-input comparison remain
historical readiness evidence only; they are not relabelled as runs of the new
authority and produced no performance claim.

## Phase D public-dispatch qualification surface

`BenchmarkDispatchQualification` is the only qualification entry point. Its
1054 stable rows have the exact name grammar
`BenchmarkDispatchQualification/<Operation>/<Corpus>/<Class>/<size>`; `size`
is zero-padded to four decimal digits, is bytes for byte corpora, and is code
units for `Q-u16-zero` and `Q-u32-zero`. Provider names are never appended.

| Operations | Corpus and classes | Exact sizes | Rows |
| --- | --- | --- | ---: |
| `ValidateASCII`, `ValidateASCIIWithErrors` | `Q-byte-zero`: `short`, `boundary`, `bulk` | short `1,15,16,17,31,32,33`; boundary `63,64,65,127,128,129`; bulk `4096` bytes | 28 |
| `ValidateUTF16LEAsASCII`, `ValidateUTF16BEAsASCII`, `ValidateUTF16AsASCII` | `Q-u16-zero`: `short`, `boundary`, `bulk` | short `1,7,8,9,15,16,17`; boundary `31,32,33,63,64,65,127,128,129`; bulk `2048` code units | 51 |
| `ValidateUTF8`, `ValidateUTF8WithErrors`, `CountUTF8`, `Latin1LengthFromUTF8`, `UTF16LengthFromUTF8`, `UTF32LengthFromUTF8` | all 14 `Q-byte-zero` rows above, then `Q-emoji/bulk/3150` | zero sizes above; emoji `3150` bytes | 90 |
| `ValidateUTF16LE`, `ValidateUTF16BE`, `ValidateUTF16LEWithErrors`, `ValidateUTF16BEWithErrors`, `ToWellFormedUTF16LE`, `ToWellFormedUTF16BE` | `Q-u16-zero`: `short`, `boundary`, `bulk` | short `1,7,8,9,15,16,17`; boundary `31,32,33,63,64,65,127,128,129`; bulk `2048` code units | 102 |
| `ValidateUTF32`, `ValidateUTF32WithErrors` | `Q-u32-zero`: `short`, `boundary`, `bulk` | short `1,3,4,5,7,8,9`; boundary `15,16,17,31,32,33`; bulk `1024` code units | 28 |
| `UTF8LengthFromLatin1`, `ConvertLatin1ToUTF8`, `ConvertLatin1ToUTF16LE`, `ConvertLatin1ToUTF16BE`, `ConvertLatin1ToUTF32` | `Q-latin1-ramp`: `short`, `boundary`, `bulk` | short `1,15,16,17,31,32,33`; boundary `63,64,65,127,128,129`; bulk `4096` bytes | 70 |
| `ConvertUTF8ToLatin1`, `ConvertUTF8ToLatin1WithErrors`, `ConvertValidUTF8ToLatin1`, `ConvertUTF8ToUTF16LE`, `ConvertUTF8ToUTF16BE`, `ConvertUTF8ToUTF16LEWithErrors`, `ConvertUTF8ToUTF16BEWithErrors`, `ConvertValidUTF8ToUTF16LE`, `ConvertValidUTF8ToUTF16BE`, `ConvertUTF8ToUTF32`, `ConvertUTF8ToUTF32WithErrors`, `ConvertValidUTF8ToUTF32` | all 14 `Q-byte-zero` rows above, then `Q-emoji/bulk/3150`, then `Q-arabic-lipsum/bulk/81685` | zero sizes above; emoji `3150` bytes; arabic `81685` bytes | 192 |
| `ConvertUTF16LEToLatin1`, `ConvertUTF16BEToLatin1`, `ConvertUTF16LEToLatin1WithErrors`, `ConvertUTF16BEToLatin1WithErrors`, `ConvertValidUTF16LEToLatin1`, `ConvertValidUTF16BEToLatin1`, `ConvertUTF16LEToUTF32`, `ConvertUTF16BEToUTF32`, `ConvertUTF16LEToUTF32WithErrors`, `ConvertUTF16BEToUTF32WithErrors`, `ConvertValidUTF16LEToUTF32`, `ConvertValidUTF16BEToUTF32`, `UTF32LengthFromUTF16LE`, `UTF32LengthFromUTF16BE`, `ConvertUTF16LEToUTF8`, `ConvertUTF16BEToUTF8`, `ConvertUTF16LEToUTF8WithErrors`, `ConvertUTF16BEToUTF8WithErrors`, `ConvertUTF16LEToUTF8WithReplacement`, `ConvertUTF16BEToUTF8WithReplacement`, `ConvertValidUTF16LEToUTF8`, `ConvertValidUTF16BEToUTF8`, `UTF8LengthFromUTF16LE`, `UTF8LengthFromUTF16BE`, `ChangeEndiannessUTF16`, `CountUTF16LE`, `CountUTF16BE`, `UTF8LengthFromUTF16LEWithReplacement`, `UTF8LengthFromUTF16BEWithReplacement` | `Q-u16-zero`: `short`, `boundary`, `bulk` | short `1,7,8,9,15,16,17`; boundary `31,32,33,63,64,65,127,128,129`; bulk `2048` code units | 493 |

`Q-byte-zero` is exactly 4096 zero bytes and `Q-u16-zero`/`Q-u32-zero` are
derived outside the timed loop from an identical 4096-byte zero blob with
`encoding/binary.NativeEndian`, never `unsafe`. Both raw blobs have SHA-256
`ad7facb2586fc6e966c004d7d1d16b024f5805ff7cb47c7a85dabd8b48892ca7`.
`Q-emoji` is the existing byte-identical embedded 3150-byte upstream blob with
SHA-256
`d6484d359bff183e4d6a4d20b3cc7056c55f372011f28b21b06462ba4d643523`.
`Q-latin1-ramp` is the deterministic 4096-byte `byte(i % 256)` sequence with
SHA-256
`c8f5d0341d54d951a71b136e6e2afcb14d11ed8489a7ae126a8fee0df6ecf193`.
`Q-arabic-lipsum` loads the frozen external 81685-byte Arabic lipsum file from
`.omx/artifacts/phase0/benchmark-corpora/corpus/unicode_lipsum/lipsum/Arabic-Lipsum.utf8.txt`
and verifies SHA-256
`b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb`
before use; a missing or mismatched artifact fails closed.
The baseline and candidate must use an identical benchmark source blob, names,
corpus bytes, order, setup, sinks, timing boundary, and byte denominators.

Every timed iteration contains the literal exported Go wrapper call. Provider
selection is inspected only before `b.Loop()` with
`runtime.FuncForPC(reflect.ValueOf(fn).Pointer())`. A run fails closed unless
both `SIMDUTF_BENCH_EXPECT_OPERATION` and `SIMDUTF_BENCH_EXPECT_TIER` are
present and exactly match the row operation and the operation-specific final
identifier allowlist. `ValidateUTF16AsASCII` inspects the native LE or BE
dispatch field. The side-reference invariant is mandatory: correctness or
diagnostic direct-provider calls stay outside the timed public-dispatch row and
cannot replace it.

Qualification uses exactly 10 old samples followed by exactly 10 new samples
on one fixed host, toolchain, explicit experiment, affinity policy, and quiet
CPU. Each row must report `0 allocs/op`. An intended bulk win is at least 3%
with `p <= 0.05`; there must be no statistically significant slowdown, and no
protected short or boundary row may have a slowdown point estimate above 2%.
Any missing row, skipped row, inconclusive comparison, guard failure, provider
mismatch, allocation, or threshold failure leaves that provider direct-only.

## Required Go comparison method

1. Commit the final `b.Loop()` benchmark harness while public dispatch selects
   scalar. Current skeleton `main` is never the performance baseline.
2. Commit the accelerated candidate separately without changing benchmark or
   subbenchmark names, setup, corpus bytes, options, result sinks, timing
   boundary, or `b.SetBytes` denominator.
3. Run exactly 10 baseline samples, then exactly 10 candidate samples,
   sequentially on the same physical host. Never interleave revisions.
4. Stop Team and all other builds, tests, fuzzers, indexers, and benchmarks
   first. `b.RunParallel`, package-parallel comparisons, concurrent checkouts,
   and background benchmark work are prohibited.
5. Use one exact package and regexp, `GOMAXPROCS=1`, the same explicit
   `GOEXPERIMENT`, toolchain, host, corpus, and affinity policy. Remote old/new
   runs use the same recorded `taskset -c <cpu>`; local old/new runs record the
   same `affinity=none` policy.
6. Prepare corpora and destinations outside `b.Loop()`. Reset mutable state
   identically when required, retain a result sink, and report `-benchmem`.
7. Set throughput to upstream input bytes: `len(src)` for `[]byte`,
   `2*len(src)` for `[]uint16`, and `4*len(src)` for `[]uint32`, including a BOM
   only when the exact upstream procedure includes it. Never use output bytes or
   code-point counts.
8. Run `benchstat` only on Go baseline versus Go candidate. Report C++
   independently with a descriptive ratio/range; never feed C++ output to
   `benchstat` or use C++ speed as a correctness threshold.
9. Retain raw old/new output, verbatim benchstat, corpus manifest, host/load
   record, commands/exits, commits/trees, Go version/experiment, selected ISA,
   and affinity metadata under the phase artifact directory.
10. Same-tree direct scalar/assembly/archsimd subbenchmarks are diagnostic only
    and cannot replace the committed old/new comparison.

Command shapes use absolute Go paths and an exact package:

```sh
# local baseline, then candidate
env GOMAXPROCS=1 GOEXPERIMENT=nosimd,runtimesecret \
  /Users/zchee/sdk/go1.26.5/bin/go test -run='^$' \
  -bench='<exact-regexp>' -benchmem -count=10 . | tee old.txt
env GOMAXPROCS=1 GOEXPERIMENT=nosimd,runtimesecret \
  /Users/zchee/sdk/go1.26.5/bin/go test -run='^$' \
  -bench='<exact-regexp>' -benchmem -count=10 . | tee new.txt
/Users/zchee/go/bin/benchstat old.txt new.txt | tee benchstat.txt

# remote baseline, then candidate on the same recorded CPU
taskset -c '<recorded-cpu>' env GOMAXPROCS=1 GOEXPERIMENT=none GOAMD64=v1 \
  /home/zchee/sdk/go1.26.5/bin/go test -run='^$' \
  -bench='<exact-regexp>' -benchmem -count=10 . | tee old.txt
taskset -c '<recorded-cpu>' env GOMAXPROCS=1 GOEXPERIMENT=none GOAMD64=v1 \
  /home/zchee/sdk/go1.26.5/bin/go test -run='^$' \
  -bench='<exact-regexp>' -benchmem -count=10 . | tee new.txt
```

For archsimd comparisons, use `simd,runtimesecret` locally and `simd` remotely
on both revisions. For normal scalar/assembly comparisons, use
`nosimd,runtimesecret` locally and `none` remotely.

## Exact pinned upstream targets

The pinned CMake tree defines these five required targets, all now proven to
configure and build in the Phase 0 C++ smoke on both required physical hosts:

| Target | Exact procedure/options behavior | Input source |
| --- | --- | --- |
| `benchmark` | repeatable `-P` substring filters, repeatable `-F` files, repeatable `-I`; omitted iterations become 30000 in code although help says 3000; `--random-utf8 SIZE` uses pinned seed 1234 | files or pinned deterministic generator |
| `shortbench` | one `--function NAME` or `--all`; default max 128 and step 10; optional file; no file means zero-filled bytes | file prefixes or built-in zero buffer |
| `benchmark_base64` | `-d`, `-e`, `-r`, `--roundtrip-url`, `-L`, optional implementation filter; decode trims one final newline and requires single-line input | files; upstream recommends `lemire/base64data` |
| `findbenchmark` | four reports over 10,000 random alphanumeric characters with `=` appended | `std::random_device`; no seed/file option |
| `benchmark_to_well_formed_utf16` | `<length> <surrogate-pair-rate> <mismatched-rate> [--datapoints=N]`; internal default 100 timing iterations | `std::random_device`; no seed/file option |

The independent Phase 0 smoke used the exact upstream commit/tree and built all
five targets. Local Release `-O3 -DNDEBUG` used Apple Clang 21.0.0/CMake
4.4.0/Ninja 1.14.0.git; remote used the immutable linux/amd64 image
`docker.io/library/gcc@sha256:a8821110531e545aaa57a6bc27d9819db1fb09307c115513361e2633eaeb4961`
with GCC 15.1.0/CMake 3.25.1/Ninja 1.11.1. Both passed
`validate_utf8_basic_tests` and a one-iteration tracked-emoji procedure
(`validate_utf8+arm64` locally, `validate_utf8+icelake` remotely). Those are
smoke results only and cannot enter a throughput comparison.

The main target's exact simdutf base procedures are retained in
`.omx/artifacts/phase0/go-host-smoke/upstream-benchmark-inventory.log`; actual
simdutf procedure listings append `+<implementation>`. Optional ICU/iconv and
competition procedures are not Go-port targets. The same inventory retains all
33 exact `shortbench` names, including Base64 and `find_equal`.

## Frozen corpus manifest

### Ready at the pinned checkout

| ID | Source/options | Bytes and SHA-256 | Valid use/denominator |
| --- | --- | --- | --- |
| `upstream-emoji-utf8` | `benchmarks/dataset/emoji.txt`; main `-F <path> -P <exact> -I <fixed>`; Base64 encode/roundtrip may use identical bytes | 3150 bytes; `d6484d359bff183e4d6a4d20b3cc7056c55f372011f28b21b06462ba4d643523` | valid UTF-8 Unicode mix or arbitrary binary/Base64 encode; denominator 3150 input bytes |
| `shortbench-zero-128` | built-in `std::vector<char>(max_size, 0)`; exact options must record `--max-size 128`, chosen step, and exact function | logical SHA-256 `38723a2e5e8a17aa7950dc008209944e898f69a7bd10a23c839d341e935fd5ca` | each prefix row uses its own input length denominator |

The canonical corpus ID remains `shortbench-zero-128`. Its original pinned
`shortbench` behavior remains unchanged: `benchmarks/shortbench.cpp:419-422`
sets the 128-byte maximum and 10-byte step, while lines 493-526 construct the
zero-filled buffer and benchmark prefixes `1, 11, ..., 121`. Pinned
`shortbench` registers `validate_ascii`, not `validate_ascii_with_errors`; do
not issue or report a `shortbench` command for the latter.

Phase 2 may materialize those same 128 zero bytes as an ignored or temporary
binary file solely for the pinned main-target procedure
`validate_ascii_with_errors+<exact-implementation>`. `main-zero-128` is only a
benchmark procedure or hierarchy label, never a new corpus ID. The main
procedure consumes the full 128-byte file with denominator 128; it does not use
the `shortbench` prefix series. Before timing on each host, verify byte count
128 and exact SHA-256
`38723a2e5e8a17aa7950dc008209944e898f69a7bd10a23c839d341e935fd5ca`.
Treat the file only as ignored benchmark evidence/cache: do not commit it as
testdata, a shipped fixture, or a new upstream corpus. This authorization does
not extend to another procedure or to another encoding or validity class.

Use the exact main-target command shape locally on arm64 and remotely on amd64:

```sh
# local arm64
"$BUILD/benchmarks/benchmark" \
  -P 'validate_ascii_with_errors+arm64' \
  -F "$CORPUS/shortbench-zero-128.bin" -I 30000

# remote amd64 comparable indicator
"$BUILD/benchmarks/benchmark" \
  -P 'validate_ascii_with_errors+haswell' \
  -F "$CORPUS/shortbench-zero-128.bin" -I 30000
```

Retain the Ice Lake fastest row separately as non-comparable under the
same-host C++ indicator contract below.

`emoji.txt` is the only benchmark payload tracked at the pinned checkout. The
tracked `wikipedia_mars` content consists of recipes and ignores generated
`.txt`, `.html`, and `.utf16`; historical README sizes are not checksums.

Do not use emoji bytes for an encoding/validity class they do not satisfy. They
cannot silently replace UTF-16, UTF-32, Latin-1-valid transcoding, ASCII-valid
bulk data, Base64 decode, or a find-hit corpus.

### Frozen upstream-recommended external corpora

[`corpus-freeze.md`](corpus-freeze.md) freezes the exact Unicode Lipsum and
Base64 DNS commits, trees, paths, byte lengths, and SHA-256 values, with the
canonical ignored byte evidence under `.omx/artifacts/phase0/benchmark-corpora/`.
A later run may materialize only those exact bytes in a non-product cache and
must recheck length and digest on each host before timing. No baseline may use
`HEAD`, a moving branch, a tag resolved only at run time, an implicit download,
or a substituted custom corpus.

For the Unicode comparison, run the exact Haswell procedure with the frozen
81,685-byte Arabic file and the source-defined 30,000 iterations:

```sh
"$BUILD/benchmarks/benchmark" \
  -P 'convert_utf8_to_utf16le+haswell' \
  -F "$CORPUS/Arabic-Lipsum.utf8.txt" -I 30000
```

For Base64 decode, run
`benchmark_base64 -d -f haswell "$CORPUS/swedenzonebase.txt"`. The tool removes
one final LF and splits the frozen file into 100,000 space-containing records;
the timed input denominator is 35,000,000 bytes, not the 35,100,000-byte raw
file size. The Go benchmark must retain the same records, spaces, forgiving
tail behavior, and denominator. The checked decode-shape receipt is under
`.omx/artifacts/phase0/benchmark-corpus-resolution/external-api/`.

### Generator comparability boundary

- `benchmark --random-utf8 SIZE` uses seed 1234 but does not expose generated
  bytes. Use it comparatively only after pinned code materializes and hashes the
  exact buffer, or use a frozen file.
- `findbenchmark` and `benchmark_to_well_formed_utf16` use
  `std::random_device` and accept no seed/file. Phase 0 freezes the exact
  benchmark-only capture/replay method in [`corpus-freeze.md`](corpus-freeze.md).
  The Phase 13 companion must capture one upstream-generated input, freeze its
  bytes and metadata, and make C++ and Go verify and replay that artifact.
  Until then, stock runs are non-comparable diagnostics. Existing raw replay
  material under `.omx` is nonauthoritative mutable evidence only.

## Same-host C++ indicator contract

Build only the detached pinned upstream tree with tests and Release benchmarks
enabled. Record compiler, effective flags, selected implementation, warmup and
iteration policy, container digest where applicable, host/load, affinity, exact
procedure, exact corpus hash, and input-byte denominator.

For the main target, always spell exact `-P`, `-F`, and `-I`; the pinned source
default is 30000 even though help text says 3000. Match operation, validity/error
class, encoding/endian, safe versus ordinary semantics, Base64 options,
destination-allocation exclusion, bytes, and selected ISA. A different or
unprovable condition makes the pair non-comparable.

Remote Go and containerized C++ must use the same recorded host CPU affinity,
with `taskset`/`--cpuset-cpus`. Local macOS has no verified hard core binding;
use the same recorded no-affinity policy and quiet window for both languages.

On amd64, retain both the upstream-selected-fastest result and the direct
Haswell result, but use only the latter as the initial Go/C++ comparable
indicator. Label an Ice Lake/AVX-512 row `upstream fastest, not comparable to
initial Go`; never suppress it or compare it as if the initial Go scope
implemented AVX-512. Do not bypass upstream runtime support checks.

## Benchmark readiness gate

Performance activity is prohibited until all applicable items are true:

- the scalar `b.Loop()` harness and accelerated candidate are separate commits
  with an identical benchmark surface;
- the selected corpus row is ready and verified on the target host;
- Go/C++ toolchains, compiler flags, immutable container digest, affinity,
  selected ISA, and explicit experiment are recorded;
- correctness, fuzz, race, build-tag, selector, and disassembly gates are green;
- Team has `pending=0`, `in_progress=0`, and `failed=0`, then is shut down;
- no unrelated workload is active.

The Phase 0 C++ toolchain/build/smoke, external corpus identity, and exact
method for unseeded-generator comparability are resolved. Phase 13 still
requires the companion harness and fixtures, on-host materialization and digest
rechecks, committed Go baseline/candidate harness revisions, serialized 10+10
Go samples and `benchstat`, affinity/load capture, later correctness/ISA gates,
and final Go and C++ reports. A failed item prohibits a “quick” retained
benchmark and blocks the corresponding Phase 13 comparable indicator.
