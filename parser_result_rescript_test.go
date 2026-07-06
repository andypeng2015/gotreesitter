package gotreesitter

import "testing"

func TestNormalizeRescriptValueIdentifierPath(t *testing.T) {
	lang := testRescriptNormalizerLanguage("rescript")
	arena := newNodeArena(arenaClassFull)
	variantName := newLeafNodeInArena(arena, 7, true, 0, 1, Point{}, Point{Column: 1})
	object := newParentNodeInArena(arena, 3, true, []*Node{variantName}, nil, 0)
	dot := newLeafNodeInArena(arena, 4, false, 1, 2, Point{Column: 1}, Point{Column: 2})
	propertyName := newLeafNodeInArena(arena, 6, true, 2, 3, Point{Column: 2}, Point{Column: 3})
	property := newParentNodeInArena(arena, 5, true, []*Node{propertyName}, nil, 0)
	member := newParentNodeInArena(arena, 2, true, []*Node{object, dot, property}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{member}, nil, 0)

	normalizeRescriptCompatibility(root, lang)

	if got, want := member.Type(lang), "value_identifier_path"; got != want {
		t.Fatalf("member type = %q, want %q", got, want)
	}
	if !member.IsNamed() {
		t.Fatal("member IsNamed = false, want true")
	}
	if got, want := member.ChildCount(), 3; got != want {
		t.Fatalf("member child count = %d, want %d", got, want)
	}
	if got, want := member.Child(0).Type(lang), "module_identifier"; got != want {
		t.Fatalf("object type = %q, want %q", got, want)
	}
	if got := member.Child(0).ChildCount(); got != 0 {
		t.Fatalf("module_identifier child count = %d, want 0", got)
	}
	if got, want := member.Child(1).Type(lang), "."; got != want {
		t.Fatalf("dot type = %q, want %q", got, want)
	}
	if member.Child(1).IsNamed() {
		t.Fatal("dot IsNamed = true, want false")
	}
	if got, want := member.Child(2).Type(lang), "value_identifier"; got != want {
		t.Fatalf("property type = %q, want %q", got, want)
	}
	if got := member.Child(2).ChildCount(); got != 0 {
		t.Fatalf("value_identifier child count = %d, want 0", got)
	}
	if got, want := root.SExpr(lang), "(source_file (value_identifier_path (module_identifier) (value_identifier)))"; got != want {
		t.Fatalf("root SExpr = %q, want %q", got, want)
	}
}

func TestNormalizeRescriptValueIdentifierPathSkipsErrorRoot(t *testing.T) {
	lang := testRescriptNormalizerLanguage("rescript")
	root, member, object, property := testRescriptMemberExpressionTree(t, lang)
	root.setHasError(true)

	normalizeRescriptCompatibility(root, lang)

	if got, want := member.Type(lang), "member_expression"; got != want {
		t.Fatalf("member type = %q, want %q", got, want)
	}
	if got, want := object.Type(lang), "variant"; got != want {
		t.Fatalf("object type = %q, want %q", got, want)
	}
	if got, want := object.ChildCount(), 1; got != want {
		t.Fatalf("variant child count = %d, want %d", got, want)
	}
	if got, want := property.Type(lang), "property_identifier"; got != want {
		t.Fatalf("property type = %q, want %q", got, want)
	}
	if got, want := property.ChildCount(), 1; got != want {
		t.Fatalf("property child count = %d, want %d", got, want)
	}
}

func TestNormalizeRescriptValueIdentifierPathSkipsNonTargetShape(t *testing.T) {
	lang := testRescriptNormalizerLanguage("rescript")
	arena := newNodeArena(arenaClassFull)
	object := newLeafNodeInArena(arena, 3, true, 0, 1, Point{}, Point{Column: 1})
	dot := newLeafNodeInArena(arena, 4, false, 1, 2, Point{Column: 1}, Point{Column: 2})
	property := newLeafNodeInArena(arena, 6, true, 2, 3, Point{Column: 2}, Point{Column: 3})
	member := newParentNodeInArena(arena, 2, true, []*Node{object, dot, property}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{member}, nil, 0)

	normalizeRescriptCompatibility(root, lang)

	if got, want := member.Type(lang), "member_expression"; got != want {
		t.Fatalf("member type = %q, want %q", got, want)
	}
	if got, want := object.Type(lang), "variant"; got != want {
		t.Fatalf("object type = %q, want %q", got, want)
	}
	if got, want := property.Type(lang), "value_identifier"; got != want {
		t.Fatalf("property type = %q, want %q", got, want)
	}
}

func testRescriptMemberExpressionTree(t *testing.T, lang *Language) (*Node, *Node, *Node, *Node) {
	t.Helper()
	arena := newNodeArena(arenaClassFull)
	variantName := newLeafNodeInArena(arena, 7, true, 0, 1, Point{}, Point{Column: 1})
	object := newParentNodeInArena(arena, 3, true, []*Node{variantName}, nil, 0)
	dot := newLeafNodeInArena(arena, 4, false, 1, 2, Point{Column: 1}, Point{Column: 2})
	propertyName := newLeafNodeInArena(arena, 6, true, 2, 3, Point{Column: 2}, Point{Column: 3})
	property := newParentNodeInArena(arena, 5, true, []*Node{propertyName}, nil, 0)
	member := newParentNodeInArena(arena, 2, true, []*Node{object, dot, property}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{member}, nil, 0)
	if got, want := member.Type(lang), "member_expression"; got != want {
		t.Fatalf("test setup member type = %q, want %q", got, want)
	}
	return root, member, object, property
}

func testRescriptNormalizerLanguage(name string) *Language {
	return &Language{
		Name: name,
		SymbolNames: []string{
			"EOF",
			"source_file",
			"member_expression",
			"variant",
			".",
			"property_identifier",
			"value_identifier",
			"variant_identifier",
			"module_identifier",
			"value_identifier_path",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "member_expression", Visible: true, Named: true},
			{Name: "variant", Visible: true, Named: true},
			{Name: ".", Visible: true, Named: false},
			{Name: "property_identifier", Visible: true, Named: true},
			{Name: "value_identifier", Visible: true, Named: true},
			{Name: "variant_identifier", Visible: true, Named: true},
			{Name: "module_identifier", Visible: true, Named: true},
			{Name: "value_identifier_path", Visible: true, Named: true},
		},
	}
}
