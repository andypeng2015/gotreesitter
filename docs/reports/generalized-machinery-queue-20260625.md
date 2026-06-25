# Generalized Machinery Queue - 2026-06-25

## Evidence Snapshot

Latest relevant commits:

| Commit | Evidence |
| --- | --- |
| `02334368` | Real-token gap guard validation. Focused gap tests PASS, Java 25/25 PASS, JSON 6/6 PASS. Dart dirty run was 22/25, but the clean report-only baseline failed to build before parity, so the Dart caveat remains unresolved and likely tangled with separate Dart WIP. |
| `7e96ed03` | Worktree prune audit. The integrate-soon queue favors generalized runtime machinery and rejects per-language result normalizers for the 206-parity machinery goal. |
| `24a711b6` | Dart baseline comparison. Confirms the current Dart signal is not clean enough to use as a deciding regression gate until the separate Dart WIP is isolated. |

## Generalized Machinery Queue

| Priority | Candidate | Source | Action | Notes |
| --- | --- | --- | --- | --- |
| 1 | Faithful C-style GLR/recovery machinery | `/home/draco/work/gts-recovery-engine`, branch `ecosystem-parity` | Extract onto a clean branch and validate with focused parity before perf. | Highest-leverage generalized path. Start with multi-link GSS plus shift-gap guard behavior, keeping grammar-specific recovery out of the extraction. |
| 2 | Leaf/node interning hot cache | `/home/draco/work/gotreesitter-glr-node-interning`, commit `23f68a7c` | Land as a low-risk allocator/perf support change after correctness smoke coverage. | General runtime cache; should be easy to review independently of parity behavior. |
| 3 | Default-off GLR merge telemetry | `/home/draco/work/gotreesitter-merge-telemetry` | Land default-off observability support. | Enables repeatable attribution for merge/fanout experiments without changing default parse behavior. |
| 4 | Timeout/cancel inheritance | `.worktrees/fix-csharp-oom`, commit `f163960b` | Extract only generic `parseForRecovery` inheritance. | Do not port C# normalizer details; those are outside the generalized machinery goal. |

## Learn-Only / Reject

| Item | Disposition | Reason |
| --- | --- | --- |
| Spanless GSS negcache broad port | Learn-only; not near-term. | Conflicts across Java, Dart, and TLA+ make this too risky for an integrate-soon generalized port. |
| JS fork reduction direct patches | Reject direct patches. | Per-grammar changes do not serve the 206-parity machinery goal, but they motivate a generated repeat-boundary resolver. |
| Language result normalizers | Reject for this queue. | Useful for individual language deltas, but not generalized GLR/recovery machinery. |

## Next Experiments

1. Create a clean extraction branch for recovery-engine multi-link GSS plus shift-gap guard. Gate first with focused gap tests, then Java and JSON parity, and keep Dart as caveated until its clean baseline builds.
2. Land leaf/node interning hot cache and default-off merge telemetry as separate low-risk support commits. Verify with focused unit coverage and the standard perf trio only after correctness stays green.
3. Prototype a generated `GeneratedRepeatAux` / LR-conflict zero-progress repeat-boundary resolver. Treat JS fork reduction evidence as motivation, not as direct source patches.
