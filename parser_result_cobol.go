package gotreesitter

import "strings"

func normalizeCobolCompatibility(root *Node, source []byte, lang *Language) {
	normalizeCobolRecoveredRootProgramDefinition(root, source, lang)
	normalizeCobolLeadingAreaStart(root, source, lang)
	normalizeCobolTopLevelDefinitionEnd(root, source, lang)
	normalizeCobolDivisionSiblingEnds(root, source, lang)
	normalizeCobolSectionSiblingEnds(root, source, lang)
	normalizeCobolTrailingTriviaSpans(root, source, lang)
	normalizeCobolRecoveredParagraphHeader(root, source, lang)
}
func normalizeCobolLeadingAreaStart(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || len(source) == 0 {
		return
	}
	rootStart, defStart, ok := cobolLeadingStarts(source)
	if !ok {
		return
	}
	setCobolNodeStart(root, source, rootStart)
	if resultChildCount(root) == 0 {
		return
	}
	def := cobolFirstProgramDefinition(root, lang)
	if def == nil {
		return
	}
	setCobolNodeStart(def, source, defStart)
	if resultChildCount(def) == 0 {
		return
	}
	for i := 0; i < resultChildCount(def); i++ {
		child := resultChildAt(def, i)
		if child != nil && !child.IsExtra() && child.Type(lang) == "identification_division" {
			setCobolNodeStart(child, source, defStart)
			break
		}
	}
}

func cobolLeadingStarts(source []byte) (uint32, uint32, bool) {
	start := firstNonWhitespaceByte(source)
	if start != 0 {
		if cobolByteColumn(source, start) == 6 && cobolFixedFormatCommentIndicator(source[start]) {
			if defStart, ok := cobolFirstFixedFormatCodeStart(source, int(start)); ok {
				return start, defStart, true
			}
		}
		return start, start, true
	}
	if len(source) < 7 {
		return 0, 0, false
	}
	switch source[6] {
	case ' ':
		contentStart, ok := firstNonWhitespaceByteFrom(source, 7)
		if !ok {
			return 0, 0, false
		}
		return 6, contentStart, true
	case '*', '/':
		if defStart, ok := cobolFirstFixedFormatCodeStart(source, 6); ok {
			return 6, defStart, true
		}
		return 6, 6, true
	case '-':
		return 6, 6, true
	default:
		return 0, 0, false
	}
}

func cobolFirstFixedFormatCodeStart(source []byte, start int) (uint32, bool) {
	if start < 0 {
		start = 0
	}
	for lineStart := cobolLineStart(source, start); lineStart < len(source); {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' && source[lineEnd] != '\r' {
			lineEnd++
		}
		if cobolLineHasContent(source, lineStart, lineEnd) {
			if indicator := lineStart + 6; indicator < lineEnd {
				switch source[indicator] {
				case '*', '/':
					lineStart = cobolNextLineStart(source, lineEnd)
					continue
				case ' ':
					if contentStart, ok := firstNonWhitespaceByteInRange(source, indicator+1, lineEnd); ok {
						return contentStart, true
					}
					lineStart = cobolNextLineStart(source, lineEnd)
					continue
				}
			}
			if contentStart, ok := firstNonWhitespaceByteInRange(source, lineStart, lineEnd); ok {
				return contentStart, true
			}
		}
		lineStart = cobolNextLineStart(source, lineEnd)
	}
	return 0, false
}

func cobolLineStart(source []byte, pos int) int {
	if pos > len(source) {
		pos = len(source)
	}
	for pos > 0 && source[pos-1] != '\n' && source[pos-1] != '\r' {
		pos--
	}
	return pos
}

func cobolNextLineStart(source []byte, lineEnd int) int {
	for lineEnd < len(source) {
		ch := source[lineEnd]
		lineEnd++
		if ch == '\n' {
			break
		}
		if ch == '\r' {
			if lineEnd < len(source) && source[lineEnd] == '\n' {
				lineEnd++
			}
			break
		}
	}
	return lineEnd
}

