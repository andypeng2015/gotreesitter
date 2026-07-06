package gotreesitter

import "testing"

func TestNormalizeFSharpSimpleLongIdentifierUnwrapsIdentifier(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier", "identifier", "long_identifier_or_op"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 3, true, 4, 17, Point{Column: 4}, Point{Column: 17})
	longIdentifier := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	longIdentifierOrOp := newParentNodeInArena(arena, 4, true, []*Node{longIdentifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{longIdentifierOrOp}, nil, 0)

	normalizeFSharpSimpleLongIdentifiers(root, lang)

	got := root.Child(0).Child(0)
	if got == nil {
		t.Fatal("root child is nil")
	}
	if got.Type(lang) != "identifier" {
		t.Fatalf("root child type = %q, want identifier", got.Type(lang))
	}
	if got.ChildCount() != 0 {
		t.Fatalf("identifier child count = %d, want 0", got.ChildCount())
	}
	if got.StartByte() != 4 || got.EndByte() != 17 {
		t.Fatalf("identifier span = %d:%d, want 4:17", got.StartByte(), got.EndByte())
	}
}

func TestNormalizeFSharpSimpleLongIdentifierSkipsErrorRoot(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier", "identifier", "long_identifier_or_op"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 3, true, 4, 17, Point{Column: 4}, Point{Column: 17})
	longIdentifier := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	longIdentifierOrOp := newParentNodeInArena(arena, 4, true, []*Node{longIdentifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{longIdentifierOrOp}, nil, 0)
	root.setHasError(true)

	normalizeFSharpSimpleLongIdentifiers(root, lang)

	got := root.Child(0).Child(0)
	if got == nil {
		t.Fatal("root child is nil")
	}
	if got.Type(lang) != "long_identifier" {
		t.Fatalf("root child type = %q, want long_identifier", got.Type(lang))
	}
}

func TestNormalizeFSharpSimpleLongIdentifierSkipsOtherContexts(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier", "identifier", "import_decl"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 3, true, 4, 10, Point{Column: 4}, Point{Column: 10})
	longIdentifier := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	importDecl := newParentNodeInArena(arena, 4, true, []*Node{longIdentifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{importDecl}, nil, 0)

	normalizeFSharpSimpleLongIdentifiers(root, lang)

	got := root.Child(0).Child(0)
	if got == nil {
		t.Fatal("import child is nil")
	}
	if got.Type(lang) != "long_identifier" {
		t.Fatalf("import child type = %q, want long_identifier", got.Type(lang))
	}
}
