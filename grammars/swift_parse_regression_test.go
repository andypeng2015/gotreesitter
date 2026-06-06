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
