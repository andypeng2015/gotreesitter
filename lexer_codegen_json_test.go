package gotreesitter

import "testing"

// TestScanJsonGenMatchesTable is the differential gate for the generated json
// lexer: scanJsonGen must return byte-for-byte the same token (and advance the
// lexer identically) as the table scan() for every state at every position
// across diffInputs.
func TestScanJsonGenMatchesTable(t *testing.T) {
	runLexerDifferential(t, jsonLexStatesForDiff, jsonImmediateForDiff, jsonZeroWidthForDiff, scanJsonGen)
}
