package grammargen

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestNamedStringChoiceTokenBecomesKeyword(t *testing.T) {
	g := NewGrammar("named_string_choice_keyword")
	g.Define("source_file", Sym("predefined_type"))
	g.Define("predefined_type", Token(Choice(
		Str("int"),
		Str("string"),
		Str("nint"),
	)))
	g.Define("identifier", Pat(`[A-Za-z_][A-Za-z0-9_]*`))
	g.SetWord("identifier")

	ng, err := Normalize(g)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	predefinedTypeSym := -1
	for i, sym := range ng.Symbols {
		if sym.Name == "predefined_type" {
			predefinedTypeSym = i
			break
		}
	}
	if predefinedTypeSym < 0 {
		t.Fatal("predefined_type symbol not found")
	}

	foundKeyword := false
	for _, symID := range ng.KeywordSymbols {
		if symID == predefinedTypeSym {
			foundKeyword = true
			break
		}
	}
	if !foundKeyword {
		t.Fatalf("predefined_type sym %d missing from keyword set", predefinedTypeSym)
	}

	for _, term := range ng.Terminals {
		if term.SymbolID == predefinedTypeSym {
			t.Fatalf("predefined_type sym %d still present in main terminals", predefinedTypeSym)
		}
	}

	foundEntry := false
	for _, entry := range ng.KeywordEntries {
		if entry.SymbolID == predefinedTypeSym {
			foundEntry = true
			break
		}
	}
	if !foundEntry {
		t.Fatalf("predefined_type sym %d missing from keyword entries", predefinedTypeSym)
	}
}

func TestBareLexicalChoiceBecomesNamedToken(t *testing.T) {
	g := NewGrammar("bare_lexical_choice_named_token")
	g.Define("source_file", Sym("builtin_type"))
	g.Define("builtin_type", Choice(
		Str("bool"),
		Pat(`(i|u)[1-9][0-9]*`),
	))
	g.Define("identifier", Pat(`[A-Za-z_][A-Za-z0-9_]*`))
	g.SetWord("identifier")

	ng, err := Normalize(g)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	builtinTypeSym := -1
	for i, sym := range ng.Symbols {
		if sym.Name == "builtin_type" {
			builtinTypeSym = i
			if sym.Kind != SymbolNamedToken {
				t.Fatalf("builtin_type kind = %v, want SymbolNamedToken", sym.Kind)
			}
			break
		}
	}
	if builtinTypeSym < 0 {
		t.Fatal("builtin_type symbol not found")
	}
	lang, err := GenerateLanguage(g)
	if err != nil {
		t.Fatalf("GenerateLanguage: %v", err)
	}
	tree, err := gotreesitter.NewParser(lang).Parse([]byte("i32"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()
	if got := tree.RootNode().SExpr(lang); got != "(source_file (builtin_type))" {
		t.Fatalf("SExpr = %s, want (source_file (builtin_type))", got)
	}
}

func TestExplicitPrecStringChoiceBecomesNonterminal(t *testing.T) {
	g := NewGrammar("explicit_prec_string_choice_nonterminal")
	g.Define("source_file", Sym("initializer"))
	g.Define("initializer", Seq(Sym("init_class"), Sym("identifier")))
	g.Define("init_class", PrecRight(0, Choice(Str("="), Str("+="), Str("-="))))
	g.Define("identifier", Pat(`[a-z]+`))

	ng, err := Normalize(g)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	initClassSym := -1
	for i, sym := range ng.Symbols {
		if sym.Name == "init_class" {
			initClassSym = i
			if sym.Kind != SymbolNonterminal {
				t.Fatalf("init_class kind = %v, want SymbolNonterminal", sym.Kind)
			}
			if !sym.Visible || !sym.Named {
				t.Fatalf("init_class metadata = visible:%v named:%v, want visible/named", sym.Visible, sym.Named)
			}
			break
		}
	}
	if initClassSym < 0 {
		t.Fatal("init_class symbol not found")
	}

	for _, term := range ng.Terminals {
		if term.SymbolID == initClassSym {
			t.Fatalf("init_class sym %d was emitted as a terminal", initClassSym)
		}
	}

	wantAlts := map[string]bool{"=": false, "+=": false, "-=": false}
	for _, prod := range ng.Productions {
		if prod.LHS != initClassSym || len(prod.RHS) != 1 {
			continue
		}
		rhs := prod.RHS[0]
		if rhs < 0 || rhs >= len(ng.Symbols) {
			continue
		}
		name := ng.Symbols[rhs].Name
		if _, ok := wantAlts[name]; ok {
			wantAlts[name] = true
		}
	}
	for name, found := range wantAlts {
		if !found {
			t.Fatalf("missing init_class -> %q production", name)
		}
	}

	lang, err := GenerateLanguage(g)
	if err != nil {
		t.Fatalf("GenerateLanguage: %v", err)
	}
	tree, err := gotreesitter.NewParser(lang).Parse([]byte("+=abc"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("parse has error: %s", root.SExpr(lang))
	}
	if got := root.SExpr(lang); got != "(source_file (initializer (init_class) (identifier)))" {
		t.Fatalf("SExpr = %s, want init_class wrapper", got)
	}
}

func TestAliasedInlinePatternWinsSameLengthNamedPatternTie(t *testing.T) {
	g := NewGrammar("aliased_inline_pattern_precedence")
	g.Define("source_file", Choice(
		Sym("preproc_include"),
		Sym("preproc_call"),
	))
	g.Define("preproc_include", Seq(
		Alias(Pat(`#[ \t]*include`), "#include", false),
		Sym("system_lib_string"),
		ImmToken(Pat(`\r?\n`)),
	))
	g.Define("preproc_call", Seq(
		Field("directive", Sym("preproc_directive")),
		Optional(Field("argument", Sym("preproc_arg"))),
		ImmToken(Pat(`\r?\n`)),
	))
	g.Define("preproc_directive", Pat(`#[ \t]*[a-zA-Z0-9]\w*`))
	g.Define("preproc_arg", Token(Pat(`[^\r\n]+`)))
	g.Define("system_lib_string", Token(Seq(
		Str("<"),
		Repeat(Pat(`[^>\n]`)),
		Str(">"),
	)))
	g.Extras = []*Rule{Pat(`[ \t]+`)}

	lang, err := GenerateLanguage(g)
	if err != nil {
		t.Fatalf("GenerateLanguage: %v", err)
	}
	tree, err := gotreesitter.NewParser(lang).Parse([]byte("#include <iostream>\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()
	if got := tree.RootNode().SExpr(lang); got != "(source_file (preproc_include (system_lib_string)))" {
		t.Fatalf("SExpr = %s, want (source_file (preproc_include (system_lib_string)))", got)
	}
}

func TestUppercaseUnicodeEscapeIdentifierDoesNotCaptureDigits(t *testing.T) {
	g := NewGrammar("unicode_escape_identifier_digit_split")
	g.Define("source_file", Choice(
		Sym("upper_case_identifier"),
		Sym("number_literal"),
	))
	g.Define("upper_case_identifier", Pat(`[A-Z\U00010400-\U00010427][A-Z0-9]*`))
	g.Define("number_literal", Pat(`[0-9]+`))

	lang, err := GenerateLanguage(g)
	if err != nil {
		t.Fatalf("GenerateLanguage: %v", err)
	}
	tree, err := gotreesitter.NewParser(lang).Parse([]byte("1"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()
	if got := tree.RootNode().SExpr(lang); got != "(source_file (number_literal))" {
		t.Fatalf("SExpr = %s, want (source_file (number_literal))", got)
	}
}
