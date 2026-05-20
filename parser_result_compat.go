package gotreesitter

// resultCompatibilityStrut is a language-specific post-build tree rewrite.
//
// These are deliberately named as struts: they hold up parity while the parser,
// scanner, or grammargen output is not yet producing the final C tree-sitter
// shape directly. New parser work should prefer fixing the underlying runtime
// or grammar generation path. When that happens, remove the corresponding strut
// and its regression tests instead of letting this registry become permanent
// architecture.
type resultCompatibilityStrut func(resultCompatibilityContext)

type resultCompatibilityContext struct {
	root   *Node
	source []byte
	parser *Parser
	lang   *Language
}

type resultCompatibilityStrutID uint8

const (
	resultCompatibilityStrutNone resultCompatibilityStrutID = iota
	resultCompatibilityStrutBash
	resultCompatibilityStrutC
	resultCompatibilityStrutCSharp
	resultCompatibilityStrutCaddy
	resultCompatibilityStrutCobol
	resultCompatibilityStrutComment
	resultCompatibilityStrutCooklang
	resultCompatibilityStrutD
	resultCompatibilityStrutDart
	resultCompatibilityStrutElixir
	resultCompatibilityStrutErlang
	resultCompatibilityStrutFortran
	resultCompatibilityStrutGo
	resultCompatibilityStrutHaskell
	resultCompatibilityStrutHCL
	resultCompatibilityStrutHTML
	resultCompatibilityStrutIni
	resultCompatibilityStrutJavaScript
	resultCompatibilityStrutLua
	resultCompatibilityStrutMake
	resultCompatibilityStrutNginx
	resultCompatibilityStrutNim
	resultCompatibilityStrutPascal
	resultCompatibilityStrutPerl
	resultCompatibilityStrutPHP
	resultCompatibilityStrutPowerShell
	resultCompatibilityStrutPug
	resultCompatibilityStrutPython
	resultCompatibilityStrutRST
	resultCompatibilityStrutRust
	resultCompatibilityStrutRuby
	resultCompatibilityStrutScala
	resultCompatibilityStrutSQL
	resultCompatibilityStrutSvelte
	resultCompatibilityStrutTypeScript
	resultCompatibilityStrutYAML
	resultCompatibilityStrutZig
)

type resultCompatibilityStrutRule struct {
	languageName string
	strut        resultCompatibilityStrutID
}

var resultCompatibilityStrutRules = []resultCompatibilityStrutRule{
	{"bash", resultCompatibilityStrutBash},
	{"c", resultCompatibilityStrutC},
	{"c_sharp", resultCompatibilityStrutCSharp},
	{"caddy", resultCompatibilityStrutCaddy},
	{"cobol", resultCompatibilityStrutCobol},
	{"COBOL", resultCompatibilityStrutCobol},
	{"comment", resultCompatibilityStrutComment},
	{"cooklang", resultCompatibilityStrutCooklang},
	{"d", resultCompatibilityStrutD},
	{"dart", resultCompatibilityStrutDart},
	{"elixir", resultCompatibilityStrutElixir},
	{"erlang", resultCompatibilityStrutErlang},
	{"fortran", resultCompatibilityStrutFortran},
	{"go", resultCompatibilityStrutGo},
	{"haskell", resultCompatibilityStrutHaskell},
	{"hcl", resultCompatibilityStrutHCL},
	{"html", resultCompatibilityStrutHTML},
	{"ini", resultCompatibilityStrutIni},
	{"javascript", resultCompatibilityStrutJavaScript},
	{"lua", resultCompatibilityStrutLua},
	{"make", resultCompatibilityStrutMake},
	{"nginx", resultCompatibilityStrutNginx},
	{"nim", resultCompatibilityStrutNim},
	{"pascal", resultCompatibilityStrutPascal},
	{"perl", resultCompatibilityStrutPerl},
	{"php", resultCompatibilityStrutPHP},
	{"powershell", resultCompatibilityStrutPowerShell},
	{"pug", resultCompatibilityStrutPug},
	{"python", resultCompatibilityStrutPython},
	{"rst", resultCompatibilityStrutRST},
	{"rust", resultCompatibilityStrutRust},
	{"ruby", resultCompatibilityStrutRuby},
	{"scala", resultCompatibilityStrutScala},
	{"sql", resultCompatibilityStrutSQL},
	{"svelte", resultCompatibilityStrutSvelte},
	{"tsx", resultCompatibilityStrutTypeScript},
	{"typescript", resultCompatibilityStrutTypeScript},
	{"yaml", resultCompatibilityStrutYAML},
	{"zig", resultCompatibilityStrutZig},
}

func resultCompatibilityStrutIDForLanguage(name string) resultCompatibilityStrutID {
	for _, rule := range resultCompatibilityStrutRules {
		if rule.languageName == name {
			return rule.strut
		}
	}
	return resultCompatibilityStrutNone
}

