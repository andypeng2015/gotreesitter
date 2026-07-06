//go:build !grammar_subset || grammar_subset_rust

package grammars

import (
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestRustExternalScannerBindsGeneratedExternalSymbolsByName(t *testing.T) {
	lang := rustExternalBindingTestLanguage(
		"string_content",
		"_raw_string_literal_start",
		"string_content",
		"_raw_string_literal_end",
		"float_literal",
	)

	scanner, ok := RustExternalScanner{}.ExternalScannerForLanguage(lang).(RustExternalScanner)
	if !ok {
		t.Fatalf("RustExternalScanner binding type = %T, want RustExternalScanner", RustExternalScanner{}.ExternalScannerForLanguage(lang))
	}
	if got, want := scanner.externalToToken[0], rustTokStringContent; got != want {
		t.Fatalf("regular string_content external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[1], rustTokRawStringStart; got != want {
		t.Fatalf("raw-string-start external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[2], rustTokRawStringContent; got != want {
		t.Fatalf("raw string_content external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.symbols[rustTokStringContent], gotreesitter.Symbol(1); got != want {
		t.Fatalf("string_content result symbol = %d, want shifted symbol %d", got, want)
	}
	if got, want := scanner.symbols[rustTokRawStringStart], gotreesitter.Symbol(2); got != want {
		t.Fatalf("raw-string-start result symbol = %d, want shifted symbol %d", got, want)
	}

	validExternal := []bool{false, true, false, false, true}
	var semanticValid [rustTokenCount]bool
	validSemantic := scanner.remapValidSymbols(validExternal, &semanticValid)
	if validSemantic[rustTokStringClose] {
		t.Fatalf("missing string-close external became valid after remap: %v", validSemantic)
	}
	if !validSemantic[rustTokRawStringStart] {
		t.Fatalf("raw-string-start external did not become valid semantic token: %v", validSemantic)
	}
	if !validSemantic[rustTokFloatLiteral] {
		t.Fatalf("float external did not become valid semantic token: %v", validSemantic)
	}
}

func TestRustExternalScannerZeroValueKeepsBuiltinSymbols(t *testing.T) {
	symbols := RustExternalScanner{}.symbolTable()
	if got, want := symbols[rustTokStringContent], rustSymStringContent; got != want {
		t.Fatalf("zero-value string-content symbol = %d, want %d", got, want)
	}
	if got, want := symbols[rustTokStringClose], rustSymStringClose; got != want {
		t.Fatalf("zero-value string-close symbol = %d, want %d", got, want)
	}
}

func rustExternalBindingTestLanguage(names ...string) *gotreesitter.Language {
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
