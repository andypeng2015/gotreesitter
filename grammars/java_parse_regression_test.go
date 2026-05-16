//go:build !grammar_subset || grammar_subset_java

package grammars

import (
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func findFirstNamedDescendant(node *gotreesitter.Node, lang *gotreesitter.Language, typ string) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	if node.IsNamed() && node.Type(lang) == typ {
		return node
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		if found := findFirstNamedDescendant(node.NamedChild(i), lang, typ); found != nil {
			return found
		}
	}
	return nil
}

func assertMainStringArrayShape(t *testing.T, tree *gotreesitter.Tree, lang *gotreesitter.Language, src []byte) {
	t.Helper()

	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}

	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("expected parse without syntax errors, got sexpr: %s", root.SExpr(lang))
	}
	if root.NamedChildCount() != 2 {
		t.Fatalf("expected root to have 2 named children, got %d: %s", root.NamedChildCount(), root.SExpr(lang))
	}
	if got := root.NamedChild(0).Type(lang); got != "package_declaration" {
		t.Fatalf("root child[0] = %q, want package_declaration", got)
	}
	if got := root.NamedChild(1).Type(lang); got != "class_declaration" {
		t.Fatalf("root child[1] = %q, want class_declaration", got)
	}

	methodDecl := findFirstNamedDescendant(root, lang, "method_declaration")
	if methodDecl == nil {
		t.Fatalf("no method_declaration in parse tree: %s", root.SExpr(lang))
	}
	nameNode := methodDecl.ChildByFieldName("name", lang)
	if nameNode == nil || nameNode.Text(src) != "main" {
		got := "<nil>"
		if nameNode != nil {
			got = nameNode.Text(src)
		}
		t.Fatalf("method name = %q, want %q", got, "main")
	}

	params := findFirstNamedDescendant(methodDecl, lang, "formal_parameters")
	if params == nil {
		t.Fatalf("method_declaration missing formal_parameters: %s", methodDecl.SExpr(lang))
	}
	paramText := strings.Join(strings.Fields(params.Text(src)), "")
	if !strings.Contains(paramText, "String[]args") {
		t.Fatalf("formal_parameters = %q, want to contain String[]args", params.Text(src))
	}

	invocation := findFirstNamedDescendant(methodDecl, lang, "method_invocation")
	if invocation == nil {
		t.Fatalf("method_declaration missing method_invocation: %s", methodDecl.SExpr(lang))
	}
	if !strings.Contains(invocation.Text(src), "System.out.println") {
		t.Fatalf("method_invocation text = %q, want to contain System.out.println", invocation.Text(src))
	}
}

func TestJavaParseMainStringArrayRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`package com.example;

public class App {
    public static void main(String[] args) {
        System.out.println("hello");
    }
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	assertMainStringArrayShape(t, tree, lang, src)
}

func TestJavaParseWithTokenSourceMainStringArrayRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`package com.example;

public class App {
    public static void main(String[] args) {
        System.out.println("hello");
    }
}
`)

	ts, err := NewJavaTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJavaTokenSource failed: %v", err)
	}
	tree, err := parser.ParseWithTokenSource(src, ts)
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	assertMainStringArrayShape(t, tree, lang, src)
}

func TestJavaParseWithTokenSourceContextualPermitsIdentifierRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  void f() {
    int permits = 1;
    permits++;
  }
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected contextual permits identifier to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseWithTokenSourceSealedPermitsClauseRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`sealed class A permits B {
}

final class B extends A {
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected sealed permits clause to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseWithTokenSourceCompactNestedGenericRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  Queue<IOConsumer<IndexWriter>> queue = new ConcurrentLinkedQueue<>();
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected compact nested generic to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaArrayInitializerTrailingCommaRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  int[] values = {
    1,
    2, // trailing comma remains optional
  };
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected trailing comma array initializer to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseSwitchRuleThenConditionalExpressionRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  private static void processLongColumn(LongColumn column, PerField pf, IndexableFieldType fieldType)
      throws IOException {
    final boolean hasPoints = fieldType.pointDimensionCount() != 0;
    if (hasPoints == false) {
      LongTupleCursor cursor = column.tuples();
      switch (fieldType.docValuesType()) {
        case NUMERIC -> {
          NumericDocValuesWriter writer = (NumericDocValuesWriter) pf.docValuesWriter;
          int batchDocID;
          while ((batchDocID = cursor.nextDoc()) != DocIdSetIterator.NO_MORE_DOCS) {
            writer.addValue(batchDocID, cursor.longValue());
          }
        }
        default ->
            throw new IllegalArgumentException(
                "LongColumn \"" + column.name() + "\" has incompatible docValuesType");
      }
      return;
    }
    final int byteWidth =
        (column.numericKind() == LongColumn.NumericKind.INT || column.numericKind() == LongColumn.NumericKind.FLOAT)
            ? Integer.BYTES
            : Long.BYTES;
  }
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected switch rule followed by conditional expression to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseSwitchRuleThenPrivateVoidMethodRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  private void storeVectorValues(Info info, IndexableField vectorField) {
    assert vectorField instanceof KnnFloatVectorField || vectorField instanceof KnnByteVectorField;
    switch (info.fieldInfo.getVectorEncoding()) {
      case BYTE -> {
        if (vectorField instanceof KnnByteVectorField byteVectorField) {
          if (info.byteVectorCount == 1) {
            throw new IllegalArgumentException(
                "Only one value per field allowed for byte vector field ["
                    + vectorField.name()
                    + "]");
          }
          info.byteVectorCount++;
          if (info.byteVectorValues == null) {
            info.byteVectorValues = new byte[1][];
          }
          info.byteVectorValues[0] =
              ArrayUtil.copyOfSubArray(
                  byteVectorField.vectorValue(), 0, info.fieldInfo.getVectorDimension());
          return;
        }
        throw new IllegalArgumentException(
            "Field ["
                + vectorField.name()
                + "] is not a byte vector field, but the field info is configured for byte vectors");
      }
      case FLOAT32 -> {
        if (vectorField instanceof KnnFloatVectorField floatVectorField) {
          if (info.floatVectorCount == 1) {
            throw new IllegalArgumentException(
                "Only one value per field allowed for float vector field ["
                    + vectorField.name()
                    + "]");
          }
          info.floatVectorCount++;
          if (info.floatVectorValues == null) {
            info.floatVectorValues = new float[1][];
          }
          info.floatVectorValues[0] =
              ArrayUtil.copyOfSubArray(
                  floatVectorField.vectorValue(), 0, info.fieldInfo.getVectorDimension());
          return;
        }
        throw new IllegalArgumentException(
            "Field ["
                + vectorField.name()
                + "] is not a float vector field, but the field info is configured for float vectors");
      }
    }
  }

