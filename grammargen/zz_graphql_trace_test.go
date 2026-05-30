package grammargen

import (
	"os"
	"sort"
	"strings"
	"testing"
)

func TestZZGraphqlTraceListType(t *testing.T) {
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
	// Prefer terminal symbols when a name collides with a nonterminal.
	byName := map[string]int{}
	for i, s := range ng.Symbols {
		if existing, ok := byName[s.Name]; ok {
			if ng.Symbols[existing].Kind == SymbolTerminal && i < ng.TokenCount() {
				// keep terminal already chosen
			}
			if s.Kind == SymbolTerminal && i < ng.TokenCount() {
				byName[s.Name] = i
			}
			continue
		}
		byName[s.Name] = i
	}
	// Force terminal IDs for our literal tokens.
	for i, s := range ng.Symbols {
		if i < ng.TokenCount() && s.Kind == SymbolTerminal {
			switch s.Name {
			case "type", "{", "}", ":", "[", "]", "name":
				byName[s.Name] = i
			}
		}
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

	dumpState := func(label string, si int) {
		set := &sets[si]
		t.Logf("--- %s = state %d (%d cores) ---", label, si, len(set.cores))
		for _, ce := range set.cores {
			prod := &ng.Productions[int(ce.prodIdx)]
			var rhs []string
			for j, s := range prod.RHS {
				if j == int(ce.dot) {
					rhs = append(rhs, ".")
				}
				rhs = append(rhs, symName(s))
			}
			if int(ce.dot) == len(prod.RHS) {
				rhs = append(rhs, ".")
			}
			// Only show items mentioning list_type / string_value / the relevant nts.
			line := symName(prod.LHS) + " -> " + strings.Join(rhs, " ")
			if strings.Contains(line, "list_type") || strings.Contains(line, "string_value") ||
				strings.Contains(line, "field_definition") || strings.Contains(line, "type ") ||
				strings.HasSuffix(line, "type") || strings.Contains(line, "value") {
				t.Logf("    %s   LA{%s}", line, laNames(&ce.lookaheads))
			}
		}
	}

	// Trace path for: type Query { users : [ User ]   then expect }
	// tokens (terminal symbol names as they appear in grammar): "type" name "{" name ":" "[" name "]" "}"
	tokenSeq := []string{"type", "name", "{", "name", ":", "[", "name", "]"}
	// resolve terminal symbol ids by literal name
	resolve := func(lit string) int {
		// try exact terminal name match
		if id, ok := byName[lit]; ok {
			return id
		}
		// brackets/keywords stored as literal strings
		t.Fatalf("cannot resolve token %q", lit)
		return -1
	}

	state := 0
	t.Logf("start state 0")
	for _, tk := range tokenSeq {
		sym := resolve(tk)
		next, ok := ctx.transitionTarget(state, sym)
		if !ok {
			t.Logf("NO TRANSITION from state %d on %q (sym %d)", state, tk, sym)
			dumpState("dead-at", state)
			return
		}
		t.Logf("shift %q (sym %d): state %d -> %d", tk, sym, state, next)
		state = next
	}
	dumpState("after-[-name-]", state)

	// In this state, what does list_type reduce on? show transitions available.
	t.Logf("=== transitions from final state %d ===", state)
	row := ctx.transitionRow(state)
	for _, tr := range row {
		t.Logf("    on %q (sym %d) -> state %d", symName(int(tr.sym)), tr.sym, tr.target)
	}
}
