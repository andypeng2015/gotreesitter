package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	grm "github.com/odvcencio/gotreesitter/grammars"
)

// TestForestDispatchParity verifies the GOT_GLR_FOREST fast path is invisible:
// for the dispatched languages (bash) the forest tree must be byte-identical to
// the production tree, and for anything the forest declines (malformed input,
// non-dispatched languages) the result must match production exactly because we
// fall back to it. The forest is ~40-100x faster on bash's deep-equivalence
// blowup, but correctness is the only thing this test asserts.
func TestForestDispatchParity(t *testing.T) {
	bash := grm.BashLanguage()

	var big strings.Builder
	big.WriteString("#!/usr/bin/env bash\n")
	for i := 0; i < 40; i++ {
		big.WriteString("f() { local v=1; echo \"$v\"; if [ -f x ]; then echo y; fi; }\n")
	}

	clean := []string{
		"#!/usr/bin/env bash\nset -euo pipefail\nf() { local v=1; echo \"$v\"; }\n",
		"for i in 1 2 3; do echo $i; done\nwhile read l; do echo \"$l\"; done < f\n",
		"if [[ $a == b* ]]; then echo y; elif [ -f x ]; then echo z; else echo n; fi\n",
		"case $x in a) echo A;; b|c) echo BC;; *) echo D;; esac\n",
		"arr=(a b c); echo \"${arr[@]}\"; declare -A m; m[k]=v\n",
		"r=$(ls | grep foo | wc -l); x=${y:-def}; z=${a//b/c}\n",
		"cat <<EOF\nbody $v\nEOF\n",
		big.String(),
	}
	// Malformed inputs the forest cannot complete — must fall back to production.
	malformed := []string{
		"if [ -f x ]; then\n  echo y\n",
		"f() { echo a\n",
		"case $x in a) echo\n",
	}

	check := func(label string, lang *gts.Language, src string) {
		gts.SetGLRForestEnabled(false)
		prod, _ := gts.NewParser(lang).Parse([]byte(src))
		want := prod.RootNode().SExpr(lang)
		gts.SetGLRForestEnabled(true)
		got, _ := gts.NewParser(lang).Parse([]byte(src))
		gts.SetGLRForestEnabled(false)
		if got.RootNode().SExpr(lang) != want {
			t.Errorf("%s: forest dispatch diverged from production for %q", label, src)
		}
	}

	for _, s := range clean {
		check("bash-clean", bash, s)
	}
	for _, s := range malformed {
		check("bash-malformed-fallback", bash, s)
	}
	// Non-dispatched languages must be untouched even with the flag on.
	check("go-untouched", grm.GoLanguage(), "package p\nfunc f() { return }\n")
	check("python-untouched", grm.PythonLanguage(), "def f():\n    return 1\n")
}
