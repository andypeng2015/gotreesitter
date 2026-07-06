package gotreesitter

func normalizeRobotCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || len(source) == 0 {
		return
	}
	scalarSym, ok := lang.SymbolByName("scalar_variable")
	if !ok {
		return
	}
	variableNameSym, ok := lang.SymbolByName("variable_name")
	if !ok {
		return
	}
	closeBraceSym, ok := lang.SymbolByName("}")
	if !ok {
		return
	}
	robotNormalizeEscapedNestedVariables(root, source, scalarSym, variableNameSym, closeBraceSym)
}

func robotNormalizeEscapedNestedVariables(n *Node, source []byte, scalarSym, variableNameSym, closeBraceSym Symbol) bool {
	if n == nil {
		return false
	}
	changed := false
	for i := 0; i < resultChildCount(n); i++ {
		if robotNormalizeEscapedNestedVariables(resultChildAt(n, i), source, scalarSym, variableNameSym, closeBraceSym) {
			changed = true
		}
	}
	if n.symbol == scalarSym && robotInsertEscapedNestedVariableErrors(n, source, variableNameSym, closeBraceSym) {
		changed = true
	}
	if changed {
		n.setHasError(true)
	}
	return changed
}

func robotInsertEscapedNestedVariableErrors(n *Node, source []byte, variableNameSym, closeBraceSym Symbol) bool {
	children := resultChildSliceForMutation(n)
	if len(children) < 3 {
		return false
	}
	out := make([]*Node, 0, len(children)+1)
	changed := false
	for i, child := range children {
		out = append(out, child)
		if i+1 >= len(children) || child == nil {
			continue
		}
		next := children[i+1]
		if next == nil || child.symbol != variableNameSym || next.symbol != closeBraceSym {
			continue
		}
		if child.endByte >= next.startByte || int(next.startByte) > len(source) {
			continue
		}
		if !robotVariableNameEndsEscapedDollar(child, source) || source[child.endByte] != '{' {
			continue
		}
		inner := newLeafNodeInArena(
			n.ownerArena,
			errorSymbol,
			true,
			child.endByte,
			child.endByte+1,
			child.endPoint,
			advancePointByBytes(child.endPoint, source[child.endByte:child.endByte+1]),
		)
		inner.setHasError(true)
		err := newParentNodeInArena(
			n.ownerArena,
			errorSymbol,
			true,
			cloneNodeSliceInArena(n.ownerArena, []*Node{inner}),
			nil,
			0,
		)
		err.startByte = child.endByte
		err.endByte = next.startByte
		err.startPoint = child.endPoint
		err.endPoint = advancePointByBytes(child.endPoint, source[child.endByte:next.startByte])
		err.setExtra(true)
		err.setHasError(true)
		out = append(out, err)
		changed = true
	}
	if !changed {
		return false
	}
	replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(n.ownerArena, out))
	n.setHasError(true)
	return true
}

func robotVariableNameEndsEscapedDollar(n *Node, source []byte) bool {
	if n == nil || n.endByte < n.startByte+2 || int(n.endByte) > len(source) {
		return false
	}
	return source[n.endByte-2] == '\\' && source[n.endByte-1] == '$'
}
