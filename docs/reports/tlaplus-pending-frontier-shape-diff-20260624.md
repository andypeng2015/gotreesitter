# TLA+ Pending-Frontier Shape Difference Sampler

Date: 2026-06-24

## Context

Prior TLA+ Bakery investigations showed that normal merge compression works, then `pendingFrontierForkStacks` re-expand; a conservative pending-frontier GSS merge made many attempts but zero merges; reject tracing then showed every observed reject was `distinct_materializing_shape`.

This experiment added isolate-only, env-gated sampling for that reject bucket. It does not relax `gssStacksHaveDistinctMaterializingShapes`; it only logs the first materializing-entry difference for bounded rejected target/fork pairs.

## Env Controls

- `GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1` enables bounded shape-diff sampling.
- `GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE_SAMPLES=N` optionally changes the sample cap; default is 16.

The sampler also adds a compact fanout extra:

```text
pending_frontier_shape_diffs={samples:16,length_mismatch:16,...}
```

## Validation

Isolate worktree:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

Commands:

```sh
go build .
bash -n cgo_harness/docker/run_parity_in_docker.sh
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate --label tlaplus-pending-frontier-shape-diff-rerun --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- "cd /workspace && env GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_STAGES=baseline,variants,summary GTS_WRINGER_BASELINE_FRAMES=base:Bakery.tla GTS_WRINGER_VARIANT_FRAMES=base:Bakery.tla GTS_WRINGER_VARIANTS=forest_off GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_KILL_AFTER=10s GTS_WRINGER_HEARTBEAT=15 GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_PARSE_PROGRESS=1 GOT_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_TOP_KEYS=5 GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_MAX_EVENTS=2000 GOT_GLR_MERGE_PRECAP_DOMINANCE=1 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 cgo_harness/docker/run_grammar_integrity_wringer.sh tlaplus cgo_harness/harness_out/tlaplus-bakery-pending-frontier-shape-diff-20260624-rerun"
```

Results:

- `go build .`: pass.
- `bash -n ...`: pass.
- Docker wrapper: `exit_code=0`, `oom_killed=false`.
- Bakery frame still times out as expected for this investigation: variant `forest_off` `rc=124`, `parser_max_stacks=516`, family `timeout_fanout_perf`.

Artifacts:

- Docker wrapper metadata: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260625T000320Z-tlaplus-pending-frontier-shape-diff-rerun/metadata.txt`
- Wringer output: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-shape-diff-20260624-rerun`
- Shape/fanout log: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-shape-diff-20260624-rerun/variants/forest_off/frame-0001.log`
- Summary: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-shape-diff-20260624-rerun/wringer_summary.json`

## Evidence

All 16 bounded shape-diff samples were first-difference `length_mismatch`. No sampled reject reached symbol, span, production, state, child-count, flags, raw-shape-ref, raw-shape-production, or hash-only buckets.

Representative samples:

```text
GLR-PENDING-FRONTIER-SHAPE-DIFF sample=1 bucket=length_mismatch index=9 target={len=10 entry[9]={sym=6/"block_comment_text_token2" span=62..63 prod=0 parse=4014 pregoto=3953 children=0 flags={named:false extra:false missing:false error:false} raw={ref:0 sym:0 prod:0 children:0}}} fork={len=9 entry[9]=<missing>} key={state:4014 byte:63} token={symbol:6 start:62 end:63} branch={target:6 fork:7}
GLR-PENDING-FRONTIER-SHAPE-DIFF sample=3 bucket=length_mismatch index=9 target={len=11 entry[9]={sym=613/"block_comment_text_repeat1" span=61..63 prod=0 parse=3953 pregoto=3953 children=0 flags={named:false extra:false missing:false error:false} raw={ref:1048594 sym:613 prod:0 children:2}}} fork={len=9 entry[9]=<missing>} key={state:4014 byte:64} token={symbol:6 start:63 end:64} branch={target:11 fork:13}
GLR-PENDING-FRONTIER-SHAPE-DIFF sample=16 bucket=length_mismatch index=11 target={len=13 entry[11]={sym=613/"block_comment_text_repeat1" span=63..65 prod=0 parse=3953 pregoto=3953 children=0 flags={named:false extra:false missing:false error:false} raw={ref:1048614 sym:613 prod:0 children:2}}} fork={len=11 entry[11]=<missing>} key={state:4014 byte:66} token={symbol:6 start:65 end:66} branch={target:27 fork:29}
```

Early aggregate:

```text
pending_frontier_gss_attempt=140 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:140,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
```

Late aggregate at timeout approach:

```text
pending_frontier_gss_attempt=13934015 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:13934015,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
```

The 512-fork bursts still occur around state `4014`:

```text
GLR-FANOUT ... token=5[1086..1088] stacks=516 ... appended=514 ... actions={reduce:512,shift:2} top_keys=[4014@1088 count=514 ...;3953@1086 count=2 ...]
GLR-FANOUT ... token=5[1088..1090] stacks=516 ... appended=514 ... actions={reduce:512,shift:2} top_keys=[4014@1090 count=514 ...;3953@1088 count=2 ...]
```

## Conclusion

For the sampled `distinct_materializing_shape` rejects, the first materializing-shape difference is length, not entry content. The longer side has one extra materializing entry at the mismatch index, commonly either `block_comment_text_token2` or `block_comment_text_repeat1`; the shorter side has no entry at that index.

This means the conservative pending-frontier GSS merge is blocked before ordinary descriptor mismatches matter. The next generalized experiment should target length-normalization/proof for block-comment repeat frontier shapes: decide whether these target/fork pairs are semantically equivalent despite different materializing sequence lengths, likely by comparing raw-shape-expanded repeat payloads or by proving a repeat-tail dominance relation before permitting any merge relaxation.
