package gotreesitter

func conflictPolicyChoice(lang *Language, tok Token, currentState StateID, actions []ParseAction) (ParseAction, bool) {
	if lang == nil || len(lang.ConflictPolicies) == 0 {
		return ParseAction{}, false
	}
	for i := range lang.ConflictPolicies {
		policy := &lang.ConflictPolicies[i]
		if policy.State != currentState || policy.Lookahead != tok.Symbol {
			continue
		}
		if chosen, ok := conflictPolicyChoiceForPolicy(lang, policy, actions); ok {
			return chosen, true
		}
	}
	return ParseAction{}, false
}

func conflictPolicyChoiceForPolicy(lang *Language, policy *ConflictPolicy, actions []ParseAction) (ParseAction, bool) {
	if lang == nil || policy == nil {
		return ParseAction{}, false
	}
	if !conflictPolicyReducesMatch(policy, actions) {
		return ParseAction{}, false
	}
	switch policy.Kind {
	case ConflictPolicyRepetitionShift:
		return repetitionShiftConflictChoice(actions)
	case ConflictPolicyShift:
		if len(policy.ReduceSymbols) == 0 {
			return ParseAction{}, false
		}
		return singleShiftConflictChoice(actions)
	default:
		return ParseAction{}, false
	}
}

func conflictPolicyReducesMatch(policy *ConflictPolicy, actions []ParseAction) bool {
	if policy == nil || len(policy.ReduceSymbols) == 0 {
		return true
	}
	foundReduce := false
	for _, act := range actions {
		if act.Type != ParseActionReduce {
			continue
		}
		foundReduce = true
		if !conflictPolicySymbolAllowed(policy.ReduceSymbols, act.Symbol) {
			return false
		}
	}
	return foundReduce
}

func conflictPolicySymbolAllowed(allowed []Symbol, sym Symbol) bool {
	for _, candidate := range allowed {
		if candidate == sym {
			return true
		}
	}
	return false
}

func singleShiftConflictChoice(actions []ParseAction) (ParseAction, bool) {
	if len(actions) < 2 {
		return ParseAction{}, false
	}
	var shift ParseAction
	shiftFound := false
	reduceFound := false
	for _, act := range actions {
		switch act.Type {
		case ParseActionShift:
			if act.Extra || act.Repetition {
				return ParseAction{}, false
			}
			if shiftFound {
				return ParseAction{}, false
			}
			shift = act
			shiftFound = true
		case ParseActionReduce:
			reduceFound = true
		default:
			return ParseAction{}, false
		}
	}
	if !shiftFound || !reduceFound {
		return ParseAction{}, false
	}
	return shift, true
}
