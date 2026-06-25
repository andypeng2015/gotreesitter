# TLA+ window-signature GSS reject classification

Date: 2026-06-25

Prior report: `tlaplus-pending-frontier-repeat-tail-window-sig-iterative-merge-20260625.md` at `3acbd0b6`.

## Scope

Code changes were made only in isolate:

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/glr.go`

Main repo parser/runtime code was not modified. This report is the only main-repo file added.

The isolate adds trace-only classification for window-signature iterative post-proof GSS rejects:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1`
- optional sample cap: `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_SAMPLES`

Default behavior is unchanged. The classifier runs only after a complete/equal window-signature proof and only after `tryMergePendingFrontierForkCandidate` rejects. It mirrors cheap candidate checks, `gssMainCanMergeForParser` preconditions, and the first failing shape in a non-mutating GSS preflight.

## Artifacts

Artifact root:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625`

Build artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625/20260625T020705Z-build-measure`

Replay artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625/20260625T020740Z-replay-window-sig-iterative-gss-reject-classify`

Each Docker wrapper artifact contains `container.log`, `inspect.json`, and `metadata.txt`.

## Commands

Package compile in isolate:

```sh
go build .
```

The host-side `go test . -run '^$'` was attempted first, but the isolate currently has pre-existing test API drift unrelated to this trace patch (`newNoTreeReduceNodeInArena`, `applyAction`, and missing generated conflict-choice symbols). `go build .` passed.

Build:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625 \
  --label build-measure \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && mkdir -p harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625 && timeout --kill-after=10s 180s /usr/bin/time -v env CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625/measure.test ."
```

Replay:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625 \
  --label replay-window-sig-iterative-gss-reject-classify \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && /usr/bin/time -v timeout --kill-after=10s 60s env -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE REPRO_LANG=tlaplus REPRO_DIR=/workspace/corpus_sources REPRO_EXTS=.tla REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_PARSE_PROGRESS=1 GOT_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_TOP_KEYS=3 GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1000 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=3 GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=2 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_SAMPLES=16 harness_out/tlaplus-bakery-window-sig-iterative-gss-reject-classify-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Explicitly not enabled:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE`

## Build Result

- wrapper rc: `0`
- `/usr/bin/time -v` exit status: `0`
- OOM: false
- wall: `0:12.45`
- max RSS: `952268 KB`

## Replay Result

The replay used Docker isolation, one grammar (`tlaplus`), one file (`Bakery.tla`), and the corpus mounted read-only at `/workspace/corpus_sources`.

- wrapper rc: `124`
- `/usr/bin/time -v` exit status: `124`
- timeout: yes
- OOM: false
- wall: `1:01.80`
- max RSS: `1566292 KB`
- last observed progress before timeout: token `181`, byte `423..424`, `max_stacks=186`, `live_stacks=6`

Latest full counter sample at elapsed `50560 ms`, token `5[422..423]`:

```text
pending_frontier_gss_attempt=840446
pending_frontier_gss_merge=71398
pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:0,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:769048}
pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
pending_frontier_repeat_tail_window_sig={attempt:840446,prefix_equal:840446,prefix_not_equal:0,window_equal:840446,window_incomplete:0,window_capped:0,candidate:840446,merge:71398,gss_reject:769048,non_equal:0,no_entries:0,samples:16}
pending_frontier_repeat_tail_window_sig_gss_reject_classify={accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,c_recovery_cost_differs:0,missing_head:0,dead_accepted_mismatch:0,score_mismatch:0,shifted_mismatch:0,top_state_byte_mismatch:0,clean_zero_error_target:0,clean_zero_error_fork:0,reach_b_can_reach_a:0,reach_prev_can_reach_n:0,payload_no_equivalent_link:0,max_link_count_no_replacement:0,replacement_not_better_precedence:0,nodes_can_merge_reach:0,nodes_can_merge_state:0,nodes_can_merge_clean:0,nodes_can_merge_uniform_byte_offset:0,recursive_can_merge_or_can_add_failure:769048,actual_merge_unknown:0,samples:16}
```

Final parse progress line before timeout:

```text
PARSE-PROGRESS phase=dispatch_begin lang=tlaplus source_bytes=22753 expected_eof=22753 elapsed_ms=50763 iter=353 tokens=181 stacks=6 live_stacks=6 max_stacks=186 node_count=180060 peak_depth=180 need_token=false single_iters=11 multi_iters=342 token_symbol=5 token_start=423 token_end=424 token_no_lookahead=false token_eof=false num_stacks=6
```

Window-signature samples remained complete/equal. Example:

```text
GLR-PENDING-FRONTIER-REPEAT-TAIL-WINDOW-SIG sample=1 equal=true complete=true bucket=flatten_equal window_start=5 first_diff=7 key={state:4014 byte:63} token={symbol:6 start:62 end:63} branch={target:6 fork:7} first_flat_diff=none
```

GSS reject samples all classified to the same bucket. Example:

```text
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT sample=1 reason=recursive_can_merge_or_can_add_failure key={state:4014 byte:68} token={symbol:6 start:67 end:68} branch={target:123 fork:129} head_links={target:1 fork:1}
```

## Classification

The cheap candidate preconditions did not explain the rejects:

- accepted/nil: `0`
- entries/missing GSS: `0`
- state/byte mismatch: `0`

The `gssMainCanMergeForParser` preconditions also did not explain them:

- C recovery cost differs: `0`
- missing head: `0`
- dead/accepted mismatch: `0`
- score mismatch: `0`
- shifted mismatch: `0`
- top state/byte mismatch: `0`
- clean-zero/error-link target: `0`
- clean-zero/error-link fork: `0`

The post-`canMerge` GSS merge classifier attributed every sampled and counted reject to recursive node/link preflight:

- recursive canMergeNodes/canAddLink failure: `769048`
- all other first-shape buckets: `0`
- unknown: `0`

This means the proof filter is not losing on stack headers, C recovery cost, clean-zero/error-link preconditions, missing heads, or top-level reach/link-cap replacement buckets. The collapse is inside recursive GSS node/link compatibility after the main heads have passed the parser-level preconditions.

## Conclusion

The iterative window-signature proof remains complete/equal for every attempted candidate in this bounded TLA+ Bakery replay. The blocker is now narrower: post-proof GSS rejects are uniformly recursive `canMergeNodes`/`canAddLink` failures.

Next generalized target: add one more trace-only sub-classifier inside the recursive bucket, preserving the same non-mutating preflight, to split the recursive failure by exact nested reason and path depth/link shape. The likely production selectivity/cache target is a memoized `(target head, fork head)` or `(node pair, link payload signature)` negative-result cache for recursive GSS preflight, keyed independently of grammar, rather than a TLA+-specific window or grammar rule.
