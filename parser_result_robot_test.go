package gotreesitter

import "testing"

func TestNormalizeRobotEscapedNestedScalarVariableInsertsErrorGap(t *testing.T) {
	source := []byte("${tc['\\${i}']}")
	lang := &Language{
		Name:        "robot",
		SymbolNames: []string{"scalar_variable", "${", "variable_name", "}"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "scalar_variable", Visible: true, Named: true},
			{Name: "${", Visible: true},
			{Name: "variable_name", Visible: true, Named: true},
			{Name: "}", Visible: true},
		},
	}
	arena := newNodeArena(arenaClassFull)
	open := robotTestLeaf(arena, source, 1, false, 0, 2)
	name := robotTestLeaf(arena, source, 2, true, 2, 8)
	close := robotTestLeaf(arena, source, 3, false, 10, 11)
	scalar := newParentNodeInArena(arena, 0, true, []*Node{open, name, close}, nil, 0)

	normalizeRobotCompatibility(scalar, source, lang)

	if got, want := scalar.ChildCount(), 4; got != want {
		t.Fatalf("scalar child count = %d, want %d; sexpr=%s", got, want, scalar.SExpr(lang))
	}
	err := scalar.Child(2)
	if err == nil || err.symbol != errorSymbol {
		t.Fatalf("inserted child symbol = %v, want ERROR", err)
	}
	if got, want := err.StartByte(), uint32(8); got != want {
		t.Fatalf("error start = %d, want %d", got, want)
	}
	if got, want := err.EndByte(), uint32(10); got != want {
		t.Fatalf("error end = %d, want %d", got, want)
	}
	if !err.IsExtra() {
		t.Fatal("inserted error was not marked extra")
	}
	if got, want := err.ChildCount(), 1; got != want {
		t.Fatalf("error child count = %d, want %d", got, want)
	}
	if got, want := err.Child(0).EndByte(), uint32(9); got != want {
		t.Fatalf("inner error end = %d, want %d", got, want)
	}
	if !scalar.HasError() {
		t.Fatal("scalar was not marked as containing an error")
	}
}

func robotTestLeaf(arena *nodeArena, source []byte, sym Symbol, named bool, start, end uint32) *Node {
	return newLeafNodeInArena(
		arena,
		sym,
		named,
		start,
		end,
		advancePointByBytes(Point{}, source[:start]),
		advancePointByBytes(Point{}, source[:end]),
	)
}
