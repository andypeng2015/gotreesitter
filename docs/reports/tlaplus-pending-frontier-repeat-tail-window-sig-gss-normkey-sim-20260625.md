# TLA+ Window-Sig GSS Reject Normalized-Key Simulation

Date: 2026-06-25

## Scope

This was a trace-only proof in the TLA+ dominance isolate:

`/home/draco/work/gotreesitter-tlaplus-dominance-isolate`

No production parser behavior was changed in the main repo. The isolate added normalized structural-key accounting at the nested `canAddLink` preflight point where an equivalent payload link exists, `nodesCanMerge(existingPrev, prev)` passes, and the recursive `canMergeNodes(existingPrev, prev)` result is observable.

The hook records the observed nested result only. It does not change merge acceptance, GSS mutation, parser output, or the actual preflight result.

## Isolate-Only Code Changes

Touched isolate files:

- `glr.go`
- `parser.go`
- `parser_config.go`
- `parser_reduce.go`

The new simulation is gated by:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM=1
```

Optional knobs:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM_MAX
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM_SAMPLES
```

Counters are emitted in the parser trace as:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_normkey={...}
```

## Keys Simulated

`strict_struct`:

Pointer-free, spanful structural hashes for `existingPrev`, `prev`, `current`, the existing link payload, the incoming payload, current-node link count, depth, and path length. Stack-entry/node signatures include symbol, state, start/end byte span, parse state, pre-goto state, production, child count, field count, dynamic precedence, and flags.

`spanless_struct`:

Same as `strict_struct`, but omits absolute start/end bytes from stack-entry/node signatures.

`shape_bucket`:

Coarse diagnostic key using node entry state/symbol, link-count buckets, payload symbol/production/child-count/field-count/flags, depth, and path length. This is not behavior-safe by itself unless broader validation keeps conflicts at zero.

## Command

Artifact root:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-normkey-sim-20260625
```

Docker wrapper artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-normkey-sim-20260625/20260625T024453Z-normkey-sim
```

Command shape:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-normkey-sim-20260625 \
  --label normkey-sim \
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
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM_SAMPLES=16 \
  /workspace/cgo_harness/harness_out/tlaplus-bakery-window-sig-iterative-gss-normkey-sim-20260625/measure.test \
  -test.run '^TestMeasureDtierVsC$' \
  -test.count=1
```

Not set:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE
```

## Runtime Result

Local harness compile before Docker:

```text
cd cgo_harness && go test . -tags treesitter_c_parity -run '^$' -count=1
ok github.com/odvcencio/gotreesitter/cgo_harness 7.764s [no tests to run]
```

Root package test compile was not a useful gate in this dirty isolate because existing root tests are out of sync with pre-existing isolate API changes:

```text
go test . -run '^$' -count=1
FAIL github.com/odvcencio/gotreesitter [build failed]
first error: ./glr_test.go:2307:93: too many arguments in call to newNoTreeReduceNodeInArena
```

Docker replay:

```text
build_rc=0
run_rc=124
timeout=true
oom_killed=false
wall=1:38.00
max_rss_kb=1576060
```

Latest progress before timeout:

```text
iter=397
tokens=203
stacks=6
live_stacks=6
max_stacks=204
node_count=228724
peak_depth=202
token=5[464..465]
```

## Counters

Last durable fanout line:

```text
pending_frontier_repeat_tail_window_sig={
  attempt:1201021,
  prefix_equal:1201021,
  prefix_not_equal:0,
  window_equal:1201021,
  window_incomplete:0,
  window_capped:0,
  candidate:1201021,
  merge:90880,
  gss_reject:1110141,
  non_equal:0,
  no_entries:0,
  samples:0
}
```

Deep classifier:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_classify={
  recursive_can_merge_or_can_add_failure:1110141,
  actual_merge_unknown:0,
  samples:16
}

pending_frontier_repeat_tail_window_sig_gss_reject_deep={
  classified:200000,
  skipped_after_max:910141,
  can_add_link_nested_can_merge_failure:200000,
  depth:{0:0,1:200000,2_4:0,5_plus:0},
  path_len:{1:0,2_4:200000,5_16:0,17_plus:0},
  head_links:{1:200000,2_4:0,5_plus:0},
  fail_links:{1:0,2_4:0,5_plus:200000}
}
```

Normalized-key simulation:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_normkey={
  strict_lookup:200000,
  strict_hit:131994,
  strict_true_hit:0,
  strict_false_hit:131994,
  strict_miss:68006,
  strict_store:68006,
  strict_conflict:0,
  strict_unique:68006,
  strict_skipped_after_max:0,
  spanless_lookup:200000,
  spanless_hit:199996,
  spanless_true_hit:0,
  spanless_false_hit:199996,
  spanless_miss:4,
  spanless_store:4,
  spanless_conflict:0,
  spanless_unique:4,
  spanless_skipped_after_max:0,
  shape_lookup:200000,
  shape_hit:199998,
  shape_true_hit:0,
  shape_false_hit:199998,
  shape_miss:2,
  shape_store:2,
  shape_conflict:0,
  shape_unique:2,
  shape_skipped_after_max:0,
  samples:16
}
```

Hit and conflict rates:

```text
strict_hit_rate   = 131994 / 200000 = 65.9970%
spanless_hit_rate = 199996 / 200000 = 99.9980%
shape_hit_rate    = 199998 / 200000 = 99.9990%

strict_conflict_rate   = 0 / 200000 = 0.0000%
spanless_conflict_rate = 0 / 200000 = 0.0000%
shape_conflict_rate    = 0 / 200000 = 0.0000%
```

All observed nested `canMergeNodes(existingPrev, prev)` outcomes were `false`:

```text
true_hits=0
false_hits=331988 across hit buckets
```

## Samples

The first 16 normkey samples all had:

```text
result=false
current_links=1
depth=0
path_len=2
current={state:4014 symbol:6}
existing_prev={state:3953 symbol:613 links:6}
prev={state:3953 symbol:613 links:1}
payload={symbol:6 production:0 child_count:0 flags:0}
```

Representative sample:

```text
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT-NORMKEY sample=1 result=false strict=0xd2f9290cee513204 spanless=0x345c7b3dc90fb9f1 shape=0x37ff4c1342b9d58 current_links=1 depth=0 path_len=2 current={state:4014 symbol:6} existing_prev={state:3953 symbol:613 links:6} prev={state:3953 symbol:613 links:1} payload={symbol:6 production:0 child_count:0 flags:0}
```

## Conclusion

Pointer-key negative caching was not useful, but normalized structural keys have strong signal on this TLA+ path. The spanless structural key collapsed 200,000 nested observations to 4 unique keys with 99.998% hits and zero conflicts. The intentionally coarse shape key collapsed them to 2 unique keys with 99.999% hits and zero conflicts.

This is not yet enough to make `shape_bucket` behavior by itself, because the run only observed negative nested outcomes on one TLA+ file. It is enough to stop treating this as a pointer-identity problem.

Next generalized machinery target: implement a trace-guarded spanless normalized-key negative cache or selectivity precheck for the nested GSS preflight path, then validate it across more grammars/files with conflict auditing enabled. The `shape_bucket` result is useful as a broader selectivity precheck candidate, but should remain diagnostic until multi-grammar conflict evidence stays at zero.
