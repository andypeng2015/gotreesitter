# Clean Baseline Gap-Guard Source Commits

Date: 2026-06-25

Main repo changes: this report only.

Source worktree: `/home/draco/work/gotreesitter-build-baseline`

Source branch: `repair/grammargen-build-baseline`

The clean build-baseline worktree has two source commits worth preserving for later integration. They are not merged into the dirty main worktree yet because the main worktree currently has overlapping WIP in parser, grammargen, grammar, and generated files. This report records the source wins so they are not stranded in side-agent chat while integration is deferred.

## Commit `a18e301f`

Message: `add: Add repeat metadata and recovery certification`

Files changed:

- `language.go`
- `generated_repeat_metadata.go`
- `parser_recover_c.go`
- `grammargen/assemble.go`
- `grammargen/diagnostics.go`
- `grammargen/lr.go`
- `grammargen/parity_test.go`
- `grammargen/python_keyword_test.go`

Purpose:

- Added generated repeat metadata plumbing.
- Added recovery certification support for grammargen output and runtime recovery paths.
- Extended parity coverage around the grammargen recovery behavior.

Validation recorded from the clean worktree:

- Focused grammargen Docker run passed.
- JSON one-grammar parity reached execution and passed.
- Java parity reached execution and held at the baseline `23/25` mismatch.

## Commit `c9312ccb`

Message: `improve(parser): improve shift-gap padding detection`

Files changed:

- `lexer.go`
- `parser.go`
- `parser_dfa_token_source.go`
- `parser_shift_gap_test.go`

Purpose:

- Improved parser shift-gap padding detection.
- Added focused coverage for shift-gap behavior.
- Kept the grammargen compile surface healthy after the parser/token-source change.

Validation recorded from the clean worktree:

- Focused package Docker run passed.
- JSON parity passed `6/6`.
- Java parity reached execution and remained at `23/25`, with no regression.
- Docker `./grammargen` compile passed.

## Integration Note

These commits are source changes on the clean branch `repair/grammargen-build-baseline`, not source changes on this dirty main worktree. The next integration pass should cherry-pick or manually port them with conflict review, because main already has overlapping edits in several of the same files:

- `language.go`
- `lexer.go`
- `parser.go`
- `parser_dfa_token_source.go`
- `parser_recover_c.go`
- `parser_shift_gap_test.go`
- `grammargen/assemble.go`
- `grammargen/diagnostics.go`
- `grammargen/parity_test.go`

Keep correctness validation separate from performance validation when integrating. For this report commit, no source files were staged or changed.
