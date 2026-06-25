# Spanless GSS Negcache Key-Refinement Audit

Date: 2026-06-25

## Purpose

This audit tested generalized key refinements for the Java conflict reported in `spanless-gss-negcache-behavior-experiment-20260625.md`.

The implementation work was done only in the isolate:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

Main repo source was not edited. This report is the only main-repo artifact.

## Implementation Scope

Added a default-off trace-only simulator:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_KEYREFINE_SIM=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_KEYREFINE_SIM_MAX
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_KEYREFINE_SIM_SAMPLES
```

Changed isolate files:

```text
glr.go
parser.go
parser_config.go
parser_reduce.go
```

The simulator is attached at the same live preflight call site used by `checkPendingFrontierWindowSigGSSRejectSpanlessNegCache`: `canAddLink` sees equivalent payload, `nodesCanMerge(existingPrev, prev)` passes, and the nested `canMergeNodes(existingPrev, prev)` result is computed.

Parser behavior was not changed for the audit runs. The prior behavior cache env was left off; the simulator computes the nested result once, records all variants, and returns that result.

Compared variants:

```text
base_spanless
spanless_current_virtual
spanless_seen_len
spanless_seen_struct
spanless_virtual_global
spanless_full_context
```

`spanless_seen_struct` was feasible: it uses `len(preflight.seen)` plus an order-independent digest of seen `gssMergePair` values using spanless node signatures. `spanless_virtual_global` uses an order-independent structural spanless digest over all preflight virtual links, not a raw pointer-only digest.

## Commands

Compile check:

```sh
cd /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness
go test . -tags treesitter_c_parity -run '^$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter/cgo_harness	7.327s [no tests to run]
```

Replay wrapper:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/spanless-gss-negcache-keyrefine-audit-20260625 \
  --label keyrefine-sim \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "<build measure.test; run exact-file replays serially>"
```

Replay artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/spanless-gss-negcache-keyrefine-audit-20260625/20260625T032754Z-keyrefine-sim
```

Wrapper result:

```text
exit_code=0
oom_killed=false
```

All exact-file replays used:

```text
GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_KEYREFINE_SIM=1
```

The prior behavior env below was intentionally not enabled:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE
```

## Per-Run Results

| Label | File | rc | Wall | Max RSS KB | Parity / progress | Key-refine summary |
|---|---|---:|---:|---:|---|---|
| `java-first` | `/workspace/corpus_sources/java/buildSrc/src/main/java/org/springframework/build/CheckstyleConventions.java` | 0 | 0:04.62 | 1125280 | `parityMatch=1/1`, `diverge=0`, `trunc=0` | all variants `conflict=6` |
| `dart-first` | `/workspace/corpus_sources/dart/.agents/skills/rebuilding-flutter-tool/scripts/rebuild.dart` | 0 | 0:05.82 | 1127680 | `parityMatch=1/1`, `diverge=0`, `trunc=0` | all variants `conflict=0` |
| `tlaplus-bakery` | `/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla` | 124 | 1:37.15 | 1572140 | Timed out; latest progress `iter=387`, `tokens=198` | all variants `conflict=0`, high hit collapse |

## Per-Variant Counters

Java final counters:

```text
base_spanless:           lookup=16 hit=15 miss=1 store=1 true_hit=9 false_hit=6 conflict=6 unique=1 skipped=0
spanless_current_virtual lookup=16 hit=15 miss=1 store=1 true_hit=9 false_hit=6 conflict=6 unique=1 skipped=0
spanless_seen_len        lookup=16 hit=15 miss=1 store=1 true_hit=9 false_hit=6 conflict=6 unique=1 skipped=0
spanless_seen_struct     lookup=16 hit=15 miss=1 store=1 true_hit=9 false_hit=6 conflict=6 unique=1 skipped=0
spanless_virtual_global  lookup=16 hit=15 miss=1 store=1 true_hit=9 false_hit=6 conflict=6 unique=1 skipped=0
spanless_full_context    lookup=16 hit=15 miss=1 store=1 true_hit=9 false_hit=6 conflict=6 unique=1 skipped=0
```

