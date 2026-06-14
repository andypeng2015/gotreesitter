package gotreesitter

func normalizeObjcCompatibility(root *Node, source []byte, lang *Language) {
	normalizeObjcMethodTypeIdentifiers(root, lang)
	normalizeObjcParameterizedArgumentTypeIdentifiers(root, lang)
	normalizeObjcSizeofTypeIdentifierOperands(root, lang)
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

func normalizeObjcParameterizedArgumentTypeIdentifiers(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "objc" {
		return
	}
	parameterizedArgumentsSym, ok1 := symbolByName(lang, "parameterized_arguments")
	typeNameSym, ok2 := symbolByName(lang, "type_name")
	identifierSym, ok3 := symbolByName(lang, "identifier")
	typeIdentifierSym, ok4 := symbolByName(lang, "type_identifier")
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return
	}
	typeIdentifierNamed := symbolIsNamed(lang, typeIdentifierSym)
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != parameterizedArgumentsSym {
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

func normalizeObjcSizeofTypeIdentifierOperands(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "objc" {
		return
	}
	sizeofSym, ok1 := symbolByName(lang, "sizeof_expression")
	typeDescriptorSym, ok2 := symbolByName(lang, "type_descriptor")
	typeIdentifierSym, ok3 := symbolByName(lang, "type_identifier")
	identifierSym, ok4 := symbolByName(lang, "identifier")
	parenthesizedSym, ok5 := symbolByName(lang, "parenthesized_expression")
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return
	}
	identifierNamed := symbolIsNamed(lang, identifierSym)
	parenthesizedNamed := symbolIsNamed(lang, parenthesizedSym)
	valueFieldID, hasValueField := lang.FieldByName("value")
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != sizeofSym || len(n.children) != 4 {
			return
		}
		typeDescriptor := n.children[2]
		if typeDescriptor == nil || typeDescriptor.symbol != typeDescriptorSym || len(typeDescriptor.children) != 1 {
			return
		}
		typeIdent := typeDescriptor.children[0]
		if typeIdent == nil || typeIdent.symbol != typeIdentifierSym {
			return
		}
		ident := newLeafNodeInArena(n.ownerArena, identifierSym, identifierNamed, typeIdent.startByte, typeIdent.endByte, typeIdent.startPoint, typeIdent.endPoint)
		paren := newParentNodeInArena(n.ownerArena, parenthesizedSym, parenthesizedNamed, []*Node{n.children[1], ident, n.children[3]}, nil, 0)
		replaceChildRangeWithSingleNode(n, 1, 4, paren)
		if hasValueField && len(n.children) > 1 {
			ensureNodeFieldStorage(n, len(n.children))
			n.fieldIDs[1] = valueFieldID
			n.fieldSources[1] = fieldSourceDirect
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
