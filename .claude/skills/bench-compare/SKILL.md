---
name: bench-compare
description: Compare benchmark results before and after a change using benchstat and this repo's old.txt/new.txt convention. Use when evaluating any performance-affecting change.
---

Compare performance with `benchstat` (install if missing: `go install golang.org/x/perf/cmd/benchstat@latest`).

1. On the baseline (pre-change) tree, run:
   `go test -run='^$' -bench=<pattern> -count=10 ./... | tee old.txt`
2. Apply the change, then run the exact same command into `new.txt`.
3. Compare: `benchstat old.txt new.txt`.

Rules:

- `old.txt`, `new.txt`, and `bench.txt` are gitignored on purpose — never commit them.
- Run benchmarks sequentially; never run anything else (builds, tests) in parallel with a benchmark run.
- Use identical `-bench` patterns and `-count` values for both runs, on the same machine.
- Report the benchstat table verbatim; call a change significant only if benchstat's p-value column says so.
