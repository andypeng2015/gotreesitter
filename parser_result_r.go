package gotreesitter

func normalizeRCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "r" || len(source) == 0 {
		return
	}
	normalizeRStringContentEscapes(root, source, lang)
}

func normalizeRStringContentEscapes(root *Node, source []byte, lang *Language) {
	stringSym, ok := symbolByName(lang, "string")
	if !ok {
		return
	}
	contentSym, ok := symbolByName(lang, "string_content")
	if !ok {
		return
	}
	escapeSym, ok := symbolByName(lang, "escape_sequence")
	if !ok {
		return
	}
	escapeNamed := symbolIsNamed(lang, escapeSym)

	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != stringSym || n.startByte+2 > n.endByte || int(n.endByte) > len(source) {
			return
		}
		quote := source[n.startByte]
		if quote != '"' && quote != '\'' {
			return
		}
		if source[n.endByte-1] != quote {
			return
		}
		content := rStringContentChild(n, contentSym)
		if content == nil {
			return
		}
		contentStart := n.startByte + 1
		contentEnd := n.endByte - 1
		contentStartPoint := advancePointByBytes(n.startPoint, source[n.startByte:contentStart])
		escapeChildren := rBuildEscapeSequenceChildren(content, source, contentStart, contentEnd, contentStartPoint, escapeSym, escapeNamed)
		replaceNodeChildrenUnfielded(content, escapeChildren)
		content.startByte = contentStart
		content.endByte = contentEnd
		content.startPoint = contentStartPoint
		content.endPoint = advancePointByBytes(contentStartPoint, source[contentStart:contentEnd])
	})
}

func rStringContentChild(n *Node, contentSym Symbol) *Node {
	for i := 0; i < resultChildCount(n); i++ {
		child := resultChildAt(n, i)
		if child != nil && child.symbol == contentSym {
			return child
		}
	}
	return nil
}

func rBuildEscapeSequenceChildren(parent *Node, source []byte, start, end uint32, startPoint Point, escapeSym Symbol, escapeNamed bool) []*Node {
	if parent == nil || start >= end || int(end) > len(source) {
		return nil
	}
	var children []*Node
	point := startPoint
	last := start
	for i := start; i < end; i++ {
		if source[i] != '\\' {
			continue
		}
		point = advancePointByBytes(point, source[last:i])
		seqEnd := rEscapeSequenceEnd(source, i, end)
		if seqEnd <= i {
			continue
		}
		seqEndPoint := advancePointByBytes(point, source[i:seqEnd])
		child := newLeafNodeInArena(
			parent.ownerArena,
			escapeSym,
			escapeNamed,
			i,
			seqEnd,
			point,
			seqEndPoint,
		)
		child.parent = parent
		child.childIndex = int32(len(children))
		children = append(children, child)
		point = seqEndPoint
		last = seqEnd
		i = seqEnd - 1
	}
	return cloneNodeSliceInArena(parent.ownerArena, children)
}

func rEscapeSequenceEnd(source []byte, start, limit uint32) uint32 {
	if start >= limit || int(limit) > len(source) || source[start] != '\\' {
		return start
	}
	i := start + 1
	if i >= limit {
		return i
	}
	switch source[i] {
	case 'x':
		end := rConsumeHex(source, i+1, limit, 2)
		if end == i+1 {
			return start
		}
		return end
	case 'u':
		if end, ok := rConsumeUnicodeEscape(source, i+1, limit, 4); ok {
			return end
		}
		return start
	case 'U':
		if end, ok := rConsumeUnicodeEscape(source, i+1, limit, 8); ok {
			return end
		}
		return start
	}
	if source[i] >= '0' && source[i] <= '7' {
		for count := 0; i < limit && count < 3 && source[i] >= '0' && source[i] <= '7'; count++ {
			i++
		}
		return i
	}
	if source[i] >= '0' && source[i] <= '9' {
		return start
	}
	return i + 1
}

func rConsumeUnicodeEscape(source []byte, start, limit uint32, maxDigits int) (uint32, bool) {
	if start < limit && source[start] == '{' {
		i := start + 1
		digitStart := i
		for count := 0; i < limit && count < maxDigits && rIsASCIIHex(source[i]); count++ {
			i++
		}
		if i > digitStart && i < limit && source[i] == '}' {
			return i + 1, true
		}
		return start, false
	}
	end := rConsumeHex(source, start, limit, maxDigits)
	if end == start {
		return start, false
	}
	return end, true
}

func rConsumeHex(source []byte, start, limit uint32, maxDigits int) uint32 {
	i := start
	for count := 0; i < limit && count < maxDigits && rIsASCIIHex(source[i]); count++ {
		i++
	}
	return i
}

func rIsASCIIHex(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'f') ||
		(b >= 'A' && b <= 'F')
}
