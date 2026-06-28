package grammars

import (
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func TestSwiftLineCommentsStayExtraComments(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte("// header\n//\n// body\nlet x = 1\n")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse swift comments: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("swift comment fixture has parse errors: %s", root.SExpr(lang))
	}
	if got, want := root.NamedChildCount(), 4; got != want {
		t.Fatalf("named child count = %d, want %d; tree: %s", got, want, root.SExpr(lang))
	}
	expectedCommentSpans := [][2]uint32{
		{0, 9},
		{10, 12},
		{13, 20},
	}
	for i, span := range expectedCommentSpans {
		child := root.NamedChild(i)
		if got := child.Type(lang); got != "comment" {
			t.Fatalf("named child %d type = %q, want comment; tree: %s", i, got, root.SExpr(lang))
		}
		if !child.IsExtra() {
			t.Fatalf("named child %d is not extra; tree: %s", i, root.SExpr(lang))
		}
		if got, want := child.StartByte(), span[0]; got != want {
			t.Fatalf("comment %d start = %d, want %d; tree: %s", i, got, want, root.SExpr(lang))
		}
		if got, want := child.EndByte(), span[1]; got != want {
			t.Fatalf("comment %d end = %d, want %d; tree: %s", i, got, want, root.SExpr(lang))
		}
	}
}

func TestSwiftMemberKeywordSelfAfterDotStaysNavigable(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte("let element = Element.self\nlet storage = _HashNode.Storage.self\n")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse swift member self: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("swift member self fixture has parse errors: %s", root.SExpr(lang))
	}
	sexpr := root.SExpr(lang)
	if got, want := strings.Count(sexpr, "(navigation_suffix (simple_identifier))"), 3; got != want {
		t.Fatalf("navigation suffix count = %d, want %d; tree: %s", got, want, sexpr)
	}
}

func TestSwiftImportThenClassParsesAsTopLevelDeclarations(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte("import Foundation\nclass Foo {}\n")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse swift import/class: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("swift import/class fixture has parse errors: %s", root.SExpr(lang))
	}
	if got, want := root.Type(lang), "source_file"; got != want {
		t.Fatalf("root type = %q, want %q", got, want)
	}
	if got, want := root.EndByte(), uint32(len(src)); got != want {
		t.Fatalf("root end = %d, want %d; tree: %s", got, want, root.SExpr(lang))
	}
	sexpr := root.SExpr(lang)
	for _, want := range []string{"(import_declaration", "(class_declaration"} {
		if !strings.Contains(sexpr, want) {
			t.Fatalf("missing %s in tree: %s", want, sexpr)
		}
	}
}

func TestSwiftLicenseHeaderImportThenClassParses(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte(`//
//  Foo.swift
//
//  Copyright (c) 2025 Foo Foundation (http://example.org/)
//
//  Permission is hereby granted, free of charge, to any person obtaining a copy
//  of this software and associated documentation files (the "Software"), to deal

import Foundation
class Foo {}
`)
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse swift license header: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("swift license header fixture has parse errors: %s", root.SExpr(lang))
	}
	if got, want := root.Type(lang), "source_file"; got != want {
		t.Fatalf("root type = %q, want %q", got, want)
	}
	if got, want := root.EndByte(), uint32(len(src)); got != want {
		t.Fatalf("root end = %d, want %d; tree: %s", got, want, root.SExpr(lang))
	}
	sexpr := root.SExpr(lang)
	if strings.Count(sexpr, "(comment)") < 7 {
		t.Fatalf("license header comments were not preserved as comments: %s", sexpr)
	}
	for _, want := range []string{"(import_declaration", "(class_declaration"} {
		if !strings.Contains(sexpr, want) {
			t.Fatalf("missing %s in tree: %s", want, sexpr)
		}
	}
}

