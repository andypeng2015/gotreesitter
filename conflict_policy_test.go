package gotreesitter

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestConflictPolicyChoiceRepetitionShift(t *testing.T) {
	lang := &Language{
		ConflictPolicies: []ConflictPolicy{{State: 42, Lookahead: 7, Kind: ConflictPolicyRepetitionShift, ReduceSymbols: []Symbol{3}}},
	}
	actions := []ParseAction{
		{Type: ParseActionReduce, Symbol: 3, ChildCount: 2},
		{Type: ParseActionShift, State: 99, Repetition: true},
	}

	chosen, ok := conflictPolicyChoice(lang, Token{Symbol: 7}, 42, actions)
	if !ok {
		t.Fatal("conflictPolicyChoice = false, want true")
	}
	if chosen.Type != ParseActionShift || chosen.State != 99 || !chosen.Repetition {
		t.Fatalf("conflictPolicyChoice picked %+v, want repetition shift", chosen)
	}
}

func TestConflictPolicyChoiceShift(t *testing.T) {
	lang := &Language{
		ConflictPolicies: []ConflictPolicy{{State: 42, Lookahead: 7, Kind: ConflictPolicyShift, ReduceSymbols: []Symbol{3}}},
	}
	actions := []ParseAction{
		{Type: ParseActionReduce, Symbol: 3, ChildCount: 1},
		{Type: ParseActionShift, State: 99},
	}

	chosen, ok := conflictPolicyChoice(lang, Token{Symbol: 7}, 42, actions)
	if !ok {
		t.Fatal("conflictPolicyChoice = false, want shift policy choice")
	}
	if chosen.Type != ParseActionShift || chosen.State != 99 {
		t.Fatalf("conflictPolicyChoice picked %+v, want shift", chosen)
	}
}

func TestConflictPolicyChoiceShiftRequiresReduceSymbols(t *testing.T) {
	lang := &Language{
		ConflictPolicies: []ConflictPolicy{{State: 42, Lookahead: 7, Kind: ConflictPolicyShift}},
	}
	actions := []ParseAction{
		{Type: ParseActionReduce, Symbol: 3, ChildCount: 1},
		{Type: ParseActionShift, State: 99},
	}

	if chosen, ok := conflictPolicyChoice(lang, Token{Symbol: 7}, 42, actions); ok {
		t.Fatalf("conflictPolicyChoice picked %+v, want no unconstrained shift policy", chosen)
	}
}

func TestConflictPolicyChoiceShiftRejectsExtraAndRepetitionShifts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		shift ParseAction
	}{
		{name: "extra", shift: ParseAction{Type: ParseActionShift, State: 99, Extra: true}},
		{name: "repetition", shift: ParseAction{Type: ParseActionShift, State: 99, Repetition: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lang := &Language{
				ConflictPolicies: []ConflictPolicy{{State: 42, Lookahead: 7, Kind: ConflictPolicyShift, ReduceSymbols: []Symbol{3}}},
			}
			actions := []ParseAction{
				{Type: ParseActionReduce, Symbol: 3, ChildCount: 1},
				tc.shift,
			}

			if chosen, ok := conflictPolicyChoice(lang, Token{Symbol: 7}, 42, actions); ok {
				t.Fatalf("conflictPolicyChoice picked %+v, want rejection", chosen)
			}
		})
	}
}

func TestConflictPolicyChoiceRejectsNonmatchingStateLookahead(t *testing.T) {
	lang := &Language{
		ConflictPolicies: []ConflictPolicy{{State: 42, Lookahead: 7, Kind: ConflictPolicyRepetitionShift}},
	}
	actions := []ParseAction{
		{Type: ParseActionReduce, Symbol: 3, ChildCount: 2},
		{Type: ParseActionShift, State: 99, Repetition: true},
	}

	if _, ok := conflictPolicyChoice(lang, Token{Symbol: 7}, 41, actions); ok {
		t.Fatal("conflictPolicyChoice matched wrong state")
	}
	if _, ok := conflictPolicyChoice(lang, Token{Symbol: 8}, 42, actions); ok {
		t.Fatal("conflictPolicyChoice matched wrong lookahead")
	}
}

func TestConflictPolicyChoiceAgreesWithLegacyRepetitionHelperShape(t *testing.T) {
	lang := &Language{
		Name:             "synthetic_policy_test",
		SymbolNames:      []string{"end", "namespace", "\\", "name", "use", "new", "program_repeat1"},
		ConflictPolicies: []ConflictPolicy{{State: 2, Lookahead: 1, Kind: ConflictPolicyRepetitionShift, ReduceSymbols: []Symbol{6}}},
	}
	actions := []ParseAction{
		{Type: ParseActionReduce, Symbol: 6, ChildCount: 2},
		{Type: ParseActionShift, State: 1846, Repetition: true},
	}

	legacy, legacyOK := phpRepetitionShiftConflictChoice(lang, Token{Symbol: 1}, 2, actions)
	policy, policyOK := conflictPolicyChoice(lang, Token{Symbol: 1}, 2, actions)
	if !legacyOK || !policyOK {
		t.Fatalf("legacyOK=%v policyOK=%v, want both true", legacyOK, policyOK)
	}
	if legacy != policy {
		t.Fatalf("policy picked %+v, legacy picked %+v", policy, legacy)
	}
}

func TestConflictPolicyGobRoundTrip(t *testing.T) {
	lang := &Language{
		Name:             "synthetic_policy_test",
		ConflictPolicies: []ConflictPolicy{{State: 42, Lookahead: 7, Kind: ConflictPolicyRepetitionShift, ReduceSymbols: []Symbol{3, 4}}},
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(lang); err != nil {
		t.Fatalf("encode Language with ConflictPolicies: %v", err)
	}
	var decoded Language
	if err := gob.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("decode Language with ConflictPolicies: %v", err)
	}
	if len(decoded.ConflictPolicies) != 1 {
		t.Fatalf("decoded %d policies, want 1", len(decoded.ConflictPolicies))
	}
	got := decoded.ConflictPolicies[0]
	if got.State != 42 || got.Lookahead != 7 || got.Kind != ConflictPolicyRepetitionShift {
		t.Fatalf("decoded policy = %+v, want state/lookahead/kind preserved", got)
	}
	if len(got.ReduceSymbols) != 2 || got.ReduceSymbols[0] != 3 || got.ReduceSymbols[1] != 4 {
		t.Fatalf("decoded ReduceSymbols = %v, want [3 4]", got.ReduceSymbols)
	}
}
