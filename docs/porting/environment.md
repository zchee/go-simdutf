# Phase 0 porting environment evidence

This conflict-safe evidence copy was captured for the Go repository skeleton at
`4f0094b43eca35e24bedc85a7a19702cbd5cbafe` (tree
`45b57f78581cb85859aa24aef76984fec0ba92e5`). Its current upstream semantic
authority is simdutf commit
`611becc2a08c27a4edc77d9a45ff74c97130129b` (tree
`c8292790d793212ca0a1faf6ae42e7f8e7b70d4f`). Phase B proved that the authority
change from direct parent `c7bef0ff14a13fd6ea52e3347da2c659383392de`
touches only eight ARM64 product files and changes no tests, fuzz targets, or
benchmarks. It did not rerun the original Phase 0 capture, so the host and
toolchain facts remain inherited environment evidence rather than current
product acceptance. Sensitive configuration values are not recorded.

## Evidence sources

- Go host and skeleton smoke transcripts:
  `.omx/artifacts/phase0/go-host-smoke/`.
- Their checked manifest:
  `.omx/artifacts/phase0/go-host-smoke/SHA256SUMS`.
- Independent pinned upstream C++ smoke summary:
  `.omx/artifacts/phase0/cpp-smoke/summary.json`.

The successful remote Go transcript is
`go-host-smoke/remote/host-and-smoke-retry.log`. The earlier
`host-and-smoke.log` is retained as a failed transport attempt: multi-source
`scp` created `skeleton.tar` as a directory. Its Go failures are not toolchain
evidence.

## Verified local physical host: darwin/arm64

Go evidence was captured at `2026-07-26T00:20:24Z`.

| Property | Verified value |
| --- | --- |
| OS/kernel | macOS 27.0 build 26A5388g; Darwin 27.0.0 |
| Hardware | Mac15,9; Apple M3 Max; 16 physical/logical CPUs |
| Go executable | `/Users/zchee/sdk/go1.26.5/bin/go` |
| Go/GOROOT | `go1.26.5 darwin/arm64`; `/Users/zchee/sdk/go1.26.5` |
| Configured experiment | `simd,runtimesecret` from Go configuration; no process-level exported `GOEXPERIMENT` |
| C++ toolchain | Apple Clang 21.0.0, CMake 4.4.0, Ninja 1.14.0.git |
| Benchmark support | `benchstat` from `golang.org/x/perf@v0.0.0-20260709024250-82a0b07e230d`; no `taskset`, `numactl`, or `perf` |

Because the Go configuration has implicit experiments, bare or implicit Go
commands are not acceptance evidence. Both explicit skeleton modes passed:

```text
GOEXPERIMENT=nosimd,runtimesecret /Users/zchee/sdk/go1.26.5/bin/go build ./...  -> exit 0
GOEXPERIMENT=nosimd,runtimesecret /Users/zchee/sdk/go1.26.5/bin/go test -count=1 ./... -> exit 0
GOEXPERIMENT=simd,runtimesecret /Users/zchee/sdk/go1.26.5/bin/go list simd/archsimd -> exit 0
GOEXPERIMENT=simd,runtimesecret /Users/zchee/sdk/go1.26.5/bin/go build ./... -> exit 0
GOEXPERIMENT=simd,runtimesecret /Users/zchee/sdk/go1.26.5/bin/go test -count=1 ./... -> exit 0
```

The independent C++ lane is now resolved for Phase 0 smoke. It configured the
exact pinned upstream tree in Release mode (`-O3 -DNDEBUG`), built the five
required benchmark targets, passed `validate_utf8_basic_tests`, and passed a
one-iteration `validate_utf8+arm64` smoke using the tracked emoji corpus. This
is build/run readiness only, not a performance result.

## Verified remote physical host: linux/amd64

Go evidence was captured at `2026-07-26T00:21:17Z` over
`ssh debian-13-trixie.gaudiy-platform`.

| Property | Verified value |
| --- | --- |
| OS/kernel | Debian GNU/Linux 13.6; Linux 6.12.90+deb13.1-cloud-amd64 |
| Hardware | Intel Xeon Platinum 8481C; 22 cores/44 threads; one NUMA node; KVM |
| Relevant features | SSE4.1/4.2, POPCNT, PCLMULQDQ, AVX/AVX2, BMI1/2, LZCNT/ABM, XSAVE/XGETBV; AVX-512 advertised but out of initial scope |
| Go executable | `/home/zchee/sdk/go1.26.5/bin/go` |
| Go/GOROOT | `go1.26.5 linux/amd64`; `/home/zchee/sdk/go1.26.5` |
| Default experiment | empty; no process-level exported `GOEXPERIMENT` |
| Host tools | rootless Podman 5.4.2, `taskset`, Git 2.47.3, `sha256sum`, `tar`, noninteractive `sudo` |
| Host-native C++ tools | compiler, CMake, and Ninja absent from login `PATH` |

