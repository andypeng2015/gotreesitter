package gotreesitter

import "testing"

// TestParseForestJSONEndToEnd drives the GSS-forest GLR loop with real JSON
// tokens and compares the resulting tree to the production parser. JSON has no
// external scanner and state-independent structural lexing, so single-lead-state
// lexing is faithful — a clean first end-to-end exercise of coalesce +
// reduce-over-DAG + the reused node-builder, no deep-equivalence anywhere.
func TestParseForestJSONEndToEnd(t *testing.T) {
	lang := loadBlobForDecode(t, "json")
	cases := []string{
		`[1, 2]`,
		`[]`,
		`[1, [2, [3, 4]], 5]`,
		`{"a": 1, "b": [true, false, null]}`,
		`{"x": {"y": {"z": -3.5}}, "w": "str"}`,
		`[{"k": [1, 2]}, {"k": []}]`,
	}
	for _, src := range cases {
		want := mustParseSExpr(t, lang, []byte(src))
		got, ok := forestParseSExpr(t, lang, []byte(src))
		if !ok {
			t.Errorf("%s: forest parse failed", src)
			continue
		}
		if got != want {
			t.Errorf("%s: mismatch\n forest=%s\n normal=%s", src, got, want)
			continue
		}
		t.Logf("OK  %-40s %s", src, got)
	}
}

func mustParseSExpr(t *testing.T, lang *Language, src []byte) string {
	tree, err := NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("normal parse %q: %v", src, err)
	}
	return tree.RootNode().SExpr(lang)
}

func forestParseSExpr(t *testing.T, lang *Language, src []byte) (string, bool) {
	lx := &Lexer{
		states:          lang.LexStates,
		asciiTable:      lang.LexAsciiTable(),
		immediateTokens: lang.ImmediateTokens,
		zeroWidthTokens: lang.ZeroWidthTokens,
		source:          src,
	}
	nextToken := func(leadState StateID) Token {
		if int(leadState) >= len(lang.LexModes) {
			return Token{}
		}
		return lx.Next(lang.LexModes[leadState].LexStateIndex())
	}
	root, ok := NewParser(lang).parseForest(newNodeArena(arenaClassFull), nextToken)
	if !ok || root == nil {
		return "", false
	}
	return root.SExpr(lang), true
}
