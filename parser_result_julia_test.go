package gotreesitter_test

import (
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestJuliaReturnRangeRecoveryCompatibility(t *testing.T) {
	lang := grammars.JuliaLanguage()
	if lang == nil {
		t.Fatal("JuliaLanguage returned nil")
	}
	source := []byte("function f()\n    while true\n        return a : b\n    end\n    return (lo + 1) : (hi - 1)\nend\n")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("Parse returned nil tree")
	}
	defer tree.Release()

	bareReturn := findNodeByText(tree.RootNode(), lang, source, "return_statement", "return a : b")
	if bareReturn == nil {
		t.Fatalf("bare return_statement not found:\n%s", tree.RootNode().SExpr(lang))
	}
	if got, want := bareReturn.ChildCount(), 3; got != want {
		t.Fatalf("bare return child count = %d, want %d; tree:\n%s", got, want, tree.RootNode().SExpr(lang))
	}
	if got := bareReturn.Child(1).Type(lang); got != "ERROR" {
		t.Fatalf("bare return child[1] = %q, want ERROR", got)
	}
	if got := bareReturn.Child(2).Type(lang); got != "quote_expression" {
		t.Fatalf("bare return child[2] = %q, want quote_expression", got)
	}

	parenReturn := findNodeByText(tree.RootNode(), lang, source, "return_statement", "return (lo + 1) : (hi - 1")
	if parenReturn == nil {
		t.Fatalf("parenthesized return_statement not found:\n%s", tree.RootNode().SExpr(lang))
	}
	if got, want := parenReturn.EndByte(), uint32(86); got != want {
		t.Fatalf("parenthesized return end = %d, want %d", got, want)
	}
	parent := parenReturn.Parent()
	if parent == nil || parent.Type(lang) != "block" {
		t.Fatalf("parenthesized return parent = %v, want block", parent)
	}
	if got, want := parent.ChildCount(), 3; got != want {
		t.Fatalf("block child count = %d, want %d; tree:\n%s", got, want, tree.RootNode().SExpr(lang))
	}
	trailing := parent.Child(2)
	if got := trailing.Type(lang); got != "ERROR" {
		t.Fatalf("block trailing child = %q, want ERROR", got)
	}
	if got := trailing.Text(source); got != ")" {
		t.Fatalf("block trailing ERROR text = %q, want %q", got, ")")
	}
}

func TestJuliaMacroArgumentJuxtapositionCompatibility(t *testing.T) {
	lang := grammars.JuliaLanguage()
	if lang == nil {
		t.Fatal("JuliaLanguage returned nil")
	}
	source := []byte("@assert 3nstmts == length(di.codelocs)\n@assert workspace.visiting[child] == length(workspace.stack) + 1 \"internal error maintaining workspace\"\n")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("Parse returned nil tree")
	}
	defer tree.Release()

	leading := findNodeByText(tree.RootNode(), lang, source, "macro_argument_list", "3nstmts == length(di.codelocs)")
	if leading == nil {
		t.Fatalf("leading macro_argument_list not found:\n%s", tree.RootNode().SExpr(lang))
	}
	if got, want := leading.ChildCount(), 1; got != want {
		t.Fatalf("leading macro arg child count = %d, want %d; tree:\n%s", got, want, tree.RootNode().SExpr(lang))
	}
	leadingExpr := leading.Child(0)
	if got := leadingExpr.Type(lang); got != "binary_expression" {
		t.Fatalf("leading child type = %q, want binary_expression", got)
	}
	leadingJuxtaposition := findNodeByText(leadingExpr, lang, source, "juxtaposition_expression", "3nstmts")
	if leadingJuxtaposition == nil {
		t.Fatalf("leading juxtaposition not found:\n%s", leadingExpr.SExpr(lang))
	}

	trailing := findNodeByText(tree.RootNode(), lang, source, "macro_argument_list", "workspace.visiting[child] == length(workspace.stack) + 1 \"internal error maintaining workspace\"")
	if trailing == nil {
		t.Fatalf("trailing macro_argument_list not found:\n%s", tree.RootNode().SExpr(lang))
	}
	if got, want := trailing.ChildCount(), 1; got != want {
		t.Fatalf("trailing macro arg child count = %d, want %d; tree:\n%s", got, want, tree.RootNode().SExpr(lang))
	}
	trailingJuxtaposition := findNodeByText(trailing.Child(0), lang, source, "juxtaposition_expression", "1 \"internal error maintaining workspace\"")
	if trailingJuxtaposition == nil {
		t.Fatalf("trailing juxtaposition not found:\n%s", trailing.Child(0).SExpr(lang))
	}
}

func findNodeByText(root *gotreesitter.Node, lang *gotreesitter.Language, source []byte, typ, text string) *gotreesitter.Node {
	if root == nil {
		return nil
	}
	if root.Type(lang) == typ && root.Text(source) == text {
		return root
	}
	for i := 0; i < root.ChildCount(); i++ {
		if found := findNodeByText(root.Child(i), lang, source, typ, text); found != nil {
			return found
		}
	}
	return nil
}
