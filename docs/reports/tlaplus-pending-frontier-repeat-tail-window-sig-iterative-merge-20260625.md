# TLA+ pending-frontier repeat-tail window-signature iterative merge

Date: 2026-06-25

Prior report: `tlaplus-pending-frontier-repeat-tail-window-signature-20260625.md` at `856f672c`.

## Scope

Code changes were made only in isolate:

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go`

Main repo parser/runtime code was not modified. This report is the only main-repo file added.

The isolate adds a production-path env mode:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1`

When enabled, `pendingFrontierRepeatTailWindowSignaturesEquivalent` uses the same explicit-stack flattener proven by the classifier run. Default behavior remains unchanged. The iterative mode preserves child traversal order, retains the window flat cap of `256`, preserves `child[i]_unavailable` mismatch semantics, and has no recursive depth cap.

## Artifacts

Artifact root:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625`

Build artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625/20260625T015428Z-build-measure`

Replay artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625/20260625T015457Z-replay-window-sig-iterative-merge`

Each Docker wrapper artifact contains `container.log`, `inspect.json`, and `metadata.txt`.

## Commands

Build:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625 \
  --label build-measure \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && mkdir -p harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625 && timeout --kill-after=10s 180s /usr/bin/time -v env CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625/measure.test ."
```

Replay:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625 \
  --label replay-window-sig-iterative-merge \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && /usr/bin/time -v timeout --kill-after=10s 60s env -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE REPRO_LANG=tlaplus REPRO_DIR=/workspace/corpus_sources REPRO_EXTS=.tla REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_PARSE_PROGRESS=1 GOT_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_TOP_KEYS=3 GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1000 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=3 GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=2 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1 harness_out/tlaplus-bakery-window-sig-iterative-merge-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Explicitly not enabled:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE`

## Build Result

- wrapper rc: `0`
- `/usr/bin/time -v` exit status: `0`
- OOM: false
- wall: `0:12.90`
- max RSS: `967044 KB`

## Replay Result

The replay used Docker isolation, one grammar (`tlaplus`), one file (`Bakery.tla`), and the corpus mounted read-only at `/workspace/corpus_sources`.

- wrapper rc: `124`
- `/usr/bin/time -v` exit status: `124`
- timeout: yes
- OOM: false
- wall: `1:03.97`
- max RSS: `1569536 KB`
- last observed progress before timeout: token `199`, byte `460..461`, `max_stacks=204`, `live_stacks=6`

Late counter sample at elapsed `53436 ms`, token `5[459..460]`:

```text
pending_frontier_gss_attempt=1130909
pending_frontier_gss_merge=87263
pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:0,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:1043646}
pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
pending_frontier_repeat_tail_window_sig={attempt:1130909,prefix_equal:1130909,prefix_not_equal:0,window_equal:1130909,window_incomplete:0,window_capped:0,candidate:1130909,merge:87263,gss_reject:1043646,non_equal:0,no_entries:0,samples:16}
```

Final parse progress line before timeout:

```text
PARSE-PROGRESS phase=dispatch_begin lang=tlaplus source_bytes=22753 expected_eof=22753 elapsed_ms=53802 iter=389 tokens=199 stacks=6 live_stacks=6 max_stacks=204 node_count=219444 peak_depth=198 need_token=false single_iters=11 multi_iters=378 token_symbol=5 token_start=460 token_end=461 token_no_lookahead=false token_eof=false num_stacks=6
```

Window samples were all complete/equal. Example:

```text
GLR-PENDING-FRONTIER-REPEAT-TAIL-WINDOW-SIG sample=16 equal=true complete=true bucket=flatten_equal window_start=5 first_diff=7 key={state:4014 byte:65} token={symbol:6 start:64 end:65} branch={target:48 fork:49} first_flat_diff=none
```

## Comparison to `856f672c`

The prior recursive-flattening window-signature run timed out and had the late counter:

```text
pending_frontier_repeat_tail_window_sig={attempt:5379912,prefix_equal:5379912,prefix_not_equal:0,window_equal:6183,window_incomplete:5373729,window_capped:0,candidate:6183,merge:5312,gss_reject:871,non_equal:0,no_entries:0,samples:16}
```

Iterative production flattening changed the proof behavior materially:

| metric | recursive window sig at `856f672c` | iterative window sig |
|---|---:|---:|
| attempts | 5379912 | 1130909 |
| window_equal | 6183 | 1130909 |
| window_incomplete | 5373729 | 0 |
| window_capped | 0 | 0 |
| candidates | 6183 | 1130909 |
| successful GSS merges | 5312 | 87263 |
| GSS rejects after proof | 871 | 1043646 |
| non_equal | 0 | 0 |
| no_entries | 0 | 0 |

The old blocker collapsed: `window_incomplete` went from `5373729` to `0`, and every attempted window proof was complete/equal in this replay.

Practical parser progress did not improve enough to be merge-ready. The prior recursive run reached roughly token `192` by 54s with `max_stacks=1063`; this iterative run reached token `199` by 53.8s with `max_stacks=204`, then timed out. Compared to the gate-off baseline cited in `856f672c`, both are still much worse: gate-off was around token `422` at 80s and around token `449` at timeout. The iterative path creates many more proof candidates and successful GSS merges, but it also spends the bounded run attempting and rejecting over a million post-proof GSS merges.

## Conclusion

Iterative flattening is a real production-path win for correctness/proof completeness: it removes the recursive depth-cap failure and leaves no incomplete, capped, unavailable, or non-equal window proofs in this TLA+ Bakery replay.

Decision: not viable as a gated production path yet. It needs bucketed or cached selectivity before merge. The next generalized step should keep the iterative proof but avoid invoking the expensive post-proof GSS merge for every complete/equal local window, likely by caching `(gss head, materializing count, windowStart)` results and/or adding a cheap bucket that distinguishes windows which can actually pass `tryMergePendingFrontierForkCandidate`.
