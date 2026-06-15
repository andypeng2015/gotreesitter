package gotreesitter_test

import (
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// countCSharpDecls walks the tree and counts class/method declaration nodes.
func countCSharpDecls(t *testing.T, lang *gotreesitter.Language, root *gotreesitter.Node) (classes, methods int) {
	t.Helper()
	var walk func(n *gotreesitter.Node)
	walk = func(n *gotreesitter.Node) {
		if n == nil {
			return
		}
		switch n.Type(lang) {
		case "class_declaration":
			classes++
		case "method_declaration":
			methods++
		}
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return classes, methods
}

// TestCSharpNamespaceWithInternalErrorRecoversDeclarations is the regression
// test for issue #115: a namespace whose body has an internal parse error used
// to collapse the whole file into a single ERROR node with zero recoverable
// declarations. Recovery should now surface the namespace's type declarations.
func TestCSharpNamespaceWithInternalErrorRecoversDeclarations(t *testing.T) {
	lang := grammars.CSharpLanguage()

	// A namespace containing a class with a deliberately broken member among
	// well-formed ones. The braces are balanced so the namespace span resolves,
	// but the body does not parse cleanly end-to-end.
	src := []byte(`namespace N
{
    public class C
    {
        public void A() {}
        public int @@@ Broken {}
        public int B() { return 0; }
    }
}`)

	tree, err := gotreesitter.NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	classes, methods := countCSharpDecls(t, lang, root)
	if classes < 1 {
		t.Fatalf("recovered classes = %d, want >= 1 (root type %s)", classes, root.Type(lang))
	}
	if methods < 1 {
		t.Fatalf("recovered methods = %d, want >= 1 (root type %s)", methods, root.Type(lang))
	}
}

// TestCSharpNamespaceWithCharLiteralBracesRecovers exercises the trivia-aware
// brace matcher: char-literal braces ('{' / '}') in a member body must not
// truncate the namespace span during recovery.
func TestCSharpNamespaceWithCharLiteralBracesRecovers(t *testing.T) {
	lang := grammars.CSharpLanguage()

	src := []byte(`namespace N
{
    public class C
    {
        char Open = '{';
        char Close = '}';
        public int @@@ Broken {}
        public bool IsBrace(char c) { return c == '{' || c == '}'; }
    }
}`)

	tree, err := gotreesitter.NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if got, want := root.EndByte(), uint32(len(src)); got != want {
		t.Fatalf("root end = %d, want %d (span truncated by char-literal braces)", got, want)
	}
	classes, _ := countCSharpDecls(t, lang, root)
	if classes < 1 {
		t.Fatalf("recovered classes = %d, want >= 1 (root type %s)", classes, root.Type(lang))
	}
}

// TestCSharpNamespaceWithVerbatimStringBracesRecovers ensures a verbatim string
// (@"...") containing braces and "" escapes does not throw off the trivia-aware
// brace matcher during recovery: the braces inside the string must be ignored so
// the namespace/class span is not truncated.
func TestCSharpNamespaceWithVerbatimStringBracesRecovers(t *testing.T) {
	lang := grammars.CSharpLanguage()

	src := []byte(`namespace N
{
    public class C
    {
        string S = @"obj }} with ""q"" and { brace";
        public int @@@ Broken {}
        public int B() { return 0; }
    }
}`)

	tree, err := gotreesitter.NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if got, want := root.EndByte(), uint32(len(src)); got != want {
		t.Fatalf("root end = %d, want %d (span truncated by verbatim-string braces)", got, want)
	}
	classes, methods := countCSharpDecls(t, lang, root)
	if classes < 1 {
		t.Fatalf("recovered classes = %d, want >= 1 (root type %s)", classes, root.Type(lang))
	}
	if methods < 1 {
		t.Fatalf("recovered methods = %d, want >= 1 (root type %s)", methods, root.Type(lang))
	}
}

// TestCSharpCleanNamespaceUnaffected guards against the recovery path altering a
// namespace that already parses cleanly.
func TestCSharpCleanNamespaceUnaffected(t *testing.T) {
	lang := grammars.CSharpLanguage()

	src := []byte(`namespace N
{
    public class C
    {
        public void A() {}
        public int B() { return 0; }
    }
    public class D
    {
        public void E() {}
    }
}`)

	tree, err := gotreesitter.NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("clean namespace reported an error")
	}
	if got, want := root.Type(lang), "compilation_unit"; got != want {
		t.Fatalf("root type = %s, want %s", got, want)
	}
	classes, methods := countCSharpDecls(t, lang, root)
	if classes != 2 {
		t.Fatalf("classes = %d, want 2", classes)
	}
	if methods != 3 {
		t.Fatalf("methods = %d, want 3", methods)
	}
}
