package gotreesitter

func normalizeBitbakeCompatibility(root *Node, lang *Language) {
	normalizeBitbakeAddtaskErrorWrappers(root, lang)
}

func normalizeBitbakeAddtaskErrorWrappers(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "bitbake" {
		return
	}
	addtaskSym, ok := symbolByName(lang, "addtask_statement")
	if !ok {
		return
	}
	rewriteResultTreeChildrenPostorder(root, func(n *Node) *Node {
		if n == nil || n.symbol != errorSymbol || resultChildCount(n) != 1 {
			return nil
		}
		child := resultChildAt(n, 0)
		if child == nil || child.symbol != addtaskSym ||
			child.startByte != n.startByte || child.endByte != n.endByte ||
			child.startPoint != n.startPoint || child.endPoint != n.endPoint {
			return nil
		}
		return child
	})
}
