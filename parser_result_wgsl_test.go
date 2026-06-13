package gotreesitter

import "testing"

func TestNormalizeWGSLEmptyReturnSemicolonRecovery(t *testing.T) {
	lang := testWGSLLanguage()
	arena := newNodeArena(arenaClassFull)
	open := newLeafNodeInArena(arena, 3, false, 0, 1, Point{}, Point{Column: 1})
	inc := newLeafNodeInArena(arena, 5, true, 4, 14, Point{Column: 4}, Point{Column: 14})
	missingReturn := newLeafNodeInArena(arena, 7, false, 14, 14, Point{Column: 14}, Point{Column: 14})
	missingReturn.setMissing(true)
	emptyReturn := newParentNodeInArena(arena, 6, true, []*Node{missingReturn}, nil, 0)
	semi := newLeafNodeInArena(arena, 8, false, 14, 15, Point{Column: 14}, Point{Column: 15})
	close := newLeafNodeInArena(arena, 4, false, 18, 19, Point{Column: 18}, Point{Column: 19})
	block := newParentNodeInArena(arena, 2, true, []*Node{open, inc, emptyReturn, semi, close}, nil, 0)

	normalizeWGSLCompatibility(block, lang)

	if got, want := block.ChildCount(), 4; got != want {
		t.Fatalf("compound child count = %d, want %d", got, want)
	}
	err := block.Child(2)
	if err == nil {
		t.Fatal("recovery child = nil")
	}
	if got, want := err.Type(lang), "ERROR"; got != want {
		t.Fatalf("recovery child type = %q, want %q", got, want)
	}
	if got, want := err.StartByte(), uint32(14); got != want {
		t.Fatalf("ERROR start = %d, want %d", got, want)
	}
	if got, want := err.EndByte(), uint32(15); got != want {
		t.Fatalf("ERROR end = %d, want %d", got, want)
	}
	if got, want := err.ChildCount(), 1; got != want {
		t.Fatalf("ERROR child count = %d, want %d", got, want)
	}
	if child := err.Child(0); child == nil || child.Type(lang) != ";" {
		t.Fatalf("ERROR child = %#v, want semicolon", child)
	}
	if !err.HasError() {
		t.Fatal("ERROR node should carry has_error")
	}
}

