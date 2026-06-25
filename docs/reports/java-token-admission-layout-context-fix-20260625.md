# Java Token Admission Layout Context Fix

Clean source branch/worktree `/home/draco/work/gotreesitter-build-baseline`, branch
`repair/java-token-admission-reduced`, contains the source commits for this win:

- `039ed2c4 fix(lexer): fix DFA token source error symbol preference`
- `163dfde0 improve(parser): improve lexer token preference for extra layout context`

`163dfde0` refines and supersedes `039ed2c4`.

## Root Cause

The generated Java parse table admitted `escape_sequence`, but the DFA token
source selected the after-whitespace lex state because the previous byte was a
newline. In Java text blocks and string-template content, that newline is
parser-visible string content, not skipped layout. The lexer therefore dropped
immediate escape tokens and returned `errorSymbol`.

## Review Finding

The initial `039ed2c4` fix could over-accept `token.immediate` after
parser-visible extra/layout because base non-error tokens won over
after-whitespace `ERROR` too broadly. Existing tests missed the negative
immediate-token case.

## Refined Fix

`163dfde0` threads parser-derived `afterExtraLayout` context through the DFA
token source and included-range token sources. Base-mode fallback over
after-whitespace `ERROR` is allowed only when the base token is non-immediate,
or when the parser is not immediately after parser-visible extra/layout. The
commit also adds negative `ImmediateTokens` regression coverage.

Changed files in `163dfde0`:

- `parser_dfa_token_source.go`
- `parser.go`
- `parser_api.go`
- `included_ranges.go`
- `parser_dfa_token_source_test.go`
- `included_ranges_test.go`

## Validation

- Host focused parser tests passed.
- Host selected grammargen tests passed.
- Docker focused parser tests: `harness_out/docker/20260625T054437Z`.
- Docker selected grammargen tests: `harness_out/docker/20260625T054446Z`.
- Docker JSON parity: `harness_out/docker/20260625T054510Z-diag-json`;
  no-error/sexpr/deep `6/6`.
- Docker Java parity: `harness_out/docker/20260625T054515Z-diag-java`;
  no-error/sexpr/deep `25/25`.

No Java-specific code, grammar blobs, or result normalizers were changed. The
source commits remain on the clean source branch and are not merged into the
dirty main worktree yet.
