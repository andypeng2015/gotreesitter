# TLA+ Pending Frontier GSS Merge Diagnostic

Date: 2026-06-24

## Context

After `docs/reports/tlaplus-frontier-fanout-20260624.md` showed that TLA+
fanout re-expands through `pendingFrontierForkStacks` after merge compression,
an isolate-only safe pending-frontier GSS merge was tried.

The experiment was part of the generalized parser machinery direction toward
206 grammar parity. It was not a grammar-specific workaround.

## Experiment Controls

The isolate run used an env-gated pending-frontier GSS merge:

- `GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1`

Fanout and merge telemetry were also enabled in the isolate. The code was
isolate-only and uncommitted.

## Safety Contract

The attempted merge path was intentionally conservative:

- Pending-frontier-only.
- No active stack merging.
- Reject accepted stacks and entries-backed stacks.
- Require the same top state and byte.
- Require the same score and shifted status.
- Preserve C recovery kind and cost compatibility.
- Require clean zero-error/missing GSS links.
- Reject distinct materializing shapes.
- Allocate `branchOrder` before merge and preserve the earliest `branchOrder`.

## Validation Before Run

- `go build .` passed.
- `bash -n cgo_harness/docker/run_parity_in_docker.sh` passed.
- A temporary focused harness passed and was then removed.
- Full package `go test` remained blocked by stale helper signatures in the
  isolate, so this diagnostic does not claim a package test pass.

## Artifacts

- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-gss-20260624c`
- `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/harness_out/docker/20260624T233027Z-tlaplus-pending-frontier-gss3`
- Readable log: `/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/tlaplus-bakery-pending-frontier-gss-20260624c/variants/forest_off/frame-0001.log`

## Result

The `forest_off` variant timed out at 90s.

Terminal parser progress was:

```text
dispatch_begin iter=1089 tokens=455 stacks=2 max_stacks=516
```

The pending-frontier GSS counters showed many attempts and no legal merges:

```text
pending_frontier_gss_attempt=19994601
pending_frontier_gss_merge=0
```

Repeated bursts still came from pending-frontier forks and concentrated at
`4014@byte`:

```text
iter=1080 stacks=516 origins={pending-frontier-fork:512,conflict-fork:2}
iter=1086 stacks=516 origins={pending-frontier-fork:512,conflict-fork:2}
```

## Conclusion

Safe materializing-shape-guarded pending-frontier GSS merge is not the fix. It
found zero legal merge opportunities.

This rules out the simple safe pre-append coalescing path. The next useful step
is reject-reason telemetry, or a proof that later reductions consume GSS
alternatives before final materialization so a broader C-style link-packing
strategy can be made correct.

Do not respond by tightening caps or adding grammar-specific checks.