func cobolLineHasContent(source []byte, start, end int) bool {
	_, ok := firstNonWhitespaceByteInRange(source, start, end)
	return ok
}

func firstNonWhitespaceByteInRange(source []byte, start, end int) (uint32, bool) {
	if start < 0 {
		start = 0
	}
	if end > len(source) {
		end = len(source)
	}
	for i := start; i < end; i++ {
		switch source[i] {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return uint32(i), true
		}
	}
	return 0, false
}

func cobolByteColumn(source []byte, pos uint32) int {
	i := int(pos)
	if i > len(source) {
		i = len(source)
	}
	start := cobolLineStart(source, i)
	return i - start
}

func cobolFixedFormatCommentIndicator(b byte) bool {
	return b == '*' || b == '/'
}

func normalizeCobolRecoveredRootProgramDefinition(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || root.Type(lang) != "start" || resultChildCount(root) != 1 {
		return
	}
	recovered := resultChildAt(root, 0)
	if recovered == nil || recovered.symbol != errorSymbol || resultChildCount(recovered) == 0 {
		return
	}
	children := resultChildSliceForMutation(recovered)
	if len(children) == 0 {
		return
	}
	idIdx := cobolFirstChildIndexOfType(children, lang, "identification_division", 0)
	if idIdx < 0 || !cobolOnlyRootLeadingComments(children[:idIdx], lang) {
		return
	}
	dataIdx := cobolFirstChildIndexOfType(children, lang, "data_division", idIdx+1)
	if dataIdx < 0 {
		return
	}
	procStart, ok := cobolProcedureDivisionSourceStart(source, children[dataIdx].startByte)
	if !ok {
		return
	}
	procChildStart := cobolFirstChildIndexStartingAtOrAfter(children, dataIdx+1, procStart)
	if procChildStart < 0 {
		return
	}
	for i := dataIdx + 1; i < procChildStart; i++ {
		child := children[i]
		if child == nil || child.IsExtra() || child.Type(lang) != "comment" {
			return
		}
	}
	trailingStart := cobolRootTrailingCommentStart(children, procChildStart, lang)
	if trailingStart <= procChildStart {
		return
	}
	tailStart := cobolRecoveredTailStart(children, procChildStart, trailingStart, lang)
	if tailStart < procChildStart || tailStart > trailingStart {
		return
	}
	procChildrenEnd := tailStart
	if procChildrenEnd == procChildStart {
		return
	}
	programSym, ok := symbolByName(lang, "program_definition")
	if !ok {
		return
	}
	procedureSym, ok := symbolByName(lang, "procedure_division")
	if !ok {
		return
	}

	rootStart, rootStartPoint := root.startByte, root.startPoint
	rootEnd, rootEndPoint := root.endByte, root.endPoint

	procedureChildren := cloneNodeSliceInArena(root.ownerArena, children[procChildStart:procChildrenEnd])
	procedureNamed := symbolIsNamed(lang, procedureSym)
	procedure := newParentNodeInArena(root.ownerArena, procedureSym, procedureNamed, procedureChildren, nil, 0)
	procedure.startByte = procStart
	procedure.startPoint = advancePointByBytes(Point{}, source[:procStart])
	cobolRefreshHasErrorFromChildren(procedure)

	programChildren := make([]*Node, 0, procChildStart-idIdx+1)
	programChildren = append(programChildren, children[idIdx:procChildStart]...)
	programChildren = append(programChildren, procedure)
	programNamed := symbolIsNamed(lang, programSym)
	program := newParentNodeInArena(root.ownerArena, programSym, programNamed, cloneNodeSliceInArena(root.ownerArena, programChildren), nil, 0)
	cobolRefreshHasErrorFromChildren(program)

	rootChildren := make([]*Node, 0, idIdx+1+1+len(children)-trailingStart)
	rootChildren = append(rootChildren, children[:idIdx]...)
	rootChildren = append(rootChildren, program)
	if tailStart < trailingStart {
		tailChildren := cloneNodeSliceInArena(root.ownerArena, children[tailStart:trailingStart])
		tail := newParentNodeInArena(root.ownerArena, errorSymbol, true, tailChildren, nil, 0)
		tail.setExtra(true)
		tail.setHasError(true)
		normalizeCobolRecoveredTailErrorChildren(tail, source, lang)
		rootChildren = append(rootChildren, tail)
	}
	rootChildren = append(rootChildren, children[trailingStart:]...)
	replaceNodeChildrenUnfielded(root, cloneNodeSliceInArena(root.ownerArena, rootChildren))
	root.startByte = rootStart
	root.startPoint = rootStartPoint
	root.endByte = rootEnd
	root.endPoint = rootEndPoint
	cobolRefreshHasErrorFromChildren(root)
}

