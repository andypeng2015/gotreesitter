package gotreesitter

func normalizeOCamlCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "ocaml" || len(source) == 0 {
		return
	}
	normalizeOCamlCollapsedNamedLeafChildren(root, source, lang)
}

var ocamlCollapsedNamedLeafChildren = map[string][]string{
	"boolean":      {"true", "false"},
	"and_operator": {"&", "&&"},
	"or_operator":  {"or", "||"},
}

func normalizeOCamlCollapsedNamedLeafChildren(root *Node, source []byte, lang *Language) {
	parentRules, childNamed := ocamlCollapsedNamedLeafSymbols(lang)
	if len(parentRules) == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		childSyms := parentRules[n.symbol]
		if len(childSyms) == 0 || resultChildCount(n) != 0 ||
			int(n.startByte) > len(source) || int(n.endByte) > len(source) || n.startByte > n.endByte {
			return
		}
		childSym, ok := childSyms[string(source[n.startByte:n.endByte])]
		if !ok {
			return
		}
		child := newLeafNodeInArena(n.ownerArena, childSym, childNamed[childSym], n.startByte, n.endByte, n.startPoint, n.endPoint)
		child.parent = n
		child.childIndex = 0
		n.children = cloneNodeSliceInArena(n.ownerArena, []*Node{child})
	})
}

func ocamlCollapsedNamedLeafSymbols(lang *Language) (map[Symbol]map[string]Symbol, map[Symbol]bool) {
	if lang == nil {
		return nil, nil
	}
	parentRules := make(map[Symbol]map[string]Symbol, len(ocamlCollapsedNamedLeafChildren))
	childNamed := make(map[Symbol]bool)
	for parentName, childNames := range ocamlCollapsedNamedLeafChildren {
		parentSym, ok := lang.symbolByNameAndNamed(parentName, true)
		if !ok {
			parentSym, ok = symbolByName(lang, parentName)
			if !ok {
				continue
			}
		}
		childSyms := make(map[string]Symbol, len(childNames))
		for _, childName := range childNames {
			childSym, ok := lang.symbolByNameAndNamed(childName, false)
			if !ok {
				childSym, ok = symbolByName(lang, childName)
				if !ok {
					continue
				}
			}
			childSyms[childName] = childSym
			childNamed[childSym] = symbolIsNamed(lang, childSym)
		}
		if len(childSyms) > 0 {
			parentRules[parentSym] = childSyms
		}
	}
	return parentRules, childNamed
}
