package grammargen

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestDiagJSXAndTSLookupTypes(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		ref      func() *gotreesitter.Language
		samples  []string
	}{
		{
			name:     "javascript",
			jsonPath: "/tmp/grammar_parity/javascript/src/grammar.json",
			ref:      grammars.JavascriptLanguage,
			samples: []string{
				"var a = <Foo></Foo>\n",
				"b = <Foo.Bar></Foo.Bar>\n",
				"c = <> <Foo /> </>\n",
				"d = <Bar> <Foo /> </Bar>\n",
				"e = <Foo bar/>\n",
				"f = <Foo bar=\"string\" baz={2} data-i8n=\"dialogs.welcome.heading\" bam />\n",
				"g = <Avatar userId={foo.creatorId} />\n",
				"h = <input checked={this.state.selectedNewStreetType === 'new-street-default' || !this.state.selectedNewStreetType}> </input>\n",
				"i = <Foo:Bar bar={}>{...children}</Foo:Bar>\n",
				"var a = <Foo></Foo>\n" +
					"b = <Foo.Bar></Foo.Bar>\n" +
					"c = <> <Foo /> </>\n" +
					"d = <Bar> <Foo /> </Bar>\n" +
					"e = <Foo bar/>\n" +
					"f = <Foo bar=\"string\" baz={2} data-i8n=\"dialogs.welcome.heading\" bam />\n" +
					"g = <Avatar userId={foo.creatorId} />\n" +
					"h = <input checked={this.state.selectedNewStreetType === 'new-street-default' || !this.state.selectedNewStreetType}> </input>\n" +
					"i = <Foo:Bar bar={}>{...children}</Foo:Bar>\n",
			},
		},
		{
			name:     "typescript",
			jsonPath: "/tmp/grammar_parity/typescript/typescript/src/grammar.json",
			ref:      grammars.TypescriptLanguage,
			samples: []string{
				"type X1 = typeof Y[keyof typeof Z];\n",
				"type X2 = (typeof Y)[keyof typeof Z];\n",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.jsonPath)
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
			refLang := tc.ref()
			adaptExternalScanner(refLang, genLang)

			genParser := gotreesitter.NewParser(genLang)
			refParser := gotreesitter.NewParser(refLang)
			for _, sample := range tc.samples {
				var lexLogs []string
				genParser.SetLogger(func(kind gotreesitter.ParserLogType, message string) {
					if kind != gotreesitter.ParserLogLex {
						return
					}
					var symID int
					if _, err := fmt.Sscanf(message, "token sym=%d", &symID); err != nil {
						lexLogs = append(lexLogs, message)
						return
					}
					name := strconv.Itoa(symID)
					if symID >= 0 && symID < len(genLang.SymbolNames) {
						name = genLang.SymbolNames[symID]
					}
					lexLogs = append(lexLogs, fmt.Sprintf("%s :: %s", name, message))
				})
				genTree, err := genParser.Parse([]byte(sample))
				if err != nil {
					t.Fatalf("gen parse %q: %v", sample, err)
				}
				genParser.SetLogger(nil)
				refTree, err := refParser.Parse([]byte(sample))
				if err != nil {
					t.Fatalf("ref parse %q: %v", sample, err)
				}
				genS := genTree.RootNode().SExpr(genLang)
				refS := refTree.RootNode().SExpr(refLang)
				t.Logf("sample: %q", strings.TrimSpace(sample))
				t.Logf("gen: %s", genS)
				t.Logf("ref: %s", refS)
				if genTree.RootNode().HasError() || refTree.RootNode().HasError() {
					for _, line := range lexLogs {
						t.Logf("lex: %s", line)
					}
					t.Logf("runtime: %s", genTree.ParseRuntime().Summary())
				}
			}
		})
	}
}

func TestDiagJSXTrace(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/javascript/src/grammar.json")
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
	refLang := grammars.JavascriptLanguage()
	adaptExternalScanner(refLang, genLang)
	parser := gotreesitter.NewParser(genLang)
	parser.SetGLRTrace(true)
	src := []byte("g = <Avatar userId={foo.creatorId} />\n")
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("sexpr: %s", tree.RootNode().SExpr(genLang))
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
}

func TestDiagJSXLineBreakTrace(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/javascript/src/grammar.json")
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
	refLang := grammars.JavascriptLanguage()
	adaptExternalScanner(refLang, genLang)
	parser := gotreesitter.NewParser(genLang)
	parser.SetGLRTrace(true)
	src := []byte("var a = <Foo></Foo>\nb = <Foo.Bar></Foo.Bar>\n")
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("sexpr: %s", tree.RootNode().SExpr(genLang))
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
}

func TestDiagTSLookupTrace(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)
	parser := gotreesitter.NewParser(genLang)
	parser.SetGLRTrace(true)
	src := []byte("type X2 = (typeof Y)[keyof typeof Z];\n")
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("sexpr: %s", tree.RootNode().SExpr(genLang))
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
}

