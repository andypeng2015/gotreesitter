# Multi-Grammar Window-Sig GSS Normkey Audit

Date: 2026-06-25

## Purpose

This audit extends the TLA+ dominance-isolate normalized structural GSS reject-key simulation beyond the single Bakery witness. The goal was signal and conflict classification only: validate whether `spanless_struct` normalized-key negative-cache/precheck evidence stays conflict-free on a small multi-grammar exact-file matrix, and record no-signal, positive, negative, or conflict evidence without changing parser behavior.

The runs used the trace-only instrumentation already present in:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate
```

No parser behavior flags were enabled. In particular, these remained unset:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_SIG_MERGE
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_EQUIV_MERGE
```

## Command Shape

Main matrix artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/multigrammar-window-sig-gss-normkey-audit-20260625/20260625T025940Z-multigrammar-normkey-audit
```

Dart aggregate follow-up artifact:

```text
/home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/multigrammar-window-sig-gss-normkey-audit-20260625/20260625T030301Z-dart-normkey-aggregate-followup
```

The main run used the isolate Docker wrapper with the corpus mounted read-only:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --out-root /home/draco/work/gotreesitter-tlaplus-dominance-isolate/cgo_harness/harness_out/multigrammar-window-sig-gss-normkey-audit-20260625 \
  --label multigrammar-normkey-audit \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "<build cgo_harness measure.test; run exact-file replays serially>"
```

Each replay used:

```sh
/usr/bin/time -v timeout --kill-after=10s 75s env \
  CGO_ENABLED=1 \
  REPRO_PROGRESS=1 \
  REPRO_SIGNATURES=1 \
  REPRO_N=1 \
  REPRO_ROUNDS=1 \
  GOT_PARSE_PROGRESS=1 \
  GOT_PARSE_PROGRESS_INTERVAL_MS=1000 \
  GOT_GLR_FANOUT_TRACE=1 \
  GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1000 \
  GOT_GLR_PENDING_FRONTIER_GSS_MERGE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_ITERATIVE=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_CLASSIFY_DEEP=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM=1 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM_MAX=200000 \
  GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_NORMKEY_SIM_SAMPLES=16 \
  /workspace/cgo_harness/harness_out/multigrammar-window-sig-gss-normkey-audit-20260625/measure.test \
  -test.run '^TestMeasureDtierVsC$' \
  -test.count=1
```

The Dart follow-up used the same shape, but lowered fanout emission controls to produce aggregate counter lines on the short file:

```text
GOT_GLR_FANOUT_TRACE_INTERVAL_MS=1
GOT_GLR_FANOUT_TRACE_MIN_STACKS=1
```

Wrapper metadata:

```text
main wrapper exit_code=0 oom_killed=false
dart follow-up exit_code=0 oom_killed=false
```

## Matrix

| Label | Grammar | Exact file | Selection note |
|---|---:|---|---|
| `tlaplus-bakery` | `tlaplus` | `/workspace/corpus_sources/tlaplus/specifications/Bakery-Boulangerie/Bakery.tla` | Requested positive witness. |
| `cpp-fmt-args` | `cpp` | `/workspace/corpus_sources/cpp/include/fmt/args.h` | Requested C++ recovery/truncation overlay witness; file existed. |
| `csv-co2` | `csv` | `/workspace/corpus_sources/csv/data/co2-concentration.csv` | Requested CSV witness; file existed. |
| `java-first` | `java` | `/workspace/corpus_sources/java/buildSrc/src/main/java/org/springframework/build/CheckstyleConventions.java` | First deterministic sorted `.java` corpus file. |
| `rust-first` | `rust` | `/workspace/corpus_sources/rust/compiler/rustc/build.rs` | First deterministic sorted `.rs` corpus file. |
| `json-first` | `json` | `/workspace/corpus_sources/json/jsconfig.json` | First deterministic sorted `.json` corpus file. |
| `dart-first` | `dart` | `/workspace/corpus_sources/dart/.agents/skills/rebuilding-flutter-tool/scripts/rebuild.dart` | Optional; first deterministic sorted `.dart` corpus file. |

