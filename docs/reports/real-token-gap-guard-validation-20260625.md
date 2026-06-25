# Real Token Gap Guard Validation - 2026-06-25

## Scope

Validation-only pass for the current dirty worktree slice that generalizes real-token attachment/shift gap guarding:

- `parser.go`: `realTokenAttachmentGapIsParserPadding`, `gapIsCoveredByGrammarExtras`, `guardRealTokenAttachmentGap`, `guardRealShiftGap`
- `parser_reduce.go` and `parser_recover_c.go` call sites
- `parser_shift_gap_test.go` focused coverage

No source files were edited. The index was clean before validation.

## Commands and Results

### 1. Worktree and Index Check

Command:

```sh
git status --short
git diff --cached --stat --exit-code
```

Result:

- `git status --short` showed a large dirty worktree with unrelated WIP across parser, grammar, scanner, and report paths.
- `git diff --cached --stat --exit-code` exited 0 with no output, confirming no staged files before validation.

### 2. Focused Docker Package Test

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh -- "cd /workspace && go test . -run 'TestReal.*Gap|TestForest.*Gap' -count=1"
```

Result:

- PASS
- Package result: `ok github.com/odvcencio/gotreesitter 0.011s`
- Harness `exit_code: 0`
- `oom_killed: false`
- Artifacts: `harness_out/docker/20260625T035158Z`

### 3. Java Single-Grammar Parity Smoke

Command:

```sh
bash cgo_harness/docker/run_single_grammar_parity.sh java
```

Result:

- PASS
- Summary: `real-corpus[aggressive]: no-error 25/25, sexpr parity 25/25, deep parity 25/25 (requireParity=false, seen=25/109)`
- Harness `exit_code: 0`
- `oom_killed: false`
- Max RSS: `1037024` KB
- Artifacts: `harness_out/docker/20260625T035217Z-diag-java`

### 4. JSON Single-Grammar Parity Canary

Command:

```sh
bash cgo_harness/docker/run_single_grammar_parity.sh json
```

Result:

- PASS
- Summary: `real-corpus[aggressive]: no-error 6/6, sexpr parity 6/6, deep parity 6/6 (requireParity=false, seen=6/6)`
- Harness `exit_code: 0`
- `oom_killed: false`
- Max RSS: `236692` KB
- Artifacts: `harness_out/docker/20260625T035311Z-diag-json`

### 5. Dart Single-Grammar Parity Smoke

Command:

```sh
bash cgo_harness/docker/run_single_grammar_parity.sh dart
```

Result:

- FAIL at harness summary level because parity mismatches were reported.
- The underlying Go test completed and printed `PASS`, but the harness returned exit code 1 because the one-grammar parity result was `MISMATCH`.
- Summary: `real-corpus[aggressive]: no-error 25/25, sexpr parity 22/25, deep parity 22/25 divs=[childCount=2,range=1] (requireParity=false, seen=28/164)`
- Harness `exit_code: 0` inside the Docker run output, followed by `RESULT: dart - MISMATCH` and process exit 1.
- `oom_killed: false`
- Max RSS: `434856` KB
- Artifacts: `harness_out/docker/20260625T035321Z-diag-dart`

Relevant mismatch lines:

```text
sample 9 (corpus_block:/tmp/grammar_parity/dart/test/corpus/big_tests.txt) deep mismatch: root/function_body/block/expression_statement: childCount (gen=2, ref=5)
gen-src[564:604]: "owner!.buildScope(this, layoutCallback);"
gen-sexpr: (expression_statement (assignment_expression (assignable_expression (identifier) (selector) (unconditional_assignable_selector (identifier))) (record_literal (record_field (this)) (record_field (identifier)))))
ref-sexpr: (expression_statement (identifier) (selector) (selector (unconditional_assignable_selector (identifier))) (selector (argument_part (arguments (argument (this)) (argument (identifier))))))

sample 11 (corpus_block:/tmp/grammar_parity/dart/test/corpus/more_expressions.txt) deep mismatch: root/class_definition/class_body/function_body/block/expression_statement: childCount (gen=2, ref=5)
gen-src[446:486]: "System.out.println(destinysChildString);"

sample 21 (corpus_block:/tmp/grammar_parity/dart/test/corpus/big_tests.txt) deep mismatch: root/function_body/block/assert_statement/assertion/assertion_arguments/logical_and_expression/equality_expression: range (gen=[42:187], ref=[42:59])
gen-src[42:187]: "textColor != null\n      && style != null\n      && margin != null\n      && _position != null\n      && _position.isFinite\n      && _opacity != null"
ref-src[42:59]: "textColor != null"
```

## Assessment

The generalized real-token gap guard slice has positive targeted signal:

- Focused gap-guard package coverage passes in Docker.
- Java real-corpus parity passes 25/25.
- JSON canary parity passes 6/6.
- No run reported timeout or OOM.

The slice is not a clean extraction/commit candidate on this validation alone because Dart still reports real-corpus mismatches under the requested optional smoke. The Dart failures are parse-shape parity mismatches, not compilation failures, timeouts, or memory failures. They may be pre-existing Dart conflict/precedence work in the dirty tree, but this validation did not establish that independently.

Recommended next step before extracting this guard slice: compare Dart against the intended baseline or isolate whether the three Dart mismatches predate the gap guard changes. If those mismatches are already accepted WIP, the guard slice otherwise has enough focused Java/JSON/package signal to consider extraction with a documented Dart caveat.
