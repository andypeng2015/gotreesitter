# TLA+ Pending-Frontier Repeat-Tail Signature Cache

Date: 2026-06-25

## Context

This report buckets the corrected durable diagnostic win from the TLA+ Bakery pending-frontier repeat-tail signature-cache experiment.

Isolate repo:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

Valid corrected artifact directory:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-sig-merge-valid-20260625
```

Valid files:

```text
direct-repeat-tail-sig-on.log
direct-repeat-tail-sig-on.status
direct-repeat-tail-sig-on.time
```

Docker metadata:

```text
/home/draco/work/gotreesitter/harness_out/docker/20260625T005103Z-tlaplus-repeat-tail-sig-merge-valid-direct/metadata.txt
```

Explicitly discard the earlier artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-sig-merge-20260625
```

That earlier artifact missed `REPRO_LANG=tlaplus` and only showed `PASS`, so it is not valid evidence for the TLA+ Bakery replay.

## Corrected Replay

The corrected direct replay included:

```text
REPRO_LANG=tlaplus
REPRO_FILE=/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla
REPRO_PROGRESS=1
REPRO_SIGNATURES=1
REPRO_N=1
REPRO_ROUNDS=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE=1
```

It also included pending-frontier GSS, reject, shape, repeat-tail trace envs, fanout trace, and merge telemetry.

Validity checks in the output prove the run exercised TLA+ Bakery rather than a default or empty replay:

```text
MEASURE-PROGRESS lang=tlaplus
PARSE-PROGRESS phase=parse_entry lang=tlaplus
GLR-FANOUT
```

## Result

The corrected replay still timed out:

```text
direct_rc=124
```

This was a 60s timeout plus kill-after, not an OOM. Docker metadata reported:

```text
oom_killed=false
memory=8g
exit_code=0
```

Resource and time data:

```text
Elapsed wall time: 1:04.76
User time: 47.55s
System time: 20.64s
Maximum resident set size: 1,486,284 KB
Exit status: 124
```

Late telemetry showed the signature cache operating, but not resolving the TLA+ Bakery pathology.

At `elapsed_ms=35480`, `iter=690`, token `5[930..931]`:

```text
stacks=259
pending_frontier_gss_attempt=9221249
pending_frontier_gss_merge=505
rejects distinct_materializing_shape=9220696 gss_merge_rejected=48
sig {cache_hit:18345634,cache_miss:96864,candidate:553,merge:505,gss_reject:48,non_equal:0,incomplete:9220696,capped:8597412,hash_collision:0,hash_only:0}
```

At `elapsed_ms=35663`, same token:

```text
stacks=516
pending_frontier_gss_attempt=9253889
pending_frontier_gss_merge=505
sig {cache_hit:18410658,cache_miss:97120,candidate:553,merge:505,gss_reject:48,non_equal:0,incomplete:9253336,capped:8630052,hash_collision:0,hash_only:0}
```

Near timeout at `elapsed_ms=35861`, `iter=693`, token `5[931..932]`:

```text
stacks=259
pending_frontier_gss_attempt=9289445
pending_frontier_gss_merge=505
rejects distinct_materializing_shape=9288892 gss_merge_rejected=48
sig {cache_hit:18481405,cache_miss:97485,candidate:553,merge:505,gss_reject:48,non_equal:0,incomplete:9288892,capped:8663158,hash_collision:0,hash_only:0}
```

Shape diff samples remained `16/16 length_mismatch`, and repeat-tail samples remained `flatten_equal`.

## What This Proves

The signature cache is mechanically working. The valid run produced millions of cache hits, about 97K misses, no hash collisions, and no `non_equal` among evaluated candidates.

It preserves the earlier proof that the sampled length-mismatch repeat-tail cases are C-visible equivalent after repeat expansion.

Compared with naive inline repeat-tail flattening, the signature-cache path is better on progress and RSS. It enables the same `505` pending-frontier GSS merges without repeatedly paying the full inline flattening cost on every comparison.

## What It Does Not Solve

This does not solve TLA+ Bakery. The corrected replay still timed out with `direct_rc=124` and still reached `max_stacks=516`.

It is also still worse than the gate-off baseline and is not shippable. Most comparisons remain `incomplete` or `capped`, which means pending-frontier comparison is still too late, too expensive, and too cap-sensitive.

## Next Experiments

The next generalized direction is to compute or cache canonical repeat-tail and materialization signatures earlier, preferably at raw-shape, GSS-link construction, or representation-sharing level.

Before broadening the machinery, classify the `incomplete` and `capped` populations. Narrow suffix signatures may be a better experiment than full-stack flattening, because the current full comparison remains dominated by capped or incomplete work.
