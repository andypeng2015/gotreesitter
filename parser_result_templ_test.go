package gotreesitter

import "testing"

func TestNormalizeTemplCompatibilityMergesComponentImportArgs(t *testing.T) {
	lang := &Language{
		Name:        "templ",
		SymbolNames: []string{"EOF", "source_file", "element", "component_import", "element_text", "argument_list", "(", ")", ",", "identifier", "@", "component_identifier"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "element", Visible: true, Named: true},
			{Name: "component_import", Visible: true, Named: true},
			{Name: "element_text", Visible: true, Named: true},
			{Name: "argument_list", Visible: true, Named: true},
			{Name: "(", Visible: true, Named: false},
			{Name: ")", Visible: true, Named: false},
			{Name: ",", Visible: true, Named: false},
			{Name: "identifier", Visible: true, Named: true},
			{Name: "@", Visible: true, Named: false},
			{Name: "component_identifier", Visible: true, Named: true},
		},
	}
	source := []byte("@counts(global, user)")
	arena := newNodeArena(arenaClassFull)
	atNode := newLeafNodeInArena(arena, 10, false, 0, 1, Point{}, Point{Column: 1})
	nameNode := newLeafNodeInArena(arena, 11, true, 1, 7, Point{Column: 1}, Point{Column: 7})
	importNode := newParentNodeInArena(arena, 3, true, []*Node{atNode, nameNode}, nil, 0)
	argsNode := newLeafNodeInArena(arena, 4, true, 7, uint32(len(source)), Point{Column: 7}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 2, true, []*Node{importNode, argsNode}, nil, 0)

	normalizeTemplCompatibility(root, source, lang)

	if got, want := root.ChildCount(), 1; got != want {
		t.Fatalf("root child count = %d, want %d", got, want)
	}
	child := root.Child(0)
	if child == nil {
		t.Fatal("merged child = nil")
	}
	if got, want := child.Type(lang), "component_import"; got != want {
		t.Fatalf("merged child type = %q, want %q", got, want)
	}
	if got, want := child.EndByte(), uint32(len(source)); got != want {
		t.Fatalf("merged child end byte = %d, want %d", got, want)
	}
	if got, want := child.ChildCount(), 3; got != want {
		t.Fatalf("component_import child count = %d, want %d", got, want)
	}
	argList := child.Child(2)
	if argList == nil {
		t.Fatal("argument_list child = nil")
	}
	if got, want := argList.Type(lang), "argument_list"; got != want {
		t.Fatalf("argument child type = %q, want %q", got, want)
	}
	if got, want := argList.ChildCount(), 5; got != want {
		t.Fatalf("argument_list child count = %d, want %d", got, want)
	}
}

func TestNormalizeTemplCompatibilitySkipsNonParenthesizedElementText(t *testing.T) {
	lang := &Language{
		Name:        "templ",
		SymbolNames: []string{"EOF", "source_file", "element", "component_import", "element_text", "argument_list", "(", ")", ",", "identifier", "@", "component_identifier"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "element", Visible: true, Named: true},
			{Name: "component_import", Visible: true, Named: true},
			{Name: "element_text", Visible: true, Named: true},
			{Name: "argument_list", Visible: true, Named: true},
			{Name: "(", Visible: true, Named: false},
			{Name: ")", Visible: true, Named: false},
			{Name: ",", Visible: true, Named: false},
			{Name: "identifier", Visible: true, Named: true},
			{Name: "@", Visible: true, Named: false},
			{Name: "component_identifier", Visible: true, Named: true},
		},
	}
	source := []byte("@counts text")
	arena := newNodeArena(arenaClassFull)
	atNode := newLeafNodeInArena(arena, 10, false, 0, 1, Point{}, Point{Column: 1})
	nameNode := newLeafNodeInArena(arena, 11, true, 1, 7, Point{Column: 1}, Point{Column: 7})
	importNode := newParentNodeInArena(arena, 3, true, []*Node{atNode, nameNode}, nil, 0)
	textNode := newLeafNodeInArena(arena, 4, true, 7, uint32(len(source)), Point{Column: 7}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 2, true, []*Node{importNode, textNode}, nil, 0)

	normalizeTemplCompatibility(root, source, lang)

	if got, want := root.ChildCount(), 2; got != want {
		t.Fatalf("root child count = %d, want %d", got, want)
	}
	if got, want := importNode.EndByte(), uint32(7); got != want {
		t.Fatalf("component_import end byte = %d, want %d", got, want)
	}
}

func TestNormalizeTemplCompatibilityBuildsSimpleStringArgument(t *testing.T) {
	lang := templCompatibilityTestLanguage()
	source := []byte("@headerComponent(\"My Blog\")")
	arena := newNodeArena(arenaClassFull)
	atNode := newLeafNodeInArena(arena, 10, false, 0, 1, Point{}, Point{Column: 1})
	nameNode := newLeafNodeInArena(arena, 11, true, 1, 16, Point{Column: 1}, Point{Column: 16})
	importNode := newParentNodeInArena(arena, 3, true, []*Node{atNode, nameNode}, nil, 0)
	argsNode := newLeafNodeInArena(arena, 4, true, 16, uint32(len(source)), Point{Column: 16}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 2, true, []*Node{importNode, argsNode}, nil, 0)

	normalizeTemplCompatibility(root, source, lang)

	argList := root.Child(0).Child(2)
	if argList == nil {
		t.Fatal("argument_list child = nil")
	}
	if got, want := argList.ChildCount(), 3; got != want {
		t.Fatalf("argument_list child count = %d, want %d", got, want)
	}
	str := argList.Child(1)
	if str == nil {
		t.Fatal("string argument = nil")
	}
	if got, want := str.Type(lang), "interpreted_string_literal"; got != want {
		t.Fatalf("string argument type = %q, want %q", got, want)
	}
	if got, want := str.ChildCount(), 3; got != want {
		t.Fatalf("string child count = %d, want %d", got, want)
	}
}

func templCompatibilityTestLanguage() *Language {
	return &Language{
		Name: "templ",
		SymbolNames: []string{
			"EOF", "source_file", "element", "component_import", "element_text",
			"argument_list", "(", ")", ",", "identifier", "@", "component_identifier",
			"interpreted_string_literal", "interpreted_string_literal_content", "\"",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "element", Visible: true, Named: true},
			{Name: "component_import", Visible: true, Named: true},
			{Name: "element_text", Visible: true, Named: true},
			{Name: "argument_list", Visible: true, Named: true},
			{Name: "(", Visible: true, Named: false},
			{Name: ")", Visible: true, Named: false},
			{Name: ",", Visible: true, Named: false},
			{Name: "identifier", Visible: true, Named: true},
			{Name: "@", Visible: true, Named: false},
			{Name: "component_identifier", Visible: true, Named: true},
			{Name: "interpreted_string_literal", Visible: true, Named: true},
			{Name: "interpreted_string_literal_content", Visible: true, Named: true},
			{Name: "\"", Visible: true, Named: false},
		},
	}
}
