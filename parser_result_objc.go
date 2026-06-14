package gotreesitter

func normalizeObjcCompatibility(root *Node, source []byte, lang *Language) {
	normalizeObjcMethodTypeIdentifiers(root, lang)
	normalizeObjcAtStringLiterals(root, lang)
}

func normalizeObjcMethodTypeIdentifiers(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "objc" {
		return
	}
	methodTypeSym, ok1 := symbolByName(lang, "method_type")
	typeNameSym, ok2 := symbolByName(lang, "type_name")
	identifierSym, ok3 := symbolByName(lang, "identifier")
	typeIdentifierSym, ok4 := symbolByName(lang, "type_identifier")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return
	}
	typeIdentifierNamed := symbolIsNamed(lang, typeIdentifierSym)
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != methodTypeSym {
			return
		}
		for i := 0; i < resultChildCount(n); i++ {
			typeName := resultChildAt(n, i)
			if typeName == nil || typeName.symbol != typeNameSym {
				continue
			}
			for j := 0; j < resultChildCount(typeName); j++ {
				child := resultChildAt(typeName, j)
				if child != nil && child.symbol == identifierSym {
					child.symbol = typeIdentifierSym
					child.setNamed(typeIdentifierNamed)
				}
			}
		}
	})
}

func normalizeObjcAtStringLiterals(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "objc" {
		return
	}
	atExpressionSym, ok1 := symbolByName(lang, "at_expression")
	stringLiteralSym, ok2 := symbolByName(lang, "string_literal")
	concatenatedStringSym, hasConcatenatedString := symbolByName(lang, "concatenated_string")
	if !ok1 || !ok2 {
		return
	}
	stringLiteralNamed := symbolIsNamed(lang, stringLiteralSym)
	concatenatedStringNamed := false
	if hasConcatenatedString {
		concatenatedStringNamed = symbolIsNamed(lang, concatenatedStringSym)
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != atExpressionSym || resultChildCount(n) != 2 {
			return
		}
		at := resultChildAt(n, 0)
		inner := resultChildAt(n, 1)
		if at == nil || inner == nil {
			return
		}
		switch inner.symbol {
		case stringLiteralSym:
			innerChildren := resultChildSliceForMutation(inner)
			if len(innerChildren) == 0 {
				return
			}
			children := make([]*Node, 0, len(innerChildren)+1)
			children = append(children, at)
			children = append(children, innerChildren...)
			children = cloneNodeSliceIfArena(n.ownerArena, children)
			n.symbol = stringLiteralSym
			n.setNamed(stringLiteralNamed)
			replaceNodeChildrenUnfielded(n, children)
			n.productionID = 0
		case concatenatedStringSym:
			if !hasConcatenatedString || resultChildCount(inner) == 0 {
				return
			}
			concatChildren := resultChildSliceForMutation(inner)
			first := concatChildren[0]
			if first == nil || first.symbol != stringLiteralSym {
				return
			}
			firstChildren := resultChildSliceForMutation(first)
			if len(firstChildren) == 0 {
				return
			}
			rebuiltFirstChildren := make([]*Node, 0, len(firstChildren)+1)
			rebuiltFirstChildren = append(rebuiltFirstChildren, at)
			rebuiltFirstChildren = append(rebuiltFirstChildren, firstChildren...)
			first.symbol = stringLiteralSym
			first.setNamed(stringLiteralNamed)
			first.startByte = at.startByte
			first.startPoint = at.startPoint
			replaceNodeChildrenUnfielded(first, cloneNodeSliceIfArena(first.ownerArena, rebuiltFirstChildren))
			first.productionID = 0
			children := cloneNodeSliceIfArena(n.ownerArena, concatChildren)
			n.symbol = concatenatedStringSym
			n.setNamed(concatenatedStringNamed)
			n.startByte = at.startByte
			n.startPoint = at.startPoint
			replaceNodeChildrenUnfielded(n, children)
			n.productionID = 0
		default:
			return
		}
	})
}
