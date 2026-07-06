package grammars

// Locates ERROR/MISSING nodes in a TypeScript parse (gated; not CI).
// GTS_WW_ERR=1 GTS_WW_FILE=<path> go test ./grammars -run TestLocateTSErrors -v

import (
	"fmt"
	"os"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestLocateTSErrors(t *testing.T) {
	if os.Getenv("GTS_WW_ERR") != "1" {
		t.Skip("set GTS_WW_ERR=1")
	}
	path := os.Getenv("GTS_WW_FILE")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lang := TypescriptLanguage()
	p := gotreesitter.NewParser(lang)
	tree, err := p.Parse(src)
	if err != nil || tree == nil {
		t.Fatalf("parse: %v", err)
	}
	root := tree.RootNode()
	rt := tree.ParseRuntime()
	fmt.Printf("WW hasErr=%v stop=%v maxStacks=%d nodes=%d rootEnd=%d srcLen=%d\n",
		root.HasError(), rt.StopReason, rt.MaxStacksSeen, rt.NodesAllocated, rt.RootEndByte, len(src))
	found := 0
	var walk func(n *gotreesitter.Node, depth int)
	walk = func(n *gotreesitter.Node, depth int) {
		if found >= 12 {
			return
		}
		if n.IsError() || n.IsMissing() {
			found++
			s := int(n.StartByte())
			e := int(n.EndByte())
			ctxLo := s - 80
			if ctxLo < 0 {
				ctxLo = 0
			}
			ctxHi := e + 80
			if ctxHi > len(src) {
				ctxHi = len(src)
			}
			snip := e
			if snip > s+120 {
				snip = s + 120
			}
			fmt.Printf("ERRNODE kind=%s missing=%v bytes=%d-%d text=%q context=%q\n",
				n.Type(lang), n.IsMissing(), s, e, string(src[s:snip]), string(src[ctxLo:ctxHi]))
			return
		}
		if !n.HasError() {
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			walk(n.Child(i), depth+1)
		}
	}
	walk(root, 0)
	fmt.Printf("WW total error nodes shown=%d\n", found)
}
