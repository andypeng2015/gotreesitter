# TLA+ pending-frontier repeat-tail window iterative flatten proof

Date: 2026-06-25

Prior report: `tlaplus-pending-frontier-repeat-tail-window-depth-cap-sweep-20260625.md` at `3718050c`.

## Scope

Code changes were made only in isolate:

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go`

Main repo parser/runtime code was not modified.

The isolate adds a classify-only env mode:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_ITERATIVE=1`

That mode is used only inside `classifyPendingFrontierRepeatTailWindow`. The existing repeat-tail merge/window-signature behavior remains unchanged. The iterative flattener uses an explicit stack, preserves child traversal order, retains `pendingFrontierRepeatTailWindowFlatCap` (`256`), preserves `child[i]_unavailable` mismatch semantics, and has no recursive depth cap.

## Artifacts

Artifact root:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625`

Build artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625/20260625T014600Z-build-measure`

Replay artifacts:

- first replay: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625/20260625T014630Z-replay-iterative-flatten`
- timed replay: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625/20260625T014817Z-replay-iterative-flatten-timed`

Each Docker wrapper artifact contains `container.log`, `inspect.json`, and `metadata.txt`.

## Commands

Build:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625 \
  --label build-measure \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && mkdir -p harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625 && timeout --kill-after=10s 180s /usr/bin/time -v env CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625/measure.test ."
```

Replay, timed form used for final RSS/wall capture:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625 \
  --label replay-iterative-flatten-timed \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && /usr/bin/time -v timeout --kill-after=10s 60s env -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE REPRO_LANG=tlaplus REPRO_DIR=/workspace/corpus_sources REPRO_EXTS=.tla REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_MIN_STACKS=1 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=5000 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=8 GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=1 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_MAX=200000 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_SAMPLES=64 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_ITERATIVE=1 harness_out/tlaplus-bakery-repeat-tail-window-iterative-flatten-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Explicitly unset/not enabled:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE`

## Build Result

- rc: 0
- OOM: false
- wall: 12.83s from `/usr/bin/time -v`
- max RSS: 954592 KB

## Replay Results

Both replays used Docker isolation, one grammar (`tlaplus`), one file (`Bakery.tla`), and the corpus mounted read-only at `/workspace/corpus_sources`.

| replay | rc | timeout | OOM | wall | max RSS | attempt | depth_cap | flat_cap | complete_equal | complete_non_equal | entries_unavailable | child_unavailable | raw_child_unavailable | other_incomplete | prefix_not_equal | skipped_after_max | samples |
|---|---:|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| first | 124 | yes | false | unavailable | unavailable | 200000 | 0 | 0 | 200000 | 0 | 0 | 0 | 0 | 0 | 0 | 19319652 | 0 |
| timed | 124 | yes | false | 1:03.70 | 1477796 KB | 200000 | 0 | 0 | 200000 | 0 | 0 | 0 | 0 | 0 | 0 | 19515739 | 0 |

Timed replay bucket summaries:

- `count_pair`: `{equal:0,target_longer:200000,fork_longer:0}`
- `count_delta`: `{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636}`
- `first_diff_from_end`: `{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0}`
- `suffix_entries`: `{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0}`
- `flat_len`: `{0:0,1_8:1349,9_32:86316,33_128:112335,129_255:0,256_plus:0}`

Late timed replay counter:

```text
pending_frontier_repeat_tail_window_classify={attempt:200000,entries_unavailable:0,flat_cap:0,depth_cap:0,child_unavailable:0,raw_child_unavailable:0,other_incomplete:0,complete_equal:200000,complete_non_equal:0,prefix_not_equal:0,skipped_after_max:19515739,samples:0,count_pair:{equal:0,target_longer:200000,fork_longer:0},count_delta:{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636},first_diff_from_end:{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0},suffix_entries:{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0},flat_len:{0:0,1_8:1349,9_32:86316,33_128:112335,129_255:0,256_plus:0}}
```

There were no classifier samples for non-equal, flat-cap, child-unavailable, raw-child-unavailable, prefix-not-equal, or other-incomplete cases because none were observed.

## Conclusion

The iterative classify-only flattener removes the depth-cap blocker for this TLA+ Bakery pending-frontier repeat-tail window proof. At the 200000 classifier attempt cap, all classified cases are `complete_equal`; there are no `depth_cap`, `flat_cap`, `complete_non_equal`, unavailable-child, or other-incomplete outcomes.

Decision: iterative flattening is enough for the repeat-tail window classifier proof. The next production experiment should move the same explicit-stack flattening shape into the actual window-signature/merge proof path and re-run parity/perf gates. Summary/hash machinery is not needed for this classifier blocker unless the production window path later hits flat-cap or memory pressure after recursive depth is removed.
