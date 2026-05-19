package gotreesitter_test

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestExtractImportsGo(t *testing.T) {
	source := []byte(`package main

import (
	alias "example.com/aliased"
	_ "example.com/sideeffect"
	. "example.com/dot"
	"example.com/plain"
)
`)
	parser := gotreesitter.NewParser(grammars.GoLanguage())
	tree, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Release()

	refs := gotreesitter.ExtractImports(tree)
	if got, want := len(refs), 5; got != want {
		t.Fatalf("ExtractImports len = %d, want %d: %#v", got, want, refs)
	}
	if refs[0].Kind != "package" || refs[0].Name != "main" {
		t.Fatalf("package ref = %#v, want main package", refs[0])
	}
	assertImportRef(t, refs[1], "go", "import", "example.com/aliased", "aliased", "alias")
	assertImportRef(t, refs[2], "go", "import", "example.com/sideeffect", "sideeffect", "_")
	assertImportRef(t, refs[3], "go", "import", "example.com/dot", "dot", ".")
	assertImportRef(t, refs[4], "go", "import", "example.com/plain", "plain", "")
}

func TestExtractImportsJava(t *testing.T) {
	source := []byte(`package example.app;

import java.util.List;
import static java.util.Collections.*;
`)
	parser := gotreesitter.NewParser(grammars.JavaLanguage())
	tree, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Release()

	refs := gotreesitter.ExtractImports(tree)
	if got, want := len(refs), 3; got != want {
		t.Fatalf("ExtractImports len = %d, want %d: %#v", got, want, refs)
	}
	if refs[0].Kind != "package" || refs[0].Path != "example.app" {
		t.Fatalf("package ref = %#v, want example.app package", refs[0])
	}
	assertImportRef(t, refs[1], "java", "import", "java.util.List", "List", "")
	assertImportRef(t, refs[2], "java", "import", "java.util.Collections", "Collections", "")
	if !refs[2].Static || !refs[2].Wildcard {
		t.Fatalf("static wildcard ref = %#v, want static wildcard", refs[2])
	}
}

func TestExtractImportsPython(t *testing.T) {
	source := []byte(`import os, sys as system
from ..pkg.sub import name as alias, other
from pkg import *
`)
	parser := gotreesitter.NewParser(grammars.PythonLanguage())
	tree, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Release()

	refs := gotreesitter.ExtractImports(tree)
	if got, want := len(refs), 5; got != want {
		t.Fatalf("ExtractImports len = %d, want %d: %#v", got, want, refs)
	}
	assertImportRef(t, refs[0], "python", "import", "os", "os", "")
	assertImportRef(t, refs[1], "python", "import", "sys", "sys", "system")
	assertImportRef(t, refs[2], "python", "from_import", "pkg.sub.name", "name", "alias")
	if refs[2].From != "pkg.sub" || refs[2].Relative != 2 {
		t.Fatalf("from ref = %#v, want relative pkg.sub", refs[2])
	}
	assertImportRef(t, refs[3], "python", "from_import", "pkg.sub.other", "other", "")
	assertImportRef(t, refs[4], "python", "from_import", "pkg", "*", "")
	if !refs[4].Wildcard {
		t.Fatalf("wildcard ref = %#v, want wildcard", refs[4])
	}
}

func TestExtractImportsStarlark(t *testing.T) {
	source := []byte(`load("@rules_python//python:defs.bzl", "py_library", py_binary_alias = "py_binary")

py_library(name = "lib")
`)
	parser := gotreesitter.NewParser(grammars.StarlarkLanguage())
	tree, err := parser.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Release()

	refs := gotreesitter.ExtractImports(tree)
	if got, want := len(refs), 2; got != want {
		t.Fatalf("ExtractImports len = %d, want %d: %#v", got, want, refs)
	}
	assertImportRef(t, refs[0], "starlark", "load", "@rules_python//python:defs.bzl:py_library", "py_library", "")
	if refs[0].From != "@rules_python//python:defs.bzl" {
		t.Fatalf("load ref = %#v, want module in From", refs[0])
	}
	assertImportRef(t, refs[1], "starlark", "load", "@rules_python//python:defs.bzl:py_binary", "py_binary", "py_binary_alias")
}

func assertImportRef(t *testing.T, ref gotreesitter.ImportRef, lang, kind, path, name, alias string) {
	t.Helper()
	if ref.Lang != lang || ref.Kind != kind || ref.Path != path || ref.Name != name || ref.Alias != alias {
		t.Fatalf("ref = %#v, want lang=%s kind=%s path=%s name=%s alias=%s", ref, lang, kind, path, name, alias)
	}
}
