package gotreesitter

import "testing"

func TestNormalizeMakeConditionalConsequenceFieldsExtendsAcrossLeadingTabs(t *testing.T) {
	lang := &Language{
		Name:        "make",
		SymbolNames: []string{"EOF", "conditional", "ifneq_directive", "\t", "recipe_line", "else_directive", "endif"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "conditional", Visible: true, Named: true},
			{Name: "ifneq_directive", Visible: true, Named: true},
			{Name: "\t", Visible: true, Named: false},
			{Name: "recipe_line", Visible: true, Named: true},
			{Name: "else_directive", Visible: true, Named: true},
			{Name: "endif", Visible: true, Named: false},
		},
		FieldNames: []string{"", "consequence"},
	}

	arena := newNodeArena(arenaClassFull)
	directive := newLeafNodeInArena(arena, 2, true, 0, 5, Point{}, Point{Column: 5})
	tab := newLeafNodeInArena(arena, 3, false, 5, 6, Point{Column: 5}, Point{Column: 6})
	recipe := newLeafNodeInArena(arena, 4, true, 6, 12, Point{Column: 6}, Point{Column: 12})
	elseDir := newLeafNodeInArena(arena, 5, true, 12, 16, Point{Column: 12}, Point{Column: 16})
	endif := newLeafNodeInArena(arena, 6, false, 16, 21, Point{Column: 16}, Point{Column: 21})
	root := newParentNodeInArena(arena, 1, true, []*Node{directive, tab, recipe, elseDir, endif}, []FieldID{0, 0, 1, 1, 0}, 0)
	root.fieldSources = []uint8{0, 0, fieldSourceDirect, fieldSourceDirect, 0}

	normalizeMakeConditionalConsequenceFields(root, lang)

	if got, want := root.fieldIDs[1], FieldID(1); got != want {
		t.Fatalf("tab field = %d, want %d", got, want)
	}
	if got, want := fieldSourceAt(root.fieldSources, 1), uint8(fieldSourceDirect); got != want {
		t.Fatalf("tab field source = %d, want %d", got, want)
	}
	if got := root.fieldIDs[4]; got != 0 {
		t.Fatalf("endif field = %d, want 0", got)
	}
}

func TestNormalizeIniSectionStartsSnapToFirstChild(t *testing.T) {
	lang := &Language{
		Name:        "ini",
		SymbolNames: []string{"EOF", "section", "section_name", "setting"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "section", Visible: true, Named: true},
			{Name: "section_name", Visible: true, Named: true},
			{Name: "setting", Visible: true, Named: true},
		},
	}

	arena := newNodeArena(arenaClassFull)
	sectionName := newLeafNodeInArena(arena, 2, true, 48, 69, Point{Row: 1}, Point{Row: 1, Column: 21})
	setting := newLeafNodeInArena(arena, 3, true, 70, 80, Point{Row: 2}, Point{Row: 2, Column: 10})
	section := newParentNodeInArena(arena, 1, true, []*Node{sectionName, setting}, nil, 0)
	section.startByte = 0
	section.startPoint = Point{}

	normalizeIniSectionStarts(section, lang)

	if got, want := section.startByte, uint32(48); got != want {
		t.Fatalf("section.startByte = %d, want %d", got, want)
	}
	if got, want := section.startPoint, sectionName.startPoint; got != want {
		t.Fatalf("section.startPoint = %#v, want %#v", got, want)
	}
}

func TestNormalizeIniDocumentBlanksDropsRootBlank(t *testing.T) {
	lang := &Language{
		Name:        "ini",
		SymbolNames: []string{"EOF", "document", "comment", "_blank", "section"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "document", Visible: true, Named: true},
			{Name: "comment", Visible: true, Named: true},
			{Name: "_blank", Visible: true, Named: true},
			{Name: "section", Visible: true, Named: true},
		},
	}

	arena := newNodeArena(arenaClassFull)
	comment := newLeafNodeInArena(arena, 2, true, 108, 149, Point{}, Point{Column: 41})
	blank := newLeafNodeInArena(arena, 3, true, 149, 150, Point{Column: 41}, Point{Row: 1})
	section := newLeafNodeInArena(arena, 4, true, 150, 600, Point{Row: 1}, Point{Row: 20})
	root := newParentNodeInArena(arena, 1, true, []*Node{comment, blank, section}, nil, 0)
	root.endByte = 600
	root.endPoint = Point{Row: 20}

	normalizeIniCompatibility(root, lang)

	if got, want := root.ChildCount(), 2; got != want {
		t.Fatalf("document child count = %d, want %d", got, want)
	}
	if got, want := root.Child(0).Type(lang), "comment"; got != want {
		t.Fatalf("child[0] type = %q, want %q", got, want)
	}
	if got, want := root.Child(1).Type(lang), "section"; got != want {
		t.Fatalf("child[1] type = %q, want %q", got, want)
	}
	if got, want := root.Child(1).childIndex, int32(1); got != want {
		t.Fatalf("section childIndex = %d, want %d", got, want)
	}
}

