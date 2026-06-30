# August 5 High-Value Parity Gates

Date: 2026-06-30
Branch: `codex/glr-parity-buckets-20260626`

## Milestone Target

The August 5 target is a strong architecture and evidence milestone, not a
claim of full C tree-sitter parity across all grammars.

Success means:

- Correct or near-correct parity on a curated high-value language set.
- Performance good enough for editor and tooling workflows on the same set.
- Clear evidence that failures are moving into generalized parser or
  grammargen machinery instead of per-grammar output normalization.
- Honest accounting of remaining C-faithfulness gaps.

## High-Value Language Set

| Language | Role | Primary correctness gate | Primary perf gate |
| --- | --- | --- | --- |
| Go | performance and incremental baseline | focused package and C-oracle corpus checks | `BenchmarkGoParseFullDFA`, `BenchmarkGoParseIncrementalSingleByteEditDFA`, `BenchmarkGoParseIncrementalNoEditDFA` |
| Rust | hard correctness proving ground | generated-grammar parity canary plus Rust recovery witnesses | full parse smoke after correctness stabilizes |
| Python | indentation and external-scanner pressure | generated Python parity plus real-corpus recovery sample | full parse smoke and incremental no-edit smoke |
| JavaScript | broad syntax pressure and clean ratchet | keep 40/40 clean gate green | full parse smoke |
| TypeScript | broad syntax plus perf pressure | generated TS/TSX parity canaries | timeout/OOM smoke separated from correctness |
| C or C++ | recovery and GLR stress | one Docker parity target and one real-corpus target | max-RSS and full parse smoke |
| C# | recovery stress with practical user value | generated C# parity and residual switch/try-finally witnesses | full parse smoke |
| Swift | normalization retirement evidence | selected recovery witnesses | secondary, not the main success metric |

## Gate Shape

Each high-value language should have separate correctness and performance gates.
Do not treat a perf timeout fix as correctness evidence, and do not treat a
correctness win as a perf win.

Correctness gate per language:

- One bounded Docker parity command.
- One real-corpus target when a stable corpus exists.
- One focused witness for the current failure family.
- One expected classification if it still fails.

Performance gate per language:

- `GOMAXPROCS=1`.
- `-count=10` where benchmark cost allows.
- `-benchtime=750ms`.
- `-benchmem`.
- `/usr/bin/time -v` on macro full-parse smoke for max RSS.

## Failure Taxonomy

Every new failure should be classified before any compatibility patch is added:

- `parser-machinery`: GLR, recovery election, result materialization, stack or forest behavior.
- `grammargen-metadata`: missing or wrong table metadata, production shape, hidden choice, supertype, alias, field, or token-admission facts.
- `scanner-faithfulness`: external scanner or lexical mode mismatch.
- `recovery-ambiguity`: C chooses a recovery version Go cannot mechanically distinguish yet.
- `irreducible-c-behavior`: behavior depends on C implementation details that are not mechanically recoverable from tables.
- `temporary-normalization`: an explicit compatibility bridge with a removal condition.

Temporary normalization must name the generalized mechanism expected to replace
it. New parser or grammargen fixes should reduce this bucket over time.

## Current Generalized Machinery Evidence

Landed slices on this branch:

- `db71d9a6 add(grammargen): add ProductionSignatures to Language struct`
- `c536921c improve(parser): wrap recovery nodes with matching signatures`
- `d99b2fe9 improve(c-recovery): Improve GLR recovery signature matching`

The recovery wrapper now uses generated production signatures, supertype maps,
and hidden-choice passthrough metadata. It preserves flattening when no unique
safe match exists, and refuses fielded or aliased signatures until wrapper
materialization can preserve those details faithfully.

Verified focused parser gates:

```sh
env GOWORK=off /usr/local/go/bin/go test . -run 'TestCAppendVisibleSpliceRecoverySplicesExtraErrorCarrier|TestCRecoveryVisibleSplice' -count=1
env GOWORK=off /usr/local/go/bin/go test . -run '^$' -count=1
```

## Rust Generated-Grammar Canary

A bounded Docker canary was run for the Rust generated grammar path:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --no-build \
  --label rust-signature-canary \
  --memory 6g \
  --cpus 1 \
  --pids 512 \
  --mount /home/draco/work/parser-repos/rust-261b20226c04:/tmp/grammar_parity/rust:ro \
  -- "cd /workspace && env GOWORK=off GOMAXPROCS=1 GOFLAGS=-p=1 /usr/bin/time -v /usr/local/go/bin/go test ./grammargen -run '^TestRustCorpusMatchLetGuardParity$' -count=1 -v"
```

Result:

- Failed: generated Rust tree diverged from checked-in Rust reference on
  `TestRustCorpusMatchLetGuardParity`.
- Artifact: `harness_out/docker/20260630T002446Z-rust-signature-canary/`.
- `oom_killed=false`.
- Max RSS: `983548 KB`.
- Wall time: `1:45.79`.

Classification: `grammargen-metadata` or `parser-machinery` in generated Rust
match/let-guard handling. This is the next Rust evidence bucket; it is not a
reason to add Rust-specific normalization.

## Rust Minimal Match-Arm Diagnostic

`TestRustGeneratedMatchArmDiagnostic` is a skipped-by-default witness for the
same generated Rust failure family. Enable it only when diagnosing the Rust
match-arm bucket:

```sh
GTS_RUST_GENERATED_MATCH_DIAGNOSTIC=1 go test ./grammargen -run '^TestRustGeneratedMatchArmDiagnostic$' -count=1 -v
```

The test narrows the canary toward the first `match x { ... }` block and
compares generated Rust against the checked-in Rust reference. Normal test runs
skip it so the branch can carry the witness without making CI red.

## Next Critical Path

1. Keep the signature-backed recovery work moving through generated grammars.
2. For Rust, reduce `TestRustCorpusMatchLetGuardParity` to the first generated
   grammar divergence, then decide whether the missing fact belongs in
   grammargen metadata or parser materialization.
3. For Java, keep a separate token-admission lane around multiline string
   escape boundaries.
4. Define one Docker correctness command and one perf smoke command for each
   high-value language before widening any scan.
5. Retire compatibility normalizers only when the generalized replacement is
   covered by a focused test and at least one real language witness.
