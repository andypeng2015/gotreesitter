package gotreesitter

import "testing"

func TestNextGLRUnionDFATokenScoresOnlyStatesProducingCandidate(t *testing.T) {
	lang := &Language{
		Name:        "glr-union-route-test",
		SymbolNames: []string{"EOF", "keyword", "identifier"},
		LexStates: []LexState{
			{Default: -1, EOF: -1},
			{Default: -1, EOF: -1, Transitions: []LexTransition{{Lo: 'a', Hi: 'a', NextState: 2}}},
			{Default: -1, EOF: -1, Transitions: []LexTransition{{Lo: 'b', Hi: 'b', NextState: 3}}},
			{AcceptToken: 1, Default: -1, EOF: -1},
			{Default: -1, EOF: -1, Transitions: []LexTransition{{Lo: 'a', Hi: 'a', NextState: 5}}},
			{Default: -1, EOF: -1, Transitions: []LexTransition{{Lo: 'b', Hi: 'b', NextState: 6}}},
			{AcceptToken: 2, Default: -1, EOF: -1},
		},
		LexModes: []LexMode{
			{},
			{LexState: 1},
			{LexState: 4},
			{LexState: 4},
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 1}}},
		},
	}
	lookup := func(state StateID, sym Symbol) uint16 {
		switch sym {
		case 1:
			if state == 1 || state == 2 || state == 3 {
				return 1
			}
		case 2:
			if state == 2 || state == 3 {
				return 1
			}
		}
		return 0
	}

	ts := acquireDFATokenSource(NewLexer(lang.LexStates, []byte("ab")), lang, lookup, nil, nil)
	defer ts.Close()
	ts.SetParserState(1)
	ts.SetGLRStates([]StateID{1, 2, 3})

	tok, ok := ts.nextGLRUnionDFAToken()
	if !ok {
		t.Fatal("nextGLRUnionDFAToken returned false")
	}
	if got, want := tok.Symbol, Symbol(2); got != want {
		t.Fatalf("token symbol = %d (%q), want %d (%q)", got, lang.SymbolNames[got], want, lang.SymbolNames[want])
	}
	if tok.StartByte != 0 || tok.EndByte != 2 {
		t.Fatalf("token span = %d..%d, want 0..2", tok.StartByte, tok.EndByte)
	}
}
