package grammars

// Temporary wave-2 profiling harness for the open-error-region quadratic.
// Gated behind WAVE2_PROFILE so it never runs in normal suites.

import (
	"os"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/odvcencio/gotreesitter"
)

func TestZZWave2ProfileCSharp(t *testing.T) {
	target := os.Getenv("WAVE2_PROFILE")
	if target == "" {
		t.Skip("set WAVE2_PROFILE=<file.cs> to run")
	}
	src, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lang := CSharpLanguage()
	p := gotreesitter.NewParser(lang)

	if prof := os.Getenv("WAVE2_CPUPROFILE"); prof != "" {
		f, err := os.Create(prof)
		if err != nil {
			t.Fatalf("cpuprofile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			t.Fatalf("start profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	start := time.Now()
	tree, err := p.Parse(src)
	dur := time.Since(start)
	if err != nil {
		t.Fatalf("parse error after %v: %v", dur, err)
	}
	root := tree.RootNode()
	t.Logf("parsed %d bytes in %v; root=%s hasError=%v span=[%d,%d)",
		len(src), dur, root.Type(lang), root.HasError(), root.StartByte(), root.EndByte())
}