Dart final counters:

```text
base_spanless:           lookup=124 hit=82 miss=42 store=42 true_hit=78 false_hit=4 conflict=0 unique=42 skipped=0
spanless_current_virtual lookup=124 hit=82 miss=42 store=42 true_hit=78 false_hit=4 conflict=0 unique=42 skipped=0
spanless_seen_len        lookup=124 hit=82 miss=42 store=42 true_hit=78 false_hit=4 conflict=0 unique=42 skipped=0
spanless_seen_struct     lookup=124 hit=82 miss=42 store=42 true_hit=78 false_hit=4 conflict=0 unique=42 skipped=0
spanless_virtual_global  lookup=124 hit=82 miss=42 store=42 true_hit=78 false_hit=4 conflict=0 unique=42 skipped=0
spanless_full_context    lookup=124 hit=82 miss=42 store=42 true_hit=78 false_hit=4 conflict=0 unique=42 skipped=0
```

TLA+ latest counters before timeout:

```text
base_spanless:           lookup=1118973 hit=1118949 miss=24 store=24 true_hit=86622 false_hit=1032327 conflict=0 unique=24 skipped=0
spanless_current_virtual lookup=1118973 hit=1118949 miss=24 store=24 true_hit=86622 false_hit=1032327 conflict=0 unique=24 skipped=0
spanless_seen_len        lookup=1118973 hit=1118949 miss=24 store=24 true_hit=86622 false_hit=1032327 conflict=0 unique=24 skipped=0
spanless_seen_struct     lookup=1118973 hit=1118949 miss=24 store=24 true_hit=86622 false_hit=1032327 conflict=0 unique=24 skipped=0
spanless_virtual_global  lookup=1118973 hit=1118949 miss=24 store=24 true_hit=86622 false_hit=1032327 conflict=0 unique=24 skipped=0
spanless_full_context    lookup=1118973 hit=1118949 miss=24 store=24 true_hit=86622 false_hit=1032327 conflict=0 unique=24 skipped=0
```

## Java Conflict Sample

The conflict samples showed no discriminator movement across variants. The Java conflict context was:

```text
base=0x7e60691dc576e927
depth=0 path_len=2 current_link_count=1
current_virtual_count=0 current_virtual_digest=0x0
global_virtual_count=0 global_virtual_digest=0x0
seen_count=1 seen_digest=0x16b608ffe29ab3ed
current={state:505 symbol:108 links:1}
existing_prev={state:2 symbol:137 links:1}
prev={state:2 symbol:137 links:1}
payload={symbol:108 production:0 child_count:0 flags:0}
```

The simulator stored one key and then observed both true and false nested results for that same generalized context:

```text
lookup=16 hit=15 true_hit=9 false_hit=6 conflict=6 unique=1
```

Adding current virtual stats, seen length, seen structural digest, global virtual digest, or their combination did not split the Java key because all of those discriminators were identical at the conflict surface.

## Interpretation

The audit did reproduce the Java conflict at the same broad live preflight surface. It also showed that the proposed generalized context additions do not explain the conflict: every tested variant has the same Java conflict count.

TLA+ still shows the useful collapse signal: 1,118,949 hits out of 1,118,973 lookups with only 24 unique keys and zero conflicts before timeout. However, this does not rescue the live preflight cache because Java remains unsafe under all tested refinements.

## Recommendation

None of the tested key variants removes the Java conflict while preserving the useful TLA+ hit collapse.

The right next move is not more broad live-preflight key growth. Move the cache later or scope it to the classifier-proven negative-only surface, where the old trace-only normkey path had zero Java conflicts. A later-surface/classifier-scoped cache matches the evidence better than trying to make the broad live preflight key distinguish success from failure.
