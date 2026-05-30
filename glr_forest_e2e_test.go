package gotreesitter

import "testing"

// TestParseForestJSONEndToEnd drives the GSS-forest GLR loop with real JSON
// tokens and compares the resulting tree to the production parser. JSON has no
// external scanner and state-independent structural lexing, so single-lead-state
// lexing is faithful — a clean first end-to-end exercise of coalesce +
// reduce-over-DAG + the reused node-builder, no deep-equivalence anywhere.
// TestParseForestEndToEnd drives the GSS-forest GLR parser (coalesce +
// reduce-over-DAG, no deep equivalence) through the production token source and
// asserts byte-identical trees vs the production parser across three real
// grammars. Extras (comments) are the next layer and are intentionally absent.
func TestParseForestEndToEnd(t *testing.T) {
	cases := []struct{ lang, src string }{
		{"json", `[1, 2]`},
		{"json", `[]`},
		{"json", `[1, [2, [3, 4]], 5]`},
		{"json", `{"a": 1, "b": [true, false, null]}`},
		{"json", `{"x": {"y": {"z": -3.5}}, "w": "str"}`},
		{"json", `[{"k": [1, 2]}, {"k": []}]`},
		{"go", "package main\n"},
		{"go", "var x = 1\n"},
		{"go", "func f() { return }\n"},
		{"go", "func add(a, b int) int { return a + b }\n"},
		{"c", "int x;\n"},
		{"c", "int f(void) { return 0; }\n"},
		{"c", "struct S { int a; };\n"},
		// NOTE: `struct S { long b; }` needs Stage 3 (forest finalization): the
		// `long b` GLR ambiguity (b as type vs field name) must be resolved by
		// per-link dynamic_precedence, not links[0]. Tracked as the next layer.
	}
	for _, c := range cases {
		lang := loadBlobForDecode(t, c.lang)
		want := mustParseSExpr(t, lang, []byte(c.src))
		got, ok := forestParseSExpr(t, lang, []byte(c.src))
		if !ok {
			t.Errorf("%s %q: forest parse failed", c.lang, c.src)
			continue
		}
		if got != want {
			t.Errorf("%s %q: mismatch\n forest=%s\n normal=%s", c.lang, c.src, got, want)
			continue
		}
		t.Logf("OK  %-5s %q", c.lang, c.src)
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
	root, ok := NewParser(lang).parseForest(newNodeArena(arenaClassFull), src)
	if !ok || root == nil {
		return "", false
	}
	return root.SExpr(lang), true
}
