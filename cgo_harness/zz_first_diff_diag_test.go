//go:build cgo && treesitter_c_parity

package cgoharness

// THROWAWAY diagnostic: parse ONE file with the production parser and the C
// oracle, walk both trees in lockstep, and dump the first structural
// difference (type, span, or child count) with full sibling context.
//
//	REPRO_LANG=jq REPRO_FILE=/path/to/builtin.jq \
//	  go test . -tags 'cgo treesitter_c_parity' -run TestFirstDiffDiag -v

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type fddSignature struct {
	File           string
	Path           string
	DiffKind       string
	GoRootType     string
	GoRootStart    uint32
	GoRootEnd      uint32
	GoRootChildren int
	GoRootHasError bool
	CRootKind      string
	CRootStart     uint32
	CRootEnd       uint32
	CRootChildren  int
	CRootHasError  bool
	GoStopReason   string
	GoType         string
	GoStart        uint32
	GoEnd          uint32
	GoChildren     int
	CType          string
	CStart         uint32
	CEnd           uint32
	CChildren      int
	GoText         string
	CText          string
}

func fddTxt(src []byte, s, e uint32) string {
	if int(e) > len(src) {
		e = uint32(len(src))
	}
	if s > e {
		s = e
	}
	r := src[s:e]
	if len(r) > 80 {
		r = r[:80]
	}
	return fmt.Sprintf("%q", string(r))
}

func fddExcerpt(src []byte, s, e uint32) string {
	if int(e) > len(src) {
		e = uint32(len(src))
	}
	if s > e {
		s = e
	}
	const context = 24
	start, end := s, e
	if start > context {
		start -= context
	} else {
		start = 0
	}
	if maxEnd := uint32(len(src)); end+context < maxEnd {
		end += context
	} else {
		end = maxEnd
	}
	r := src[start:end]
	if len(r) > 120 {
		r = r[:120]
	}
	return string(r)
}

func fddDiffKind(g *gts.Node, lang *gts.Language, c *sitter.Node) string {
	switch {
	case g.Type(lang) != c.Kind():
		return "type"
	case g.StartByte() != uint32(c.StartByte()) || g.EndByte() != uint32(c.EndByte()):
		return "span"
	case g.ChildCount() != int(c.ChildCount()):
		return "child-count"
	case g.IsNamed() != c.IsNamed():
		return "named"
	case g.IsMissing() != c.IsMissing():
		return "missing"
	default:
		return ""
	}
}

func fddFirst(g *gts.Node, lang *gts.Language, c *sitter.Node, path string) *fddSignature {
	if kind := fddDiffKind(g, lang, c); kind != "" {
		return &fddSignature{
			Path:       path,
			DiffKind:   kind,
			GoType:     g.Type(lang),
			GoStart:    g.StartByte(),
			GoEnd:      g.EndByte(),
			GoChildren: g.ChildCount(),
			CType:      c.Kind(),
			CStart:     uint32(c.StartByte()),
			CEnd:       uint32(c.EndByte()),
			CChildren:  int(c.ChildCount()),
		}
	}
	for i := 0; i < g.ChildCount(); i++ {
		childPath := fmt.Sprintf("%s[%d]", path, i)
		gChild := g.Child(i)
		cChild := c.Child(uint(i))
		if parityCompareFields {
			goField := g.FieldNameForChild(i, lang)
			cField := c.FieldNameForChild(uint32(i))
			if goField != cField {
				return &fddSignature{
					Path:       childPath,
					DiffKind:   "field-name",
					GoType:     gChild.Type(lang),
					GoStart:    gChild.StartByte(),
					GoEnd:      gChild.EndByte(),
					GoChildren: gChild.ChildCount(),
					CType:      cChild.Kind(),
					CStart:     uint32(cChild.StartByte()),
					CEnd:       uint32(cChild.EndByte()),
					CChildren:  int(cChild.ChildCount()),
				}
			}
		}
		if sig := fddFirst(gChild, lang, cChild, childPath); sig != nil {
			return sig
		}
	}
	return nil
}

func fddBuildSignature(file string, goRoot *gts.Node, lang *gts.Language, cRoot *sitter.Node, src []byte, stopReason string) *fddSignature {
	sig := fddFirst(goRoot, lang, cRoot, "root")
	if sig == nil {
		return nil
	}
	sig.File = file
	sig.GoRootType = goRoot.Type(lang)
	sig.GoRootStart = goRoot.StartByte()
	sig.GoRootEnd = goRoot.EndByte()
	sig.GoRootChildren = goRoot.ChildCount()
	sig.GoRootHasError = goRoot.HasError()
	sig.CRootKind = cRoot.Kind()
	sig.CRootStart = uint32(cRoot.StartByte())
	sig.CRootEnd = uint32(cRoot.EndByte())
	sig.CRootChildren = int(cRoot.ChildCount())
	sig.CRootHasError = cRoot.HasError()
	sig.GoStopReason = stopReason
	sig.GoText = fddExcerpt(src, sig.GoStart, sig.GoEnd)
	sig.CText = fddExcerpt(src, sig.CStart, sig.CEnd)
	return sig
}