func resultCompatibilityStrutForLanguage(name string) resultCompatibilityStrut {
	id := resultCompatibilityStrutIDForLanguage(name)
	if id == resultCompatibilityStrutNone {
		return nil
	}
	return func(ctx resultCompatibilityContext) {
		runResultCompatibilityStrut(id, ctx)
	}
}

// normalizeResultCompatibility applies narrow post-build tree rewrites that
// keep gotreesitter output aligned with C tree-sitter and existing recovery
// expectations for grammars with known normalization gaps.
func normalizeResultCompatibility(root *Node, source []byte, p *Parser) {
	var lang *Language
	if p != nil {
		lang = p.language
	}
	if root == nil || lang == nil {
		return
	}
	if id := resultCompatibilityStrutIDForLanguage(lang.Name); id != resultCompatibilityStrutNone {
		runResultCompatibilityStrut(id, resultCompatibilityContext{
			root:   root,
			source: source,
			parser: p,
			lang:   lang,
		})
	}
	normalizeResultCollapsedNamedLeafChildren(root, lang)
}

func runResultCompatibilityStrut(id resultCompatibilityStrutID, ctx resultCompatibilityContext) {
	switch id {
	case resultCompatibilityStrutBash:
		normalizeBashProgramVariableAssignments(ctx.root, ctx.lang)
		normalizeBashGeneratedCommandAssignments(ctx.root, ctx.source, ctx.lang)
		normalizeBashCommandNameArguments(ctx.root, ctx.lang)
	case resultCompatibilityStrutC:
		normalizeCCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutCSharp:
		normalizeCSharpCompatibility(ctx.root, ctx.source, ctx.parser, ctx.lang)
	case resultCompatibilityStrutCaddy:
		normalizeTopLevelTrailingLineBreakSpan(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutCobol:
		normalizeCobolCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutComment:
		normalizeCommentTrailingExtraTrivia(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutCooklang:
		normalizeCooklangTrailingStepTail(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutD:
		normalizeDCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutDart:
		normalizeDartCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutElixir:
		normalizeElixirNestedCallTargetFields(ctx.root, ctx.lang)
	case resultCompatibilityStrutErlang:
		normalizeErlangSourceFileForms(ctx.root, ctx.lang)
	case resultCompatibilityStrutFortran:
		normalizeFortranStatementLineBreaks(ctx.root, ctx.source, ctx.lang)
		normalizeTopLevelTrailingLineBreakSpan(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutGo:
		normalizeGoReturnedTreeCompatibility(ctx.root, ctx.source, ctx.parser, ctx.lang)
	case resultCompatibilityStrutHaskell:
		normalizeHaskellCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutHCL:
		normalizeHCLConfigFileRoot(ctx.root, ctx.lang)
	case resultCompatibilityStrutHTML:
		normalizeHTMLCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutIni:
		normalizeIniSectionStarts(ctx.root, ctx.lang)
	case resultCompatibilityStrutJavaScript:
		normalizeJavaScriptCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutLua:
		normalizeLuaChunkLocalDeclarationFields(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutMake:
		normalizeMakeConditionalConsequenceFields(ctx.root, ctx.lang)
	case resultCompatibilityStrutNginx:
		normalizeNginxAttributeLineBreaks(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutNim:
		normalizeNimTopLevelCallEnd(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutPascal:
		normalizePascalTopLevelProgramEnd(ctx.root, ctx.source, ctx.lang)
		normalizePascalTrailingExtraTrivia(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutPerl:
		normalizePerlCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutPHP:
		normalizePHPCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutPowerShell:
		normalizePowerShellProgramShape(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutPug:
		normalizeTopLevelTrailingLineBreakSpan(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutPython:
		normalizePythonCompatibilityWithParser(ctx.root, ctx.source, ctx.parser, ctx.lang)
	case resultCompatibilityStrutRST:
		normalizeRSTTopLevelSectionEnd(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutRust:
		normalizeRustCompatibility(ctx.root, ctx.source, ctx.parser, ctx.lang)
	case resultCompatibilityStrutRuby:
		normalizeRubyThenStarts(ctx.root, ctx.lang)
		normalizeRubyTopLevelModuleBounds(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutScala:
		normalizeScalaCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutSQL:
		normalizeSQLRecoveredSelectRoot(ctx.root, ctx.lang)
	case resultCompatibilityStrutSvelte:
		normalizeSvelteTrailingExtraTrivia(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutTypeScript:
		normalizeTypeScriptTreeCompatibility(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutYAML:
		normalizeYAMLRecoveredRoot(ctx.root, ctx.source, ctx.lang)
	case resultCompatibilityStrutZig:
		normalizeZigEmptyInitListFields(ctx.root, ctx.lang)
	}
}
