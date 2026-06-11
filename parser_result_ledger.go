package gotreesitter

import "bytes"

func normalizeLedgerCompatibility(root *Node, source []byte, p *Parser, lang *Language) {
	if root == nil || lang == nil || lang.Name != "ledger" || len(source) == 0 {
		return
	}
	normalizeLedgerRecoveredDateSuffix(root, source, lang)
	normalizeLedgerShatteredRoot(root, source, p, lang)
}

func normalizeLedgerRecoveredDateSuffix(root *Node, source []byte, lang *Language) {
	walkResultTree(root, func(n *Node) {
		if n == nil || n.Type(lang) != "plain_xact" || n.ownerArena == nil {
			return
		}
		children := resultChildSliceForMutation(n)
		for i := 0; i+1 < len(children); i++ {
			date := children[i]
			next := children[i+1]
			if date == nil || next == nil || date.Type(lang) != "date" || next.startByte <= date.endByte {
				continue
			}
			if !ledgerBytesAllDigits(source, date.endByte, next.startByte) {
				continue
			}
			err := ledgerNewLeaf(n.ownerArena, errorSymbol, true, date.endByte, next.startByte, source)
			err.setExtra(true)
			err.setHasError(true)

			out := make([]*Node, 0, len(children)+1)
			out = append(out, children[:i+1]...)
			out = append(out, err)
			out = append(out, children[i+1:]...)
			replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(n.ownerArena, out))
			ledgerMarkErrorAncestors(n)
			return
		}
	})
}

func normalizeLedgerShatteredRoot(root *Node, source []byte, p *Parser, lang *Language) {
	if p == nil || root.ownerArena == nil || root.Type(lang) != "ERROR" {
		return
	}
	sourceFileSym, ok := symbolByName(lang, "source_file")
	if !ok {
		return
	}
	children, ok := ledgerRecoverTopLevelChildren(source, p, lang, root.ownerArena)
	if !ok || len(children) == 0 {
		return
	}
	retagResultRoot(root, sourceFileSym, symbolIsNamed(lang, sourceFileSym))
	root.startByte = 0
	root.startPoint = Point{}
	root.endByte = uint32(len(source))
	root.endPoint = advancePointByBytes(Point{}, source)
	replaceNodeChildrenUnfielded(root, cloneNodeSliceInArena(root.ownerArena, children))
	ledgerMarkErrorAncestors(root)
}

func ledgerRecoverTopLevelChildren(source []byte, p *Parser, lang *Language, arena *nodeArena) ([]*Node, bool) {
	var out []*Node
	for pos := uint32(0); pos < uint32(len(source)); {
		lineEnd, contentEnd := ledgerLineBounds(source, pos)
		if contentEnd == pos {
			if lineEnd > pos {
				out = append(out, ledgerNewLeaf(arena, ledgerSymbol(lang, "\n"), false, pos, lineEnd, source))
			}
			pos = lineEnd
			continue
		}
		line := source[pos:contentEnd]
		if ledgerIsYearDirective(line) {
			err := ledgerNewYearError(arena, lang, source, pos, contentEnd)
			if err == nil {
				return nil, false
			}
			out = append(out, err)
			if lineEnd > contentEnd {
				out = append(out, ledgerNewLeaf(arena, ledgerSymbol(lang, "\n"), false, contentEnd, lineEnd, source))
			}
			pos = lineEnd
			continue
		}

		chunkEnd := lineEnd
		if ledgerLineStartsWith(line, "test") {
			var ok bool
			chunkEnd, ok = ledgerFindTestBlockEnd(source, pos)
			if !ok {
				return nil, false
			}
		} else if ledgerLineStartsWithDigit(line) {
			chunkEnd = ledgerFindTransactionEnd(source, lineEnd)
		}

		nodes, ok := ledgerParseTopLevelChunk(source, pos, chunkEnd, p, lang, arena)
		if !ok {
			return nil, false
		}
		out = append(out, nodes...)
		pos = chunkEnd
	}
	return out, true
}

