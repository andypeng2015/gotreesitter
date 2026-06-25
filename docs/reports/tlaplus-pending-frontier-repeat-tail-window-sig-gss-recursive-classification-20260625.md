# TLA+ window-signature recursive GSS reject classification

Date: 2026-06-25

Prior report: `tlaplus-pending-frontier-repeat-tail-window-sig-gss-reject-classification-20260625.md` at `de3dac0f`.

## Scope

Code changes were made only in isolate:

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/glr.go`

Main repo parser/runtime code was not modified. This report is the only main-repo file added.

The isolate deepens the existing trace-only classifier for `recursive_can_merge_or_can_add_failure`. Default behavior is unchanged. The deeper attribution runs only when both classifier gates are enabled:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP=1`

Bound knob used in this run:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP_MAX=200000`

The deep classifier is non-mutating. It mirrors `gssMainPreflight.canMergeNodes`, `canAddLink`, and replacement preflight using the same virtual-link preflight state and attributes the first nested failure. It also buckets recursion depth, path length, head link counts, and failing-node link counts.

## Artifacts

Artifact root:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625`

Build artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625/20260625T021802Z-build-measure`

Replay artifact:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625/20260625T021839Z-replay-window-sig-iterative-gss-recursive-classify`

Each Docker wrapper artifact contains `container.log`, `inspect.json`, and `metadata.txt`.

## Commands

Isolate compile check:

```sh
go build .
```

Build:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625 \
  --label build-measure \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && mkdir -p harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625 && timeout --kill-after=10s 180s /usr/bin/time -v env CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625/measure.test ."
```

Replay:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625 \
  --label replay-window-sig-iterative-gss-recursive-classify \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && /usr/bin/time -v timeout --kill-after=10s 60s env -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE -u GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE REPRO_LANG=tlaplus REPRO_DIR=/workspace/corpus_sources REPRO_EXTS=.tla REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_PARSE_PROGRESS=1 GOT_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_TOP_KEYS=3 GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1000 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_TOP_KEYS=3 GOT_GLR_MERGE_TELEMETRY_MIN_SEEN=2 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_TRACE_SAMPLES=16 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP_MAX=200000 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_SAMPLES=16 harness_out/tlaplus-bakery-window-sig-iterative-gss-recursive-classify-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Explicitly not enabled:

- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE`
- `GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE`

## Build Result

- wrapper rc: `0`
- `/usr/bin/time -v` exit status: `0`
- OOM: false
- wall: `0:16.20`
- max RSS: `877560 KB`

## Replay Result

The replay used Docker isolation, one grammar (`tlaplus`), one file (`Bakery.tla`), and the corpus mounted read-only at `/workspace/corpus_sources`.

- wrapper rc: `124`
- `/usr/bin/time -v` exit status: `124`
- timeout: yes
- OOM: false
- wall: `1:04.36`
- max RSS: `1776464 KB`
- deep classifier cap: `200000`
- final progress before timeout: token `170`, byte `411..412`, `max_stacks=174`, `live_stacks=6`

Latest full counter sample at elapsed `41106 ms`, token `5[410..411]`:

```text
pending_frontier_gss_attempt=684582
pending_frontier_gss_merge=62142
pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:0,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:622440}
pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
pending_frontier_repeat_tail_window_sig={attempt:684582,prefix_equal:684582,prefix_not_equal:0,window_equal:684582,window_incomplete:0,window_capped:0,candidate:684582,merge:62142,gss_reject:622440,non_equal:0,no_entries:0,samples:16}
pending_frontier_repeat_tail_window_sig_gss_reject_classify={accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,c_recovery_cost_differs:0,missing_head:0,dead_accepted_mismatch:0,score_mismatch:0,shifted_mismatch:0,top_state_byte_mismatch:0,clean_zero_error_target:0,clean_zero_error_fork:0,reach_b_can_reach_a:0,reach_prev_can_reach_n:0,payload_no_equivalent_link:0,max_link_count_no_replacement:0,replacement_not_better_precedence:0,nodes_can_merge_reach:0,nodes_can_merge_state:0,nodes_can_merge_clean:0,nodes_can_merge_uniform_byte_offset:0,recursive_can_merge_or_can_add_failure:622440,actual_merge_unknown:0,samples:16}
pending_frontier_repeat_tail_window_sig_gss_reject_deep={classified:200000,skipped_after_max:422440,can_merge_nodes_b_reaches_a:0,can_merge_nodes_seen_cycle:0,can_add_link_nil_node:0,can_add_link_prev_is_node:0,can_add_link_prev_reaches_node:0,can_add_link_no_payload_equivalent_link:0,can_add_link_nodes_can_merge_reject_reach:0,can_add_link_nodes_can_merge_reject_state:0,can_add_link_nodes_can_merge_reject_clean:0,can_add_link_nodes_can_merge_reject_uniform_byte_offset:0,can_add_link_nested_can_merge_failure:200000,max_link_count_no_replacement:0,replacement_not_better_precedence:0,replacement_nested_can_merge_failure:0,unknown:0,depth:{0:0,1:200000,2_4:0,5_plus:0},path_len:{1:0,2_4:200000,5_16:0,17_plus:0},head_links:{1:200000,2_4:0,5_plus:0},fail_links:{1:0,2_4:0,5_plus:200000}}
```

