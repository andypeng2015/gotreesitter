//go:build lexgen

package gotreesitter

// lookupGenScan maps a language name to its generated switch-DFA lexer
// (cmd/lexgen output). Compiled only with -tags lexgen. Each generated lexer is
// gated by its differential test before it is trusted.
func lookupGenScan(name string) lexerScanFn {
	switch name {
	case "json":
		return scanJsonGen
	case "javascript":
		return scanJavascriptGen
	case "typescript":
		return scanTypescriptGen
	case "python":
		return scanPythonGen
	}
	return nil
}
