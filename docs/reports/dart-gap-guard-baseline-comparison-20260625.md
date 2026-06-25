# Dart Gap Guard Baseline Comparison - 2026-06-25

## Scope

Question: determine whether the Dart `22/25` mismatches reported in
`docs/reports/real-token-gap-guard-validation-20260625.md` are pre-existing on
clean committed HEAD or specific to the dirty worktree.

Dirty-worktree reference result from
`docs/reports/real-token-gap-guard-validation-20260625.md`:

```text
bash cgo_harness/docker/run_single_grammar_parity.sh dart
real-corpus[aggressive]: no-error 25/25, sexpr parity 22/25, deep parity 22/25 divs=[childCount=2,range=1] (requireParity=false, seen=28/164)
RESULT: dart - MISMATCH
samples: 9, 11, 21
```

## Clean HEAD Run

Clean baseline worktree:

```text
/tmp/gotreesitter-clean-head-7e96ed03
HEAD: 7e96ed0399ed25c5ed4489fcb3bf51b55595ac55
```

Commands:

```sh
git worktree add --detach /tmp/gotreesitter-clean-head-7e96ed03 7e96ed03
cd /tmp/gotreesitter-clean-head-7e96ed03
bash cgo_harness/docker/run_single_grammar_parity.sh dart
```

Outcome:

```text
RESULT: dart - FAILED (exit=1) | NO SUMMARY
SUMMARY: 0 passed, 1 failed, 0 OOM out of 1 grammars
artifacts: /tmp/gotreesitter-clean-head-7e96ed03/harness_out/docker/20260625T040433Z-diag-dart
oom_killed: false
```

The run failed at package build time before Dart corpus samples were evaluated:

```text
FAIL github.com/odvcencio/gotreesitter/grammargen [build failed]
grammargen/diagnostics.go:558:15: assignment mismatch: 1 variable but resolveConflicts returns 2 values
grammargen/diagnostics.go:558:40: not enough arguments in call to resolveConflicts
grammargen/diagnostics.go:575:13: assignment mismatch: 1 variable but resolveConflicts returns 2 values
grammargen/diagnostics.go:575:38: not enough arguments in call to resolveConflicts
grammargen/lr_lalr.go:270:42: ctx.definitionBoundaryTagBySym undefined
grammargen/lr_lalr.go:271:21: ctx.definitionBoundaryTagBySym undefined
grammargen/lr_repetition_conflict_test.go:921:13: meta.GeneratedRepeatAux undefined
grammargen/python_keyword_test.go:160:11: lang.CRecoveryCostCompetitionCapable undefined
grammargen/python_keyword_test.go:163:10: lang.CRecoveryCostCompetitionEnabledByDefault undefined
grammargen/python_keyword_test.go:167:11: lang.CRecoveryCostCompetitionEnabledByDefault undefined
```

## Sample Comparison

Clean committed HEAD did not produce a real-corpus parity summary, so there are
no clean-head Dart sample IDs to compare.

Dirty-worktree mismatching sample IDs remain:

```text
sample 9:  childCount mismatch
sample 11: childCount mismatch
sample 21: range mismatch
```

Match status against dirty samples `9/11/21`: unavailable. The clean baseline
failed before sample execution, so it neither confirms nor refutes the dirty
worktree's `22/25` result.

## Dirty Dart WIP Inspection

Only changed files/tests were inspected. The dirty worktree contains
Dart-specific changes outside the real-token gap guard slice:

```text
M  grammargen/dart_h5_eager_default_reduce_test.go
M  grammars/dart_scanner.go
M  grammars/grammar_blobs/dart.bin
?? cgo_harness/dart_runtime_recovery_cgo_test.go
?? dart_runtime_recovery_test.go
?? grammargen/dart_cast_precedence_parity_test.go
?? grammargen/dart_lr_split_parity_test.go
?? grammargen/dart_runtime_recovery_parity_test.go
?? grammargen/dart_switch_object_pattern_parity_test.go
?? grammars/dart_scanner_test.go
```

Relevant observed deltas:

- `grammars/dart_scanner.go` now remaps external scanner symbols by language
  metadata and changes Dart scanner token dispatch.
- `grammars/grammar_blobs/dart.bin` changed substantially
  (`91927 -> 685057` bytes).
- `grammargen/dart_h5_eager_default_reduce_test.go` now uses
  `generateDartParityLanguageWithTimeout`, which enables Dart LR splitting.
- New untracked Dart parity/recovery tests target cast precedence, LR splitting,
  runtime recovery, and switch object patterns.

These are plausible Dart parse-shape influences and are unrelated to the
real-token gap guard functions validated in the dirty-worktree report.

## Conclusion

Clean committed HEAD `7e96ed03` does not establish that the Dart `22/25`
mismatch is pre-existing, because the requested clean-head Dart parity command
does not build and therefore produces no parity summary or mismatch samples.

The dirty worktree's Dart `22/25` result remains a caveat for the gap-guard
validation, but the caveat cannot be attributed to the real-token gap guard
slice from this comparison. The dirty tree contains separate Dart scanner,
grammar blob, LR-splitting, and Dart parity/recovery WIP that are more likely
owners of Dart-specific parse-shape changes.

Status for the exact questions:

```text
Clean HEAD Dart parity outcome: build failed before parity; no summary
Matches dirty samples 9/11/21: unavailable; clean run produced no sample IDs
Gap-guard slice cleared by Dart baseline: no, still caveated by unavailable baseline plus dirty Dart WIP
```