  private void storeValues(Info info, IndexableField field) {
    if (info.storedValues == null) {
      info.storedValues = new ArrayList<>();
    }
    BytesRef binaryValue = field.binaryValue();
    if (binaryValue != null) {
      info.storedValues.add(binaryValue);
      return;
    }
  }
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected switch rule followed by private void method to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseSwitchRuleThenClassLiteralArgumentRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  protected void addRandomFields(Document doc) {
    switch (vectorEncoding) {
      case BYTE -> doc.add(new KnnByteVectorField("v2", randomVector8(30), similarityFunction));
      case FLOAT32 ->
          doc.add(new KnnFloatVectorField("v2", randomNormalizedVector(30), similarityFunction));
    }
  }

  protected boolean mergeIsStable() {
    return false;
  }

  private int getVectorsMaxDimensions(String fieldName) {
    return Codec.getDefault().knnVectorsFormat().getMaxDimensions(fieldName);
  }

  public void testFieldConstructor() {
    float[] v = new float[1];
    KnnFloatVectorField field = new KnnFloatVectorField("f", v);
    assertEquals(1, field.fieldType().vectorDimension());
    assertEquals(VectorSimilarityFunction.EUCLIDEAN, field.fieldType().vectorSimilarityFunction());
    assertSame(v, field.vectorValue());
  }

  public void testFieldConstructorExceptions() {
    expectThrows(IllegalArgumentException.class, () -> new KnnFloatVectorField(null, new float[1]));
    expectThrows(IllegalArgumentException.class, () -> new KnnFloatVectorField("f", null));
    expectThrows(
        IllegalArgumentException.class,
        () -> new KnnFloatVectorField("f", new float[1], (VectorSimilarityFunction) null));
    expectThrows(IllegalArgumentException.class, () -> new KnnFloatVectorField("f", new float[0]));
  }
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected switch rule followed by class literal invocation argument to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseWithTokenSourceShiftExpressionRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  void f() {
    int shifted = value >> 1;
  }
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected shift expression to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseShiftExpressionWithParenthesizedRightOperandRegression(t *testing.T) {
	lang := JavaLanguage()
	src := []byte(`class T {
  void f(int bits, int numMantissaBits) {
    int smallfloat = bits >> (24 - numMantissaBits);
  }
}
`)

	t.Run("dfa", func(t *testing.T) {
		parser := gotreesitter.NewParser(lang)
		tree, err := parser.Parse(src)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if tree == nil || tree.RootNode() == nil {
			t.Fatal("parse returned nil root")
		}
		if root := tree.RootNode(); root.HasError() {
			t.Fatalf("expected parenthesized shift expression to parse without syntax errors, got: %s", root.SExpr(lang))
		}
	})

	t.Run("token_source", func(t *testing.T) {
		parser := gotreesitter.NewParser(lang)
		tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
			return NewJavaTokenSource(source, lang)
		})
		if err != nil {
			t.Fatalf("parse with token source failed: %v", err)
		}
		if tree == nil || tree.RootNode() == nil {
			t.Fatal("parse returned nil root")
		}
		if root := tree.RootNode(); root.HasError() {
			t.Fatalf("expected parenthesized shift expression to parse without syntax errors, got: %s", root.SExpr(lang))
		}
	})
}

func TestJavaParseWithTokenSourceUnsignedShiftExpressionRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  void f() {
    int shifted = value >>> 1;
  }
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected unsigned shift expression to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseWithTokenSourceTripleCompactGenericRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  Map<Class<? extends TW>, List<Class<? extends X>>> entries;
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected triple compact generic to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseWithTokenSourceUnderscoreResourceRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  void f() throws Exception {
    try (Closeable _ = resource()) {
    }
  }
}
`)

	tree, err := parser.ParseWithTokenSourceFactory(src, func(source []byte) (gotreesitter.TokenSource, error) {
		return NewJavaTokenSource(source, lang)
	})
	if err != nil {
		t.Fatalf("parse with token source failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected underscore resource to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseEnhancedForCompactNestedGenericRegression(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  void f() {
    for (Map.Entry<String, List<X>> ent : xs.entrySet()) {
      String field = ent.getKey();
    }
  }
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected compact nested generic enhanced-for to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}

func TestJavaParseShiftExpressionAfterCompactAngleSplitter(t *testing.T) {
	lang := JavaLanguage()
	parser := gotreesitter.NewParser(lang)

	src := []byte(`class T {
  void f() {
    int shifted = value >> 1;
  }
}
`)

	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if root := tree.RootNode(); root.HasError() {
		t.Fatalf("expected Java shift expression to parse without syntax errors, got: %s", root.SExpr(lang))
	}
}
