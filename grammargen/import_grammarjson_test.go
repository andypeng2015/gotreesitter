package grammargen

import "testing"

func TestApplyImportGrammarShapeHintsPowerShellBinaryRepeat(t *testing.T) {
	g := NewGrammar("powershell")
	applyImportGrammarShapeHints(g)
	if !g.BinaryRepeatMode {
		t.Fatal("PowerShell import should use binary repeat mode")
	}
}
