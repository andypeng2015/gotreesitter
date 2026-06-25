# Pending-Frontier GSS Candidate-Key Audit

Date: 2026-06-25

## Purpose

This trace-only audit tested whether caching the whole pending-frontier GSS candidate outcome at `tryMergePendingFrontierForkCandidate(target, fork)` avoids the Java true/false conflict seen by the broader live nested spanless cache.

Implementation work was done only in the isolate:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

Main repo source was not edited. This report is the only main-repo artifact.

## Implementation Scope

Added a default-off trace-only simulator:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_CANDIDATE_KEY_SIM=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_CANDIDATE_KEY_SIM_MAX
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_CANDIDATE_KEY_SIM_SAMPLES
```

Changed isolate files:

```text
parser_config.go
parser.go
parser_reduce.go
```

The simulator records only the observed boolean result of `tryMergePendingFrontierForkCandidate`: `true` for a completed `tryGSSMainMergeForParser` merge and `false` for any candidate reject. It does not skip, alter, or replay parser behavior.

Compared variants:

```text
candidate_spanless_heads
candidate_head_shape
candidate_strict_heads
candidate_window_context
candidate_full
```

`candidate_window_context` was feasible on the window-signature path. The window proof is already available where `pendingFrontierRepeatTailWindowSignaturesEquivalent` returns `proof`, so the window-sig branch passes a compact proof digest into the candidate function. Non-window callers keep a zero proof key, but this audit matrix exercises the window-sig branch.

`candidate_full` adds strict head signatures plus branch-order ordering bucket and C-recovery merge-cost equivalence. No raw pointer-only candidate key was used as a main variant.

## Commands

Compile check:

```sh
cd /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness
go test . -tags treesitter_c_parity -run '^$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter/cgo_harness	3.892s [no tests to run]
```

Replay wrapper:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/pending-frontier-gss-candidate-key-audit-20260625 \
  --label candidate-key-sim \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "<build measure.test; run exact-file replays serially>"
```

Artifacts:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/pending-frontier-gss-candidate-key-audit-20260625/20260625T034312Z-candidate-key-sim
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/pending-frontier-gss-candidate-key-audit-20260625/20260625T034313Z-candidate-key-sim-inner
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
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_CANDIDATE_KEY_SIM=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_CANDIDATE_KEY_SIM_MAX=500000
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_CANDIDATE_KEY_SIM_SAMPLES=24
```

The prior behavior env was intentionally not enabled:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE
```

## Per-Run Results

| Label | File | rc | Wall | Max RSS KB | Parity / progress | Candidate summary |
|---|---|---:|---:|---:|---|---|
| `java-first` | `/workspace/corpus_sources/java/buildSrc/src/main/java/org/springframework/build/CheckstyleConventions.java` | 0 | 0:04.52 | 1397480 | `parityMatch=1/1`, `diverge=0`, `trunc=0` | all variants conflicted; Java canary failed |
| `dart-first` | `/workspace/corpus_sources/dart/.agents/skills/rebuilding-flutter-tool/scripts/rebuild.dart` | 0 | 0:08.13 | 1313512 | `parityMatch=1/1`, `diverge=0`, `trunc=0` | all variants conflicted |
| `tlaplus-bakery` | `/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla` | 124 | 1:35.44 | 1599332 | Timed out; latest progress `iter=363`, `tokens=186` | strong reject hit signal, but all variants conflicted |

## Per-Variant Counters

Java final counters:

```text
candidate_spanless_heads  lookup=16 hit=15 miss=1  store=1  true_hit=9     false_hit=6      conflict=6      unique=1      skipped=0
candidate_head_shape      lookup=16 hit=15 miss=1  store=1  true_hit=9     false_hit=6      conflict=6      unique=1      skipped=0
candidate_strict_heads    lookup=16 hit=15 miss=1  store=1  true_hit=9     false_hit=6      conflict=6      unique=1      skipped=0
candidate_window_context  lookup=16 hit=5  miss=11 store=11 true_hit=5     false_hit=0      conflict=5      unique=11     skipped=0
candidate_full            lookup=16 hit=15 miss=1  store=1  true_hit=9     false_hit=6      conflict=6      unique=1      skipped=0
```

Dart final counters:

