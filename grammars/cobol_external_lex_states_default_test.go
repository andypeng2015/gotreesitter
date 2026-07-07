//go:build !cobol_precise_els

package grammars

import "testing"

func TestCobolExternalLexStatesRemainStagedByDefault(t *testing.T) {
	if got := len(LookupExternalLexStates("cobol")); got != 0 {
		t.Fatalf("LookupExternalLexStates(%q) rows = %d, want 0 without cobol_precise_els", "cobol", got)
	}
	lang := CobolLanguage()
	if got := len(lang.ExternalLexStates); got != 0 {
		t.Fatalf("CobolLanguage ExternalLexStates rows = %d, want 0 without cobol_precise_els", got)
	}
	if lang.CRecoveryCostCompetitionEnabledByDefault {
		t.Fatal("cobol C recovery election default-enabled without cobol_precise_els")
	}
}
