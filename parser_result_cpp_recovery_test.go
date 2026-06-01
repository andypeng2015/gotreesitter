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
	if got, want := root.ChildCount(), 2; got != want {
		t.Fatalf("root child count = %d, want %d\n%s", got, want, root.SExpr(lang))
	}
}