func TestDiagJSTSStateFacts(t *testing.T) {
	tests := []struct {
		name      string
		jsonPath  string
		states    []int
		prodIndex []int
	}{
		{
			name:      "javascript",
			jsonPath:  "/tmp/grammar_parity/javascript/src/grammar.json",
			states:    []int{836, 916, 1717},
			prodIndex: []int{1023},
		},
		{
			name:      "typescript",
			jsonPath:  "/tmp/grammar_parity/typescript/typescript/src/grammar.json",
			states:    []int{425, 313, 249},
			prodIndex: []int{531},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.jsonPath)
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
			for _, st := range tc.states {
				if st >= len(ctx.itemSets) {
					t.Fatalf("state %d out of range", st)
				}
				merged := ctx.provenance != nil && ctx.provenance.isMerged(st)
				mergeCount := 0
				if ctx.provenance != nil {
					mergeCount = len(ctx.provenance.origins(st))
				}
				t.Logf("state %d merged=%v mergeCount=%d actions=%s",
					st, merged, mergeCount, diagFormatStateActions(ng, tables, st))
			}
			for _, pi := range tc.prodIndex {
				if pi >= len(ng.Productions) {
					t.Fatalf("prod %d out of range", pi)
				}
				p := ng.Productions[pi]
				t.Logf("prod %d: %s prec=%d explicit=%v assoc=%v", pi, diagFormatProd(ng, pi, len(p.RHS)), p.Prec, p.HasExplicitPrec, p.Assoc)
			}
			genLang, err := generateWithTimeout(gram, 90*time.Second)
			if err != nil {
				t.Fatalf("generate language: %v", err)
			}
			for _, st := range tc.states {
				t.Logf("final state %d externalLexState=%d actions=%s", st, genLang.LexModes[st].ExternalLexState, diagFormatLangStateActions(genLang, st))
			}
		})
	}
}

func TestDiagJSXRuntimeState61(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/javascript/src/grammar.json")
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
	if 61 >= len(ctx.itemSets) {
		t.Fatalf("state 61 out of range")
	}
	t.Logf("lr state 61 actions=%s", diagFormatStateActions(ng, tables, 61))
	for _, ce := range ctx.itemSets[61].cores {
		t.Logf("core: %s", diagFormatProd(ng, ce.prodIdx, ce.dot))
	}
	if 538 < len(ng.Productions) {
		p := ng.Productions[538]
		t.Logf("prod 538: %s prec=%d explicit=%v assoc=%v", diagFormatProd(ng, 538, len(p.RHS)), p.Prec, p.HasExplicitPrec, p.Assoc)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate language: %v", err)
	}
	t.Logf("final state 61 externalLexState=%d actions=%s", genLang.LexModes[61].ExternalLexState, diagFormatLangStateActions(genLang, 61))
	if 179 < len(genLang.SymbolNames) {
		t.Logf("runtime sym 179=%s", genLang.SymbolNames[179])
	}
}

func TestDiagJSXPostClosingElementStates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/javascript/src/grammar.json")
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
	for _, st := range []int{1694, 344, 61} {
		if st >= len(genLang.LexModes) {
			t.Fatalf("state %d out of range", st)
		}
		els := genLang.LexModes[st].ExternalLexState
		t.Logf("final state %d externalLexState=%d actions=%s", st, els, diagFormatLangStateActions(genLang, st))
		if int(els) < len(genLang.ExternalLexStates) {
			var names []string
			for i, ok := range genLang.ExternalLexStates[els] {
				if !ok {
					continue
				}
				name := "?"
				if i < len(genLang.ExternalSymbols) && int(genLang.ExternalSymbols[i]) < len(genLang.SymbolNames) {
					name = genLang.SymbolNames[genLang.ExternalSymbols[i]]
				}
				names = append(names, fmt.Sprintf("%d:%s", i, name))
			}
			t.Logf("external row %d = [%s]", els, strings.Join(names, ", "))
		}
	}
}

func TestDiagJSXAttributeAliasProductions(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/javascript/src/grammar.json")
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
	targets := map[string]bool{
		"jsx_attribute":       true,
		"_jsx_attribute_name": true,
		"_jsx_identifier":     true,
	}
	for i, p := range ng.Productions {
		lhs := ng.Symbols[p.LHS].Name
		if !targets[lhs] {
			continue
		}
		t.Logf("prod %d pid=%d: %s aliases=%v", i, p.ProductionID, diagFormatProd(ng, i, len(p.RHS)), p.Aliases)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate language: %v", err)
	}
	for _, prodIdx := range []int{1018, 1019, 1023} {
		if prodIdx < 0 || prodIdx >= len(ng.Productions) {
			continue
		}
		pid := ng.Productions[prodIdx].ProductionID
		if pid < 0 || pid >= len(genLang.AliasSequences) {
			t.Logf("prod %d pid=%d alias seq: out of range", prodIdx, pid)
			continue
		}
		seq := genLang.AliasSequences[pid]
		var names []string
		for i, sym := range seq {
			if sym == 0 {
				continue
			}
			name := "?"
			if int(sym) < len(genLang.SymbolNames) {
				name = genLang.SymbolNames[sym]
			}
			names = append(names, fmt.Sprintf("%d:%s", i, name))
		}
		t.Logf("prod %d pid=%d alias seq [%s]", prodIdx, pid, strings.Join(names, ", "))
	}
	for _, sym := range []int{117, 127} {
		if sym < 0 || sym >= len(ng.Symbols) {
			continue
		}
		si := ng.Symbols[sym]
		t.Logf("sym %d: name=%q kind=%v visible=%v named=%v extra=%v", sym, si.Name, si.Kind, si.Visible, si.Named, si.IsExtra)
	}
}

func TestDiagTSParenedTypeStates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	for _, st := range []int{1185, 569, 770, 1633, 425, 249, 313} {
		if st >= len(ctx.itemSets) {
			t.Fatalf("state %d out of range", st)
		}
		merged := ctx.provenance != nil && ctx.provenance.isMerged(st)
		mergeCount := 0
		if ctx.provenance != nil {
			mergeCount = len(ctx.provenance.origins(st))
		}
		t.Logf("state %d merged=%v mergeCount=%d actions=%s", st, merged, mergeCount, diagFormatStateActions(ng, tables, st))
	}
}

func TestDiagTSParenedTypeRuntimeStates(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	for _, st := range []int{1185, 569, 770, 1633, 425, 249, 313} {
		if st >= len(genLang.LexModes) {
			t.Fatalf("state %d out of range", st)
		}
		t.Logf("final state %d externalLexState=%d actions=%s", st, genLang.LexModes[st].ExternalLexState, diagFormatLangStateActions(genLang, st))
	}
}

