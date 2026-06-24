# Tier IV Failure Families

This inventory classifies the current Tier IV surface by failure family so the
debt burn-down can target generalized machinery instead of one-off grammar
normalizers.

## Source Of Truth

The canonical current classification is
`cgo_harness/tier_scan/tier_classification.tsv`, reconciled against the latest
scan artifact:
`harness_out/tier_scan/20260621T035630Z-206-inventory-telemetry/tier_scan.txt`,
with targeted wringer reconciliations noted in the TSV.
The generated `docs/reports/tiers.md` and `docs/reports/tiers.json` artifacts
were stale before regeneration and should be treated as publication artifacts,
not as the hand-edited source.

`regex` is no longer Tier IV. The latest scan reports `regex` as clean
(`1/1`, no divergence, truncation, error tree, or panic), so its previous
`IV-perf` row is retired and the clean ratchet includes it.

After that correction and the 2026-06-23 `beancount` reconciliation, the Tier
IV inventory has 166 rows:

| family | count |
| --- | ---: |
| recovery/error-cost | 100 |
| truncation/node budget | 13 |
| timeout/fanout/perf | 19 |
| scanner fidelity | 4 |
| unknown accepted-shape | 26 |
| version/corpus | 2 |
| shift-gap/pre-recovery | 1 |
| extmap/no-corpus | 1 |

## 2026-06-23 Full-Integrity Wringer Family Evidence

Fresh full-integrity wringer runs under
`cgo_harness/harness_out/wringer_full_integrity_families_20260623/{cpp,beancount,norg,csv}`
closed cleanly at the harness layer. Each run selected one suspicious file
(`selected=1`, `suspicious=1`), executed the closed three-action plan
(`planned=3`, `completed=3`, `incomplete=0`, `unexpected=0`, `open=0`), and
passed strict wringer assertions. These are targeted N=1 family witnesses, not
replacement scan truth for the TSV.

- `cpp` sampled `/workspace/corpus_sources/cpp/include/fmt/args.h`.
  Baseline and stack-cap-2 remained nonmatching. Go stopped with
  `goStop=no_stacks_alive`; the Go `translation_unit` spans `0:641` while C
  spans `0:7192`, both roots have errors, and the sampled frame has
  `lastTokenEnd=652` versus `expectedEOF=7192`. This is truncation or
  pre-recovery evidence inside the broader recovery family, not a clean
  recovery-only witness.
- `beancount` was the accepted-shape/materialization proof target in this
  family run: it accepted a full-span no-error tree but diverged on root child
  materialization. The later
  `cgo_harness/harness_out/wringer_beancount_all_root_tail_20260623T192449/wringer_summary.json`
  run closed clean `3/3` with `family_counts {clean:3}`, resolving this via
  the generalized accepted clean-tail forest/root finalization fix.
- `norg` sampled `/workspace/corpus_sources/norg/doc/cheatsheet.norg`.
  The witness remains nonmatching with `goStop=no_stacks_alive`: Go
  materializes `document` span `0:381` while C spans `0:5382`, with no root
  errors. The observed max-stack count is high at baseline (`164`), and
  stack-cap-2 lowers the observed max but does not produce parity. This
  scanner row also carries truncation/node-budget overlay in the sampled
  frame.
- `csv` sampled `/workspace/corpus_sources/csv/data/co2-concentration.csv`.
  Baseline, stack-cap-2, and first-diff replay timed out, but the plan closed.
  The latest parser-progress heartbeat at timeout is
  `forest_reduce_visit_begin` with `iter=tokens=1831`,
  `token_start=7622`, `token_end=7632`, `stacks=0`, `frontier_len=1`,
  `work_len=49`, `reducer_visits=82`, `reduce_symbol=16`, `child_count=2`,
  and `no_extras=true`. This reinforces `csv` as forward-progress forest
  parent/materialization throughput rather than a no-token-progress loop.

## Families

### Recovery/Error-Cost

Count: 100.

Representative grammars: `asm`, `c`, `cpp`, `dart`, `glsl`, `html`, `rust`,
`swift`, `wgsl`.

These rows are dominated by cases where Go and C choose different recovery
versions, root envelopes, child counts, or error placements. The repeated note
pattern is "faithful C error-cost version competition"; several grammars also
show localized recovery-shape residuals after compatibility normalizers.

Generalized machinery needed: a faithful C-compatible recovery scorer and
version competition path, plus attribution that separates pre-recovery
fragmentation from recovery election and result materialization.

Next experiments:

- Pick one high-volume language with stable witnesses, such as `c`, `html`, or
  `dart`, and trace C-vs-Go recovery candidate scoring at the first divergent
  version choice.
