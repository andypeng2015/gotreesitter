package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestObjcMethodPointerTypeUsesTypeIdentifier(t *testing.T) {
	src := []byte("@interface Tester : NSObject\n+ (void) connectWithPorts: (NSArray*)portArray;\n- (NSUInteger) count;\n@end\n")
	lang := grammars.ObjcLanguage()
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil || tree == nil || tree.RootNode() == nil {
		t.Fatalf("parse failed: tree=%v err=%v", tree, err)
	}
	defer tree.Release()

	sexpr := tree.RootNode().SExpr(lang)
	if !strings.Contains(sexpr, "(type_name (type_identifier) (abstract_pointer_declarator))") {
		t.Fatalf("method pointer type was not normalized to type_identifier: %s", sexpr)
	}
	if !strings.Contains(sexpr, "(method_type (type_name (type_identifier)))") {
		t.Fatalf("method scalar type was not normalized to type_identifier: %s", sexpr)
	}
}

func TestObjcAtStringLiteralIsFlattened(t *testing.T) {
	src := []byte("int main() { NSLog(@\"one\"); }\n")
	lang := grammars.ObjcLanguage()
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil || tree == nil || tree.RootNode() == nil {
		t.Fatalf("parse failed: tree=%v err=%v", tree, err)
	}
	defer tree.Release()

	sexpr := tree.RootNode().SExpr(lang)
	if strings.Contains(sexpr, "at_expression") {
		t.Fatalf("ObjC string literal still wrapped in at_expression: %s", sexpr)
	}
	lit := firstObjcNodeByType(tree.RootNode(), lang, "string_literal")
	if lit == nil {
		t.Fatalf("missing string_literal: %s", sexpr)
	}
	if got, want := lit.ChildCount(), 4; got != want {
		t.Fatalf("string_literal child count = %d, want %d; tree=%s", got, want, sexpr)
	}
	if got := lit.Child(0).Type(lang); got != "@" {
		t.Fatalf("string_literal child 0 = %q, want @; tree=%s", got, sexpr)
	}
}

func TestObjcAtConcatenatedStringLiteralIsFlattened(t *testing.T) {
	src := []byte("int main() { NSLog(@\"one\" @\"two\"); }\n")
	lang := grammars.ObjcLanguage()
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil || tree == nil || tree.RootNode() == nil {
		t.Fatalf("parse failed: tree=%v err=%v", tree, err)
	}
	defer tree.Release()

	sexpr := tree.RootNode().SExpr(lang)
	if strings.Contains(sexpr, "at_expression") {
		t.Fatalf("ObjC concatenated string still wrapped in at_expression: %s", sexpr)
	}
	concat := firstObjcNodeByType(tree.RootNode(), lang, "concatenated_string")
	if concat == nil {
		t.Fatalf("missing concatenated_string: %s", sexpr)
	}
	if got, want := concat.ChildCount(), 2; got != want {
		t.Fatalf("concatenated_string child count = %d, want %d; tree=%s", got, want, sexpr)
	}
	first := concat.Child(0)
	if got, want := first.Type(lang), "string_literal"; got != want {
		t.Fatalf("first concat child = %q, want %q; tree=%s", got, want, sexpr)
	}
	if got := first.Child(0).Type(lang); got != "@" {
		t.Fatalf("first string_literal child 0 = %q, want @; tree=%s", got, sexpr)
	}
	if got, want := first.StartByte(), uint32(19); got != want {
		t.Fatalf("first string_literal start = %d, want %d; tree=%s", got, want, sexpr)
	}
}

func firstObjcNodeByType(n *gts.Node, lang *gts.Language, typ string) *gts.Node {
	if n == nil {
		return nil
	}
	if n.Type(lang) == typ {
		return n
	}
	for i := 0; i < n.ChildCount(); i++ {
		if found := firstObjcNodeByType(n.Child(i), lang, typ); found != nil {
			return found
		}
	}
	return nil
}
