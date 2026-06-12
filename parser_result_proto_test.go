package gotreesitter

import "testing"

func TestNormalizeProtoCompatibilityRestoresKeyTypeChildren(t *testing.T) {
	lang := &Language{
		Name:        "proto",
		SymbolNames: []string{"EOF", "source_file", "key_type", "string"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "key_type", Visible: true, Named: true},
			{Name: "string", Visible: true, Named: false},
		},
	}
	arena := newNodeArena(arenaClassFull)
	source := []byte("string")
	keyType := newLeafNodeInArena(arena, 2, true, 0, 6, Point{}, Point{Column: 6})
	root := newParentNodeInArena(arena, 1, true, []*Node{keyType}, nil, 0)

	normalizeProtoCompatibility(root, source, lang)

	if got, want := keyType.ChildCount(), 1; got != want {
		t.Fatalf("key_type child count = %d, want %d", got, want)
	}
	child := keyType.Child(0)
	if child == nil {
		t.Fatal("key_type child = nil")
	}
	if got, want := child.Type(lang), "string"; got != want {
		t.Fatalf("key_type child type = %q, want %q", got, want)
	}
	if child.IsNamed() {
		t.Fatal("restored key_type child should be anonymous")
	}
}

func TestNormalizeProtoCompatibilitySkipsErrorRoot(t *testing.T) {
	lang := &Language{
		Name:        "proto",
		SymbolNames: []string{"EOF", "source_file", "key_type", "string"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "key_type", Visible: true, Named: true},
			{Name: "string", Visible: true, Named: false},
		},
	}
	arena := newNodeArena(arenaClassFull)
	source := []byte("string")
	keyType := newLeafNodeInArena(arena, 2, true, 0, 6, Point{}, Point{Column: 6})
	root := newParentNodeInArena(arena, 1, true, []*Node{keyType}, nil, 0)
	root.setHasError(true)

	normalizeProtoCompatibility(root, source, lang)

	if got := keyType.ChildCount(); got != 0 {
		t.Fatalf("key_type child count = %d, want 0 for error root", got)
	}
}
