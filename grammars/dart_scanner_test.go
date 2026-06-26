//go:build !grammar_subset || grammar_subset_dart

package grammars

import (
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestDartExternalScannerBindsShiftedExternalSymbolsByName(t *testing.T) {
	lang := dartExternalBindingTestLanguage(
		"_extension_only",
		"_documentation_block_comment",
		"_template_chars_raw_slash",
		"_template_chars_double",
		"_block_comment",
	)

	scanner, ok := DartExternalScanner{}.ExternalScannerForLanguage(lang).(DartExternalScanner)
	if !ok {
		t.Fatalf("DartExternalScanner binding type = %T, want DartExternalScanner", DartExternalScanner{}.ExternalScannerForLanguage(lang))
	}
	if got, want := scanner.externalToToken[0], -1; got != want {
		t.Fatalf("extension-only external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[1], dartTokDocBlockComment; got != want {
		t.Fatalf("doc-comment external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[2], dartTokTemplateCharsRawSlash; got != want {
		t.Fatalf("raw-slash external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[3], dartTokTemplateCharsDouble; got != want {
		t.Fatalf("template-double external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.symbols[dartTokTemplateCharsDouble], gotreesitter.Symbol(4); got != want {
		t.Fatalf("template-double result symbol = %d, want %d", got, want)
	}
	if got, want := scanner.symbols[dartTokDocBlockComment], gotreesitter.Symbol(2); got != want {
		t.Fatalf("doc-comment result symbol = %d, want %d", got, want)
	}

	validExternal := []bool{false, false, false, true, false}
	var semanticValid [dartTokenCount]bool
	validSemantic := scanner.remapValidSymbols(validExternal, &semanticValid)
	if !validSemantic[dartTokTemplateCharsDouble] {
		t.Fatalf("shifted template-double external did not become valid semantic token: %v", validSemantic)
	}
	if got, want := dartTemplateResultSymbol(validSemantic, scanner.symbolTable()), gotreesitter.Symbol(4); got != want {
		t.Fatalf("template scanner result symbol = %d, want shifted symbol %d", got, want)
	}
}

func TestDartExternalScannerZeroValueKeepsLegacySymbols(t *testing.T) {
	scanner := DartExternalScanner{}
	symbols := scanner.symbolTable()
	if got, want := symbols[dartTokTemplateCharsDouble], dartSymTemplateCharsDouble; got != want {
		t.Fatalf("zero-value template-double symbol = %d, want %d", got, want)
	}
	if got, want := symbols[dartTokDocBlockComment], dartSymDocBlockComment; got != want {
		t.Fatalf("zero-value doc-comment symbol = %d, want %d", got, want)
	}
}

func dartExternalBindingTestLanguage(names ...string) *gotreesitter.Language {
	symbolNames := make([]string, len(names)+1)
	symbols := make([]gotreesitter.Symbol, len(names))
	for i, name := range names {
		symbolNames[i+1] = name
		symbols[i] = gotreesitter.Symbol(i + 1)
	}
	return &gotreesitter.Language{
		SymbolNames:     symbolNames,
		ExternalSymbols: symbols,
	}
}
