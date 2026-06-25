# The GSS-Forest Solution — design & what's needed to support it

_Deep design, 2026-06-08. Synthesizes: the forest algorithm read, the production
error-recovery spec, and an honest lock-filtered dispatch baseline on the
merge-blowup cluster._

## Executive summary

The GSS-forest is the **architecturally-correct fix** for the #1 perf gap vs C: the
production parser spends **43% of CPU in `lookupNodeEquivCache`** doing deep
node-equivalence to dedup forks (profiled on ledger: 175K stacks / 932 bytes, 99.98%
of comparisons prove inequality). The forest eliminates that entire class by
coalescing GSS nodes on `(state, byteOffset)` + links (tree-sitter C's model), with
**no deep-equivalence walk anywhere**.

It is mature (1457 lines, 3-stage C-faithful, safe production fallback, 36–803× on 8
verified languages) and the honest baseline shows the path forward is **four concrete,
addressable gates** — not one monolith. **Three cluster grammars are promotable today**
for pure speedup with zero behavior change.

## Architecture (recap)

Three stages in `glr_forest.go`:
1. **Coalesce** (`coalesceForest`) — one node per `(state, byteOffset)`, ≤8 links
   (`forestMaxLinksPerNode`), dedup competing reductions by `(prev, symbol, span)` keeping
   higher dynamic-precedence `score`. No deep-equivalence walk.
2. **Reduce-over-DAG** (`reduceOverForest`/`forestReducer`) — bounded DFS enumerating
   length-`childCount` paths through links; sticky `capped` at `forestReduceStepCap=65536`
   (declines on exponential-ambiguity grammars → production fallback; this is the haskell
   "relocation" guard).
3. **Finalize** (`collectForestRootAndExtras` + `bestLink`) — at EOF, walk down the
   highest-`score` link from the accept node to the start-symbol root; fold extras.

Key latent asset: nodes already carry an **`errorCost`** field threaded through
`coalesceForest`/`slab.alloc`/`forestCoalesceWouldDropForCap` — **the infrastructure for
tree-sitter C's error_cost recovery exists but nothing generates error-cost alternatives
yet.** That is the seam the recovery feature plugs into.

## The honest dispatch baseline (lock-filtered real corpora)

Earlier dispatch numbers were **junk-confounded** — ad-hoc "smallest files" selection
grabbed README/Cargo.lock/`.sh` files. Filtering by the lock's per-language extensions
(the same matcher `build_real_corpus` uses) gives the real picture. Forest vs production,
byte-range comparison:

| Grammar | clean | diverge | declines | Class |
|---|--:|--:|---|---|
| nix | 29/30 | 0 | 1 no-shift | **promote-now** |
| make | 18/19 | 0 | 1 no-shift | **promote-now** (production was ~14 s/file) |
| gitignore | 21/30 | 0 | 0 | **promote-now** |
| authzed | 7/30 | 0 | 18 no-shift, 5 eof | recovery-gated |
| org | 9/30 | **18** | 2 no-shift | divergence-bug |
| gitattributes | 0/16 | **10** | 0 | divergence-bug |
| dockerfile | 0/30 | 0 | 30 no-shift | completeness-gap |
| dtd | 0/6 | 1 | 5 no-shift | thin/mixed |

The four classes below are the entire "what's needed."

## Gate 1 — Promote-now (nix, make, gitignore)

Zero divergence, high clean dispatch. Because the gate is **forest-vs-production**, a
match means **the forest tree is identical to today's production tree** — promoting these
is a **pure speedup with no behavior change**, even though all three are parity-blocked
vs C (the forest diverges from C exactly as production does — unchanged).

**Needed:** run the full-corpus byte-range gate (`TestForestCorpusParity` with
`GTS_FOREST_LANGS=nix,make,gitignore`, in Docker), then add them to `languageWantsForest`.
Low-risk (the production fallback already protects declined files). This is the immediate
high-value action; `make` alone goes from ~14 s to sub-ms where it dispatches.

## Gate 2 — Error recovery (the core feature; unlocks authzed, ledger, json5, …)

The recovery-gated grammars are **clean where they dispatch** and decline on
`no-shift-death`. The naive error-skip prototype failed because it kept the frontier in
the same state, so a no-shift-death just became `eof-no-accept` later — the error leaf
never integrated into an accepting reduction.

**The right design is tree-sitter C's error_cost recovery, not production's multi-stack
cascade.** Production's recovery (relex → kill-stuck-stack-if-others-survive →
unique-missing-shift → grammar-recover → in-place-extended error leaf, with resurrection
gated behind a wide-stack *retry pass*) is an artifact of its multi-stack-kill/resurrect
dynamics, which the single-frontier forest does not have. The forest is C-faithful, so it
should match **C's** canonical recovery (and for these parity-blocked grammars, that is
*more* correct than matching production anyway).