func TestDiagTSLookupTraceWithSplit(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	gram.EnableLRSplitting = true
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)
	parser := gotreesitter.NewParser(genLang)
	parser.SetGLRTrace(true)
	src := []byte("type X2 = (typeof Y)[keyof typeof Z];\n")
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("sexpr: %s", tree.RootNode().SExpr(genLang))
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
}

func TestDiagMiniTSParenedTypeQuery(t *testing.T) {
	g := NewGrammar("mini_ts_type_query")
	g.Rules["program"] = Choice(Sym("type_alias_declaration"), Sym("expression_statement"))
	g.Rules["type_alias_declaration"] = Seq(Str("type"), Sym("identifier"), Str("="), Sym("type"), Str(";"))
	g.Rules["expression_statement"] = Seq(Sym("expression"), Str(";"))
	g.Rules["type"] = Choice(Sym("lookup_type"), Sym("parenthesized_type"), Sym("type_query"), Sym("identifier"), Sym("index_type_query"))
	g.Rules["primary_type"] = Choice(Sym("parenthesized_type"), Sym("type_query"), Sym("identifier"))
	g.Rules["parenthesized_type"] = Seq(Str("("), Sym("type"), Str(")"))
	g.Rules["lookup_type"] = Seq(Sym("primary_type"), Str("["), Sym("type"), Str("]"))
	g.Rules["type_query"] = PrecRight(0, Seq(Str("typeof"), Sym("identifier")))
	g.Rules["index_type_query"] = Seq(Str("keyof"), Sym("type_query"))
	g.Rules["expression"] = Choice(Sym("parenthesized_expression"), Sym("unary_expression"), Sym("subscript_expression"), Sym("primary_expression"))
	g.Rules["parenthesized_expression"] = Seq(Str("("), Sym("expression"), Str(")"))
	g.Rules["unary_expression"] = Seq(Str("typeof"), Sym("primary_expression"))
	g.Rules["subscript_expression"] = Seq(Sym("primary_expression"), Str("["), Sym("expression"), Str("]"))
	g.Rules["primary_expression"] = Sym("identifier")
	g.Rules["identifier"] = Pat("[A-Za-z_][A-Za-z0-9_]*")
	g.RuleOrder = []string{
		"program",
		"type_alias_declaration",
		"expression_statement",
		"type",
		"primary_type",
		"parenthesized_type",
		"lookup_type",
		"type_query",
		"index_type_query",
		"expression",
		"parenthesized_expression",
		"unary_expression",
		"subscript_expression",
		"primary_expression",
		"identifier",
	}

	lang, err := GenerateLanguage(g)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	parser := gotreesitter.NewParser(lang)
	src := []byte("type X = (typeof Y)[keyof typeof Z];")
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("sexpr: %s", tree.RootNode().SExpr(lang))
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
}

func TestDiagTSLookupBuildStatesFromRuntime(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	_, ctx, err := buildLRTablesWithProvenance(ng)
	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatalf("build tables: %v", err)
	}

	targetProd := map[int]bool{
		531:   true, // primary_expression -> identifier
		532:   true, // primary_expression -> property_identifier
		20709: true,
		20710: true,
		20711: true,
		20712: true,
		20713: true,
		20714: true,
		20716: true,
		20730: true,
	}
	runtimeStates := []int{1185, 569, 770, 1633, 425, 249, 313}
	for _, runtimeState := range runtimeStates {
		buildState := runtimeState - 1
		if buildState < 0 || buildState >= len(ctx.itemSets) {
			t.Fatalf("build state %d out of range for runtime state %d", buildState, runtimeState)
		}
		itemSet := &ctx.itemSets[buildState]
		t.Logf("runtime state %d => build state %d merged=%v mergeCount=%d",
			runtimeState, buildState, ctx.provenance.isMerged(buildState), len(ctx.provenance.origins(buildState)))
		if acts, ok := tables.ActionTable[buildState]; ok {
			for _, sym := range []int{26, 43, 44, 46} {
				if sym >= 0 && sym < len(ng.Symbols) {
					t.Logf("  actions on %s: %s", diagSymbolName(ng, sym), diagFormatActions(ng, acts[sym]))
				}
			}
		}
		for _, ce := range itemSet.cores {
			if !targetProd[ce.prodIdx] {
				continue
			}
			t.Logf("  core prod=%d dot=%d %s lookaheads=%s",
				ce.prodIdx, ce.dot, diagFormatProd(ng, ce.prodIdx, ce.dot), diagFormatLookaheads(ng, &ce.lookaheads))
		}
	}
}

func TestDiagTSGeneratedActionIndexCount(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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

	getIdx := func(state int, sym int) uint16 {
		if state < 0 {
			return 0
		}
		if state < len(genLang.ParseTable) {
			if sym < 0 || sym >= len(genLang.ParseTable[state]) {
				return 0
			}
			return genLang.ParseTable[state][sym]
		}
		smallBase := int(genLang.LargeStateCount)
		smallIdx := state - smallBase
		if smallIdx < 0 || smallIdx >= len(genLang.SmallParseTableMap) {
			return 0
		}
		offset := int(genLang.SmallParseTableMap[smallIdx])
		if offset >= len(genLang.SmallParseTable) {
			return 0
		}
		groupCount := genLang.SmallParseTable[offset]
		pos := offset + 1
		for i := uint16(0); i < groupCount; i++ {
			if pos+1 >= len(genLang.SmallParseTable) {
				return 0
			}
			sectionValue := genLang.SmallParseTable[pos]
			symbolCount := genLang.SmallParseTable[pos+1]
			pos += 2
			for j := uint16(0); j < symbolCount; j++ {
				if pos >= len(genLang.SmallParseTable) {
					return 0
				}
				if int(genLang.SmallParseTable[pos]) == sym {
					return sectionValue
				}
				pos++
			}
		}
		return 0
	}

	t.Logf("parseActions=%d stateCount=%d largeStateCount=%d", len(genLang.ParseActions), genLang.StateCount, genLang.LargeStateCount)
	for _, sym := range []int{26, 43, 44, 46} {
		idx := getIdx(1633, sym)
		var acts []string
		if int(idx) < len(genLang.ParseActions) {
			for _, act := range genLang.ParseActions[idx].Actions {
				switch act.Type {
				case gotreesitter.ParseActionShift:
					acts = append(acts, fmt.Sprintf("shift(%d)", act.State))
				case gotreesitter.ParseActionReduce:
					acts = append(acts, fmt.Sprintf("reduce(%d,%d)", act.Symbol, act.ProductionID))
				case gotreesitter.ParseActionAccept:
					acts = append(acts, "accept")
				case gotreesitter.ParseActionRecover:
					acts = append(acts, fmt.Sprintf("recover(%d)", act.State))
				}
			}
		}
		t.Logf("runtime state 1633 sym=%d idx=%d actions=%v", sym, idx, acts)
	}
}