func cobolFirstChildIndexOfType(children []*Node, lang *Language, typ string, start int) int {
	if start < 0 {
		start = 0
	}
	for i := start; i < len(children); i++ {
		child := children[i]
		if child != nil && !child.IsExtra() && child.Type(lang) == typ {
			return i
		}
	}
	return -1
}

func cobolOnlyRootLeadingComments(children []*Node, lang *Language) bool {
	for _, child := range children {
		if child == nil || child.IsExtra() || child.Type(lang) != "comment" {
			return false
		}
	}
	return true
}

func cobolProcedureDivisionSourceStart(source []byte, after uint32) (uint32, bool) {
	if int(after) > len(source) {
		return 0, false
	}
	idx := cobolIndexFoldASCII(source, int(after), len(source), "procedure division")
	if idx < 0 {
		return 0, false
	}
	return uint32(idx), true
}

func cobolFirstChildIndexStartingAtOrAfter(children []*Node, start int, byteStart uint32) int {
	if start < 0 {
		start = 0
	}
	for i := start; i < len(children); i++ {
		child := children[i]
		if child != nil && child.startByte >= byteStart {
			return i
		}
	}
	return -1
}

func cobolRootTrailingCommentStart(children []*Node, min int, lang *Language) int {
	i := len(children)
	for i > min {
		child := children[i-1]
		if child == nil || child.IsExtra() || child.Type(lang) != "comment" {
			break
		}
		i--
	}
	return i
}

func cobolRecoveredTailStart(children []*Node, start, end int, lang *Language) int {
	firstErr := -1
	for i := start; i < end; i++ {
		child := children[i]
		if child != nil && (child.IsError() || child.HasError()) {
			firstErr = i
			break
		}
	}
	if firstErr < 0 {
		return end
	}
	tailStart := firstErr
	for i := firstErr - 1; i >= start; i-- {
		child := children[i]
		if child != nil && !child.IsExtra() && child.Type(lang) == "paragraph_header" {
			tailStart = i + 1
			break
		}
	}
	return tailStart
}

func normalizeCobolRecoveredTailErrorChildren(tail *Node, source []byte, lang *Language) {
	if tail == nil || tail.symbol != errorSymbol || int(tail.endByte) > len(source) {
		return
	}
	commentEntrySym, ok := symbolByName(lang, "comment_entry")
	if !ok {
		return
	}
	commentEntryNamed := symbolIsNamed(lang, commentEntrySym)
	children := resultChildSliceForMutation(tail)
	if len(children) == 0 {
		return
	}
	out := make([]*Node, 0, len(children))
	cursor := tail.startByte
	for _, child := range children {
		if child == nil {
			continue
		}
		if cursor < child.startByte {
			out = cobolAppendCommentEntryMarkers(out, tail.ownerArena, source, cursor, child.startByte, commentEntrySym, commentEntryNamed)
			cursor = child.startByte
		}
		if child.IsError() || child.HasError() {
			out = cobolAppendCommentEntryMarkers(out, tail.ownerArena, source, cursor, child.endByte, commentEntrySym, commentEntryNamed)
			cursor = child.endByte
			continue
		}
		out = append(out, child)
		cursor = child.endByte
	}
	if cursor < tail.endByte {
		out = cobolAppendCommentEntryMarkers(out, tail.ownerArena, source, cursor, tail.endByte, commentEntrySym, commentEntryNamed)
	}
	if len(out) == 0 {
		return
	}
	replaceNodeChildrenUnfielded(tail, cloneNodeSliceInArena(tail.ownerArena, out))
	last := out[len(out)-1]
	tail.endByte = last.endByte
	tail.endPoint = last.endPoint
	tail.setExtra(true)
	tail.setHasError(true)
}

