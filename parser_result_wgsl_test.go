package gotreesitter

import "testing"

func TestNormalizeWGSLEmptyReturnSemicolonRecovery(t *testing.T) {
	lang := &Language{
		Name:        "wgsl",
		SymbolNames: []string{"EOF", "source_file", "compound_statement", "{", "}", "increment_statement", "return_statement", "return", ";", "ERROR"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "compound_statement", Visible: true, Named: true},
			{Name: "{", Visible: true, Named: false},
			{Name: "}", Visible: true, Named: false},
			{Name: "increment_statement", Visible: true, Named: true},
			{Name: "return_statement", Visible: true, Named: true},
			{Name: "return", Visible: true, Named: false},
			{Name: ";", Visible: true, Named: false},
			{Name: "ERROR", Visible: true, Named: true},
		},
	}
	arena := newNodeArena(arenaClassFull)
	open := newLeafNodeInArena(arena, 3, false, 0, 1, Point{}, Point{Column: 1})
	inc := newLeafNodeInArena(arena, 5, true, 4, 14, Point{Column: 4}, Point{Column: 14})
	missingReturn := newLeafNodeInArena(arena, 7, false, 14, 14, Point{Column: 14}, Point{Column: 14})
	missingReturn.setMissing(true)
	emptyReturn := newParentNodeInArena(arena, 6, true, []*Node{missingReturn}, nil, 0)
	semi := newLeafNodeInArena(arena, 8, false, 14, 15, Point{Column: 14}, Point{Column: 15})
	close := newLeafNodeInArena(arena, 4, false, 18, 19, Point{Column: 18}, Point{Column: 19})
	block := newParentNodeInArena(arena, 2, true, []*Node{open, inc, emptyReturn, semi, close}, nil, 0)

	normalizeWGSLCompatibility(block, lang)

	if got, want := block.ChildCount(), 4; got != want {
		t.Fatalf("compound child count = %d, want %d", got, want)
	}
	err := block.Child(2)
	if err == nil {
		t.Fatal("recovery child = nil")
	}
	if got, want := err.Type(lang), "ERROR"; got != want {
		t.Fatalf("recovery child type = %q, want %q", got, want)
	}
	if got, want := err.StartByte(), uint32(14); got != want {
		t.Fatalf("ERROR start = %d, want %d", got, want)
	}
	if got, want := err.EndByte(), uint32(15); got != want {
		t.Fatalf("ERROR end = %d, want %d", got, want)
	}
	if got, want := err.ChildCount(), 1; got != want {
		t.Fatalf("ERROR child count = %d, want %d", got, want)
	}
	if child := err.Child(0); child == nil || child.Type(lang) != ";" {
		t.Fatalf("ERROR child = %#v, want semicolon", child)
	}
	if !err.HasError() {
		t.Fatal("ERROR node should carry has_error")
	}
}
