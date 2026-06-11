package gotreesitter

func normalizeIniCompatibility(root *Node, lang *Language) {
	normalizeIniSectionStarts(root, lang)
	normalizeIniDocumentBlanks(root, lang)
}

func normalizeIniSectionStarts(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "ini" {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n.Type(lang) == "section" {
			for i := 0; i < resultChildCount(n); i++ {
				child := resultChildAt(n, i)
				if child == nil {
					continue
				}
				if n.startByte < child.startByte {
					n.startByte = child.startByte
					n.startPoint = child.startPoint
				}
				break
			}
		}
	})
}

func normalizeIniDocumentBlanks(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "ini" || root.Type(lang) != "document" {
		return
	}
	children := resultChildSliceForMutation(root)
	if len(children) == 0 {
		return
	}
	out := make([]*Node, 0, len(children))
	changed := false
	for _, child := range children {
		if child != nil && child.Type(lang) == "_blank" {
			changed = true
			continue
		}
		out = append(out, child)
	}
	if !changed {
		return
	}
	replaceNodeChildrenUnfielded(root, cloneNodeSliceIfArena(root.ownerArena, out))
}
