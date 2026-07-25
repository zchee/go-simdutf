---
name: port-routine
description: Port a routine from upstream C++ simdutf to Go following this repo's tiered implementation and testing policy. Use when implementing, extending, or reviewing any simdutf routine (UTF-8/16/32 validation/transcoding, Base64).
---

Port one routine at a time from upstream [simdutf/simdutf](https://github.com/simdutf/simdutf).

## Steps

1. **Read upstream first.** Locate the routine in the upstream repo: per-ISA kernels live under `src/<isa>/` (`arm64`, `haswell`, `icelake`, `westmere`, `rvv`, `ppc64`, `lsx`, `lasx`), shared vector algorithms under `src/generic/`, and the portable non-SIMD fallback under `src/fallback/`. Understand the algorithm and its edge cases (truncated sequences, surrogates, overlong encodings) before writing any Go.
2. **Write the pure-Go scalar implementation.** This is the reference oracle and portable fallback. Match upstream semantics exactly, including error kinds and error positions.
3. **Port upstream test vectors** into a Go test using the map-based named-case pattern. Cover the edge cases upstream tests cover.
4. **Add a benchmark** for the scalar path using `for b.Loop()`.
5. **Add accelerated variants** (these may land in follow-up PRs):
   - hand-written `.s` assembly for amd64 and/or arm64, or
   - amd64 `simd/archsimd` intrinsics behind `//go:build goexperiment.simd`.

   Wire each variant into the CPU-feature dispatch with scalar as the fallback.
6. **Fuzz-difference every accelerated variant against scalar**: a `Fuzz*` test feeding the same input to both and requiring identical outputs, including errors.
7. **Verify**: `go test ./...` and, when the intrinsics tier is touched, `GOEXPERIMENT=simd go test ./...`.