func cobolAppendCommentEntryMarkers(out []*Node, arena *nodeArena, source []byte, start, end uint32, sym Symbol, named bool) []*Node {
	if start >= end || int(end) > len(source) {
		return out
	}
	for lineStart := cobolLineStart(source, int(start)); lineStart < int(end); {
		lineEnd := lineStart
		for lineEnd < len(source) && source[lineEnd] != '\n' && source[lineEnd] != '\r' {
			lineEnd++
		}
		if uint32(lineEnd) > start && uint32(lineEnd) <= end && cobolLineShouldEmitCommentEntry(source, lineStart, lineEnd) {
			point := advancePointByBytes(Point{}, source[:lineEnd])
			out = append(out, newLeafNodeInArena(arena, sym, named, uint32(lineEnd), uint32(lineEnd), point, point))
		}
		next := cobolNextLineStart(source, lineEnd)
		if next <= lineStart {
			break
		}
		lineStart = next
	}
	return out
}

func cobolLineShouldEmitCommentEntry(source []byte, lineStart, lineEnd int) bool {
	if !cobolLineHasContent(source, lineStart, lineEnd) {
		return false
	}
	if cobolLineLooksFixedFormatComment(source, lineStart) {
		return false
	}
	return true
}

func firstNonWhitespaceByteFrom(source []byte, start int) (uint32, bool) {
	if start < 0 {
		start = 0
	}
	for i := start; i < len(source); i++ {
		switch source[i] {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return uint32(i), true
		}
	}
	return 0, false
}

func normalizeCobolTopLevelDefinitionEnd(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || root.Type(lang) != "start" || resultChildCount(root) == 0 {
		return
	}
	def := cobolFirstProgramDefinition(root, lang)
	if def == nil {
		return
	}
	if idDiv := cobolSoleIdentificationDivision(def, lang); idDiv != nil {
		end := cobolProgramIDStatementEnd(source, def.startByte, def.endByte)
		if end != 0 && end < def.endByte {
			setCobolNodeEnd(def, source, end)
			if end < idDiv.endByte {
				setCobolNodeEnd(idDiv, source, end)
			}
			return
		}
	}
	end := lastNonTriviaByteEnd(source)
	if end == 0 || end >= def.endByte {
		return
	}
	setCobolNodeEnd(def, source, end)
}

func cobolSoleIdentificationDivision(def *Node, lang *Language) *Node {
	var idDiv *Node
	for i := 0; i < resultChildCount(def); i++ {
		child := resultChildAt(def, i)
		if child == nil || child.IsExtra() {
			continue
		}
		if child.Type(lang) != "identification_division" {
			return nil
		}
		if idDiv != nil {
			return nil
		}
		idDiv = child
	}
	return idDiv
}

func cobolProgramIDStatementEnd(source []byte, start, end uint32) uint32 {
	if start >= end || int(end) > len(source) {
		return 0
	}
	idx := cobolIndexFoldASCII(source, int(start), int(end), "program-id.")
	if idx < 0 {
		return 0
	}
	for i := idx + len("program-id."); i < int(end); i++ {
		switch source[i] {
		case '.':
			return uint32(i + 1)
		case '\n', '\r':
			return 0
		}
	}
	return 0
}

