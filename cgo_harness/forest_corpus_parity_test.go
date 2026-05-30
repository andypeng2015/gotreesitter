package cgoharness

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	gts "github.com/odvcencio/gotreesitter"
	grm "github.com/odvcencio/gotreesitter/grammars"
)

// TestForestCorpusParity is the correctness gate for promoting a language onto
// the GSS-forest fast path (Parser.tryForestFastPath / languageWantsForest).
//
// The runtime dispatch falls back to the production parser whenever the forest
// declines (failure / error node / truncation), so it can never regress those
// cases. The ONE risk it cannot catch is a forest that produces a clean,
// complete, but STRUCTURALLY DIFFERENT tree — a silent divergence. This test
// closes that gap: for every real-corpus file it parses with the production
// parser and with the forest, and on every file the forest would dispatch
// (clean + complete) it asserts byte-identical s-expressions. Any divergence
// fails the test and blocks default-on for that language.
//
// Languages come from GTS_FOREST_LANGS (comma-separated; default "bash") so a
// candidate (swift, fortran, ...) can be vetted before it is added to the
// runtime allowlist. It also reports dispatch rate and wall speedup, which is
// the "wall" half of "full parity wall and correctness".
//
// Run heavy (real corpus) under Docker per the repo's testing discipline:
//
//	cgo_harness/docker/run_forest_corpus_parity.sh
func TestForestCorpusParity(t *testing.T) {
	// Opt-in: this is a heavy real-corpus gate that currently FAILS by design
	// while the forest is pre-parity (it reports the divergences blocking
	// default-on). Keep it out of the default `go test ./...` run; the Docker
	// runner and manual promotion checks set GTS_FOREST_CORPUS=1.
	if strings.TrimSpace(os.Getenv("GTS_FOREST_CORPUS")) == "" {
		t.Skip("set GTS_FOREST_CORPUS=1 to run the forest real-corpus parity gate")
	}
	langs := strings.Split(envOr("GTS_FOREST_LANGS", "bash"), ",")
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
		lang := load()
		runForestLangParity(t, name, lang, files)
	}
	if !anyRun {
		t.Skip("no forest corpus available for requested languages")
	}
}

func runForestLangParity(t *testing.T, name string, lang *gts.Language, files []string) {
	t.Helper()
	var (
		total, dispatched, fellBack, diverged int
		prodNanos, forestNanos                int64
		divergedFiles                         []string
	)
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Errorf("%s: read %s: %v", name, f, err)
			continue
		}
		total++

		// Production baseline (forest off).
		gts.SetGLRForestEnabled(false)
		st := time.Now()
		prodTree, _ := gts.NewParser(lang).Parse(src)
		prodNanos += time.Since(st).Nanoseconds()
		if prodTree == nil || prodTree.RootNode() == nil {
			t.Errorf("%s: production produced no tree for %s", name, filepath.Base(f))
			continue
		}
		want := prodTree.RootNode().SExpr(lang)

		// Forest result + the dispatch acceptance criteria (mirrors
		// tryForestFastPath: clean, no error node, reaches the last
		// non-whitespace byte).
		st = time.Now()
		root, ok := gts.NewParser(lang).ParseForestExperimental(src)
		forestNanos += time.Since(st).Nanoseconds()
		if !ok || root == nil || root.HasError() || root.EndByte() < lastNonWSByte(src) {
			fellBack++
			continue
		}
		dispatched++
		if got := root.SExpr(lang); got != want {
			diverged++
			divergedFiles = append(divergedFiles, filepath.Base(f))
		}
	}

	speedup := 0.0
	if forestNanos > 0 {
		speedup = float64(prodNanos) / float64(forestNanos)
	}
	t.Logf("%-8s files=%d dispatched=%d fellback=%d diverged=%d | prod=%.1fms forest=%.1fms speedup=%.1fx",
		name, total, dispatched, fellBack, diverged,
		float64(prodNanos)/1e6, float64(forestNanos)/1e6, speedup)
	if diverged > 0 {
		sort.Strings(divergedFiles)
		t.Errorf("%s: %d/%d dispatched files DIVERGED from production (blocks forest default-on): %s",
			name, diverged, dispatched, strings.Join(divergedFiles, ", "))
	}
}

func forestLanguageLoaders() map[string]func() *gts.Language {
	out := map[string]func() *gts.Language{}
	for _, e := range grm.AllLanguages() {
		out[e.Name] = e.Language
	}
	return out
}

func forestRepoRoot(t *testing.T) string {
	t.Helper()
	if v := strings.TrimSpace(os.Getenv("GTS_REPO_ROOT")); v != "" {
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// Tests run from cgo_harness/; corpus_real lives there or at repo root.
	for _, cand := range []string{wd, filepath.Dir(wd)} {
		if _, err := os.Stat(filepath.Join(cand, "cgo_harness", "corpus_real")); err == nil {
			return cand
		}
		if _, err := os.Stat(filepath.Join(cand, "corpus_real")); err == nil {
			return cand
		}
	}
	return wd
}

func forestCorpusDir(repoRoot, lang string) string {
	for _, cand := range []string{
		filepath.Join(repoRoot, "cgo_harness", "corpus_real", lang),
		filepath.Join(repoRoot, "corpus_real", lang),
	} {
		if info, err := os.Stat(cand); err == nil && info.IsDir() {
			return cand
		}
	}
	return ""
}

func forestCorpusFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus dir %s: %v", dir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files
}

func lastNonWSByte(src []byte) uint32 {
	end := len(src)
	for end > 0 {
		switch src[end-1] {
		case ' ', '\t', '\r', '\n':
			end--
			continue
		}
		break
	}
	return uint32(end)
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

var _ = fmt.Sprintf
