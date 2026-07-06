package grammars

// Debug harness for the bash perf-cliff campaign (Phase 1 target #1).
// Not a regression test: everything is gated behind GTS_BASH_CLIFF_DEBUG=1.
//
// Usage:
//   GOWORK=off GTS_BASH_CLIFF_DEBUG=1 go test ./grammars -run TestBashCliffDebug -v -count=1 -timeout 0
// Optional:
//   GTS_BASH_CLIFF_FILE=<path>        input file (default corpus small__release.sh)
//   GTS_BASH_CLIFF_CPUPROFILE=<path>  write CPU profile
//   GOT_C_RECOVERY=0                  disable C-recovery gate
//   GOT_GLR_MAX_STACKS=N              override stack cap

import (
	"fmt"
	"os"
	"runtime/pprof"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestBashCliffDebug(t *testing.T) {
	if os.Getenv("GTS_BASH_CLIFF_DEBUG") != "1" {
		t.Skip("set GTS_BASH_CLIFF_DEBUG=1 to run")
	}
	path := os.Getenv("GTS_BASH_CLIFF_FILE")
	if path == "" {
		path = "../cgo_harness/corpus_real/bash/small__release.sh"
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	lang := BashLanguage()
	if lang == nil {
		t.Fatal("nil bash language")
	}

	if prof := os.Getenv("GTS_BASH_CLIFF_CPUPROFILE"); prof != "" {
		f, err := os.Create(prof)
		if err != nil {
			t.Fatalf("create profile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			t.Fatalf("start profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	p := gotreesitter.NewParser(lang)
	if v := os.Getenv("GTS_BASH_CLIFF_TIMEOUT_MS"); v != "" {
		var ms int
		fmt.Sscanf(v, "%d", &ms)
		p.SetTimeoutMicros(uint64(ms) * 1000)
	}
	if os.Getenv("GTS_BASH_CLIFF_GLRTRACE") == "1" {
		p.SetGLRTrace(true)
	}
	start := time.Now()
	tree, err := p.Parse(src)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	root := tree.RootNode()
	rt := tree.ParseRuntime()
	fmt.Printf("BASHCLIFF file=%s bytes=%d elapsed=%v\n", path, len(src), elapsed)
	fmt.Printf("BASHCLIFF hasError=%v rootType=%s rootSpan=[%d,%d) truncated=%v stopReason=%v\n",
		root.HasError(), root.Type(lang), root.StartByte(), root.EndByte(), rt.Truncated, rt.StopReason)
	fmt.Printf("BASHCLIFF maxStacksSeen=%d tokensConsumed=%d mergeIn=%d mergeOut=%d\n",
		rt.MaxStacksSeen, rt.TokensConsumed, rt.MergeStacksIn, rt.MergeStacksOut)
	fmt.Printf("BASHCLIFF singleIter=%d multiIter=%d singleTok=%d multiTok=%d\n",
		rt.SingleStackIterations, rt.MultiStackIterations, rt.SingleStackTokens, rt.MultiStackTokens)
	fmt.Printf("BASHCLIFF loopNs=%d mergeNs=%d cullNs=%d dispatchNs=%d tokenNs=%d\n",
		rt.ParserLoopNanos, rt.GLRMergeNanos, rt.GLRCullNanos, rt.ActionDispatchNanos, rt.TokenNextNanos)
	fmt.Printf("BASHCLIFF arenaBytes=%d gssNodes=%d\n", rt.ArenaBytesAllocated, rt.GSSNodesAllocated)

	// Walk for ERROR/MISSING nodes and print their spans plus source context.
	var walk func(n *gotreesitter.Node, depth int)
	errCount := 0
	walk = func(n *gotreesitter.Node, depth int) {
		if n == nil || errCount > 20 {
			return
		}
		t := n.Type(lang)
		if t == "ERROR" || n.IsMissing() {
			errCount++
			s, e := n.StartByte(), n.EndByte()
			ctxStart := int(s) - 30
			if ctxStart < 0 {
				ctxStart = 0
			}
			ctxEnd := int(e) + 30
			if ctxEnd > len(src) {
				ctxEnd = len(src)
			}
			fmt.Printf("BASHCLIFF ERRNODE type=%s missing=%v span=[%d,%d) ctx=%q\n", t, n.IsMissing(), s, e, string(src[ctxStart:ctxEnd]))
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i), depth+1)
		}
	}
	walk(root, 0)
	fmt.Printf("BASHCLIFF totalErrNodes=%d\n", errCount)
	if os.Getenv("GTS_BASH_CLIFF_SEXPR") == "1" {
		fmt.Printf("BASHCLIFF SEXPR %s\n", root.SExpr(lang))
	}
	// Find deepest node with hasError set.
	var findErr func(n *gotreesitter.Node, depth int)
	findErr = func(n *gotreesitter.Node, depth int) {
		if n == nil || !n.HasError() {
			return
		}
		anyChildErr := false
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if c != nil && c.HasError() {
				anyChildErr = true
				findErr(c, depth+1)
			}
		}
		if !anyChildErr {
			s, e := n.StartByte(), n.EndByte()
			fmt.Printf("BASHCLIFF DEEPEST-HASERR type=%s span=[%d,%d) depth=%d childCount=%d src=%q\n",
				n.Type(lang), s, e, depth, n.ChildCount(), string(src[s:min(int(e), len(src))]))
		}
	}
	findErr(root, 0)
}
