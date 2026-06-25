# Current Family Representatives Wringer - 2026-06-25

Source worktree: `/home/draco/work/gotreesitter-build-baseline`

Source branch: `repair/java-token-admission-reduced`

Source HEAD: `163dfde0`

This follows `docs/reports/current-206-smoke-inventory-20260625.md` and probes representatives from the current N=1 residual families. This is diagnostic evidence only; it is not parser or grammar work.

## Command

```bash
GTS_CORPUS_DIR=/home/draco/work/gotreesitter-corpora/corpus_sources \
GTS_WRINGER_PARALLEL_DRY_RUN=0 \
GTS_WRINGER_PARALLELISM=5 \
GTS_WRINGER_N=1 \
GTS_WRINGER_ROUNDS=1 \
GTS_WRINGER_TIMEOUT=90 \
GTS_WRINGER_KILL_AFTER=20s \
GTS_WRINGER_MODE=full-integrity \
GTS_WRINGER_MAX_SUSPICIOUS=1 \
GTS_WRINGER_MAX_DIAG_FILES=1 \
GTS_WRINGER_PARSE_PROGRESS=1 \
GTS_WRINGER_TIER_SCAN_MAX_FRAME_ROWS=2000 \
GTS_WRINGER_KEEP_PARSER_PROGRESS_ROWS=0 \
bash cgo_harness/docker/run_grammar_integrity_wringer_parallel.sh \
  --langs 'earthfile tlaplus fsharp dockerfile regex' \
  --parallelism 5 \
  cgo_harness/harness_out/grammar_integrity_wringer_parallel/current-family-reps-20260625
```

Artifact root:

`/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/grammar_integrity_wringer_parallel/current-family-reps-20260625`

## Parent Summary

The parent run completed cleanly: rc 0, complete, open workers 0, workers=5, grammars=5, selected_frames=5, residual_frames=5, planned_actions=30, completed_actions=30, incomplete_actions=0, failed=0.

Merged family counts:

| Family | Count |
| --- | ---: |
| truncation_frontier_loss | 2 |
| version_or_corpus | 2 |
| recovery_error_shape | 1 |

Compact family labels need interpretation, especially for `fsharp` and `regex`.

## Per-Grammar Diagnostics

| Grammar | Representative file | Baseline | Key signal | Classification |
| --- | --- | --- | --- | --- |
| earthfile | `earthfile/Earthfile` | truncation_frontier_loss | `no_stacks_alive`; Go root span `0:16030` vs C `0:42326`; `62.12x`; stack2/stack8/node3 still diverge+trunc; forest `go_no_tree`; first diff root | frontier budget/runtime frontier stop |
| tlaplus | `tlaplus/specifications/Bakery-Boulangerie/Bakery.tla` | `memory_budget` | Go root span `0:2368` vs C `0:22753`; `24.36x`; stack2 moves to `iteration_limit`; stack8/node3 remain `memory_budget`; forest diverges with stop=none | frontier budget/runtime frontier stop |
| fsharp | `fsharp/.github/skills/fsharp-diagnostics/server/DesignTimeBuild.fs` | accepted, not truncated | type diff at `root[0][9][0][1]`; `1230.15x`; first diff `value_declaration_left` vs `function_declaration_left`; stack2 still diverges but drops to `14.20x` | accepted shape/materialization, even though compact label says version_or_corpus/unclassified_type_diagnostic |
| dockerfile | `dockerfile/3.10/alpine3.22/Dockerfile` | recovery_error_shape | accepted root `ERROR` vs `source_file`; errTree=1; `1644.17x`; stack variants show the same accepted root divergence; forest `go_no_tree` | recovery/version competition with error-cost overlap |
| regex | `regex/configs/examples/regexExample.regex` | accepted child-count diff | diff at `root[0][70]`; current `0/1` while TSV says CLEAN; first diff `any_character` has Go child `'.'`, C has no child | stale CLEAN regression plus accepted shape/token-node accounting detail |

## Conclusions

Do not touch per-grammar patches from this evidence.

The next generalized machinery target should start with frontier budget/runtime frontier stop. It is the largest current family and repeats across `earthfile` and `tlaplus`, with variants failing to clear it.

Accepted shape/materialization and recovery error-shape are separate follow-up tracks.

`regex` needs a focused regression confirmation before changing ratchet rows.

No source, parser, or grammar changes were made; this is a report-only change.
