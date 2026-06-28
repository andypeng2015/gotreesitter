package gotreesitter

import "testing"

func TestNormalizeResultCompatibilityRestoresRStringContentEscapes(t *testing.T) {
	lang := &Language{
		Name:        "r",
		SymbolNames: []string{"EOF", "program", "string", "string_content", "escape_sequence", "\""},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "program", Visible: true, Named: true},
			{Name: "string", Visible: true, Named: true},
			{Name: "string_content", Visible: true, Named: true},
			{Name: "escape_sequence", Visible: true, Named: true},
			{Name: "\"", Visible: true, Named: false},
		},
	}
	source := []byte("\"ABI: (\\\\d+)\"")
	arena := newNodeArena(arenaClassFull)
	open := newLeafNodeInArena(arena, 5, false, 0, 1, Point{}, Point{Column: 1})
	content := newLeafNodeInArena(arena, 3, true, 7, 9, Point{Column: 7}, Point{Column: 9})
	close := newLeafNodeInArena(arena, 5, false, uint32(len(source)-1), uint32(len(source)), Point{Column: uint32(len(source) - 1)}, Point{Column: uint32(len(source))})
	str := newParentNodeInArena(arena, 2, true, []*Node{open, content, close}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{str}, nil, 0)

	normalizeResultCompatibility(root, source, &Parser{language: lang})

	if got, want := content.StartByte(), uint32(1); got != want {
		t.Fatalf("content.StartByte() = %d, want %d", got, want)
	}
	if got, want := content.EndByte(), uint32(len(source)-1); got != want {
		t.Fatalf("content.EndByte() = %d, want %d", got, want)
	}
	if got, want := content.ChildCount(), 1; got != want {
		t.Fatalf("content.ChildCount() = %d, want %d", got, want)
	}
	child := content.Child(0)
	if child == nil {
		t.Fatal("content.Child(0) = nil")
	}
	if got, want := child.Type(lang), "escape_sequence"; got != want {
		t.Fatalf("child.Type() = %q, want %q", got, want)
	}
	if !child.IsNamed() {
		t.Fatal("restored escape_sequence child should be named")
	}
	if got, want := child.StartByte(), uint32(7); got != want {
		t.Fatalf("child.StartByte() = %d, want %d", got, want)
	}
	if got, want := child.EndByte(), uint32(9); got != want {
		t.Fatalf("child.EndByte() = %d, want %d", got, want)
	}
	if child.parent != content {
		t.Fatal("escape_sequence parent was not restored")
	}
}

func TestREscapeSequenceEndMatchesGrammarAlternatives(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want uint32
	}{
		{name: "ordinary escape", src: `\n`, want: 2},
		{name: "backslash escape", src: `\\`, want: 2},
		{name: "octal escape", src: `\123`, want: 4},
		{name: "hex escape", src: `\x4f`, want: 4},
		{name: "unicode escape", src: `\u{4f}`, want: 6},
		{name: "digit eight is not escape", src: `\8`, want: 0},
		{name: "bare hex prefix is not escape", src: `\x`, want: 0},
		{name: "empty unicode braces are not escape", src: `\u{}`, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(tc.src)
			if got := rEscapeSequenceEnd(source, 0, uint32(len(source))); got != tc.want {
				t.Fatalf("rEscapeSequenceEnd(%q) = %d, want %d", tc.src, got, tc.want)
			}
		})
	}
}