func cobolIndexFoldASCII(source []byte, start, end int, needle string) int {
	if start < 0 {
		start = 0
	}
	if end > len(source) {
		end = len(source)
	}
	if len(needle) == 0 || start >= end || end-start < len(needle) {
		return -1
	}
	for i := start; i+len(needle) <= end; i++ {
		if cobolEqualFoldASCII(source[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func cobolEqualFoldASCII(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := range b {
		if asciiLower(b[i]) != asciiLower(s[i]) {
			return false
		}
	}
	return true
}

func setCobolNodeStart(n *Node, source []byte, start uint32) {
	if n == nil || n.startByte == start || int(start) > len(source) {
		return
	}
	n.startByte = start
	n.startPoint = advancePointByBytes(Point{}, source[:start])
}

func setCobolNodeEnd(n *Node, source []byte, end uint32) {
	if n == nil || n.endByte == end || int(end) > len(source) {
		return
	}
	n.endByte = end
	n.endPoint = advancePointByBytes(Point{}, source[:end])
}

func normalizeCobolDivisionSiblingEnds(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || root.Type(lang) != "start" || resultChildCount(root) == 0 {
		return
	}
	def := cobolFirstProgramDefinition(root, lang)
	if def == nil {
		return
	}
	childCount := resultChildCount(def)
	for i := 0; i+1 < childCount; i++ {
		cur := resultChildAt(def, i)
		next := resultChildAt(def, i+1)
		if cur == nil || next == nil || cur.IsExtra() || next.IsExtra() {
			continue
		}
		if !strings.HasSuffix(cur.Type(lang), "_division") {
			continue
		}
		end := cobolNodeEndBeforeSibling(next, source, lang)
		if end == 0 || end <= cur.startByte || end >= cur.endByte {
			continue
		}
		cur.endByte = end
		cur.endPoint = advancePointByBytes(Point{}, source[:end])
	}
}

func normalizeCobolSectionSiblingEnds(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || len(source) == 0 {
		return
	}
	walkResultTreePostorder(root, func(parent *Node) {
		childCount := resultChildCount(parent)
		for i := 0; i+1 < childCount; i++ {
			cur := resultChildAt(parent, i)
			next := resultChildAt(parent, i+1)
			if cur == nil || next == nil || cur.IsExtra() || cur.IsError() || cur.IsMissing() {
				continue
			}
			if !strings.HasSuffix(cur.Type(lang), "_section") {
				continue
			}
			end := cobolNodeEndBeforeSibling(next, source, lang)
			if end == 0 || end <= cur.startByte || end >= cur.endByte {
				continue
			}
			setCobolNodeEnd(cur, source, end)
		}
		if childCount == 0 {
			return
		}
		last := resultChildAt(parent, childCount-1)
		if last == nil || last.IsExtra() || last.IsError() || last.IsMissing() || !strings.HasSuffix(last.Type(lang), "_section") {
			return
		}
		if parent.endByte > last.startByte && parent.endByte < last.endByte {
			setCobolNodeEnd(last, source, parent.endByte)
		}
	})
}

func cobolNodeEndBeforeSibling(next *Node, source []byte, lang *Language) uint32 {
	if next == nil || int(next.startByte) > len(source) {
		return 0
	}
	if next != nil && next.Type(lang) == "comment" && int(next.startByte) <= len(source) {
		lineStart := cobolLineStart(source, int(next.startByte))
		if lineStart < int(next.startByte) && cobolLineLooksFixedFormatComment(source, lineStart) {
			return lastNonTriviaByteEnd(source[:lineStart])
		}
	}
	return lastNonTriviaByteEnd(source[:next.startByte])
}

func normalizeCobolTrailingTriviaSpans(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || len(source) == 0 {
		return
	}
	walkResultTreePostorder(root, func(n *Node) {
		if n == nil || n.Type(lang) == "start" || n.IsExtra() || n.IsMissing() || n.IsError() || n.HasError() {
			return
		}
		childCount := resultChildCount(n)
		if childCount == 0 {
			normalizeCobolTrailingTriviaLeafSpan(n, source, lang)
			return
		}
		if int(n.endByte) > len(source) {
			return
		}
		last := (*Node)(nil)
		for i := childCount - 1; i >= 0; i-- {
			child := resultChildAt(n, i)
			if child == nil || child.IsMissing() {
				continue
			}
			last = child
			break
		}
		if last == nil || last.endByte >= n.endByte || last.endByte < n.startByte {
			return
		}
		if !cobolBytesAreTrailingTrivia(source, last.endByte, n.endByte) {
			return
		}
		setCobolNodeEnd(n, source, last.endByte)
	})
}

func normalizeCobolTrailingTriviaLeafSpan(n *Node, source []byte, lang *Language) {
	if n == nil || int(n.endByte) > len(source) || !cobolTrailingTriviaLeafCanTrim(n.Type(lang)) {
		return
	}
	end := lastNonTriviaByteEnd(source[n.startByte:n.endByte])
	if end == 0 {
		return
	}
	absoluteEnd := n.startByte + end
	if absoluteEnd <= n.startByte || absoluteEnd >= n.endByte {
		return
	}
	setCobolNodeEnd(n, source, absoluteEnd)
}

func cobolTrailingTriviaLeafCanTrim(typ string) bool {
	return strings.HasSuffix(typ, "_statement") || strings.HasSuffix(typ, "_statement_loop")
}

func cobolBytesAreTrailingTrivia(source []byte, start, end uint32) bool {
	if start >= end || int(end) > len(source) {
		return false
	}
	lineStart := int(start)
	for lineStart > 0 && source[lineStart-1] != '\n' && source[lineStart-1] != '\r' {
		lineStart--
	}
	column := int(start) - lineStart
	for i := int(start); i < int(end); i++ {
		switch source[i] {
		case ' ', '\t', '\f':
			column++
		case '\n':
			lineStart = i + 1
			column = 0
		case '\r':
			lineStart = i + 1
			column = 0
		default:
			if cobolLineLooksFixedFormatComment(source, lineStart) {
				column++
				continue
			}
			if column < 6 && cobolLineLooksFixedFormat(source, lineStart) {
				column++
				continue
			}
			if column >= 72 && cobolLineLooksFixedFormat(source, lineStart) {
				column++
				continue
			}
			return false
		}
	}
	return true
}

func cobolLineLooksFixedFormat(source []byte, lineStart int) bool {
	indicator := lineStart + 6
	if lineStart < 0 || indicator >= len(source) {
		return false
	}
	switch source[indicator] {
	case ' ', '*', '-', '/':
		return true
	default:
		return false
	}
}

func cobolLineLooksFixedFormatComment(source []byte, lineStart int) bool {
	indicator := lineStart + 6
	return lineStart >= 0 && indicator < len(source) && cobolFixedFormatCommentIndicator(source[indicator])
}

func normalizeCobolRecoveredParagraphHeader(root *Node, source []byte, lang *Language) {
	if root == nil || !isCobolLanguage(lang) || !lang.GeneratedByGrammargen || root.Type(lang) != "start" || !root.hasError() {
		return
	}
	rootChildren := resultChildSliceForMutation(root)
	if len(rootChildren) < 2 {
		return
	}
	errNode := rootChildren[len(rootChildren)-1]
	if errNode == nil || errNode.symbol != errorSymbol || !errNode.hasError() || errNode.startByte >= errNode.endByte || int(errNode.endByte) > len(source) {
		return
	}
	dot := cobolTrailingParagraphErrorDot(errNode, lang)
	if dot == nil || dot.endByte != errNode.endByte || dot.startByte <= errNode.startByte || int(dot.endByte) > len(source) {
		return
	}
	if !cobolBytesAreParagraphLabel(source[errNode.startByte:dot.startByte]) {
		return
	}
	program := cobolFirstProgramDefinition(root, lang)
	if program == nil {
		return
	}
	procedure := cobolFirstChildOfType(program, lang, "procedure_division")
	if procedure == nil {
		return
	}
	paragraphSym, ok := symbolByName(lang, "paragraph_header")
	if !ok {
		return
	}

	rootStart, rootStartPoint := root.startByte, root.startPoint
	rootEnd, rootEndPoint := root.endByte, root.endPoint
	procStart, procStartPoint := procedure.startByte, procedure.startPoint

	cobolClearErrorFlags(dot)
	header := newParentNodeInArena(root.ownerArena, paragraphSym, symbolIsNamed(lang, paragraphSym), []*Node{dot}, nil, 0)
	header.startByte = errNode.startByte
	header.startPoint = errNode.startPoint
	header.endByte = errNode.endByte
	header.endPoint = errNode.endPoint
	header.setHasError(false)

	procChildren := append(resultChildSliceForMutation(procedure), header)
	replaceNodeChildrenUnfielded(procedure, cloneNodeSliceInArena(procedure.ownerArena, procChildren))
	procedure.startByte = procStart
	procedure.startPoint = procStartPoint
	procedure.endByte = rootEnd
	procedure.endPoint = rootEndPoint
	cobolRefreshHasErrorFromChildren(procedure)

	keptRootChildren := cloneNodeSliceInArena(root.ownerArena, rootChildren[:len(rootChildren)-1])
	replaceNodeChildrenUnfielded(root, keptRootChildren)
	root.startByte = rootStart
	root.startPoint = rootStartPoint
	root.endByte = rootEnd
	root.endPoint = rootEndPoint
	cobolRefreshHasErrorFromChildren(root)
}

func cobolFirstProgramDefinition(root *Node, lang *Language) *Node {
	for i := 0; i < resultChildCount(root); i++ {
		child := resultChildAt(root, i)
		if child != nil && !child.IsExtra() && child.Type(lang) == "program_definition" {
			return child
		}
	}
	return nil
}

func cobolFirstChildOfType(parent *Node, lang *Language, typ string) *Node {
	for i := 0; i < resultChildCount(parent); i++ {
		child := resultChildAt(parent, i)
		if child != nil && !child.IsExtra() && child.Type(lang) == typ {
			return child
		}
	}
	return nil
}

func cobolTrailingParagraphErrorDot(errNode *Node, lang *Language) *Node {
	for i := resultChildCount(errNode) - 1; i >= 0; i-- {
		child := resultChildAt(errNode, i)
		if child != nil && child.Type(lang) == "." {
			return child
		}
	}
	return nil
}

func cobolBytesAreParagraphLabel(b []byte) bool {
	seen := false
	for _, c := range b {
		switch {
		case c == ' ' || c == '\t':
			if seen {
				return false
			}
		case c >= 'a' && c <= 'z':
			seen = true
		case c >= 'A' && c <= 'Z':
			seen = true
		case c >= '0' && c <= '9':
			seen = true
		case c == '-' || c == '_':
			seen = true
		default:
			return false
		}
	}
	return seen
}

func cobolClearErrorFlags(n *Node) {
	if n == nil {
		return
	}
	n.setHasError(false)
	for i := 0; i < resultChildCount(n); i++ {
		cobolClearErrorFlags(resultChildAt(n, i))
	}
}

func cobolRefreshHasErrorFromChildren(n *Node) {
	if n == nil {
		return
	}
	n.setHasError(false)
	for i := 0; i < resultChildCount(n); i++ {
		child := resultChildAt(n, i)
		if child != nil && (child.IsError() || child.HasError()) {
			n.setHasError(true)
			return
		}
	}
}
