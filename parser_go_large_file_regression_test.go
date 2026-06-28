package gotreesitter_test

import (
	"os"
	"path/filepath"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestGoCobraLargeFileParseRegression(t *testing.T) {
	root := os.Getenv("GTS_COBRA_REGRESSION_ROOT")
	if root == "" {
		t.Skip("GTS_COBRA_REGRESSION_ROOT not set")
	}
	lang := grammars.GoLanguage()
	for _, name := range []string{"command.go", "command_test.go", "completions_test.go"} {
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			tree, err := gotreesitter.NewParserPool(lang).Parse(src)
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}
			if tree == nil || tree.RootNode() == nil {
				t.Fatal("Parse returned nil tree/root")
			}
			defer tree.Release()
			if got, want := tree.RootNode().Type(lang), "source_file"; got != want {
				t.Fatalf("root type = %q, want %q", got, want)
			}
			if got := tree.ParseStopReason(); got != gotreesitter.ParseStopAccepted {
				t.Fatalf("ParseStopReason = %s, want %s", got, gotreesitter.ParseStopAccepted)
			}
		})
	}
}
