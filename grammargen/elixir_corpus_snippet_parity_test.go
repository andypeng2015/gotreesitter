package grammargen

import (
	"os"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestElixirImportedCorpusSnippetParity(t *testing.T) {
	genLang, refLang := loadImportedParityLanguages(t, "elixir")
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, elixirOperatorLeftAssociativeCorpusBlock)
}

func TestElixirImportedAddSubOperatorParity(t *testing.T) {
	genLang, refLang := loadImportedParityLanguages(t, "elixir")
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, "a + b + c\na - b - c\n")
}

func TestElixirImportedGuardedDefDoBlockParity(t *testing.T) {
	genLang, refLang := loadImportedParityLanguages(t, "elixir")
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, "defmodule M do\n  def func(x) when is_integer(x) do\n    priv(x) + priv(x)\n  end\nend\n")
}

func TestElixirImportedForReduceDoBlockParity(t *testing.T) {
	genLang, refLang := loadImportedParityLanguages(t, "elixir")
	assertGeneratedAndReferenceNoError(t, genLang, refLang, "for x <- [1, 2, 1], reduce: %{} do\n  acc -> Map.update(acc, x, 1, & &1 + 1)\nend\n")
}

func TestElixirImportedDoEndStabClauseBodyParity(t *testing.T) {
	genLang, refLang := loadImportedParityLanguages(t, "elixir")
	assertGeneratedAndReferenceNoError(t, genLang, refLang, elixirDoEndStabClauseBodyCorpusBlock)
}

func TestElixirImportedQuotedInterpolationNoError(t *testing.T) {
	genLang, refLang := loadImportedParityLanguages(t, "elixir")
	assertGeneratedAndReferenceNoError(t, genLang, refLang, ":\"with #{1 + 1} interpol\"\n\"with #{1 + 1} interpol\"\n")
}

func TestElixirImportedLRSplitCorpusSnippetParity(t *testing.T) {
	genLang, refLang := loadImportedElixirLRSplitParityLanguages(t)
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, elixirOperatorLeftAssociativeCorpusBlock)
}

const elixirOperatorLeftAssociativeCorpusBlock = "a ** b ** c\n\n" +
	"a * b * c\n" +
	"a / b / c\n\n" +
	"a + b + c\n" +
	"a - b - c\n\n" +
	"a ^^^ b ^^^ c\n\n" +
	"a in b in c\n" +
	"a not in b not in c\n\n" +
	"a |> b |> c\n" +
	"a <<< b <<< c\n" +
	"a >>> b >>> c\n" +
	"a <<~ b <<~ c\n" +
	"a ~>> b ~>> c\n" +
	"a <~ b <~ c\n" +
	"a ~> b ~> c\n" +
	"a <~> b <~> c\n" +
	"a <|> b <|> c\n\n" +
	"a < b < c\n" +
	"a > b > c\n" +
	"a <= b <= c\n" +
	"a >= b >= c\n\n" +
	"a == b == c\n" +
	"a != b != c\n" +
	"a =~ b =~ c\n" +
	"a === b === c\n" +
	"a !== b !== c\n\n" +
	"a && b && c\n" +
	"a &&& b &&& c\n" +
	"a and b and c\n\n" +
	"a || b || c\n" +
	"a ||| b ||| c\n" +
	"a or b or c\n\n" +
	"a <- b <- c\n" +
	"a \\\\ b \\\\ c\n"

const elixirDoEndStabClauseBodyCorpusBlock = "fun do\n" +
	"  1 ->\n" +
	"    1\n" +
	"    x\n\n" +
	"  1 ->\n" +
	"    1\n" +
	"end\n\n" +
	"fun do\n" +
	"  1 ->\n" +
	"    1\n" +
	"    Mod.fun\n\n" +
	"  1 ->\n" +
	"    1\n" +
	"end\n\n" +
	"fun do\n" +
	"  1 ->\n" +
	"    1\n" +
	"    mod.fun\n\n" +
	"  1 ->\n" +
	"    1\n" +
	"end\n\n" +
	"fun do\n" +
	"  1 ->\n" +
	"    1\n\n" +
	"  x 1 ->\n" +
	"    1\n" +
	"end\n"

func assertGeneratedAndReferenceNoError(t *testing.T, genLang, refLang *gotreesitter.Language, src string) {
	t.Helper()

	data := []byte(src)
	genTree, err := gotreesitter.NewParser(genLang).Parse(data)
	if err != nil {
		t.Fatalf("generated parse: %v", err)
	}
	refTree, err := gotreesitter.NewParser(refLang).Parse(data)
	if err != nil {
		t.Fatalf("reference parse: %v", err)
	}

	genRoot := genTree.RootNode()
	refRoot := refTree.RootNode()
	if refRoot.HasError() {
		t.Fatalf("reference tree has ERROR: %s", safeSExpr(refRoot, refLang, 256))
	}
	if genRoot.HasError() {
		t.Fatalf("generated tree has ERROR while reference is clean\nGEN: %s\nREF: %s", safeSExpr(genRoot, genLang, 256), safeSExpr(refRoot, refLang, 256))
	}
}

func loadImportedElixirLRSplitParityLanguages(t *testing.T) (*gotreesitter.Language, *gotreesitter.Language) {
	t.Helper()

	var grammarSpec importParityGrammar
	for _, g := range importParityGrammars {
		if g.name == "elixir" {
			grammarSpec = g
			break
		}
	}
	if grammarSpec.name == "" {
		t.Fatal("elixir import parity grammar not found")
	}
	if grammarSpec.jsonPath != "" {
		grammarSpec.jsonPath = fallbackParitySeedPath(grammarSpec.jsonPath)
		if _, err := os.Stat(grammarSpec.jsonPath); err != nil {
			t.Skipf("elixir grammar.json not available: %v", err)
		}
	} else if grammarSpec.path != "" {
		grammarSpec.path = fallbackParitySeedPath(grammarSpec.path)
		if _, err := os.Stat(grammarSpec.path); err != nil {
			t.Skipf("elixir grammar.js not available: %v", err)
		}
	}

	gram, err := importParityGrammarSource(grammarSpec)
	if err != nil {
		t.Fatalf("import elixir grammar: %v", err)
	}
	gram.EnableLRSplitting = true

	timeout := grammarSpec.genTimeout
	if timeout == 0 {
		timeout = 180 * time.Second
	}
	genLang, err := generateWithTimeout(gram, timeout)
	if err != nil {
		t.Fatalf("generate elixir language: %v", err)
	}

	refLang := grammarSpec.blobFunc()
	adaptExternalScanner(refLang, genLang)
	return genLang, refLang
}