- Record whether the divergence is introduced before recovery, during
  version competition, or while materializing the accepted result.
- Replace grammar-local result compatibility only when the same rule is proven
  across multiple recovery families.

### Truncation/Node Budget

Count: 13.

Representative grammars: `chatito`, `doxygen`, `jsdoc`, `promql`, `tlaplus`,
`turtle`, `wolfram`.

These rows still fail through `node_limit`, `iteration_limit`,
`memory_budget`, or `no_stacks_alive` behavior after the parser enters a large
or ambiguous search space. Some also carry error-tree symptoms, but the first
blocking condition is budget exhaustion or premature stop.

Generalized machinery needed: better frontier pruning, reuse/invalidation
scope control, bounded materialization, and counters that expose where stack
fanout turns into node pressure.

Next experiments:

- Run single-grammar Docker parity on one representative at a time with
  `GOT_PARSE_NODE_LIMIT_SCALE=3` and controlled `GOT_GLR_MAX_STACKS` values.
- Capture max RSS and failure byte for each run to distinguish allocator
  retention from real GLR fanout.
- Keep correctness and performance gates separate: do not accept faster
  truncation as a correctness improvement.

### Timeout/Fanout/Perf

Count: 19.

Representative grammars: `clojure`, `comment`, `csv`, `diff`, `fish`,
`markdown_inline`, `typescript`.

These rows are unmeasured or timeout/OOM in diagnostic scan metadata, usually
because external command execution or GLR fanout exceeds the scan budget. They
are not proven clean merely because a previous ratchet row existed.

Generalized machinery needed: deterministic timeout classification,
single-language Docker repro presets, fanout telemetry, and stable max-RSS
collection on macro workloads.

#### 2026-06-22 Forest Telemetry Split

Bounded Docker probes split the forest timeout surface into at least two
subfamilies, but these N=1 samples do not replace the canonical scan truth in
`cgo_harness/tier_scan/tier_classification.tsv`.

- `tlaplus`:
  `harness_out/docker/20260622T020344Z-tlaplus-forest-telemetry/container.log`
  exited 124 without OOM. Telemetry stalls inside forest reduce/worklist with
  the token index pinned around `iter=1083/1084`, `token_start=2712/2713`,
  `reduce_symbol=613`, `child_count=2`, and `work_len` growing from roughly
  9k to 60k+. `recover_count=0`, `reducer_capped=false`, and
  `reducer_steps=0`. Classification: pinned-token forest worklist convergence
  failure / dirty reprocessing blowup.
- `csv`:
  `harness_out/docker/20260622T021042Z-forest-classify-csv/container.log`
  exited 124 without OOM. It also timed out in forest reduce/worklist, but with
  forward token progress; a representative late line reports `iter=2665`,
  `tokens=2665`, `token_start=11097`, `token_end=11107`, `frontier_len=1`,
  `work_len=32`, `reduce_symbol=16`, `child_count=2`, `recover_count=0`,
  `reducer_capped=false`, and `reducer_steps=0`. Classification:
  bounded-worklist but too-slow forest reduce throughput, not the TLA+
  pinned-token convergence failure.
- `fish`, `beancount`, and `racket` no longer reproduce timeout on the sampled
  N=1 probes. Each completed with forest enabled, but still reported parity
  divergence: `fish` and `racket` remain recovery-shape divergences with
  `errTree=1`; `beancount` later resolved cleanly in
  `cgo_harness/harness_out/wringer_beancount_all_root_tail_20260623T192449/wringer_summary.json`
  after the generalized accepted clean-tail forest/root finalization fix.

Code-read proof for the current guard: `forestReduceStepCap` only increments
in generic DFS paths, not the child-count 2 specialized no-extras reducer path.
Both `tlaplus` and `csv` repeatedly hit `child_count=2` with
`reducer_steps=0`, so the existing reduce cap cannot trip for this family.

Follow-up generalized forest guard experiment:

- Implemented generalized forest guard in `glr_forest.go`: per-token
  `forestReduceVisitCap = 1 << 15` and
  `forestWorklistVisitCap = 1 << 15`. Specialized no-extra reducers now count
  visits, and `flushVisits` stops promptly when capped.
- Verification passed for the code change: `gofmt -w glr_forest.go`,
  `go test -run '^$' .`, and `git diff --check -- glr_forest.go`.
