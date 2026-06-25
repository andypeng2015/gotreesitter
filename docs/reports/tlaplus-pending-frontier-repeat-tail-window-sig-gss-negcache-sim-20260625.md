# TLA+ Window-Sig GSS Reject Negative-Cache Simulation

Date: 2026-06-25

## Scope

This was a trace-only proof in the TLA+ dominance isolate:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate`

No production parser behavior was changed in the main repo. The isolate added prospective negative-cache accounting at the nested recursive GSS rejection point classified as `can_add_link_nested_can_merge_failure`.

## Isolate-Only Code Changes

Touched isolate files:

- `glr.go`
- `parser.go`
- `parser_config.go`
- `parser_reduce.go`

The simulation is gated by:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NEGCACHE_SIM=1
```

Optional knobs:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NEGCACHE_SIM_MAX
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NEGCACHE_SIM_SAMPLES
```

The hook is trace-only. It records counters before returning the existing deep-classifier rejection and does not affect merge acceptance, GSS mutation, or parser output.

## Keys Simulated

Conservative key:

```text
(existingPrev pointer, prev pointer, current node pointer,
 existingEntry payload shallow signature, entry payload shallow signature,
 preflight virtual-link count, preflight virtual-link digest,
 depth, pathLen)
```

Diagnostic relaxed key:

```text
(existingPrev pointer, prev pointer)
```

Both maps store the first rejection reason seen for each key and count conflicts if a later lookup reports a different reason.

## Command

Artifact root:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-negcache-sim-20260625
```

Docker wrapper artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-negcache-sim-20260625/20260625T023034Z-negcache-sim
```

Command shape:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-negcache-sim-20260625 \
  --label negcache-sim \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "<custom build + one-file Bakery replay>"
```

The inner replay built `cgo_harness/measure.test` in Docker, then ran:

```sh
/usr/bin/time -v timeout --kill-after=10s 90s env \
  CGO_ENABLED=1 \
  REPRO_LANG=tlaplus \
  REPRO_DIR=/workspace/corpus_sources \
  REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla \
  REPRO_PROGRESS=1 \
  REPRO_SIGNATURES=1 \
  REPRO_N=1 \
  REPRO_ROUNDS=1 \
  GOT_PARSE_PROGRESS=1 \
  GOT_PARSE_PROGRESS_INTERVAL_MS=1000 \
  GOT_GLR_FANOUT_TRACE=1 \
  GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1000 \
  GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NEGCACHE_SIM=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NEGCACHE_SIM_SAMPLES=16 \
  /workspace/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-negcache-sim-20260625/measure.test \
  -test.run '^TestMeasureDtierVsC$' \
  -test.count=1
```

Not set:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE
```

## Runtime Result

Build:

```text
build_rc=0
```

Replay:

```text
run_rc=124
timeout=true
oom_killed=false
wall=1:37.57
max_rss_kb=1640516
```

Latest progress before timeout:

```text
iter=383
tokens=196
stacks=6
live_stacks=6
max_stacks=198
node_count=212610
peak_depth=195
token=5[457..458]
```

## Counters

Last durable fanout line:

```text
pending_frontier_repeat_tail_window_sig={
  attempt:1092530,
  prefix_equal:1092530,
  prefix_not_equal:0,
  window_equal:1092530,
  window_incomplete:0,
  window_capped:0,
  candidate:1092530,
  merge:85250,
  gss_reject:1007280,
  non_equal:0,
  no_entries:0,
  samples:0
}
```

Deep classifier:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_classify={
  recursive_can_merge_or_can_add_failure:1007280,
  actual_merge_unknown:0,
  samples:16
}

pending_frontier_repeat_tail_window_sig_gss_reject_deep={
  classified:200000,
  skipped_after_max:807280,
  can_add_link_nested_can_merge_failure:200000,
  depth:{0:0,1:200000,2_4:0,5_plus:0},
  path_len:{1:0,2_4:200000,5_16:0,17_plus:0},
  head_links:{1:200000,2_4:0,5_plus:0},
  fail_links:{1:0,2_4:0,5_plus:200000}
}
```

Negative-cache simulation:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_negcache={
  safe_lookup:200000,
  safe_hit:0,
  safe_miss:200000,
  safe_store:200000,
  safe_conflict:0,
  safe_unique:200000,
  relaxed_lookup:200000,
  relaxed_hit:0,
  relaxed_miss:200000,
  relaxed_store:200000,
  relaxed_conflict:0,
  relaxed_unique:200000,
  skipped_after_max:0,
  samples:16
}
```

Hit rates:

```text
safe_hit_rate = 0 / 200000 = 0.00%
relaxed_hit_rate = 0 / 200000 = 0.00%
```

## Samples

The first 16 negcache samples all had:

```text
reason=can_add_link_no_payload_equivalent_link
virtual_count=0
virtual_digest=0x0
depth=0
path_len=2
```

The paired public deep classifier samples reported the outer classification as:

```text
deep={reason:can_add_link_nested_can_merge_failure depth:1 path_len:4 fail_links:6}
```

Representative sample:

```text
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT-NEGCACHE sample=1 reason=can_add_link_no_payload_equivalent_link safe={existing_prev:0xc03ce85700 prev:0xc03ce85a00 current:0xc03ce85740 existing_sig:0x4a3d1358e261420b entry_sig:0x4a3d1358e261420b virtual_count:0 virtual_digest:0x0 depth:0 path_len:2} relaxed={existing_prev:0xc03ce85700 prev:0xc03ce85a00}
```

## Conclusion

This proof does not support an actual pointer-keyed negative cache for the current recursive reject path. Even the relaxed `(existingPrev, prev)` diagnostic key had zero recurrence across the full 200,000 classified nested failures.

The next generalized machinery target should not be behavior-changing negative caching on these keys. The better next move is a different selectivity precheck before deep recursive classification, or a normalized structural key that intentionally abstracts away per-attempt pointer churn. Any behavior-changing cache should first prove a high hit rate with zero conflicts under that safer normalized key.

