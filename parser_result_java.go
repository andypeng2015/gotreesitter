package gotreesitter

func normalizeJavaCompatibility(root *Node, source []byte, lang *Language) {
	normalizeJavaPrimitiveTypeTokens(root, source, lang)
	normalizeJavaDottedAssignmentDeclarations(root, source, lang)
	normalizeJavaCollapsedLeafChildren(root, source, lang)
	normalizeJavaRecoveredProgramRoot(root, lang)
}

func normalizeJavaPrimitiveTypeTokens(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "java" {
		return
	}
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if len(n.children) == 0 && javaPrimitiveTypeWrapper(n.Type(lang)) {
			normalizeCollapsedTextToken(n, source, lang, javaPrimitiveTypeToken)
		}
		for _, child := range n.children {
			walk(child)
		}
	}
	walk(root)
}

func javaPrimitiveTypeWrapper(name string) bool {
	switch name {
	case "boolean_type", "integral_type", "floating_point_type", "void_type":
		return true
	default:
		return false
	}
}

func javaPrimitiveTypeToken(text string) bool {
	switch text {
	case "boolean", "byte", "short", "int", "long", "char", "float", "double", "void":
		return true
	default:
		return false
	}
}

func normalizeJavaCollapsedLeafChildren(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "java" || len(source) == 0 {
		return
	}
	normalizeCollapsedNamedLeafChildrenBySource(
		root,
		source,
		lang,
		"modifiers",
		"abstract",
		"default",
		"final",
		"native",
		"non-sealed",
		"private",
		"protected",
		"public",
		"sealed",
		"static",
		"strictfp",
		"synchronized",
		"transient",
		"volatile",
	)
	normalizeCollapsedNamedLeafChildrenBySource(root, source, lang, "asterisk", "*")
}

func normalizeJavaDottedAssignmentDeclarations(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "java" || len(source) == 0 {
		return
	}
	ctx, ok := newJavaDottedAssignmentContext(lang)
	if !ok {
		return
	}
	changed := false
	walkResultTree(root, func(n *Node) {
		if rewriteJavaDottedAssignmentDeclaration(n, source, lang, ctx) {
			changed = true
		}
	})
	if changed {
		refreshJavaResultErrors(root)
	}
}

type javaDottedAssignmentContext struct {
	localVariableDeclarationSym Symbol
	typeIdentifierSym           Symbol
	identifierSym               Symbol
	variableDeclaratorSym       Symbol
	expressionStatementSym      Symbol
	assignmentExpressionSym     Symbol
	fieldAccessSym              Symbol
	dotSym                      Symbol
	eqSym                       Symbol
	fieldAccessFieldIDs         []FieldID
	assignmentFieldIDs          []FieldID
}

func newJavaDottedAssignmentContext(lang *Language) (javaDottedAssignmentContext, bool) {
	var ctx javaDottedAssignmentContext
	var ok bool
	if ctx.localVariableDeclarationSym, ok = symbolByName(lang, "local_variable_declaration"); !ok {
		return ctx, false
	}
	if ctx.typeIdentifierSym, ok = symbolByName(lang, "type_identifier"); !ok {
		return ctx, false
	}
	if ctx.identifierSym, ok = symbolByName(lang, "identifier"); !ok {
		return ctx, false
	}
	if ctx.variableDeclaratorSym, ok = symbolByName(lang, "variable_declarator"); !ok {
		return ctx, false
	}
	if ctx.expressionStatementSym, ok = symbolByName(lang, "expression_statement"); !ok {
		return ctx, false
	}
	if ctx.assignmentExpressionSym, ok = symbolByName(lang, "assignment_expression"); !ok {
		return ctx, false
	}
	if ctx.fieldAccessSym, ok = symbolByName(lang, "field_access"); !ok {
		return ctx, false
	}
	if ctx.dotSym, ok = symbolByName(lang, "."); !ok {
		return ctx, false
	}
	if ctx.eqSym, ok = symbolByName(lang, "="); !ok {
		return ctx, false
	}
	ctx.fieldAccessFieldIDs = javaFieldIDs(lang, "object", "", "field")
	ctx.assignmentFieldIDs = javaFieldIDs(lang, "left", "operator", "right")
	return ctx, true
}

