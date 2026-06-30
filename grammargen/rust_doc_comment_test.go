package grammargen

import (
	"os"
	"testing"
	"time"

	"github.com/odvcencio/gotreesitter/grammars"
)

func TestRustDocCommentContentParity(t *testing.T) {
	jsonPath := rustGrammarJSONPathForTest(t)
	source, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Skipf("Rust grammar.json not available: %v", err)
	}
	gram, err := ImportGrammarJSON(source)
	if err != nil {
		t.Fatalf("import Rust grammar.json: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate Rust language: %v", err)
	}
	refLang := grammars.RustLanguage()
	adaptExternalScanner(refLang, genLang)

	cases := []struct {
		name string
		src  string
	}{
		{name: "plain", src: "/// A lifetime definition\npub struct X {}\n"},
		{name: "lifetime", src: "/// A lifetime definition, e.g. 'a: 'b+'c+'d\npub struct X {}\n"},
		{name: "backtick_lifetime", src: "/// A lifetime definition, e.g. `'a: 'b+'c+'d`\npub struct X {}\n"},
		{name: "inner", src: "//! Module docs with 'a: 'b+'c+'d\nmod x {}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertGeneratedAndReferenceDeepParity(t, genLang, refLang, tc.src)
		})
	}
}
