# TLA+ pending-frontier repeat-tail window incomplete classification

Date: 2026-06-25

Related prior reports:

- `b2913404` - full-stack repeat-tail signature cache experiment.
- `856f672c` - pending-frontier repeat-tail window-signature experiment.

## Scope

Instrumentation was implemented only in isolate:

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go`

Main repo parser/runtime was not modified.

## Classifier definition

New env-gated trace-only path:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY=1`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_MAX=<n>`, default `200000`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_SAMPLES=<n>`, default `32`

The classifier runs at the same distinct-materializing-shape pending-frontier point as the prior window-signature proof. It does not merge. When enabled, it computes the window proof/classification, updates counters/samples, then preserves the original distinct-shape rejection path.

Classifier logic:

1. Count materializing result entries for target/fork and bucket count pair/delta.
2. Classify zero counts or failed `stackMaterializingResultEntries` as `entries_unavailable`.
3. Find the first materializing-entry difference with `pendingFrontierEntryFirstDiffBucket`.
4. If no difference and counts are equal, classify as `complete_equal`.
5. If counts differ with no entry mismatch, set `firstDiff=max(0,minCount-1)` and bucket as length mismatch.
6. Set `windowStart=max(0,firstDiff-2)` and verify the prefix before the window remains equal. A prefix mismatch is `prefix_not_equal`.
7. Flatten `targetEntries[windowStart:]` and `forkEntries[windowStart:]` with `pendingFrontierRepeatTailWindowFlatCap`.
8. Incomplete flatten reasons are split by mismatch text:
   - contains `flat_cap` -> `flat_cap`
   - contains `depth_cap` -> `depth_cap`
   - contains `raw_child_unavailable` -> `raw_child_unavailable`
   - contains `child[` or `child_unavailable` -> `child_unavailable`
   - otherwise -> `other_incomplete`
9. Complete flattened windows are compared with `pendingFrontierCompareFlatEntries`, yielding `complete_equal` or `complete_non_equal`.
10. Attempts after the max-attempt guard are counted as `skipped_after_max`.

Additional shape telemetry:

- count pair: equal, target longer, fork longer
- count delta: 0, 1, 2-4, 5-16, 17+
- first-diff distance from end
- suffix materializing entry count
- flat output length bucket

## Run artifacts

Artifact root:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-classify-20260625`

Build artifact:

`20260625T012052Z-build-measure`

Build command inside Docker:

```sh
cd /workspace/cgo_harness &&
mkdir -p harness_out/tlaplus-bakery-repeat-tail-window-classify-20260625 &&
/usr/bin/time -v env CGO_ENABLED=1 go test -c -tags treesitter_c_parity \
  -o harness_out/tlaplus-bakery-repeat-tail-window-classify-20260625/measure.test .
```

Build result:

- rc: 0
- elapsed: 10.88s
- max RSS: 897980 KB
- OOM: false

First replay attempt:

- artifact: `20260625T012121Z-replay-classify`
- rc: 0
- result: non-durable; `/workspace/corpus_sources/.../Bakery.tla` was not mounted and the file was skipped.

Durable replay artifact:

`20260625T012313Z-replay-classify-mounted`

The replay mounted `/home/draco/work/gotreesitter-corpora/corpus_sources` read-only at `/workspace/corpus_sources`.

Replay command inside Docker:

```sh
cd /workspace/cgo_harness &&
timeout --kill-after=10s 60s env \
  REPRO_LANG=tlaplus \
  REPRO_DIR=/workspace/corpus_sources \
  REPRO_EXTS=.tla \
  REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla \
  REPRO_PROGRESS=1 \
  REPRO_SIGNATURES=1 \
  REPRO_N=1 \
  REPRO_ROUNDS=1 \
  GOT_GLR_FANOUT_TRACE=1 \
  GOT_GLR_FANOUT_TRACE_MIN_STACKS=1 \
  GOT_GLR_FANOUT_TRACE_INTERVAL_MS=5000 \
  GOT_GLR_MERGE_TELEMETRY=1 \
  GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=8 \
  GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=1 \
  GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 \
  GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_MAX=200000 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_SAMPLES=64 \
  /usr/bin/time -v harness_out/tlaplus-bakery-repeat-tail-window-classify-20260625/measure.test \
    -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v
