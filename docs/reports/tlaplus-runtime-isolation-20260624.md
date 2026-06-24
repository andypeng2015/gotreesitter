# TLA+ Runtime Isolation Evidence - 2026-06-24

## Context

A clean telemetry branch completes the TLA+ Bakery frame as a truncation or
memory-budget nonmatch. The dirty main/runtime slice times out in
`merge_cull_begin`.

This report buckets the isolation runs that separate the dirty parser/runtime
core from the dirty forest overlay.

## Artifacts

| Bucket | Artifact |
| --- | --- |
| dirty main | `cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-bakery-current-dirty-20260624T203729Z` |
| clean telemetry | `/home/draco/work/gotreesitter-merge-telemetry/cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-merge-telemetry-bakery-20260624T214707Z` |
| runtime isolate | `/home/draco/work/gotreesitter-tlaplus-runtime-isolate/cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-runtime-isolate-20260624T215810Z` |
| glr/parser isolate | `/home/draco/work/gotreesitter-tlaplus-glr-parser-isolate/cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-glr-parser-isolate-20260624T221214Z` |
| noforest isolate | `/home/draco/work/gotreesitter-tlaplus-runtime-noforest-isolate/cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-runtime-noforest-isolate-20260624T222821Z` |

## Key Outcomes

- Dirty main baseline: timeout `rc=124` at `merge_cull_begin`, tokens about
  `551`, and stacks/live/max `1548`.
- Clean telemetry branch: build and run complete with `rc=0` for Bakery,
  nonmatching as truncation or memory-budget, with no terminal timeout.
- Full dirty runtime slice on clean telemetry: timeout `rc=124` at
  `merge_cull_begin`; this is sufficient to reproduce the TLA+ fanout timeout.
- GLR/parser compile-dependent subset: timeout `rc=124` with terminal
  `dispatch_begin`, but it shows the same early `state=4014` same-key overflow
  rows.
- Remaining runtime overlay: timeout at `forest_reduce_visit_begin`; the dirty
  forest is a separate overlay.
- Noforest isolate: replacing dirty `glr_forest.go` with the clean forest
  returns the timeout to `merge_cull_begin`; `forest_off` still times out in
  `merge_cull_begin`.

## Telemetry Pattern

Observed same-key pressure concentrates at `state=4014`, bytes `64..68`:

| seen | kept | cap | overflow | gss_attempt | equiv_test | cost_preserve |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 23 | 6 | 6 | 11 | 0 | 0 | 0 |
| 30 | 6 | 6 | 18 | 0 | 0 | 0 |
| 37 | 6 | 6 | 25 | 0 | 0 | 0 |
| 43 | 6 | 6 | 31 | 0 | 0 | 0 |
| 49 | 6 | 6 | 37 | 0 | 0 | 0 |

The repeated rows show overflow growing before any observed GSS attempt,
equivalence test, or cost-preservation event.

## Conclusion

The dirty parser/runtime core is sufficient for the TLA+ fanout timeout. The
dirty forest adds a separate forest throughput overlay, but it is not the
primary merge/cull cause.

The next machinery experiment should target generalized same-key pre-cap
dominance and survivor ordering for rank-equivalent candidates before cap
overflow, while preserving deep equivalence checks. This should be treated as a
runtime machinery experiment, not a grammar patch and not cap widening.
