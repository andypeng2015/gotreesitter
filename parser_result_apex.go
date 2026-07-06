package gotreesitter

func normalizeApexCompatibility(root *Node, source []byte, lang *Language) {
	normalizeApexGenericLocalDeclarations(root, source, lang)
	normalizeApexClassLiteralAccess(root, source, lang)
}

type apexCompatSymbols struct {
	localVariableDeclaration Symbol
	genericType              Symbol
	typeArguments            Symbol
	typeIdentifier           Symbol
	scopedTypeIdentifier     Symbol
	variableDeclarator       Symbol
	fieldAccess              Symbol
	identifier               Symbol
	lt                       Symbol
	gt                       Symbol
	dot                      Symbol
}

func loadApexCompatSymbols(lang *Language) (apexCompatSymbols, bool) {
	var s apexCompatSymbols
	var ok bool
	if s.localVariableDeclaration, ok = symbolByName(lang, "local_variable_declaration"); !ok {
		return s, false
	}
	if s.genericType, ok = symbolByName(lang, "generic_type"); !ok {
		return s, false
	}
	if s.typeArguments, ok = symbolByName(lang, "type_arguments"); !ok {
		return s, false
	}
	if s.typeIdentifier, ok = symbolByName(lang, "type_identifier"); !ok {
		return s, false
	}
	if s.scopedTypeIdentifier, ok = symbolByName(lang, "scoped_type_identifier"); !ok {
		return s, false
	}
	if s.variableDeclarator, ok = symbolByName(lang, "variable_declarator"); !ok {
		return s, false
	}
	if s.fieldAccess, ok = symbolByName(lang, "field_access"); !ok {
		return s, false
	}
	if s.identifier, ok = symbolByName(lang, "identifier"); !ok {
		return s, false
	}
	if s.lt, ok = symbolByName(lang, "<"); !ok {
		return s, false
	}
	if s.gt, ok = symbolByName(lang, ">"); !ok {
		return s, false
	}
	if s.dot, ok = symbolByName(lang, "."); !ok {
		return s, false
	}
	return s, true
}

func normalizeApexGenericLocalDeclarations(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "apex" {
		return
	}
	syms, ok := loadApexCompatSymbols(lang)
	if !ok {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.Type(lang) != "expression_statement" || resultChildCount(n) != 2 {
			return
		}
		expr := resultChildAt(n, 0)
		semi := resultChildAt(n, 1)
		if expr == nil || semi == nil || expr.Type(lang) != "binary_expression" || semi.Type(lang) != ";" {
			return
		}
		assign := apexFindDeclarationAssignment(expr, lang)
		if assign == nil || assign.startByte <= expr.startByte || int(assign.startByte) > len(source) {
			return
		}
		typeStart := expr.startByte
		typeEnd := apexTrimASCIIWhitespaceEnd(source, typeStart, assign.startByte)
		if typeEnd <= typeStart || !apexRangeContainsByte(source, typeStart, typeEnd, '<') {
			return
		}
		typeNode, next, ok := apexParseType(source, typeStart, typeEnd, lang, syms, n.ownerArena)
		if !ok || next != typeEnd || typeNode == nil || typeNode.Type(lang) != "generic_type" {
			return
		}
		retagResultRoot(assign, syms.variableDeclarator, symbolIsNamed(lang, syms.variableDeclarator))
		retagResultRoot(n, syms.localVariableDeclaration, symbolIsNamed(lang, syms.localVariableDeclaration))
		replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(n.ownerArena, []*Node{typeNode, assign, semi}))
	})
}

func apexFindDeclarationAssignment(n *Node, lang *Language) *Node {
	if n == nil {
		return nil
	}
	if n.Type(lang) == "assignment_expression" && resultChildCount(n) >= 3 {
		left := resultChildAt(n, 0)
		if left != nil && left.Type(lang) == "identifier" {
			return n
		}
	}
	for i := resultChildCount(n) - 1; i >= 0; i-- {
		if found := apexFindDeclarationAssignment(resultChildAt(n, i), lang); found != nil {
			return found
		}
	}
	return nil
}

func apexParseType(source []byte, start, end uint32, lang *Language, syms apexCompatSymbols, arena *nodeArena) (*Node, uint32, bool) {
	start = apexSkipASCIIWhitespace(source, start, end)
	base, next, ok := apexParseQualifiedTypeIdentifier(source, start, end, lang, syms, arena)
	if !ok {
		return nil, start, false
	}
	next = apexSkipASCIIWhitespace(source, next, end)
	if next >= end || source[next] != '<' {
		return base, next, true
	}
	lt := apexLeaf(source, next, next+1, syms.lt, false, arena)
	arg, afterArg, ok := apexParseType(source, next+1, end, lang, syms, arena)
	if !ok {
		return nil, start, false
	}
	afterArg = apexSkipASCIIWhitespace(source, afterArg, end)
	if afterArg >= end || source[afterArg] != '>' {
		return nil, start, false
	}
	gt := apexLeaf(source, afterArg, afterArg+1, syms.gt, false, arena)
	typeArgs := newParentNodeInArena(arena, syms.typeArguments, symbolIsNamed(lang, syms.typeArguments), []*Node{lt, arg, gt}, nil, 0)
	generic := newParentNodeInArena(arena, syms.genericType, symbolIsNamed(lang, syms.genericType), []*Node{base, typeArgs}, nil, 0)
	return generic, afterArg + 1, true
}

