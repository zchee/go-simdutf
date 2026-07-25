# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Package `simdutf` is a Go port of the C++ [simdutf/simdutf](https://github.com/simdutf/simdutf) library: Unicode routines (UTF-8/UTF-16/UTF-32 validation and transcoding) and Base64. When porting a routine, consult the upstream implementation for algorithm semantics and edge-case behavior rather than inventing — keep names and structure diffable against upstream.

## Implementation tiers

Every routine ships as up to three implementations, selected by runtime CPU-feature dispatch:

1. **Pure-Go scalar** — written first; the portable fallback and the correctness oracle for all accelerated variants.
2. **Hand-written Go assembly** (`.s`) for amd64 and arm64. arm64 (NEON) acceleration must use this tier.
3. **`simd/archsimd` intrinsics** behind the `//go:build goexperiment.simd` tag — amd64-only as of Go 1.26, and both building and testing this tier require `GOEXPERIMENT=simd`.

amd64 and arm64 come first; other upstream architectures (AVX-512, RISC-V Vector, LoongArch64, POWER) come later.

## Testing

- Port upstream simdutf test vectors into Go tests.
- Every accelerated implementation needs a fuzz test differencing it against the pure-Go scalar reference (identical outputs, including errors).

## Conventions

- Every `.go` file starts with the Apache 2.0 license header — copy it from `doc.go`.
