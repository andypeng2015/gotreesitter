package gotreesitter

func normalizeLuauCompatibility(root *Node, source []byte, lang *Language) {
	normalizeLuauRecoveredErrorEndIdentifier(root, source, lang)
}

func normalizeLuauRecoveredErrorEndIdentifier(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "luau" || len(source) == 0 {
		return
	}
	endSym, ok := lang.symbolByNameAndNamed("end", false)
	if !ok {
		return
	}
	identifierSym, ok := lang.symbolByNameAndNamed("identifier", true)
	if !ok {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != errorSymbol || resultChildCount(n) != 1 {
			return
		}
		child := resultChildAt(n, 0)
		if child == nil || child.symbol != endSym || child.startByte != n.startByte || child.endByte != n.endByte {
			return
		}
		if int(child.endByte) > len(source) || string(source[child.startByte:child.endByte]) != "end" {
			return
		}
		child.symbol = identifierSym
		child.setNamed(true)
	})
}