Rootless Podman reported no systemd user session and fell back to `cgroupfs`.
That did not block the passing containerized C++ smoke.

The corrected Go retry copied the exact `git archive HEAD` skeleton. SHA-256
`b75d0fcf8c881206c807f836d299fb48bd5a83343521bafbb62b05cec12d21cf`
matched locally and remotely. These passed via the absolute Go executable:

```text
GOEXPERIMENT=none /home/zchee/sdk/go1.26.5/bin/go build ./... -> exit 0
GOEXPERIMENT=none /home/zchee/sdk/go1.26.5/bin/go test -count=1 ./... -> exit 0
GOEXPERIMENT=simd /home/zchee/sdk/go1.26.5/bin/go list simd/archsimd -> exit 0
GOEXPERIMENT=simd /home/zchee/sdk/go1.26.5/bin/go build ./... -> exit 0
GOEXPERIMENT=simd /home/zchee/sdk/go1.26.5/bin/go test -count=1 ./... -> exit 0
```

The independent C++ lane resolved the remote toolchain blocker with immutable
linux/amd64 image digest
`docker.io/library/gcc@sha256:a8821110531e545aaa57a6bc27d9819db1fb09307c115513361e2633eaeb4961`
(index digest
`sha256:476d602721ab76a00efbc132c944a353ceb404bdb012cc402f749f1e04d57ae9`).
Inside it, GCC/G++ 15.1.0, CMake 3.25.1, and Ninja 1.11.1 configured Release
`-O3 -DNDEBUG`, built all five required targets, passed
`validate_utf8_basic_tests`, and passed a one-iteration
`validate_utf8+icelake` emoji smoke. The upstream archive commit was verified;
the evidence archive SHA-256 is
`5a087d0923265716219f8a7dd892521240ded108dd054191213c7d1f9cee4bc7`
and its safe local validation passed with 43 members and zero mismatches.

## Compile-only scalar portability

Using `CGO_ENABLED=0` and `GOEXPERIMENT=nosimd,runtimesecret`, the local Go
toolchain successfully cross-built:

- `linux/riscv64`, a non-amd64/non-arm64 little-endian target; and
- `linux/s390x`, a big-endian target.

`go test -c` returned exit 0 for both but emitted no test binaries because the
skeleton has no test files; absence was checked explicitly. Repeat after scalar
native-endian source and tests exist, then require emitted binary hashes.

## Phase 13 matrix still required

Phase 0 smoke does not satisfy the final matrix. Exact committed candidate
checkouts must later produce:

| Host | Required lanes | Additional proof |
| --- | --- | --- |
| darwin/arm64 | explicit `nosimd,runtimesecret` and `simd,runtimesecret`: vet, targeted/full/race tests, required fuzz targets | selected NEON/scalar tier, arm64 disassembly, negative archsimd import/symbol proof, serialized comparable Go and C++ reports |
| linux/amd64 | explicit `none` and `simd` through `/home/zchee/sdk/go1.26.5/bin/go`: vet, targeted/full/race tests, required fuzz targets | `GOAMD64=v1` objects, CPUID/XGETBV negative matrix, selected tier, per-symbol disassembly, serialized comparable Go and C++ reports |
| portability | one non-initial little-endian and one big-endian target with acceleration excluded | emitted `go test -c` artifacts, SHA-256, scalar/native-endian compilation |

## Remaining blockers and no-go conditions

1. No product implementation, tests, or Go benchmark harness existed for this
   smoke, so implementation readiness is not claimed.
2. External Unicode Lipsum and Base64 identities are frozen by immutable
   commit/tree/path/blob, byte length, and SHA-256 in
   [`corpus-freeze.md`](corpus-freeze.md). A later timing run must materialize
   exactly those bytes in its non-product cache and recheck them on-host.
3. Pinned `findbenchmark` and `benchmark_to_well_formed_utf16` use
   `std::random_device` with no seed/file input. Phase 0 freezes the exact
   benchmark-only capture/replay method in [`corpus-freeze.md`](corpus-freeze.md);
   producing its companion harness, fixtures, and comparative results remains
   a Phase 13 gate. Any existing raw replay artifacts under `.omx` are
   nonauthoritative mutable evidence only and do not satisfy or extend Phase 0.
4. macOS has no verified hard CPU-affinity tool. Later local comparisons must
   record the same no-affinity policy and quiet-host conditions rather than
   claim Linux-style pinning.
5. The passing one-iteration C++ invocations are smoke only. They must not be
   reported as throughput or substituted for the Phase 13 repetition protocol.
