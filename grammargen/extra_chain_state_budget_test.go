package grammargen

import (
	"strings"
	"testing"
)

// TestExtraChainSyntheticStateBudgetFormula unit-tests the budget math in
// isolation: max(multiplier * mainStateCount, floor), with an env override
// for testing/diagnosis. See GEN_COST_RCA (wave7) and the doc comment on
// extraChainSyntheticStateBudget for how the default multiplier/floor were
// chosen from measured fleet data (cobol's copy_statement extra is the
// largest observed legitimate nonterminal-extra chain: 22027 main states ->
// 171240 synthetic states, ~7.8x).
func TestExtraChainSyntheticStateBudgetFormula(t *testing.T) {
	t.Run("floor dominates for small grammars", func(t *testing.T) {
		got := extraChainSyntheticStateBudget(1000)
		want := extraChainSyntheticStateFloor
		if got != want {
			t.Errorf("extraChainSyntheticStateBudget(1000) = %d, want floor %d", got, want)
		}
	})

	t.Run("multiplier dominates for large grammars", func(t *testing.T) {
		// 22027 main states mirrors cobol's measured mainStateCount.
		got := extraChainSyntheticStateBudget(22027)
		want := 22027 * extraChainSyntheticStateMultiplier
		if got != want {
			t.Errorf("extraChainSyntheticStateBudget(22027) = %d, want %d", got, want)
		}
		// cobol legitimately needs 171240 synthetic states; the computed cap
		// must clear that with real headroom or the guard would break cobol.
		const cobolObservedSynthetic = 171240
		if got <= cobolObservedSynthetic {
			t.Fatalf("computed budget %d does not clear cobol's observed synthetic-state count %d", got, cobolObservedSynthetic)
		}
	})

	t.Run("env override wins regardless of mainStateCount", func(t *testing.T) {
		t.Setenv(extraChainStateBudgetEnv, "7")
		if got := extraChainSyntheticStateBudget(1_000_000); got != 7 {
			t.Errorf("extraChainSyntheticStateBudget(1_000_000) with override = %d, want 7", got)
		}
	})

	t.Run("invalid env override falls back to the computed default", func(t *testing.T) {
		for _, raw := range []string{"0", "-5", "not-a-number", ""} {
			t.Setenv(extraChainStateBudgetEnv, raw)
			got := extraChainSyntheticStateBudget(1000)
			want := extraChainSyntheticStateFloor
			if got != want {
				t.Errorf("extraChainSyntheticStateBudget(1000) with override %q = %d, want fallback %d", raw, got, want)
			}
		}
	})
}

