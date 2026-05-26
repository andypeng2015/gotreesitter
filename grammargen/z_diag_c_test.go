package grammargen

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestDiagCFailingSamples(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	refLang := grammars.CLanguage()
	adaptExternalScanner(refLang, genLang)

	for _, path := range []string{
		"/tmp/grammar_parity_c_clone/examples/cluster.c",
		"/tmp/grammar_parity_c_clone/examples/malloc.c",
		"/tmp/grammar_parity_c_clone/examples/parser.c",
	} {
		t.Run(path, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read sample: %v", err)
			}
			genTree, err := gotreesitter.NewParser(genLang).Parse(src)
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			refTree, err := gotreesitter.NewParser(refLang).Parse(src)
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}

			root := genTree.RootNode()
			t.Logf("gen root=%s err=%v runtime=%s", root.Type(genLang), root.HasError(), genTree.ParseRuntime().Summary())
			t.Logf("ref root=%s err=%v", refTree.RootNode().Type(refLang), refTree.RootNode().HasError())
			t.Logf("gen sexpr: %.500s", root.SExpr(genLang))
			t.Logf("ref sexpr: %.500s", refTree.RootNode().SExpr(refLang))
			t.Logf("rootLooksLikeCTopLevel=%v", diagRootLooksLikeCTopLevel(root, genLang))
			if badIdx, badType := firstUnexpectedCTopLevelChild(root, genLang); badIdx >= 0 {
				t.Logf("first unexpected top-level child: %d:%s", badIdx, badType)
				if bad := root.Child(badIdx); bad != nil {
					start := int(bad.StartByte())
					end := int(bad.EndByte())
					ctxStart := start - 120
					if ctxStart < 0 {
						ctxStart = 0
					}
					ctxEnd := end + 120
					if ctxEnd > len(src) {
						ctxEnd = len(src)
					}
					t.Logf("bad child sexpr: %.500s", bad.SExpr(genLang))
					t.Logf("bad child context: %q", string(src[ctxStart:ctxEnd]))
					winStart := badIdx - 2
					if winStart < 0 {
						winStart = 0
					}
					winEnd := badIdx + 15
					if winEnd >= root.ChildCount() {
						winEnd = root.ChildCount() - 1
					}
					for j := winStart; j <= winEnd; j++ {
						sib := root.Child(j)
						if sib == nil {
							continue
						}
						t.Logf("sibling[%d]=%s [%d:%d] err=%v sexpr=%.240s", j, sib.Type(genLang), sib.StartByte(), sib.EndByte(), sib.HasError() || sib.IsError(), sib.SExpr(genLang))
					}
				}
			} else {
				t.Log("all top-level children satisfy C root heuristic")
			}

			var childSummaries []string
			for i := 0; i < root.ChildCount() && i < 20; i++ {
				child := root.Child(i)
				if child == nil {
					continue
				}
				childSummaries = append(childSummaries, fmt.Sprintf("%d:%s[%d:%d] err=%v", i, child.Type(genLang), child.StartByte(), child.EndByte(), child.HasError() || child.IsError()))
			}
			t.Logf("root children: %s", strings.Join(childSummaries, " | "))

			firstErr := firstErrorNode(root)
			if firstErr == nil {
				t.Log("no explicit error node")
				return
			}
			start := int(firstErr.StartByte())
			end := int(firstErr.EndByte())
			if start < 0 {
				start = 0
			}
			if end < start {
				end = start
			}
			ctxStart := start - 120
			if ctxStart < 0 {
				ctxStart = 0
			}
			ctxEnd := end + 120
			if ctxEnd > len(src) {
				ctxEnd = len(src)
			}
			t.Logf("first error=%s [%d:%d]", firstErr.Type(genLang), start, end)
			t.Logf("error context: %q", string(src[ctxStart:ctxEnd]))
			t.Logf("error sexpr: %.500s", firstErr.SExpr(genLang))
			if parent := firstErr.Parent(); parent != nil {
				t.Logf("error parent: %.500s", parent.SExpr(genLang))
			}
		})
	}
}