## Per-Run Results

| Label | rc | Timeout | OOM | Wall | Max RSS KB | Parity/test result | Normkey classification |
|---|---:|---:|---:|---:|---:|---|---|
| `tlaplus-bakery` | 124 | true | false | 1:19.85 | 1572300 | Timed out under trace before completion. | `negative-only-no-conflict` |
| `cpp-fmt-args` | 0 | false | false | 0:06.99 | 1123820 | `parityMatch=0/1`, `diverge=1`, `trunc=1`, `errTree=1` | `no-signal` |
| `csv-co2` | 0 | false | false | 0:03.38 | 1126720 | `parityMatch=1/1` | `no-signal` |
| `java-first` | 0 | false | false | 0:03.19 | 1125544 | `parityMatch=1/1` | `negative-only-no-conflict` |
| `rust-first` | 0 | false | false | 0:05.37 | 1308008 | `parityMatch=1/1` | `no-signal` |
| `json-first` | 0 | false | false | 0:05.23 | 1307328 | `parityMatch=1/1` | `no-signal` |
| `dart-first` | 0 | false | false | 0:06.97 | 1245840 | `parityMatch=1/1` | `negative-only-no-conflict` |

Notes:

- `cpp-fmt-args` reproduced the requested overlay shape but did not produce normalized-key aggregate evidence in this run.
- `rust-first` produced one pending-frontier window signature attempt, but it merged and had `gss_reject=0`; no normkey lookup was exercised.
- The original Dart run emitted normkey samples but no aggregate line; the follow-up generated aggregate counters and is used for Dart classification.

## Counter Excerpts

### TLA+ Bakery

Latest progress before timeout:

```text
PARSE-PROGRESS phase=dispatch_begin lang=tlaplus source_bytes=22753 expected_eof=22753 elapsed_ms=68110 iter=387 tokens=198 stacks=6 live_stacks=6 max_stacks=199 node_count=217154 peak_depth=197 need_token=false single_iters=11 multi_iters=376 token_symbol=5 token_start=459 token_end=460 token_no_lookahead=false token_eof=false num_stacks=6
```

Counters:

```text
pending_frontier_repeat_tail_window_sig={attempt:1113083,prefix_equal:1113083,prefix_not_equal:0,window_equal:1113083,window_incomplete:0,window_capped:0,candidate:1113083,merge:86332,gss_reject:1026751,non_equal:0,no_entries:0,samples:0}

pending_frontier_repeat_tail_window_sig_gss_reject_classify={recursive_can_merge_or_can_add_failure:1026751,actual_merge_unknown:0,samples:16}

pending_frontier_repeat_tail_window_sig_gss_reject_deep={classified:200000,skipped_after_max:826751,can_add_link_nested_can_merge_failure:200000,unknown:0,depth:{0:0,1:200000,2_4:0,5_plus:0},path_len:{1:0,2_4:200000,5_16:0,17_plus:0},head_links:{1:200000,2_4:0,5_plus:0},fail_links:{1:0,2_4:0,5_plus:200000}}

pending_frontier_repeat_tail_window_sig_gss_reject_normkey={strict_lookup:200000,strict_hit:131994,strict_true_hit:0,strict_false_hit:131994,strict_miss:68006,strict_store:68006,strict_conflict:0,strict_unique:68006,spanless_lookup:200000,spanless_hit:199996,spanless_true_hit:0,spanless_false_hit:199996,spanless_miss:4,spanless_store:4,spanless_conflict:0,spanless_unique:4,shape_lookup:200000,shape_hit:199998,shape_true_hit:0,shape_false_hit:199998,shape_miss:2,shape_store:2,shape_conflict:0,shape_unique:2,samples:16}
```

Interpretation: strong negative-only signal. `spanless_struct` collapsed 200,000 nested observations to 4 unique keys with 199,996 false hits and zero conflicts.

