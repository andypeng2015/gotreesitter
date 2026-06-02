package grammars

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestSwiftLineCommentsStayExtraComments(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte("// header\n//\n// body\nlet x = 1\n")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse swift comments: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("swift comment fixture has parse errors: %s", root.SExpr(lang))
	}
	if got, want := root.NamedChildCount(), 4; got != want {
		t.Fatalf("named child count = %d, want %d; tree: %s", got, want, root.SExpr(lang))
	}
	expectedCommentSpans := [][2]uint32{
		{0, 9},
		{10, 12},
		{13, 20},
	}
	for i, span := range expectedCommentSpans {
		child := root.NamedChild(i)
		if got := child.Type(lang); got != "comment" {
			t.Fatalf("named child %d type = %q, want comment; tree: %s", i, got, root.SExpr(lang))
		}
		if !child.IsExtra() {
			t.Fatalf("named child %d is not extra; tree: %s", i, root.SExpr(lang))
		}
		if got, want := child.StartByte(), span[0]; got != want {
			t.Fatalf("comment %d start = %d, want %d; tree: %s", i, got, want, root.SExpr(lang))
		}
		if got, want := child.EndByte(), span[1]; got != want {
			t.Fatalf("comment %d end = %d, want %d; tree: %s", i, got, want, root.SExpr(lang))
		}
	}
}
