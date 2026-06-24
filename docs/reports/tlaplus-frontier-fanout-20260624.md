# TLA+ Frontier Fanout Telemetry

Date: 2026-06-24

## Context

The same-rank pre-cap dominance experiment was implemented only in
`/home/draco/work/gotreesitter-tlaplus-dominance-isolate`. It was not committed
and should not be treated as a production win.

That isolate reduced early merge-key pressure for `4014@64..68` from
`kept=6` with growing overflow to `kept=2` and `overflow=0`. TLA+ still timed
out at `merge_cull_begin`, with stack counts around `516/518`.

## Fanout Telemetry

Additional fanout telemetry was added only in the isolate and run with
dominance and merge telemetry enabled. The isolate-only controls were:

- `GOT_GLR_FANOUT_TRACE=1`
- `GOT_GLR_FANOUT_TRACE_TOP_KEYS`
- `GOT_GLR_FANOUT_TRACE_MIN_STACKS`
- `GOT_GLR_FANOUT_TRACE_INTERVAL_MS`

## Artifacts

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-fanout`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260624T230854Z-tlaplus-bakery-fanout`
- Readable key log: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-fanout/variants/forest_off/frame-0001.log`

## Key Evidence

Merge compression is visible immediately before the re-expansion:

```text
after_merge_cull iter=474 token=5[611..612] stacks=2 top_keys=[4014@612 count=2 ...]
```

The next fanout expands almost entirely through pending frontier forks:

```text
drain_pending_frontier_forks iter=475 token=5[612..613] stacks=235 ... origins={pending-frontier-fork:232,conflict-fork:1} actions={reduce:232,shift:1} top_keys=[4014@613 count=234 ...;3953@612 count=1 ...]
```

The first burst at or above 500 stacks shows the same shape:

```text
drain_pending_frontier_forks iter=696 token=5[932..934] stacks=516 ... origins={pending-frontier-fork:512,conflict-fork:2} actions={reduce:512,shift:2} top_keys=[4014@934 count=514 ...;3953@932 count=2 ...]
```

## Conclusion

The TLA+ fanout is not solved by same-key survivor dominance. Merge compression
works; the pathological re-expansion source is pending frontier reduce fanout
from `completeConflictReduceFrontier` / `pendingFrontierForkStacks`.

That fanout concentrates back into `4014@byte` with tiny `3953@byte` unshifted
companions.

## Next Directed Experiment

Instrument or redesign pending frontier reduce fanout with GSS/link sharing or
same-frontier coalescing before appending stacks. Preserve C semantics and do
not introduce grammar checks.

Do not pursue tighter per-key caps as the next fix; the evidence points to
frontier reduce fanout before stack append, not survivor selection inside the
same merge key.
