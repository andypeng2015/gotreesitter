# Worktree Prune Audit - 2026-06-25

Baseline: `/home/draco/work/gotreesitter` at `023343684c0ab4365689e01483305256f0edc68b` (`parity/c-oracle-clears`).

Commands run:

- `git worktree list --porcelain`
- `git worktree prune --dry-run --verbose`
- per-worktree `git status --porcelain --untracked-files=all`
- per-worktree `git rev-list --count <baseline>..HEAD`
- `git worktree prune --verbose`

## Action Taken

`git worktree prune --dry-run --verbose` reported no missing or stale worktree registrations. The subsequent `git worktree prune --verbose` completed as a no-op.

No worktree directories were removed. Every non-main worktree either has local dirty/untracked files, unique commits not reachable from the baseline, or both.

## Registered Worktrees

| Worktree | Branch | HEAD | Dirty files | Unique commits | Action |
| --- | --- | --- | ---: | ---: | --- |
| `/home/draco/work/gotreesitter` | `parity/c-oracle-clears` | `02334368` | 161 | 0 | Preserved main dirty worktree |
| `/home/draco/work/gotreesitter-glr-node-interning` | `feature/glr-node-interning` | `9ef6330a` | 0 | 2 | Preserved unique commits |
| `/home/draco/work/gotreesitter-js-fork-reduction` | `feature/js-fork-reduction` | `631f90a4` | 12 | 3 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter-js-residual` | `feature/js-fork-residual` | `a5f32d84` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-merge-telemetry` | `diagnostic/glr-merge-telemetry-20260624` | `b1aee1a2` | 0 | 3 | Preserved unique commits |
| `/home/draco/work/gotreesitter-tlaplus-glr-parser-isolate` | detached | `b1aee1a2` | 19 | 3 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter-tlaplus-glr-stack-isolate` | detached | `b1aee1a2` | 10 | 3 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter-tlaplus-remaining-runtime-isolate` | detached | `b1aee1a2` | 31 | 3 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter-tlaplus-runtime-isolate` | detached | `b1aee1a2` | 31 | 3 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter-tlaplus-runtime-noforest-isolate` | detached | `b1aee1a2` | 30 | 3 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter-typst-lift` | `codex/typst-tieriv-lift-20260613` | `8e5de2dc` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wgsl-lift` | `codex/wgsl-tieriv-lift-20260613` | `93c72a48` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-awk-current` | `wt/awk-final-lift-current` | `3500cecd` | 2 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-bicep-clean` | detached | `34a4ecb9` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-bitbake-tieriv-20260612` | `codex/bitbake-tieriv-20260612` | `8c02ee58` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-circom` | `wt/circom-iv-lift` | `5c3da7e2` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-csharp` | `wt/codex-csharp-tier-exit` | `62512a4a` | 4 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-csharp-tieriv-20260612` | `codex/csharp-tieriv-20260612` | `ee2f5779` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-finished-tree` | `wt/finished-tree-recovery` | `44ccad17` | 4 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-hlsl-tieriv` | detached | `b6f88d77` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-hurl-tieriv-20260612` | `codex/hurl-tieriv-20260612` | `b8d7fd1e` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-hyprlang` | `wt/codex-hyprlang-residual` | `0cd7a2c1` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-julia-tieriv-20260612` | `codex/julia-tieriv-20260612` | `6f325483` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-kotlin-final` | `wt/kotlin-final-lift` | `cb06adfc` | 4 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-regex-clean` | `wt/regex-clean-lift` | `5c3da7e2` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-regex-final` | `wt/regex-final-lift` | `6a957cfe` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter-wt-robot-tieriv-20260612` | `codex/robot-tieriv-20260612` | `3faf5b16` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-templ-tieriv-20260612` | `codex/templ-tieriv-20260612` | `18dbb487` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-typst-tieriv-20260612` | `codex/typst-tieriv-20260612` | `5c9807c9` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter-wt-wgsl-followup-20260612` | `codex/wgsl-followup-20260612` | `8183c085` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.claude/worktrees/agent-a61043882e8fb692b` | `worktree-agent-a61043882e8fb692b` | `0f7b1dc2` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter/.claude/worktrees/agent-aa8304bf475f2c99c` | `worktree-agent-aa8304bf475f2c99c` | `0f7b1dc2` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter/.claude/worktrees/agent-ab07027756d31be33` | `worktree-agent-ab07027756d31be33` | `0f7b1dc2` | 1 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter/.worktrees/code-understanding-sdk` | `feature/code-understanding-sdk` | `a8dc5dc6` | 12 | 0 | Preserved dirty files |
| `/home/draco/work/gotreesitter/.worktrees/fix-csharp-oom` | `fix/csharp-namespace-recovery-oom` | `34d08c48` | 0 | 2 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.worktrees/fix-go-tags` | `fix/go-inferred-tags-return-types` | `61d560fd` | 0 | 2 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.worktrees/fix-swift-comments` | `fix/swift-line-comment-lexer` | `a2a08183` | 0 | 2 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.worktrees/fix-test-expectations` | `fix/post-parity-test-expectations` | `90de5c93` | 0 | 1 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.worktrees/forest-fail-fast` | `perf/forest-fail-fast` | `4fd99449` | 2 | 82 | Preserved dirty + unique commits |
| `/home/draco/work/gotreesitter/.worktrees/lexer-codegen` | `perf/lexer-codegen` | `31cd89f2` | 0 | 3 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.worktrees/pr-ordering` | `integration/pr-101-104` | `cb46934b` | 0 | 8 | Preserved unique commits |
| `/home/draco/work/gotreesitter/.worktrees/pr104-check` | `integration/pr104-after-main` | `26991c56` | 0 | 2 | Preserved unique commits |
| `/home/draco/work/gts-perf-measurability` | `wt/aspen-perf` | `b5a7200f` | 3 | 1 | Preserved dirty + unique commits |
| `/home/draco/work/gts-recovery-engine` | `ecosystem-parity` | `597a696d` | 2 | 14 | Preserved dirty + unique commits |
| `/home/draco/work/gts-recovery-fanout` | `wt/redwood-recovery-fanout` | `0a03ff5f` | 2 | 0 | Preserved dirty files |
| `/home/draco/work/gts-recovery-stage1` | `recovery-port-stage1` | `beeeb5ed` | 3 | 0 | Preserved dirty files |
| `/home/draco/work/gts-shape-fixes` | `wt/willow-shape-fixes` | `7b1c1f94` | 2 | 0 | Preserved dirty files |
| `/home/draco/work/gts-unknowns` | `wt/elm-unknowns` | `18d9018d` | 5 | 3 | Preserved dirty + unique commits |
| `/tmp/gotreesitter-baseline` | detached | `3f73231a` | 5 | 0 | Preserved dirty files |
| `/tmp/gotreesitter-headprev` | detached | `e6b13146` | 5 | 0 | Preserved dirty files |
| `/tmp/gotreesitter-pr113` | `pr-113-iterative-normalize` | `03f06249` | 5 | 1 | Preserved dirty + unique commits |
| `/tmp/gts-baseline-comment-2447400` | detached | `f56abc23` | 10 | 0 | Preserved dirty files |

## Potentially Useful Unique Work

The following clean worktrees have unique commits and are good candidates for explicit review or integration rather than pruning:

- `/home/draco/work/gotreesitter-glr-node-interning` (`feature/glr-node-interning`): `9ef6330a optimize(parser): Reduce JS GLR fork pressure by resolving state 985`; also `23f68a7c` hot-path leaf intern cache.
- `/home/draco/work/gotreesitter-js-residual` (`feature/js-fork-residual`): `a5f32d84 fix(parser): Fix JS parser zero-progress repeat boundaries`.
- `/home/draco/work/gotreesitter-merge-telemetry` (`diagnostic/glr-merge-telemetry-20260624`): GLR merge telemetry instrumentation and conflict-resolution stats.
- `/home/draco/work/gotreesitter-typst-lift` (`codex/typst-tieriv-lift-20260613`): `8e5de2dc fix(parser): Fix Typst parser zero-width group commas`.
- `/home/draco/work/gotreesitter-wgsl-lift` (`codex/wgsl-tieriv-lift-20260613`): `93c72a48 improve(parser): Improve WGSL parser recovery and parity`.
- `/home/draco/work/gotreesitter-wt-bitbake-tieriv-20260612` (`codex/bitbake-tieriv-20260612`): `8c02ee58 add(parser): add bitbake addtask error unwrapping`.
- `/home/draco/work/gotreesitter-wt-csharp-tieriv-20260612` (`codex/csharp-tieriv-20260612`): `ee2f5779 fix(csharp): fix csharp parity for null, conditionals, generics, new`.
- `/home/draco/work/gotreesitter-wt-hurl-tieriv-20260612` (`codex/hurl-tieriv-20260612`): `b8d7fd1e fix(parser): fix Hurl trailing delimiter error normalization`.
- `/home/draco/work/gotreesitter-wt-hyprlang` (`wt/codex-hyprlang-residual`): `0cd7a2c1 fix(hyprlang): fix hyprlang parity with boolean normalization`.
- `/home/draco/work/gotreesitter-wt-julia-tieriv-20260612` (`codex/julia-tieriv-20260612`): `6f325483 add(julia): Add Julia return range recovery normalization`.
- `/home/draco/work/gotreesitter-wt-robot-tieriv-20260612` (`codex/robot-tieriv-20260612`): `3faf5b16 improve(robot parser): fix robot escaped nested variable error-shape residual`.
- `/home/draco/work/gotreesitter-wt-templ-tieriv-20260612` (`codex/templ-tieriv-20260612`): `18dbb487 improve(parser): Improve templ parity via argument normalization`.
- `/home/draco/work/gotreesitter-wt-typst-tieriv-20260612` (`codex/typst-tieriv-20260612`): `5c9807c9 add(typst): Add Typst nested list result normalization`.
- `/home/draco/work/gotreesitter-wt-wgsl-followup-20260612` (`codex/wgsl-followup-20260612`): `8183c085 fix(wgsl): normalize WGSL empty-return semicolon recovery`.
- `/home/draco/work/gotreesitter/.worktrees/fix-csharp-oom` (`fix/csharp-namespace-recovery-oom`): `f163960b fix(parser): Propagate timeout and cancellation to recovery parser`.
- `/home/draco/work/gotreesitter/.worktrees/fix-go-tags` (`fix/go-inferred-tags-return-types`): `371fb280 fix(grammars): fix Go tag inference capturing return types`.
- `/home/draco/work/gotreesitter/.worktrees/fix-swift-comments` (`fix/swift-line-comment-lexer`): `d5f32c56 improve(swift parser): improve swift top-level declaration recovery`.
- `/home/draco/work/gotreesitter/.worktrees/fix-test-expectations` (`fix/post-parity-test-expectations`): `90de5c93 update(parser): update optional chain and normalization tests`.
- `/home/draco/work/gotreesitter/.worktrees/lexer-codegen` (`perf/lexer-codegen`): lexgen switch-DFA lexer generation work.
- `/home/draco/work/gotreesitter/.worktrees/pr-ordering` (`integration/pr-101-104`): integration of PRs 101-104, including structural corpus parity and JS/TS/Python normalization fixes.
- `/home/draco/work/gotreesitter/.worktrees/pr104-check` (`integration/pr104-after-main`): PR 104 structural corpus parity check after main.

The following dirty worktrees also have unique commits and were preserved because they combine committed work with local WIP:

- `/home/draco/work/gotreesitter-js-fork-reduction` (`feature/js-fork-reduction`): JS fork pressure / comma-boundary fixes plus local changes.
- `/home/draco/work/gotreesitter-tlaplus-glr-parser-isolate` (detached): GLR merge telemetry commits plus parser-isolate WIP.
- `/home/draco/work/gotreesitter-tlaplus-glr-stack-isolate` (detached): GLR merge telemetry commits plus stack-isolate WIP.
- `/home/draco/work/gotreesitter-tlaplus-remaining-runtime-isolate` (detached): GLR merge telemetry commits plus runtime WIP.
- `/home/draco/work/gotreesitter-tlaplus-runtime-isolate` (detached): GLR merge telemetry commits plus runtime WIP.
- `/home/draco/work/gotreesitter-tlaplus-runtime-noforest-isolate` (detached): GLR merge telemetry commits plus no-forest runtime WIP.
- `/home/draco/work/gotreesitter/.worktrees/forest-fail-fast` (`perf/forest-fail-fast`): large forest fast-path/perf stack with 82 unique commits plus local parser edits.
- `/home/draco/work/gts-perf-measurability` (`wt/aspen-perf`): `b5a7200f fix(parser): cap crystal GLR survivor stacks to unblock measurement` plus perf probe/profile WIP.
- `/home/draco/work/gts-recovery-engine` (`ecosystem-parity`): 14 unique recovery/GLR commits plus local harness diagnostics.
- `/home/draco/work/gts-unknowns` (`wt/elm-unknowns`): HTTP/Kotlin/Dhall parity commits plus INI and harness WIP.
- `/tmp/gotreesitter-pr113` (`pr-113-iterative-normalize`): `03f06249 fix: iterative DFS in normalizeGoDotLeafChildren (issue #110)` plus deleted bench files.

## Dirty Work Preserved Without Unique Commits

These worktrees had no commits unreachable from the baseline, but they had modified or untracked files and were therefore not removed:

- `/home/draco/work/gotreesitter-wt-awk-current` (`wt/awk-final-lift-current`)
- `/home/draco/work/gotreesitter-wt-bicep-clean` (detached)
- `/home/draco/work/gotreesitter-wt-circom` (`wt/circom-iv-lift`)
- `/home/draco/work/gotreesitter-wt-csharp` (`wt/codex-csharp-tier-exit`)
- `/home/draco/work/gotreesitter-wt-finished-tree` (`wt/finished-tree-recovery`)
- `/home/draco/work/gotreesitter-wt-hlsl-tieriv` (detached)
- `/home/draco/work/gotreesitter-wt-kotlin-final` (`wt/kotlin-final-lift`)
- `/home/draco/work/gotreesitter-wt-regex-clean` (`wt/regex-clean-lift`)
- `/home/draco/work/gotreesitter-wt-regex-final` (`wt/regex-final-lift`)
- `/home/draco/work/gotreesitter/.claude/worktrees/agent-a61043882e8fb692b` (`worktree-agent-a61043882e8fb692b`)
- `/home/draco/work/gotreesitter/.claude/worktrees/agent-aa8304bf475f2c99c` (`worktree-agent-aa8304bf475f2c99c`)
- `/home/draco/work/gotreesitter/.claude/worktrees/agent-ab07027756d31be33` (`worktree-agent-ab07027756d31be33`)
- `/home/draco/work/gotreesitter/.worktrees/code-understanding-sdk` (`feature/code-understanding-sdk`)
- `/home/draco/work/gts-recovery-fanout` (`wt/redwood-recovery-fanout`)
- `/home/draco/work/gts-recovery-stage1` (`recovery-port-stage1`)
- `/home/draco/work/gts-shape-fixes` (`wt/willow-shape-fixes`)
- `/tmp/gotreesitter-baseline` (detached)
- `/tmp/gotreesitter-headprev` (detached)
- `/tmp/gts-baseline-comment-2447400` (detached)

## Conclusion

There was nothing safe to remove under the requested policy. The durable win is the audit itself: it confirms there are no stale registrations to prune and identifies the worktrees that contain potentially useful committed or local WIP.
