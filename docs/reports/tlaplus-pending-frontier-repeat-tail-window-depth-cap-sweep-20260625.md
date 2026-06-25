# TLA+ pending-frontier repeat-tail window depth-cap sweep

Date: 2026-06-25

Prior report: `tlaplus-pending-frontier-repeat-tail-window-incomplete-classification-20260625.md` at `4dfefdb8`.

## Scope

Code changes were made only in isolate:

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go`

Main repo parser/runtime code was not modified.

The isolate now has a classify-only env override:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_DEPTH_CAP=<n>`
- default: `pendingFrontierRepeatTailDepthCap` (`8`)

The override is used only by `classifyPendingFrontierRepeatTailWindow`. Existing repeat-tail flattening, signature, and window-signature behavior still use the existing default depth cap. No repeat-tail merge/window-signature env was enabled.

## Artifacts

Artifact root:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625`

Build artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625/20260625T013357Z-build-measure`

Replay artifacts:

- cap 12: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625/20260625T013426Z-replay-depth-cap-12`
- cap 16: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625/20260625T013545Z-replay-depth-cap-16`
- cap 24: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625/20260625T013707Z-replay-depth-cap-24`

Each Docker wrapper artifact contains `container.log`, `inspect.json`, and `metadata.txt`.

## Commands

Build:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625 \
  --label build-measure \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && mkdir -p harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625 && timeout --kill-after=10s 180s /usr/bin/time -v env CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625/measure.test ."
```

Replay template, run independently with `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_DEPTH_CAP` set to `12`, `16`, and `24`:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625 \
  --label replay-depth-cap-<cap> \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && timeout --kill-after=10s 60s env REPRO_LANG=tlaplus REPRO_DIR=/workspace/corpus_sources REPRO_EXTS=.tla REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_MIN_STACKS=1 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=5000 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=8 GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=1 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_MAX=200000 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_SAMPLES=64 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_CLASSIFY_DEPTH_CAP=<cap> /usr/bin/time -v harness_out/tlaplus-bakery-repeat-tail-window-depth-cap-sweep-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Unset/not enabled:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE`

## Build Result

- rc: 0
- OOM: false
- wall: 13.51s from `/usr/bin/time -v`
- max RSS: 947216 KB

## Replay Results

All replays used Docker isolation, one grammar (`tlaplus`), one file (`Bakery.tla`), and the corpus mounted read-only at `/workspace/corpus_sources`.

| depth cap | rc | timeout | OOM | Docker wall | replay max RSS | attempt | depth_cap | flat_cap | complete_equal | complete_non_equal | entries/child/raw unavailable | skipped_after_max |
|---:|---:|---|---|---:|---|---:|---:|---:|---:|---:|---:|---:|
| 12 | 124 | yes | false | 65.17s | unavailable | 200000 | 183489 | 0 | 16511 | 0 | 0 | 19875396 |
| 16 | 124 | yes | false | 65.05s | unavailable | 200000 | 169855 | 0 | 30145 | 0 | 0 | 19875396 |
| 24 | 124 | yes | false | 61.58s | unavailable | 200000 | 134763 | 0 | 65237 | 0 | 0 | 18731601 |

Replay max RSS is unavailable because the timeout killed the `/usr/bin/time -v` wrapper before it emitted stats. Docker inspect reports `OOMKilled=false` for all three replay containers.

Common buckets for all caps:

- `count_pair`: `{equal:0,target_longer:200000,fork_longer:0}`
- `count_delta`: `{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636}`
- `first_diff_from_end`: `{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0}`
- `suffix_entries`: `{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0}`

Flat length buckets:

| depth cap | flat_len 0 | 1_8 | 9_32 | 33_128 | 129_255 | 256_plus |
|---:|---:|---:|---:|---:|---:|---:|
| 12 | 0 | 1349 | 142541 | 56110 | 0 | 0 |
| 16 | 0 | 1349 | 130514 | 68137 | 0 | 0 |
| 24 | 0 | 1349 | 106466 | 92185 | 0 | 0 |

Late counter lines:

```text
cap=12 pending_frontier_repeat_tail_window_classify={attempt:200000,entries_unavailable:0,flat_cap:0,depth_cap:183489,child_unavailable:0,raw_child_unavailable:0,other_incomplete:0,complete_equal:16511,complete_non_equal:0,prefix_not_equal:0,skipped_after_max:19875396,samples:64,count_pair:{equal:0,target_longer:200000,fork_longer:0},count_delta:{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636},first_diff_from_end:{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0},suffix_entries:{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0},flat_len:{0:0,1_8:1349,9_32:142541,33_128:56110,129_255:0,256_plus:0}}
cap=16 pending_frontier_repeat_tail_window_classify={attempt:200000,entries_unavailable:0,flat_cap:0,depth_cap:169855,child_unavailable:0,raw_child_unavailable:0,other_incomplete:0,complete_equal:30145,complete_non_equal:0,prefix_not_equal:0,skipped_after_max:19875396,samples:64,count_pair:{equal:0,target_longer:200000,fork_longer:0},count_delta:{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636},first_diff_from_end:{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0},suffix_entries:{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0},flat_len:{0:0,1_8:1349,9_32:130514,33_128:68137,129_255:0,256_plus:0}}
cap=24 pending_frontier_repeat_tail_window_classify={attempt:200000,entries_unavailable:0,flat_cap:0,depth_cap:134763,child_unavailable:0,raw_child_unavailable:0,other_incomplete:0,complete_equal:65237,complete_non_equal:0,prefix_not_equal:0,skipped_after_max:18731601,samples:64,count_pair:{equal:0,target_longer:200000,fork_longer:0},count_delta:{0:0,1:10085,2_4:28217,5_16:85062,17_plus:76636},first_diff_from_end:{0:0,1:200000,2_4:0,5_16:0,17_plus:0,unknown:0},suffix_entries:{1:0,2_4:0,5_16:99602,17_plus:100398,unknown:0},flat_len:{0:0,1_8:1349,9_32:106466,33_128:92185,129_255:0,256_plus:0}}
```

## Conclusion

Raising the classify-only depth cap from 8 to 12/16/24 converts more cases from `depth_cap` to `complete_equal`, and it does not create any observed `flat_cap`, `complete_non_equal`, entry unavailable, child unavailable, raw child unavailable, or prefix-not-equal cases.

However, even cap 24 leaves `depth_cap=134763` out of `attempt=200000` (67.38%). That is still the dominant incomplete cause.

Decision point: the next experiment should continue toward a bounded or iterative repeat-tail flattener before moving to summary/hash machinery. The evidence says the flat output cap is not the immediate limiting factor, but cap 24 is not deep enough to prove that ordinary bounded recursion is sufficient. A larger bounded cap sweep or an iterative repeat-chain flattener can test whether `depth_cap` collapses fully without producing `flat_cap`; summary/hash machinery becomes the better path if the remaining depth failures move into materialization size/flat-cap pressure.
