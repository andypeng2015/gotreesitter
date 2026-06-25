# TLA+ Pending-Frontier Repeat-Tail Window Signature Experiment

Date: 2026-06-25

Isolate: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate`

Main repo changes: this report only.

## Question

The earlier full-stack repeat-tail signature experiment was not a good generalized proof path: it flattened complete materializing streams with a cap of 64 entries, so most comparisons were incomplete or capped. This experiment tested a directed alternative at the same decision point in `tryMergePendingFrontierFork`, after `gssStacksHaveDistinctMaterializingShapes(target, fork)` rejects the ordinary GSS merge by materializing-shape hash.

The diagnostic proof condition was:

1. Build `stackMaterializingResultEntries` for target and fork.
2. Find the first strict raw/materializing entry divergence before the count mismatch using the same fields as `pendingFrontierEntryFirstDiffBucket`: symbol, span, production, parse state, pre-goto state, child count, exact flags, raw shape ref, and raw shape production.
3. Choose `windowStart = max(0, firstDiff - 2)`. If there is no strict diff before `minCount` and the counts differ, use `firstDiff = max(0, minCount - 1)`.
4. Require strict prefix equality for `[0:windowStart]`.
5. Flatten only the suffix windows `targetEntries[windowStart:]` and `forkEntries[windowStart:]`, with repeat expansion and a bounded cap of 256 flattened entries.
6. If the suffix windows are complete and equal, call the existing `tryMergePendingFrontierForkCandidate(target, fork)` path and preserve branch order.

No grammar-specific symbol names were used by the new merge proof.

## Run

Build artifact:

`cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-sig-20260625/measure.test`

Docker wrapper artifact:

`cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-sig-20260625/20260625T010721Z-direct-window-sig-on`

Direct replay artifacts:

- `cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-sig-20260625/direct-window-sig-on.log`
- `cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-sig-20260625/direct-window-sig-on.status`
- `cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-sig-20260625/direct-window-sig-on.time`

Exact replay shape:

```sh
timeout --kill-after=10s 60s env \
  REPRO_LANG=tlaplus \
  REPRO_DIR=/workspace/corpus_sources \
  REPRO_EXTS=.tla \
  REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla \
  REPRO_PROGRESS=1 \
  REPRO_SIGNATURES=1 \
  REPRO_N=1 \
  REPRO_ROUNDS=1 \
  GOT_PARSE_PROGRESS=1 \
  GOT_PARSE_PROGRESS_INTERVAL_MS=1000 \
  GOT_GLR_FANOUT_TRACE=1 \
  GOT_GLR_FANOUT_TRACE_TOP_KEYS=3 \
  GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 \
  GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1000 \
  GOT_GLR_MERGE_TELEMETRY=1 \
  GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=3 \
  GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=2 \
  GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 \
  GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 \
  ./harness_out/tlaplus-bakery-repeat-tail-window-sig-20260625/measure.test \
  -test.run '^TestMeasureDtierVsC$' -test.v
```

## Result

The run timed out:

- direct status: `direct_rc=124`
- `/usr/bin/time -v`: elapsed `1:05.05`, max RSS `1581252 kB`, exit status `124`
- Docker wrapper: exit code `0` because the direct command captured status; `oom_killed=false`

Late counter sample at elapsed `54003 ms`, token `5[452..453]`:

```text
pending_frontier_gss_attempt=5379912
pending_frontier_gss_merge=5312
pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:5373729,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:871}
pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
pending_frontier_repeat_tail_window_sig={attempt:5379912,prefix_equal:5379912,prefix_not_equal:0,window_equal:6183,window_incomplete:5373729,window_capped:0,candidate:6183,merge:5312,gss_reject:871,non_equal:0,no_entries:0,samples:16}
```

Interpretation:

- The proof condition is real: all sampled complete windows were equal, and the run found `6183` complete/equal windows with `5312` successful pending-frontier GSS merges by 54s.
- The new proof avoided the old "flat cap" failure mode for the local windows: `window_capped=0`.
- The dominant failure mode moved to incomplete suffix flattening: `window_incomplete=5373729`.
- There were no strict prefix failures and no unequal complete windows in this run: `prefix_not_equal=0`, `non_equal=0`.
- Despite additional merges, the diagnostic was too expensive and did not improve the bounded run. It reached only about token `192` by 54s with `max_stacks=1063`.

## Comparison

Gate-off baseline artifact:

`cgo_harness/harness_out/tlaplus-bakery-pending-frontier-gss-20260624c/baseline`

Baseline timed out at 90s, but had better parser progress: at 80s heartbeat it was around token `422`, `max_stacks=516`; at timeout it was around token `449`, `max_stacks=516`.

Full-stack signature artifact:

`cgo_harness/harness_out/tlaplus-bakery-repeat-tail-sig-merge-valid-20260625/direct-repeat-tail-sig-on.*`

The valid full-stack signature run also timed out at 60s: elapsed `1:04.76`, max RSS `1486284 kB`, exit status `124`. Its main durable finding was the old problem: full flattened signatures were usually incomplete/capped, so it could not provide a useful generalized merge proof.

Repeat-tail equivalence artifact:

`cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625/direct-repeat-tail-equiv-on.log`

That run also timed out. A late sample near 53.8s had `pending_frontier_repeat_tail_equiv_candidate=553`, `merge=505`, and `non_equal=1804313`. The window signature found many more candidates/merges (`6183`/`5312` by 54s), but at much higher attempt cost and with worse token progress.

## Conclusion

This is useful evidence but not a ready direction as implemented.

The generalized strict-prefix plus local flattened-suffix proof is sound as a diagnostic shape and avoids the old full-stack cap failure on the sampled successful cases. However, it still asks the wrong amount of work per rejected pending-frontier fork: millions of attempts compute windows, and almost all fail because suffix flattening is incomplete rather than capped. The resulting run times out and regresses progress versus the gate-off baseline.

The next generalized experiment should keep the proof shape but make it cheaper and more selective:

- run only when the first strict divergence is near the current frontier/tail, not for every distinct materializing shape;
- avoid flattening from a long suffix when `windowStart` is still far from the end;
- add a cheap raw-shape child-availability precheck before expansion;
- consider a cache keyed by `(gss head, materializing count, windowStart)` only after the selectivity gate exists.

Do not promote `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG` as-is.
