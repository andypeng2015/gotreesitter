# TLA+ Merge-Cull Diagnostic, 2026-06-24

## Scope

This note records the validated TLA+ merge-cull diagnostic and merge-cap sweep for the representative Bakery frame. It is a diagnostic report only; no parser or grammar changes are included.

Representative frame:

- Path: `/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla`
- Size: `22753`
- SHA-256: `3b2a3f370e5f0fde8660b3ef09932d579c9bd2ccd249a5ce2304007bf24da8d5`

Artifacts:

- Main wringer: `cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-bakery-merge-cull-20260624T200522Z`
- Cap sweep: `cgo_harness/harness_out/grammar_integrity_wringer/tlaplus-bakery-merge-cap-sweep-20260624T201823Z`

## Main Wringer Result

Baseline replay times out in `go_parse_start`, with the parser phase at `merge_cull_begin`. At the timeout point, live/max stacks are `1548/1548`, `node_count` is `1643268`, and the replay is around token byte `2027..2028`.

Variant outcomes:

| Variant | Outcome | Live/Max Stacks | Node Count | Notes |
| --- | --- | ---: | ---: | --- |
| `forest_off` | Timeout in `merge_cull_begin` | `1548/1548` | `513296` | Forest is not the primary cause. |
| `stack2` | Timeout in `merge_cull_begin` | `1548/1548` | `1719849` | Stack cap alone does not clear the stall. |
| `stack8` | Timeout in `merge_cull_begin` | `780/1548` | `1729759` | Fewer live stacks at timeout, still stalls. |
| `stack48` | Timeout in `merge_cull_begin` | `780/1548` | `1704594` | Same stall shape as `stack8`. |
| `merge24` | Timeout in `merge_cull_begin` | `1855/2064` | `1098683` | Larger merge cap still stalls. |
| `merge1` | Completes with `rc=0`, but diverges badly | n/a | n/a | Go root is `ERROR[0:4758]` with 3 errors; C root is `source_file[0:22753]` with 0 errors. Stop reason is `no_stacks_alive`; stop action is `reduce-chain-cycle`; `cRecoveryGateReason` is `external_scanner_requires_precise_externallexstates`. |

The `merge1` result demonstrates that aggressive culling can force completion by over-pruning viable state. It loses parity against C and is not a correctness-preserving fix.

## Merge-Cap Sweep

The cap sweep summary is from `summary.tsv` under the cap-sweep artifact directory.

| Cap | RC | Phase | Token Bytes | Tokens | Iteration | Live/Max Stacks | Node Count |
| ---: | ---: | --- | --- | ---: | ---: | ---: | ---: |
| 2 | 124 | `merge_cull_begin` | `2445..2446` | 883 | 2836 | `518/518` | `1529817` |
| 4 | 124 | `merge_cull_begin` | `1228..1241` | 402 | 932 | `413/1032` | `601649` |
| 8 | 124 | `merge_cull_begin` | `1564..1618` | 510 | 1255 | `2064/2064` | `1541994` |
| 12 | 124 | `merge_cull_begin` | `1082..1084` | 379 | 862 | `2217/3096` | `1247829` |
| 16 | 124 | `merge_cull_begin` | `631..633` | 258 | 508 | `1993/1993` | `498423` |

All tested caps greater than or equal to 2 still time out in `merge_cull_begin`. Lowering the cap changes where and how the stall appears, but does not remove the underlying survivor-quality problem.

## Conclusion

Cap-only is not a viable generalized fix. `cap=1` over-prunes and loses C parity, while caps `>=2` still stall in merge-cull. The next generalized target should be merge/cull survivor quality and dominance telemetry/rules. This should not be handled as a per-grammar patch, and the current evidence does not justify a wider cap sweep.
