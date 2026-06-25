# Spanless GSS Negcache Behavior Experiment

Date: 2026-06-25

## Purpose

This experiment implemented the next generalized machinery step from `multigrammar-window-sig-gss-normkey-audit-20260625.md` in the TLA+ dominance isolate:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

The main repository source was not edited. This report is the only main-repo artifact.

## Implementation Scope

The isolate now has a default-off behavior gate:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE=1
```

Optional envs added:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE_MAX
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE_VERIFY_HITS=1
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE_SAMPLES
```

Changed isolate files:

```text
glr.go
parser.go
parser_config.go
parser_reduce.go
```

The cache is parser-owned and reset per parse. It is wired only through `tryGSSMainMergeForParser`, which is the pending-frontier `tryGSSMainMergeForParser` path that TLA+ exercised. No per-grammar logic, grammar names, or grammar-specific states were added.

The live preflight now carries classifier-style `depth` and `pathLen` through `canMergeNodes` / `canAddLink`, computes the spanless structural key with the existing `gssRejectNormStrictKey(..., spanful=false)`, and caches only negative `canMergeNodes(existingPrev, prev)` outcomes. Cache hits return `false` directly unless `VERIFY_HITS=1`; verified hits compute the real result with the behavior hook disabled, report conflicts, and bypass conflicted keys.

New trace counter:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_spanless_negcache={lookup,hit,miss,store,bypass,conflict,verified,unique,skipped_after_max,samples}
```

The behavior flags below were not enabled:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE
```

## Commands

Compile check:

```sh
cd /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness
go test . -tags treesitter_c_parity -run '^$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter/cgo_harness	2.364s [no tests to run]
```

Replay artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/spanless-gss-negcache-behavior-experiment-20260625/20260625T031401Z-spanless-negcache-behavior
```

Wrapper result:

```text
exit_code=0
oom_killed=false
```

Replay shape:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/spanless-gss-negcache-behavior-experiment-20260625 \
  --label spanless-negcache-behavior \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "<build measure.test; run exact-file replays serially>"
```

Each replay used the pending-frontier window signature path, old normkey simulation, and the new spanless negcache. TLA+ used `VERIFY_HITS=0`; Java and Dart used `VERIFY_HITS=1`.

## Results

| Label | File | Verify hits | rc | Wall | Max RSS KB | Parity / progress | New negcache result |
|---|---|---:|---:|---:|---:|---|---|
| `tlaplus-bakery` | `/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla` | 0 | 124 | 1:38.00 | 1598604 | Timed out; latest progress `iter=399`, `tokens=204` | `lookup=1245054 hit=1151932 miss=24 store=4 bypass=93098 conflict=0 verified=0 unique=24` |
| `java-first` | `/workspace/corpus_sources/java/buildSrc/src/main/java/org/springframework/build/CheckstyleConventions.java` | 1 | 0 | 0:03.48 | 1398232 | `parityMatch=1/1`, `diverge=0`, `trunc=0` | `lookup=16 hit=0 miss=1 store=0 bypass=15 conflict=1 verified=0 unique=1` |
| `dart-first` | `/workspace/corpus_sources/dart/.agents/skills/rebuilding-flutter-tool/scripts/rebuild.dart` | 1 | 0 | 0:05.22 | 1306888 | `parityMatch=1/1`, `diverge=0`, `trunc=0` | `lookup=112 hit=2 miss=42 store=4 bypass=68 conflict=0 verified=2 unique=42` |

Prior TLA+ 75s baseline from `multigrammar-window-sig-gss-normkey-audit-20260625.md`:

```text
iter=387 tokens=198 wall=1:19.85 RSS=1572300
spanless_lookup=200000 spanless_hit=199996 spanless_unique=4 spanless_conflict=0
```

This run reached `iter=399`, `tokens=204` before the 90s timeout. It progressed farther in absolute terms, but not enough to claim a decisive throughput win because timeout length and tracing cadence differed.

## Counter Notes

TLA+ kept the old trace-only normkey signal unchanged:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_normkey={strict_lookup:200000,strict_hit:131994,strict_true_hit:0,strict_false_hit:131994,strict_miss:68006,strict_store:68006,strict_conflict:0,strict_unique:68006,strict_skipped_after_max:0,spanless_lookup:200000,spanless_hit:199996,spanless_true_hit:0,spanless_false_hit:199996,spanless_miss:4,spanless_store:4,spanless_conflict:0,spanless_unique:4,spanless_skipped_after_max:0,shape_lookup:200000,shape_hit:199998,shape_true_hit:0,shape_false_hit:199998,shape_miss:2,shape_store:2,shape_conflict:0,shape_unique:2,shape_skipped_after_max:0,samples:16}
```

The new behavior cache saw a large TLA+ negative-hit signal:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_spanless_negcache={lookup:1245054,hit:1151932,miss:24,store:4,bypass:93098,conflict:0,verified:0,unique:24,skipped_after_max:0,samples:16}
```

Java exposed a conflict in verified mode:

```text
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT-SPANLESS-NEGCACHE sample=1 reason=miss result=true key=0x7e60691dc576e927
GLR-PENDING-FRONTIER-WINDOW-SIG-GSS-REJECT-SPANLESS-NEGCACHE sample=2 reason=conflict result=false key=0x7e60691dc576e927

pending_frontier_repeat_tail_window_sig_gss_reject_spanless_negcache={lookup:16,hit:0,miss:1,store:0,bypass:15,conflict:1,verified:0,unique:1,skipped_after_max:0,samples:2}
```

Dart verified cleanly:

```text
pending_frontier_repeat_tail_window_sig_gss_reject_spanless_negcache={lookup:112,hit:2,miss:42,store:4,bypass:68,conflict:0,verified:2,unique:42,skipped_after_max:0,samples:16}
```

## Interpretation

The broad live `spanless_struct` behavior key is not production-port-ready. TLA+ shows the intended high-selectivity negative-hit behavior, and Dart verifies cleanly, but Java produced one same-key true/false conflict under `VERIFY_HITS=1`. Correctness was preserved in the experiment because the conflict path marks that key bypassed and returns the computed result.

The conflict also explains a distinction from the previous trace-only audit: the old normkey simulator observed the nested reject classifier path, while this behavior cache observes the live preflight call site before the final rejection is known. That broader observation surface can see both successful and failing `canMergeNodes` outcomes for the same spanless structural key.

## Recommendation

Do not port this broad `spanless_struct` live behavior cache as-is. The next experiment should narrow the cache key or scope so it only covers the negative-only reject class proven by the deep classifier, or add enough structural discriminators to eliminate the Java conflict while preserving the TLA+ collapse. Keep `VERIFY_HITS=1` mandatory for any expanded matrix until Java-style same-key conflicts remain zero across multiple grammars.
