package grammargen

import (
	"slices"
	"testing"
)

func TestApplyImportGrammarShapeHintsPowerShellBinaryRepeat(t *testing.T) {
	for _, name := range []string{"d", "objc", "perl", "powershell"} {
		t.Run(name, func(t *testing.T) {
			g := NewGrammar(name)
			applyImportGrammarShapeHints(g)
			if !g.BinaryRepeatMode {
				t.Fatalf("%s import should use binary repeat mode", name)
			}
		})
	}
}

func TestApplyImportGrammarShapeHintsElixirPreciseExternalLexStates(t *testing.T) {
	g := NewGrammar("elixir")
	applyImportGrammarShapeHints(g)
	if !g.PreferPreciseExternalLexStates {
		t.Fatalf("elixir import should prefer precise external lex states")
	}
	if !g.PreferRemoteCallOperatorReduces {
		t.Fatalf("elixir import should prefer remote-call operator reduces")
	}
	if !slices.Equal(g.PreserveHiddenChoicePassthrough, []string{"_capture_expression"}) {
		t.Fatalf("elixir preserved hidden passthrough = %v, want [_capture_expression]", g.PreserveHiddenChoicePassthrough)
	}
}

func TestApplyImportGrammarPostShapeHintsPerlHeredocContent(t *testing.T) {
	g := NewGrammar("perl")
	g.Define("heredoc_content", Seq(
		Sym("_heredoc_start"),
		Repeat(Choice(
			Sym("_heredoc_middle"),
			Sym("escape_sequence"),
			Sym("_interpolations"),
			Sym("_interpolation_fallbacks"),
		)),
		Sym("heredoc_end"),
	))

	applyImportGrammarPostShapeHints(g)

	rule := g.Rules["heredoc_content"]
	if rule == nil || rule.Kind != RuleSeq || len(rule.Children) != 3 {
		t.Fatalf("heredoc_content rule = %#v, want compact seq", rule)
	}
	repeat := rule.Children[1]
	if repeat == nil || repeat.Kind != RuleRepeat || len(repeat.Children) != 1 {
		t.Fatalf("middle rule = %#v, want repeat", repeat)
	}
	choice := repeat.Children[0]
	if choice == nil || choice.Kind != RuleChoice || len(choice.Children) != 2 {
		t.Fatalf("repeat content = %#v, want compact two-way choice", choice)
	}
	if got := []string{choice.Children[0].Value, choice.Children[1].Value}; got[0] != "_heredoc_middle" || got[1] != "escape_sequence" {
		t.Fatalf("compact heredoc alternatives = %v, want [_heredoc_middle escape_sequence]", got)
	}
}

// TestApplyImportGrammarPostShapeHintsRubyHeredocBody mirrors
// TestApplyImportGrammarPostShapeHintsPerlHeredocContent for ruby's
// heredoc_body, which has the identical pathology: a nonterminal grammar
// extra whose REPEAT(CHOICE(...)) body includes a visible `interpolation`
// alternative that re-enters `_statements` (the entire statement grammar).
// See GEN_COST_RCA (wave7, "ruby - memory in add_nonterminal_extra_chains")
// and the perl precedent this rewrite is modeled on.
func TestApplyImportGrammarPostShapeHintsRubyHeredocBody(t *testing.T) {
	g := NewGrammar("ruby")
	g.Define("heredoc_body", Seq(
		Sym("_heredoc_body_start"),
		Repeat(Choice(
			Sym("heredoc_content"),
			Sym("interpolation"),
			Sym("escape_sequence"),
		)),
		Sym("heredoc_end"),
	))

	applyImportGrammarPostShapeHints(g)

	rule := g.Rules["heredoc_body"]
	if rule == nil || rule.Kind != RuleSeq || len(rule.Children) != 3 {
		t.Fatalf("heredoc_body rule = %#v, want compact seq", rule)
	}
	if rule.Children[0].Value != "_heredoc_body_start" {
		t.Fatalf("heredoc_body start = %#v, want _heredoc_body_start", rule.Children[0])
	}
	if rule.Children[2].Value != "heredoc_end" {
		t.Fatalf("heredoc_body end = %#v, want heredoc_end", rule.Children[2])
	}
	repeat := rule.Children[1]
	if repeat == nil || repeat.Kind != RuleRepeat || len(repeat.Children) != 1 {
		t.Fatalf("middle rule = %#v, want repeat", repeat)
	}
	choice := repeat.Children[0]
	if choice == nil || choice.Kind != RuleChoice || len(choice.Children) != 2 {
		t.Fatalf("repeat content = %#v, want compact two-way choice", choice)
	}
	if got := []string{choice.Children[0].Value, choice.Children[1].Value}; got[0] != "heredoc_content" || got[1] != "escape_sequence" {
		t.Fatalf("compact heredoc alternatives = %v, want [heredoc_content escape_sequence]", got)
	}
	// The recursive interpolation -> _statements path must be gone: it is the
	// nonterminal-extra chain that never converges (GEN_COST_RCA).
	for _, child := range choice.Children {
		if child.Value == "interpolation" {
			t.Fatalf("rewritten heredoc_body must not reference interpolation, got %#v", rule)
		}
	}
}

// TestApplyImportGrammarPostShapeHintsRubyHeredocBodyIsGatedToRuby confirms
// the rewrite is scoped to lang name "ruby" only, as required: crystal's
// heredoc_body has the identical shape (same rule and symbol names,
// independently confirmed against tree-sitter-crystal's grammar.json) but is
// out of scope for this change and must be left untouched. It remains
// protected only by the defense-in-depth synthetic-state budget guard in
// addNonterminalExtraChains.
func TestApplyImportGrammarPostShapeHintsRubyHeredocBodyIsGatedToRuby(t *testing.T) {
	original := Seq(
		Sym("_heredoc_body_start"),
		Repeat(Choice(
			Sym("heredoc_content"),
			Sym("interpolation"),
			Sym("escape_sequence"),
		)),
		Sym("heredoc_end"),
	)

	g := NewGrammar("crystal")
	g.Define("heredoc_body", cloneRule(original))

	applyImportGrammarPostShapeHints(g)

	rule := g.Rules["heredoc_body"]
	repeat := rule.Children[1]
	choice := repeat.Children[0]
	if len(choice.Children) != 3 {
		t.Fatalf("crystal's heredoc_body should be left untouched (3-way choice), got %#v", choice)
	}
	found := false
	for _, child := range choice.Children {
		if child.Value == "interpolation" {
			found = true
		}
	}
	if !found {
		t.Fatalf("crystal's heredoc_body should still reference interpolation, got %#v", choice)
	}
}