```

Replay result:

- rc: 124
- elapsed from Docker state: 62.613676s
- timeout: expected `timeout --kill-after=10s 60s`
- OOM: false
- replay max RSS: unavailable because the timeout killed the `/usr/bin/time` wrapper before it emitted stats
- parser reached late fanout at elapsed_ms around 52041 before container timeout

No `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1` merge env was set.

## Key counters

Late durable counter line:

```text
pending_frontier_repeat_tail_window_classify={attempt:200000,entries_unavailable:0,flat_cap:0,depth_cap:193491,child_unavailable:0,raw_child_unavailable:0,other_incomplete:0,complete_equal:6509,complete_non_equal:0,prefix_not_equal:0,skipped_after_max:19842756,samples:64,count_pair:{equal:0,target_longer:200000,fork_longer:0},count_delta:{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636},first_diff_from_end:{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0},suffix_entries:{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0},flat_len:{0:0,1_8:1349,9_32:154569,33_128:44082,129_255:0,256_plus:0}}
```

Interpretation:

- Classified attempts hit the max guard exactly: `attempt=200000`.
- Dominant incomplete cause is `depth_cap=193491` (96.75% of classified attempts).
- `complete_equal=6509` (3.25%).
- No observed `entries_unavailable`, `flat_cap`, child unavailable, raw child unavailable, `other_incomplete`, `complete_non_equal`, or `prefix_not_equal`.
- All classified attempts had `target_longer`.
- All classified attempts had `first_diff_from_end=1`.
- Count deltas are broad: `5_16=85062`, `17_plus=76636`, `2_4=28217`, `1=10085`.
- Suffix entry buckets split roughly evenly between `5_16=99602` and `17_plus=100398`.
- Flat lengths before depth cap are mostly `9_32=154569`, then `33_128=44082`.

Sample incomplete line shape:

```text
GLR-PENDING-FRONTIER-REPEAT-TAIL-WINDOW-CLASSIFY sample=1 reason=depth_cap equal=false complete=false bucket=span_mismatch counts={target:16 fork:9 delta:7} first_diff=7 first_diff_from_end=1 window_start=5 suffix_entries=11 flat_len=12 key={state:4014 byte:69} token={symbol:6 start:68 end:69} branch={target:166 fork:173} first_flat_diff=entry[2]:depth_cap ...
```

The samples consistently point at nested repeat flattening in the block comment text area, with `first_flat_diff=entry[2]:depth_cap` and token/state around `state=4014`, `symbol=6`, `block_comment_text_token2`.

## Conclusion

The prior `window_incomplete` bucket is not caused by entry unavailability, flat output cap, or raw child unavailability in this run. It is overwhelmingly a depth cap while flattening the suffix window.

This means the previous window proof avoided the flat cap but failed because the flattening recursion depth limit is too shallow for the nested repeat representation encountered in TLA+ block-comment text tails.

## Next directed experiment

Run a classify-only replay with a configurable repeat-tail flatten depth cap, still no merge:

- Add `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_DEPTH_CAP=<n>` or a classify-only override.
- Keep `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_MAX=200000`.
- Test depth caps `12`, `16`, and `24` independently on the same Bakery replay.

Expected decision point:

- If `depth_cap` collapses into `complete_equal` without `flat_cap` rising, then the next safe precheck is a deeper bounded flatten or iterative repeat-chain flattener.
- If `depth_cap` becomes `flat_cap`, then the right next step is a repeat-chain summary/hash precheck instead of deeper materialization.
