# GSS-Forest GLR — evaluation (is it the fix for the merge-blowup cluster?)

_Evaluated 2026-06-08 against the glr_merge-dominant cluster (ledger, authzed, json5,
gitignore, make, nginx). Commit `0f7b1dc2`._

## Verdict

**Yes — the forest is the architecturally-correct fix for the GLR-merge blowup, and it is
far along.** Where it dispatches it produces byte-clean trees and collapses the blowup by
5–858×. The thing gating broad rollout is **error recovery** (its absence caps the dispatch
rate), not correctness or the merge design.

## What it is (architecture)

A faithful tree-sitter-C GLR in 1457 lines (`glr_forest.go`), default-on (`GOT_GLR_FOREST`),
with a **safe production fallback** on any decline (failure / error node / truncation /
incomplete), so a dispatched language can never regress the cases it declines.

Three stages:
1. **Coalesce** (`coalesceForest`) — merge GSS nodes by `(state, byteOffset)` key into one
   node carrying one `gssLink` per surviving parse (= C's `StackNode.links[]`), **with no
   deep-equivalence walk.** This is exactly what eliminates the production parser's #1 cost.
2. **Reduce-over-forest** (`reduceOverForest`) — enumerate length-`childCount` paths through
   the links (bounded DFS), capped by `forestReduceStepCap`.
3. **Disambiguate** — dynamic-precedence (highest cumulative `score`) resolved cheaply during
   coalesce via link dedup on `(prev, symbol, span)`.

This directly answers the profiled production bottleneck: production spends **43% of CPU in
`lookupNodeEquivCache`** doing deep node-equivalence to confirm 175K forks don't merge (ledger,
932 bytes). The forest's Stage-1 coalesce merges on scalars — that whole cost class disappears.

## Empirical results (forest vs production, this machine)

| Grammar | Forest | Production | Speedup | Byte-clean where dispatched |
|---|--:|--:|--:|---|
| gitignore | 1.48 ms | 1269 ms | **858×** | ✓ (the one "diverge" is prod *truncating* at 3018B while forest completes 110KB) |
| ledger | 0.96 ms | 161 ms | **169×** | ✓ 2/2 dispatched clean |
| json5 | 0.49 ms | 6.1 ms | 12.5× | ✓ |
| authzed | 0.25 ms | 1.19 ms | 4.8× | ✓ |
| make | 0.x ms (2/15 dispatched) | 13,991 ms | huge | ✓ 2/2 clean (most files decline — no recovery) |
| nginx | dispatched 2/6 | 6.3 ms | — | ✗ **2/2 dispatched DIVERGED — a genuine forest bug** |
| authzed | 0/15 dispatched in corpus dir | 1.19 ms | 4.8× (single .zed file) | corpus dir is stray .md files; real .zed dispatches clean |

Byte-range parity (15 smallest files each) refines the picture:
- **Clean where dispatched:** ledger 2/2, make 2/2, gitignore 7/8, json5 3/4. The handful of
  byte "divergences" on ledger/gitignore are **production truncating** a stray large file
  (Cargo.lock, package-lock.json) while the forest completes it — forest is *more* correct, so
  forest-vs-production is a confounded gate for these already-parity-blocked grammars.
- **Genuine forest bug: nginx** — both dispatched files produced byte-different trees from
  production. nginx must NOT be promoted until that divergence is fixed. The byte-range gate
  caught this where an s-expr-only check would not have.

## The two real limitations

1. **No error recovery (the dominant gate).** The forest declines any file with an error node
   or incompleteness. On real corpora that's most files for some grammars (ledger dispatched
   only 2/15; gitignore 8/14). Dispatch rate — not correctness — is what currently caps the win.
   This is the single highest-leverage investment: forest error recovery would convert the low
   dispatch rates into high ones across the whole cluster.
2. **Reduce-over-forest can relocate the blowup.** For exponentially-ambiguous grammars
   (haskell), Stage-2 path enumeration explodes and `forestReduceStepCap` makes it decline.
   The coalesce fixes the *merge* cost; it does not fix grammars whose ambiguity is in the
   *number of reduce paths*. Those need fork-reduction or grammar fixes regardless.

## Current rollout state

Gated to 8 verified-clean languages (`languageWantsForest`: bash, erlang, cmake, css, scss,
awk, javascript, c_sharp) with measured 3–803× wins. Promotion gate is
`cgo_harness/forest_corpus_parity_test.go` (forest-vs-production byte-range). php is byte-clean
but held out on a net-wall *loss* (1/3 dispatch; the 2/3 failed-forest attempts cost more than
the third saves) — the same dispatch-rate ceiling.

## Recommendations (in priority order)

1. **Invest in forest error recovery.** It is the rate-limiter for the entire cluster. Even a
   minimal "use the forest for the clean prefix, recover at the first error" would lift dispatch
   rates and let ledger/json5/authzed/gitignore promote for 5–858× wins.
2. **Gate promotion against the C oracle, not production**, for parity-blocked grammars
   (`forest_oracle_parity_test.go`). Forest-vs-production is confounded when production truncates.
3. **Don't pursue per-grammar fork-reduction for the coalesce-amenable cluster** — the forest
   already solves their merge cost more generally and more safely.
4. Keep the dispatch-rate / wall A-B discipline already encoded in the code comments: only
   promote when on/off is a net wall win on the real corpus.
