//go:build lexgen

package gotreesitter

import "testing"

func TestScanJavascriptGenMatchesTable(t *testing.T) {
	runLexerDifferential(t, javascriptLexStatesForDiff, javascriptImmediateForDiff, javascriptZeroWidthForDiff, scanJavascriptGen)
}