func TestDiagTSNamespaceTrace(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)

	samples := []string{
		"namespace ts {}\n",
		"namespace ts { const enum X { A = 1, } }\n",
		"/// <reference path=\"utilities.ts\"/>\n/// <reference path=\"scanner.ts\"/>\n\nnamespace ts { const enum X { A = 1, } }\n",
		"/// <reference path=\"utilities.ts\"/>\n/// <reference path=\"scanner.ts\"/>\n\nnamespace ts {\n    const enum X { A = 1, }\n    let NodeConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n}\n",
		"/// <reference path=\"utilities.ts\"/>\n/// <reference path=\"scanner.ts\"/>\n\nnamespace ts {\n    const enum SignatureFlags {\n        None = 0,\n        Yield = 1 << 0,\n        Await = 1 << 1,\n        Type  = 1 << 2,\n        RequireCompleteParameterList = 1 << 3,\n        IgnoreMissingOpenBrace = 1 << 4,\n        JSDoc = 1 << 5,\n    }\n\n    // tslint:disable variable-name\n    let NodeConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n    let TokenConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n    let IdentifierConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n    let SourceFileConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n    // tslint:enable variable-name\n}\n",
	}
	for _, sample := range samples {
		t.Run(strings.TrimSpace(sample), func(t *testing.T) {
			genParser := gotreesitter.NewParser(genLang)
			genParser.SetGLRTrace(true)
			genTree, err := genParser.Parse([]byte(sample))
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			refParser := gotreesitter.NewParser(refLang)
			refTree, err := refParser.Parse([]byte(sample))
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}
			t.Logf("gen: %s", genTree.RootNode().SExpr(genLang))
			t.Logf("gen runtime: %s", genTree.ParseRuntime().Summary())
			t.Logf("ref: %s", refTree.RootNode().SExpr(refLang))
		})
	}
}

func TestDiagTSParserPrefixFailure(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)
	genParser := gotreesitter.NewParser(genLang)
	refParser := gotreesitter.NewParser(refLang)

	srcBytes, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	lines := strings.Split(string(srcBytes), "\n")
	if len(lines) < 2 {
		t.Fatal("parser.ts unexpectedly short")
	}

	type prefixResult struct {
		genErr  bool
		refErr  bool
		summary string
		sexpr   string
	}
	checkPrefix := func(n int, capture bool) prefixResult {
		if n < 1 {
			n = 1
		}
		if n > len(lines) {
			n = len(lines)
		}
		prefix := strings.Join(lines[:n], "\n")
		genTree, err := genParser.Parse([]byte(prefix))
		if err != nil {
			t.Fatalf("gen parse prefix %d: %v", n, err)
		}
		refTree, err := refParser.Parse([]byte(prefix))
		if err != nil {
			t.Fatalf("ref parse prefix %d: %v", n, err)
		}
		out := prefixResult{
			genErr: genTree.RootNode().HasError(),
			refErr: refTree.RootNode().HasError(),
		}
		if capture {
			out.summary = genTree.ParseRuntime().Summary()
			out.sexpr = genTree.RootNode().SExpr(genLang)
		}
		return out
	}

	hi := len(lines)
	full := checkPrefix(hi, true)
	t.Logf("full file lines=%d genErr=%v refErr=%v runtime=%s", len(lines), full.genErr, full.refErr, full.summary)
	if !full.genErr || full.refErr {
		t.Fatalf("expected full prefix to reproduce gen-only error, got genErr=%v refErr=%v", full.genErr, full.refErr)
	}

	lo := 1
	window := 1024
	checkpoints := 0
	for cur := window; cur < hi; cur += window {
		res := checkPrefix(cur, false)
		checkpoints++
		t.Logf("checkpoint line=%d genErr=%v refErr=%v", cur, res.genErr, res.refErr)
		if res.genErr && !res.refErr {
			lo = cur - window + 1
			if lo < 1 {
				lo = 1
			}
			hi = cur
			break
		}
		lo = cur + 1
	}
	t.Logf("checkpoint scans=%d narrowed range=[%d,%d]", checkpoints, lo, hi)

	for lo < hi {
		mid := lo + (hi-lo)/2
		res := checkPrefix(mid, false)
		if res.genErr && !res.refErr {
			hi = mid
		} else {
			lo = mid + 1
		}
	}

	prefixLines := lo
	lastGood := checkPrefix(prefixLines-1, true)
	firstBad := checkPrefix(prefixLines, true)
	start := prefixLines - 8
	if start < 1 {
		start = 1
	}
	t.Logf("first failing prefix lines=%d", prefixLines)
	t.Logf("last good runtime: %s", lastGood.summary)
	t.Logf("first bad runtime: %s", firstBad.summary)
	t.Logf("last good gen: %.400s", lastGood.sexpr)
	t.Logf("first bad gen: %.400s", firstBad.sexpr)
	for i := start; i <= prefixLines; i++ {
		t.Logf("%4d: %s", i, lines[i-1])
	}
}

