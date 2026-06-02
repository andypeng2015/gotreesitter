//go:build lexgen

package gotreesitter

import "testing"

func TestScanTypescriptGenMatchesTable(t *testing.T) {
	runLexerDifferential(t, typescriptLexStatesForDiff, typescriptImmediateForDiff, typescriptZeroWidthForDiff, scanTypescriptGen)
}
func TestScanPythonGenMatchesTable(t *testing.T) {
	runLexerDifferential(t, pythonLexStatesForDiff, pythonImmediateForDiff, pythonZeroWidthForDiff, scanPythonGen)
}