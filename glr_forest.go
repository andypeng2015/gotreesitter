package gotreesitter

import "os"

// GSS-FOREST REWRITE (perf/glr-gss-forest) — the only safe cut at the #1
// machinery gap vs tree-sitter C: deep stack-merge node-equivalence is ~46% of
// a fork-heavy parse, because we materialize one tree per stack and must
// deep-compare to dedup. tree-sitter C never compares: its graph-structured
// stack coalesces versions by (state, position) and keeps subtree alternatives
// as forest LINKS (lib/src/stack.c ts_stack_can_merge = 4 scalars + add_link),
// collapsing the forest at finalization by dynamic_precedence/error_cost.
//
// We already have (state, position) merge keys (mergeKeyForStack) and the
// disambiguator (stackCompareMerge: accepted > error-rank > score > shifted).
// The missing piece is a multi-link GSS node. This file builds it behind a flag
// so the default (table-driven dedup) path is untouched until the forest path
// reaches byte-for-byte parity.
//
// STAGED PLAN (see project_glr_merge_design memory + the gss-forest-rewrite
// spore). Stages 1 and 2 are coupled — coalesce produces alternatives that only
// parse correctly once reduce traverses all of them, so parity is expected only
// when BOTH land:
//
//	Stage 0  DONE — instrument. dedup fires 0.2%, fan-out bounded 12-20, so the
//	         forest is narrow (cheap) and the 46% is genuinely wasted compares.
//	Stage 1  DAG node + coalesce-by-(state,position) on push (this file).
//	Stage 2  reduce-over-DAG: enumerate all length-N paths through the links
//	         (C ts_stack_pop_count). The crux; needs error_cost/version bounding.
//	Stage 3  forest finalization: pick per node by score, matching tree-sitter's
//	         dynamic_precedence-then-first-match selection for byte-identical out.

// glrForestEnabled gates the experimental GSS-forest reduce path. Off by default
// (read once from GOT_GLR_FOREST=1) so production keeps the proven dedup path.
var glrForestEnabled = os.Getenv("GOT_GLR_FOREST") == "1"

// SetGLRForestEnabled toggles the GSS-forest path at runtime (tests/benchmarks).
func SetGLRForestEnabled(on bool) { glrForestEnabled = on }

// gssLink is one alternative predecessor in the forest DAG: the subtree consumed
// to reach this node, and the prior node it was consumed from. A coalesced node
// (one per (state, position)) carries one link per surviving parse that reached
// it — exactly tree-sitter C's StackNode.links[].
type gssLink struct {
	prev    *gssForestNode
	subtree stackEntry
}

// gssForestNode is a coalesced graph-structured-stack node: all parses that
// reach (state, byteOffset) share this single node; their differing histories
// are the links. This replaces the singly-linked gssNode{entry, prev} chain in
// the forest path. score carries the best accumulated dynamic precedence among
// the links for finalization tie-breaks.
type gssForestNode struct {
	state      StateID
	byteOffset uint32
	links      []gssLink
	score      int
	errorCost  int
}

// coalesceForest merges a parse reaching (state, byteOffset) with subtree `entry`
// from predecessor `prev` into the forest: if a node already exists for that
// (state, byteOffset) it gains a link (O(1), no deep-compare — the heart of the
// win); otherwise a new node is created. `index` maps (state, byteOffset) to the
// node so coalescing is a map lookup, not a stack scan.
//
// Stage 1 scaffold: builds the DAG. Correct trees require Stage 2 (reduce walks
// every link); until then this is exercised only under the flag + parity gate.
func coalesceForest(index map[gssForestKey]*gssForestNode, state StateID, byteOffset uint32, prev *gssForestNode, entry stackEntry, score, errorCost int) *gssForestNode {
	key := gssForestKey{state: state, byteOffset: byteOffset}
	node := index[key]
	if node == nil {
		node = &gssForestNode{state: state, byteOffset: byteOffset, score: score, errorCost: errorCost}
		index[key] = node
	} else {
		// Keep the better disambiguator at the coalesced node (mirrors how
		// stackCompareMerge ranks; full per-link selection is Stage 3).
		if errorCost < node.errorCost || (errorCost == node.errorCost && score > node.score) {
			node.score, node.errorCost = score, errorCost
		}
	}
	node.links = append(node.links, gssLink{prev: prev, subtree: entry})
	return node
}

type gssForestKey struct {
	state      StateID
	byteOffset uint32
}

// reduceOverForest enumerates every length-childCount path of subtrees ending at
// `node`, invoking visit with the collected children (most-recent-last) and the
// predecessor node the reduction pops back to. This is Stage 2 — the crux that
// replaces single-chain reduce with DAG traversal so a coalesced node's multiple
// histories all reduce. NOT YET IMPLEMENTED: returns without visiting so the
// flag path is inert until this lands. The walk is a bounded DFS over links
// (depth == childCount); C bounds the analogous walk by error_cost + version cap
// to keep ambiguous grammars from exploding.
func reduceOverForest(node *gssForestNode, childCount int, visit func(children []stackEntry, popTo *gssForestNode)) {
	_ = node
	_ = childCount
	_ = visit
	// TODO(Stage 2): DFS over node.links to depth childCount, accumulate the
	// subtree on each link, and call visit at depth 0 with popTo = the link's
	// prev at that depth. Dedup identical (popTo,children) results so the
	// reduced node coalesces too. This is where the 46% deep-compare cost is
	// eliminated — no equivalence walk, just link traversal + map coalescing.
}
