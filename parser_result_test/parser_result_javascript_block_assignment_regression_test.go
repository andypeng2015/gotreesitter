package parserresult_test

import (
	"strings"
	"testing"
)

func TestJavaScriptStandaloneBlockBeforeSimpleAssignment(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "simple assignment", src: "{a}b=c"},
		{name: "compound assignment", src: "{a}b+=c"},
		{name: "explicit semicolon", src: "{a};b=c"},
		{name: "call expression", src: "{a}b()"},
		{name: "identifier expression", src: "{a}b"},
		{name: "if block assignment", src: "if(x){}y=z"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tree, lang := parseByLanguageName(t, "javascript", tc.src)
			root := tree.RootNode()
			if got := root.Type(lang); got != "program" {
				t.Fatalf("root type = %q, want program: %s", got, root.SExpr(lang))
			}
			if root.HasError() {
				t.Fatalf("unexpected javascript parse error: %s", root.SExpr(lang))
			}
			if tc.src == "{a}b=c" {
				sexpr := root.SExpr(lang)
				if !strings.Contains(sexpr, "(statement_block") {
					t.Fatalf("standalone block missing statement_block: %s", sexpr)
				}
				if !strings.Contains(sexpr, "(assignment_expression") {
					t.Fatalf("assignment tail missing assignment_expression: %s", sexpr)
				}
			}
		})
	}
}

func TestJavaScriptJSXAttributeAfterExpressionStillParses(t *testing.T) {
	tests := []string{
		"<A a={b} c={d} />",
		"<A>{b}</A>",
	}

	for _, src := range tests {
		t.Run(src, func(t *testing.T) {
			tree, lang := parseByLanguageName(t, "javascript", src)
			root := tree.RootNode()
			if got := root.Type(lang); got != "program" {
				t.Fatalf("root type = %q, want program: %s", got, root.SExpr(lang))
			}
			if root.HasError() {
				t.Fatalf("unexpected javascript JSX parse error: %s", root.SExpr(lang))
			}
		})
	}
}

func TestTypeScriptStandaloneBlockBeforeSimpleAssignment(t *testing.T) {
	for _, langName := range []string{"typescript", "tsx"} {
		t.Run(langName, func(t *testing.T) {
			tree, lang := parseByLanguageName(t, langName, "{a}b=c")
			root := tree.RootNode()
			if got := root.Type(lang); got != "program" {
				t.Fatalf("%s root type = %q, want program: %s", langName, got, root.SExpr(lang))
			}
			if root.HasError() {
				t.Fatalf("unexpected %s parse error: %s", langName, root.SExpr(lang))
			}
		})
	}
}
