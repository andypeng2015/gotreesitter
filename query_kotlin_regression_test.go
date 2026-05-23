package gotreesitter_test

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestKotlinNestedImportQueryMatchesRepeatedHeaders(t *testing.T) {
	src := []byte("package com.example\n\nimport foo.bar.Baz\nimport foo.qux.*\nfun main() {}\n")
	lang := grammars.KotlinLanguage()
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer tree.Release()
	if tree.RootNode().HasError() {
		t.Fatalf("root has error:\n%s", tree.RootNode().SExpr(lang))
	}

	q, err := gotreesitter.NewQuery(`
		(source_file
			(import_list
				(import_header (identifier) @imp (wildcard_import)? @is_star)
			)
		)
	`, lang)
	if err != nil {
		t.Fatalf("NewQuery failed: %v", err)
	}

	cursor := q.Exec(tree.RootNode(), lang, src)
	var got []string
	var wildcard bool
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, capture := range match.Captures {
			switch capture.Name {
			case "imp":
				got = append(got, capture.Text(src))
			case "is_star":
				wildcard = true
			}
		}
	}

	want := []string{"foo.bar.Baz", "foo.qux"}
	if len(got) != len(want) {
		t.Fatalf("captures = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("captures = %v, want %v", got, want)
		}
	}
	if !wildcard {
		t.Fatal("expected wildcard_import capture")
	}
}
