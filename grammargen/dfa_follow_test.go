package grammargen

import "testing"

func TestComputeLexModesFollowExpansionSkipsImmediateTokens(t *testing.T) {
	lexModes, stateToMode, _ := computeLexModes(
		1,
		5,
		func(state, sym int) bool {
			return sym == 1
		},
		nil,
		nil,
		-1,
		map[int]bool{2: true},
		nil,
		0,
		nil,
		map[int]bool{1: true, 2: true, 3: true},
		func(state int) []int {
			return []int{2, 3}
		},
		nil,
		nil,
	)

	if len(stateToMode) != 1 || len(lexModes) == 0 {
		t.Fatalf("unexpected lex mode result: stateToMode=%v modes=%d", stateToMode, len(lexModes))
	}

	mode := lexModes[stateToMode[0]]
	if !mode.validSymbols[1] {
		t.Fatal("directly valid symbol missing from lex mode")
	}
	if mode.validSymbols[2] {
		t.Fatal("immediate follow token leaked into lex mode")
	}
	if !mode.validSymbols[3] {
		t.Fatal("non-immediate follow token should remain in lex mode")
	}
}

func TestComputeLexModesFollowExpansionSkipsCatchAllPatternTokens(t *testing.T) {
	lexModes, stateToMode, _ := computeLexModes(
		1,
		6,
		func(state, sym int) bool {
			return sym == 1
		},
		nil,
		nil,
		-1,
		nil,
		nil,
		0,
		nil,
		map[int]bool{1: true, 2: true, 3: true},
		func(state int) []int {
			return []int{2, 3}
		},
		nil,
		map[int]bool{2: true},
	)

	if len(stateToMode) != 1 || len(lexModes) == 0 {
		t.Fatalf("unexpected lex mode result: stateToMode=%v modes=%d", stateToMode, len(lexModes))
	}

	mode := lexModes[stateToMode[0]]
	if !mode.validSymbols[1] {
		t.Fatal("directly valid symbol missing from lex mode")
	}
	if mode.validSymbols[2] {
		t.Fatal("catch-all follow token leaked into lex mode")
	}
	if !mode.validSymbols[3] {
		t.Fatal("non-catch-all follow token should remain in lex mode")
	}
}

func TestRuleStartsWithCatchAllPattern(t *testing.T) {
	tests := []struct {
		name string
		rule *Rule
		want bool
	}{
		{
			name: "negated_char_class_repeat",
			rule: Repeat1(Pat(`[^()]`)),
			want: true,
		},
		{
			name: "literal_prefix_then_broad_pattern",
			rule: Seq(Str("!"), Repeat(Pat(`[^\n]`))),
			want: false,
		},
		{
			name: "identifier_prefix_class",
			rule: Seq(Pat(`[a-zA-Z_]`), Repeat(Pat(`[a-zA-Z0-9_]`))),
			want: false,
		},
		{
			name: "optional_prefix_before_broad_pattern",
			rule: Seq(Optional(Str("%")), Repeat1(Pat(`[^\n]`))),
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ruleStartsWithCatchAllPattern(tc.rule); got != tc.want {
				t.Fatalf("ruleStartsWithCatchAllPattern() = %v, want %v", got, tc.want)
			}
		})
	}
}
