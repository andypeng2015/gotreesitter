package gotreesitter

import "testing"

func TestBuildPowerShellVariableMemberAccessBuildsRecoveredPath(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "member_access", "variable", "\\", ".", "member_name", "simple_name", "ERROR",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "member_access", Visible: true, Named: true},
			{Name: "variable", Visible: true, Named: true},
			{Name: "\\", Visible: true, Named: false},
			{Name: ".", Visible: true, Named: false},
			{Name: "member_name", Visible: true, Named: true},
			{Name: "simple_name", Visible: true, Named: true},
			{Name: "ERROR", Visible: true, Named: true},
		},
	}

	source := []byte("$targetPsHome\\pwrshplugin.dll")
	arena := newNodeArena(arenaClassFull)
	node := buildPowerShellVariableMemberAccess(arena, source, lang, 0, len(source))
	if node == nil {
		t.Fatal("node = nil")
	}
	if got, want := node.Type(lang), "member_access"; got != want {
		t.Fatalf("node.Type = %q, want %q", got, want)
	}
	if got, want := len(node.children), 4; got != want {
		t.Fatalf("len(node.children) = %d, want %d", got, want)
	}
	if got, want := node.children[1].Type(lang), "ERROR"; got != want {
		t.Fatalf("node.children[1].Type = %q, want %q", got, want)
	}
	if !node.children[1].isExtra() {
		t.Fatalf("node.children[1].isExtra = false, want true")
	}
}

func TestBuildPowerShellExpandableStringLiteralKeepsFullRangeWithVariable(t *testing.T) {
	lang := &Language{
		Name:        "powershell",
		SymbolNames: []string{"EOF", "expandable_string_literal", "variable"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "expandable_string_literal", Visible: true, Named: true},
			{Name: "variable", Visible: true, Named: true},
		},
	}

	source := []byte("\"Creating $pluginBasePath\"")
	arena := newNodeArena(arenaClassFull)
	node := buildPowerShellExpandableStringLiteral(arena, source, lang, 0, len(source))
	if node == nil {
		t.Fatal("node = nil")
	}
	if got, want := node.Type(lang), "expandable_string_literal"; got != want {
		t.Fatalf("node.Type = %q, want %q", got, want)
	}
	if got, want := node.startByte, uint32(0); got != want {
		t.Fatalf("node.startByte = %d, want %d", got, want)
	}
	if got, want := node.endByte, uint32(len(source)); got != want {
		t.Fatalf("node.endByte = %d, want %d", got, want)
	}
	if got, want := len(node.children), 1; got != want {
		t.Fatalf("len(node.children) = %d, want %d", got, want)
	}
	if got, want := node.children[0].Type(lang), "variable"; got != want {
		t.Fatalf("node.children[0].Type = %q, want %q", got, want)
	}
}

func TestBuildPowerShellTypeLiteralMarksRecoveredPlusTailExtra(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "type_literal", "[", "]", "type_spec", "type_name", "type_identifier", ".", "+", "simple_name", "ERROR",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "type_literal", Visible: true, Named: true},
			{Name: "[", Visible: true, Named: false},
			{Name: "]", Visible: true, Named: false},
			{Name: "type_spec", Visible: true, Named: true},
			{Name: "type_name", Visible: true, Named: true},
			{Name: "type_identifier", Visible: true, Named: true},
			{Name: ".", Visible: true, Named: false},
			{Name: "+", Visible: true, Named: false},
			{Name: "simple_name", Visible: true, Named: true},
			{Name: "ERROR", Visible: true, Named: true},
		},
	}

	source := []byte("[System.Environment+SpecialFolder]")
	arena := newNodeArena(arenaClassFull)
	node := buildPowerShellTypeLiteral(arena, source, lang, 0, len(source))
	if node == nil {
		t.Fatal("node = nil")
	}
	if got, want := len(node.children), 4; got != want {
		t.Fatalf("len(node.children) = %d, want %d", got, want)
	}
	if got, want := node.children[2].Type(lang), "ERROR"; got != want {
		t.Fatalf("node.children[2].Type = %q, want %q", got, want)
	}
	if !node.children[2].isExtra() {
		t.Fatalf("node.children[2].isExtra = false, want true")
	}
	if !node.HasError() {
		t.Fatalf("node.HasError = false, want true")
	}
}

