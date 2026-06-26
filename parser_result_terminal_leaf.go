package gotreesitter

// normalizeResultTerminalLeafNodes removes redundant terminal payload children
// that can be introduced by generalized reduction/materialization paths. C
// tree-sitter exposes terminal and terminal-alias symbols as leaves, so a
// visible leaf-like parent with one visible anonymous terminal child and no
// semantic decorations should keep the parent's symbol/flags and adopt the
// child's token span.
func normalizeResultTerminalLeafNodes(root *Node, lang *Language) normalizationPassCounters {
	var counters normalizationPassCounters
	if root == nil || lang == nil || root.hasError() {
		return counters
	}
	aliasTargets := resultVisibleAliasTargetSet(lang)
	walkResultTree(root, func(n *Node) {
		counters.nodesVisited++
		if !resultCanCollapseTerminalLeafNode(n, lang, aliasTargets) {
			return
		}
		child := resultChildAt(n, 0)
		if !resultCanCollapseTerminalLeafChild(child, lang, aliasTargets) {
			return
		}
		n.startByte = child.startByte
		n.endByte = child.endByte
		n.startPoint = child.startPoint
		n.endPoint = child.endPoint
		n.children = nil
		n.fieldIDs = nil
		n.fieldSources = nil
		if n.ownerArena != nil {
			n.ownerArena.clearFinalChildRefs(n)
		}
		counters.nodesRewritten++
	})
	return counters
}

func resultCanCollapseTerminalLeafNode(n *Node, lang *Language, aliasTargets []bool) bool {
	if n == nil || lang == nil || n.isExtra() || n.isMissing() || n.hasError() {
		return false
	}
	if !resultSymbolIsVisibleTerminalLeafLike(lang, n.symbol, aliasTargets) {
		return false
	}
	if resultChildCount(n) != 1 {
		return false
	}
	return !resultNodeHasChildFields(n)
}

func resultCanCollapseTerminalLeafChild(child *Node, lang *Language, aliasTargets []bool) bool {
	if child == nil || lang == nil || child.isNamed() || child.isExtra() || child.isMissing() || child.hasError() {
		return false
	}
	return resultSymbolIsVisibleTerminalLeafLike(lang, child.symbol, aliasTargets)
}

func resultSymbolIsVisibleTerminalLeafLike(lang *Language, sym Symbol, aliasTargets []bool) bool {
	return resultSymbolIsVisibleTerminal(lang, sym) || resultSymbolIsVisibleAliasTarget(lang, sym, aliasTargets)
}

func resultSymbolIsVisibleTerminal(lang *Language, sym Symbol) bool {
	return lang != nil && uint32(sym) < lang.TokenCount && symbolIsVisible(lang, sym)
}

func resultSymbolIsVisibleAliasTarget(lang *Language, sym Symbol, aliasTargets []bool) bool {
	if lang == nil || int(sym) >= len(aliasTargets) {
		return false
	}
	return aliasTargets[sym]
}

func resultVisibleAliasTargetSet(lang *Language) []bool {
	if lang == nil || len(lang.AliasSequences) == 0 {
		return nil
	}
	aliasTargets := make([]bool, len(lang.SymbolNames))
	for _, seq := range lang.AliasSequences {
		for _, alias := range seq {
			if int(alias) < len(aliasTargets) && symbolIsVisible(lang, alias) {
				aliasTargets[alias] = true
			}
		}
	}
	return aliasTargets
}

func resultNodeHasChildFields(n *Node) bool {
	if n == nil {
		return false
	}
	for _, fid := range n.fieldIDs {
		if fid != 0 {
			return true
		}
	}
	for _, source := range n.fieldSources {
		if source != fieldSourceNone {
			return true
		}
	}
	return false
}