func ledgerParseTopLevelChunk(source []byte, start, end uint32, p *Parser, lang *Language, arena *nodeArena) ([]*Node, bool) {
	if start >= end || end > uint32(len(source)) {
		return nil, false
	}
	tree, err := p.parseForRecovery(source[start:end])
	if err != nil || tree == nil || tree.RootNode() == nil {
		if tree != nil {
			tree.Release()
		}
		return nil, false
	}
	defer tree.Release()
	startPoint := advancePointByBytes(Point{}, source[:start])
	offsetRoot := tree.RootNodeWithOffset(start, startPoint)
	if offsetRoot == nil || offsetRoot.Type(lang) != "source_file" {
		return nil, false
	}
	children := resultChildSliceForMutation(offsetRoot)
	out := make([]*Node, 0, len(children))
	for _, child := range children {
		if child != nil {
			out = append(out, cloneTreeNodesIntoArena(child, arena))
		}
	}
	return out, len(out) > 0
}

func ledgerNewYearError(arena *nodeArena, lang *Language, source []byte, start, end uint32) *Node {
	ySym := ledgerSymbol(lang, "Y")
	if ySym == 0 || start >= end {
		return nil
	}
	y := ledgerNewLeaf(arena, ySym, false, start, start+1, source)
	err := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceInArena(arena, []*Node{y}), nil, 0)
	err.startByte = start
	err.startPoint = advancePointByBytes(Point{}, source[:start])
	err.endByte = end
	err.endPoint = advancePointByBytes(Point{}, source[:end])
	err.setExtra(true)
	err.setHasError(true)
	return err
}

func ledgerLineBounds(source []byte, start uint32) (lineEnd, contentEnd uint32) {
	end := uint32(len(source))
	for i := start; i < end; i++ {
		if source[i] == '\n' {
			return i + 1, i
		}
	}
	return end, end
}

func ledgerFindTransactionEnd(source []byte, pos uint32) uint32 {
	for pos < uint32(len(source)) {
		lineEnd, contentEnd := ledgerLineBounds(source, pos)
		if contentEnd == pos {
			return pos
		}
		pos = lineEnd
	}
	return uint32(len(source))
}

func ledgerFindTestBlockEnd(source []byte, start uint32) (uint32, bool) {
	for pos := start; pos < uint32(len(source)); {
		lineEnd, contentEnd := ledgerLineBounds(source, pos)
		line := source[pos:contentEnd]
		if bytes.Equal(bytes.TrimSpace(line), []byte("end test")) {
			return contentEnd, true
		}
		pos = lineEnd
	}
	return 0, false
}

func ledgerIsYearDirective(line []byte) bool {
	if len(line) < 2 || line[0] != 'Y' {
		return false
	}
	for _, b := range line[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func ledgerLineStartsWith(line []byte, word string) bool {
	if !bytes.HasPrefix(line, []byte(word)) {
		return false
	}
	return len(line) == len(word) || line[len(word)] == ' ' || line[len(word)] == '\t'
}

func ledgerLineStartsWithDigit(line []byte) bool {
	return len(line) > 0 && line[0] >= '0' && line[0] <= '9'
}

func ledgerBytesAllDigits(source []byte, start, end uint32) bool {
	if start >= end || end > uint32(len(source)) {
		return false
	}
	for _, b := range source[start:end] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

func ledgerNewLeaf(arena *nodeArena, sym Symbol, named bool, start, end uint32, source []byte) *Node {
	return newLeafNodeInArena(arena, sym, named, start, end,
		advancePointByBytes(Point{}, source[:start]),
		advancePointByBytes(Point{}, source[:end]))
}

func ledgerSymbol(lang *Language, name string) Symbol {
	sym, _ := symbolByName(lang, name)
	return sym
}

func ledgerMarkErrorAncestors(n *Node) {
	for cur := n; cur != nil; cur = cur.parent {
		cur.setHasError(true)
		nodeBumpEquivVersion(cur)
	}
}
