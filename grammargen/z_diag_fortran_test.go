package grammargen

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func loadDiagFortranGrammar(t *testing.T) *Grammar {
	t.Helper()

	var data []byte
	var err error
	paths := []string{
		"/tmp/fortran_diag/src/grammar.json",
		"/tmp/grammar_parity/fortran/src/grammar.json",
		filepath.Join(os.TempDir(), "fortran_diag", "src", "grammar.json"),
	}
	if matches, globErr := filepath.Glob(filepath.Join(os.TempDir(), "fortran_diag.*", "src", "grammar.json")); globErr == nil {
		paths = append(paths, matches...)
	}
	for _, path := range paths {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("fortran grammar.json not available: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	gram.ChoiceLiftThreshold = 4
	return gram
}

func TestDiagFortranTinySamples(t *testing.T) {
	gram := loadDiagFortranGrammar(t)
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	refLang := grammars.FortranLanguage()
	adaptExternalScanner(refLang, genLang)

	samples := []struct {
		name string
		src  string
	}{
		{
			name: "program_only",
			src:  "PROGRAM TEST\nEND PROGRAM\n",
		},
		{
			name: "program_comment",
			src:  "PROGRAM TEST\n  ! placeholder file until I have real examples\nEND PROGRAM\n",
		},
		{
			name: "subroutine_only",
			src:  "SUBROUTINE FOO\nEND SUBROUTINE\n",
		},
		{
			name: "program_assignment",
			src:  "PROGRAM TEST\n  X = 1\nEND PROGRAM\n",
		},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			genTree, err := gotreesitter.NewParser(genLang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			refTree, err := gotreesitter.NewParser(refLang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}
			t.Logf("genErr=%v refErr=%v runtime=%s", genTree.RootNode().HasError(), refTree.RootNode().HasError(), genTree.ParseRuntime().Summary())
			t.Logf("gen=%s", genTree.RootNode().SExpr(genLang))
			t.Logf("ref=%s", refTree.RootNode().SExpr(refLang))
		})
	}
}

func TestDiagFortranProgramPrefixStates(t *testing.T) {
	gram := loadDiagFortranGrammar(t)
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	addNonterminalExtraChains(tables, ng, ctx)
	if err := resolveConflicts(tables, ng); err != nil {
		t.Fatalf("resolveConflicts: %v", err)
	}
	if err := resolveConflicts(tables, ng); err != nil {
		t.Fatalf("resolveConflicts: %v", err)
	}

	for _, name := range []string{
		"program", "_name", "name", "identifier", "comment", "_end_of_statement",
		"_external_end_of_statement", ";", "endprogram", "translation_unit",
		"_translation_unit_choice_lift2", "program_statement",
	} {
		ids := diagFindAllSymbols(ng, name)
		if len(ids) == 0 {
			continue
		}
		for _, id := range ids {
			t.Logf("symbol %q -> id=%d kind=%v visible=%v named=%v extra=%v",
				name, id, ng.Symbols[id].Kind, ng.Symbols[id].Visible, ng.Symbols[id].Named, ng.Symbols[id].IsExtra)
		}
	}

	programTok := -1
	commentTok := -1
	eosExtTok := -1
	eosSemiTok := -1
	for i, sym := range ng.Symbols {
		if sym.Kind != SymbolTerminal && sym.Kind != SymbolNamedToken && sym.Kind != SymbolExternal {
			continue
		}
		switch sym.Name {
		case "program":
			if programTok < 0 {
				programTok = i
			}
		case "comment":
			if commentTok < 0 {
				commentTok = i
			}
		case "_external_end_of_statement":
			if eosExtTok < 0 {
				eosExtTok = i
			}
		case ";":
			if eosSemiTok < 0 {
				eosSemiTok = i
			}
		}
	}
	t.Logf("state0 on program: %s", diagFormatActions(ng, tables.ActionTable[0][programTok]))
	if commentTok >= 0 {
		t.Logf("state0 on comment: %s", diagFormatActions(ng, tables.ActionTable[0][commentTok]))
	}

	programStatementIDs := diagFindAllSymbols(ng, "program_statement")
	if len(programStatementIDs) != 1 {
		t.Fatalf("program_statement symbol ids = %v", programStatementIDs)
	}
	programStatementID := programStatementIDs[0]

	found := 0
	for state := 0; state < len(ctx.itemSets); state++ {
		set := &ctx.itemSets[state]
		for _, ce := range set.cores {
			prod := &ng.Productions[ce.prodIdx]
			if prod.LHS != programStatementID {
				continue
			}
			if ce.dot >= len(prod.RHS) {
				continue
			}
			nextSym := prod.RHS[ce.dot]
			nextName := diagSymbolName(ng, nextSym)
			if nextName != "_name" && nextName != "_end_of_statement" &&
				nextName != "_external_end_of_statement" && nextName != ";" {
				continue
			}
			t.Logf("state=%d merge=%d item=%s", state, diagMergeCount(ctx, state), diagFormatProd(ng, ce.prodIdx, ce.dot))
			if programTok >= 0 {
				t.Logf("  on program: %s", diagFormatActions(ng, tables.ActionTable[state][programTok]))
			}
			if commentTok >= 0 {
				t.Logf("  on comment: %s", diagFormatActions(ng, tables.ActionTable[state][commentTok]))
			}
			if eosExtTok >= 0 {
				t.Logf("  on external_eos: %s", diagFormatActions(ng, tables.ActionTable[state][eosExtTok]))
			}
			if eosSemiTok >= 0 {
				t.Logf("  on semicolon: %s", diagFormatActions(ng, tables.ActionTable[state][eosSemiTok]))
			}
			found++
			if found >= 12 {
				return
			}
		}
	}
	t.Fatalf("no interesting program_statement prefix states found")
}

func TestDiagFortranOperatorNameAttribution(t *testing.T) {
	gram := loadDiagFortranGrammar(t)
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	addNonterminalExtraChains(tables, ng, ctx)

	opIDs := diagFindAllSymbols(ng, "operator_name")
	if len(opIDs) != 1 {
		t.Fatalf("operator_name symbol ids = %v", opIDs)
	}
	opID := opIDs[0]
	programID := -1
	for _, id := range diagFindAllSymbols(ng, "program") {
		if ng.Symbols[id].Kind == SymbolTerminal || ng.Symbols[id].Kind == SymbolNamedToken {
			programID = id
			break
		}
	}
	if programID < 0 {
		t.Fatal("program terminal symbol not found")
	}

	immediateTokens := make(map[int]bool)
	for _, term := range ng.Terminals {
		if term.Immediate {
			immediateTokens[term.SymbolID] = true
		}
	}
	keywordSet := make(map[int]bool, len(ng.KeywordSymbols))
	for _, ks := range ng.KeywordSymbols {
		keywordSet[ks] = true
	}
	followTokens := buildFollowTokensFunc(tables, ng.TokenCount())
	unsafeFollow := followUnsafePatternTokenSet(ng)
	lexModes, stateToMode, _ := computeLexModes(
		tables.StateCount,
		ng.TokenCount(),
		func(state, sym int) bool {
			if acts, ok := tables.ActionTable[state]; ok {
				if entry, ok := acts[sym]; ok && len(entry) > 0 {
					return true
				}
			}
			return false
		},
		computeStringPrefixExtensions(ng.Terminals),
		ng.ExtraSymbols,
		tables.ExtraChainStateStart,
		immediateTokens,
		ng.ExternalSymbols,
		ng.WordSymbolID,
		keywordSet,
		terminalPatternSymSet(ng),
		followTokens,
		patternImmediateTokenSet(ng),
		unsafeFollow,
	)
	acts := tables.ActionTable[0][programID]
	oldState := -1
	for _, act := range acts {
		if act.kind == lrShift {
			oldState = act.state
			break
		}
	}
	if oldState < 0 {
		t.Fatalf("initial state on program = %s", diagFormatActions(ng, acts))
	}
	if oldState >= len(stateToMode) {
		t.Fatalf("old state %d out of range (len=%d)", oldState, len(stateToMode))
	}
	modeIdx := stateToMode[oldState]
	mode := lexModes[modeIdx]

	t.Logf("operator_name id=%d unsafeFollow=%v", opID, unsafeFollow[opID])
	t.Logf("program shift target oldState=%d runtimeState=%d modeIdx=%d operatorAction=%v operatorFollow=%v modeHasOperator=%v",
		oldState, oldState+1, modeIdx, len(tables.ActionTable[oldState][opID]) > 0, containsIntFortran(followTokens(oldState), opID), mode.validSymbols[opID])
}

func containsIntFortran(xs []int, want int) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
