package gotreesitter

func normalizeHaskellCompatibility(root *Node, source []byte, lang *Language) {
	normalizeHaskellImportsSpan(root, source, lang)
	normalizeHaskellZeroWidthTokens(root, lang)
	normalizeHaskellRootImportField(root, lang)
	normalizeHaskellDeclarationsSpan(root, source, lang)
	normalizeHaskellLocalBindsStarts(root, source, lang)
	normalizeHaskellQuasiquoteStarts(root, source, lang)
}
func normalizeHaskellImportsSpan(root *Node, source []byte, lang *Language) {
	childCount := resultChildCount(root)
	if root == nil || childCount < 2 || len(source) == 0 || lang == nil || lang.Name != "haskell" {
		return
	}
	for i := 0; i+1 < childCount; i++ {
		left := resultChildAt(root, i)
		right := resultChildAt(root, i+1)
		if left == nil || right == nil {
			continue
		}
		if left.Type(lang) != "imports" {
			continue
		}
		if left.endByte >= right.startByte {
			continue
		}
		if left.endByte > uint32(len(source)) || right.startByte > uint32(len(source)) {
			continue
		}
		gap := source[left.endByte:right.startByte]
		if !bytesAreTrivia(gap) {
			continue
		}
		left.endByte = right.startByte
		left.endPoint = advancePointByBytes(left.endPoint, gap)
	}
}

func normalizeHaskellZeroWidthTokens(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "haskell" || resultChildCount(root) == 0 {
		return
	}
	children := resultDenseChildrenForMutation(root)
	filtered := children[:0]
	for _, child := range children {
		if child == nil {
			continue
		}
		if child.Type(lang) == "_token1" && child.startByte == child.endByte {
			continue
		}
		filtered = append(filtered, child)
	}
	root.children = filtered
}

func normalizeHaskellRootImportField(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "haskell" || resultChildCount(root) == 0 {
		return
	}
	if len(lang.FieldNames) == 0 {
		return
	}
	children := resultDenseChildrenForMutation(root)
	for i, child := range children {
		if child == nil {
			continue
		}
		fid := FieldID(0)
		for j, name := range lang.FieldNames {
			if name == child.Type(lang) {
				fid = FieldID(j)
				break
			}
		}
		if fid == 0 {
			continue
		}
		if len(root.fieldIDs) < len(children) {
			fieldIDs := make([]FieldID, len(children))
			copy(fieldIDs, root.fieldIDs)
			root.fieldIDs = fieldIDs
		}
		if len(root.fieldSources) < len(children) {
			fieldSources := make([]uint8, len(children))
			copy(fieldSources, root.fieldSources)
			root.fieldSources = fieldSources
		}
		root.fieldIDs[i] = fid
		root.fieldSources[i] = fieldSourceInherited
	}
}

func normalizeHaskellDeclarationsSpan(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "haskell" || len(source) == 0 {
		return
	}
	for i := 0; i < resultChildCount(root); i++ {
		child := resultChildAt(root, i)
		if child == nil || child.Type(lang) != "declarations" {
			continue
		}
		if child.endByte >= root.endByte || root.endByte > uint32(len(source)) {
			continue
		}
		gap := source[child.endByte:root.endByte]
		if !bytesAreTrivia(gap) {
			continue
		}
		extendNodeEndTo(child, root.endByte, source)
	}
}

func normalizeHaskellLocalBindsStarts(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "haskell" || len(source) == 0 {
		return
	}
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if n.Type(lang) == "let_in" && resultChildCount(n) >= 2 {
			letNode := resultChildAt(n, 0)
			localBinds := resultChildAt(n, 1)
			if letNode != nil && localBinds != nil && letNode.Type(lang) == "let" && localBinds.Type(lang) == "local_binds" && letNode.endByte < localBinds.startByte && localBinds.startByte <= uint32(len(source)) {
				gap := source[letNode.endByte:localBinds.startByte]
				if len(gap) > 0 && bytesAreTrivia(gap) && !bytesContainLineBreak(gap) {
					localBinds.startByte = letNode.endByte
					localBinds.startPoint = letNode.endPoint
				}
			}
		}
		for i := 0; i < resultChildCount(n); i++ {
			walk(resultChildAt(n, i))
		}
	}
	walk(root)
}

func normalizeHaskellQuasiquoteStarts(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "haskell" || len(source) == 0 {
		return
	}
	var walk func(*Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		if n.Type(lang) == "quasiquote" && n.startByte > 0 {
			start := int(n.startByte)
			if source[start-1] == ' ' && start < len(source) && source[start] == '[' {
				n.startByte--
				if n.startPoint.Column > 0 {
					n.startPoint.Column--
				} else if n.startPoint.Row > 0 {
					n.startPoint = advancePointByBytes(Point{}, source[:n.startByte])
				}
			}
		}
		for i := 0; i < resultChildCount(n); i++ {
			walk(resultChildAt(n, i))
		}
	}
	walk(root)
}