Final parse progress line before timeout:

```text
PARSE-PROGRESS phase=dispatch_begin lang=tlaplus source_bytes=22753 expected_eof=22753 elapsed_ms=41545 iter=331 tokens=170 stacks=6 live_stacks=6 max_stacks=174 node_count=157906 peak_depth=169 need_token=false single_iters=11 multi_iters=320 token_symbol=5 token_start=411 token_end=412 token_no_lookahead=false token_eof=false num_stacks=6
```

Representative samples:

```text
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT sample=1 reason=recursive_can_merge_or_can_add_failure key={state:4014 byte:68} token={symbol:6 start:67 end:68} branch={target:123 fork:129} head_links={target:1 fork:1} deep={reason:can_add_link_nested_can_merge_failure depth:1 path_len:4 fail_links:6}
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT sample=16 reason=recursive_can_merge_or_can_add_failure key={state:4014 byte:70} token={symbol:6 start:69 end:70} branch={target:243 fork:249} head_links={target:1 fork:1} deep={reason:can_add_link_nested_can_merge_failure depth:1 path_len:4 fail_links:6}
```

## Classification

The ordinary window-signature counters remain complete/equal for every attempted candidate observed in the run:

- `attempt=684582`
- `prefix_equal=684582`
- `window_equal=684582`
- `candidate=684582`
- `merge=62142`
- `gss_reject=622440`
- every cheap/preflight reject bucket remains `0`
- `recursive_can_merge_or_can_add_failure=622440`

The deep recursive classifier hit the configured cap early:

- classified: `200000`
- skipped after cap at latest full sample: `422440`
- `can_add_link_nested_can_merge_failure=200000`
- all other deep reasons: `0`
- depth bucket: `depth:{0:0,1:200000,2_4:0,5_plus:0}`
- path bucket: `path_len:{1:0,2_4:200000,5_16:0,17_plus:0}`
- head links: `head_links:{1:200000,2_4:0,5_plus:0}`
- failing links: `fail_links:{1:0,2_4:0,5_plus:200000}`

The sampled path shape is also uniform: the post-proof target/fork heads both have one link, then the recursive `canAddLink` step finds an equivalent-payload link whose previous node passes the cheap `nodesCanMerge` screen but fails the nested `canMergeNodes` call one level down. The failing node has at least five links in every classified case.

## Conclusion

This narrows the blocker one level past the previous report. The failure is not a top-level reach, payload, replacement, state, clean, or uniform-byte-offset issue. In the first 200,000 classified rejects, every recursive failure is:

`can_add_link_nested_can_merge_failure`

The concrete generalized next move is a negative cache key for recursive GSS preflight, focused below `canAddLink` when:

- target/fork heads have a single link,
- `stackEntryPayloadsEquivalentIgnoringDynamic(existingEntry, entry)` succeeds,
- `nodesCanMerge(existingPrev, prev)` succeeds,
- and nested `canMergeNodes(existingPrev, prev)` rejects.

The cache key should be over the recursive node pair plus the link payload signature used by `canAddLink`, not over TLA+ grammar state. A useful next trace-only pass would keep the same deep classifier and split `can_add_link_nested_can_merge_failure` by the nested `canMergeNodes` child reason without collapsing it back to the parent label, then validate whether the repeated negative key is `(existingPrev, prev)`-stable enough for a selectivity precheck or a memoized negative-result cache.