// TestNonterminalExtraChainSyntheticStateBudgetTripsFast is the regression
// test requested alongside the ruby heredoc_body fix: a synthetic grammar
// with the same pathological shape as ruby's pre-rewrite heredoc_body (a
// nonterminal grammar extra whose body embeds a statement-embedding
// alternative - "interpolation" re-entering the full statement grammar, the
// same relationship as ruby's interpolation -> _statements) must hard-fail
// fast and clearly once an artificially tiny synthetic-state budget is
// exceeded, instead of growing unboundedly.
//
// This grammar is small enough that it fully converges on its own (339
// synthetic states from 20 main states - see the "natural" subtest), so this
// test stays fast even if the budget guard were ever disabled; the budget
// override just makes it fail long before reaching that natural size.
func TestNonterminalExtraChainSyntheticStateBudgetTripsFast(t *testing.T) {
	buildBudgetGrammar := func() (*NormalizedGrammar, *LRTables, *lrContext) {
		g := NewGrammar("extra_chain_budget_synthetic")
		g.Define("source_file", Repeat(Sym("statement")))
		g.Define("statement", Choice(
			Sym("if_statement"),
			Sym("while_statement"),
			Sym("assign_statement"),
			Sym("word"),
		))
		g.Define("word", Pat(`[a-z]+`))
		g.Define("if_statement", Seq(Str("if"), Sym("word"), Str("then"), Sym("statement")))
		g.Define("while_statement", Seq(Str("while"), Sym("word"), Str("do"), Sym("statement")))
		g.Define("assign_statement", Seq(Sym("word"), Str("="), Sym("statement")))
		// interpolation mirrors ruby's pre-rewrite interpolation -> _statements:
		// a nonterminal-extra alternative that re-enters the entire statement
		// grammar rather than staying scanner-delimited.
		g.Define("interpolation", Seq(Str("#{"), Optional(Sym("statement")), Str("}")))
		g.Define("heredoc_content", Pat(`[^#}]+`))
		// heredoc_body mirrors ruby's exact pre-rewrite shape:
		// SEQ(start, REPEAT(CHOICE(content, interpolation)), end), declared as a
		// nonterminal grammar extra (see GEN_COST_RCA).
		g.Define("heredoc_body", Seq(
			Str("<<~X"),
			Repeat(Choice(Sym("heredoc_content"), Sym("interpolation"))),
			Str("X_END"),
		))
		g.SetExtras(Pat(`\s`), Sym("heredoc_body"))

		ng, err := Normalize(g)
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		tables, ctx, err := buildLRTablesWithProvenance(ng)
		if err != nil {
			t.Fatalf("build LR tables: %v", err)
		}
		return ng, tables, ctx
	}

	t.Run("natural size converges (sanity, no override)", func(t *testing.T) {
		ng, tables, ctx := buildBudgetGrammar()
		addNonterminalExtraChains(tables, ng, ctx)
		synthetic := tables.StateCount - tables.ExtraChainStateStart
		if synthetic < 100 {
			t.Fatalf("expected the statement-embedding extra to mint a nontrivial number of synthetic states, got %d - budget-trip subtest below would not be meaningful", synthetic)
		}
		t.Logf("natural synthetic-state count: %d (main=%d)", synthetic, tables.ExtraChainStateStart)
	})

	t.Run("tiny budget trips fast with a clear, named error", func(t *testing.T) {
		const tinyBudget = 8
		t.Setenv(extraChainStateBudgetEnv, "8")
		ng, tables, ctx := buildBudgetGrammar()

		defer func() {
			r := recover()
			if r == nil {
				t.Fatalf("expected addNonterminalExtraChains to panic once the synthetic-state budget was exceeded, but it returned normally (states=%d, main=%d)",
					tables.StateCount, tables.ExtraChainStateStart)
			}
			budgetErr, ok := r.(*ExtraChainSyntheticStateBudgetError)
			if !ok {
				// Not our budget error - something else went wrong; re-raise so
				// the test fails loudly with the real panic and stack trace.
				panic(r)
			}
			if budgetErr.Budget != tinyBudget {
				t.Errorf("Budget = %d, want %d", budgetErr.Budget, tinyBudget)
			}
			if budgetErr.SyntheticCount < budgetErr.Budget {
				t.Errorf("SyntheticCount (%d) should be >= Budget (%d) at the point of failure", budgetErr.SyntheticCount, budgetErr.Budget)
			}
			if budgetErr.Grammar != "extra_chain_budget_synthetic" {
				t.Errorf("Grammar = %q, want %q", budgetErr.Grammar, "extra_chain_budget_synthetic")
			}
			if budgetErr.Symbol == "" || budgetErr.Symbol == "<unknown>" {
				t.Errorf("expected the error to name the offending nonterminal-extra symbol, got %q", budgetErr.Symbol)
			}
			if !strings.Contains(budgetErr.Symbol, "heredoc_body") {
				t.Errorf("expected the offending symbol to mention heredoc_body, got %q", budgetErr.Symbol)
			}
			msg := budgetErr.Error()
			for _, want := range []string{"synthetic-state", "budget", "heredoc_body", "extra_chain_budget_synthetic"} {
				if !strings.Contains(msg, want) {
					t.Errorf("error message %q missing expected substring %q", msg, want)
				}
			}
			t.Logf("got expected budget error: %v", budgetErr)
		}()

		addNonterminalExtraChains(tables, ng, ctx)
		t.Fatalf("unreachable: addNonterminalExtraChains should have panicked (states=%d, main=%d)", tables.StateCount, tables.ExtraChainStateStart)
	})
}
