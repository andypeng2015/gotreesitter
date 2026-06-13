package gotreesitter

func normalizeWGSLCompatibility(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "wgsl" {
		return
	}
	walkResultTree(root, func(n *Node) {
		normalizeWGSLEmptyReturnSemicolonRecovery(n, lang)
	})
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
