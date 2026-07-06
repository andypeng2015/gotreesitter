package gotreesitter

import "testing"

func TestNormalizeHyprlangBooleanAssignmentValues(t *testing.T) {
	lang := &Language{
		Name: "hyprlang",
		SymbolNames: []string{
			"EOF", "configuration", "assignment", "name", "=", "string", "boolean",
			"true", "false", "on", "off", "yes", "no",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "configuration", Visible: true, Named: true},
			{Name: "assignment", Visible: true, Named: true},
			{Name: "name", Visible: true, Named: true},
			{Name: "=", Visible: true, Named: false},
			{Name: "string", Visible: true, Named: true},
			{Name: "boolean", Visible: true, Named: true},
			{Name: "true", Visible: true, Named: false},
			{Name: "false", Visible: true, Named: false},
			{Name: "on", Visible: true, Named: false},
			{Name: "off", Visible: true, Named: false},
			{Name: "yes", Visible: true, Named: false},
			{Name: "no", Visible: true, Named: false},
		},
	}
	source := []byte("resize_on_border = true\nname = myBezier\n")
	arena := newNodeArena(arenaClassFull)
	boolName := newLeafNodeInArena(arena, 3, true, 0, 16, Point{}, Point{Column: 16})
	boolEquals := newLeafNodeInArena(arena, 4, false, 17, 18, Point{Column: 17}, Point{Column: 18})
	boolValue := newLeafNodeInArena(arena, 5, true, 18, 23, Point{Column: 18}, Point{Column: 23})
	boolAssignment := newParentNodeInArena(arena, 2, true, []*Node{boolName, boolEquals, boolValue}, nil, 0)
	stringName := newLeafNodeInArena(arena, 3, true, 24, 28, Point{Row: 1}, Point{Row: 1, Column: 4})
	stringEquals := newLeafNodeInArena(arena, 4, false, 29, 30, Point{Row: 1, Column: 5}, Point{Row: 1, Column: 6})
	stringValue := newLeafNodeInArena(arena, 5, true, 30, 39, Point{Row: 1, Column: 6}, Point{Row: 1, Column: 15})
	stringAssignment := newParentNodeInArena(arena, 2, true, []*Node{stringName, stringEquals, stringValue}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{boolAssignment, stringAssignment}, nil, 0)

	normalizeHyprlangBooleanAssignmentValues(root, source, lang)

	if got, want := boolValue.Type(lang), "boolean"; got != want {
		t.Fatalf("bool value type = %q, want %q", got, want)
	}
	if got, want := boolValue.ChildCount(), 1; got != want {
		t.Fatalf("bool value child count = %d, want %d", got, want)
	}
	if child := boolValue.Child(0); child == nil || child.Type(lang) != "true" || child.IsNamed() {
		t.Fatalf("bool child = %#v, want anonymous true", child)
	}
	if got, want := stringValue.Type(lang), "string"; got != want {
		t.Fatalf("non-boolean value type = %q, want %q", got, want)
	}
	if got := stringValue.ChildCount(); got != 0 {
		t.Fatalf("non-boolean value child count = %d, want 0", got)
	}
}
