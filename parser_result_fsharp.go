package gotreesitter

func normalizeFSharpCompatibility(root *Node, source []byte, lang *Language) {
	normalizeFSharpSimpleLongIdentifiers(root, lang)
}

func normalizeFSharpSimpleLongIdentifiers(root *Node, lang *Language) {
	if root == nil || lang == nil || root.HasError() {
		return
	}
	longIdentSym, ok := lang.symbolByNameAndNamed("long_identifier", true)
	if !ok {
		longIdentSym, ok = symbolByName(lang, "long_identifier")
	}
	if !ok {
		return
	}
	identSym, ok := lang.symbolByNameAndNamed("identifier", true)
	if !ok {
		identSym, ok = symbolByName(lang, "identifier")
	}
	if !ok {
		return
	}
	longIdentOrOpSym, ok := lang.symbolByNameAndNamed("long_identifier_or_op", true)
	if !ok {
		longIdentOrOpSym, ok = symbolByName(lang, "long_identifier_or_op")
	}
	if !ok {
		return
	}

	normalizeFSharpSimpleLongIdentifiersInNode(root, longIdentSym, identSym, longIdentOrOpSym, root.symbol == longIdentOrOpSym)
}

func normalizeFSharpSimpleLongIdentifiersInNode(n *Node, longIdentSym, identSym, longIdentOrOpSym Symbol, inLongIdentifierOrOp bool) {
	if n == nil {
		return
	}
	childCount := resultChildCount(n)
	for i := 0; i < childCount; i++ {
		child := resultChildAt(n, i)
		childInLongIdentifierOrOp := inLongIdentifierOrOp || (child != nil && child.symbol == longIdentOrOpSym)
		if inLongIdentifierOrOp && fsharpSimpleLongIdentifier(child, longIdentSym, identSym) {
			replacement := resultChildAt(child, 0)
			replaceChildRangeWithSingleNode(n, i, i+1, replacement)
			child = replacement
		}
		normalizeFSharpSimpleLongIdentifiersInNode(child, longIdentSym, identSym, longIdentOrOpSym, childInLongIdentifierOrOp)
	}
}

func fsharpSimpleLongIdentifier(n *Node, longIdentSym, identSym Symbol) bool {
	if n == nil || n.symbol != longIdentSym || resultChildCount(n) != 1 || n.HasError() || n.IsMissing() || n.IsExtra() {
		return false
	}
	child := resultChildAt(n, 0)
	return child != nil &&
		child.symbol == identSym &&
		child.StartByte() == n.StartByte() &&
		child.EndByte() == n.EndByte() &&
		!child.HasError() &&
		!child.IsMissing() &&
		!child.IsExtra()
}
