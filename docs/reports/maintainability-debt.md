# Maintainability Debt Inventory

This inventory lists the debt families that repeatedly slow Tier IV work. The
goal is to burn them down in safe slices while preserving parser/runtime and
grammar behavior.

## Result Compatibility And Normalizers

The `parser_result*.go` surface contains many language-local compatibility
paths that mutate child lists, fields, aliases, spans, or root envelopes to
match the C oracle. They are useful as regression locks, but they also make it
hard to tell whether a new fix belongs in generic result materialization,
recovery election, external scanner parity, or a local normalizer.

Safe first slice: inventory direct child/field mutations and require new work
to go through approved helper functions or clearly named compatibility files.
Do not rewrite behavior in this pass. The hygiene checker is warning-only by
default and is expected to surface existing direct mutations of private
`children`, `fieldIDs`, and `fieldSources` fields while this debt is being
classified.

## Retry And Cap Policy Tables

Retry behavior, stack caps, node budgets, and language-specific parse limits
are distributed across parser code, retry helpers, harness scripts, and TSV
notes. That makes it difficult to reproduce a Tier IV result and harder to
attribute a pass/fail change to a policy change.

Safe first slice: document the policy table inputs and add static checks that
spot new language-name switches in core parser files. Later slices can move
the policy into explicit tables with diagnostic output.

## `parseInternal` Complexity

`parseInternal` remains the highest-risk control point because it joins token
selection, GLR stack progression, recovery, retry, budget enforcement,
incremental reuse, and result construction. Optimizations or correctness fixes
inside this function can accidentally mix correctness and performance changes.

Safe first slice: keep new instrumentation warning-only and isolate attribution
into named phases: `Tree.Edit(edit)`, reuse-cursor/reuse-selection, and
reparse/rebuild. Defer behavior changes until a single-language parity witness
proves the needed phase.

## Core Parser Language-Name Switches

Language-name checks in core parser files are a maintainability smell because
they encode grammar behavior in generic runtime paths. Some existing cases may
be intentional compatibility gates, but new ones should be visible during
review.

Safe first slice: add a warning-mode hygiene checker that reports suspicious
`LanguageName`, `GrammarName`, or `lang == "..."` style checks in core parser
files. Warnings are informational by default and should not block the ratchet.

## External Scanner Ports

Scanner fidelity failures are a separate family from parser recovery. Layout
and markdown-class scanners need token-stream validation before tree-level
diffs are meaningful.

Safe first slice: maintain a list of scanner-port debt by grammar and require
focused scanner fixtures before broad corpus promotion. Keep scanner work out
of parser runtime changes unless token-stream parity is already proven.

## Dead-Code Caution

This repository has many harnesses, debug paths, and compatibility helpers that
look unused from one package view but are part of parity diagnosis. Dead-code
cleanup should use structural evidence and avoid removing diagnostic tools
during active Tier IV work.

Safe first slice: use `canopy` for dead-code or impact analysis and remove
only code with a clear owner, no harness reference, and no report dependency.

## Static Hygiene Checks

Warning-mode static checks are the first safe implementation slice because they
create visibility without changing parser behavior. The checks should stay
simple:

- flag direct child/field mutation in `parser_result*.go` outside approved
  helper files;
- flag language-name switches in core parser files;
- exit 0 by default so they can be run during inventory work without blocking
  existing ratchet regressions;
- cap example output by default while still reporting full summary counts;
- offer an explicit strict mode once a staged burn-down has a ratcheted
  baseline.

The initial checker lives beside the reports as documentation tooling rather
than production parser machinery.

### Current Hygiene Baseline

As of this inventory slice, `python3 docs/reports/maint_hygiene_check.py`
reports:

| Kind | Warnings |
| --- | ---: |
| `result-mutation` | 408 |
| `language-switch` | 1 |
| **Total** | **409** |

Top offender files by warning count:

| File | Warnings | Dominant kind |
| --- | ---: | --- |
| `parser_result_python.go` | 41 | `result-mutation` |
| `parser_result_authzed.go` | 30 | `result-mutation` |
| `parser_result_javascript_typescript.go` | 29 | `result-mutation` |
| `parser_result_typescript.go` | 26 | `result-mutation` |
| `parser_result_powershell.go` | 23 | `result-mutation` |
| `parser_result_scala_compilation.go` | 21 | `result-mutation` |
| `parser_result_c.go` | 20 | `result-mutation` |
| `parser_result_c_test.go` | 20 | `result-mutation` |
| `parser_result_csharp.go` | 20 | `result-mutation` |
| `parser_result_kotlin.go` | 13 | `result-mutation` |

The single `language-switch` warning is in `parser.go`.

### Burn-Down Process

Use the checker as an inventory and ratchet tool, not as parser machinery:

1. Keep the default checker warning-only so active parity work can inspect the
   debt without blocking unrelated changes.
2. Use `--json` in automation or local scripts to capture exact warning
   records, group by kind, and choose one small file or helper family per
   follow-up slice.
3. Prefer existing helper APIs such as `replaceNodeChildrenUnfielded`,
   `replaceChildRangeWithSingleNode`, `replaceChildRangeWithNodes`,
   `setNodeChildFieldDirect`, and `setNodeChildField` when the rewrite is
   mechanically identical.
4. Keep behavior changes out of hygiene slices. If a migration changes field
   overwrite semantics, parent/index updates, arena ownership, final-child refs,
   or compatibility ordering, leave it for a focused parser-result follow-up
   with language-specific validation.
5. After a slice reduces warnings, record the new count and run the narrowest
   relevant Go test or Docker parity target for the touched language. Do not
   mix correctness validation with performance validation.
6. Enable `--strict` only after a maintained baseline exists, then ratchet the
   allowed warning set down deliberately.

This first slice stayed tooling/docs-only. The top examples include broad
language compatibility normalizers that combine child replacement, field reset,
parent/index repair, arena cloning, and final-child reference behavior. Small
candidate field assignments exist in files such as `parser_result_bash.go` and
`parser_result_powershell_expr.go`, but changing them here would move the
requested 409-warning inventory baseline while this pass is establishing the
reporting surface. Those candidates should be handled in a follow-up slice with
focused language tests.
