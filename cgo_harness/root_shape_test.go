//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"fmt"
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TestRootShapeParity compares the root node structure produced by C tree-sitter
// vs gotreesitter for various error scenarios. This validates the correctness of
// the synthetic root wrapping logic in buildSyntheticRootTree.
func TestRootShapeParity(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "fully_invalid_expression",
			source: `x.Query("..." + input)`,
		},
		{
			name:   "xgrep_pattern",
			source: `__METAVAR_DB.Query("..." + __METAVAR_INPUT)`,
		},
		{
			name:   "partial_valid_package_then_error",
			source: "package main\nfunc broken",
		},
		{
			name:   "valid_with_embedded_error",
			source: "package main\nfunc f() { x. }\nfunc g() {}",
		},
		{
			name:   "fully_valid",
			source: "package main\nfunc f() {}\n",
		},
		{
			name:   "single_identifier",
			source: "x",
		},
		{
			name:   "valid_package_invalid_func_valid_func",
			source: "package main\nimport \"fmt\"\nfunc bad( {}\nfunc good() { fmt.Println() }\n",
		},
	}

	// Load Go language via grammars registry
	entry, ok := parityEntriesByName["go"]
	if !ok {
		t.Fatal("missing Go language in grammars registry")
	}
	goLang := entry.Language()

	cLang, err := ParityCLanguage("go")
	if err != nil {
		t.Fatalf("load C Go language: %v", err)
	}
	_ = grammars.AllLanguages() // ensure init

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.source)

			// C tree-sitter
			cParser := sitter.NewParser()
			defer cParser.Close()
			if err := cParser.SetLanguage(cLang); err != nil {
				t.Fatalf("C parser SetLanguage: %v", err)
			}
			cTree := cParser.Parse(src, nil)
			if cTree == nil || cTree.RootNode() == nil {
				t.Fatal("C parser returned nil tree")
			}
			defer cTree.Close()
			cRoot := cTree.RootNode()

			// gotreesitter
			goPool := gotreesitter.NewParserPool(goLang)
			goTree, err := goPool.Parse(src)
			if err != nil {
				t.Fatalf("Go parse error: %v", err)
			}
			defer goTree.Release()
			goRoot := goTree.RootNode()

			// Print both trees
			cStr := dumpCNodeNew(cRoot, 0)
			goStr := dumpGoNodeNew(goRoot, goLang, 0)

			t.Logf("Input: %q\n\n=== C tree-sitter ===\n%s\n=== gotreesitter ===\n%s", tc.source, cStr, goStr)

			// Use the existing full parity comparison
			var errs []string
			compareNodes(goRoot, goLang, cRoot, "root", &errs)
			if len(errs) > 0 {
				for _, e := range errs {
					t.Errorf("  %s", e)
				}
			}
		})
	}
}

func dumpCNodeNew(n *sitter.Node, depth int) string {
	if n == nil {
		return ""
	}
	indent := strings.Repeat("  ", depth)
	result := fmt.Sprintf("%s%s [%d-%d]", indent, n.Kind(), n.StartByte(), n.EndByte())
	if n.IsNamed() {
		result += " (named)"
	}
	if n.HasError() {
		result += " [HAS_ERROR]"
	}
	result += "\n"
	if depth > 5 {
		if n.ChildCount() > 0 {
			result += indent + "  ...\n"
		}
		return result
	}
	for i := uint(0); i < uint(n.ChildCount()); i++ {
		child := n.Child(i)
		if child != nil {
			result += dumpCNodeNew(child, depth+1)
		}
	}
	return result
}

func dumpGoNodeNew(n *gotreesitter.Node, lang *gotreesitter.Language, depth int) string {
	if n == nil {
		return ""
	}
	indent := strings.Repeat("  ", depth)
	result := fmt.Sprintf("%s%s [%d-%d]", indent, n.Type(lang), n.StartByte(), n.EndByte())
	if n.IsNamed() {
		result += " (named)"
	}
	if n.HasError() {
		result += " [HAS_ERROR]"
	}
	result += "\n"
	if depth > 5 {
		if n.ChildCount() > 0 {
			result += indent + "  ...\n"
		}
		return result
	}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child != nil {
			result += dumpGoNodeNew(child, lang, depth+1)
		}
	}
	return result
}
