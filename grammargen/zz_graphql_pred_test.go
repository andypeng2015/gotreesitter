package grammargen

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestZZGraphqlPred(t *testing.T) {
	src, err := os.ReadFile("/tmp/grammar_parity/graphql/src/grammar.json")
	if err != nil {
		t.Skipf("grammar.json not available: %v", err)
	}
	g, err := ImportGrammarJSON(src)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	g.BinaryRepeatMode = true
	ng, err := Normalize(g)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	symName := func(id int) string {
		if id >= 0 && id < len(ng.Symbols) {
			return ng.Symbols[id].Name
		}
		return "?"
	}
	_, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	sets := ctx.itemSets

	// Build predecessor map.
	type pred struct {
		state int
		sym   int
	}
	preds := map[int][]pred{}
	for s := range sets {
		row := ctx.transitionRow(s)
		for _, tr := range row {
			preds[int(tr.target)] = append(preds[int(tr.target)], pred{state: s, sym: int(tr.sym)})
		}
	}

	dumpPreds := func(target int) {
		ps := preds[target]
		sort.Slice(ps, func(i, j int) bool { return ps[i].state < ps[j].state })
		var parts []string
		for _, p := range ps {
			parts = append(parts, fmt.Sprintf("%d-%s->", p.state, symName(p.sym)))
		}
		merged := ""
		if ctx.provenance != nil && ctx.provenance.isMerged(target) {
			merged = fmt.Sprintf(" [MERGED, %d origins]", len(ctx.provenance.origins(target)))
		}
		t.Logf("state %d preds: %s%s", target, strings.Join(parts, " "), merged)
	}

	// list_type completion states (446 = under-pop, 504 = full).
	for _, s := range []int{446, 504} {
		dumpPreds(s)
	}

	// Walk back from 446 to find the [ . type ] state and its preds.
	// Find the "[ . type ]" in-progress state that transitions to the type-goto leading to 446.
	// First find predecessors of 446 (reached via type-goto). Then preds of those (the "[" state).
	for _, p := range preds[446] {
		t.Logf("  446 reached from state %d on %s", p.state, symName(p.sym))
		dumpPreds(p.state)
	}
	for _, p := range preds[504] {
		t.Logf("  504 reached from state %d on %s", p.state, symName(p.sym))
		dumpPreds(p.state)
	}

	// Dump state 277 transitions (the [ . type ] state) — does it have multiple
	// transitions on `type`? And dump its preds (the `[`-shift states).
	t.Logf("=== state 277 transitions ===")
	for _, tr := range ctx.transitionRow(277) {
		t.Logf("  277 on %s (sym %d) -> %d", symName(int(tr.sym)), tr.sym, tr.target)
	}
	dumpPreds(277)

	// Are there actually multiple [ . type ] states? Find all states with a core
	// item list_type -> [ . type ].
	t.Logf("=== all [ . type ] in-progress states (list_type dot=1) ===")
	for si := range sets {
		for _, ce := range sets[si].cores {
			prod := &ng.Productions[int(ce.prodIdx)]
			if symName(prod.LHS) == "list_type" && int(ce.dot) == 1 {
				t.Logf("  state %d : list_type -> [ . type ]   (cores=%d)", si, len(sets[si].cores))
			}
		}
	}
}
