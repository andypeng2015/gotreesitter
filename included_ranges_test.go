package gotreesitter

import "testing"

type stubTokenSource struct {
	tokens     []Token
	i          int
	state      StateID
	nextCalls  int
	skipCalls  int
	relexTok   Token
	relexOK    bool
	canRelex   bool
	relexSeen  Token
	relexCalls int
}

func (s *stubTokenSource) Next() Token {
	s.nextCalls++
	if s.i >= len(s.tokens) {
		return Token{}
	}
	t := s.tokens[s.i]
	s.i++
	return t
}

func (s *stubTokenSource) SkipToByte(offset uint32) Token {
	s.skipCalls++
	for {
		t := s.Next()
		if t.Symbol == 0 || t.StartByte >= offset {
			return t
		}
	}
}

func (s *stubTokenSource) SkipToByteWithPoint(offset uint32, _ Point) Token {
	return s.SkipToByte(offset)
}

func (s *stubTokenSource) SetParserState(state StateID) {
	s.state = state
}

func (s *stubTokenSource) SetGLRStates(states []StateID) {
	// stub: no-op
}

func (s *stubTokenSource) CanRelexFromTokenStart(tok Token) bool {
	return s.canRelex
}

func (s *stubTokenSource) RelexFromTokenStart(tok Token) (Token, bool) {
	s.relexCalls++
	s.relexSeen = tok
	if !s.relexOK {
		return Token{}, false
	}
	return s.relexTok, true
}

func TestNormalizeIncludedRanges(t *testing.T) {
	in := []Range{
		{StartByte: 20, EndByte: 30},
		{StartByte: 10, EndByte: 15},
		{StartByte: 15, EndByte: 18},
		{StartByte: 18, EndByte: 18}, // empty, dropped
		{StartByte: 28, EndByte: 35}, // merge with 20-30
	}
	out := normalizeIncludedRanges(in)
	if len(out) != 2 {
		t.Fatalf("normalize len: got %d, want 2", len(out))
	}
	if out[0].StartByte != 10 || out[0].EndByte != 18 {
		t.Fatalf("range0: got %d-%d, want 10-18", out[0].StartByte, out[0].EndByte)
	}
	if out[1].StartByte != 20 || out[1].EndByte != 35 {
		t.Fatalf("range1: got %d-%d, want 20-35", out[1].StartByte, out[1].EndByte)
	}
}

func TestIncludedRangeTokenSourceFiltersTokens(t *testing.T) {
	base := &stubTokenSource{
		tokens: []Token{
			{Symbol: 1, StartByte: 0, EndByte: 5},
			{Symbol: 2, StartByte: 12, EndByte: 15},
			{Symbol: 3, StartByte: 21, EndByte: 22},
			{},
		},
	}
	ts := newIncludedRangeTokenSource(base, []Range{{StartByte: 10, EndByte: 20}}).(*includedRangeTokenSource)

	tok := ts.Next()
	if tok.Symbol != 2 {
		t.Fatalf("first token: got %d, want 2", tok.Symbol)
	}
	tok = ts.Next()
	if tok.Symbol != 0 {
		t.Fatalf("second token should be EOF-like, got %d", tok.Symbol)
	}
}

func TestIncludedRangeTokenSourceDelegatesParserState(t *testing.T) {
	base := &stubTokenSource{
		tokens: []Token{{}},
	}
	ts := newIncludedRangeTokenSource(base, []Range{{StartByte: 0, EndByte: 1}}).(*includedRangeTokenSource)
	ts.SetParserState(42)
	if base.state != 42 {
		t.Fatalf("delegated parser state: got %d, want 42", base.state)
	}
}

func TestIncludedRangeTokenSourceRelexRejectsOriginalOutsideRangeWithoutCallingBase(t *testing.T) {
	base := &stubTokenSource{
		canRelex: true,
		relexOK:  true,
		relexTok: Token{
			Symbol:     1,
			StartByte:  0,
			EndByte:    5,
			StartPoint: Point{},
			EndPoint:   Point{Column: 5},
		},
	}
	ts := newIncludedRangeTokenSource(base, []Range{{StartByte: 10, EndByte: 20}}).(*includedRangeTokenSource)

	tok := Token{
		Symbol:     1,
		StartByte:  0,
		EndByte:    5,
		StartPoint: Point{},
		EndPoint:   Point{Column: 5},
	}
	if got, ok := ts.RelexFromTokenStart(tok); ok {
		t.Fatalf("RelexFromTokenStart = (%+v, true), want false for original token outside included range", got)
	}
	if base.relexCalls != 0 {
		t.Fatalf("base RelexFromTokenStart calls = %d, want 0", base.relexCalls)
	}
	if base.nextCalls != 0 || base.skipCalls != 0 || base.i != 0 {
		t.Fatalf("rejected relex advanced base: next=%d skip=%d i=%d, want 0/0/0", base.nextCalls, base.skipCalls, base.i)
	}
	if ts.idx != 0 {
		t.Fatalf("rejected relex changed included range index to %d, want 0", ts.idx)
	}
}

func TestIncludedRangeTokenSourceRelexRejectsBaseStartChangeWithoutFiltering(t *testing.T) {
	base := &stubTokenSource{
		tokens: []Token{
			{
				Symbol:     3,
				StartByte:  12,
				EndByte:    14,
				StartPoint: Point{Column: 12},
				EndPoint:   Point{Column: 14},
			},
		},
		canRelex: true,
		relexOK:  true,
		relexTok: Token{
			Symbol:     2,
			StartByte:  11,
			EndByte:    12,
			StartPoint: Point{Column: 11},
			EndPoint:   Point{Column: 12},
		},
	}
	ts := newIncludedRangeTokenSource(base, []Range{{StartByte: 10, EndByte: 20}}).(*includedRangeTokenSource)

	tok := Token{
		Symbol:     1,
		StartByte:  10,
		EndByte:    11,
		StartPoint: Point{Column: 10},
		EndPoint:   Point{Column: 11},
	}
	if got, ok := ts.RelexFromTokenStart(tok); ok {
		t.Fatalf("RelexFromTokenStart = (%+v, true), want false after base changes start", got)
	}
	if base.relexCalls != 1 {
		t.Fatalf("base RelexFromTokenStart calls = %d, want 1", base.relexCalls)
	}
	if base.relexSeen.StartByte != tok.StartByte || base.relexSeen.StartPoint != tok.StartPoint {
		t.Fatalf("base saw token %+v, want %+v", base.relexSeen, tok)
	}
	if base.nextCalls != 0 || base.skipCalls != 0 || base.i != 0 {
		t.Fatalf("rejected relex advanced base through filtering: next=%d skip=%d i=%d, want 0/0/0", base.nextCalls, base.skipCalls, base.i)
	}
	if ts.idx != 0 {
		t.Fatalf("rejected relex changed included range index to %d, want 0", ts.idx)
	}
}

func TestParserSetIncludedRangesRoundTrip(t *testing.T) {
	p := NewParser(nil)
	p.SetIncludedRanges([]Range{
		{StartByte: 5, EndByte: 8},
		{StartByte: 1, EndByte: 3},
	})
	got := p.IncludedRanges()
	if len(got) != 2 {
		t.Fatalf("IncludedRanges len: got %d, want 2", len(got))
	}
	if got[0].StartByte != 1 || got[1].StartByte != 5 {
		t.Fatalf("IncludedRanges not sorted: got %v", got)
	}
}
