//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"sort"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

const (
	// Ratchets: these should only move in the "stricter" direction over time.
	minCuratedStructuralLanguages = 206
	minCuratedHighlightLanguages  = 200
	maxKnownDegradedStructural    = 0
	maxKnownDegradedNoErrorClean  = 1
	maxKnownDegradedHighlight     = 49
	maxParitySkips                = 0
)

// TestParityGateCoverageRatchet prevents silent narrowing of correctness gates.
// Update these thresholds only when intentionally tightening/loosening policy.
func TestParityGateCoverageRatchet(t *testing.T) {
	if got := len(curatedStructuralLanguages); got < minCuratedStructuralLanguages {
		t.Fatalf("curatedStructuralLanguages shrank: got=%d min=%d", got, minCuratedStructuralLanguages)
	}
	if got := len(curatedHighlightLanguages); got < minCuratedHighlightLanguages {
		t.Fatalf("curatedHighlightLanguages shrank: got=%d min=%d", got, minCuratedHighlightLanguages)
	}
	if got := len(knownDegradedStructural); got > maxKnownDegradedStructural {
		t.Fatalf("knownDegradedStructural grew: got=%d max=%d", got, maxKnownDegradedStructural)
	}
	if got := len(knownDegradedNoErrorClean); got > maxKnownDegradedNoErrorClean {
		t.Fatalf("knownDegradedNoErrorClean grew: got=%d max=%d", got, maxKnownDegradedNoErrorClean)
	}
	for name := range knownDegradedNoErrorClean {
		if _, ok := knownDegradedStructural[name]; !ok {
			t.Fatalf("knownDegradedNoErrorClean[%q] is not in knownDegradedStructural", name)
		}
	}
	if got := len(knownDegradedHighlight); got > maxKnownDegradedHighlight {
		t.Fatalf("knownDegradedHighlight grew: got=%d max=%d", got, maxKnownDegradedHighlight)
	}
	if got := len(paritySkips); got > maxParitySkips {
		t.Fatalf("paritySkips grew: got=%d max=%d", got, maxParitySkips)
	}
}

// parityCaseByName looks up a parityCases entry by language name. Returns
// ok=false if the language has no registry/smoke-sample entry at all, which
// itself indicates a dead knownDegradedStructural entry.
func parityCaseByName(name string) (parityCase, bool) {
	for _, tc := range parityCases {
		if tc.name == name {
			return tc, true
		}
	}
	return parityCase{}, false
}

// TestParityKnownDegradedStructuralStillDiverges is the "stale skip" ratchet.
//
// TestParityGateCoverageRatchet above only stops knownDegradedStructural from
// GROWING past its ceiling; it never confirms the entries it already
// contains still describe a real divergence. That one-directional check is
// exactly how the list went stale in the first place: agda, apex, doxygen,
// hare, jsdoc, and rst all sat skipped long after their fresh-parse output
// matched (or came to match) the C reference exactly, because nothing ever
// re-verified them.
//
// For every remaining knownDegradedStructural entry, this test bypasses the
// skip and re-parses the language's smoke sample with both gotreesitter and
// the pinned C reference parser (the same comparison TestParityFreshParse
// performs), then requires at least one real node divergence. A language
// that now parses byte-for-byte identically to the C reference must be
// removed from knownDegradedStructural — this test fails loudly instead of
// letting the skip rot silently.
//
// Requires the C reference toolchain/cache, so — like the rest of the
// structural gate — it only runs in the exhaustive parity lane:
//
//	GOWORK=off GTS_PARITY_ALLOW_HOST=1 GTS_PARITY_MODE=exhaustive \
//	GTS_PARITY_C_REF_BUILD_CACHE=<cache dir> \
//	go test . -tags treesitter_c_parity -run TestParityKnownDegradedStructuralStillDiverges -v
func TestParityKnownDegradedStructuralStillDiverges(t *testing.T) {
	parityRequireExhaustive(t, "TestParityKnownDegradedStructuralStillDiverges")

	names := make([]string, 0, len(knownDegradedStructural))
	for name := range knownDegradedStructural {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			if parityLanguageExcluded(name) {
				t.Skipf("[%s] excluded via GTS_PARITY_SKIP_LANGS", name)
			}
			parityMaybeParallel(t)
			scheduleParityMemoryScavenge(t)

			tc, ok := parityCaseByName(name)
			if !ok {
				t.Fatalf("knownDegradedStructural[%q] has no matching language registry entry (dead skip-list entry, remove it)", name)
			}

			cLang, err := ParityCLanguage(tc.name)
			if err != nil {
				if skipReason := parityReferenceSkipReason(err); skipReason != "" {
					t.Skipf("[%s] skip C reference parser: %s", tc.name, skipReason)
				}
				t.Fatalf("[%s] load C parser from languages.lock: %v", tc.name, err)
			}

			src := normalizedSource(tc.name, tc.source)
			goTree, goLang, err := parseWithGo(tc, src, nil)
			if err != nil {
				t.Fatalf("[%s] gotreesitter parse error: %v", tc.name, err)
			}
			defer releaseGoTree(goTree)

			cParser := sitter.NewParser()
			defer cParser.Close()
			if err := cParser.SetLanguage(cLang); err != nil {
				if skipReason := parityReferenceSkipReason(err); skipReason != "" {
					t.Skipf("[%s] skip C reference parser SetLanguage: %s", tc.name, skipReason)
				}
				t.Fatalf("[%s] C parser SetLanguage error: %v", tc.name, err)
			}
			cTree := cParser.Parse(src, nil)
			if cTree == nil || cTree.RootNode() == nil {
				t.Fatalf("[%s] C reference parser returned nil tree", tc.name)
			}
			defer cTree.Close()

			var errs []string
			compareNodes(goTree.RootNode(), goLang, cTree.RootNode(), "root", &errs)
			if len(errs) == 0 {
				t.Fatalf("stale skip: %q is listed in knownDegradedStructural but its fresh parse now matches the C reference exactly (0 node divergences) — remove %q from knownDegradedStructural", name, name)
			}
			t.Logf("[%s] confirmed live divergence: %d node mismatch(es)", name, len(errs))
		})
	}
}
