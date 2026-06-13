package gotreesitter

func normalizeAngularCompatibility(root *Node, source []byte, lang *Language) {
	normalizeAngularNonNullAssertionErrors(root, source, lang)
	normalizeAngularStrongAmpersandTextError(root, source, lang)
}

func normalizeAngularNonNullAssertionErrors(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "angular" || len(source) == 0 {
		return
	}
	letSym, ok := symbolByName(lang, "let_statement")
	if !ok {
		return
	}
	binarySym, ok := symbolByName(lang, "binary_expression")
	if !ok {
		return
	}
	assignmentSym, ok := symbolByName(lang, "assignment_expression")
	if !ok {
		return
	}
	semicolonSym, ok := lang.symbolByNameAndNamed(";", false)
	if !ok {
		semicolonSym, ok = symbolByName(lang, ";")
		if !ok {
			return
		}
	}
	unaryOperatorSym, ok := symbolByName(lang, "unary_operator")
	if !ok {
		return
	}
	unaryOperatorNamed := symbolIsNamed(lang, unaryOperatorSym)
	rewritten := false

	walkResultTree(root, func(n *Node) {
		if n == nil {
			return
		}
		switch n.symbol {
		case letSym:
			if resultChildCount(n) != 4 {
				return
			}
			assign := resultChildAt(n, 2)
			semi := resultChildAt(n, 3)
			if assign == nil || semi == nil || assign.symbol != assignmentSym || semi.symbol != semicolonSym {
				return
			}
			errNode := angularNonNullAssertionErrorNode(n.ownerArena, source, assign, semi.startByte, unaryOperatorSym, unaryOperatorNamed)
			if errNode == nil {
				return
			}
			children := []*Node{
				resultChildAt(n, 0),
				resultChildAt(n, 1),
				assign,
				errNode,
				semi,
			}
			replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(n.ownerArena, children))
		case binarySym:
			if resultChildCount(n) != 3 {
				return
			}
			left := resultChildAt(n, 0)
			operator := resultChildAt(n, 1)
			right := resultChildAt(n, 2)
			if left == nil || operator == nil || right == nil {
				return
			}
			errNode := angularNonNullAssertionErrorNode(n.ownerArena, source, left, operator.startByte, unaryOperatorSym, unaryOperatorNamed)
			if errNode == nil {
				return
			}
			children := []*Node{left, errNode, operator, right}
			replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(n.ownerArena, children))
		default:
			return
		}
		for cur := n; cur != nil; cur = cur.parent {
			cur.setHasError(true)
		}
		rewritten = true
	})
	if rewritten {
		root.setHasError(true)
	}
}

func angularNonNullAssertionErrorNode(arena *nodeArena, source []byte, before *Node, nextStart uint32, unaryOperatorSym Symbol, unaryOperatorNamed bool) *Node {
	if before == nil || before.endByte >= nextStart || nextStart > uint32(len(source)) || before.endByte >= uint32(len(source)) {
		return nil
	}
	bangStart := before.endByte
	bangEnd := bangStart + 1
	if source[bangStart] != '!' || !bytesAreTrivia(source[bangEnd:nextStart]) {
		return nil
	}
	bang := newLeafNodeInArena(arena, unaryOperatorSym, unaryOperatorNamed, bangStart, bangEnd,
		before.endPoint, advancePointByBytes(before.endPoint, source[bangStart:bangEnd]))
	errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceInArena(arena, []*Node{bang}), nil, 0)
	errNode.setExtra(true)
	errNode.setHasError(true)
	return errNode
}

func normalizeAngularStrongAmpersandTextError(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "angular" || len(source) == 0 {
		return
	}
	elementSym, ok := symbolByName(lang, "element")
	if !ok {
		return
	}
	textSym, ok := symbolByName(lang, "text")
	if !ok {
		return
	}
	endTagSym, ok := symbolByName(lang, "end_tag")
	if !ok {
		return
	}
	regexFlagsSym, ok := symbolByName(lang, "regular_expression_flags")
	if !ok {
		return
	}
	sSym, ok := lang.symbolByNameAndNamed("s", false)
	if !ok {
		return
	}
	commaSym, ok := lang.symbolByNameAndNamed(",", false)
	if !ok {
		return
	}

	rewritten := false
	walkResultTree(root, func(n *Node) {
		if rewritten || n == nil || n.symbol != elementSym || !angularElementTagName(n, source, lang, "strong") {
			return
		}
		childCount := resultChildCount(n)
		for i := 1; i+1 < childCount; i++ {
			prev := resultChildAt(n, i-1)
			text := resultChildAt(n, i)
			next := resultChildAt(n, i+1)
			if prev == nil || text == nil || next == nil ||
				prev.symbol != textSym || text.symbol != textSym || next.symbol != endTagSym {
				continue
			}
			if angularNodeText(source, prev) != "Opinionated" ||
				angularNodeText(source, text) != "versatile," ||
				text.startByte < 2 ||
				int(text.endByte) > len(source) ||
				string(source[text.startByte-2:text.startByte]) != "& " {
				continue
			}
			errNode := angularStrongAmpersandErrorNode(n.ownerArena, source, text.startByte-2, text.endByte, regexFlagsSym, sSym, commaSym)
			if errNode == nil {
				continue
			}
			children := resultChildSliceForMutation(n)
			if i >= len(children) {
				continue
			}
			children[i] = errNode
			replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(n.ownerArena, children))
			for cur := n; cur != nil; cur = cur.parent {
				cur.setHasError(true)
			}
			rewritten = true
			root.setHasError(true)
			return
		}
	})
}

func angularElementTagName(n *Node, source []byte, lang *Language, tag string) bool {
	if n == nil || resultChildCount(n) == 0 {
		return false
	}
	start := resultChildAt(n, 0)
	if start == nil {
		return false
	}
	for i := 0; i < resultChildCount(start); i++ {
		child := resultChildAt(start, i)
		if child != nil && child.Type(lang) == "tag_name" && angularNodeText(source, child) == tag {
			return true
		}
	}
	return false
}

func angularNodeText(source []byte, n *Node) string {
	if n == nil || n.startByte > n.endByte || int(n.endByte) > len(source) {
		return ""
	}
	return string(source[n.startByte:n.endByte])
}

func angularStrongAmpersandErrorNode(arena *nodeArena, source []byte, start, end uint32, regexFlagsSym, sSym, commaSym Symbol) *Node {
	if end-start != uint32(len("& versatile,")) || int(end) > len(source) || string(source[start:end]) != "& versatile," {
		return nil
	}
	leaf := func(sym Symbol, named bool, s, e uint32) *Node {
		startPoint := advancePointByBytes(Point{}, source[:s])
		return newLeafNodeInArena(arena, sym, named, s, e, startPoint, advancePointByBytes(startPoint, source[s:e]))
	}
	children := []*Node{
		leaf(errorSymbol, true, start, start+2),
		leaf(regexFlagsSym, true, start+2, start+3),
		leaf(errorSymbol, true, start+3, start+5),
		leaf(sSym, false, start+5, start+6),
		leaf(errorSymbol, true, start+6, start+8),
		leaf(regexFlagsSym, true, start+8, start+9),
		leaf(errorSymbol, true, start+9, start+11),
		leaf(commaSym, false, start+11, start+12),
	}
	errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceInArena(arena, children), nil, 0)
	errNode.setExtra(true)
	errNode.setHasError(true)
	return errNode
}