func TestDiagCGNUAsmMiniSamples(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	refLang := grammars.CLanguage()
	adaptExternalScanner(refLang, genLang)

	samples := []string{
		"asm volatile (\"addq %2,%0; adcq %3,%1\" : \"=r\"(x), [y] \"=r\"((uintptr_t) y) : \"r\"(z) : \"r2\");\n",
		"int main() { asm(\"addq %2,%0; adcq %3,%1\" : \"=r\"(x), [y] \"=r\"((uintptr_t) y) : \"r\"(z) : \"r2\"); }\n",
		"asm volatile (\n    \"mov r0, %0\\n\"\n    \"mov r1, %[y]\\n\"\n    \"add r2, r0, r1\\n\"\n    \"mov %1, r2\\n\"\n    :     \"r\"  (z)\n    :     \"=r\" (x),\n      [y] \"=r\" ((uintptr_t) y)\n    : \"r2\");\n",
		"int main() {\n  int var;\n  __asm__(\n    \"nop;\"\n    : [var] \"=r\"(var)\n    :\n    : \"eax\", \"ra\" \"x\"\n  );\n\n  asm(\"addq %2,%0; adcq %3,%1\"\n      : \"+m\"(rp[0]), \"+d\"(high)\n      : \"r\"(c1), \"g\"(0)\n      : \"cc\"\n  );\n}\n",
	}
	for _, sample := range samples {
		t.Run(sample, func(t *testing.T) {
			genTree, err := gotreesitter.NewParser(genLang).Parse([]byte(sample))
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			refTree, err := gotreesitter.NewParser(refLang).Parse([]byte(sample))
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}
			t.Logf("runtime=%s", genTree.ParseRuntime().Summary())
			t.Logf("gen=%s", genTree.RootNode().SExpr(genLang))
			t.Logf("ref=%s", refTree.RootNode().SExpr(refLang))
		})
	}
}

func TestDiagCStringWrapperPrecedence(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}

	var targetProd int = -1
	for i, p := range ng.Productions {
		if ng.Symbols[p.LHS].Name != "_string" || len(p.RHS) != 1 {
			continue
		}
		rhsName := ng.Symbols[p.RHS[0]].Name
		t.Logf("prod[%d] %s prec=%d explicit=%v assoc=%v", i, diagFormatProd(ng, i, len(p.RHS)), p.Prec, p.HasExplicitPrec, p.Assoc)
		if rhsName == "concatenated_string" {
			targetProd = i
		}
	}
	if targetProd < 0 {
		t.Fatal("missing _string -> concatenated_string production")
	}

	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	matched := 0
	for state, itemSet := range ctx.itemSets {
		hasTarget := false
		for _, ce := range itemSet.cores {
			if ce.prodIdx == targetProd {
				hasTarget = true
				break
			}
		}
		if !hasTarget {
			continue
		}
		matched++
		t.Logf("state %d actions=%s", state, diagFormatStateActions(ng, tables, state))
		for _, ce := range itemSet.cores {
			if ce.prodIdx == targetProd {
				t.Logf("  item%s %s", diagFormatLookaheads(ng, &ce.lookaheads), diagFormatProd(ng, ce.prodIdx, ce.dot))
			}
		}
		if matched >= 8 {
			break
		}
	}
}

func TestDiagCRecoveredFunctionCandidates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	src, err := os.ReadFile("/tmp/grammar_parity_c_clone/examples/cluster.c")
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	tree, err := gotreesitter.NewParser(genLang).Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	root := tree.RootNode()
	t.Logf("root=%s err=%v runtime=%s", root.Type(genLang), root.HasError(), tree.ParseRuntime().Summary())
	for i := 0; i+2 < root.ChildCount(); i++ {
		head := root.Child(i)
		decl := root.Child(i + 1)
		open := root.Child(i + 2)
		if !diagHasRecoveredFunctionHead(head, genLang) {
			continue
		}
		if decl == nil || decl.Type(genLang) != "function_declarator" {
			continue
		}
		if open == nil || open.Type(genLang) != "{" {
			continue
		}
		closeIdx := diagFindRecoveredFunctionClose(root, i+2, genLang)
		t.Logf("candidate at %d: head=%s decl=%s open=%s closeIdx=%d", i, head.Type(genLang), decl.Type(genLang), open.Type(genLang), closeIdx)
		for j := i; j < root.ChildCount() && j < i+20; j++ {
			child := root.Child(j)
			if child == nil {
				continue
			}
			t.Logf("  child[%d]=%s [%d:%d] err=%v", j, child.Type(genLang), child.StartByte(), child.EndByte(), child.HasError() || child.IsError())
		}
		if closeIdx >= 0 {
			break
		}
	}
}

