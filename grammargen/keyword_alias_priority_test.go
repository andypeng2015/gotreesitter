package grammargen

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestInlineAliasedKeywordPatternBeatsIdentifierWithoutWord(t *testing.T) {
	g := NewGrammar("alias_keyword_priority")
	g.SetExtras(Pat(`\s+`))
	g.Define("source_file", Sym("statement"))
	g.Define("statement", Seq(
		Alias(Pat("[iI][fF]"), "if", false),
		Sym("identifier"),
	))
	g.Define("identifier", Pat(`[a-zA-Z_][a-zA-Z_]*`))

	lang, err := GenerateLanguage(g)
	if err != nil {
		t.Fatalf("GenerateLanguage: %v", err)
	}

	tree, err := gotreesitter.NewParser(lang).Parse([]byte("IF name"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	root := tree.RootNode()
	if root == nil {
		t.Fatal("root is nil")
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", root.SExpr(lang))
	}
	if got, want := root.SExpr(lang), "(source_file (statement (identifier)))"; got != want {
		t.Fatalf("SExpr = %s, want %s", got, want)
	}
}
