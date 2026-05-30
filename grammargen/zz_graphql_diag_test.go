package grammargen

import (
	"os"
	"sort"
	"strings"
	"testing"
)

func TestZZGraphqlStringValueReduceLookaheads(t *testing.T) {
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
	t.Logf("productions=%d symbols=%d tokenCount=%d externals=%d",
		len(ng.Productions), len(ng.Symbols), ng.TokenCount(), len(ng.ExternalSymbols))

	symName := func(id int) string {
		if id >= 0 && id < len(ng.Symbols) {
			return ng.Symbols[id].Name
		}
		return "?"
	}

	// Find symbol IDs of interest.
	want := map[string]int{}
	for i, s := range ng.Symbols {
		switch s.Name {
		case "string_value", "list_type", "value", "type", "description":
			want[s.Name] = i
		}
	}
	t.Logf("symbols of interest: %v", want)

	_, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTables: %v", err)
	}
	sets := ctx.itemSets
	t.Logf("total states: %d", len(sets))

	// Helper to render lookahead set as token names.
	laNames := func(b *bitset) []string {
		var out []string
		for tok := 0; tok < ng.TokenCount(); tok++ {
			if b.contains(tok) {
				out = append(out, symName(tok))
			}
		}
		sort.Strings(out)
		return out
	}

	// Find every COMPLETED item (dot == len(RHS)) whose LHS is string_value or list_type.
	for _, targetName := range []string{"string_value", "list_type"} {
		tid, ok := want[targetName]
		if !ok {
			continue
		}
		t.Logf("=== completed-reduce states for %s (sym %d) ===", targetName, tid)
		count := 0
		var union *bitset
		for si := range sets {
			set := &sets[si]
			for _, ce := range set.cores {
				prod := &ng.Productions[int(ce.prodIdx)]
				if prod.LHS == tid && int(ce.dot) == len(prod.RHS) {
					count++
					las := laNames(&ce.lookaheads)
					if union == nil {
						b := newBitset(ng.TokenCount())
						union = &b
					}
					union.unionWith(&ce.lookaheads)
					t.Logf("  state %d reduces %s->[%s] on LA{%s}",
						si, targetName, rhsStr(ng, prod, symName), strings.Join(las, ","))
				}
			}
		}
		if union != nil {
			t.Logf("  UNION of all %s reduce LAs (%d states): {%s}", targetName, count, strings.Join(laNames(union), ","))
		}
	}
}

func rhsStr(ng *NormalizedGrammar, prod *Production, symName func(int) string) string {
	var parts []string
	for _, s := range prod.RHS {
		parts = append(parts, symName(s))
	}
	return strings.Join(parts, " ")
}
