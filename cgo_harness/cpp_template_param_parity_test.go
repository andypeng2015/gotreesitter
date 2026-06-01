//go:build cgo && treesitter_c_parity

package cgoharness

import "testing"

func TestCppTemplateTypeParameterParity(t *testing.T) {
	src := []byte("void f(Vector<Rule> *elements) {}\n")
	tc := parityCase{name: "cpp", source: string(src)}
	runParityCase(t, tc, "template-type-parameter", src)
}

func TestCppCollapsedKeywordCompatibilityParity(t *testing.T) {
	src := []byte(`namespace tree_sitter {
namespace rules {
struct Rule {
  Rule(Rule &&other) noexcept;
  bool operator==(const Rule &other) const;
};
Rule::Rule(Rule &&other) noexcept {}
bool Rule::operator==(const Rule &other) const { return true; }
void f() {
  match([&](auto rule) { return; }, [=](auto rule) { return; });
}
}
}
`)
	tc := parityCase{name: "cpp", source: string(src)}
	runParityCase(t, tc, "collapsed-keyword-compatibility", src)
}

func TestCppMalformedClassFunctionDefinitionRecoveryParity(t *testing.T) {
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
	tc := parityCase{name: "cpp", source: string(src)}
	runParityCase(t, tc, "malformed-class-function-definition-recovery", src)
}