func TestNormalizePowerShellErrorProgramRootRetagsCompatibleRoot(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "ERROR", "program", "comment", "param_block", "statement_list",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "ERROR", Visible: true, Named: true},
			{Name: "program", Visible: true, Named: true},
			{Name: "comment", Visible: true, Named: true},
			{Name: "param_block", Visible: true, Named: true},
			{Name: "statement_list", Visible: true, Named: true},
		},
	}
	arena := newNodeArena(arenaClassFull)
	children := []*Node{
		newLeafNodeInArena(arena, 3, true, 0, 9, Point{}, Point{Column: 9}),
		newLeafNodeInArena(arena, 4, true, 10, 20, Point{Column: 10}, Point{Column: 20}),
		newLeafNodeInArena(arena, 5, true, 22, 40, Point{Column: 22}, Point{Column: 40}),
	}
	root := newParentNodeInArena(arena, 1, true, children, nil, 0)

	normalizePowerShellErrorProgramRoot(root, lang)

	if got := root.Type(lang); got != "program" {
		t.Fatalf("root type = %q, want program", got)
	}
	if !root.IsNamed() {
		t.Fatal("root is not named")
	}
}

func TestNormalizePowerShellAssignmentOperatorTokensRestoresCommandArgumentSep(t *testing.T) {
	for _, sourceText := range []string{" ", ":"} {
		source := []byte(sourceText)
		lang := &Language{
			Name: "powershell",
			SymbolNames: []string{
				"EOF", "program", "command_argument_sep", sourceText,
			},
			SymbolMetadata: []SymbolMetadata{
				{Name: "EOF", Visible: false, Named: false},
				{Name: "program", Visible: true, Named: true},
				{Name: "command_argument_sep", Visible: true, Named: true},
				{Name: sourceText, Visible: true, Named: false},
			},
		}
		arena := newNodeArena(arenaClassFull)
		sep := newLeafNodeInArena(arena, 2, true, 0, uint32(len(source)), Point{}, Point{Column: uint32(len(source))})
		root := newParentNodeInArena(arena, 1, true, []*Node{sep}, nil, 0)

		normalizePowerShellAssignmentOperatorTokens(root, source, lang)

		if got := sep.ChildCount(); got != 1 {
			t.Fatalf("sep child count for %q = %d, want 1", sourceText, got)
		}
		child := sep.Child(0)
		if child == nil {
			t.Fatalf("sep child for %q = nil", sourceText)
		}
		if got := child.Type(lang); got != sourceText {
			t.Fatalf("sep child type for %q = %q", sourceText, got)
		}
		if child.IsNamed() {
			t.Fatalf("sep child for %q is named", sourceText)
		}
	}
}

func TestNormalizePowerShellAssignmentOperatorTokensRestoresComparisonOperator(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "program", "comparison_operator", "-match",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "program", Visible: true, Named: true},
			{Name: "comparison_operator", Visible: true, Named: true},
			{Name: "-match", Visible: true, Named: false},
		},
	}
	source := []byte("-match")
	arena := newNodeArena(arenaClassFull)
	op := newLeafNodeInArena(arena, 2, true, 0, uint32(len(source)), Point{}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 1, true, []*Node{op}, nil, 0)

	normalizePowerShellAssignmentOperatorTokens(root, source, lang)

	if got := op.ChildCount(); got != 1 {
		t.Fatalf("comparison_operator child count = %d, want 1", got)
	}
	child := op.Child(0)
	if child == nil {
		t.Fatal("comparison_operator child = nil")
	}
	if got := child.Type(lang); got != "-match" {
		t.Fatalf("comparison_operator child type = %q, want -match", got)
	}
	if child.IsNamed() {
		t.Fatal("comparison_operator child is named")
	}
}

func TestNormalizePowerShellAssignmentOperatorTokensRestoresFormatOperator(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "program", "format_operator", "-f",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "program", Visible: true, Named: true},
			{Name: "format_operator", Visible: true, Named: true},
			{Name: "-f", Visible: true, Named: false},
		},
	}
	source := []byte("-f")
	arena := newNodeArena(arenaClassFull)
	op := newLeafNodeInArena(arena, 2, true, 0, uint32(len(source)), Point{}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 1, true, []*Node{op}, nil, 0)

	normalizePowerShellAssignmentOperatorTokens(root, source, lang)

	if got := op.ChildCount(); got != 1 {
		t.Fatalf("format_operator child count = %d, want 1", got)
	}
	child := op.Child(0)
	if child == nil {
		t.Fatal("format_operator child = nil")
	}
	if got := child.Type(lang); got != "-f" {
		t.Fatalf("format_operator child type = %q, want -f", got)
	}
	if child.IsNamed() {
		t.Fatal("format_operator child is named")
	}
}

