package gotreesitter

import "testing"

// TestAliasedHiddenChildedWrapperNestsUnderAlias guards the single-iteration
// repeat1(alias($._x, $.y)) case. When a hidden node flattens to exactly ONE
// visible descendant that is itself an internal node (it has children), aliasing
// the hidden node to an outer name must NEST the inner wrapper under the new
// alias rather than rename-through (collapse) it. Mirrors upstream tree-sitter:
// a single list_item produced by repeat1(alias($._list_item, $.list_item)) and
// then aliased to (section) yields (section (list_item ...)), not a renamed
// (section ...) with the list_item layer evaporated.
func TestAliasedHiddenChildedWrapperNestsUnderAlias(t *testing.T) {
	lang := &Language{
		SymbolCount: 5,
		SymbolMetadata: []SymbolMetadata{
			{Visible: false, Named: false}, // 0: hidden container (_list)
			{Visible: true, Named: false},  // 1: anon leaf (list_marker)
			{Visible: true, Named: true},   // 2: named leaf (paragraph)
			{Visible: true, Named: true},   // 3: visible wrapper (list_item)
			{Visible: true, Named: true},   // 4: outer alias (section)
		},
	}
	arena := acquireNodeArena(arenaClassFull)

	marker := newLeafNodeInArena(arena, 1, false, 0, 1, Point{Column: 0}, Point{Column: 1})
	para := newLeafNodeInArena(arena, 2, true, 2, 6, Point{Column: 2}, Point{Column: 6})
	// list_item: a VISIBLE internal node with two children.
	listItem := newParentNodeInArena(arena, 3, true, []*Node{marker, para}, nil, 11)
	// hidden _list holding the single list_item.
	hidden := newParentNodeInArena(arena, 0, false, []*Node{listItem}, nil, 12)

	aliased := aliasedNodeInArena(arena, lang, hidden, 4)
	if aliased == nil {
		t.Fatal("expected aliased node")
	}
	if got, want := aliased.symbol, Symbol(4); got != want {
		t.Fatalf("symbol = %d, want %d (section)", got, want)
	}
	// Must NEST: section has exactly one child, the list_item wrapper.
	if got, want := aliased.ChildCount(), 1; got != want {
		t.Fatalf("child count = %d, want %d (list_item must be nested, not collapsed)", got, want)
	}
	inner := aliased.children[0]
	if got, want := inner.symbol, Symbol(3); got != want {
		t.Fatalf("inner symbol = %d, want %d (list_item)", got, want)
	}
	if got, want := inner.ChildCount(), 2; got != want {
		t.Fatalf("inner child count = %d, want %d (list_item's marker+paragraph)", got, want)
	}
}

// TestAliasedHiddenBareLeafStillRenamesThrough guards the companion side of the
// discriminator: a hidden node whose lone visible descendant is a childless LEAF
// (a token-shaped wrapper) must still rename-through (collapse) — the alias
// relabels it in place, e.g. alias($._hidden_token, $.name) -> (name).
func TestAliasedHiddenBareLeafStillRenamesThrough(t *testing.T) {
	lang := &Language{
		SymbolCount: 3,
		SymbolMetadata: []SymbolMetadata{
			{Visible: false, Named: false}, // 0: hidden container
			{Visible: true, Named: false},  // 1: visible leaf (childless)
			{Visible: true, Named: true},   // 2: outer alias
		},
	}
	arena := acquireNodeArena(arenaClassFull)

	leaf := newLeafNodeInArena(arena, 1, false, 4, 9, Point{Column: 4}, Point{Column: 9})
	hidden := newParentNodeInArena(arena, 0, false, []*Node{leaf}, nil, 17)

	aliased := aliasedNodeInArena(arena, lang, hidden, 2)
	if aliased == nil {
		t.Fatal("expected aliased node")
	}
	if got, want := aliased.symbol, Symbol(2); got != want {
		t.Fatalf("symbol = %d, want %d", got, want)
	}
	// Must rename-through: bare leaf collapses, childCount==0.
	if got, want := aliased.ChildCount(), 0; got != want {
		t.Fatalf("child count = %d, want %d (bare leaf must rename-through)", got, want)
	}
	if got, want := aliased.StartByte(), uint32(4); got != want {
		t.Fatalf("start byte = %d, want %d", got, want)
	}
	if got, want := aliased.EndByte(), uint32(9); got != want {
		t.Fatalf("end byte = %d, want %d", got, want)
	}
}