Design, using the existing machinery:
1. **On a stuck frontier** (no shift for the token at any node): for each frontier node,
   consult the production recover-action table (`recoverActionForState` /
   `buildRecoverActionsByState`, already built) to find the nearest **recover-capable
   state**, pop to it, and coalesce an `errorSymbol`-leaf alternative there with
   `errorCost += tokenWidth`. Popping to a recover state is what lets reductions continue
   and the parse reach accept — the piece the naive skip lacked.
2. **Extend error leaves in place** with production's `pushOrExtendErrorNode` semantics
   (same `parseState`, forward bytes, leaf-only) so consecutive bad tokens group into one
   error node.
3. **Make selection error_cost-aware**: `bestLink` and `collectForestRootAndExtras`
   currently pick by `score` only — they must prefer **lowest `errorCost`** first, then
   `score`. The error-recovery alternatives then lose to any clean path and win only when
   nothing clean survives, matching C's min-cost selection.
4. **Bound it** with an error_cost ceiling (C prunes high-cost recovery) so a pathological
   file declines to production rather than exploring exponential error paths.
5. **Synthetic-root retag**: reproduce the production rule (root → `errorSymbol` when
   multiple top fragments carry an error and the language isn't dart-complete/sql/swift/
   python-complete; else `expectedRootSymbol`) in `finalizeForestRoot`.

This is the largest piece, but it is **bounded** — it reuses the existing recover-action
table and the existing `errorCost` plumbing; it does **not** require reproducing
production's retry/widening architecture (the forest has no merge blowup to widen around).

## Gate 3 — Divergence bugs (org, gitattributes, nginx, dtd)

The forest produces a clean-but-**byte-wrong** tree (the one failure mode the production
fallback cannot catch — hence the byte-range gate exists). Concrete root-causes:
- **org**: Stage-3 selection mismatch — production yields `headline:863-886`, forest
  yields `section:863-1151` at the same position. The forest's highest-`score` `bestLink`
  selects a different valid parse (section-nesting vs flat headline). Fix is in
  **dynamic-precedence/score parity**: the forest's per-link `score` accumulation or the
  `bestLink` tie-break diverges from production's `stackCompareMerge`/dynamic_precedence.
- **gitattributes** (10/10 diverge), **nginx** (2/2), **dtd** (1): same family — verify
  each is score-selection vs a genuine reduce/coalesce structural bug. These **block
  promotion** and need per-grammar byte-divergence root-causing (the
  `TestForestFirstIssue` diagnostic added here pinpoints the first differing node).

## Gate 4 — Completeness gap (dockerfile)

100% `no-shift-death` and dockerfile **has an external scanner**. The forest drives the
production token source (`acquireDFATokenSource`) but sets scanner state per *step* from
the frontier (`SetGLRStates`/`SetParserState`) — **external-scanner checkpoint/restore
across the coalesced frontier likely differs from production's per-stack handling**, so
the scanner never emits the tokens dockerfile needs and every file dies at the first
shift. Investigate `updateCurrentExternalTokenCheckpoint` / `recordCurrentExternalLeafCheckpoint`
in the forest loop vs production. This is a forest×external-scanner integration gap; other
scanner grammars (authzed uses one too and dispatches 7/30) may share partial symptoms.

## Supporting infrastructure

1. **Corpus fixes (prerequisite for honest gating):** `ledger`'s lock path is **broken**
   (pins a stale root-relative `non-profit-test-data.ledger`; the real file is in
   `contrib/.../tests/`). `json5`/`nginx`/`ron` are thin (1–3 files) — broaden the lock
   extension (`json5` → `.json5` gives 3) or repick the source repo (nginx's source is
   oddly `elm/compiler`). **All forest measurement must use the lock filter** — the
   confounded baseline nearly buried `make` (actually 95% clean) as "declines".
2. **Gate against the C oracle, not production**, for parity-blocked grammars
   (`forest_oracle_parity_test.go`) — production is a truncating/diverging baseline. (For
   the promote-now no-regression case, forest-vs-production is the correct gate.)
3. **Incremental path:** `languageAllowsForestIncrementalPath` (5 langs) gates whether a
   forest tree may feed the incremental parser; extend per promoted grammar after the
   edited-corpus matrix proves it.
4. **Net-wall A/B discipline:** keep the existing rule — promote only when `GOT_GLR_FOREST`
   on/off is a net wall *win* on the real corpus (php is held out at 1/3 dispatch because
   the 2/3 failed-forest attempts cost more than the third saves). The promote-now three
   clear this; recovery lifts the recovery-gated set over the bar.

## Roadmap (priority order)

1. **Promote nix, make, gitignore** — verify full-corpus byte-range gate, add to
   `languageWantsForest`. Immediate, low-risk, large win (esp. make).
2. **Implement error_cost recovery** (Gate 2) — the core feature; unlocks the
   recovery-gated tail (authzed/ledger/json5/…). Reuses recover-action table + `errorCost`.
