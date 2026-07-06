package gotreesitter

func normalizeCrystalCompatibility(root *Node, source []byte, lang *Language) {
	normalizeCrystalBraceContainerStarts(root, source, lang)
}

func normalizeCrystalBraceContainerStarts(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "crystal" || len(source) == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || !crystalBraceContainerType(n.Type(lang)) || resultChildCount(n) == 0 {
			return
		}
		open := resultChildAt(n, 0)
		if open == nil || open.Type(lang) != "{" || open.IsNamed() {
			return
		}
		if open.startByte+1 != open.endByte || int(open.startByte) >= len(source) || source[open.startByte] != '{' {
			return
		}
		n.startByte = open.endByte
		n.startPoint = open.endPoint
		open.startByte = open.endByte
		open.startPoint = open.endPoint
	})
}

func crystalBraceContainerType(name string) bool {
	switch name {
	case "hash", "named_tuple":
		return true
	default:
		return false
	}
}
