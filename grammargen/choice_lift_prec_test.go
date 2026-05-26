package grammargen

import (
	"testing"
)

func TestChoiceLiftHelperBlankDoesNotInheritSiblingPrecedence(t *testing.T) {
	st := newSymbolTable()
	helper := st.addSymbol("_source_file_choice_lift1", SymbolInfo{
		Name: "_source_file_choice_lift1", Kind: SymbolNonterminal,
	})
	st.addSymbol("program", SymbolInfo{Name: "program", Kind: SymbolNonterminal})

	prodID := 0
	prods := flattenRule2(Choice(Blank(), Prec(1, Sym("program"))), helper, st, &prodID)
	if len(prods) != 2 {
		t.Fatalf("flattenRule2 produced %d productions, want 2", len(prods))
	}

	var epsilon, lifted *Production
	for i := range prods {
		prod := &prods[i]
		if len(prod.RHS) == 0 {
			epsilon = prod
		} else {
			lifted = prod
		}
	}
	if epsilon == nil || lifted == nil {
		t.Fatalf("expected one blank and one non-blank production, got %#v", prods)
	}
	if lifted.Prec != 1 {
		t.Fatalf("lifted production precedence = %d, want 1", lifted.Prec)
	}
	if epsilon.Prec != 0 {
		t.Fatalf("epsilon production precedence = %d, want 0", epsilon.Prec)
	}
}

func TestChoiceLiftHelperConflictDoesNotResolveToNoAction(t *testing.T) {
	g := NewGrammar("choice_lift_conflict")
	g.ChoiceLiftThreshold = 1
	g.SetExtras(Pat(`\s+`))
	g.Define("source_file", Choice(
		Seq(
			Choice(Blank(), Prec(1, Sym("program"))),
			Sym("program"),
		),
		Choice(Blank(), Sym("program")),
	))
	g.Define("program", Sym("program_statement"))
	g.Define("program_statement", Seq(Str("program"), Sym("identifier")))
	g.Define("identifier", Pat(`[a-z]+`))

	ng, err := Normalize(g)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	tables, _, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	programTok := -1
	for i, sym := range ng.Symbols {
		if sym.Kind == SymbolTerminal && sym.Name == "program" {
			programTok = i
			break
		}
	}
	if programTok < 0 {
		t.Fatalf("program token not found")
	}
	if len(tables.ActionTable[0][programTok]) == 0 {
		t.Fatalf("expected pre-resolve conflict on program in state 0")
	}
	if err := resolveConflicts(tables, ng); err != nil {
		t.Fatalf("resolveConflicts: %v", err)
	}
	if len(tables.ActionTable[0][programTok]) == 0 {
		t.Fatalf("conflict resolution erased all actions on program in state 0")
	}
}
