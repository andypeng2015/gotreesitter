//go:build cgo && treesitter_c_parity

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestFormatHighlightCaptureMismatchIncludesCountsAndFirstCaptures(t *testing.T) {
	onlyGo := []highlightCapture{
		{Name: "keyword", StartByte: 10, EndByte: 15},
		{Name: "type", StartByte: 20, EndByte: 24},
	}
	onlyC := []highlightCapture{
		{Name: "function", StartByte: 30, EndByte: 38},
	}

	got := formatHighlightCaptureMismatch(onlyGo, onlyC)
	want := "capture mismatch go_only=2 c_only=1 first_go_only=@keyword[10:15] first_c_only=@function[30:38]"
	if got != want {
		t.Fatalf("mismatch detail = %q, want %q", got, want)
	}
}

func TestDiffHighlightCaptures(t *testing.T) {
	goCaps := []highlightCapture{
		{Name: "keyword", StartByte: 0, EndByte: 5},
		{Name: "type", StartByte: 6, EndByte: 9},
	}
	cCaps := []highlightCapture{
		{Name: "keyword", StartByte: 0, EndByte: 5},
		{Name: "function", StartByte: 6, EndByte: 9},
	}

	onlyGo, onlyC := diffHighlightCaptures(goCaps, cCaps)
	if len(onlyGo) != 1 || onlyGo[0] != goCaps[1] {
		t.Fatalf("onlyGo = %#v, want %#v", onlyGo, []highlightCapture{goCaps[1]})
	}
	if len(onlyC) != 1 || onlyC[0] != cCaps[1] {
		t.Fatalf("onlyC = %#v, want %#v", onlyC, []highlightCapture{cCaps[1]})
	}
}

func TestCollectSamplesUsesRequestedLanguageOrder(t *testing.T) {
	root := t.TempDir()
	rubyPath := filepath.Join(root, "a.rb")
	tsPath := filepath.Join(root, "a.ts")
	if err := os.WriteFile(rubyPath, []byte("puts 'x'\n"), 0o644); err != nil {
		t.Fatalf("write ruby sample: %v", err)
	}
	if err := os.WriteFile(tsPath, []byte("const x = 1;\n"), 0o644); err != nil {
		t.Fatalf("write typescript sample: %v", err)
	}

	manifest := corpusManifest{Sets: []corpusSetSpec{
		{Name: "ruby", Language: "ruby", Files: []string{rubyPath}},
		{Name: "typescript", Language: "typescript", Files: []string{tsPath}},
	}}
	selected := map[string]struct{}{"ruby": {}, "typescript": {}}

	samples, err := collectSamples(root, manifest, selected, []string{"typescript", "ruby"})
	if err != nil {
		t.Fatalf("collectSamples: %v", err)
	}
	if len(samples) != 2 {
		t.Fatalf("samples length = %d, want 2", len(samples))
	}
	if got, want := samples[0].Language, "typescript"; got != want {
		t.Fatalf("first language = %q, want %q", got, want)
	}
}

func TestRenderSummaryUsesRequestedLanguageOrder(t *testing.T) {
	rows := []reportRow{
		{Language: "ruby", Sample: "ruby.rb", Mode: "cgo_full", MedianNS: 10},
		{Language: "ruby", Sample: "ruby.rb", Mode: "go_full", MedianNS: 10},
		{Language: "typescript", Sample: "a.ts", Mode: "cgo_full", MedianNS: 10},
		{Language: "typescript", Sample: "a.ts", Mode: "go_full", MedianNS: 10},
		{Language: "php", Sample: "a.php", Mode: "cgo_full", MedianNS: 10},
		{Language: "php", Sample: "a.php", Mode: "go_full", MedianNS: 10},
	}

	summary := renderSummary(rows, []string{"typescript", "php", "ruby"})
	ts := strings.Index(summary, "| typescript |")
	php := strings.Index(summary, "| php |")
	ruby := strings.Index(summary, "| ruby |")
	if ts < 0 || php < 0 || ruby < 0 {
		t.Fatalf("summary missing rows:\n%s", summary)
	}
	if !(ts < php && php < ruby) {
		t.Fatalf("summary order mismatch:\n%s", summary)
	}
}

func TestRunGoEditReportsIncrementalAttribution(t *testing.T) {
	lang := grammars.RustLanguage()
	r := &runner{
		name:     "rust",
		goLang:   lang,
		goParser: gotreesitter.NewParser(lang),
		support: grammars.ParseSupport{
			Name:    "rust",
			Backend: grammars.ParseBackendDFA,
		},
	}

	source := []byte(strings.Repeat("// comment abc\nfn main() {}\n", 32))
	stats, err := runGoEdit(r, source, false)
	if err != nil {
		t.Fatalf("runGoEdit: %v", err)
	}
	if stats.SetupParseNS <= 0 {
		t.Fatalf("SetupParseNS = %d, want > 0", stats.SetupParseNS)
	}
	if stats.TreeEditNS <= 0 {
		t.Fatalf("TreeEditNS = %d, want > 0", stats.TreeEditNS)
	}
	if stats.ParseWallNS != stats.IncrementalReuseNS+stats.IncrementalReparseNS {
		t.Fatalf("ParseWallNS = %d, want reuse + reparse = %d", stats.ParseWallNS, stats.IncrementalReuseNS+stats.IncrementalReparseNS)
	}
}
