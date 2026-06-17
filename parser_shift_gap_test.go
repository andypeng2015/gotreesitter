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

func TestRealTokenAttachmentGapRejectsCommentSource(t *testing.T) {
	source := []byte("call(arg1/*c*/)")
	stack := newGLRStack(1)
	stack.byteOffset = uint32(len("call(arg1"))
	tok := Token{
		Symbol:    1,
		StartByte: uint32(len(source) - 1),
		EndByte:   uint32(len(source)),
	}

	if realTokenAttachmentGapIsParserPadding(source, &stack, tok) {
		t.Fatalf("realTokenAttachmentGapIsParserPadding = true, want false for comment gap %q", source[stack.byteOffset:tok.StartByte])
	}

	parser := &Parser{glrTrace: false}
	if parser.guardRealTokenAttachmentGap(source, &stack, tok, "test") {
		t.Fatal("guardRealTokenAttachmentGap = true, want false")
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

func TestRealShiftGapAllowsNoLookaheadToken(t *testing.T) {
	source := []byte("call(arg1/*c*/)")
	stack := newGLRStack(1)
	stack.byteOffset = uint32(len("call(arg1"))
	tok := Token{
		Symbol:      1,
		StartByte:   uint32(len(source) - 1),
		EndByte:     uint32(len(source)),
		NoLookahead: true,
	}

	if !realShiftGapIsParserPadding(source, &stack, tok) {
		t.Fatal("realShiftGapIsParserPadding = false, want true for NoLookahead token")
	}

	parser := &Parser{glrTrace: false}
	if !parser.guardRealTokenAttachmentGap(source, &stack, tok, "test") {
		t.Fatal("guardRealTokenAttachmentGap = false, want true for NoLookahead token")
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

func TestRecoverActionRejectsCommentGap(t *testing.T) {
	source := []byte("1/*c*/*2")
	parser := NewParser(buildArithmeticRecoverLanguage())
	tree, err := parser.ParseWithTokenSource(source, &recoverCommentGapTokenSource{src: source})
	if err != nil {
		t.Fatalf("ParseWithTokenSource failed: %v", err)
	}

	if got, want := tree.ParseRuntime().StopReason, ParseStopNoStacksAlive; got != want {
		t.Fatalf("parse stop reason = %q, want %q; tree=%v", got, want, tree.RootNode())
	}
	if root := tree.RootNode(); root != nil && root.EndByte() > 1 {
		t.Fatalf("recovery attached stale-gap token; root end byte = %d, want <= 1", root.EndByte())
	}
}

type recoverCommentGapTokenSource struct {
	src []byte
	pos int
	row uint32
	col uint32
}

func (ts *recoverCommentGapTokenSource) Next() Token {
	for ts.pos < len(ts.src) {
		switch ts.src[ts.pos] {
		case ' ', '\t', '\n':
			ts.advance()
			continue
		case '/':
			if ts.pos+1 < len(ts.src) && ts.src[ts.pos+1] == '*' {
				ts.skipBlockComment()
				continue
			}
		case '+':
			return ts.singleByteToken(2)
		case '*':
			return ts.singleByteToken(3)
		}
		if ts.src[ts.pos] >= '0' && ts.src[ts.pos] <= '9' {
			return ts.numberToken()
		}
		ts.advance()
	}
	pt := Point{Row: ts.row, Column: ts.col}
	return Token{StartByte: uint32(ts.pos), EndByte: uint32(ts.pos), StartPoint: pt, EndPoint: pt}
}

func (ts *recoverCommentGapTokenSource) singleByteToken(sym Symbol) Token {
	start := ts.pos
	startPt := Point{Row: ts.row, Column: ts.col}
	ts.advance()
	return Token{
		Symbol:     sym,
		StartByte:  uint32(start),
		EndByte:    uint32(ts.pos),
		StartPoint: startPt,
		EndPoint:   Point{Row: ts.row, Column: ts.col},
		Text:       string(ts.src[start:ts.pos]),
	}
}

func (ts *recoverCommentGapTokenSource) numberToken() Token {
	start := ts.pos
	startPt := Point{Row: ts.row, Column: ts.col}
	for ts.pos < len(ts.src) && ts.src[ts.pos] >= '0' && ts.src[ts.pos] <= '9' {
		ts.advance()
	}
	return Token{
		Symbol:     1,
		StartByte:  uint32(start),
		EndByte:    uint32(ts.pos),
		StartPoint: startPt,
		EndPoint:   Point{Row: ts.row, Column: ts.col},
		Text:       string(ts.src[start:ts.pos]),
	}
}

func (ts *recoverCommentGapTokenSource) skipBlockComment() {
	ts.advance()
	ts.advance()
	for ts.pos < len(ts.src) {
		if ts.src[ts.pos] == '*' && ts.pos+1 < len(ts.src) && ts.src[ts.pos+1] == '/' {
			ts.advance()
			ts.advance()
			return
		}
		ts.advance()
	}
}

func (ts *recoverCommentGapTokenSource) advance() {
	if ts.pos >= len(ts.src) {
		return
	}
	if ts.src[ts.pos] == '\n' {
		ts.row++
		ts.col = 0
		ts.pos++
		return
	}
	ts.pos++
	ts.col++
}