func TestDiagTSFullParserRuntimeSummary(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	srcBytes, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	tree, err := gotreesitter.NewParser(genLang).Parse(srcBytes)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
	t.Logf("sexpr-prefix: %.400s", tree.RootNode().SExpr(genLang))
}

func TestDiagTSFirstErrorNode(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	srcBytes, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	tree, err := gotreesitter.NewParser(genLang).Parse(srcBytes)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	root := tree.RootNode()
	t.Logf("runtime: %s", tree.ParseRuntime().Summary())
	if !root.HasError() {
		t.Fatal("expected parse error")
	}

	var firstErr *gotreesitter.Node
	var firstMissing *gotreesitter.Node
	var firstHasErr *gotreesitter.Node
	var walk func(*gotreesitter.Node)
	walk = func(n *gotreesitter.Node) {
		if n == nil {
			return
		}
		if n.HasError() && n != root && (firstHasErr == nil || n.StartByte() < firstHasErr.StartByte() || (n.StartByte() == firstHasErr.StartByte() && n.EndByte() < firstHasErr.EndByte())) {
			firstHasErr = n
		}
		if n.IsError() && (firstErr == nil || n.StartByte() < firstErr.StartByte() || (n.StartByte() == firstErr.StartByte() && n.EndByte() < firstErr.EndByte())) {
			firstErr = n
		}
		if n.IsMissing() && (firstMissing == nil || n.StartByte() < firstMissing.StartByte() || (n.StartByte() == firstMissing.StartByte() && n.EndByte() < firstMissing.EndByte())) {
			firstMissing = n
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	target := firstErr
	targetLabel := "error"
	if target == nil {
		target = firstMissing
		targetLabel = "missing"
	}
	if target == nil {
		target = firstHasErr
		targetLabel = "hasError"
	}
	if target == nil {
		t.Fatal("root has error but no explicit error, missing, or error-carrying subtree found")
	}

	start := int(target.StartByte())
	end := int(target.EndByte())
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
	if ctxEnd > len(srcBytes) {
		ctxEnd = len(srcBytes)
	}

	var path []string
	for cur := target; cur != nil; cur = cur.Parent() {
		path = append(path, cur.Type(genLang))
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	parent := target.Parent()
	var siblings []string
	if parent != nil {
		for i := 0; i < parent.ChildCount(); i++ {
			child := parent.Child(i)
			if child == nil {
				continue
			}
			siblings = append(siblings, fmt.Sprintf("%d:%s[%d:%d]", i, child.Type(genLang), child.StartByte(), child.EndByte()))
		}
	}

	t.Logf("first %s: type=%s range=[%d:%d] point=[%d:%d]-[%d:%d] hasError=%v missing=%v explicitError=%v", targetLabel, target.Type(genLang), start, end, target.StartPoint().Row, target.StartPoint().Column, target.EndPoint().Row, target.EndPoint().Column, target.HasError(), target.IsMissing(), target.IsError())
	t.Logf("path: %s", strings.Join(path, " > "))
	if len(siblings) > 0 {
		t.Logf("parent children: %s", strings.Join(siblings, " | "))
	}
	t.Logf("context: %q", string(srcBytes[ctxStart:ctxEnd]))
	t.Logf("target sexpr: %s", target.SExpr(genLang))
	if parent != nil {
		t.Logf("parent sexpr: %.800s", parent.SExpr(genLang))
	}
}

func TestDiagTSParserPrefixCheckpoints(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)

	srcBytes, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	lines := strings.Split(string(srcBytes), "\n")
	genParser := gotreesitter.NewParser(genLang)
	refParser := gotreesitter.NewParser(refLang)

	for _, n := range []int{20, 40, 60, 80, 100, 120, 160, 200} {
		if n > len(lines) {
			break
		}
		prefix := strings.Join(lines[:n], "\n")
		genTree, err := genParser.Parse([]byte(prefix))
		if err != nil {
			t.Fatalf("gen parse prefix %d: %v", n, err)
		}
		refTree, err := refParser.Parse([]byte(prefix))
		if err != nil {
			t.Fatalf("ref parse prefix %d: %v", n, err)
		}
		t.Logf("prefix=%d genErr=%v refErr=%v runtime=%s", n, genTree.RootNode().HasError(), refTree.RootNode().HasError(), genTree.ParseRuntime().Summary())
		t.Logf("gen[%d]: %.260s", n, genTree.RootNode().SExpr(genLang))
		t.Logf("ref[%d]: %.260s", n, refTree.RootNode().SExpr(refLang))
	}
}

func TestDiagTSSyntheticClosedNamespacePrefixes(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)

	srcBytes, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	lines := strings.Split(string(srcBytes), "\n")
	genParser := gotreesitter.NewParser(genLang)
	refParser := gotreesitter.NewParser(refLang)

	for _, n := range []int{20, 35, 39, 53, 80, 120, 200, 400, 800, 1200, 1600} {
		if n > len(lines) {
			break
		}
		prefix := strings.Join(lines[:n], "\n") + "\n}\n"
		genTree, err := genParser.Parse([]byte(prefix))
		if err != nil {
			t.Fatalf("gen parse prefix %d: %v", n, err)
		}
		refTree, err := refParser.Parse([]byte(prefix))
		if err != nil {
			t.Fatalf("ref parse prefix %d: %v", n, err)
		}
		t.Logf("closed-prefix=%d genErr=%v refErr=%v runtime=%s", n, genTree.RootNode().HasError(), refTree.RootNode().HasError(), genTree.ParseRuntime().Summary())
		t.Logf("gen-closed[%d]: %.320s", n, genTree.RootNode().SExpr(genLang))
		t.Logf("ref-closed[%d]: %.320s", n, refTree.RootNode().SExpr(refLang))
	}
}

func TestDiagTSRelevantConflicts(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
	if err != nil {
		t.Fatalf("read grammar: %v", err)
	}
	gram, err := ImportGrammarJSON(data)
	if err != nil {
		t.Fatalf("import grammar: %v", err)
	}
	report, err := GenerateWithReport(gram)
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	ng, err := Normalize(gram)
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	match := 0
	for _, d := range report.Conflicts {
		s := d.String(ng)
		if strings.Contains(s, "lookup_type") ||
			strings.Contains(s, "parenthesized_type") ||
			strings.Contains(s, "type_query") ||
			strings.Contains(s, "subscript_expression") {
			t.Logf("conflict[%d]:\n%s", match, s)
			match++
			if match >= 20 {
				break
			}
		}
	}
	t.Logf("matched conflicts: %d / %d", match, len(report.Conflicts))
}

func TestDiagTSRelevantProductions(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	targets := map[string]bool{
		"type_query":         true,
		"parenthesized_type": true,
		"lookup_type":        true,
		"primary_type":       true,
	}
	for i, p := range ng.Productions {
		lhs := ng.Symbols[p.LHS].Name
		if !targets[lhs] {
			continue
		}
		t.Logf("prod %d: %s prec=%d explicit=%v assoc=%v", i, diagFormatProd(ng, i, len(p.RHS)), p.Prec, p.HasExplicitPrec, p.Assoc)
	}
}

func TestDiagTSCursorTailSamples(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)

	fullSrc, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	fullText := string(fullSrc)
	start := strings.Index(fullText, "function createSyntaxCursor(sourceFile: SourceFile): SyntaxCursor {")
	end := strings.Index(fullText[start:], "function isDeclarationFileName(fileName: string): boolean {")
	if start < 0 || end < 0 {
		t.Fatal("failed to locate createSyntaxCursor snippet in parser.ts")
	}
	extracted := fullText[start : start+end]

	samples := []struct {
		name   string
		source string
	}{
		{
			name: "minimal_const_enum_in_function",
			source: "namespace ts {\n" +
				"  function f() {\n" +
				"    const enum InvalidPosition {\n" +
				"      Value = -1\n" +
				"    }\n" +
				"  }\n" +
				"}\n",
		},
		{
			name: "returned_object_then_const_enum",
			source: "namespace ts {\n" +
				"  interface SyntaxCursor { currentNode(position: number): IncrementalNode; }\n" +
				"  interface IncrementalNode {}\n" +
				"  function f(current: IncrementalNode): SyntaxCursor {\n" +
				"    return {\n" +
				"      currentNode(position: number) {\n" +
				"        return <IncrementalNode>current;\n" +
				"      }\n" +
				"    };\n" +
				"    const enum InvalidPosition {\n" +
				"      Value = -1\n" +
				"    }\n" +
				"  }\n" +
				"}\n",
		},
		{
			name: "returned_object_local_function_const_enum",
			source: "namespace ts {\n" +
				"  interface SyntaxCursor { currentNode(position: number): IncrementalNode; }\n" +
				"  interface IncrementalNode { pos: number; end: number; }\n" +
				"  interface SourceFile { statements: IncrementalNode[]; }\n" +
				"  function forEachChild(node: any, a: any, b: any): void {}\n" +
				"  function f(sourceFile: SourceFile): SyntaxCursor {\n" +
				"    let currentArray = sourceFile.statements;\n" +
				"    let currentArrayIndex = 0;\n" +
				"    let current = currentArray[currentArrayIndex];\n" +
				"    let lastQueriedPosition = 0;\n" +
				"    return {\n" +
				"      currentNode(position: number) {\n" +
				"        if (position !== lastQueriedPosition) {\n" +
				"          if (current && current.end === position && currentArrayIndex < (currentArray.length - 1)) {\n" +
				"            currentArrayIndex++;\n" +
				"            current = currentArray[currentArrayIndex];\n" +
				"          }\n" +
				"        }\n" +
				"        lastQueriedPosition = position;\n" +
				"        return <IncrementalNode>current;\n" +
				"      }\n" +
				"    };\n" +
				"    function findHighestListElementThatStartsAtPosition(position: number) {\n" +
				"      currentArrayIndex = InvalidPosition.Value;\n" +
				"      forEachChild(sourceFile, visitNode, visitArray);\n" +
				"      function visitNode(node: IncrementalNode) {\n" +
				"        return !!node;\n" +
				"      }\n" +
				"      function visitArray(array: IncrementalNode[]) {\n" +
				"        return !!array;\n" +
				"      }\n" +
				"    }\n" +
				"    const enum InvalidPosition {\n" +
				"      Value = -1\n" +
				"    }\n" +
				"  }\n" +
				"}\n",
		},
		{
			name:   "extracted_cursor_tail",
			source: "namespace ts {\n" + extracted + "\n}\n",
		},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			genTree, err := gotreesitter.NewParser(genLang).Parse([]byte(tc.source))
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			refTree, err := gotreesitter.NewParser(refLang).Parse([]byte(tc.source))
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}
			t.Logf("genErr=%v refErr=%v", genTree.RootNode().HasError(), refTree.RootNode().HasError())
			t.Logf("gen runtime: %s", genTree.ParseRuntime().Summary())
			t.Logf("gen: %.1200s", genTree.RootNode().SExpr(genLang))
			t.Logf("ref: %.1200s", refTree.RootNode().SExpr(refLang))
		})
	}
}