- `tlaplus` guard artifact:
  `/home/draco/work/gotreesitter/harness_out/docker/20260622T022159Z-tlaplus-forest-guard/container.log`
  exited 124 with `oom_killed=false`. Key line:
  `forest_decline ... iter=1083 ... reducer_visits=32769 ... decline_reason=reduce-visit-cap ... reduce_symbol=613 child_count=2`,
  followed by `forest_fast_path_end ... used=false` and
  `parse_internal_begin`. Interpretation: `tlaplus` is no longer an
  unbounded forest reduce/worklist stall; the remaining timeout is now in
  production fallback GLR.
- Production fallback signature from the same `tlaplus` artifact: repeated
  `dispatch_begin`/`dispatch_end` plus
  `merge_cull_begin`/`merge_cull_end`, token pinned around `tokens=450`,
  `token_start=1339`, `token_end=1341`, stacks cycling 12/18,
  `consumed_token=false`, and `node_count` growing past 1.1M.
  Classification: production GLR no-token-progress merge/reduce loop,
  separate generalized machinery target.
- `csv` guard artifact:
  `/home/draco/work/gotreesitter/harness_out/docker/20260622T022411Z-csv-forest-guard/container.log`
  exited 124 with `oom_killed=false`. No `forest_decline`; it still showed
  forward-progress forest reduce throughput, for example
  `iter=2917 tokens=2917 token_start=12147 token_end=12157 work_len=52 reducer_visits=82`.
  Interpretation: `csv` remains in the slow forward-progress forest class.

#### 2026-06-23 CSV parser-progress wringer

Artifact:
`cgo_harness/harness_out/wringer-csv-progress-baseline-20260623T163446Z`.
Docker wrapper exited 0; baseline scan rc 1; selected baseline frame rc 124
timeout; infra status 0; strict plan assertions passed with no incomplete or
unexpected actions
(`incomplete_planned_actions=0`, `unexpected_observed_actions=0`) and
lifecycle closed (`starts=1 terminals=1 open=0`). Selected frame
`/workspace/corpus_sources/csv/data/co2-concentration.csv` is 18,547 bytes
with sha256 `c1a4a970864145940a28225cae288618b156cb32f9a2a1b6606ba7124134febb`.
The baseline log has 107 `PARSE-PROGRESS` rows; `baseline/frames.jsonl`,
`frame_catalog`, and `frame_matrix` surface the same count and terminal
`last_parser_progress`, while `wringer_plan` has 8 heartbeat-derived parser
progress records and a terminal timeout.

Last `MEASURE-PROGRESS` is `go_parse_start`. The terminal status row and last
parser progress agree on `parser_phase=forest_reduce_parent_end`; the final
telemetry has `iter=2317`, `tokens=2317`, `token_start=9647`,
`token_end=9657`, `stacks=0`, `live_stacks=0`, `max_stacks=0`,
`frontier_len=1`, `work_len=30`, `reducer_visits=82`,
`reduce_symbol=16`, and `parent_start=0 parent_end=9647`.
Classification: not token-loop or no-token-progress, because token count and
token span advance; not stack/merge-cull dominated, because stack counters are
zero. This is slow forward-progress forest reduce parent/materialization
throughput around reduce symbol 16 and parent span 0..9647.

- `awk` smoke artifact:
  `/home/draco/work/gotreesitter/harness_out/docker/20260622T022541Z-awk-forest-guard-smoke/container.log`
  exited 0; `forest_fast_path_end used=true`; `parityMatch=1/1(100%)`.

Follow-up production GLR no-token-progress guard:

- Implemented parser guard in `parser.go` with
  `maxConsecutiveNoTokenDispatches=128`. It emits
  `PARSE-PROGRESS phase=no_token_progress_stop` before finalizing with
  `ParseStopIterationLimit`.
- TLA+ guard artifact:
  `/home/draco/work/gotreesitter/harness_out/docker/20260622T024729Z-tlaplus-no-token-progress-guard-final/container.log`
  exited 0 with `oom_killed=false` and no 75s timeout. Key evidence:
  `no_token_progress_stop token_symbol=5 token_start=374 token_end=376 count=129 cap=128 any_reduced=true consumed_token=false`;
  `stop_reason=iteration_limit`; `MEASURE-DTIER parityMatch=0/1 diverge=1 trunc=1`.
  Interpretation: the production GLR no-token-progress class is now
  bounded/diagnostic, not solved for parity.
- `awk` guard artifact:
  `/home/draco/work/gotreesitter/harness_out/docker/20260622T024837Z-awk-no-token-progress-guard/container.log`
  exited 0 with `oom_killed=false`; `forest_fast_path_end used=true`;
  `parityMatch=1/1 diverge=0 trunc=0`. Interpretation: the guard did not
  break a known clean forest case.
