//go:build !grammar_subset || grammar_subset_cobol

package grammars

import (
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestCobolFixedFormatCommentCorpusBlockParsesClean(t *testing.T) {
	src := []byte("aaaaaa identification division.\n" +
		"aaaaaa program-id. a.  ,,, ;;;                                          aaaaa\n" +
		"      *aaaa\n")

	tree, err := gotreesitter.NewParser(CobolLanguage()).Parse(src)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("Parse returned nil tree/root")
	}
	if tree.RootNode().HasError() {
		t.Fatalf("COBOL corpus block parsed with error: %s", tree.RootNode().SExpr(CobolLanguage()))
	}
	if got, want := tree.RootNode().Type(CobolLanguage()), "start"; got != want {
		t.Fatalf("root type = %q, want %q", got, want)
	}
}
