package grammargen

import (
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
