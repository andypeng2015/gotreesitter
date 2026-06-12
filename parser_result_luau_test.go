package gotreesitter

import "testing"

func TestNormalizeLuauRecoveredErrorEndIdentifier(t *testing.T) {
	lang := testLuauNormalizerLanguage("luau")
	source := []byte("end")
	arena := newNodeArena(arenaClassFull)
	end := newLeafNodeInArena(arena, 3, false, 0, 3, Point{}, Point{Column: 3})
	errNode := newParentNodeInArena(arena, errorSymbol, true, []*Node{end}, nil, 0)
	errNode.setExtra(true)
	errNode.setHasError(true)
	root := newParentNodeInArena(arena, 1, true, []*Node{errNode}, nil, 0)
	root.setHasError(true)

	normalizeLuauCompatibility(root, source, lang)

	if got, want := end.Type(lang), "identifier"; got != want {
		t.Fatalf("recovered child type = %q, want %q", got, want)
	}
	if !end.IsNamed() {
		t.Fatal("recovered child IsNamed = false, want true")
	}
}

func TestNormalizeLuauRecoveredErrorEndIdentifierNegativeCanaries(t *testing.T) {
	t.Run("multi-child ERROR is not rewritten", func(t *testing.T) {
		lang := testLuauNormalizerLanguage("luau")
		source := []byte("end end")
		arena := newNodeArena(arenaClassFull)
		firstEnd := newLeafNodeInArena(arena, 3, false, 0, 3, Point{}, Point{Column: 3})
		secondEnd := newLeafNodeInArena(arena, 3, false, 4, 7, Point{Column: 4}, Point{Column: 7})
		root := testLuauNormalizerRoot(arena, newParentNodeInArena(arena, errorSymbol, true, []*Node{firstEnd, secondEnd}, nil, 0))

		normalizeLuauCompatibility(root, source, lang)

		if got, want := firstEnd.Type(lang), "end"; got != want {
			t.Fatalf("first child type = %q, want %q", got, want)
		}
		if firstEnd.IsNamed() {
			t.Fatal("first child IsNamed = true, want false")
		}
	})

	t.Run("same child with non-end text is not rewritten", func(t *testing.T) {
		lang := testLuauNormalizerLanguage("luau")
		source := []byte("and")
		arena := newNodeArena(arenaClassFull)
		end := newLeafNodeInArena(arena, 3, false, 0, 3, Point{}, Point{Column: 3})
		root := testLuauNormalizerRoot(arena, newParentNodeInArena(arena, errorSymbol, true, []*Node{end}, nil, 0))

		normalizeLuauCompatibility(root, source, lang)

		if got, want := end.Type(lang), "end"; got != want {
			t.Fatalf("child type = %q, want %q", got, want)
		}
		if end.IsNamed() {
			t.Fatal("child IsNamed = true, want false")
		}
	})

	t.Run("non-Luau language is not rewritten", func(t *testing.T) {
		lang := testLuauNormalizerLanguage("lua")
		source := []byte("end")
		arena := newNodeArena(arenaClassFull)
		end := newLeafNodeInArena(arena, 3, false, 0, 3, Point{}, Point{Column: 3})
		root := testLuauNormalizerRoot(arena, newParentNodeInArena(arena, errorSymbol, true, []*Node{end}, nil, 0))

		normalizeLuauRecoveredErrorEndIdentifier(root, source, lang)

		if got, want := end.Type(lang), "end"; got != want {
			t.Fatalf("child type = %q, want %q", got, want)
		}
		if end.IsNamed() {
			t.Fatal("child IsNamed = true, want false")
		}
	})

	t.Run("already-named non-end child is not rewritten", func(t *testing.T) {
		lang := testLuauNormalizerLanguage("luau")
		source := []byte("end")
		arena := newNodeArena(arenaClassFull)
		chunk := newLeafNodeInArena(arena, 1, true, 0, 3, Point{}, Point{Column: 3})
		root := testLuauNormalizerRoot(arena, newParentNodeInArena(arena, errorSymbol, true, []*Node{chunk}, nil, 0))

		normalizeLuauCompatibility(root, source, lang)

		if got, want := chunk.Type(lang), "chunk"; got != want {
			t.Fatalf("child type = %q, want %q", got, want)
		}
		if !chunk.IsNamed() {
			t.Fatal("child IsNamed = false, want true")
		}
	})
}

func TestParseIterationsForLanguageLuau(t *testing.T) {
	sourceLen := 2788
	if got, want := parseIterationsForLanguage(sourceLen, &Language{Name: "luau"}), sourceLen*68; got != want {
		t.Fatalf("luau iteration budget = %d, want %d", got, want)
	}
	if got, want := parseIterationsForLanguage(sourceLen, &Language{Name: "go"}), parseIterations(sourceLen); got != want {
		t.Fatalf("default iteration budget changed: got %d, want %d", got, want)
	}
}

func testLuauNormalizerLanguage(name string) *Language {
	return &Language{
		Name:        name,
		SymbolNames: []string{"EOF", "chunk", "identifier", "end"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "chunk", Visible: true, Named: true},
			{Name: "identifier", Visible: true, Named: true},
			{Name: "end", Visible: true, Named: false},
		},
	}
}

func testLuauNormalizerRoot(arena *nodeArena, errNode *Node) *Node {
	errNode.setExtra(true)
	errNode.setHasError(true)
	root := newParentNodeInArena(arena, 1, true, []*Node{errNode}, nil, 0)
	root.setHasError(true)
	return root
}
