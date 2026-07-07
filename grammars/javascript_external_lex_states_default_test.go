//go:build !javascript_precise_els

package grammars

import "testing"

func TestJavascriptExternalLexStatesRemainStagedByDefault(t *testing.T) {
	if got := len(LookupExternalLexStates("javascript")); got != 0 {
		t.Fatalf("LookupExternalLexStates(\"javascript\") rows = %d, want 0 without javascript_precise_els", got)
	}
	lang := JavascriptLanguage()
	if len(lang.ExternalLexStates) != 0 {
		t.Fatalf("JavascriptLanguage ExternalLexStates rows = %d, want 0 without javascript_precise_els", len(lang.ExternalLexStates))
	}
	if lang.CRecoveryCostCompetitionEnabledByDefault {
		t.Fatal("javascript C recovery election default-enabled without javascript_precise_els")
	}
}