func TestNormalizeWGSLArgumentListErrorWrapper(t *testing.T) {
	lang := testWGSLLanguage()
	arena := newNodeArena(arenaClassFull)
	open := newLeafNodeInArena(arena, 11, false, 0, 1, Point{}, Point{Column: 1})
	x := newLeafNodeInArena(arena, 16, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	comma := newLeafNodeInArena(arena, 17, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	y := newLeafNodeInArena(arena, 16, true, 4, 5, Point{Column: 4}, Point{Column: 5})
	err := newParentNodeInArena(arena, errorSymbol, true, []*Node{x, comma, y}, nil, 0)
	err.setHasError(true)
	close := newLeafNodeInArena(arena, 12, false, 5, 6, Point{Column: 5}, Point{Column: 6})
	args := newParentNodeInArena(arena, 10, true, []*Node{open, err, close}, nil, 0)
	args.setHasError(true)
	call := newParentNodeInArena(arena, 9, true, []*Node{
		newLeafNodeInArena(arena, 15, false, 0, 0, Point{}, Point{}),
		args,
	}, nil, 0)
	call.setHasError(true)

	normalizeWGSLCompatibility(call, lang)

	if got, want := args.ChildCount(), 5; got != want {
		t.Fatalf("argument child count = %d, want %d", got, want)
	}
	for i, want := range []string{"(", "identifier", ",", "identifier", ")"} {
		if got := args.Child(i).Type(lang); got != want {
			t.Fatalf("argument child %d type = %q, want %q", i, got, want)
		}
	}
	if args.HasError() || call.HasError() {
		t.Fatalf("repaired argument list should be clean: args=%v call=%v", args.HasError(), call.HasError())
	}
}

func TestNormalizeWGSLTrailingArgumentMissingIdentifier(t *testing.T) {
	lang := testWGSLLanguage()
	arena := newNodeArena(arenaClassFull)
	open := newLeafNodeInArena(arena, 11, false, 0, 1, Point{}, Point{Column: 1})
	value := newLeafNodeInArena(arena, 16, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	comma := newLeafNodeInArena(arena, 17, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	missing := newLeafNodeInArena(arena, 16, true, 3, 3, Point{Column: 3}, Point{Column: 3})
	missing.setMissing(true)
	close := newLeafNodeInArena(arena, 12, false, 6, 7, Point{Column: 6}, Point{Column: 7})
	args := newParentNodeInArena(arena, 10, true, []*Node{open, value, comma, missing, close}, nil, 0)
	args.setHasError(true)

	normalizeWGSLCompatibility(args, lang)

	if got, want := args.ChildCount(), 4; got != want {
		t.Fatalf("argument child count = %d, want %d", got, want)
	}
	for i, want := range []string{"(", "identifier", ",", ")"} {
		if got := args.Child(i).Type(lang); got != want {
			t.Fatalf("argument child %d type = %q, want %q", i, got, want)
		}
	}
	if args.HasError() {
		t.Fatal("argument list with removed missing identifier should be clean")
	}
}

func TestNormalizeWGSLAtomicArrayRecovery(t *testing.T) {
	lang := testWGSLLanguage()
	arena := newNodeArena(arenaClassFull)
	array := newLeafNodeInArena(arena, 20, false, 0, 5, Point{}, Point{Column: 5})
	open := newLeafNodeInArena(arena, 13, false, 5, 6, Point{Column: 5}, Point{Column: 6})
	atomicIdent := newLeafNodeInArena(arena, 16, true, 6, 12, Point{Column: 6}, Point{Column: 12})
	atomicType := newParentNodeInArena(arena, 14, true, []*Node{atomicIdent}, nil, 0)
	err := newParentNodeInArena(arena, errorSymbol, true, []*Node{open, atomicType}, nil, 0)
	err.setHasError(true)
	nestedOpen := newLeafNodeInArena(arena, 13, false, 12, 13, Point{Column: 12}, Point{Column: 13})
	u32 := newLeafNodeInArena(arena, 21, false, 13, 16, Point{Column: 13}, Point{Column: 16})
	u32Type := newParentNodeInArena(arena, 14, true, []*Node{u32}, nil, 0)
	close := newLeafNodeInArena(arena, 19, false, 16, 17, Point{Column: 16}, Point{Column: 17})
	typ := newParentNodeInArena(arena, 14, true, []*Node{array, err, nestedOpen, u32Type, close}, nil, 0)
	typ.setHasError(true)

	normalizeWGSLCompatibility(typ, lang)

	if got, want := typ.ChildCount(), 5; got != want {
		t.Fatalf("type_declaration child count = %d, want %d", got, want)
	}
	for i, want := range []string{"array", "<", "ERROR", "type_declaration", ">"} {
		if got := typ.Child(i).Type(lang); got != want {
			t.Fatalf("type child %d type = %q, want %q", i, got, want)
		}
	}
	recovered := typ.Child(2)
	if got, want := recovered.StartByte(), uint32(6); got != want {
		t.Fatalf("ERROR start = %d, want %d", got, want)
	}
	if got, want := recovered.EndByte(), uint32(13); got != want {
		t.Fatalf("ERROR end = %d, want %d", got, want)
	}
	if !typ.HasError() || !recovered.HasError() {
		t.Fatalf("type_declaration and ERROR should carry has_error: type=%v error=%v", typ.HasError(), recovered.HasError())
	}
	if !recovered.IsExtra() {
		t.Fatal("C-compatible atomic recovery ERROR should be extra")
	}
	if typ.Child(1).HasError() || recovered.Child(0).HasError() || recovered.Child(1).HasError() {
		t.Fatal("C-compatible atomic recovery leaves/token children should not carry has_error")
	}
}

func testWGSLLanguage() *Language {
	return &Language{
		Name: "wgsl",
		SymbolNames: []string{
			"EOF",
			"source_file",
			"compound_statement",
			"{",
			"}",
			"increment_statement",
			"return_statement",
			"return",
			";",
			"type_constructor_or_function_call_expression",
			"argument_list_expression",
			"(",
			")",
			"<",
			"type_declaration",
			"vec4",
			"identifier",
			",",
			"ERROR",
			">",
			"array",
			"u32",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "compound_statement", Visible: true, Named: true},
			{Name: "{", Visible: true, Named: false},
			{Name: "}", Visible: true, Named: false},
			{Name: "increment_statement", Visible: true, Named: true},
			{Name: "return_statement", Visible: true, Named: true},
			{Name: "return", Visible: true, Named: false},
			{Name: ";", Visible: true, Named: false},
			{Name: "type_constructor_or_function_call_expression", Visible: true, Named: true},
			{Name: "argument_list_expression", Visible: true, Named: true},
			{Name: "(", Visible: true, Named: false},
			{Name: ")", Visible: true, Named: false},
			{Name: "<", Visible: true, Named: false},
			{Name: "type_declaration", Visible: true, Named: true},
			{Name: "vec4", Visible: true, Named: false},
			{Name: "identifier", Visible: true, Named: true},
			{Name: ",", Visible: true, Named: false},
			{Name: "ERROR", Visible: true, Named: true},
			{Name: ">", Visible: true, Named: false},
			{Name: "array", Visible: true, Named: false},
			{Name: "u32", Visible: true, Named: false},
		},
	}
}
