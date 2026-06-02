package gotreesitter_test

// Parse-level A/B for the generated lexer. Run the SAME benchmark twice — the
// only difference is which lexer Lexer.Next dispatches to (the codegen path is
// wired by language name under -tags lexgen):
//
//	GTS_LEXGEN_BENCH=path.ts go test . -run '^$' -bench BenchmarkParseLexgen
//	GTS_LEXGEN_BENCH=path.ts go test . -run '^$' -bench BenchmarkParseLexgen -tags lexgen
//
// Point GTS_LEXGEN_BENCH at a real source file (e.g. a corpus .ts) — repeated
// synthetic source is pathological for some grammars' GLR and not representative.
//
// Measured on cgo_harness/corpus_real/typescript/large__parser.ts (376 KB):
// table ~59 ms vs generated ~55 ms => ~7% faster parse. The ~1.9x lexer win
// (BenchmarkLexJson) becomes a modest parse win because lexing is ~20% of parse
// wall. python is ~neutral: its INDENT/DEDENT external scanner dominates lexing
// and the DFA codegen does not replace it.

import (
	"os"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func benchParseFile(b *testing.B, langName string) {
	path := os.Getenv("GTS_LEXGEN_BENCH")
	if path == "" {
		b.Skip("set GTS_LEXGEN_BENCH=<source file> to run the parse-level A/B")
	}
	src, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("read %s: %v", path, err)
	}
	entry := grammars.DetectLanguageByName(langName)
	if entry == nil || entry.Language == nil {
		b.Skipf("language missing: %s", langName)
	}
	lang := entry.Language()
	b.SetBytes(int64(len(src)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := gts.NewParser(lang)
		tree, err := p.Parse(src)
		if err != nil {
			b.Fatalf("parse: %v", err)
		}
		tree.Release()
	}
}

func BenchmarkParseLexgenTypescript(b *testing.B) { benchParseFile(b, "typescript") }
func BenchmarkParseLexgenPython(b *testing.B)     { benchParseFile(b, "python") }
