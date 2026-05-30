package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	grm "github.com/odvcencio/gotreesitter/grammars"
)

// TestForestDispatchParity verifies the (default-on) forest fast path is
// invisible: for a dispatched language (css ∈ languageWantsForest) the forest
// tree must be byte-identical to production — same s-expr AND same root byte
// span — and anything the forest declines (malformed input, non-dispatched
// languages) must match production because we fall back to it.
// SetGLRForestEnabled(false) yields the production baseline; (true) is the
// default-on dispatch.
func TestForestDispatchParity(t *testing.T) {
	css := grm.CssLanguage()

	var big strings.Builder
	for i := 0; i < 60; i++ {
		big.WriteString(".cls-" + cssN(i) + " { color: red; margin: 0 1px 2px 3px; padding: 1em; }\n")
		big.WriteString("#id-" + cssN(i) + " > a:hover, .x .y { background: url(/img.png) no-repeat; }\n")
	}
	clean := []string{
		"a { color: red; }\n",
		".cls { margin: 0; padding: 1px 2px; z-index: 5; }\n",
		"@media (max-width: 600px) { .x { display: none; } }\n",
		"div > p + span ~ a:not(.z)::before { content: \"x\"; }\n",
		":root { --c: #fff; } body { color: var(--c); transform: matrix(1,2,3,4,5,6); }\n",
		big.String(),
	}
	malformed := []string{
		"a { color: red;\n",
		".x { ; } @media\n",
	}

	check := func(label string, lang *gts.Language, src string) {
		gts.SetGLRForestEnabled(false)
		prod, _ := gts.NewParser(lang).Parse([]byte(src))
		want := prod.RootNode().SExpr(lang)
		wantEnd := prod.RootNode().EndByte()
		gts.SetGLRForestEnabled(true)
		got, _ := gts.NewParser(lang).Parse([]byte(src))
		if got.RootNode().SExpr(lang) != want {
			t.Errorf("%s: forest dispatch s-expr diverged for %q", label, src)
		}
		if got.RootNode().EndByte() != wantEnd {
			t.Errorf("%s: forest dispatch root endByte %d != production %d for %q",
				label, got.RootNode().EndByte(), wantEnd, src)
		}
	}

	for _, s := range clean {
		check("css-clean", css, s)
	}
	for _, s := range malformed {
		check("css-malformed-fallback", css, s)
	}
	// Non-dispatched languages must be untouched even with the switch on.
	check("go-untouched", grm.GoLanguage(), "package p\nfunc f() { return }\n")
	check("bash-untouched", grm.BashLanguage(), "f() { echo a; }\n")
	gts.SetGLRForestEnabled(true)
}

func cssN(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
