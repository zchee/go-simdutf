# Public API manifest for simdutf `c7bef0ff14a1`

This manifest records the Go-observable semantic surface and its live
implementation status. Its sole upstream authority is commit
`c7bef0ff14a13fd6ea52e3347da2c659383392de` (tree
`4cbac4c5d1ce0d7f98cc35360d53725433f12811`), using the detached checkout's
`include/simdutf/implementation.h`, `include/simdutf/error.h`, and
`include/simdutf/encoding_types.h`. C++ span/template mechanics are collapsed;
character-width and enum-type overloads that Go observes remain separate.

## Schema

`api-manifest.tsv` has one row per Go-observable semantic overload, public semantic type/constant, or evidence-backed exclusion. `header_path_line` pins declaration evidence. `feature_gate` records the controlling upstream gate. `overload_disposition` states mapping/collapse/split/exclusion. Unit, result/count, destination, and alias columns freeze observable contracts. Source columns contain exact pinned-tree occurrence paths or an explicit `not_applicable` reason. `status` is `planned` or `implemented` for in-scope declarations and `excluded` only with `exclusion_reason`. `implemented` records a completed API row; it does not by itself claim that a phase hard gate is complete.

The frozen mappings use `[]byte`, `[]uint16`, and `[]uint32`; `size_t` becomes `int`; width-distinct overloads use `UTF16`; UTF-16 LE/BE slices represent raw storage; native UTF-16 functions delegate by native endian. Ordinary destination-taking operations preserve the upstream sufficiently-sized destination precondition with a stable Go bounds panic. Bounded Latin-1/UTF-16-to-UTF-8 safe forms return only written `int`; Base64 safe forms return `(Result, written int)`.

`Find` and `FindUTF16` translate the pinned return pointer into a Go `int`:
the zero-based input byte/code-unit index of the first match, or `len(input)`
when the value is absent. The pinned `findbenchmark` and `shortbench`
`find_equal` procedure exercise only `const char *`; `FindUTF16` therefore has
no exact upstream benchmark correspondence.

The pinned Base64 benchmark sources likewise pass only `const char *` input.
Consequently the UTF-16 overload rows `MaximalBinaryLengthFromBase64UTF16`,
`BinaryLengthFromBase64UTF16`, `Base64ToBinaryUTF16`, and
`Base64ToBinarySafeUTF16` are exact benchmark `not_applicable`; a benchmark for
the byte overload is not evidence for a `const char16_t *` procedure.

## Totals

- Total classification rows: **164**
- In-scope callable rows: **147** (**144** operations plus **3** pinned
  result/value helpers)
- In-scope public type rows: **6**
- In-scope public constant rows: **2**
- Evidence-backed exclusions: **9**
- Implementation status: **19 implemented**, **136 planned**, **9 excluded**
- Candidate-ledger symbols reconciled: **133 / 133**
- Additional public semantic/type/overload/exclusion rows found independently:
  **24**

### By family

| Family | Rows |
|---|---:|
| ASCII | 5 |
| Base64 | 29 |
| C++ mechanics | 3 |
| Encoding detection | 9 |
| Find | 2 |
| Result/error | 7 |
| Shared helper | 1 |
| Transcoding/length | 86 |
| UTF-16 | 16 |
| UTF-32 | 2 |
| UTF-8 | 4 |

## Candidate-ledger reconciliation

All 133 distinct `simdutf::...` symbols in `.omx/plans/port-simdutf-dec3aad192f4.md` have at least one classified manifest row. Missing ledger symbols: **none**. The independent AST/header pass expands name-only entries where Go observes distinct types or widths: three `to_string` enum inputs, byte/UTF-16 `find`, byte/UTF-16 Base64 length/decode/details/safe forms, and byte/UTF-16 Base64 predicates. This contributes 11 additional overload rows beyond the ledger's one-row-per-name shape. It also classifies 13 public type, constant, and value-helper declarations: 11 in scope (6 types, 2 constants, and 3 helpers) plus 2 exclusions (`endianness` and `match_system`).

## Legitimate exclusions