func TestDiagCGNUAsmColonStates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("build tables: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate language: %v", err)
	}
	colonSym := -1
	for i, s := range ng.Symbols {
		if s.Name == ":" {
			colonSym = i
			break
		}
	}
	if colonSym < 0 {
		t.Fatal("missing colon symbol")
	}
	asmStates := 0
	wrapperStates := 0
	for state, itemSet := range ctx.itemSets {
		var sawAsmAfterString bool
		var sawStringWrapperReduce bool
		var sawConcatReduce bool
		for _, ce := range itemSet.cores {
			p := ng.Productions[ce.prodIdx]
			lhs := ng.Symbols[p.LHS].Name
			if lhs == "gnu_asm_expression" && ce.dot > 0 && ce.dot <= len(p.RHS) && ng.Symbols[p.RHS[ce.dot-1]].Name == "_string" {
				sawAsmAfterString = true
			}
			if lhs == "_string" && ce.dot == len(p.RHS) && len(p.RHS) == 1 && ng.Symbols[p.RHS[0]].Name == "concatenated_string" {
				sawStringWrapperReduce = true
			}
			if lhs == "concatenated_string" && ce.dot == len(p.RHS) {
				sawConcatReduce = true
			}
		}
		if sawAsmAfterString {
			asmStates++
			t.Logf("asm-after-string state %d actions=%s", state, diagFormatStateActions(ng, tables, state))
			for _, ce := range itemSet.cores {
				p := ng.Productions[ce.prodIdx]
				lhs := ng.Symbols[p.LHS].Name
				if lhs == "gnu_asm_expression" {
					t.Logf("  item%s %s", diagFormatLookaheads(ng, &ce.lookaheads), diagFormatProd(ng, ce.prodIdx, ce.dot))
				}
			}
			if acts, ok := tables.ActionTable[state][colonSym]; ok {
				t.Logf("  colon actions: %s", diagFormatActions(ng, acts))
			} else {
				t.Log("  colon actions: <none>")
			}
			t.Logf("  final actions(remapped=%d): %s", state+1, diagFormatLangStateActions(genLang, state+1))
		}
		if sawStringWrapperReduce || sawConcatReduce {
			wrapperStates++
			t.Logf("string-wrapper state %d actions=%s", state, diagFormatStateActions(ng, tables, state))
			for _, ce := range itemSet.cores {
				p := ng.Productions[ce.prodIdx]
				lhs := ng.Symbols[p.LHS].Name
				if lhs == "_string" || lhs == "concatenated_string" {
					t.Logf("  item%s %s", diagFormatLookaheads(ng, &ce.lookaheads), diagFormatProd(ng, ce.prodIdx, ce.dot))
				}
			}
			if acts, ok := tables.ActionTable[state][colonSym]; ok {
				t.Logf("  colon actions: %s", diagFormatActions(ng, acts))
			} else {
				t.Log("  colon actions: <none>")
			}
			t.Logf("  final actions(remapped=%d): %s", state+1, diagFormatLangStateActions(genLang, state+1))
		}
	}
	t.Logf("asmStates=%d wrapperStates=%d", asmStates, wrapperStates)
	for _, st := range []int{1175, 3647, 3974} {
		t.Logf("selected raw state %d actions=%s", st, diagFormatStateActions(ng, tables, st))
		t.Logf("selected final state %d actions=%s", st+1, diagFormatLangStateActions(genLang, st+1))
	}
}

func TestDiagCGNUAsmRuntimeTrace(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate language: %v", err)
	}
	src := []byte("asm volatile (\n    \"mov r0, %0\\n\"\n    \"mov r1, %[y]\\n\"\n    \"add r2, r0, r1\\n\"\n    \"mov %1, r2\\n\"\n    :     \"r\"  (z)\n    :     \"=r\" (x),\n      [y] \"=r\" ((uintptr_t) y)\n    : \"r2\");\n")
	parser := gotreesitter.NewParser(genLang)
	parser.SetGLRTrace(true)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("runtime=%s", tree.ParseRuntime().Summary())
	t.Logf("sexpr=%.1200s", tree.RootNode().SExpr(genLang))
}

func diagHasRecoveredFunctionHead(node *gotreesitter.Node, lang *gotreesitter.Language) bool {
	if node == nil || lang == nil {
		return false
	}
	if diagIsRecoveredFunctionHeadSpecifier(node, lang) {
		return true
	}
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Type(lang) == "_declaration_specifiers" {
			for j := 0; j < child.ChildCount(); j++ {
				if diagIsRecoveredFunctionHeadSpecifier(child.Child(j), lang) {
					return true
				}
			}
			continue
		}
		if diagIsRecoveredFunctionHeadSpecifier(child, lang) {
			return true
		}
	}
	return false
}