```text
candidate_spanless_heads  lookup=112 hit=69  miss=43 store=43 true_hit=67    false_hit=2      conflict=6      unique=43     skipped=0
candidate_head_shape      lookup=112 hit=102 miss=10 store=10 true_hit=96    false_hit=6      conflict=6      unique=10     skipped=0
candidate_strict_heads    lookup=112 hit=58  miss=54 store=54 true_hit=56    false_hit=2      conflict=5      unique=54     skipped=0
candidate_window_context  lookup=112 hit=42  miss=70 store=70 true_hit=40    false_hit=2      conflict=6      unique=70     skipped=0
candidate_full            lookup=112 hit=58  miss=54 store=54 true_hit=56    false_hit=2      conflict=5      unique=54     skipped=0
```

TLA+ latest counters before timeout:

```text
candidate_spanless_heads  lookup=929187 hit=927922 miss=1265   store=1265   true_hit=75318 false_hit=852604 conflict=838186 unique=1265   skipped=0
candidate_head_shape      lookup=929187 hit=928840 miss=347    store=347    true_hit=76066 false_hit=852774 conflict=852774 unique=347    skipped=0
candidate_strict_heads    lookup=929187 hit=614891 miss=314296 store=314296 true_hit=49683 false_hit=565208 conflict=508    unique=314296 skipped=0
candidate_window_context  lookup=929187 hit=869880 miss=59307  store=59307  true_hit=45502 false_hit=824378 conflict=14444  unique=59307  skipped=0
candidate_full            lookup=929187 hit=614891 miss=314296 store=314296 true_hit=49683 false_hit=565208 conflict=508    unique=314296 skipped=0
```

## Samples

Java reproduced the original canary conflict at the candidate level. The same target/fork head shape first merged and later rejected:

```text
sample=1 variant=candidate_spanless_heads result=true  key=0x4817c114b8da4d4 target={state:505 byte:1306 score:0 shifted:true branch:2 head_state:505 head_symbol:108 head_links:1 head_link_states:[2] head_link_symbols:[137] head_link_counts:[1]} fork={state:505 byte:1306 score:0 shifted:true branch:3 head_state:505 head_symbol:108 head_links:1 head_link_states:[2] head_link_symbols:[137] head_link_counts:[1]}
sample=10 variant=candidate_spanless_heads result=false key=0x4817c114b8da4d4 target={state:505 byte:1306 score:0 shifted:true branch:2 head_state:505 head_symbol:108 head_links:1 head_link_states:[2] head_link_symbols:[137] head_link_counts:[1]} fork={state:505 byte:1306 score:0 shifted:true branch:8 head_state:505 head_symbol:108 head_links:1 head_link_states:[2] head_link_symbols:[137] head_link_counts:[1]}
```

The window-context proof split Java more than the other variants, but it still conflicted:

```text
candidate_window_context lookup=16 hit=5 miss=11 store=11 true_hit=5 false_hit=0 conflict=5 unique=11
```

Dart and TLA+ were not just no-signal controls. Both also observed same-key true/false outcomes in every tested variant.

## Interpretation

Candidate-level keying does not avoid the Java conflict. It moves the observation point from nested `canMergeNodes` to the whole pending-frontier candidate, but the top-level candidate key can still be true and false for the same generalized shape.

TLA+ still shows the desired reject-collapse pressure: `candidate_spanless_heads` saw only `1265` unique keys for `929187` lookups and `852604` false hits before timeout. However, that collapse is inseparable from massive conflicts (`838186`). The stricter variants reduce conflicts substantially, but they do not eliminate them, and they lose much of the collapse (`314296` unique keys for `candidate_strict_heads` / `candidate_full`).

The window proof context was available and useful as a discriminator, but not sufficient. `candidate_window_context` reduced Java to `5` conflicts and TLA+ to `14444` conflicts, still disqualifying it for a behavior gate.

## Recommendation

Do not run a candidate-level negative-cache behavior experiment from these keys. The Java canary still conflicts, Dart also conflicts, and TLA+ reject collapse is only strong on variants that are plainly unsafe.

The next safe behavior experiment is not a broad candidate-level cache/precheck. Abandon this cache/precheck line as the near-term generalized machinery lever. If caching is revisited, it should be scoped to a later, classifier-proven negative-only surface with verification kept on during rollout; otherwise move effort to a different generalized GLR lever such as reuse/invalidation scope, recovery/materialization cost, or allocator/retention work.
