# Java Token Layout Canary - 2026-06-25

## Source

- Worktree: `/home/draco/work/gotreesitter-build-baseline`
- Branch: `repair/java-token-admission-reduced`
- HEAD: `163dfde0406db0c2f832943cc46f16f5c709dc4d`
- Commit: `163dfde0 improve(parser): improve lexer token preference for extra layout context`

## Command Shape

Per-grammar Docker tier scan using:

- `GTS_TIER_SCAN_ISOLATE_FILES=1`
- `GTS_TIER_SCAN_N=40`
- `GTS_TIER_SCAN_ROUNDS=1`
- `GTS_TIER_SCAN_TIMEOUT=180`
- `GTS_TIER_SCAN_KILL_AFTER=30`
- `GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1`
- Corpus root: `/home/draco/work/gotreesitter-corpora/corpus_sources`

Artifact root:

`/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/tier_scan/java-token-layout-canary-20260625`

## Results

| Grammar | RC | Classification | Parity | Counters | Notes |
| --- | ---: | --- | --- | --- | --- |
| `java` | 1 | `tier_iv` | `parityMatch=39/40(98%)` | `diverge=1 trunc=1 errTree=1 panics=0` | Stale published/TSV row was 22/40; residual frame 32/40 `SpellCheckService.java`. |
| `json` | 0 | clean | `parityMatch=40/40(100%)` | `diverge=0 trunc=0 errTree=0 panics=0` | Stays clean. |
| `go` | 0 | clean | `parityMatch=40/40(100%)` | | Stale TSV row was IV-unknown 25/40, so current classification appears stale for this 40-file canary. |
| `python` | 0 | clean | `parityMatch=40/40(100%)` | | Stale TSV row was IV-unknown 6/40, so current classification appears stale for this 40-file canary. |

## Java Residual

Residual frame 32:

- Path: `/home/draco/work/gotreesitter-corpora/corpus_sources/java/framework-docs/src/main/java/org/springframework/docs/core/aot/hints/importruntimehints/SpellCheckService.java`
- Go root: `ERROR` span `0:1218`, child count 16, errors 6
- C root: `program` span `0:1499`, child count 9, errors 0
- Go stop: `no_stacks_alive`
- Runtime: `tokens=106 lastTokenEnd=1224 expectedEOF=1499 iterations=159/44970 nodes=186/300000 maxStacks=6`

Classification: runtime frontier stop / recovery error-cost, not token-source text-block escape bug.

## Conclusions

- `163dfde0` materially improves Java but does not close Java in the tier scan harness: 22/40 stale published row -> 39/40 current canary.
- Go and Python should be re-inventoried because their current 40-file canaries are clean while published/TSV rows are stale.
- Next work should be full telemetry-first 206 re-inventory before more machinery edits.
- No per-grammar patch or parser change was made by this report.