func TestDiagTSFullParserRootChildren(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	src, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	tree, err := gotreesitter.NewParser(genLang).Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	root := tree.RootNode()
	t.Logf("root=%s err=%v childCount=%d runtime=%s", root.Type(genLang), root.HasError(), root.ChildCount(), tree.ParseRuntime().Summary())
	for i := 0; i < root.ChildCount() && i < 12; i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		t.Logf("child[%d]=%s [%d:%d] err=%v sexpr=%.400s", i, child.Type(genLang), child.StartByte(), child.EndByte(), child.HasError() || child.IsError(), child.SExpr(genLang))
	}
	start := root.ChildCount() - 8
	if start < 12 {
		start = 12
	}
	for i := start; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		t.Logf("tail child[%d]=%s [%d:%d] err=%v sexpr=%.200s", i, child.Type(genLang), child.StartByte(), child.EndByte(), child.HasError() || child.IsError(), child.SExpr(genLang))
	}
}

func TestDiagTSFullParserFirstDivergence(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)
	src, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	genTree, err := gotreesitter.NewParser(genLang).Parse(src)
	if err != nil {
		t.Fatalf("gen parse: %v", err)
	}
	refTree, err := gotreesitter.NewParser(refLang).Parse(src)
	if err != nil {
		t.Fatalf("ref parse: %v", err)
	}
	t.Logf("gen runtime: %s", genTree.ParseRuntime().Summary())
	divs := compareTreesDeep(genTree.RootNode(), genLang, refTree.RootNode(), refLang, "root", 8)
	if len(divs) == 0 {
		t.Log("no deep divergences")
		return
	}
	for i, dv := range divs {
		t.Logf("div[%d]: %s", i, dv.String())
		genN := findNodeByPath(genTree.RootNode(), genLang, dv.Path)
		refN := findNodeByPath(refTree.RootNode(), refLang, dv.Path)
		if genN != nil {
			t.Logf("gen-sexpr: %.600s", safeSExpr(genN, genLang, 600))
		}
		if refN != nil {
			t.Logf("ref-sexpr: %.600s", safeSExpr(refN, refLang, 600))
		}
	}
}

