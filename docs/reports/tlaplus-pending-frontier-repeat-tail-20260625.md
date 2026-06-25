# TLA+ Pending-Frontier Repeat-Tail Sampler

Date: 2026-06-25

## Question

The prior pending-frontier shape-diff sampler showed that GSS merge attempts still had zero merges because every sampled reject was `distinct_materializing_shape`, and every sampled first difference was `length_mismatch`.

The open question for this slice was whether the longer side's extra `block_comment_text_token2` or `block_comment_text_repeat1` entry represents a real C-visible frontier difference, or only a different repeat-tail encoding that becomes equal after expanding repeat helper entries.

## Method

In isolate worktree only:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

Added an env-gated sampler:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE_SAMPLES=N
```

The sampler only observes pending-frontier `distinct_materializing_shape` rejects. For `length_mismatch` samples whose longer-side extra entry is `block_comment_text_token2`, `block_comment_text_repeat1`, or a generated repeat helper, it flattens generated-repeat entries through raw-shape children with tight caps and compares the flattened C-visible descriptor streams.

No merge guard was relaxed and parser behavior was not changed.

## Validation

Commands:

```sh
go build .
bash -n cgo_harness/docker/run_parity_in_docker.sh
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate --label tlaplus-pending-frontier-repeat-tail --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- "cd /workspace && env GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_STAGES=baseline,variants,summary GTS_WRINGER_BASELINE_FRAMES=base:Bakery.tla GTS_WRINGER_VARIANT_FRAMES=base:Bakery.tla GTS_WRINGER_VARIANTS=forest_off GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_KILL_AFTER=10s GTS_WRINGER_HEARTBEAT=15 GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_PARSE_PROGRESS=1 GOT_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_TOP_KEYS=5 GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_MAX_EVENTS=2000 GOT_GLR_MERGE_PRECAP_DOMINANCE=1 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 GOT_GLR_PENDING_FRONTIER_SHAPE_DIFF_TRACE=1 GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_TRACE=1 cgo_harness/docker/run_grammar_integrity_wringer.sh tlaplus cgo_harness/harness_out/tlaplus-bakery-pending-frontier-repeat-tail-20260625"
```

Results:

- `go build .`: pass.
- `bash -n ...`: pass.
- Docker wringer: `exit_code=0`, `oom_killed=false`.
- Bakery baseline and `forest_off` frames still timed out as expected.

The wringer did not propagate the diagnostic `GOT_GLR_*` envs into the replayed `measure.test` command, so the same built Docker `measure.test` binary was replayed directly with those envs attached to the parser process. That direct replay timed out with `exit_code=124`, `oom_killed=false`, and produced the repeat-tail samples.

## Artifacts

- Wringer output: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-repeat-tail-20260625`
- Wringer wrapper metadata: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260625T001716Z-tlaplus-pending-frontier-repeat-tail/metadata.txt`
- Direct diagnostic log: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-repeat-tail-20260625/direct-repeat-tail.log`
- Direct wrapper metadata: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260625T002036Z-tlaplus-pending-frontier-repeat-tail-direct/metadata.txt`

## Evidence

The direct diagnostic replay emitted 16 bounded repeat-tail samples:

```text
16 flatten_equal
```

Representative token-tail sample:

```text
GLR-PENDING-FRONTIER-REPEAT-TAIL sample=1 bucket=flatten_equal equivalent_flattened=true key={state:4014 byte:63} token={symbol:6 start:62 end:63} target_extra=target[9]={sym=6/"block_comment_text_token2" span=62..63 prod=0 parse=4014 pregoto=3953 children=0 flags={named:false extra:false missing:false error:false} raw={ref:0 sym:0 prod:0 children:0}} first_flat_diff=none
```

Representative repeat-helper sample:

```text
GLR-PENDING-FRONTIER-REPEAT-TAIL sample=3 bucket=flatten_equal equivalent_flattened=true key={state:4014 byte:64} token={symbol:6 start:63 end:64} target_extra=target[9]={sym=613/"block_comment_text_repeat1" span=61..63 prod=0 parse=3953 pregoto=3953 children=0 flags={named:false extra:false missing:false error:false} raw={ref:1048594 sym:613 prod:0 children:2}} first_flat_diff=none
```

The old shape-diff sampler still saw only length mismatches:

```text
pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
```

Late aggregate near timeout:

```text
pending_frontier_gss_attempt=14999886 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:14999886,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
```

Hot states remained the same:

```text
top_keys=[4014@1164 count=269 ...;3953@1163 count=1 ...]
```

## Conclusion

This buckets the sampled TLA+ pending-frontier length mismatch as repeat-tail equivalent after flattening. The longer frontier's extra `block_comment_text_token2` or `block_comment_text_repeat1` does not create a C-visible descriptor difference in these samples; after flattening, target and fork streams are equal.

Next generalized machinery should not be a per-grammar parser behavior change. The next proof/implementation slice should define a guarded repeat-tail-normalized frontier equivalence relation, probably using raw-shape expansion with caps, and only then consider whether pending-frontier GSS merge can safely use that relation.
