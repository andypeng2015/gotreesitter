package grammargen

import "testing"

func TestBuildHiddenChoicePassthroughSymbolsMarksOnlyNeutralHiddenChoices(t *testing.T) {
	ng := &NormalizedGrammar{
		Symbols: []SymbolInfo{
			{Name: "EOF", Kind: SymbolTerminal},
			{Name: "_choice", Kind: SymbolNonterminal, Named: true},
			{Name: "_sequence", Kind: SymbolNonterminal, Named: true},
			{Name: "leaf", Kind: SymbolNonterminal, Visible: true, Named: true},
			{Name: "_supertype", Kind: SymbolNonterminal, Named: true, Supertype: true},
			{Name: "_aliased", Kind: SymbolNonterminal, Named: true},
			{Name: "parent", Kind: SymbolNonterminal, Visible: true, Named: true},
			{Name: "_wrapper", Kind: SymbolNonterminal, Named: true},
		},
		Productions: []Production{
			{LHS: 1, RHS: []int{2}},
			{LHS: 1, RHS: []int{3}},
			{LHS: 2, RHS: []int{3, 0}},
			{LHS: 4, RHS: []int{3}},
			{LHS: 5, RHS: []int{3}},
			{LHS: 6, RHS: []int{5}, Aliases: []AliasInfo{{ChildIndex: 0, Name: "alias", Named: true}}},
			{LHS: 7, RHS: []int{3}},
		},
	}

	got := buildHiddenChoicePassthroughSymbols(ng, len(ng.Symbols))
	if len(got) != len(ng.Symbols) || !got[1] {
		t.Fatalf("_choice passthrough mark = %v, want true", got)
	}
	for _, sym := range []int{2, 4, 5, 7} {
		if got[sym] {
			t.Fatalf("symbol %s was marked passthrough; full table %v", ng.Symbols[sym].Name, got)
		}
	}
}
