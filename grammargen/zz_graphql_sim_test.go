package grammargen

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// Full LR(1) simulation at the item-set level (shift + reduce) to find the
// exact state/token where a parse dies, using the in-memory itemSets + LALR
// reduce lookaheads (no DFA / runtime table involved).
func TestZZGraphqlSim(t *testing.T) {
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
	termID := func(name string) int {
		for i := 0; i < ng.TokenCount(); i++ {
			if ng.Symbols[i].Name == name {
				return i
			}
		}
		t.Fatalf("no terminal %q", name)
		return -1
	}
	_, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	sets := ctx.itemSets

	laNames := func(b *bitset) string {
		var out []string
		for tok := 0; tok < ng.TokenCount(); tok++ {
			if b.contains(tok) {
				out = append(out, symName(tok))
			}
		}
		sort.Strings(out)
		return strings.Join(out, ",")
	}

	// Simulate.
	sim := func(name string, toks []int) {
		t.Logf("==== SIM %s ====", name)
		stack := []int{0} // state stack
		ip := 0
		step := 0
		for {
			step++
			if step > 200 {
				t.Logf("  step cap")
				return
			}
			cur := stack[len(stack)-1]
			var la int
			if ip < len(toks) {
				la = toks[ip]
			} else {
				la = 0 // $end
			}
			set := &sets[cur]
			// Look for shift on la.
			if tgt, ok := ctx.transitionTarget(cur, la); ok {
				stack = append(stack, tgt)
				ip++
				continue
			}
			// Look for a reduce: completed item whose lookahead includes la.
			var rprod = -1
			for _, ce := range set.cores {
				prod := &ng.Productions[int(ce.prodIdx)]
				if int(ce.dot) == len(prod.RHS) && ce.lookaheads.contains(la) {
					rprod = int(ce.prodIdx)
					break
				}
			}
			if rprod >= 0 {
				prod := &ng.Productions[rprod]
				// pop |RHS|
				if len(prod.RHS) > 0 {
					stack = stack[:len(stack)-len(prod.RHS)]
				}
				back := stack[len(stack)-1]
				goTo, ok := ctx.transitionTarget(back, prod.LHS)
				if !ok {
					t.Logf("  REDUCE %s->%s then NO GOTO from state %d on %s",
						symName(prod.LHS), rhsStr(ng, prod, symName), back, symName(prod.LHS))
					return
				}
				stack = append(stack, goTo)
				if prod.LHS == ng.StartSymbol || strings.HasPrefix(symName(prod.LHS), "S'") {
					t.Logf("  ACCEPT via %s", symName(prod.LHS))
				}
				continue
			}
			// Accept?
			if la == 0 {
				// check augmented accept item
				for _, ce := range set.cores {
					prod := &ng.Productions[int(ce.prodIdx)]
					if int(ce.prodIdx) == ng.AugmentProdID && int(ce.dot) == len(prod.RHS) {
						t.Logf("  ACCEPT")
						return
					}
				}
			}
			// Dead.
			t.Logf("  DEAD at state %d on lookahead %q (sym %d)", cur, symName(la), la)
			// Dump completed items in this state to show what reduce LAs exist.
			for _, ce := range set.cores {
				prod := &ng.Productions[int(ce.prodIdx)]
				if int(ce.dot) == len(prod.RHS) {
					t.Logf("      completed: %s->%s  LA{%s}",
						symName(prod.LHS), rhsStr(ng, prod, symName), laNames(&ce.lookaheads))
				}
			}
			return
		}
	}

	// type Query { users : [ User ] }
	sim("type Q { a: [B] }", []int{
		termID("type"), termID("name"), termID("{"), termID("name"),
		termID(":"), termID("["), termID("name"), termID("]"), termID("}"),
	})

	// also the working version: type Q { a: B }  (plain named_type)
	sim("type Q { a: B }", []int{
		termID("type"), termID("name"), termID("{"), termID("name"),
		termID(":"), termID("name"), termID("}"),
	})
}
