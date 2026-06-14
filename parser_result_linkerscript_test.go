package gotreesitter

import "testing"

func TestNormalizeLinkerscriptErrorNamedness(t *testing.T) {
	lang := &Language{Name: "linkerscript", SymbolNames: []string{"EOF", "linkerscript"}}
	arena := newNodeArena(0)
	errNode := newLeafNodeInArena(arena, errorSymbol, false, 1, 2, Point{Column: 1}, Point{Column: 2})
	root := newParentNodeInArena(arena, 1, true, []*Node{errNode}, nil, 0)

	normalizeLinkerscriptCompatibility(root, []byte("abc"), lang)

	if !errNode.IsNamed() {
		t.Fatal("ERROR IsNamed = false, want true")
	}
}

func TestNormalizeLinkerscriptEmptyRootSpan(t *testing.T) {
	source := []byte("/*c*/\nOUTPUT_FORMAT(\"elf64\")")
	lang := &Language{Name: "linkerscript", SymbolNames: []string{"EOF", "linkerscript", "comment"}}
	arena := newNodeArena(0)
	comment := newLeafNodeInArena(arena, 2, true, 0, 5, Point{}, Point{Column: 5})
	errNode := newLeafNodeInArena(arena, errorSymbol, true, 6, uint32(len(source)), Point{Row: 1}, Point{Row: 1, Column: 22})
	root := newParentNodeInArena(arena, 1, true, []*Node{comment, errNode}, nil, 0)
	root.endByte = 0
	root.endPoint = Point{}

	normalizeLinkerscriptCompatibility(root, source, lang)

	if got, want := root.EndByte(), uint32(len(source)); got != want {
		t.Fatalf("root end byte = %d, want %d", got, want)
	}
}