func TestNormalizePowerShellAssignmentOperatorTokensRestoresFileRedirectionOperator(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "program", "file_redirection_operator", ">",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "program", Visible: true, Named: true},
			{Name: "file_redirection_operator", Visible: true, Named: true},
			{Name: ">", Visible: true, Named: false},
		},
	}
	source := []byte(">")
	arena := newNodeArena(arenaClassFull)
	op := newLeafNodeInArena(arena, 2, true, 0, uint32(len(source)), Point{}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 1, true, []*Node{op}, nil, 0)

	normalizePowerShellAssignmentOperatorTokens(root, source, lang)

	if got := op.ChildCount(); got != 1 {
		t.Fatalf("file_redirection_operator child count = %d, want 1", got)
	}
	child := op.Child(0)
	if child == nil {
		t.Fatal("file_redirection_operator child = nil")
	}
	if got := child.Type(lang); got != ">" {
		t.Fatalf("file_redirection_operator child type = %q, want >", got)
	}
	if child.IsNamed() {
		t.Fatal("file_redirection_operator child is named")
	}
}

func TestNormalizePowerShellAssignmentOperatorTokensRestoresMergingRedirectionOperator(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "program", "merging_redirection_operator", "2>&1",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "program", Visible: true, Named: true},
			{Name: "merging_redirection_operator", Visible: true, Named: true},
			{Name: "2>&1", Visible: true, Named: false},
		},
	}
	source := []byte("2>&1")
	arena := newNodeArena(arenaClassFull)
	op := newLeafNodeInArena(arena, 2, true, 0, uint32(len(source)), Point{}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 1, true, []*Node{op}, nil, 0)

	normalizePowerShellAssignmentOperatorTokens(root, source, lang)

	if got := op.ChildCount(); got != 1 {
		t.Fatalf("merging_redirection_operator child count = %d, want 1", got)
	}
	child := op.Child(0)
	if child == nil {
		t.Fatal("merging_redirection_operator child = nil")
	}
	if got := child.Type(lang); got != "2>&1" {
		t.Fatalf("merging_redirection_operator child type = %q, want 2>&1", got)
	}
	if child.IsNamed() {
		t.Fatal("merging_redirection_operator child is named")
	}
}

func TestNormalizePowerShellAssignmentOperatorTokensRestoresCommandInvokationOperator(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "program", "command_invokation_operator", "&",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "program", Visible: true, Named: true},
			{Name: "command_invokation_operator", Visible: true, Named: true},
			{Name: "&", Visible: true, Named: false},
		},
	}
	source := []byte("&")
	arena := newNodeArena(arenaClassFull)
	op := newLeafNodeInArena(arena, 2, true, 0, uint32(len(source)), Point{}, Point{Column: uint32(len(source))})
	root := newParentNodeInArena(arena, 1, true, []*Node{op}, nil, 0)

	normalizePowerShellAssignmentOperatorTokens(root, source, lang)

	if got := op.ChildCount(); got != 1 {
		t.Fatalf("command_invokation_operator child count = %d, want 1", got)
	}
	child := op.Child(0)
	if child == nil {
		t.Fatal("command_invokation_operator child = nil")
	}
	if got := child.Type(lang); got != "&" {
		t.Fatalf("command_invokation_operator child type = %q, want &", got)
	}
	if child.IsNamed() {
		t.Fatal("command_invokation_operator child is named")
	}
}

func TestNormalizePowerShellPathCommandNameWrapsVariable(t *testing.T) {
	lang := &Language{
		Name: "powershell",
		SymbolNames: []string{
			"EOF", "program", "command_name", "variable", "path_command_name",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "program", Visible: true, Named: true},
			{Name: "command_name", Visible: true, Named: true},
			{Name: "variable", Visible: true, Named: true},
			{Name: "path_command_name", Visible: true, Named: true},
		},
	}

	source := []byte("& $sb")
	arena := newNodeArena(arenaClassFull)
	variable := newLeafNodeInArena(arena, 3, true, 2, uint32(len(source)), Point{Column: 2}, Point{Column: uint32(len(source))})
	commandName := newParentNodeInArena(arena, 2, true, []*Node{variable}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{commandName}, nil, 0)

	normalizePowerShellPathCommandNameVariables(root, source, lang)

	wrapped := commandName.children[0]
	if got, want := wrapped.Type(lang), "path_command_name"; got != want {
		t.Fatalf("wrapped.Type = %q, want %q", got, want)
	}
	if got, want := len(wrapped.children), 1; got != want {
		t.Fatalf("len(wrapped.children) = %d, want %d", got, want)
	}
	if got, want := wrapped.children[0].Type(lang), "variable"; got != want {
		t.Fatalf("wrapped.children[0].Type = %q, want %q", got, want)
	}
	if got, want := wrapped.startByte, uint32(2); got != want {
		t.Fatalf("wrapped.startByte = %d, want %d", got, want)
	}
	if got, want := wrapped.endByte, uint32(len(source)); got != want {
		t.Fatalf("wrapped.endByte = %d, want %d", got, want)
	}
}
