//go:build cgo && treesitter_c_parity

package cgoharness

// Forest-vs-C oracle gate. Unlike TestForestCorpusParity (forest vs the
// production parser), this compares the GSS-forest fast path DIRECTLY against
// tree-sitter-c, skipping the production leg entirely. That matters for
// languages whose production parse is too slow to use as the parity baseline —
// notably haskell, whose O(n^2) deep-merge blowup makes the production-vs-forest
// gate time out. Here the forest is the only gotreesitter parse we run, so a
// pathologically slow production path can't block vetting the forest.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sitter "github.com/tree-sitter/go-tree-sitter"

	gts "github.com/odvcencio/gotreesitter"
)

// TestForestVsCOracleParity asserts that, for every real-corpus file the forest
// would DISPATCH (clean, no error node, reaches the last non-whitespace byte),
// the forest tree is byte-identical to the tree-sitter-c oracle (compareNodes:
// Type / StartByte / EndByte / IsNamed / IsMissing / ChildCount, recursively).
// Files the forest declines fall back to production at runtime and are NOT
// required to match C here. It also reports dispatch rate and the C-vs-forest
// wall speedup, which is the "is it worth promoting" half of the decision.
//
// The forest tree comes from ParseForestExperimental, which finalizes the root
// through the same finalizeResultRoot -> per-language compatibility pass the
// runtime forest path uses, so it is a faithful stand-in for what promotion
// would return.
//
// Heavy (real corpus + CGo) -> opt-in:
//
//	GTS_FOREST_ORACLE=1 GTS_FOREST_ORACLE_LANGS=haskell \
//	  go test ./cgo_harness -tags treesitter_c_parity -run TestForestVsCOracleParity -v
func TestForestVsCOracleParity(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GTS_FOREST_ORACLE")) == "" {
		t.Skip("set GTS_FOREST_ORACLE=1 to run the forest-vs-C oracle gate")
	}
	langs := strings.Split(envOr("GTS_FOREST_ORACLE_LANGS", "haskell"), ",")
	loaders := forestLanguageLoaders()
	repoRoot := forestRepoRoot(t)

	anyRun := false
	for _, raw := range langs {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		load, ok := loaders[name]
		if !ok {
			t.Errorf("%s: unknown language (not in grammars.AllLanguages)", name)
			continue
		}
		lang := load()

		cLang, err := ParityCLanguage(name)
		if err != nil {
			if reason := parityReferenceSkipReason(err); reason != "" {
				t.Logf("%s: skip — no C reference parser: %s", name, reason)
				continue
			}
			t.Errorf("%s: ParityCLanguage: %v", name, err)
			continue
		}

		dir := forestCorpusDir(repoRoot, name)
		if dir == "" {
			t.Logf("%s: no corpus_real/%s directory — skipping", name, name)
			continue
		}
		files := forestCorpusFiles(t, dir)
		if len(files) == 0 {
			t.Logf("%s: corpus_real/%s empty — skipping", name, name)
			continue
		}
		anyRun = true

		var (
			total, dispatched, fellBack, diverged int
			cNanos, forestNanos                   int64
			divergedFiles                         []string
			fallbackReasons                       = map[string]int{}
		)

		for _, f := range files {
			func(f string) {
				src, err := os.ReadFile(f)
				if err != nil {
					t.Errorf("%s: read %s: %v", name, f, err)
					return
				}
				total++

				// Forest result — the ONLY gotreesitter parse here (no
				// production baseline, so an O(n^2) production path can't
				// block the gate). Run it under a per-file budget: the forest
				// REDUCE DFS (glr_forest.go forestReducer.dfs) is uncapped and
				// can itself blow up on high-ambiguity grammars (haskell), so a
				// candidate whose forest can't finish in time is, for promotion
				// purposes, equivalent to a decline.
				type forestParseResult struct {
					tree *gts.Tree
					ok   bool
				}
				ch := make(chan forestParseResult, 1)
				parser := gts.NewParser(lang)
				st := time.Now()
				go func() {
					tr, okp := parser.ParseForestExperimental(src)
					ch <- forestParseResult{tr, okp}
				}()
				var forestTree *gts.Tree
				var ok bool
				select {
				case r := <-ch:
					forestTree, ok = r.tree, r.ok
				case <-time.After(forestOracleBudget()):
					forestNanos += time.Since(st).Nanoseconds()
					fellBack++
					fallbackReasons["timeout"]++
					// Leak the still-running parse goroutine; it dies at process
					// exit. Don't touch its (eventual) tree — racing it is unsafe.
					return
				}
				forestNanos += time.Since(st).Nanoseconds()
				if forestTree != nil {
					defer forestTree.Release()
				}
				var root *gts.Node
				if forestTree != nil {
					root = forestTree.RootNode()
				}
				// Dispatch acceptance criteria — mirrors tryForestFastPath.
				if !ok || root == nil || root.HasError() || root.EndByte() < lastNonWSByte(src) {
					fellBack++
					fallbackReasons[forestFallbackReason(ok, root, src)]++
					return
				}
				dispatched++

				// C oracle (only for dispatched files).
				cParser := sitter.NewParser()
				defer cParser.Close()
				if err := cParser.SetLanguage(cLang); err != nil {
					t.Errorf("%s: C SetLanguage: %v", name, err)
					return
				}
				ct := time.Now()
				cTree := cParser.Parse(src, nil)
				cNanos += time.Since(ct).Nanoseconds()
				if cTree == nil || cTree.RootNode() == nil {
					t.Errorf("%s: C produced no tree for %s", name, filepath.Base(f))
					return
				}
				defer cTree.Close()

				var errs []string
				compareNodes(root, lang, cTree.RootNode(), "root", &errs)
				if len(errs) > 0 {
					diverged++
					divergedFiles = append(divergedFiles, filepath.Base(f))
					shown := errs
					if len(shown) > 6 {
						shown = shown[:6]
					}
					t.Logf("%s: %s forest!=C: %s", name, filepath.Base(f), strings.Join(shown, " | "))
				}
			}(f)
		}

		speedup := 0.0
		if forestNanos > 0 {
			speedup = float64(cNanos) / float64(forestNanos)
		}
		t.Logf("%-8s files=%d dispatched=%d fellback=%d diverged=%d | c=%.1fms forest=%.1fms forest_vs_c=%.2fx",
			name, total, dispatched, fellBack, diverged,
			float64(cNanos)/1e6, float64(forestNanos)/1e6, speedup)
		if fellBack > 0 {
			t.Logf("%-8s fallback reasons: %s", name, formatFallbackReasons(fallbackReasons))
		}
		if diverged > 0 {
			divergedFiles = append([]string(nil), divergedFiles...)
			t.Errorf("%s: %d/%d dispatched files DIVERGED from the C oracle (blocks forest promotion): %s",
				name, diverged, dispatched, strings.Join(divergedFiles, ", "))
		}
	}

	if !anyRun {
		t.Skip("no forest-oracle corpus available for requested languages")
	}
}

// forestOracleBudget is the per-file wall budget for a single forest parse.
// Overridable via GTS_FOREST_ORACLE_BUDGET (a Go duration, e.g. "30s").
func forestOracleBudget() time.Duration {
	if v := strings.TrimSpace(os.Getenv("GTS_FOREST_ORACLE_BUDGET")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 10 * time.Second
}