func apexParseQualifiedTypeIdentifier(source []byte, start, end uint32, lang *Language, syms apexCompatSymbols, arena *nodeArena) (*Node, uint32, bool) {
	partStart := apexSkipASCIIWhitespace(source, start, end)
	partEnd := apexScanIdentifier(source, partStart, end)
	if partEnd == partStart {
		return nil, start, false
	}
	children := []*Node{apexLeaf(source, partStart, partEnd, syms.typeIdentifier, true, arena)}
	next := apexSkipASCIIWhitespace(source, partEnd, end)
	for next < end && source[next] == '.' {
		dot := apexLeaf(source, next, next+1, syms.dot, false, arena)
		nameStart := apexSkipASCIIWhitespace(source, next+1, end)
		nameEnd := apexScanIdentifier(source, nameStart, end)
		if nameEnd == nameStart {
			return nil, start, false
		}
		children = append(children, dot, apexLeaf(source, nameStart, nameEnd, syms.typeIdentifier, true, arena))
		next = apexSkipASCIIWhitespace(source, nameEnd, end)
	}
	if len(children) == 1 {
		return children[0], next, true
	}
	return newParentNodeInArena(arena, syms.scopedTypeIdentifier, symbolIsNamed(lang, syms.scopedTypeIdentifier), cloneNodeSliceIfArena(arena, children), nil, 0), next, true
}

func normalizeApexClassLiteralAccess(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "apex" {
		return
	}
	fieldAccessSym, fieldAccessNamed, ok := symbolMeta(lang, "field_access")
	if !ok {
		return
	}
	identifierSym, identifierNamed, ok := symbolMeta(lang, "identifier")
	if !ok {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.Type(lang) != "class_literal" || resultChildCount(n) != 3 {
			return
		}
		left := resultChildAt(n, 0)
		dot := resultChildAt(n, 1)
		right := resultChildAt(n, 2)
		if left == nil || dot == nil || right == nil ||
			left.Type(lang) != "type_identifier" ||
			dot.Type(lang) != "." ||
			right.Type(lang) != "class" ||
			!apexNodeTextEquals(source, right, "class") {
			return
		}
		retagResultRoot(n, fieldAccessSym, fieldAccessNamed)
		retagResultRoot(left, identifierSym, identifierNamed)
		retagResultRoot(right, identifierSym, identifierNamed)
	})
}

func apexLeaf(source []byte, start, end uint32, sym Symbol, named bool, arena *nodeArena) *Node {
	return newLeafNodeInArena(
		arena,
		sym,
		named,
		start,
		end,
		advancePointByBytes(Point{}, source[:start]),
		advancePointByBytes(Point{}, source[:end]),
	)
}

func apexTrimASCIIWhitespaceEnd(source []byte, start, end uint32) uint32 {
	if int(end) > len(source) {
		end = uint32(len(source))
	}
	for end > start {
		switch source[end-1] {
		case ' ', '\t', '\n', '\r', '\f':
			end--
		default:
			return end
		}
	}
	return end
}

func apexSkipASCIIWhitespace(source []byte, start, end uint32) uint32 {
	if int(end) > len(source) {
		end = uint32(len(source))
	}
	for start < end {
		switch source[start] {
		case ' ', '\t', '\n', '\r', '\f':
			start++
		default:
			return start
		}
	}
	return start
}

func apexScanIdentifier(source []byte, start, end uint32) uint32 {
	if start >= end || int(end) > len(source) || !apexIsIdentifierStart(source[start]) {
		return start
	}
	i := start + 1
	for i < end && apexIsIdentifierContinue(source[i]) {
		i++
	}
	return i
}

func apexIsIdentifierStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func apexIsIdentifierContinue(b byte) bool {
	return apexIsIdentifierStart(b) || (b >= '0' && b <= '9')
}

func apexRangeContainsByte(source []byte, start, end uint32, target byte) bool {
	if int(end) > len(source) {
		return false
	}
	for i := start; i < end; i++ {
		if source[i] == target {
			return true
		}
	}
	return false
}

func apexNodeTextEquals(source []byte, n *Node, want string) bool {
	return n != nil &&
		n.startByte <= n.endByte &&
		int(n.endByte) <= len(source) &&
		string(source[n.startByte:n.endByte]) == want
}