func TestNormalizeZigEmptyInitListFieldConstantCleared(t *testing.T) {
	lang := &Language{
		Name:        "zig",
		SymbolNames: []string{"EOF", "SuffixExpr", ".", "InitList", "{", "}"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "SuffixExpr", Visible: true, Named: true},
			{Name: ".", Visible: true, Named: false},
			{Name: "InitList", Visible: true, Named: true},
			{Name: "{", Visible: true, Named: false},
			{Name: "}", Visible: true, Named: false},
		},
		FieldNames: []string{"", "field_constant"},
	}

	arena := newNodeArena(arenaClassFull)
	dot := newLeafNodeInArena(arena, 2, false, 0, 1, Point{}, Point{Column: 1})
	open := newLeafNodeInArena(arena, 4, false, 1, 2, Point{Column: 1}, Point{Column: 2})
	close := newLeafNodeInArena(arena, 5, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	initList := newParentNodeInArena(arena, 3, true, []*Node{open, close}, nil, 0)
	parent := newParentNodeInArena(arena, 1, true, []*Node{dot, initList}, []FieldID{0, 1}, 0)
	parent.fieldSources = []uint8{0, fieldSourceDirect}

	normalizeZigEmptyInitListFields(parent, lang)

	if got := parent.fieldIDs[1]; got != 0 {
		t.Fatalf("fieldIDs[1] = %d, want 0", got)
	}
	if got := fieldSourceAt(parent.fieldSources, 1); got != 0 {
		t.Fatalf("fieldSources[1] = %d, want 0", got)
	}
}

func TestNormalizeZigDottedInitListFieldConstantCleared(t *testing.T) {
	lang := &Language{
		Name:        "zig",
		SymbolNames: []string{"EOF", "SuffixExpr", ".", "InitList", "{", "STRINGLITERALSINGLE", "}"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "SuffixExpr", Visible: true, Named: true},
			{Name: ".", Visible: true, Named: false},
			{Name: "InitList", Visible: true, Named: true},
			{Name: "{", Visible: true, Named: false},
			{Name: "STRINGLITERALSINGLE", Visible: true, Named: true},
			{Name: "}", Visible: true, Named: false},
		},
		FieldNames: []string{"", "field_constant"},
	}

	arena := newNodeArena(arenaClassFull)
	dot := newLeafNodeInArena(arena, 2, false, 0, 1, Point{}, Point{Column: 1})
	open := newLeafNodeInArena(arena, 4, false, 1, 2, Point{Column: 1}, Point{Column: 2})
	value := newLeafNodeInArena(arena, 5, true, 2, 6, Point{Column: 2}, Point{Column: 6})
	close := newLeafNodeInArena(arena, 6, false, 6, 7, Point{Column: 6}, Point{Column: 7})
	initList := newParentNodeInArena(arena, 3, true, []*Node{open, value, close}, nil, 0)
	parent := newParentNodeInArena(arena, 1, true, []*Node{dot, initList}, []FieldID{0, 1}, 0)
	parent.fieldSources = []uint8{0, fieldSourceDirect}

	normalizeZigEmptyInitListFields(parent, lang)

	if got := parent.fieldIDs[1]; got != 0 {
		t.Fatalf("fieldIDs[1] = %d, want 0", got)
	}
	if got := fieldSourceAt(parent.fieldSources, 1); got != 0 {
		t.Fatalf("fieldSources[1] = %d, want 0", got)
	}
}