// countSwiftNodeType walks the tree and counts nodes of the given type.
func countSwiftNodeType(lang *gotreesitter.Language, n *gotreesitter.Node, typ string) int {
	if n == nil {
		return 0
	}
	count := 0
	if n.Type(lang) == typ {
		count++
	}
	for i := 0; i < n.ChildCount(); i++ {
		count += countSwiftNodeType(lang, n.Child(i), typ)
	}
	return count
}

// TestSwiftComparisonInConditionRecoversFunction is the regression test for
// issue #118: a comparison operator (< / > / ==) in an if/while condition used
// to make the body brace be consumed as a trailing closure, collapsing the
// whole function into ERROR nodes with no recoverable function_declaration.
func TestSwiftComparisonInConditionRecoversFunction(t *testing.T) {
	lang := SwiftLanguage()
	cases := []struct {
		name string
		src  string
	}{
		{"if-greater", "func a() { if x > 0 { foo() } }"},
		{"if-less", "func a() { if x < 0 { foo() } }"},
		{"if-equal", "func a() { if x == 0 { foo() } }"},
		{"while-greater", "func a() { while x > 0 { foo() } }"},
		{"compound", "func a() { if x > 0 && y < 1 { foo() } }"},
		{"nested", "func a() { if x > 0 { if y < 2 { foo() } } }"},
		{"if-else", "func a() { if x > 0 { a() } else { b() } }"},
		{"class-methods", "class C {\n  func a() { if x > 0 { foo() } }\n  func b() { bar() }\n}"},
		{"struct-method", "struct S {\n  func a() { if x > 0 { foo() } }\n}"},
		{"extension-method", "extension S {\n  func a() { if x > 0 { foo() } }\n}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.src)
			parser := gotreesitter.NewParser(lang)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if root.HasError() {
				t.Fatalf("recovered tree still reports error: %s", root.SExpr(lang))
			}
			if got, want := root.EndByte(), uint32(len(src)); got != want {
				t.Fatalf("root end = %d, want %d (span not byte-faithful): %s", got, want, root.SExpr(lang))
			}
			if got := countSwiftNodeType(lang, root, "function_declaration"); got < 1 {
				t.Fatalf("function_declaration count = %d, want >= 1: %s", got, root.SExpr(lang))
			}
		})
	}
}

// TestSwiftComparisonConditionTreeIsFaithful checks that the recovered tree is
// structurally correct: a proper if_statement whose condition is the bare
// comparison_expression (the synthetic parenthesis used during recovery is
// stripped) with byte-faithful spans.
func TestSwiftComparisonConditionTreeIsFaithful(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte("func a() { if x > 0 { foo() } }")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	root := tree.RootNode()
	sexpr := root.SExpr(lang)
	for _, want := range []string{"(function_declaration", "(if_statement", "(comparison_expression"} {
		if !strings.Contains(sexpr, want) {
			t.Fatalf("missing %s in recovered tree: %s", want, sexpr)
		}
	}
	// The synthetic parenthesis must not survive as a tuple_expression condition.
	if countSwiftNodeType(lang, root, "tuple_expression") != 0 {
		t.Fatalf("synthetic parenthesis leaked as tuple_expression: %s", sexpr)
	}
	if countSwiftNodeType(lang, root, "lambda_literal") != 0 {
		t.Fatalf("if body misparsed as trailing-closure lambda_literal: %s", sexpr)
	}
}

// TestSwiftNormalTrailingClosureUnaffected guards against the recovery pass
// disturbing a legitimate trailing closure (which must stay a lambda_literal).
func TestSwiftNormalTrailingClosureUnaffected(t *testing.T) {
	lang := SwiftLanguage()
	src := []byte("func a() { items.map { x in x } }")
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer tree.Release()
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("clean trailing-closure source reported error: %s", root.SExpr(lang))
	}
	if countSwiftNodeType(lang, root, "lambda_literal") != 1 {
		t.Fatalf("expected exactly one lambda_literal: %s", root.SExpr(lang))
	}
}
