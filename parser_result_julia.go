package gotreesitter

func normalizeJuliaCompatibility(root *Node, source []byte, lang *Language) {
	normalizeJuliaRecoveredReturnRange(root, source, lang)
	normalizeJuliaMacroArgumentJuxtaposition(root, source, lang)
}

func normalizeJuliaMacroArgumentJuxtaposition(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "julia" || len(source) == 0 {
		return
	}
	argListSym, ok := symbolByName(lang, "macro_argument_list")
	if !ok {
		return
	}
	binarySym, ok := symbolByName(lang, "binary_expression")
	if !ok {
		return
	}
	juxtapositionSym, ok := symbolByName(lang, "juxtaposition_expression")
	if !ok {
		return
	}
	integerSym, ok := symbolByName(lang, "integer_literal")
	if !ok {
		return
	}
	stringSym, ok := symbolByName(lang, "string_literal")
	if !ok {
		return
	}
	walkResultTree(root, func(argList *Node) {
		if argList == nil || argList.symbol != argListSym || resultChildCount(argList) != 2 {
			return
		}
		first := resultChildAt(argList, 0)
		second := resultChildAt(argList, 1)
		switch {
		case juliaFuseLeadingMacroInteger(argList, first, second, source, binarySym, juxtapositionSym, integerSym):
		case juliaFuseTrailingMacroString(argList, first, second, source, binarySym, juxtapositionSym, integerSym, stringSym):
		}
	})
}

func juliaFuseLeadingMacroInteger(argList, first, second *Node, source []byte, binarySym, juxtapositionSym, integerSym Symbol) bool {
	if argList == nil || first == nil || second == nil || first.symbol != integerSym || second.symbol != binarySym || resultChildCount(second) != 3 {
		return false
	}
	left := resultChildAt(second, 0)
	op := resultChildAt(second, 1)
	right := resultChildAt(second, 2)
	if left == nil || op == nil || right == nil {
		return false
	}
	if first.endByte != left.startByte || int(left.startByte) > len(source) {
		return false
	}
	juxtaposition := newParentNodeInArena(argList.ownerArena, juxtapositionSym, true, cloneNodeSliceInArena(argList.ownerArena, []*Node{first, left}), nil, 0)
	replaceNodeChildrenUnfielded(second, cloneNodeSliceInArena(second.ownerArena, []*Node{juxtaposition, op, right}))
	second.productionID = 0
	replaceNodeChildrenUnfielded(argList, cloneNodeSliceInArena(argList.ownerArena, []*Node{second}))
	return true
}

func juliaFuseTrailingMacroString(argList, first, second *Node, source []byte, binarySym, juxtapositionSym, integerSym, stringSym Symbol) bool {
	if argList == nil || first == nil || second == nil || first.symbol != binarySym || second.symbol != stringSym {
		return false
	}
	if !juliaFuseTrailingMacroStringInBinary(first, second, source, binarySym, juxtapositionSym, integerSym) {
		return false
	}
	replaceNodeChildrenUnfielded(argList, cloneNodeSliceInArena(argList.ownerArena, []*Node{first}))
	return true
}

func juliaFuseTrailingMacroStringInBinary(binary, str *Node, source []byte, binarySym, juxtapositionSym, integerSym Symbol) bool {
	if binary == nil || str == nil || binary.symbol != binarySym || resultChildCount(binary) != 3 {
		return false
	}
	children := resultChildSliceForMutation(binary)
	if len(children) != 3 {
		return false
	}
	right := children[2]
	if right == nil {
		return false
	}
	if right.symbol == binarySym && juliaFuseTrailingMacroStringInBinary(right, str, source, binarySym, juxtapositionSym, integerSym) {
		replaceNodeChildrenUnfielded(binary, cloneNodeSliceInArena(binary.ownerArena, children))
		binary.productionID = 0
		return true
	}
	if right.symbol != integerSym || !juliaMacroWhitespaceGap(source, right.endByte, str.startByte) {
		return false
	}
	juxtaposition := newParentNodeInArena(binary.ownerArena, juxtapositionSym, true, cloneNodeSliceInArena(binary.ownerArena, []*Node{right, str}), nil, 0)
	children[2] = juxtaposition
	replaceNodeChildrenUnfielded(binary, cloneNodeSliceInArena(binary.ownerArena, children))
	binary.productionID = 0
	return true
}

func juliaMacroWhitespaceGap(source []byte, start, end uint32) bool {
	if start >= end || int(end) > len(source) {
		return false
	}
	for _, b := range source[start:end] {
		if b != ' ' && b != '\t' {
			return false
		}
	}
	return true
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