3. **Fix the divergence bugs** (Gate 3) — start with org's score-selection parity; it
   likely generalizes to gitattributes/nginx/dtd.
4. **Forest × external-scanner** (Gate 4, dockerfile).
5. **Infra**: fix the ledger lock path; wire the C-oracle gate; extend the incremental
   allowlist as grammars promote.

## Execution log (2026-06-08)

**Gate 1 — SHIPPED.** `nix` and `gitignore` added to `languageWantsForest` (default-on)
after the full-corpus byte-range gate: **0 divergence** across all dispatched files
(gitignore 33/44, nix 635/703) AND net-wall **wins** (nix 5.2×, gitignore 1.7×, trees
byte-identical via the normal `Parse()` path). **`make` held back**: byte-range clean
(19/20) but net-wall **neutral** (1.0×) on real git/git Makefiles — its few expensive
files decline (no-shift), and typical Makefiles parse fast in production too, so the
forest adds little. make promotes once recovery (Gate 2) lands.

**Gate 2 — IMPLEMENTED, flag-gated (`GOT_GLR_FOREST_RECOVER`), NOT yet shippable.**
Replaced naive error-skip with **recover-action error_cost recovery** (each stuck frontier
node uses production's `recoverActionForState` to pop to a recover-capable state, coalescing
an `errorSymbol` leaf with `errorCost += tokenWidth`) **plus a synthetic error root at the
`eof` branch** (`collectForestErrorRoot`: pick the actor that consumed the most input at
lowest error cost, materialize its bestLink chain as top-level fragments, wrap in the
expected root retagged to `errorSymbol`). This was the missing piece — it converts the
`eof-no-accept` stalls into completed error trees. Result: recovery now **completes** the
parse for the recovery-gated set and **matches production on the majority** (authzed
**81/110 byte-identical**, incl. error files). But **29/110 authzed files still diverge**
in error-node placement, so it is **not production-exact across a full corpus** and ships
to **no grammar by default** (`languageWantsForestRecover` returns false; the global flag
drives it for refinement). **Remaining work:** close the error-node-placement gap (the 29
files) — likely the in-place error-leaf extension keying and the synthetic-root fragment
selection vs production's exact `pushOrExtendErrorNode`/result-selection rules.

**Gate 3 — root-caused (org), not fixed.** The forest's clean-but-wrong tree on `org`
(prod `headline` vs forest `section`) is a **selection-model divergence**: forest coalesce
re-entrancy *builds* a `drawer`/`section` alternative (dynamic_precedence=1) that
production's aggressively-culled forward stack progression never materializes, and
`bestLink`'s highest-score rule then commits to it — whereas production's finalists are both
score-0 and decided by `branchOrder` (earliest fork), a tie-break the forest's `gssLink`
lacks. Fixing it means reconciling the forest's score-dominant selection with production's
`branchOrder`-decided, prec-blind resolution (or bounding forest exploration to production's
reachable-state set) — delicate, risks the 10 shipped languages. org/gitattributes/nginx
stay out. The byte-range gate is exactly what catches this class.

## Unifying insight (2026-06-08): the remaining gaps are ONE root cause

The authzed recovery 29-file gap (Gate 2) and the org/gitattributes/nginx clean
divergences (Gate 3) are **the same selection-model divergence**. Inspecting the authzed
divergences: the forest recovers/reduces into *different valid constructs* than production
(`expected.zed`: forest keeps `definition:0-18` where production wraps `ERROR:0-92`;
`basic.zed`: forest `block_repeat1:70-182` vs production `relation:70-118`) — not an
error-node-grouping bug. Both classes stem from: forest coalesce re-entrancy **explores
more actor combinations** than production's aggressively-culled forward progression
(`maxGLRStacks=8`, `maxStacksPerMergeKey=6`), then `bestLink`'s highest-`score` rule
**commits to alternatives production prunes/never builds**, with no `branchOrder`
(earliest-fork) tie-break.

**So one fix — reconciling the forest's selection with production's culled,
`branchOrder`-decided resolution (or bounding forest exploration to production's
reachable-state set) — closes BOTH the recovery gap and the divergence-bug cluster.** It
is deep and risks the 10 shipped languages (the `bestLink`/coalesce path is shared), so it
is a deliberate initiative, not a drive-by. Until then: nix/gitignore ship (verified
clean), recovery stays flag-gated, and org/gitattributes/nginx/authzed-recovery stay out.

## Validation note

All measurement here is host-only with capped memory budget (`GOT_PARSE_MEMORY_BUDGET_MB`);
the experimental recovery scaffolding in `glr_forest.go` is flag-gated OFF
(`GOT_GLR_FOREST_RECOVER`) and the decline-reason/diagnostic code is no-op in the default
path. Heavy full-corpus parity gates run in Docker per the repo's testing discipline.
