# TLA+ Pending-Frontier GSS Reject Telemetry

Date: 2026-06-24

## Context

This follows the TLA+ pending-frontier fanout diagnostics:

- `docs/reports/tlaplus-frontier-fanout-20260624.md` showed merge compression followed by re-expansion through `pendingFrontierForkStacks`.
- `docs/reports/tlaplus-pending-frontier-gss-merge-20260624.md` showed a materializing-shape-guarded pending-frontier GSS merge produced many attempts and zero merges.

The isolate experiment added reject-reason counters to the existing pending-frontier GSS attempt path. It did not relax the merge guard or change parser behavior: the diagnostic path still requires C recovery kind equality, non-distinct materializing shape, no accepted/nil stacks, no `entries`, present GSS heads, equal top state and byte offset, and a successful `tryGSSMainMergeForParser`.

## Isolate Changes

Worktree: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate`

Changed isolate files:

- `parser.go`: added `pendingFrontierGSSRejectCounters`, reset per parse, and fanout trace formatting.
- `parser_config.go`: added `GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE`.
- `parser_reduce.go`: counted pending-frontier GSS reject reasons in the guarded merge attempt path.

The fanout trace now emits a compact field when reject tracing is enabled:

```text
pending_frontier_gss_rejects={c_recovery_kind_mismatch:...,distinct_materializing_shape:...,accepted_or_nil:...,entries_or_missing_gss:...,state_byte_mismatch:...,gss_merge_rejected:...}
```

## Validation

Isolate validation:

```text
cd /home/draco/work/gotreesitter-tlaplus-dominance-isolate
go build .
bash -n cgo_harness/docker/run_parity_in_docker.sh
```

Both commands exited `0`.

Docker wringer command:

```text
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate --label tlaplus-pending-frontier-rejects --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- "cd /workspace && env GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_STAGES=baseline,variants,summary GTS_WRINGER_BASELINE_FRAMES=base:Bakery.tla GTS_WRINGER_VARIANT_FRAMES=base:Bakery.tla GTS_WRINGER_VARIANTS=forest_off GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_KILL_AFTER=10s GTS_WRINGER_HEARTBEAT=15 GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_PARSE_PROGRESS=1 GOT_PARSE_PROGRESS_INTERVAL_MS=1000 GOT_GLR_FANOUT_TRACE=1 GOT_GLR_FANOUT_TRACE_TOP_KEYS=5 GOT_GLR_FANOUT_TRACE_MIN_STACKS=2 GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1 GOT_GLR_MERGE_TELEMETRY=1 GOT_GLR_MERGE_TELEMETRY_MAX_EVENTS=2000 GOT_GLR_MERGE_PRECAP_DOMINANCE=1 GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 GOT_GLR_PENDING_FRONTIER_GSS_REJECT_TRACE=1 cgo_harness/docker/run_grammar_integrity_wringer.sh tlaplus cgo_harness/harness_out/tlaplus-bakery-pending-frontier-rejects-20260624"
```

Wrapper result:

- Exit code: `0`
- OOM killed: `false`
- Host artifact path: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260624T234757Z-tlaplus-pending-frontier-rejects`
- Wringer output path: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-rejects-20260624`
- Readable trace log: `cgo_harness/harness_out/tlaplus-bakery-pending-frontier-rejects-20260624/variants/forest_off/frame-0001.log`

The baseline frame timed out at 60 seconds, as expected for this diagnostic scope. Its frame log was written `0600` as root inside the Docker output. The variant trace log is readable and contains the reject counts below.

## Evidence

Early reject telemetry shows attempts immediately dominated by distinct materializing shape:

```text
line 91: stacks=9 pending_frontier_gss_attempt=5 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:5,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} ... top_keys=[4014@64 count=9 ... origins={pending-frontier-fork:5,conflict-fork:2}]
```

Representative late 512-fork bursts:

```text
line 6175: stacks=516 snapshot=260 pending_frontier_gss_attempt=12621870 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:12621870,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} appended=514 ... origins={pending-frontier-fork:512,conflict-fork:2} ... top_keys=[4014@1011 count=514 ...]

line 6224: stacks=516 snapshot=260 pending_frontier_gss_attempt=12773035 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:12773035,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} appended=514 ... origins={pending-frontier-fork:512,conflict-fork:2} ... top_keys=[4014@1014 count=514 ...]

line 6248: stacks=516 snapshot=260 pending_frontier_gss_attempt=12848924 pending_frontier_gss_merge=0 pending_frontier_gss_rejects={c_recovery_kind_mismatch:0,distinct_materializing_shape:12848924,accepted_or_nil:0,entries_or_missing_gss:0,state_byte_mismatch:0,gss_merge_rejected:0} appended=514 ... origins={pending-frontier-fork:512,conflict-fork:2} ... top_keys=[4014@1015 count=514 ...]
```

Negative search:

```text
rg "gss_merge_rejected:[1-9]|state_byte_mismatch:[1-9]|entries_or_missing_gss:[1-9]|accepted_or_nil:[1-9]|c_recovery_kind_mismatch:[1-9]" cgo_harness/harness_out/tlaplus-bakery-pending-frontier-rejects-20260624/variants/forest_off/frame-0001.log
```

Exit code was `1`, with no matches.

## Conclusion

The dominant and observed-only pending-frontier GSS reject reason is `distinct_materializing_shape`. No candidate reaches `gss_merge_rejected`, and no attempts are blocked by C recovery kind, nil/accepted stacks, entries/missing GSS, or state/byte mismatch in this frame.

The burst shape remains the same class as prior evidence: pending-frontier fanout repeatedly rebuilds around state `4014`, including 512 pending-frontier forks in a `stacks=516` burst (`512` pending-frontier forks plus `2` conflict forks, producing `4014@1011`, `4014@1014`, and `4014@1015` top-key bursts in this shortened run).

## Next Experiment

The next useful diagnostic is shape-difference attribution for the `distinct_materializing_shape` bucket: sample bounded materializing-shape fingerprints for a small number of rejected pairs at the 512-fork burst, grouped by first differing production/symbol/path depth. That should identify whether the safe generalized merge needs a stronger equivalence proof, a narrower shape-preserving compression point, or an earlier frontier representation change.
