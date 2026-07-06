package gotreesitter

func normalizeLinkerscriptCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "linkerscript" {
		return
	}
	normalizeLinkerscriptErrorNamedness(root, lang)
	normalizeLinkerscriptEmptyRootSpan(root, source, lang)
}

func normalizeLinkerscriptErrorNamedness(root *Node, lang *Language) {
	walkResultTreeBounded(root, func(n *Node) {
		if n != nil && n.symbol == errorSymbol {
			n.setNamed(true)
		}
	})
}

func normalizeLinkerscriptEmptyRootSpan(root *Node, source []byte, lang *Language) {
	if len(source) == 0 || root.Type(lang) != "linkerscript" || root.startByte != 0 || root.endByte != 0 {
		return
	}
	count := resultChildCount(root)
	if count == 0 {
		return
	}
	last := resultChildAt(root, count-1)
	if last == nil || last.endByte == 0 || last.endByte > uint32(len(source)) {
		return
	}
	extendNodeEndTo(root, uint32(len(source)), source)
}