func fddQuote(s string) string {
	return strconv.Quote(s)
}

func fddPrintSignature(lang string, sig *fddSignature) {
	fmt.Printf("DIVERGE-SIG lang=%s file=%s base=%s goRoot=%s goRootSpan=%d:%d goRootCC=%d goRootErr=%v cRoot=%s cRootSpan=%d:%d cRootCC=%d cRootErr=%v goStop=%s path=%s diff=%s goType=%s cType=%s goSpan=%d:%d cSpan=%d:%d goCC=%d cCC=%d goText=%s cText=%s\n",
		fddQuote(lang),
		fddQuote(sig.File),
		fddQuote(filepath.Base(sig.File)),
		fddQuote(sig.GoRootType),
		sig.GoRootStart,
		sig.GoRootEnd,
		sig.GoRootChildren,
		sig.GoRootHasError,
		fddQuote(sig.CRootKind),
		sig.CRootStart,
		sig.CRootEnd,
		sig.CRootChildren,
		sig.CRootHasError,
		fddQuote(sig.GoStopReason),
		fddQuote(sig.Path),
		fddQuote(sig.DiffKind),
		fddQuote(sig.GoType),
		fddQuote(sig.CType),
		sig.GoStart,
		sig.GoEnd,
		sig.CStart,
		sig.CEnd,
		sig.GoChildren,
		sig.CChildren,
		fddQuote(sig.GoText),
		fddQuote(sig.CText),
	)
}

func fddDumpBoth(g *gts.Node, lang *gts.Language, c *sitter.Node, src []byte, path string, t *testing.T) {
	t.Logf("  FIRST-DIFF @%s", path)
	t.Logf("    go: type=%q [%d:%d] cc=%d", g.Type(lang), g.StartByte(), g.EndByte(), g.ChildCount())
	t.Logf("    c : kind=%q [%d:%d] cc=%d", c.Kind(), c.StartByte(), c.EndByte(), int(c.ChildCount()))
	for i := 0; i < g.ChildCount(); i++ {
		ch := g.Child(i)
		t.Logf("      go.child[%d]: type=%q [%d:%d] named=%v %s", i, ch.Type(lang), ch.StartByte(), ch.EndByte(), ch.IsNamed(), fddTxt(src, ch.StartByte(), ch.EndByte()))
	}
	for i := 0; i < int(c.ChildCount()); i++ {
		ch := c.Child(uint(i))
		t.Logf("      c.child[%d]: kind=%q [%d:%d] named=%v %s", i, ch.Kind(), ch.StartByte(), ch.EndByte(), ch.IsNamed(), fddTxt(src, uint32(ch.StartByte()), uint32(ch.EndByte())))
	}
}

func fddWalk(g *gts.Node, lang *gts.Language, c *sitter.Node, src []byte, path string, t *testing.T) bool {
	if fddDiffKind(g, lang, c) != "" {
		fddDumpBoth(g, lang, c, src, path, t)
		return true
	}
	for i := 0; i < g.ChildCount(); i++ {
		if fddWalk(g.Child(i), lang, c.Child(uint(i)), src, fmt.Sprintf("%s[%d]", path, i), t) {
			return true
		}
	}
	return false
}

func TestFirstDiffDiag(t *testing.T) {
	name := os.Getenv("REPRO_LANG")
	file := os.Getenv("REPRO_FILE")
	if name == "" || file == "" {
		t.Skip("set REPRO_LANG and REPRO_FILE")
	}
	var goLang *gts.Language
	for _, e := range grammars.AllLanguages() {
		if e.Name == name {
			goLang = e.Language()
			break
		}
	}
	if goLang == nil {
		t.Fatalf("%s: not in grammars.AllLanguages", name)
	}
	cLang, err := ParityCLanguage(name)
	if err != nil {
		t.Fatalf("%s: no C reference: %v", name, err)
	}
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if os.Getenv("REPRO_DEBUG_DFA") == "1" {
		gts.DebugDFA.Store(true)
		defer gts.DebugDFA.Store(false)
	}
	gp := gts.NewParser(goLang)
	if os.Getenv("REPRO_GLR_TRACE") == "1" {
		gp.SetGLRTrace(true)
	}
	tr, _ := gp.Parse(src)
	if tr == nil || tr.RootNode() == nil {
		t.Fatalf("go parse failed")
	}
	defer tr.Release()
	cp := sitter.NewParser()
	defer cp.Close()
	_ = cp.SetLanguage(cLang)
	ct := cp.Parse(src, nil)
	if ct == nil || ct.RootNode() == nil {
		t.Fatalf("c parse failed")
	}
	defer ct.Close()
	t.Logf("=== %s (%d bytes) ===", file, len(src))
	t.Logf("  go stopReason=%v rootHasError=%v cRootHasError=%v", tr.ParseStopReason(), tr.RootNode().HasError(), ct.RootNode().HasError())
	if !fddWalk(tr.RootNode(), goLang, ct.RootNode(), src, "root", t) {
		t.Logf("  (no structural divergence)")
	}
}
