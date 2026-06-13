package gotreesitter

func normalizeJuliaCompatibility(root *Node, source []byte, lang *Language) {
	normalizeJuliaRecoveredReturnRange(root, source, lang)
}

func normalizeJuliaRecoveredReturnRange(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "julia" || len(source) == 0 {
		return
	}
	blockSym, ok := symbolByName(lang, "block")
	if !ok {
		return
	}
	returnSym, ok := symbolByName(lang, "return_statement")
	if !ok {
		return
	}
	rangeSym, ok := symbolByName(lang, "range_expression")
	if !ok {
		return
	}
	quoteSym, ok := symbolByName(lang, "quote_expression")
	if !ok {
		return
	}
	binarySym, ok := symbolByName(lang, "binary_expression")
	if !ok {
		return
	}
	parenSym, ok := symbolByName(lang, "parenthesized_expression")
	if !ok {
		return
	}
	integerSym, ok := symbolByName(lang, "integer_literal")
	if !ok {
		return
	}
	walkResultTree(root, func(block *Node) {
		if block == nil || block.symbol != blockSym || resultChildCount(block) == 0 {
			return
		}
		children := resultChildSliceForMutation(block)
		blockEndByte := block.endByte
		blockEndPoint := block.endPoint
		rewritten := false
		out := make([]*Node, 0, len(children)+1)
		for _, child := range children {
			out = append(out, child)
			closeParen, ok := juliaRewriteReturnRange(child, source, returnSym, rangeSym, quoteSym, binarySym, parenSym, integerSym)
			if !ok {
				continue
			}
			if closeParen != nil {
				closeErr := newParentNodeInArena(block.ownerArena, errorSymbol, true, []*Node{closeParen}, nil, 0)
				closeErr.startByte = closeParen.startByte
				closeErr.startPoint = closeParen.startPoint
				closeErr.endByte = closeParen.endByte
				closeErr.endPoint = closeParen.endPoint
				closeErr.setHasError(true)
				closeErr.setExtra(true)
				out = append(out, closeErr)
			}
			rewritten = ok || rewritten
		}
		if rewritten {
			replaceNodeChildrenUnfielded(block, cloneNodeSliceInArena(block.ownerArena, out))
			block.endByte = blockEndByte
			block.endPoint = blockEndPoint
			block.setHasError(true)
			root.setHasError(true)
		}
	})
}

func juliaRewriteReturnRange(ret *Node, source []byte, returnSym, rangeSym, quoteSym, binarySym, parenSym, integerSym Symbol) (*Node, bool) {
	if ret == nil || ret.symbol != returnSym || resultChildCount(ret) != 2 {
		return nil, false
	}
	expr := resultChildAt(ret, 1)
	if expr == nil || expr.symbol != rangeSym || resultChildCount(expr) != 3 {
		return nil, false
	}
	left := resultChildAt(expr, 0)
	colon := resultChildAt(expr, 1)
	right := resultChildAt(expr, 2)
	if juliaRewriteBareReturnRange(ret, expr, left, colon, right, source, quoteSym) {
		return nil, true
	}
	if left == nil || colon == nil || right == nil || right.symbol != parenSym || resultChildCount(right) != 3 {
		return nil, false
	}
	open := resultChildAt(right, 0)
	inner := resultChildAt(right, 1)
	closeParen := resultChildAt(right, 2)
	if open == nil || inner == nil || closeParen == nil || inner.symbol != binarySym || resultChildCount(inner) != 3 {
		return nil, false
	}
	innerLeft := resultChildAt(inner, 0)
	innerOp := resultChildAt(inner, 1)
	innerRight := resultChildAt(inner, 2)
	if innerLeft == nil || innerOp == nil || innerRight == nil || innerRight.symbol != integerSym {
		return nil, false
	}
	if colon.startByte+1 != colon.endByte || open.startByte+1 != open.endByte || closeParen.startByte+1 != closeParen.endByte {
		return nil, false
	}
	if int(closeParen.endByte) > len(source) ||
		source[colon.startByte] != ':' ||
		source[open.startByte] != '(' ||
		source[closeParen.startByte] != ')' {
		return nil, false
	}
	if colon.endByte > open.startByte || open.endByte > innerLeft.startByte || innerLeft.endByte > innerOp.startByte {
		return nil, false
	}
	err := newParentNodeInArena(expr.ownerArena, errorSymbol, true, []*Node{colon, open}, nil, 0)
	err.startByte = colon.startByte
	err.startPoint = colon.startPoint
	err.endByte = innerLeft.endByte
	err.endPoint = innerLeft.endPoint
	err.setHasError(true)
	err.setExtra(true)

	expr.symbol = binarySym
	expr.setNamed(true)
	expr.children = cloneNodeSliceInArena(expr.ownerArena, []*Node{left, err, innerOp, innerRight})
	expr.fieldIDs = nil
	expr.fieldSources = nil
	expr.productionID = 0
	expr.endByte = innerRight.endByte
	expr.endPoint = innerRight.endPoint
	expr.setHasError(true)
	populateParentNode(expr, expr.children)

	ret.endByte = expr.endByte
	ret.endPoint = expr.endPoint
	ret.setHasError(true)
	return closeParen, true
}

func juliaRewriteBareReturnRange(ret, expr, left, colon, right *Node, source []byte, quoteSym Symbol) bool {
	if ret == nil || expr == nil || left == nil || colon == nil || right == nil {
		return false
	}
	if resultChildCount(left) != 0 || resultChildCount(right) != 0 {
		return false
	}
	if colon.startByte+1 != colon.endByte || int(colon.endByte) > len(source) || source[colon.startByte] != ':' {
		return false
	}
	if left.endByte > colon.startByte || colon.endByte > right.startByte {
		return false
	}
	err := newLeafNodeInArena(expr.ownerArena, errorSymbol, true, left.startByte, left.endByte, left.startPoint, left.endPoint)
	err.setHasError(true)
	err.setExtra(true)

	expr.symbol = quoteSym
	expr.setNamed(true)
	expr.startByte = colon.startByte
	expr.startPoint = colon.startPoint
	expr.children = cloneNodeSliceInArena(expr.ownerArena, []*Node{colon, right})
	expr.fieldIDs = nil
	expr.fieldSources = nil
	expr.productionID = 0
	expr.setHasError(false)
	populateParentNode(expr, expr.children)

	ret.children = cloneNodeSliceInArena(ret.ownerArena, []*Node{resultChildAt(ret, 0), err, expr})
	ret.fieldIDs = nil
	ret.fieldSources = nil
	ret.setHasError(true)
	populateParentNode(ret, ret.children)
	return true
}