func rewriteJavaDottedAssignmentDeclaration(n *Node, source []byte, lang *Language, ctx javaDottedAssignmentContext) bool {
	if n == nil || n.symbol != ctx.localVariableDeclarationSym || resultChildCount(n) != 4 {
		return false
	}
	receiver := resultChildAt(n, 0)
	dot := resultChildAt(n, 1)
	declarator := resultChildAt(n, 2)
	semicolon := resultChildAt(n, 3)
	if receiver == nil || receiver.symbol != ctx.typeIdentifierSym || resultChildCount(receiver) != 0 ||
		dot == nil || !dot.IsError() || !javaNodeTextEquals(dot, source, ".") ||
		declarator == nil || declarator.symbol != ctx.variableDeclaratorSym || resultChildCount(declarator) < 3 ||
		semicolon == nil || semicolon.Type(lang) != ";" {
		return false
	}
	field := resultChildAt(declarator, 0)
	eq := resultChildAt(declarator, 1)
	rhs := resultChildAt(declarator, 2)
	if field == nil || field.symbol != ctx.identifierSym || resultChildCount(field) != 0 ||
		eq == nil || eq.symbol != ctx.eqSym ||
		rhs == nil ||
		!javaSimpleIdentifierNode(receiver, source) ||
		!javaSimpleIdentifierNode(field, source) {
		return false
	}

	receiver.symbol = ctx.identifierSym
	receiver.setNamed(symbolIsNamed(lang, ctx.identifierSym))
	receiver.setExtra(false)
	receiver.setMissing(false)
	receiver.setHasError(false)
	dot.symbol = ctx.dotSym
	dot.setNamed(symbolIsNamed(lang, ctx.dotSym))
	dot.setExtra(false)
	dot.setMissing(false)
	dot.setHasError(false)

	arena := n.ownerArena
	fieldAccess := newParentNodeInArena(arena, ctx.fieldAccessSym, symbolIsNamed(lang, ctx.fieldAccessSym), cloneNodeSliceInArena(arena, []*Node{
		receiver,
		dot,
		field,
	}), cloneFieldIDSliceInArena(arena, ctx.fieldAccessFieldIDs), 0)
	assignment := newParentNodeInArena(arena, ctx.assignmentExpressionSym, symbolIsNamed(lang, ctx.assignmentExpressionSym), cloneNodeSliceInArena(arena, []*Node{
		fieldAccess,
		eq,
		rhs,
	}), cloneFieldIDSliceInArena(arena, ctx.assignmentFieldIDs), 0)

	n.symbol = ctx.expressionStatementSym
	n.setNamed(symbolIsNamed(lang, ctx.expressionStatementSym))
	n.setHasError(false)
	replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(arena, []*Node{assignment, semicolon}))
	refreshResultRootError(n)
	return true
}

func javaFieldIDs(lang *Language, names ...string) []FieldID {
	if lang == nil || len(names) == 0 {
		return nil
	}
	fieldIDs := make([]FieldID, len(names))
	any := false
	for i, name := range names {
		if name == "" {
			continue
		}
		fid, ok := lang.FieldByName(name)
		if !ok {
			continue
		}
		fieldIDs[i] = fid
		any = true
	}
	if !any {
		return nil
	}
	return fieldIDs
}

func javaSimpleIdentifierNode(n *Node, source []byte) bool {
	if n == nil || int(n.startByte) > len(source) || int(n.endByte) > len(source) || n.startByte >= n.endByte {
		return false
	}
	text := source[n.startByte:n.endByte]
	if !javaIdentifierStartByte(text[0]) {
		return false
	}
	for _, c := range text[1:] {
		if !javaIdentifierContinueByte(c) {
			return false
		}
	}
	return true
}

func javaIdentifierStartByte(c byte) bool {
	return c == '_' || c == '$' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func javaIdentifierContinueByte(c byte) bool {
	return javaIdentifierStartByte(c) || (c >= '0' && c <= '9')
}

func javaNodeTextEquals(n *Node, source []byte, text string) bool {
	if n == nil || int(n.startByte) > len(source) || int(n.endByte) > len(source) || n.startByte > n.endByte {
		return false
	}
	span := source[n.startByte:n.endByte]
	if len(span) != len(text) {
		return false
	}
	for i := range span {
		if span[i] != text[i] {
			return false
		}
	}
	return true
}

func refreshJavaResultErrors(root *Node) bool {
	if root == nil {
		return false
	}
	hasError := root.IsError()
	for i := 0; i < resultChildCount(root); i++ {
		if refreshJavaResultErrors(resultChildAt(root, i)) {
			hasError = true
		}
	}
	root.setHasError(hasError)
	return hasError
}

func normalizeJavaRecoveredProgramRoot(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "java" || !root.IsError() || javaResultChildrenHaveError(root) {
		return
	}
	programSym, ok := symbolByName(lang, "program")
	if !ok {
		return
	}
	retagResultRootAndRefreshError(root, programSym, symbolIsNamed(lang, programSym))
}

func javaResultChildrenHaveError(root *Node) bool {
	if root == nil {
		return false
	}
	for i := 0; i < resultChildCount(root); i++ {
		child := resultChildAt(root, i)
		if child != nil && (child.IsError() || child.HasError()) {
			return true
		}
	}
	return false
}