- Current next targets: profile and optimize `csv` forward-progress forest
  reduce parent materialization, especially specialized no-extra child-count-2
  reductions and parent span materialization around `reduce_symbol=16`, without
  changing grammar-specific behavior. Treat `cpp`/`norg` as truncation-overlay
  witnesses rather than clean recovery or scanner-only examples.
- 2026-06-23 `beancount` forest A/B proof:
  `cgo_harness/harness_out/wringer_beancount_forest_ab_20260623`, frame
  `base:basic.beancount` sha prefix `fca081a81af7`. Baseline and explicit
  `forest` both nonmatch with materialized root child count Go 121 vs C 30;
  `forest_off` matches with `parityMatch=1/1`. Conclusion: this is forest
  EOF/error-root finalization/root-materialization machinery, not scanner,
  recovery, or grammar-specific behavior.
- 2026-06-23 `beancount` all-root-tail follow-up:
  `cgo_harness/harness_out/wringer_beancount_all_root_tail_20260623T192449/wringer_summary.json`
  closed clean `3/3` with `family_counts {clean:3}` after the generalized
  accepted clean-tail forest/root finalization fix.

Generalized next experiment: the forest pinned-token guard and production GLR
no-token-progress guard are done. The next target is `csv` forward-progress
forest reduce parent/materialization profiling and optimization, especially the
specialized no-extra child-count-2 reduction path and parent span
materialization around `reduce_symbol=16`, without changing grammar-specific
behavior. `beancount` is resolved by the generalized forest/root finalization
fix; follow with a fresh 206 grammar re-inventory/classification before further
parser machinery. These N=1 probes still do not replace full TSV scan truth.

Next experiments:

- Re-run one timeout grammar at a time under Docker with the same corpus and a
  fixed stack cap.
- Separate external scanner command timeout from parser-internal fanout.
- Preserve the diagnostic metadata in the TSV until a grammar has an explicit
  clean scan.

### Scanner Fidelity

Count: 4.

Representative grammars: `agda`, `djot`, `haskell`, `norg`.

These rows require external scanner parity work rather than generic recovery
changes. Layout-sensitive or markdown-class scanners tend to fail before the
parser has a comparable token stream.

Generalized machinery needed: scanner-port checklists, lex-state table
validation against pinned upstream C behavior, and token stream diff tooling
that can run before tree comparison.

Next experiments:

- Diff token streams for one scanner grammar before touching parser runtime.
- Add focused scanner parity fixtures before broad corpus runs.
- Keep scanner fixes isolated from parser machinery changes.

### Unknown Accepted-Shape

Count: 26.

Representative grammars: `ada`, `apex`, `go`, `java`, `kotlin`, `python`,
`xml`, `zig`.

These rows are current non-clean scan truth but the dominant family is not yet
proven. Many have high parity counts and no truncation, which suggests accepted
tree shape, alias, field, or compatibility gaps rather than broad recovery
failure.

Generalized machinery needed: result-shape diff clustering, alias/field
normalizer audit, and a path to retire language-specific normalizers once a
cross-language invariant is found.

Next experiments:

- Cluster first divergent path by node kind, field name, alias, and byte span.
- Prioritize high-pass rows with `39/40` or similar counts because a single
  witness can identify the missing invariant cheaply.
- Require a follow-up scan before promoting any row to `CLEAN`.

### Version/Corpus

Count: 2.

Representative grammars: `cobol`, `disassembly`.

These are not currently parser-runtime failures. The corpus does not match the
pinned grammar's modeled language surface, and C fails too on representative
samples.

Generalized machinery needed: corpus-version metadata and an explicit
"unsupported by pinned grammar" classification path.

Next experiments:

- Keep these out of parser hot-path work.
- Refresh corpus or grammar pins only as a separate compatibility task.

### Shift-Gap/Pre-Recovery

Count: 1.

Representative grammar: `make`.

This family diverges before the C-recovery gate can help. The current witness
shows the same signature with and without the gate, pointing at retry,
pre-error fragmentation, or shift-gap behavior.

Generalized machinery needed: pre-recovery trace attribution that records
where accepted shift choices stop matching C before error-cost competition.

Next experiments:

- Trace the first byte where Go and C stop agreeing on viable shifts.
- Avoid adding another recovery gate until the pre-recovery choice is explained.

### Extmap/No-Corpus

Count: 1.

Representative grammar: `elsa`.

The current classification has no measured corpus surface (`0/0`) and should
not drive parser runtime work until the inventory has a real sample set.

Generalized machinery needed: extension-map and corpus coverage hygiene that
distinguishes "no data" from "clean" and from "failed."

Next experiments:

- Add or confirm corpus coverage before assigning a parser failure family.
- Keep the row Tier IV until the scan has actual files.
