# Java Token Admission Source Fix

Date: 2026-06-25

Main repo changes: this report only.

Source worktree: `/home/draco/work/gotreesitter-build-baseline`

Source branch: `repair/java-token-admission-reduced`

Source commit:

```text
039ed2c4 fix(lexer): fix DFA token source error symbol preference
```

This report records the clean-branch source fix for the Java parity residual
previously classified as a grammargen DFA/LR token-admission issue at repeated
multiline-string boundaries. The source commit is on the clean build-baseline
branch above. It is not merged into this dirty main worktree yet.

## Changed Files In Source Commit

- `parser_dfa_token_source.go`
- `parser_dfa_token_source_test.go`
- `grammargen/counted_repetition_truncation_test.go`

## Root Cause

The generated Java parse table admitted `escape_sequence` at the failing text
block boundary, but the DFA token source selected the after-whitespace lexer
state because the previous byte was a newline.

For Java text blocks, that newline was real string-fragment content, not skipped
layout. After-whitespace mode dropped the immediate escape tokens and returned
`errorSymbol`, so the parser saw an error even though base mode had a valid
`escape_sequence` token available.

## Fix

The source commit generalizes token-source preference between base mode and
after-whitespace mode:

- keep a real base-mode token when after-whitespace mode only produces
  `errorSymbol`;
- still prefer a real after-whitespace token where it exists;
- keep base `errorSymbol` losing to a real after-whitespace token.

The change is not Java-specific. It fixes the token-source preference rule that
made a real admitted token lose to `errorSymbol` solely because the previous byte
looked like whitespace.

## Validation

Validation was run from the clean build-baseline worktree after source commit
`039ed2c4`.

Focused Docker tests passed:

```text
/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T051655Z
```

JSON parity passed `6/6`:

```text
/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T051738Z-diag-json
```

Java parity improved from `23/25` to `25/25`:

```text
/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T051750Z-diag-java
```

## Integration Note

Do not treat this report as an integration of the source fix. The source commit
exists on clean branch `repair/java-token-admission-reduced` in worktree
`/home/draco/work/gotreesitter-build-baseline`.

The dirty main worktree has overlapping WIP, including
`parser_dfa_token_source.go` and `parser_dfa_token_source_test.go`, so the source
change should be cherry-picked or manually ported in a separate integration pass
with conflict review. Keep correctness validation separate from performance
validation when integrating.