- `implementation`/registry APIs are C++ dispatch mechanics, not semantic operations.
- `SIMDUTF_SPAN`, templates, and C++23 constexpr wrappers collapse into Go slices/runtime calls.
- `SIMDUTF_ATOMIC_REF` Base64 overloads are conditional experimental APIs; the pinned header states they are not normally tested or fuzzed.
- `std::text_encoding` adapters are conditional C++ standard-library integration.
- `endianness` and `match_system` are C++ native-endian mechanics. Section 5.3
  maps native Go operations to internal `encoding/binary.NativeEndian`
  delegation, so neither becomes an exported convenience API.

Each exclusion has a dedicated TSV row with its exact gate and header evidence.

### Exact `SIMDUTF_SPAN` exclusion evidence

At the pinned tree, `include/simdutf/implementation.h` is the only public header
containing `SIMDUTF_SPAN`-guarded public span/template declarations. Excluding
the import-only header gate at lines 16-22, its exact guarded declaration and
support blocks are:

```text
72-152,207-225,244-250,266-280,298-313,327-341,358-373,390-403,418-431,446-459,
479-492,512-525,545-557,578-591,611-624,644-657,673-687,703-717,733-747,
768-781,803-816,833-852,870-893,909-926,940-956,997-1014,1033-1050,1069-1084,
1105-1120,1141-1156,1171-1186,1204-1220,1236-1253,1275-1292,1313-1330,
1348-1365,1383-1400,1418-1434,1452-1469,1494-1511,1527-1544,1558-1576,
1590-1607,1623-1640,1654-1670,1687-1703,1723-1739,1761-1778,1800-1816,
1840-1870,1891-1908,1928-1945,1963-1980,2001-2018,2037-2054,2076-2093,
2112-2129,2150-2167,2190-2207,2227-2244,2264-2281,2299-2316,2334-2351,
2369-2386,2405-2422,2447-2466,2489-2508,2531-2550,2569-2586,2603-2620,
2641-2657,2675-2690,2708-2723,2744-2759,2779-2795,2815-2831,2849-2866,
2883-2900,2917-2934,2951-2965,2987-3002,3017-3031,3046-3060,3080-3096,
3116-3133,3150-3167,3187-3202,3219-3234,3254-3271,3292-3309,3333-3352,
3401-3416,3437-3452,3472-3488,3508-3524,3541-3559,3576-3593,3610-3627,
3646-3661,3678-3692,3709-3723,3742-3756,3775-3790,3809-3824,3844-3857,
3875-3888,3906-3919,3937-3951,3968-3984,4004-4018,4036-4050,4068-4082,
4245-4260,4278-4291,4309-4324,4343-4356,4416-4437,4492-4509,4539-4558,
4605-4614,4678-4698,4752-4773,4828-4849,5045-5083,7187-7258,7226-7256
```

The final two ranges are intentionally nested: the public free-function span
wrapper block at 7187-7258 contains an atomic-Base64 span block at 7226-7256.
`include/simdutf/portability.h:18-27` only decides whether `SIMDUTF_SPAN` is
defined; it declares no semantic overload. The list contains exactly 126
ranges, is ordered by opening line, and includes both nested ranges
`7187-7258` and `7226-7256`; no shorthand such as “per-operation blocks” is
a declaration anchor.

Reproduce the range set from the pinned header by matching every exact
`#if SIMDUTF_SPAN` whose closing line is `#endif // SIMDUTF_SPAN`, pairing with
a stack, sorting by opening line, and asserting the count is 126. The unrelated
import-only gate at `16-22` closes with a plain `#endif` and is intentionally
outside this declaration-range set.

## Reproduction and completeness commands

Run from the Go repository root with `UP=/var/folders/dp/89wwdp754235m6jmdzj7p5lm0000gn/T/go-simdutf-upstream.ynLkUM/simdutf`:

