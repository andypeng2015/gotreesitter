# TLA+ Pending-Frontier Repeat-Tail Equivalence Merge

Date: 2026-06-25

## Question

Can repeat-tail-normalized pending-frontier equivalence safely enable real GSS merges in the sampled TLA+ Bakery timeout case, and is the naive inline merge path a viable performance fix?

This was tested as generalized parser machinery, not as a TLA+ grammar patch. The isolate changed only:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_config.go
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser.go
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/parser_reduce.go
```

## Method

The isolate replayed the Bakery TLA+ case with the guarded repeat-tail-normalized pending-frontier merge path enabled:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE=1
```

The direct replay also attached progress, fanout, merge, reject, shape, and repeat-tail diagnostic envs so parser-level telemetry reached the measured process.

The diagnostic run was executed directly against the built `measure.test` binary because the wringer/tier-scan propagation preserves `GOT_PARSE_PROGRESS*` but not arbitrary `GOT_GLR_*` diagnostics. That harness debt means parser diagnostic envs currently require direct replay for this slice.

## Validation

Build status:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625/build-measure.status
build_rc=0
```

Direct ON replay status:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625/direct-repeat-tail-equiv-on.status
direct_rc=124
```

Docker wrapper metadata:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260625T003118Z-tlaplus-repeat-tail-equiv-on-direct/metadata.txt
label=tlaplus-repeat-tail-equiv-on-direct
memory=8g
exit_code=0
oom_killed=false
```

The direct replay still timed out with exit status 124. It was not OOM-killed. The time log reported:

```text
Maximum resident set size (kbytes): 1574664
Exit status: 124
```

## Artifacts

- Artifact directory: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625`
- Direct ON log: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625/direct-repeat-tail-equiv-on.log`
- Direct ON status: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625/direct-repeat-tail-equiv-on.status`
- Build status: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-repeat-tail-equiv-merge-20260625/build-measure.status`
- Docker metadata: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260625T003118Z-tlaplus-repeat-tail-equiv-on-direct/metadata.txt`
- Prior OFF comparison log: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-repeat-tail-20260625/direct-repeat-tail.log`

## Evidence

The ON telemetry proves the guarded bypass fired. A late representative line around token 423..424 reported:

```text
pending_frontier_repeat_tail_equiv_candidate=553
pending_frontier_repeat_tail_equiv_merge=505
pending_frontier_repeat_tail_equiv_gss_reject=48
pending_frontier_gss_merge=505
```

The timeout tail around token 430..431 still showed the parser stuck in the pending-frontier scan, but with real guarded merges:

```text
pending_frontier_gss_attempt=1804866
pending_frontier_gss_merge=505
pending_frontier_repeat_tail_equiv_candidate=553
pending_frontier_repeat_tail_equiv_merge=505
pending_frontier_repeat_tail_equiv_gss_reject=48
pending_frontier_repeat_tail_equiv_non_equal=1804313
stacks=180
max_stacks=355
tokens=187
token_start=430
elapsed_ms=53846
```

The prior OFF comparison had no pending-frontier GSS merges at a comparable late point:

```text
pending_frontier_gss_attempt=14999886
pending_frontier_gss_merge=0
max_stacks=516
tokens=400
token_start=1164
elapsed_ms=53913
```

Both OFF and ON retained the same sampled shape-diff signature:

```text
pending_frontier_shape_diffs={samples:16,length_mismatch:16,symbol_mismatch:0,span_mismatch:0,production_mismatch:0,parse_state_mismatch:0,pre_goto_state_mismatch:0,child_count_mismatch:0,flags_mismatch:0,raw_shape_ref_mismatch:0,raw_shape_production_mismatch:0,hash_only_or_unknown:0}
```

Both runs also retained 16 `flatten_equal` repeat-tail samples, confirming that the sampled length mismatches are repeat-tail-normalized equivalents rather than C-visible descriptor differences.

## Conclusion

Repeat-tail-normalized equivalence is a valid merge-enabling relation for the sampled TLA+ case: under the guard, it produced real GSS merges (`505`) where the prior OFF run produced none.

It is not yet a viable performance fix in the naive inline form. The ON run reduced frontier peak from `516` to about `355`, but it slowed token progress badly: about 187 tokens / byte 430 at ~53.8s versus about 400 tokens / byte 1164 at ~53.9s in the OFF comparison. The direct ON replay still timed out with `direct_rc=124`, no OOM, and max RSS around 1.5 GB.

The likely cost is the inline flatten-and-scan work inside the O(N^2) pending-frontier merge path. The large ON `pending_frontier_repeat_tail_equiv_non_equal=1804313` volume also needs classification before this relation is broadened.

## Next Actions

Move repeat-tail normalization out of the hot pending-frontier scan path before attempting to keep the optimization. The next generalized parser-machine slice should likely use canonical flattened shape/signature caching, or earlier GSS/frontier representation sharing, so equivalence checks are mostly keyed lookups rather than repeated flattening.

Classify the huge `non_equal` population before broadening the guard. Separate true C-visible differences from cacheable repeat-tail misses, cap fallout, and cases that should be handled by earlier sharing.

Fix the harness propagation debt so wringer/tier-scan can pass arbitrary `GOT_GLR_*` diagnostics through to parser processes, instead of requiring direct replay for these experiments.