func diagIsRecoveredFunctionHeadSpecifier(node *gotreesitter.Node, lang *gotreesitter.Language) bool {
	if node == nil || lang == nil {
		return false
	}
	switch node.Type(lang) {
	case "primitive_type", "storage_class_specifier", "type_qualifier", "sized_type_specifier", "type_identifier", "struct_specifier", "union_specifier", "enum_specifier", "attribute_specifier", "attribute_declaration", "ms_declspec_modifier":
		return true
	default:
		return false
	}
}

func diagFindRecoveredFunctionClose(root *gotreesitter.Node, openIdx int, lang *gotreesitter.Language) int {
	depth := 0
	for i := openIdx; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func TestDiagCGNUAsmExactTrace(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	src := []byte("asm volatile (\n    \"mov r0, %0\\n\"\n    \"mov r1, %[y]\\n\"\n    \"add r2, r0, r1\\n\"\n    \"mov %1, r2\\n\"\n    :     \"r\"  (z)\n    :     \"=r\" (x),\n      [y] \"=r\" ((uintptr_t) y)\n    : \"r2\");\n")
	parser := gotreesitter.NewParser(genLang)
	parser.SetGLRTrace(true)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("runtime=%s", tree.ParseRuntime().Summary())
	t.Logf("sexpr=%s", tree.RootNode().SExpr(genLang))
}

func TestDiagCGNUAsmStates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	for _, state := range []int{1486, 3647, 1175, 3974, 4871, 169} {
		if state >= len(ctx.itemSets) {
			t.Fatalf("state %d out of range", state)
		}
		t.Logf("state %d actions=%s", state, diagFormatStateActions(ng, tables, state))
		for _, ce := range ctx.itemSets[state].cores {
			t.Logf("  item%s %s", diagFormatLookaheads(ng, &ce.lookaheads), diagFormatProd(ng, ce.prodIdx, ce.dot))
		}
	}
}

func TestDiagCImmediateNewlineStates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	const sym = 147
	found := 0
	for state, acts := range tables.ActionTable {
		entry := acts[sym]
		if len(entry) == 0 {
			continue
		}
		t.Logf("state %d on 147=%s", state, diagFormatActions(ng, entry))
		for _, ce := range ctx.itemSets[state].cores {
			t.Logf("  item%s %s", diagFormatLookaheads(ng, &ce.lookaheads), diagFormatProd(ng, ce.prodIdx, ce.dot))
		}
		found++
		if found >= 6 {
			break
		}
	}
	if found == 0 {
		t.Fatal("no states found with action on _preproc_include_token1")
	}
}

func TestDiagCGNUAsmLexModes(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("buildLRTablesWithProvenance: %v", err)
	}
	addNonterminalExtraChains(tables, ng, ctx)
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
		func() map[int]bool {
			m := make(map[int]bool)
			for _, t := range ng.Terminals {
				if t.Immediate {
					m[t.SymbolID] = true
				}
			}
			return m
		}(),
		ng.ExternalSymbols,
		ng.WordSymbolID,
		func() map[int]bool {
			m := make(map[int]bool, len(ng.KeywordSymbols))
			for _, ks := range ng.KeywordSymbols {
				m[ks] = true
			}
			return m
		}(),
		terminalPatternSymSet(ng),
		buildFollowTokensFunc(tables, ng.TokenCount()),
		patternImmediateTokenSet(ng),
		followUnsafePatternTokenSet(ng),
	)

	src := []byte("asm volatile (\n    \"mov r0, %0\\n\"\n    \"mov r1, %[y]\\n\"\n    \"add r2, r0, r1\\n\"\n    \"mov %1, r2\\n\"\n    :     \"r\"  (z)\n    :     \"=r\" (x),\n      [y] \"=r\" ((uintptr_t) y)\n    : \"r2\");\n")
	for _, state := range []int{1486, 3647, 1175, 3974} {
		if state >= len(genLang.LexModes) {
			t.Fatalf("runtime state %d out of range", state)
		}
		if state < len(stateToMode) {
			modeIdx := stateToMode[state]
			valid147 := modeIdx >= 0 && modeIdx < len(lexModes) && lexModes[modeIdx].validSymbols[147]
			t.Logf("runtime state=%d mode=%d valid147=%v", state, modeIdx, valid147)
		}
		lexState := int(genLang.LexModes[state].LexState)
		t.Logf("runtime state=%d lexState=%d", state, lexState)
		for _, probe := range []int{96, 101, 107, 120} {
			if probe > len(src) {
				continue
			}
			lexer := gotreesitter.NewLexer(genLang.LexStates, src[probe:])
			tok := lexer.Next(uint16(lexState))
			name := ""
			if int(tok.Symbol) < len(genLang.SymbolNames) {
				name = genLang.SymbolNames[tok.Symbol]
			}
			t.Logf("  probe=%d tok=%d %q span=[%d:%d] text=%q", probe, tok.Symbol, name, probe+int(tok.StartByte), probe+int(tok.EndByte), string(src[probe+int(tok.StartByte):probe+int(tok.EndByte)]))
			diagTraceLexScan(t, "c-lex", genLang.LexStates, genLang.SymbolNames, lexState, src[probe:])
		}
	}
}

