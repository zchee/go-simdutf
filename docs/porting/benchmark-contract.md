# Benchmark and corpus contract

This conflict-safe contract is pinned to
`simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f`. Phase 0 executed
skeleton Go build/test checks, independent one-iteration exact-upstream C++
smoke procedures, and froze the exact method needed to compare unseeded
random-input targets later. It produced no performance claim.

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
