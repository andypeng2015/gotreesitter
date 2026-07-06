//go:build !grammar_subset || grammar_subset_python

package grammars

import (
	"slices"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestPythonExternalScannerSpec(t *testing.T) {
	spec, ok := LookupExternalScannerSpec("python")
	if !ok {
		t.Fatal("missing python external scanner spec")
	}
	if got, want := spec.UpstreamRepo, "https://github.com/tree-sitter/tree-sitter-python"; got != want {
		t.Fatalf("python repo = %q, want %q", got, want)
	}
	if got, want := spec.Externals, []string{
		"_newline",
		"_indent",
		"_dedent",
		"string_start",
		"_string_content",
		"escape_interpolation",
		"string_end",
		"comment",
		"]",
		")",
		"}",
		"except",
	}; !slices.Equal(got, want) {
		t.Fatalf("python externals = %v, want %v", got, want)
	}
}

func TestPythonExternalScannerBindsShiftedExternalSymbolsByName(t *testing.T) {
	lang := pythonExternalBindingTestLanguage(
		"_extension_only",
		"escape_interpolation",
		"_indent",
		"string_start",
		"_newline",
	)

	scanner, ok := PythonExternalScanner{}.ExternalScannerForLanguage(lang).(PythonExternalScanner)
	if !ok {
		t.Fatalf("PythonExternalScanner binding type = %T, want PythonExternalScanner", PythonExternalScanner{}.ExternalScannerForLanguage(lang))
	}
	if got, want := scanner.externalToToken[0], -1; got != want {
		t.Fatalf("extension-only external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[1], pyTokEscapeInterpolation; got != want {
		t.Fatalf("escape-interpolation external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[2], pyTokIndent; got != want {
		t.Fatalf("indent external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[3], pyTokStringStart; got != want {
		t.Fatalf("string-start external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.externalToToken[4], pyTokNewline; got != want {
		t.Fatalf("newline external mapped to token %d, want %d", got, want)
	}
	if got, want := scanner.symbols[pyTokEscapeInterpolation], gotreesitter.Symbol(2); got != want {
		t.Fatalf("escape-interpolation result symbol = %d, want %d", got, want)
	}

	validExternal := []bool{false, true, false, false, false}
	var semanticValid [pyTokenCount]bool
	validSemantic := scanner.remapValidSymbols(validExternal, &semanticValid)
	if !validSemantic[pyTokEscapeInterpolation] {
		t.Fatalf("shifted escape-interpolation external did not become valid semantic token: %v", validSemantic)
	}
}

func TestPythonExternalScannerZeroValueKeepsBuiltinSymbols(t *testing.T) {
	symbols := PythonExternalScanner{}.symbolTable()
	if got, want := symbols[pyTokNewline], pySymNewline; got != want {
		t.Fatalf("zero-value newline symbol = %d, want %d", got, want)
	}
	if got, want := symbols[pyTokStringContent], pySymStringContent; got != want {
		t.Fatalf("zero-value string-content symbol = %d, want %d", got, want)
	}
}

func pythonExternalBindingTestLanguage(names ...string) *gotreesitter.Language {
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
