package gotreesitter

import "testing"

type recordingParserStateTokenSource struct {
	parserStates []StateID
	glrStates    [][]StateID
}

func (s *recordingParserStateTokenSource) Next() Token { return Token{} }

func (s *recordingParserStateTokenSource) SetParserState(state StateID) {
	s.parserStates = append(s.parserStates, state)
}

func (s *recordingParserStateTokenSource) SetGLRStates(states []StateID) {
	s.glrStates = append(s.glrStates, append([]StateID(nil), states...))
}

func TestUpdateCurrentRelexParserStateTokenSourceExcludesShiftedStacks(t *testing.T) {
	p := &Parser{}
	ts := &recordingParserStateTokenSource{}
	scratch := &parserScratch{}
	stacks := []glrStack{
		{entries: []stackEntry{{state: 10}}, shifted: true},
		{entries: []stackEntry{{state: 20}}},
		{entries: []stackEntry{{state: 30}}, shifted: true},
		{entries: []stackEntry{{state: 40}}},
	}

	if ok := p.updateCurrentRelexParserStateTokenSource(ts, stacks, scratch); !ok {
		t.Fatal("updateCurrentRelexParserStateTokenSource returned false, want true")
	}
	if got, want := len(ts.parserStates), 1; got != want {
		t.Fatalf("SetParserState calls = %d, want %d", got, want)
	}
	if got, want := ts.parserStates[0], StateID(20); got != want {
		t.Fatalf("parser state = %d, want first live unshifted state %d", got, want)
	}
	if got, want := len(ts.glrStates), 1; got != want {
		t.Fatalf("SetGLRStates calls = %d, want %d", got, want)
	}
	wantStates := []StateID{20, 40}
	if len(ts.glrStates[0]) != len(wantStates) {
		t.Fatalf("GLR states = %v, want %v", ts.glrStates[0], wantStates)
	}
	for i, want := range wantStates {
		if ts.glrStates[0][i] != want {
			t.Fatalf("GLR states = %v, want %v", ts.glrStates[0], wantStates)
		}
	}
}
