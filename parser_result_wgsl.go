package gotreesitter

func normalizeWGSLCompatibility(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "wgsl" {
		return
	}
	walkResultTree(root, func(n *Node) {
		normalizeWGSLEmptyReturnSemicolonRecovery(n, lang)
		normalizeWGSLArgumentListErrorWrapper(n, lang)
		normalizeWGSLTrailingArgumentMissingIdentifier(n, lang)
		normalizeWGSLAtomicArrayRecovery(n, lang)
		normalizeWGSLRecoveredCallLHSWrapper(n, lang)
		normalizeWGSLConstAssignmentRecovery(n, lang)
	})
	refreshWGSLHasError(root)
}

func normalizeWGSLEmptyReturnSemicolonRecovery(n *Node, lang *Language) {
	if n == nil || n.Type(lang) != "compound_statement" || resultChildCount(n) < 2 {
		return
	}
	errSym, errNamed := errorSymbol, true
	children := resultChildSliceForMutation(n)
	if len(children) < 2 {
		return
	}
	out := make([]*Node, 0, len(children))
	changed := false
	for i := 0; i < len(children); i++ {
		cur := children[i]
		if i+1 < len(children) && wgslIsEmptyReturnStatement(cur, lang) {
			next := children[i+1]
			if next != nil && next.Type(lang) == ";" && next.StartByte() == cur.StartByte() && next.EndByte() == cur.StartByte()+1 {
				err := newParentNodeInArena(n.ownerArena, errSym, errNamed, []*Node{next}, nil, 0)
				err.setHasError(true)
				out = append(out, err)
				i++
				changed = true
				continue
			}
		}
		out = append(out, cur)
	}
	if changed {
		replaceNodeChildrenUnfielded(n, out)
		n.setHasError(true)
	}
}

func wgslIsEmptyReturnStatement(n *Node, lang *Language) bool {
	if n == nil || n.Type(lang) != "return_statement" || n.StartByte() != n.EndByte() {
		return false
	}
	switch resultChildCount(n) {
	case 0:
		return true
	case 1:
		child := resultChildAt(n, 0)
		return child != nil && child.Type(lang) == "return" && child.IsMissing() &&
			child.StartByte() == n.StartByte() && child.EndByte() == n.EndByte()
	default:
		return false
	}
}

func normalizeWGSLArgumentListErrorWrapper(n *Node, lang *Language) {
	if n == nil || n.Type(lang) != "argument_list_expression" || resultChildCount(n) != 3 {
		return
	}
	open := resultChildAt(n, 0)
	err := resultChildAt(n, 1)
	close := resultChildAt(n, 2)
	if open == nil || open.Type(lang) != "(" || err == nil || !err.IsError() || close == nil || close.Type(lang) != ")" {
		return
	}
	if err.StartByte() != open.EndByte() || err.EndByte() != close.StartByte() || resultChildCount(err) == 0 {
		return
	}
	errChildren := resultChildSliceForMutation(err)
	if !wgslLooksLikeArgumentSequence(errChildren, lang) {
		return
	}
	out := make([]*Node, 0, len(errChildren)+2)
	out = append(out, open)
	out = append(out, errChildren...)
	out = append(out, close)
	replaceNodeChildrenUnfielded(n, out)
}

func wgslLooksLikeArgumentSequence(children []*Node, lang *Language) bool {
	wantExpr := true
	seenExpr := false
	for _, child := range children {
		if child == nil {
			return false
		}
		if wantExpr {
			if child.Type(lang) == "," || child.Type(lang) == ")" || child.IsError() || child.HasError() {
				return false
			}
			seenExpr = true
			wantExpr = false
			continue
		}
		if child.Type(lang) != "," {
			return false
		}
		wantExpr = true
	}
	return seenExpr && !wantExpr
}

func normalizeWGSLTrailingArgumentMissingIdentifier(n *Node, lang *Language) {
	if n == nil || n.Type(lang) != "argument_list_expression" || resultChildCount(n) < 4 {
		return
	}
	children := resultChildSliceForMutation(n)
	if len(children) < 4 {
		return
	}
	close := children[len(children)-1]
	missing := children[len(children)-2]
	comma := children[len(children)-3]
	if close == nil || close.Type(lang) != ")" ||
		missing == nil || missing.Type(lang) != "identifier" || missing.StartByte() != missing.EndByte() ||
		comma == nil || comma.Type(lang) != "," || comma.EndByte() != missing.StartByte() {
		return
	}
	out := make([]*Node, 0, len(children)-1)
	out = append(out, children[:len(children)-2]...)
	out = append(out, close)
	replaceNodeChildrenUnfielded(n, out)
}

