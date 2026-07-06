package gotreesitter_test

import (
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestCppMalformedClassFunctionDefinitionRecovery(t *testing.T) {
	src := []byte(`int main() {
  a<T>();
  // <- function

  a::b();
  // ^ function

  a::b<C, D>();
  // ^ function

  this->b<C, D>();
  //    ^ function

  auto x = y;
  // <- type

  vector<T> a;
  // <- type

  std::vector<T> a;
  //   ^ type
}

class C : D{
  A();
  // <- function

  void efg() {
    // ^ function
  }
}

void A::b() {
  //    ^ function
}
`)
	lang := grammars.CppLanguage()
	tree, err := gts.NewParser(lang).ParseWithTokenSource(src, grammars.NewCTokenSourceOrEOF(src, lang))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer tree.Release()
	root := tree.RootNode()
	if got, want := root.Type(lang), "translation_unit"; got != want {
		t.Fatalf("root type = %q, want %q\n%s", got, want, root.SExpr(lang))
	}
	if !root.HasError() {
		t.Fatalf("root.HasError = false, want true")
	}
	// The C oracle (tree-sitter's cost-based error-recovery competition,
	// ts_parser__recover) folds the "void" token following the malformed
	// class body into the *next* declarator's qualified_identifier, fusing
	// the malformed class and `void A::b() {}` into a single 2-child
	// function_definition with a nested ERROR around the misplaced "A".
	// That fold is an artifact of the faithful C recovery-cost port
	// (parser_recover_c.go / errorCostCompetitionEnabled). cpp is not
	// gated on it (its external scanner lacks precise ExternalLexStates —
	// see DiagnoseCRecoveryGate — and, empirically, broadly enabling that
	// port for cpp regresses oracle agreement on a 300-file LLVM corpus
	// walk: 160/300 -> 142/299 with 26 new clean->error false positives,
	// so it is intentionally NOT gated on here). Without that mechanism,
	// gotreesitter's generic opportunistic top-level resync
	// (tryResyncErrorRecoveryMode) instead localizes the damage to a single
	// ERROR node spanning exactly the malformed class body and correctly
	// keeps `int main() {...}` and `void A::b() {...}` as clean, separate,
	// well-formed top-level siblings: 3 children, zero content loss, and
	// (checked above) HasError=true.
	if got, want := root.ChildCount(), 3; got != want {
		t.Fatalf("root child count = %d, want %d\n%s", got, want, root.SExpr(lang))
	}
	if errNode := root.Child(1); errNode.Type(lang) != "ERROR" {
		t.Fatalf("root.Child(1) = %q, want ERROR\n%s", errNode.Type(lang), root.SExpr(lang))
	} else if got, want := errNode.StartByte(), uint32(234); got != want {
		t.Fatalf("ERROR node start = %d, want %d (start of malformed class body)", got, want)
	} else if got, want := errNode.EndByte(), uint32(310); got != want {
		t.Fatalf("ERROR node end = %d, want %d (end of malformed class body)", got, want)
	}
	if got, want := root.Child(2).Type(lang), "function_definition"; got != want {
		t.Fatalf("root.Child(2) = %q, want %q (clean void A::b() {...})\n%s", got, want, root.SExpr(lang))
	} else if root.Child(2).HasError() {
		t.Fatalf("root.Child(2) (void A::b() {...}) unexpectedly has an error\n%s", root.SExpr(lang))
	}
}
