package gotreesitter

// normalizeCollapsedNamedLeafChildren restores collapsed single-anonymous-child
// nodes. When a named node (parentName) wraps a single anonymous token
// (childName) and the collapse logic strips the child, this function
// reconstructs the child so the tree matches C tree-sitter output.
func normalizeCollapsedNamedLeafChildren(root *Node, lang *Language, parentName, childName string) {
	normalizeCollapsedNamedLeafChildrenWithStats(root, lang, parentName, childName)
}

func normalizeCollapsedNamedLeafChildrenBySource(root *Node, source []byte, lang *Language, parentName string, childNames ...string) {
	normalizeCollapsedNamedLeafChildrenBySourceWithStats(root, source, lang, parentName, childNames...)
}

func normalizeCollapsedNamedLeafChildrenWithStats(root *Node, lang *Language, parentName, childName string) normalizationPassCounters {
	var counters normalizationPassCounters
	if root == nil || lang == nil {
		return counters
	}
	parentSym, ok := symbolByName(lang, parentName)
	if !ok {
		return counters
	}
	childSym, childOk := symbolByName(lang, childName)
	if !childOk {
		return counters
	}
	childNamed := false
	if int(childSym) < len(lang.SymbolMetadata) {
		childNamed = lang.SymbolMetadata[childSym].Named
	}
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		counters.nodesVisited++
		childCount := resultChildCount(n)
		if n.symbol == parentSym && childCount == 0 {
			child := newLeafNodeInArena(n.ownerArena, childSym, childNamed, n.startByte, n.endByte, n.startPoint, n.endPoint)
			n.children = cloneNodeSliceInArena(n.ownerArena, []*Node{child})
			counters.nodesRewritten++
		}
		for i := 0; i < childCount; i++ {
			child := resultChildAt(n, i)
			walk(child)
		}
	}
	walk(root)
	return counters
}

func normalizeCollapsedNamedLeafChildrenBySourceWithStats(root *Node, source []byte, lang *Language, parentName string, childNames ...string) normalizationPassCounters {
	var counters normalizationPassCounters
	if root == nil || lang == nil || len(source) == 0 || len(childNames) == 0 {
		return counters
	}
	parentSym, ok := symbolByName(lang, parentName)
	if !ok {
		return counters
	}
	childSyms := make(map[string]Symbol, len(childNames))
	childNamed := make(map[Symbol]bool, len(childNames))
	for _, childName := range childNames {
		childSym, ok := symbolByName(lang, childName)
		if !ok {
			continue
		}
		childSyms[childName] = childSym
		if int(childSym) < len(lang.SymbolMetadata) {
			childNamed[childSym] = lang.SymbolMetadata[childSym].Named
		}
	}
	if len(childSyms) == 0 {
		return counters
	}
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		counters.nodesVisited++
		childCount := resultChildCount(n)
		if n.symbol == parentSym && childCount == 0 && int(n.startByte) <= len(source) && int(n.endByte) <= len(source) && n.startByte <= n.endByte {
			if childSym, ok := childSyms[string(source[n.startByte:n.endByte])]; ok {
				child := newLeafNodeInArena(n.ownerArena, childSym, childNamed[childSym], n.startByte, n.endByte, n.startPoint, n.endPoint)
				n.children = cloneNodeSliceInArena(n.ownerArena, []*Node{child})
				counters.nodesRewritten++
			}
		}
		for i := 0; i < childCount; i++ {
			child := resultChildAt(n, i)
			walk(child)
		}
	}
	walk(root)
	return counters
}
