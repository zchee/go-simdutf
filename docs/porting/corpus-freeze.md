# Phase 0 benchmark corpus freeze

This document closes the Phase 0 external-corpus identification work for the
fixed semantic authority
[`simdutf/simdutf@dec3aad192f47081110d9c766d4917bad243906f`](https://github.com/simdutf/simdutf/tree/dec3aad192f47081110d9c766d4917bad243906f).
It is intentionally limited to the two files copied under
`.omx/artifacts/phase0/benchmark-corpora/corpus/`; those ignored artifacts and
their [provenance](../../.omx/artifacts/phase0/benchmark-corpora/provenance.md)
are the canonical byte evidence.

## Frozen external inputs

The pinned README explicitly recommends the Unicode dataset and gives
`lipsum/Arabic-Lipsum.utf8.txt` as its example
([README lines 2838--2853](https://github.com/simdutf/simdutf/blob/dec3aad192f47081110d9c766d4917bad243906f/README.md#L2838-L2853)).
It also demonstrates Base64 DNS input using `base64data/dns/*.txt`
([README lines 2868--2874](https://github.com/simdutf/simdutf/blob/dec3aad192f47081110d9c766d4917bad243906f/README.md#L2868-L2874)).

| ID | Immutable source | Exact selected upstream path/blob | Bytes | SHA-256 |
| --- | --- | --- | ---: | --- |
| `unicode-lipsum-arabic-utf8` | [`lemire/unicode_lipsum@b0f1d0c1cb0cb168fc08dbf0e3b7100cdec517dc`](https://github.com/lemire/unicode_lipsum/tree/b0f1d0c1cb0cb168fc08dbf0e3b7100cdec517dc), tree `6008726380b0539e1cdc74956fa4554efdb13db5` | [`lipsum/Arabic-Lipsum.utf8.txt`](https://github.com/lemire/unicode_lipsum/blob/b0f1d0c1cb0cb168fc08dbf0e3b7100cdec517dc/lipsum/Arabic-Lipsum.utf8.txt), blob `a02d207a57ff12e04d9b41611e7756c88a751935` | 81,685 | `b20003e7999187985e931b1b0404f9f273576b3e9bbd77bda7466de5f26a15bb` |
| `base64data-dns` | [`lemire/base64data@3e89f846f15b04e7ee16713984dbdd9d8b5928d4`](https://github.com/lemire/base64data/tree/3e89f846f15b04e7ee16713984dbdd9d8b5928d4), tree `91030cda0f4daaa22ed3ca7e0f581bd31407fca0` | [`dns/swedenzonebase.txt`](https://github.com/lemire/base64data/blob/3e89f846f15b04e7ee16713984dbdd9d8b5928d4/dns/swedenzonebase.txt), blob `a57bef558b1fb09153a252432e5c6162d9498bb2`, the complete `dns/*.txt` expansion at that tree | 35,100,000 | `d837553e61f96476a75350142085983b366beb9754ed6f21ff9299878d2a1de0` |

The external repositories were freshly cloned during the UTC retrieval window
2026-07-26T00:32:00Z--2026-07-26T00:32:49Z. Each detached worktree was clean:
empty porcelain status and zero exits from both indexed and worktree diff
checks. The canonical two-row sorted `sha256<TAB>byte_count<TAB>path` manifest
has SHA-256
`40456572d957bf16d0f9bd0d190cf6befe98674a40edcdb2010478fd95c112fb`.

The artifacts are ignored runtime evidence—not product source or testdata. Use
only the frozen files and verify them with:

```sh
(cd .omx/artifacts/phase0/benchmark-corpora && shasum -a 256 -c SHA256SUMS)
```

### License/notice boundary

This is a factual provenance record, not legal advice or a new license claim.
At its frozen revision, `unicode_lipsum` commits no `LICENSE`, `COPYING`, or
`NOTICE`; its [README](https://github.com/lemire/unicode_lipsum/blob/b0f1d0c1cb0cb168fc08dbf0e3b7100cdec517dc/README.md#L1-L18)
attributes lipsum material to `rusticstuff/simdutf8` by Hans Kratz and states
MIT/Apache licensing. `base64data` likewise commits no conventional license or
notice and its [README](https://github.com/lemire/base64data/blob/3e89f846f15b04e7ee16713984dbdd9d8b5928d4/README.md#L1-L4)
only describes the DNS source. GitHub's license endpoint returned HTTP 404 for
both frozen repositories during retrieval. Thus no unsupported licensing
conclusion has been inferred; a source-specific review is mandatory before any
redistribution or shipping of either external payload.

## Phase 13 companion method for unseeded benchmark generators

The pinned `findbenchmark` creates 10,000 alphanumeric bytes from a fresh
`std::random_device`-seeded `std::mt19937`, appends `=`, and accepts no
input/seed option ([source lines 24--62](https://github.com/simdutf/simdutf/blob/dec3aad192f47081110d9c766d4917bad243906f/benchmarks/find/findbenchmark.cpp#L24-L62)).
The pinned UTF-16 target similarly seeds `std::mt19937` from
`std::random_device`; its generator uses the documented basic, high-surrogate,
low-surrogate, and percent distributions ([lines 23--77](https://github.com/simdutf/simdutf/blob/dec3aad192f47081110d9c766d4917bad243906f/benchmarks/benchmark_to_well_formed_utf16.cpp#L23-L77)).
Neither stock target can establish identical Go/C++ bytes. Its fresh random
output remains an exact-upstream diagnostic and is not comparable to a Go run.

Phase 0 freezes the exact method: a separate benchmark-only Phase 13 companion
captures one input using the pinned generator, serializes it before timing,
records its size, parameters, and SHA-256, and provides a verification/replay
mode consumed by both C++ and Go. It must not modify, replace, or claim to be the
stock upstream executable.

1. The find capture reproduces the exact 62-character alphabet,
   `std::random_device`-seeded `std::mt19937`, uniform `[0,61]` distribution,
   10,000 draws, and terminal `=`, yielding a 10,001-byte artifact.
2. The UTF-16 capture reproduces the pinned generator and records the requested
   length, actual code-unit count, surrogate-pair and mismatch percentages, and
   datapoint count. Its artifact is BOM-less little-endian UTF-16; replay rejects
   any size, parameter, or digest mismatch without regeneration.
3. C++ and Go verify and consume the same captured bytes. Input preparation and
   destination allocation stay outside timing, and the denominator is input
   bytes: `len(findInput)` or `2*len(utf16Input)`.
4. The companion preserves ten warmups per fixer, 100 timed iterations, the
   exact `get_chunk_range_simple` sequence, and the pinned prefix behavior. The
   source times `data[0:start+length]`; its constructed `input_data` subvector is
   not passed to `bench`
   ([lines 258--280](https://github.com/simdutf/simdutf/blob/dec3aad192f47081110d9c766d4917bad243906f/benchmarks/benchmark_to_well_formed_utf16.cpp#L258-L280)).
5. Reports are labelled `Phase 13 companion comparability harness`, not stock
   upstream timing, and are never fed to Go `benchstat`.

No companion harness, captured fixture, or timing result is required or made
authoritative by Phase 0. Any existing replay material under `.omx` is
nonauthoritative mutable evidence only. Producing and validating the companion,
materializing inputs on each timing host, and running comparative measurements
remain Phase 13 gates.

## Phase 0 status

**Resolved:** the two upstream-recommended external corpus selections are
immutable, copied byte-for-byte, individually hashed, and covered by a
deterministic aggregate manifest. The exact benchmark-only capture/replay method
for the two unseeded generators is frozen for Phase 13 implementation.

**Deferred:** Phase 13 owns the companion harness and captured fixtures,
host-side materialization and hash rechecks, full C++ timing, committed Go
scalar-baseline/candidate benchmark revisions and their actual serialized 10+10
timings, Go-only `benchstat`, affinity/load capture, and final Go/C++ reports.
The absent source licenses remain a redistribution/shipping review constraint.
No performance result is claimed by the Phase 0 evidence.