### Java First

Latest progress:

```text
MEASURE-PROGRESS lang=java file=1/1 base="CheckstyleConventions.java" path="/workspace/corpus_sources/java/buildSrc/src/main/java/org/springframework/build/CheckstyleConventions.java" bytes=3283 phase=file_complete elapsed_ms=32 go_ms=16 c_ms=0
```

Counters:

```text
pending_frontier_repeat_tail_window_sig={attempt:16,prefix_equal:16,prefix_not_equal:0,window_equal:16,window_incomplete:0,window_capped:0,candidate:16,merge:10,gss_reject:6,non_equal:0,no_entries:0,samples:0}

pending_frontier_repeat_tail_window_sig_gss_reject_classify={recursive_can_merge_or_can_add_failure:6,actual_merge_unknown:0,samples:6}

pending_frontier_repeat_tail_window_sig_gss_reject_deep={classified:6,skipped_after_max:0,can_add_link_nested_can_merge_failure:6,unknown:0,depth:{0:0,1:0,2_4:6,5_plus:0},path_len:{1:0,2_4:0,5_16:6,17_plus:0},head_links:{1:6,2_4:0,5_plus:0},fail_links:{1:0,2_4:0,5_plus:6}}

pending_frontier_repeat_tail_window_sig_gss_reject_normkey={strict_lookup:12,strict_hit:5,strict_true_hit:0,strict_false_hit:5,strict_miss:7,strict_store:7,strict_conflict:0,strict_unique:7,spanless_lookup:12,spanless_hit:9,spanless_true_hit:0,spanless_false_hit:9,spanless_miss:3,spanless_store:3,spanless_conflict:0,spanless_unique:3,shape_lookup:12,shape_hit:10,shape_true_hit:0,shape_false_hit:10,shape_miss:2,shape_store:2,shape_conflict:0,shape_unique:2,samples:12}
```

Interpretation: small but independent negative-only signal. The same recursive nested failure class appears, with `spanless_struct` hits and zero conflicts.

### Rust First

Latest progress:

```text
MEASURE-PROGRESS lang=rust file=1/1 base="build.rs" path="/workspace/corpus_sources/rust/compiler/rustc/build.rs" bytes=1460 phase=file_complete elapsed_ms=53 go_ms=22 c_ms=0
```

Counter:

```text
pending_frontier_repeat_tail_window_sig={attempt:1,prefix_equal:1,prefix_not_equal:0,window_equal:1,window_incomplete:0,window_capped:0,candidate:1,merge:1,gss_reject:0,non_equal:0,no_entries:0,samples:0}
```

Interpretation: pending-frontier signal was seen, but no GSS reject occurred; no normalized reject-key lookup was exercised.

### Dart Follow-Up

Latest progress:

```text
MEASURE-PROGRESS lang=dart file=1/1 base="rebuild.dart" path="/workspace/corpus_sources/dart/.agents/skills/rebuilding-flutter-tool/scripts/rebuild.dart" bytes=1233 phase=file_complete elapsed_ms=63 go_ms=27 c_ms=0
```

Representative aggregate counters:

```text
pending_frontier_repeat_tail_window_sig={attempt:112,prefix_equal:112,prefix_not_equal:0,window_equal:112,window_incomplete:0,window_capped:0,candidate:112,merge:106,gss_reject:6,non_equal:0,no_entries:0,samples:0}

pending_frontier_repeat_tail_window_sig_gss_reject_classify={recursive_can_merge_or_can_add_failure:6,actual_merge_unknown:0,samples:6}

pending_frontier_repeat_tail_window_sig_gss_reject_deep={classified:6,skipped_after_max:0,can_add_link_nested_can_merge_failure:6,unknown:0,depth:{0:0,1:6,2_4:0,5_plus:0},path_len:{1:0,2_4:6,5_16:0,17_plus:0},head_links:{1:6,2_4:0,5_plus:0},fail_links:{1:0,2_4:0,5_plus:6}}

pending_frontier_repeat_tail_window_sig_gss_reject_normkey={strict_lookup:6,strict_hit:2,strict_true_hit:0,strict_false_hit:2,strict_miss:4,strict_store:4,strict_conflict:0,strict_unique:4,spanless_lookup:6,spanless_hit:2,spanless_true_hit:0,spanless_false_hit:2,spanless_miss:4,spanless_store:4,spanless_conflict:0,spanless_unique:4,shape_lookup:6,shape_hit:4,shape_true_hit:0,shape_false_hit:4,shape_miss:2,shape_store:2,shape_conflict:0,shape_unique:2,samples:6}
```