func TestDiagTSNamespaceBodyReparseAsProgram(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	src, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}

	open := bytes.Index(src, []byte("namespace ts {"))
	if open < 0 {
		t.Fatal("namespace prefix not found")
	}
	open += len("namespace ts {")
	close := bytes.LastIndexByte(src, '}')
	if close <= open {
		t.Fatalf("invalid body bounds open=%d close=%d", open, close)
	}
	body := src[open:close]
	isTrivia := func(b []byte) bool {
		for _, c := range b {
			switch c {
			case ' ', '\t', '\n', '\r':
				continue
			default:
				return false
			}
		}
		return true
	}
	tree, err := gotreesitter.NewParser(genLang).Parse(body)
	if err != nil {
		t.Fatalf("parse body: %v", err)
	}
	root := tree.RootNode()
	t.Logf("body root=%s err=%v childCount=%d runtime=%s", root.Type(genLang), root.HasError(), root.ChildCount(), tree.ParseRuntime().Summary())
	triviaErrors := 0
	nonTriviaErrors := 0
	typeCounts := map[string]int{}
	for i := 0; i < root.ChildCount() && i < 12; i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		t.Logf("child[%d]=%s [%d:%d] err=%v sexpr=%.300s", i, child.Type(genLang), child.StartByte(), child.EndByte(), child.HasError() || child.IsError(), child.SExpr(genLang))
	}
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child == nil {
			continue
		}
		typeCounts[child.Type(genLang)]++
		if child.Type(genLang) != "ERROR" {
			continue
		}
		if int(child.EndByte()) <= len(body) && isTrivia(body[child.StartByte():child.EndByte()]) {
			triviaErrors++
		} else {
			nonTriviaErrors++
		}
	}
	t.Logf("root type counts: ERROR=%d comment=%d expression_statement=%d lexical_declaration=%d function_declaration=%d class_declaration=%d interface_declaration=%d",
		typeCounts["ERROR"], typeCounts["comment"], typeCounts["expression_statement"], typeCounts["lexical_declaration"], typeCounts["function_declaration"], typeCounts["class_declaration"], typeCounts["interface_declaration"])
	t.Logf("ERROR breakdown: trivia=%d non-trivia=%d", triviaErrors, nonTriviaErrors)
}

func TestDiagTSFullParserStatementBlockTypes(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	src, err := os.ReadFile("/tmp/grammar_parity/typescript/examples/parser.ts")
	if err != nil {
		t.Fatalf("read parser.ts: %v", err)
	}
	tree, err := gotreesitter.NewParser(genLang).Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	root := tree.RootNode()
	if root == nil || root.ChildCount() < 3 {
		t.Fatalf("unexpected root: %v", root)
	}
	stmtBlock := findFirstDescendantByType(root, genLang, "statement_block")
	if stmtBlock == nil {
		t.Fatal("statement_block not found")
	}
	t.Logf("statement_block childCount=%d range=[%d:%d]", stmtBlock.ChildCount(), stmtBlock.StartByte(), stmtBlock.EndByte())
	typeCounts := map[string]int{}
	for i := 0; i < stmtBlock.ChildCount(); i++ {
		child := stmtBlock.Child(i)
		if child == nil {
			continue
		}
		typeCounts[child.Type(genLang)]++
		if i < 30 {
			t.Logf("child[%d]=%s [%d:%d] err=%v sexpr=%.220s", i, child.Type(genLang), child.StartByte(), child.EndByte(), child.HasError() || child.IsError(), child.SExpr(genLang))
		}
	}
	keys := make([]string, 0, len(typeCounts))
	for k := range typeCounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if typeCounts[k] >= 5 {
			t.Logf("typeCount[%s]=%d", k, typeCounts[k])
		}
	}
}

