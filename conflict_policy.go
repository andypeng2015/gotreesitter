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

func (p *Parser) deterministicConflictChoiceForDispatch(source []byte, s *glrStack, tok Token, currentState StateID, actions []ParseAction, maxStacksSeen int, reuse *reuseCursor) (ParseAction, bool) {
	if p == nil || p.language == nil {
		return ParseAction{}, false
	}
	if p.language.Name == "gomod" {
		if next, ok := gomodRepetitionShiftConflictChoice(p.language, currentState, actions); ok {
			return next, true
		}
	}
	if reuse != nil {
		return ParseAction{}, false
	}
	if chosen, ok := conflictPolicyChoice(p.language, tok, currentState, actions); ok {
		return chosen, true
	}
	if generatedRepeatBoundaryConflict(p.language, actions) {
		return ParseAction{}, false
	}
	if p.language.GeneratedByGrammargen {
		return ParseAction{}, false
	}
	var chosen ParseAction
	var ok bool
	switch p.language.Name {
	case "java":
		if next, ok := p.javaSwitchArrowConflictChoice(s, tok, actions); ok {
			return next, true
		}
		chosen, ok = javaRepetitionShiftConflictChoiceForDispatch(p.language, source, tok, currentState, actions)
	case "c_sharp":
		chosen, ok = csharpRepetitionShiftConflictChoice(p.language, tok, actions)
	case "c":
		chosen, ok = cRepetitionShiftConflictChoice(p.language, actions)
	case "rust":
		if p.noTreeBenchmarkOnly {
			return ParseAction{}, false
		}
		chosen, ok = rustRepetitionShiftConflictChoice(p.language, tok, currentState, actions)
	case "typescript":
		chosen, ok = typescriptRepetitionShiftConflictChoiceForDispatch(p.language, tok, currentState, actions)
	case "tsx":
		chosen, ok = tsxRepetitionReduceConflictChoice(p.language, tok, currentState, actions)
	case "javascript":
		chosen, ok = javascriptRepetitionShiftConflictChoiceForDispatch(p.language, tok, currentState, actions)
	case "python":
		chosen, ok = pythonRepetitionShiftConflictChoice(p.language, tok, currentState, actions)
	case "r":
		chosen, ok = rRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "php":
		chosen, ok = phpRepetitionShiftConflictChoice(p.language, tok, currentState, actions)
	case "perl":
		chosen, ok = perlRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "sql":
		chosen, ok = sqlRepetitionShiftConflictChoice(p.language, tok, currentState, actions)
	case "dart":
		chosen, ok = dartRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "hcl":
		chosen, ok = hclRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "haskell":
		chosen, ok = haskellRepeatBoundaryConflictChoice(p.language, currentState, actions)
	case "make":
		chosen, ok = makeRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "swift":
		chosen, ok = swiftBraceTypeExpressionConflictChoice(p.language, tok, currentState, actions)
	case "d":
		chosen, ok = dRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "clojure":
		chosen, ok = clojureRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "awk":
		chosen, ok = awkRepetitionShiftConflictChoice(p.language, currentState, actions)
	case "scheme":
		chosen, ok = schemeRepetitionShiftConflictChoice(p.language, tok, currentState, actions)
	case "dot":
		chosen, ok = dotRepetitionShiftConflictChoice(p.language, tok, currentState, actions)
	case "kotlin":
		chosen, ok = kotlinObjectLiteralConflictChoice(p.language, actions)
	case "erlang":
		chosen, ok = erlangMacroCallExprConflictChoice(p.language, actions)
	}
	return chosen, ok
}

func generatedRepeatBoundaryConflict(lang *Language, actions []ParseAction) bool {
	if lang == nil || len(actions) < 2 {
		return false
	}
	// The repeat-boundary rejection exists so grammargen-generated grammars
	// without an explicit ConflictPolicy fork instead of trusting hand-written
	// per-language shortcuts that were never tuned for them. Embedded blobs
	// (c_sharp, java, c, ...) get GeneratedRepeatAux retrofitted by
	// InferGeneratedRepeatAuxMetadata (load_language.go / embedded_loader.go),
	// but their repeat-boundary conflicts are exactly what the per-language
	// deterministic choices below were written for; rejecting here disables
	// those choices and the GLR loop forks on every repetition boundary
	// (C# designer-style blocks grew live stacks linearly with input size:
	// MaxStacksSeen 2064 at 300 statements, arena exhaustion, never accepts).
	// Scope the rejection to languages that actually rely on generated
	// policies. For grammargen languages this is a no-op: both call sites
	// (deterministicConflictChoiceForDispatch below, forestResolveConflict in
	// parser.go) check GeneratedByGrammargen immediately after this predicate
	// and bail out identically.
	if !lang.GeneratedByGrammargen && len(lang.ConflictPolicies) == 0 {
		return false
	}
	shiftFound := false
	generatedReduceFound := false
	for _, act := range actions {
		switch act.Type {
		case ParseActionShift:
			if !act.Repetition || act.Extra || shiftFound {
				return false
			}
			shiftFound = true
		case ParseActionReduce:
			if languageSymbolIsGeneratedRepeatAux(lang, act.Symbol) {
				generatedReduceFound = true
			}
		default:
			return false
		}
	}
	return shiftFound && generatedReduceFound
}

func languageSymbolIsGeneratedRepeatAux(lang *Language, sym Symbol) bool {
	if lang == nil {
		return false
	}
	idx := int(sym)
	if idx < 0 || idx >= len(lang.SymbolMetadata) {
		return false
	}
	return lang.SymbolMetadata[idx].GeneratedRepeatAux
}