Interpretation: small negative-only signal with zero conflicts. Unlike TLA+ and Java, `spanless_struct` is not more selective than strict on this tiny file, but it remains conflict-free.

## No-Signal Runs

The following runs completed but produced no normalized reject-key aggregate counters:

```text
cpp-fmt-args: MEASURE-DTIER cpp mode=prod files=1 medianRatio=6.30x aggRatio=6.30x parityMatch=0/1(0%) diverge=1 trunc=1 errTree=1 panics=0 goNS=21936900 cNS=3482193

csv-co2: MEASURE-DTIER csv mode=prod files=1 medianRatio=2.48x aggRatio=2.48x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=6457270 cNS=2602137

json-first: MEASURE-DTIER json mode=prod files=1 medianRatio=39.61x aggRatio=39.61x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=1389719 cNS=35088
```

`rust-first` had one pending-frontier signature attempt but no reject, so it is grouped as no-signal for normalized reject-key conflict classification.

## Interpretation

Evidence produced:

- No conflicts were observed for `strict_struct`, `spanless_struct`, or `shape_bucket` in any run that exercised normkey lookup.
- TLA+ remains the strong positive witness for a spanless normalized-key negative cache or precheck: `spanless_lookup=200000`, `spanless_hit=199996`, `spanless_conflict=0`, `spanless_unique=4`.
- Java provides independent, small, clean-mainstream negative-only evidence: `spanless_lookup=12`, `spanless_hit=9`, `spanless_conflict=0`.
- Dart provides optional Tier IV negative-only evidence after increasing trace emission frequency: `spanless_lookup=6`, `spanless_hit=2`, `spanless_conflict=0`.
- C++ `args.h` still shows recovery/truncation overlay behavior, but this exact run did not exercise the normalized reject-key path. It is useful as a parity/recovery witness, not as normkey evidence.
- CSV and JSON were clean canaries with no normalized reject-key signal.
- No mixed positive/negative outcome and no conflict were observed in this bounded audit.

This is still not broad enough to make `shape_bucket` behavior-bearing. Its zero-conflict result is useful as a selectivity diagnostic, but the safer next behavior experiment is the more structural spanless key.

## Recommended Next Experiment

Implement a trace-guarded `spanless_struct` normalized-key negative cache or selectivity precheck at the nested GSS preflight path where the isolate currently observes `canAddLink`/`canMergeNodes` failure. Keep it generalized and behavior-disabled by default:

```text
GOT_GLR_PENDING_FRONTIER_REPEAT_TAIL_WINDOW_SIG_GSS_REJECT_SPANLESS_NEGCACHE=1
```

Suggested shape:

- Key only the observed nested failure class, not grammar names or grammar-specific states.
- Cache negative nested `canMergeNodes(existingPrev, prev)` results by the spanless structural key.
- Keep conflict auditing live under the same trace family: if the same spanless key ever observes both true and false nested outcomes, report conflict and bypass the cache for that key.
- Validate first on the same exact-file matrix, then expand one grammar/file at a time, preserving Docker isolation and bounded timeouts.
- Keep `shape_bucket` diagnostic-only until multi-grammar conflict evidence remains at zero over a materially larger corpus.

The audit supports continuing with generalized parser/GSS machinery. It does not support a per-grammar patch.
