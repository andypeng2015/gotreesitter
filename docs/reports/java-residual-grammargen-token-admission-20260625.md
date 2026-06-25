# Java residual grammargen token admission report, 2026-06-25

## Scope

This report records the Java residual classification from clean build-baseline
commit `0e78f674`:

```text
0e78f674 add(docs): Add Java parity diagnostic report
```

The source classification came from the clean worktree at:

```text
/home/draco/work/gotreesitter-build-baseline
```

This is a documentation-only report. It does not change parser, lexer,
grammargen, Java grammar, or harness source.

## Residual state

After the clean build-baseline plus the gap guard, Java remains stable at:

```text
no-error 23/25
sexpr parity 23/25
deep parity 23/25
```

The two residual failures are:

- sample 6: `literals.txt`
- sample 15: `expressions.txt`

Both failures produce generated `ERROR` nodes while the clean C reference parse
recognizes the same regions as `escape_sequence` inside Java text-block or
string-template string literal repeat bodies.

Java has no external scanner in this grammar revision, so external scanner
state or provenance is not part of this residual.

## Classification

This residual is not the stale real-token gap class.

It is also not currently classified as C-recovery, result selection, or
external scanner/provenance. The C reference parse is clean, the generated parse
is the side producing `ERROR`, and the failures are localized to repeated
multiline-string bodies around backslash-heavy boundaries.

Best current family:

```text
grammargen DFA/LR token admission or precedence at repeated multiline-string
boundaries
```

The repeated body must switch from a broad multiline string fragment token to an
escape token at a backslash boundary. The reference admits `escape_sequence`;
the generated parser leaves the boundary as `ERROR`.

## Next generalized experiment

Avoid a Java-specific fix until the admission behavior is reproduced in a
smaller generalized case.

Build a reduced grammar with:

- a broad fragment token,
- an immediate or high-precedence escape token,
- both alternatives inside a repeat body.

Then compare:

- generated DFA accept candidates at the backslash boundary,
- LR valid-symbol filtering for the same byte position,
- the selected generated token against the C reference token behavior.

The experiment should identify whether the fault is in DFA accept precedence,
LR valid-symbol admission at the repeat boundary, or repeat-body handling of the
hidden escape rule that becomes visible as `escape_sequence`.
