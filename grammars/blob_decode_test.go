package grammars_test

import (
	"testing"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// TestDecodeLanguageBlob_Roundtrip exercises the public pathway a consumer
// package uses: fetch blob bytes (here from the built-in store, but in prod
// typically from the consumer's own go:embed), decode into a *Language, and
// verify the result is usable.
func TestDecodeLanguageBlob_Roundtrip(t *testing.T) {
	blob := grammars.BlobByName("go")
	if len(blob) == 0 {
		t.Fatal("precondition: BlobByName(go) returned empty blob")
	}

	lang, err := grammars.DecodeLanguageBlob(blob)
	if err != nil {
		t.Fatalf("DecodeLanguageBlob: unexpected error: %v", err)
	}
	if lang == nil {
		t.Fatal("DecodeLanguageBlob: returned nil language without error")
	}
	if lang.SymbolCount == 0 {
		t.Error("decoded language has zero SymbolCount; blob decode did not populate grammar tables")
	}
	if lang.TokenCount == 0 {
		t.Error("decoded language has zero TokenCount; blob decode did not populate grammar tables")
	}

	// The decoded Language must be usable for parsing through the public API.
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse([]byte("package main\n"))
	if err != nil {
		t.Fatalf("parser.Parse returned error: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parser produced no tree for minimal go source")
	}
	if tree.RootNode().HasError() {
		t.Errorf("parser reported error on minimal go source: %s", tree.RootNode().SExpr(lang))
	}
}

// TestDecodeLanguageBlob_Garbage ensures the decoder rejects non-blob input
// with an error instead of panicking — consumers need a predictable failure
// mode for shipping their own embedded blobs.
func TestDecodeLanguageBlob_Garbage(t *testing.T) {
	lang, err := grammars.DecodeLanguageBlob([]byte("not a grammar blob"))
	if err == nil {
		t.Fatalf("DecodeLanguageBlob(garbage): expected error, got lang=%v", lang)
	}
	if lang != nil {
		t.Errorf("DecodeLanguageBlob(garbage): expected nil language on error, got non-nil")
	}
}

// TestDecodeLanguageBlob_Empty verifies nil/empty input is rejected cleanly.
func TestDecodeLanguageBlob_Empty(t *testing.T) {
	if _, err := grammars.DecodeLanguageBlob(nil); err == nil {
		t.Error("DecodeLanguageBlob(nil): expected error, got nil")
	}
	if _, err := grammars.DecodeLanguageBlob([]byte{}); err == nil {
		t.Error("DecodeLanguageBlob(empty): expected error, got nil")
	}
}