```sh
git -C "$UP" rev-parse HEAD HEAD^{tree}
grep '^| ' .omx/plans/port-simdutf-dec3aad192f4.md | sed -E 's/^\| [^|]+ \| `([^`]+)`.*/\1/' | grep '^simdutf::' | sort -u
clang++ -std=c++20 -I"$UP/include" -Xclang -ast-dump=json -fsyntax-only /tmp/manifest.cc > /tmp/manifest-ast.json
jq -r '.. | objects | select(.kind == "FunctionDecl") | [.name, .type.qualType, (.loc.file // ""), (.loc.line // 0)] | @tsv' /tmp/manifest-ast.json > /tmp/all-functions.tsv
# Compare AST function names present in the public free-function region against manifest symbols; output must be empty.
python3 - "$UP/include/simdutf/implementation.h" <<'PY'
import csv, re, sys
src = "".join(open(sys.argv[1]).readlines()[:5086])
known = {r["upstream_symbol"].split("::")[-1].split()[0] for r in csv.DictReader(open("docs/porting/api-manifest.tsv"), delimiter="\t")}
names = {row[0] for row in csv.reader(open("/tmp/all-functions.tsv"), delimiter="\t") if row and re.search(r"\b" + re.escape(row[0]) + r"\s*\(", src)}
nonpublic = {"base64_to_binary_details_impl", "convert", "convert_safe_constexpr", "convert_valid", "convert_with_errors", "count_code_points", "data", "is_base64", "is_base64_or_padding", "is_ignorable", "min", "ref", "size", "tail_encode_base64", "validate", "validate_with_errors"}
unclassified = sorted(names - known - nonpublic)
print("\n".join(unclassified))
assert not unclassified
PY
rg -n '^#(if|elif|else|endif).*SIMDUTF_(FEATURE|SPAN|CPLUSPLUS23|ATOMIC)' "$UP/include/simdutf/implementation.h"
rg -n 'enum (error_code|encoding_type|endianness|base64_options|last_chunk_handling_options)|struct (result|full_result)|check_bom|bom_byte_size' "$UP/include/simdutf"
python3 - "$UP/include/simdutf/implementation.h" <<'PY'
import re, sys
lines = open(sys.argv[1]).read().splitlines()
stack, ranges = [], []
for line_number, line in enumerate(lines, 1):
    if re.match(r'\s*#if\s+SIMDUTF_SPAN\b', line):
        stack.append(line_number)
    elif re.match(r'\s*#endif\s*//\s*SIMDUTF_SPAN\b', line):
        ranges.append((stack.pop(), line_number))
assert stack == [16], stack  # import-only gate closes with an unlabelled #endif
ranges.sort()
print(",".join(f"{start}-{end}" for start, end in ranges))
assert len(ranges) == 126
PY
rg -n 'simdutf::find|find_equal|char16_t' "$UP/benchmarks/find" "$UP/benchmarks/shortbench.cpp"
rg -n 'maximal_binary_length_from_base64|binary_length_from_base64|base64_to_binary|char16_t' "$UP/benchmarks/base64/benchmark_base64.cpp" "$UP/benchmarks/shortbench.cpp"
python3 - <<'PY'
import csv
p='docs/porting/api-manifest.tsv'
rows=list(csv.reader(open(p), delimiter='\t'))
assert len(set(map(len, rows))) == 1, set(map(len, rows))
h=dict(enumerate(rows[0])); idx={v:k for k,v in h.items()}
for n,r in enumerate(rows[1:], 2):
    assert r[idx['status']] in ("planned", "implemented", "excluded"), (n, r[idx['status']])
    assert r[idx['overload_disposition']], n
    if r[idx['status']] == "excluded": assert r[idx['exclusion_reason']] != "not_applicable", n
statuses = {status: 0 for status in ("implemented", "planned", "excluded")}
for r in rows[1:]: statuses[r[idx['status']]] += 1
assert statuses == {"implemented": 19, "planned": 136, "excluded": 9}, statuses
print(f"{len(rows)-1} rows, {len(rows[0])} columns; {statuses}")
PY
```

The AST command classifies function overloads; the `rg` commands independently cover gates, public semantic types, helpers, and conditional exclusions that a function-only scan would miss.
