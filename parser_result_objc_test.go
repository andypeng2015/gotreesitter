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

func TestObjcProtocolArgumentTypeUsesTypeIdentifier(t *testing.T) {
	src := []byte("@interface CallbackClient : NSObject <ClientProtocol>\n@end\n")
	lang := grammars.ObjcLanguage()
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil || tree == nil || tree.RootNode() == nil {
		t.Fatalf("parse failed: tree=%v err=%v", tree, err)
	}
	defer tree.Release()

	typeName := firstObjcNodeByTypeAndText(tree.RootNode(), lang, src, "type_name", "ClientProtocol")
	if typeName == nil {
		t.Fatalf("missing protocol type_name: %s", tree.RootNode().SExpr(lang))
	}
	if got, want := typeName.Child(0).Type(lang), "type_identifier"; got != want {
		t.Fatalf("protocol argument child = %q, want %q; tree=%s", got, want, tree.RootNode().SExpr(lang))
	}
}

func TestObjcSizeofTypeIdentifierOperandMatchesOracleShape(t *testing.T) {
	src := []byte("void f(){ int a = sizeof(GCInfo); int b = sizeof(int); }\n")
	lang := grammars.ObjcLanguage()
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil || tree == nil || tree.RootNode() == nil {
		t.Fatalf("parse failed: tree=%v err=%v", tree, err)
	}
	defer tree.Release()

	unknown := firstObjcNodeByTypeAndText(tree.RootNode(), lang, src, "sizeof_expression", "sizeof(GCInfo)")
	if unknown == nil {
		t.Fatalf("missing sizeof(GCInfo): %s", tree.RootNode().SExpr(lang))
	}
	if got, want := unknown.ChildCount(), 2; got != want {
		t.Fatalf("sizeof(GCInfo) child count = %d, want %d; tree=%s", got, want, tree.RootNode().SExpr(lang))
	}
	paren := unknown.Child(1)
	if got, want := paren.Type(lang), "parenthesized_expression"; got != want {
		t.Fatalf("sizeof(GCInfo) child 1 = %q, want %q; tree=%s", got, want, tree.RootNode().SExpr(lang))
	}
	if got, want := paren.Child(1).Type(lang), "identifier"; got != want {
		t.Fatalf("sizeof(GCInfo) operand = %q, want %q; tree=%s", got, want, tree.RootNode().SExpr(lang))
	}

	primitive := firstObjcNodeByTypeAndText(tree.RootNode(), lang, src, "sizeof_expression", "sizeof(int)")
	if primitive == nil {
		t.Fatalf("missing sizeof(int): %s", tree.RootNode().SExpr(lang))
	}
	if got, want := primitive.ChildCount(), 4; got != want {
		t.Fatalf("sizeof(int) child count = %d, want %d; tree=%s", got, want, tree.RootNode().SExpr(lang))
	}
	if got, want := primitive.Child(2).Type(lang), "type_descriptor"; got != want {
		t.Fatalf("sizeof(int) child 2 = %q, want %q; tree=%s", got, want, tree.RootNode().SExpr(lang))
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

func firstObjcNodeByTypeAndText(n *gts.Node, lang *gts.Language, src []byte, typ, text string) *gts.Node {
	if n == nil {
		return nil
	}
	if n.Type(lang) == typ && string(n.Text(src)) == text {
		return n
	}
	for i := 0; i < n.ChildCount(); i++ {
		if found := firstObjcNodeByTypeAndText(n.Child(i), lang, src, typ, text); found != nil {
			return found
		}
	}
	return nil
}