func TestDiagCPointerAssignmentShape(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity_c_clone/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	refLang := grammars.CLanguage()
	adaptExternalScanner(refLang, genLang)

	src := []byte("void f() { *lookahead = reusable_node->tree; *is_fresh = true; *count = 0; }\n")
	genTree, err := gotreesitter.NewParser(genLang).Parse(src)
	if err != nil {
		t.Fatalf("gen parse: %v", err)
	}
	refTree, err := gotreesitter.NewParser(refLang).Parse(src)
	if err != nil {
		t.Fatalf("ref parse: %v", err)
	}
	t.Logf("gen=%s", genTree.RootNode().SExpr(genLang))
	t.Logf("ref=%s", refTree.RootNode().SExpr(refLang))
	var genExpr *gotreesitter.Node
	var walk func(*gotreesitter.Node)
	walk = func(n *gotreesitter.Node) {
		if n == nil || genExpr != nil {
			return
		}
		if n.Type(genLang) == "pointer_expression" {
			genExpr = n
			return
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(genTree.RootNode())
	if genExpr == nil {
		t.Fatal("no pointer_expression found")
	}
	t.Logf("gen expr type=%s childCount=%d", genExpr.Type(genLang), genExpr.ChildCount())
	for i := 0; i < genExpr.ChildCount(); i++ {
		child := genExpr.Child(i)
		if child == nil {
			continue
		}
		t.Logf("  gen child[%d]=%s childCount=%d sexpr=%s", i, child.Type(genLang), child.ChildCount(), child.SExpr(genLang))
	}
}

func firstErrorNode(root *gotreesitter.Node) *gotreesitter.Node {
	if root == nil {
		return nil
	}
	var best *gotreesitter.Node
	var walk func(*gotreesitter.Node)
	walk = func(n *gotreesitter.Node) {
		if n == nil {
			return
		}
		if n.IsError() && (best == nil || n.StartByte() < best.StartByte() || (n.StartByte() == best.StartByte() && n.EndByte() < best.EndByte())) {
			best = n
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return best
}

func firstUnexpectedCTopLevelChild(root *gotreesitter.Node, lang *gotreesitter.Language) (int, string) {
	if root == nil || lang == nil {
		return -1, ""
	}
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "preproc_if",
			"preproc_ifdef",
			"preproc_include",
			"preproc_def",
			"preproc_function_def",
			"preproc_call",
			"declaration",
			"expression_statement",
			"function_definition",
			"linkage_specification",
			"type_definition",
			"struct_specifier",
			"union_specifier",
			"enum_specifier",
			"class_specifier",
			"namespace_definition",
			"template_declaration",
			"comment",
			";":
			continue
		default:
			return i, child.Type(lang)
		}
	}
	return -1, ""
}

func diagRootLooksLikeCTopLevel(root *gotreesitter.Node, lang *gotreesitter.Language) bool {
	if root == nil || lang == nil || root.ChildCount() == 0 {
		return false
	}
	sawTopLevel := false
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		switch child.Type(lang) {
		case "preproc_if",
			"preproc_ifdef",
			"preproc_include",
			"preproc_def",
			"preproc_function_def",
			"preproc_call",
			"declaration",
			"expression_statement",
			"function_definition",
			"linkage_specification",
			"type_definition",
			"struct_specifier",
			"union_specifier",
			"enum_specifier",
			"class_specifier",
			"namespace_definition",
			"template_declaration",
			"comment",
			";":
			sawTopLevel = true
		default:
			return false
		}
	}
	return sawTopLevel
}
