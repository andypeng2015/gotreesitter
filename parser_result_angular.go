package gotreesitter

func normalizeAngularCompatibility(root *Node, source []byte, lang *Language) {
	normalizeAngularNonNullAssertionErrors(root, source, lang)
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