func normalizeWGSLAtomicArrayRecovery(n *Node, lang *Language) {
	if n == nil || n.Type(lang) != "type_declaration" || resultChildCount(n) != 5 {
		return
	}
	children := resultChildSliceForMutation(n)
	if len(children) != 5 ||
		children[0] == nil || children[0].Type(lang) != "array" ||
		children[1] == nil || !children[1].IsError() ||
		children[2] == nil || children[2].Type(lang) != "<" ||
		children[3] == nil || children[3].Type(lang) != "type_declaration" ||
		children[4] == nil || children[4].Type(lang) != ">" {
		return
	}
	errChildren := resultChildSliceForMutation(children[1])
	if len(errChildren) != 2 ||
		errChildren[0] == nil || errChildren[0].Type(lang) != "<" ||
		errChildren[1] == nil || children[1].EndByte() != children[2].StartByte() {
		return
	}
	atomicType := errChildren[1]
	if atomicType.Type(lang) != "type_declaration" || resultChildCount(atomicType) != 1 {
		return
	}
	atomicIdent := resultChildAt(atomicType, 0)
	if atomicIdent == nil || atomicIdent.Type(lang) != "identifier" || atomicIdent.StartByte() != errChildren[0].EndByte() {
		return
	}
	open := errChildren[0]
	nestedOpen := children[2]
	open.setHasError(false)
	atomicType.setHasError(false)
	atomicIdent.setHasError(false)
	nestedOpen.setHasError(false)
	err := newParentNodeInArena(n.ownerArena, errorSymbol, true, []*Node{atomicType, nestedOpen}, nil, 0)
	err.setHasError(true)
	err.setExtra(true)
	replaceNodeChildrenUnfielded(n, []*Node{children[0], open, err, children[3], children[4]})
}

func normalizeWGSLRecoveredCallLHSWrapper(n *Node, lang *Language) {
	if n == nil || !n.IsError() || resultChildCount(n) == 0 {
		return
	}
	children := resultChildSliceForMutation(n)
	changed := false
	for i, child := range children {
		if child == nil || child.Type(lang) != "lhs_expression" || resultChildCount(child) != 1 {
			continue
		}
		inner := resultChildAt(child, 0)
		if inner == nil || inner.Type(lang) != "identifier" || inner.StartByte() != child.StartByte() || inner.EndByte() != child.EndByte() {
			continue
		}
		children[i] = inner
		changed = true
	}
	if changed {
		replaceNodeChildrenUnfielded(n, children)
	}
}

func normalizeWGSLConstAssignmentRecovery(n *Node, lang *Language) {
	if n == nil || n.Type(lang) != "compound_statement" || resultChildCount(n) < 2 {
		return
	}
	lhsSym, lhsNamed, ok := symbolMeta(lang, "lhs_expression")
	if !ok {
		return
	}
	children := resultChildSliceForMutation(n)
	out := make([]*Node, 0, len(children))
	changed := false
	for i := 0; i < len(children); i++ {
		cur := children[i]
		if i+1 < len(children) && wgslIsConstKeywordError(cur, lang) && wgslLooksLikeConstAssignmentTail(children[i+1], lang) {
			assign := children[i+1]
			assignChildren := resultChildSliceForMutation(assign)
			constIdent := resultChildAt(cur, 0)
			nameLHS := assignChildren[0]
			nameIdent := resultChildAt(nameLHS, 0)
			constLHS := newParentNodeInArena(n.ownerArena, lhsSym, lhsNamed, []*Node{constIdent}, nil, 0)
			nameErr := newParentNodeInArena(n.ownerArena, errorSymbol, true, []*Node{nameIdent}, nil, 0)
			nameErr.setExtra(true)
			nameErr.setHasError(true)
			rebuiltChildren := make([]*Node, 0, len(assignChildren)+1)
			rebuiltChildren = append(rebuiltChildren, constLHS, nameErr)
			rebuiltChildren = append(rebuiltChildren, assignChildren[1:]...)
			replaceNodeChildrenUnfielded(assign, rebuiltChildren)
			assign.startByte = cur.StartByte()
			assign.startPoint = cur.StartPoint()
			assign.setHasError(true)
			out = append(out, assign)
			i++
			changed = true
			continue
		}
		out = append(out, cur)
	}
	if changed {
		replaceNodeChildrenUnfielded(n, out)
		n.setHasError(true)
	}
}

func wgslIsConstKeywordError(n *Node, lang *Language) bool {
	if n == nil || !n.IsError() || resultChildCount(n) != 1 {
		return false
	}
	child := resultChildAt(n, 0)
	return child != nil && child.Type(lang) == "identifier" && child.StartByte() == n.StartByte() &&
		child.EndByte() == n.EndByte()
}

func wgslLooksLikeConstAssignmentTail(n *Node, lang *Language) bool {
	if n == nil || n.Type(lang) != "assignment_statement" || resultChildCount(n) < 3 {
		return false
	}
	lhs := resultChildAt(n, 0)
	if lhs == nil || lhs.Type(lang) != "lhs_expression" || resultChildCount(lhs) != 1 {
		return false
	}
	name := resultChildAt(lhs, 0)
	eq := resultChildAt(n, 1)
	return name != nil && name.Type(lang) == "identifier" &&
		eq != nil && eq.Type(lang) == "=" &&
		name.EndByte() == name.StartByte()+1 &&
		name.EndByte() < eq.StartByte()
}

func refreshWGSLHasError(n *Node) bool {
	if n == nil {
		return false
	}
	hasErr := n.IsError()
	for i := 0; i < resultChildCount(n); i++ {
		if refreshWGSLHasError(resultChildAt(n, i)) {
			hasErr = true
		}
	}
	n.setHasError(hasErr)
	return hasErr
}
