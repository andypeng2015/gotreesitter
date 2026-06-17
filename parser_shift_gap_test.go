package gotreesitter

import (
	"testing"
	"unsafe"
)

func TestRealShiftGapRejectsNonTriviaSource(t *testing.T) {
	source := []byte("call(arg1, arg8)")
	stack := newGLRStack(1)
	stack.byteOffset = uint32(len("call(arg1"))
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source)),
	}

	if realShiftGapIsParserPadding(source, &stack, tok) {
		t.Fatalf("realShiftGapIsParserPadding = true, want false for gap %q", source[stack.byteOffset:tok.StartByte])
	}

	parser := &Parser{glrTrace: false}
	if parser.guardRealShiftGap(source, &stack, tok) {
		t.Fatal("guardRealShiftGap = true, want false")
	}
	if !stack.dead {
		t.Fatal("stack.dead = false, want true")
	}
}

func TestRealShiftGapAllowsTriviaOnlySource(t *testing.T) {
	source := []byte("call(arg1   \n\t\f\v)")
	stack := newGLRStack(1)
	stack.byteOffset = uint32(len("call(arg1"))
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source)),
	}

	if !realShiftGapIsParserPadding(source, &stack, tok) {
		t.Fatalf("realShiftGapIsParserPadding = false, want true for gap %q", source[stack.byteOffset:tok.StartByte])
	}

	parser := &Parser{glrTrace: false}
	if !parser.guardRealShiftGap(source, &stack, tok) {
		t.Fatal("guardRealShiftGap = false, want true")
	}
	if stack.dead {
		t.Fatal("stack.dead = true, want false")
	}
}

func TestForestRealShiftGapRejectsNonTriviaSource(t *testing.T) {
	source := []byte("call(arg1, arg8)")
	node := &gssForestNode{byteOffset: uint32(len("call(arg1"))}
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source)),
	}

	parser := &Parser{glrTrace: false}
	if parser.guardForestRealShiftGap(source, node, tok) {
		t.Fatal("guardForestRealShiftGap = true, want false")
	}
}

func TestForestRecoveryGapRejectsNonTriviaSource(t *testing.T) {
	source := []byte("call(arg1, arg8)")
	node := &gssForestNode{byteOffset: uint32(len("call(arg1"))}
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source)),
	}

	nextIndex := newGSSForestIndex(0)
	var nextFrontier []*gssForestNode
	parser := &Parser{glrTrace: false}
	if parser.guardForestRealShiftGap(source, node, tok) {
		leaf := &Node{}
		sh := coalesceForest(&nextIndex, &gssForestNodeSlab{}, node.state, tok.EndByte, node,
			stackEntry{node: unsafe.Pointer(leaf), state: node.state, kind: stackEntryKindNode},
			0, node.errorCost+int(tok.EndByte-tok.StartByte))
		nextFrontier = append(nextFrontier, sh)
	}
	if len(nextFrontier) != 0 {
		t.Fatalf("forest recovery accepted non-padding gap; next frontier len = %d", len(nextFrontier))
	}
}

func TestRealShiftGapAllowsLeadingBOMPadding(t *testing.T) {
	for _, source := range [][]byte{
		[]byte("\xef\xbb\xbfa"),
		[]byte("\xef\xbb\xbf\n\ta"),
	} {
		stack := newGLRStack(1)
		tok := Token{
			Symbol:    1,
			StartByte: uint32(len(source) - 1),
			EndByte:   uint32(len(source)),
		}

		if !realShiftGapIsParserPadding(source, &stack, tok) {
			t.Fatalf("realShiftGapIsParserPadding(%q) = false, want true", source[:tok.StartByte])
		}
	}
}

func TestRealShiftGapRejectsNonLeadingBOM(t *testing.T) {
	source := []byte("call(arg1\xef\xbb\xbf)")
	stack := newGLRStack(1)
	stack.byteOffset = uint32(len("call(arg1"))
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source)),
	}

	if realShiftGapIsParserPadding(source, &stack, tok) {
		t.Fatalf("realShiftGapIsParserPadding = true, want false for non-leading BOM gap %q", source[stack.byteOffset:tok.StartByte])
	}
}

func TestRealShiftGapAllowsSyntheticMissingToken(t *testing.T) {
	source := []byte("call(arg1, arg8)")
	stack := newGLRStack(1)
	stack.byteOffset = uint32(len("call(arg1"))
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source) - 1),
		Missing:   true,
	}

	if !realShiftGapIsParserPadding(source, &stack, tok) {
		t.Fatal("realShiftGapIsParserPadding = false, want true for synthetic missing token")
	}
}
