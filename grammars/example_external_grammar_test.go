package grammars_test

import (
	"fmt"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// ExampleDecodeLanguageBlob_externalGrammar demonstrates the full pattern for
// adding a grammar to gotreesitter from a consumer package, without modifying
// gotreesitter itself. A language-server or application package ships its own
// precompiled grammar blob via go:embed and registers it at init() time.
//
// This is the recommended path for niche grammars (language servers, DSLs,
// in-house tooling) that don't belong in gotreesitter's built-in registry.
//
// A real consumer would replace the BlobByName call below with an embedded blob:
//
//	import _ "embed"
//
//	//go:embed qmljs.bin
//	var qmljsBlob []byte
//
//	func init() {
//	    // 1. Register the hand-ported Go external scanner first so it is
//	    //    attached when the Language is decoded on first access.
//	    grammars.RegisterExternalScanner("qmljs", newQMLScanner())
//
//	    // 2. Register the language with a lazy loader that decodes the
//	    //    embedded blob on demand. Consumers own the blob bytes; gotreesitter
//	    //    just decodes them.
//	    grammars.Register(grammars.LangEntry{
//	        Name:           "qmljs",
//	        Extensions:     []string{".qml"},
//	        Language:       loadQMLJS,
//	        GrammarSource:  grammars.GrammarSourceTS2GoBlob,
//	        HighlightQuery: qmljsHighlights,
//	    })
//	}
//
//	var loadQMLJS = sync.OnceValue(func() *gotreesitter.Language {
//	    lang, err := grammars.DecodeLanguageBlob(qmljsBlob)
//	    if err != nil {
//	        panic("qmljs grammar blob is corrupt: " + err.Error())
//	    }
//	    return lang
//	})
//
// After this init() runs, the grammar is indistinguishable from a built-in:
// grammars.DetectLanguage("foo.qml") returns the entry, AllLanguages() enumerates
// it, and syntax highlighting / code fences resolve normally.
func ExampleDecodeLanguageBlob_externalGrammar() {
	// Pretend this came from a consumer-side go:embed directive.
	blob := grammars.BlobByName("go")

	lang, err := grammars.DecodeLanguageBlob(blob)
	if err != nil {
		fmt.Println("decode failed:", err)
		return
	}

	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse([]byte("package main\n"))
	if err != nil {
		fmt.Println("parse failed:", err)
		return
	}
	fmt.Println("parsed:", !tree.RootNode().HasError())
	// Output: parsed: true
}
