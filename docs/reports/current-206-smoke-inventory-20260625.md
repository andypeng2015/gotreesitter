# Current 206 Smoke Inventory - 2026-06-25

## Scope

This report banks a telemetry-first N=1 smoke re-inventory of the current 206 grammar set.
It is not a release scan, not a full-corpus parity scan, and must not replace the full
N=40/all-files release gate.

Source worktree/branch:

- Worktree: `/home/draco/work/gotreesitter-build-baseline`
- Branch: `repair/java-token-admission-reduced`
- HEAD: `163dfde0406db0c2f832943cc46f16f5c709dc4d`
- HEAD summary: `163dfde0 improve(parser): improve lexer token preference for extra layout context`

## Command

```bash
GTS_CORPUS_DIR=/home/draco/work/gotreesitter-corpora/corpus_sources \
GTS_TIER_SCAN_N=1 \
GTS_TIER_SCAN_ROUNDS=1 \
GTS_TIER_SCAN_TIMEOUT=90 \
GTS_TIER_SCAN_KILL_AFTER=20s \
GTS_TIER_SCAN_ISOLATE_FILES=1 \
GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 \
GTS_TIER_SCAN_PARALLELISM=8 \
GTS_TIER_SCAN_SHARDS=32 \
GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS=0 \
GTS_TIER_SCAN_MAX_FRAME_ROWS=2000 \
bash cgo_harness/docker/run_tier_scan_parallel.sh \
  cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625
```

Artifact root:

`/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625`

## Control Plane Status

The parent command returned rc 1 because scan/ratchet classification failed. The merge completed.

| Metric | Value |
| --- | ---: |
| Selected | 206 |
| Visited | 206 |
| Measured | 205 |
| Classified | 206 |
| In progress | 0 |
| Parent telemetry complete_worker_count | 32 |
| Parent telemetry running | 0 |
| Parent telemetry pending | 0 |
| Terminal timeout/fail family rows | 0 |

## Counts

| Classification | Count |
| --- | ---: |
| Clean | 146 |
| Tier IV | 59 |
| Unmeasured | 1 |

The unmeasured grammar is `elsa`, which is a zero-files external.

## Diagnostic Family Counts

| Family | Count |
| --- | ---: |
| runtime_frontier_stop | 41 |
| recovery_error_cost | 30 |
| accepted_shape_materialization | 11 |
| accepted_divergence_cost | 13 |
| terminal_timeout_or_fail | 0 |
| scanner_token_accounting | 0 |
| unclear_needs_diagnostic | 0 |

Overlaps exist between `recovery_error_cost` and the `accepted_divergence_cost` /
frontier families.

## Signature Buckets

| Signature | Grammars |
| --- | ---: |
| `files=1 diverge=0 trunc=0 errTree=0 panics=0` | 141 |
| `files=1 diverge=1 trunc=1 errTree=0 panics=0` | 23 |
| `files=1 diverge=1 trunc=1 errTree=1 panics=0` | 18 |
| `files=1 diverge=1 trunc=0 errTree=0 panics=0` | 11 |
| `files=1 diverge=1 trunc=0 errTree=1 panics=0` | 7 |
| `files=1 diverge=0 trunc=0 errTree=1 panics=0` | 5 |

## Example Queues

Top runtime frontier examples:

`earthfile`, `commonlisp`, `vimdoc`, `asm`, `tlaplus`, `doxygen`, `glsl`, `less`,
`scala`, `nushell`.

Top accepted-shape/materialization examples:

`fsharp`, `perl`, `swift`, `cylc`, `pug`, `fennel`, `nginx`, `just`, `html`,
`purescript`, `regex`.

## Stale Classification Signal

| Signal | Value |
| --- | ---: |
| Current clean with IV row | 107 |
| Current non-clean with stale clean/missing/non-IV row | 1 |
| Same-denominator parity differs | 2 |
| Sample-size differs from TSV | 158 |

The current non-clean grammar with a stale clean/missing/non-IV row is `regex`.
The same-denominator parity differs grammars are `graphql` and `regex`.

This means the old 166 Tier-IV inventory is stale. The next step should be
current-state reclassification, not parser edits from old assumptions.

## Next Diagnostic Queue

1. Run N=40 focused tier scans for the current N=1 Tier-IV set before promoting any clean rows.
2. Run wringer/deep diagnostics on representatives: `earthfile` or `tlaplus` for frontier stop, `fsharp` or `dockerfile` for accepted expensive shape/recovery, `regex` for stale CLEAN regression.
3. Only after repeated witnesses agree, map the failure family to generalized machinery.

No source/parser/grammar changes were made; report only.
