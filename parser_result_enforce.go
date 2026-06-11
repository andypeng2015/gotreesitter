package gotreesitter

func normalizeEnforceCompatibility(root *Node, source []byte, lang *Language) {
	normalizeEnforceConstIntFormalParameter(root, source, lang)
}

func normalizeEnforceConstIntFormalParameter(root *Node, source []byte, lang *Language) {
	formalParameterSym, ok := symbolByName(lang, "formal_parameter")
	if !ok {
		return
	}
	formalParameterModifierSym, ok := symbolByName(lang, "formal_parameter_modifier")
	if !ok {
		return
	}
	typeIdentifierSym, ok := symbolByName(lang, "type_identifier")
	if !ok {
		return
	}
	typeIntSym, ok := symbolByName(lang, "type_int")
	if !ok {
		return
	}
	identifierSym, ok := symbolByName(lang, "identifier")
	if !ok {
		return
	}

	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != formalParameterSym || resultChildCount(n) != 4 {
			return
		}
		typeIdent := resultChildAt(n, 0)
		intIdent := resultChildAt(n, 1)
		eq := resultChildAt(n, 2)
		value := resultChildAt(n, 3)
		if typeIdent == nil || intIdent == nil || eq == nil || value == nil ||
			typeIdent.symbol != typeIdentifierSym ||
			intIdent.symbol != identifierSym ||
			resultChildCount(typeIdent) != 1 ||
			resultChildCount(intIdent) != 0 ||
			int(eq.startByte) >= len(source) || eq.startByte >= eq.endByte ||
			source[eq.startByte] != '=' {
			return
		}
		if !enforceNodeTextEquals(source, typeIdent, "const") || !enforceNodeTextEquals(source, intIdent, "int") {
			return
		}

		nameStart, nameEnd, ok := enforceIdentifierGap(source, intIdent.endByte, eq.startByte)
		if !ok {
			return
		}

		retagResultRoot(typeIdent, formalParameterModifierSym, symbolIsNamed(lang, formalParameterModifierSym))
		replaceNodeChildrenUnfielded(typeIdent, nil)
		retagResultRoot(intIdent, typeIntSym, symbolIsNamed(lang, typeIntSym))

		name := newLeafNodeInArena(n.ownerArena, identifierSym, symbolIsNamed(lang, identifierSym),
			nameStart, nameEnd,
			advancePointByBytes(Point{}, source[:nameStart]),
			advancePointByBytes(Point{}, source[:nameEnd]))
		replaceChildRangeWithNodes(n, 0, 4, []*Node{typeIdent, intIdent, name, eq, value})
	})
}

func enforceNodeTextEquals(source []byte, n *Node, want string) bool {
	return n != nil &&
		n.startByte <= n.endByte &&
		int(n.endByte) <= len(source) &&
		string(source[n.startByte:n.endByte]) == want
}

func enforceIdentifierGap(source []byte, start, end uint32) (uint32, uint32, bool) {
	if start >= end || int(end) > len(source) {
		return 0, 0, false
	}
	i := start
	for i < end && isEnforceASCIIWhitespace(source[i]) {
		i++
	}
	nameStart := i
	if i >= end || !isEnforceIdentifierStart(source[i]) {
		return 0, 0, false
	}
	i++
	for i < end && isEnforceIdentifierContinue(source[i]) {
		i++
	}
	nameEnd := i
	for i < end && isEnforceASCIIWhitespace(source[i]) {
		i++
	}
	if i != end {
		return 0, 0, false
	}
	return nameStart, nameEnd, true
}

func isEnforceASCIIWhitespace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func isEnforceIdentifierStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isEnforceIdentifierContinue(b byte) bool {
	return isEnforceIdentifierStart(b) || (b >= '0' && b <= '9')
}