func TestDiagTSNamespaceEnumThenLet(t *testing.T) {
	data, err := os.ReadFile("/tmp/grammar_parity/typescript/typescript/src/grammar.json")
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
	refLang := grammars.TypescriptLanguage()
	adaptExternalScanner(refLang, genLang)

	samples := []string{
		"namespace ts {\n    const enum SignatureFlags {\n        None = 0,\n        Yield = 1 << 0,\n        Await = 1 << 1,\n        Type  = 1 << 2,\n        RequireCompleteParameterList = 1 << 3,\n        IgnoreMissingOpenBrace = 1 << 4,\n        JSDoc = 1 << 5,\n    }\n}\n",
		"namespace ts {\n    const enum SignatureFlags {\n        None = 0,\n        Yield = 1 << 0,\n        Await = 1 << 1,\n        Type  = 1 << 2,\n        RequireCompleteParameterList = 1 << 3,\n        IgnoreMissingOpenBrace = 1 << 4,\n        JSDoc = 1 << 5,\n    }\n\n    // tslint:disable variable-name\n    let NodeConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n}\n",
		"namespace ts {\n    const enum SignatureFlags {\n        None = 0,\n        Yield = 1 << 0,\n        Await = 1 << 1,\n        Type  = 1 << 2,\n        RequireCompleteParameterList = 1 << 3,\n        IgnoreMissingOpenBrace = 1 << 4,\n        JSDoc = 1 << 5,\n    }\n\n    // tslint:disable variable-name\n    let NodeConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n    let TokenConstructor: new (kind: SyntaxKind, pos: number, end: number) => Node;\n}\n",
	}
	for i, sample := range samples {
		t.Run(fmt.Sprintf("sample_%d", i), func(t *testing.T) {
			genTree, err := gotreesitter.NewParser(genLang).Parse([]byte(sample))
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			refTree, err := gotreesitter.NewParser(refLang).Parse([]byte(sample))
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}
			t.Logf("genErr=%v refErr=%v", genTree.RootNode().HasError(), refTree.RootNode().HasError())
			t.Logf("gen runtime: %s", genTree.ParseRuntime().Summary())
			t.Logf("gen: %.1200s", genTree.RootNode().SExpr(genLang))
			t.Logf("ref: %.1200s", refTree.RootNode().SExpr(refLang))
		})
	}
}

func findFirstDescendantByType(n *gotreesitter.Node, lang *gotreesitter.Language, want string) *gotreesitter.Node {
	if n == nil || lang == nil {
		return nil
	}
	if n.Type(lang) == want {
		return n
	}
	for i := 0; i < n.ChildCount(); i++ {
		if found := findFirstDescendantByType(n.Child(i), lang, want); found != nil {
			return found
		}
	}
	return nil
}

func diagFormatStateActions(ng *NormalizedGrammar, tables *LRTables, state int) string {
	acts := tables.ActionTable[state]
	if len(acts) == 0 {
		return "[]"
	}
	type pair struct {
		sym  int
		text string
	}
	items := make([]pair, 0, len(acts))
	for sym, alts := range acts {
		items = append(items, pair{sym: sym, text: diagSymbolName(ng, sym) + "=" + diagFormatActions(ng, alts)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].sym < items[j].sym })
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = it.text
	}
	return "[" + strings.Join(parts, "; ") + "]"
}

func diagFormatLangStateActions(lang *gotreesitter.Language, state int) string {
	if lang == nil || state < 0 {
		return "[]"
	}
	getIdx := func(state int, sym int) uint16 {
		if state < 0 {
			return 0
		}
		if state < len(lang.ParseTable) {
			if sym < 0 || sym >= len(lang.ParseTable[state]) {
				return 0
			}
			return lang.ParseTable[state][sym]
		}
		smallBase := int(lang.LargeStateCount)
		smallIdx := state - smallBase
		if smallIdx < 0 || smallIdx >= len(lang.SmallParseTableMap) {
			return 0
		}
		offset := int(lang.SmallParseTableMap[smallIdx])
		if offset >= len(lang.SmallParseTable) {
			return 0
		}
		groupCount := lang.SmallParseTable[offset]
		pos := offset + 1
		for i := uint16(0); i < groupCount; i++ {
			if pos+1 >= len(lang.SmallParseTable) {
				return 0
			}
			sectionValue := lang.SmallParseTable[pos]
			symbolCount := lang.SmallParseTable[pos+1]
			pos += 2
			for j := uint16(0); j < symbolCount; j++ {
				if pos >= len(lang.SmallParseTable) {
					return 0
				}
				if int(lang.SmallParseTable[pos]) == sym {
					return sectionValue
				}
				pos++
			}
		}
		return 0
	}
	type pair struct {
		sym  int
		text string
	}
	var items []pair
	for sym := 0; sym < int(lang.SymbolCount); sym++ {
		idx := getIdx(state, sym)
		if idx == 0 || int(idx) >= len(lang.ParseActions) {
			continue
		}
		var acts []string
		for _, act := range lang.ParseActions[idx].Actions {
			switch act.Type {
			case gotreesitter.ParseActionShift:
				acts = append(acts, fmt.Sprintf("shift(%d)", act.State))
			case gotreesitter.ParseActionReduce:
				name := "?"
				if int(act.Symbol) < len(lang.SymbolNames) {
					name = lang.SymbolNames[act.Symbol]
				}
				acts = append(acts, fmt.Sprintf("reduce(%s/%d,cnt=%d,prod=%d)", name, act.Symbol, act.ChildCount, act.ProductionID))
			case gotreesitter.ParseActionAccept:
				acts = append(acts, "accept")
			case gotreesitter.ParseActionRecover:
				acts = append(acts, fmt.Sprintf("recover(%d)", act.State))
			}
		}
		name := "?"
		if sym < len(lang.SymbolNames) {
			name = lang.SymbolNames[sym]
		}
		items = append(items, pair{sym: sym, text: fmt.Sprintf("%s(%d)=[%s]", name, sym, strings.Join(acts, ", "))})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].sym < items[j].sym })
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = it.text
	}
	return "[" + strings.Join(parts, "; ") + "]"
}
