package gotreesitter

import "bytes"

func normalizeIniCompatibility(root *Node, source []byte, lang *Language) resultCompatibilityResult {
	result := normalizeIniMypyContinuationRecovery(root, source, lang)
	normalizeIniSectionStarts(root, lang)
	normalizeIniDocumentBlanks(root, lang)
	return result
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

type iniSourceLine struct {
	start        uint32
	end          uint32
	contentStart uint32
	contentEnd   uint32
}

func normalizeIniMypyContinuationRecovery(root *Node, source []byte, lang *Language) resultCompatibilityResult {
	if root == nil || lang == nil || lang.Name != "ini" || len(source) == 0 {
		return resultCompatibilityResult{}
	}
	if !bytes.Contains(source, []byte("enable_error_code")) {
		return resultCompatibilityResult{}
	}
	if normalizeIniMypyFlatErrorRoot(root, source, lang) {
		return resultCompatibilityResult{}
	}
	if result, ok := normalizeIniMypyFlatDocumentContinuation(root, source, lang); ok {
		return result
	}
	if result, ok := normalizeIniMypyContinuationErrorDocument(root, source, lang); ok {
		return result
	}
	return resultCompatibilityResult{}
}

func normalizeIniMypyFlatErrorRoot(root *Node, source []byte, lang *Language) bool {
	if root.Type(lang) != "document" || resultChildCount(root) != 1 || !bytes.Contains(source, []byte("files =\n    ")) {
		return false
	}
	section := resultChildAt(root, 0)
	if section == nil || section.Type(lang) != "section" {
		return false
	}
	settingNameSym, eqSym, ok := iniRecoverySymbols(lang)
	if !ok {
		return false
	}
	reusable := iniReusableChildrenByStart(section)
	children := make([]*Node, 0, resultChildCount(section)+16)
	inFilesContinuation := false
	sawFirstContinuation := false
	sawMalformedFiles := false
	afterMalformedFiles := false
	for _, line := range iniSourceLines(source) {
		if line.contentStart >= line.contentEnd {
			continue
		}
		text := source[line.contentStart:line.contentEnd]
		if bytes.HasPrefix(text, []byte("#")) || bytes.Equal(text, []byte("[mypy]")) {
			if reused := reusable[line.contentStart]; reused != nil {
				children = append(children, reused)
			}
			continue
		}
		if bytes.Equal(text, []byte("files =")) {
			if reused := reusable[line.contentStart]; reused != nil {
				children = append(children, reused)
			}
			inFilesContinuation = true
			continue
		}
		if inFilesContinuation && line.contentStart > line.start {
			sawMalformedFiles = true
			if !sawFirstContinuation {
				children = append(children, iniNewLeaf(root.ownerArena, source, settingNameSym, true, line.contentStart, line.contentEnd))
				sawFirstContinuation = true
			} else {
				children = append(children, iniNewErrorLeaf(root.ownerArena, source, line.contentStart, line.contentEnd))
			}
			continue
		}
		if inFilesContinuation {
			inFilesContinuation = false
			afterMalformedFiles = sawMalformedFiles
		}
		if !afterMalformedFiles {
			return false
		}
		iniAppendErrorLineParts(&children, root.ownerArena, source, line, eqSym)
	}
	if !sawMalformedFiles || len(children) == 0 {
		return false
	}
	retagResultRoot(root, errorSymbol, true)
	root.setHasError(true)
	replaceNodeChildrenUnfielded(root, cloneNodeSliceIfArena(root.ownerArena, children))
	extendNodeToTrailingWhitespace(root, source)
	return true
}

func normalizeIniMypyContinuationErrorDocument(root *Node, source []byte, lang *Language) (resultCompatibilityResult, bool) {
	if root.Type(lang) != "ERROR" || resultChildCount(root) < 2 || !bytes.Contains(source, []byte("enable_error_code = \n    ")) {
		return resultCompatibilityResult{}, false
	}
	documentSym, ok := symbolByName(lang, "document")
	if !ok {
		return resultCompatibilityResult{}, false
	}
	settingNameSym, _, ok := iniRecoverySymbols(lang)
	if !ok {
		return resultCompatibilityResult{}, false
	}
	section := resultChildAt(root, 0)
	firstErr := resultChildAt(root, 1)
	if section == nil || firstErr == nil || section.Type(lang) != "section" || firstErr.Type(lang) != "ERROR" || resultChildCount(firstErr) == 0 {
		return resultCompatibilityResult{}, false
	}
	firstName := resultChildAt(firstErr, 0)
	if firstName == nil || firstName.symbol != settingNameSym {
		return resultCompatibilityResult{}, false
	}
	continuationLines := iniContinuationLinesFrom(source, firstErr.startByte)
	if len(continuationLines) <= 1 {
		return resultCompatibilityResult{}, false
	}
	errLines := continuationLines[1:]
	rebuiltSection := iniRebuildSectionWithLineBreakSpans(section, source, firstErr.startByte)
	errNode := iniBuildContinuationErrorNode(root.ownerArena, source, firstErr, firstName, errLines)
	if rebuiltSection == nil || errNode == nil {
		return resultCompatibilityResult{}, false
	}
	retagResultRoot(root, documentSym, true)
	root.setHasError(true)
	replaceNodeChildrenUnfielded(root, cloneNodeSliceIfArena(root.ownerArena, []*Node{rebuiltSection, errNode}))
	root.childIndex = -1
	extendNodeToTrailingWhitespace(root, source)
	return resultCompatibilityResult{
		iniMypyEnableErrorContinuation: true,
		iniContinuationStart:           firstErr.startByte,
		iniContinuationEnd:             errNode.endByte,
	}, true
}

func normalizeIniMypyFlatDocumentContinuation(root *Node, source []byte, lang *Language) (resultCompatibilityResult, bool) {
	if root.Type(lang) != "document" || resultChildCount(root) < 3 || !bytes.Contains(source, []byte("enable_error_code = \n    ")) {
		return resultCompatibilityResult{}, false
	}
	sectionSym, ok := symbolByName(lang, "section")
	if !ok {
		return resultCompatibilityResult{}, false
	}
	settingNameSym, _, ok := iniRecoverySymbols(lang)
	if !ok {
		return resultCompatibilityResult{}, false
	}
	firstContinuation := iniFirstContinuationContentStart(source, []byte("enable_error_code = \n"))
	if firstContinuation == 0 {
		return resultCompatibilityResult{}, false
	}
	firstIdx := -1
	for i := 0; i < resultChildCount(root); i++ {
		child := resultChildAt(root, i)
		if child != nil && child.startByte == firstContinuation && child.symbol == settingNameSym {
			firstIdx = i
			break
		}
	}
	if firstIdx <= 0 {
		return resultCompatibilityResult{}, false
	}
	section := iniBuildSectionFromTopLevelChildren(root.ownerArena, lang, sectionSym, source, resultChildSliceRangeForMutation(root, 0, firstIdx), firstContinuation)
	if section == nil {
		return resultCompatibilityResult{}, false
	}
	firstName := resultChildAt(root, firstIdx)
	errLines := iniContinuationLinesFrom(source, firstContinuation)
	if len(errLines) == 0 {
		return resultCompatibilityResult{}, false
	}
	errNode := iniBuildContinuationErrorNode(root.ownerArena, source, firstName, firstName, errLines[1:])
	if errNode == nil {
		return resultCompatibilityResult{}, false
	}
	replaceNodeChildrenUnfielded(root, cloneNodeSliceIfArena(root.ownerArena, []*Node{section, errNode}))
	root.setHasError(true)
	root.childIndex = -1
	extendNodeToTrailingWhitespace(root, source)
	return resultCompatibilityResult{
		iniMypyEnableErrorContinuation: true,
		iniContinuationStart:           firstContinuation,
		iniContinuationEnd:             errNode.endByte,
	}, true
}

func iniDeferredCompatibilityAccepted(root *Node, source []byte, lang *Language, result resultCompatibilityResult) bool {
	if !result.iniMypyEnableErrorContinuation {
		return false
	}
	if root == nil || lang == nil || lang.Name != "ini" || len(source) == 0 {
		return false
	}
	if root.endByte != uint32(len(source)) || !root.HasError() || root.Type(lang) != "document" || resultChildCount(root) != 2 {
		return false
	}
	if !bytes.Contains(source, []byte("enable_error_code = \n    ")) {
		return false
	}
	firstContinuation := iniFirstContinuationContentStart(source, []byte("enable_error_code = \n"))
	if firstContinuation == 0 || firstContinuation != result.iniContinuationStart {
		return false
	}
	continuationLines := iniContinuationLinesFrom(source, firstContinuation)
	if len(continuationLines) == 0 || continuationLines[len(continuationLines)-1].contentEnd != result.iniContinuationEnd {
		return false
	}
	section := resultChildAt(root, 0)
	errNode := resultChildAt(root, 1)
	if section == nil || section.Type(lang) != "section" || errNode == nil || errNode.Type(lang) != "ERROR" {
		return false
	}
	if markerEnd := iniMypyEnableErrorMarkerEnd(source); markerEnd == 0 || section.endByte != markerEnd {
		return false
	}
	if errNode.startByte != result.iniContinuationStart || errNode.endByte != result.iniContinuationEnd || resultChildCount(errNode) != len(continuationLines) {
		return false
	}
	for i, line := range continuationLines {
		child := resultChildAt(errNode, i)
		if child == nil || child.startByte != line.contentStart || child.endByte != line.contentEnd {
			return false
		}
		if i == 0 {
			if child.Type(lang) != "setting_name" {
				return false
			}
			continue
		}
		if child.Type(lang) != "ERROR" {
			return false
		}
	}
	return true
}

func iniRecoverySymbols(lang *Language) (Symbol, Symbol, bool) {
	syms, ok := languageSymbols(lang, "setting_name", "=")
	if !ok {
		return 0, 0, false
	}
	return syms[0], syms[1], true
}

func iniReusableChildrenByStart(section *Node) map[uint32]*Node {
	reusable := make(map[uint32]*Node)
	for i := 0; i < resultChildCount(section); i++ {
		child := resultChildAt(section, i)
		if child != nil {
			reusable[child.startByte] = child
		}
	}
	return reusable
}

func iniContinuationLinesFrom(source []byte, firstStart uint32) []iniSourceLine {
	var lines []iniSourceLine
	inBlock := false
	for _, line := range iniSourceLines(source) {
		if !inBlock {
			if line.contentStart != firstStart {
				continue
			}
			if !iniIsContinuationLine(source, line) {
				return nil
			}
			inBlock = true
			lines = append(lines, line)
			continue
		}
		if !iniIsContinuationLine(source, line) {
			break
		}
		lines = append(lines, line)
	}
	return lines
}

func iniIsContinuationLine(source []byte, line iniSourceLine) bool {
	if line.contentStart >= line.contentEnd || line.contentStart <= line.start {
		return false
	}
	text := source[line.contentStart:line.contentEnd]
	return !bytes.HasPrefix(text, []byte("#")) && !bytes.HasPrefix(text, []byte("["))
}

func iniFirstContinuationContentStart(source []byte, marker []byte) uint32 {
	idx := bytes.Index(source, marker)
	if idx < 0 {
		return 0
	}
	pos := idx + len(marker)
	for pos < len(source) {
		switch source[pos] {
		case ' ', '\t':
			pos++
			continue
		case '\n', '\r':
			return 0
		default:
			return uint32(pos)
		}
	}
	return 0
}

func iniMypyEnableErrorMarkerEnd(source []byte) uint32 {
	marker := []byte("enable_error_code = \n")
	idx := bytes.Index(source, marker)
	if idx < 0 {
		return 0
	}
	return uint32(idx + len(marker))
}

func iniSourceLines(source []byte) []iniSourceLine {
	lines := make([]iniSourceLine, 0, bytes.Count(source, []byte{'\n'})+1)
	for start := 0; start < len(source); {
		end := start
		for end < len(source) && source[end] != '\n' {
			end++
		}
		lineEnd := end
		if end < len(source) {
			end++
		}
		contentStart := start
		for contentStart < lineEnd && (source[contentStart] == ' ' || source[contentStart] == '\t') {
			contentStart++
		}
		contentEnd := lineEnd
		for contentEnd > contentStart && (source[contentEnd-1] == ' ' || source[contentEnd-1] == '\t' || source[contentEnd-1] == '\r') {
			contentEnd--
		}
		lines = append(lines, iniSourceLine{
			start:        uint32(start),
			end:          uint32(end),
			contentStart: uint32(contentStart),
			contentEnd:   uint32(contentEnd),
		})
		start = end
	}
	return lines
}

func iniRebuildSectionWithLineBreakSpans(section *Node, source []byte, nextStart uint32) *Node {
	if section == nil {
		return nil
	}
	arena := section.ownerArena
	children := make([]*Node, 0, resultChildCount(section))
	for i := 0; i < resultChildCount(section); i++ {
		child := resultChildAt(section, i)
		if child == nil {
			continue
		}
		cloned := cloneNodeInArena(arena, child)
		if cloned == nil {
			continue
		}
		iniExtendNodeThroughFollowingLineBreak(cloned, source)
		children = append(children, cloned)
	}
	if len(children) == 0 {
		return nil
	}
	rebuilt := newParentNodeInArena(arena, section.symbol, section.isNamed(), cloneNodeSliceIfArena(arena, children), nil, 0)
	rebuilt.startByte = section.startByte
	rebuilt.startPoint = section.startPoint
	end := children[len(children)-1].endByte
	if end < nextStart {
		for nextStart > end && (source[nextStart-1] == ' ' || source[nextStart-1] == '\t') {
			nextStart--
		}
		if nextStart > end {
			end = nextStart
		}
	}
	rebuilt.endByte = end
	rebuilt.endPoint = advancePointByBytes(Point{}, source[:end])
	rebuilt.setHasError(section.hasError())
	return rebuilt
}

func iniBuildSectionFromTopLevelChildren(arena *nodeArena, lang *Language, sectionSym Symbol, source []byte, original []*Node, nextStart uint32) *Node {
	if len(original) == 0 {
		return nil
	}
	children := make([]*Node, 0, len(original))
	for i, child := range original {
		if child == nil {
			continue
		}
		if lang != nil && child.Type(lang) == "_blank" {
			continue
		}
		cloned := cloneNodeInArena(arena, child)
		if cloned == nil {
			continue
		}
		if i+1 < len(original) && original[i+1] != nil && lang != nil && original[i+1].Type(lang) == "_blank" {
			setNodeEndTo(cloned, original[i+1].startByte, source)
		} else {
			iniExtendNodeThroughFollowingLineBreak(cloned, source)
		}
		children = append(children, cloned)
	}
	if len(children) == 0 {
		return nil
	}
	section := newParentNodeInArena(arena, sectionSym, true, cloneNodeSliceIfArena(arena, children), nil, 0)
	end := children[len(children)-1].endByte
	if end < nextStart {
		for nextStart > end && (source[nextStart-1] == ' ' || source[nextStart-1] == '\t') {
			nextStart--
		}
		if nextStart > end {
			end = nextStart
		}
	}
	section.endByte = end
	section.endPoint = advancePointByBytes(Point{}, source[:end])
	return section
}

func iniBuildContinuationErrorNode(arena *nodeArena, source []byte, firstErr, firstName *Node, lines []iniSourceLine) *Node {
	children := make([]*Node, 0, len(lines)+1)
	children = append(children, cloneNodeInArena(arena, firstName))
	errEnd := firstErr.endByte
	for _, line := range lines {
		leaf := iniNewErrorLeaf(arena, source, line.contentStart, line.contentEnd)
		children = append(children, leaf)
		errEnd = line.contentEnd
	}
	errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, children), nil, 0)
	errNode.setExtra(true)
	errNode.setHasError(true)
	errNode.startByte = firstErr.startByte
	errNode.startPoint = firstErr.startPoint
	errNode.endByte = errEnd
	errNode.endPoint = advancePointByBytes(Point{}, source[:errEnd])
	return errNode
}

func iniAppendErrorLineParts(children *[]*Node, arena *nodeArena, source []byte, line iniSourceLine, eqSym Symbol) {
	eq := bytes.IndexByte(source[line.contentStart:line.contentEnd], '=')
	if eq < 0 {
		*children = append(*children, iniNewErrorLeaf(arena, source, line.contentStart, line.contentEnd))
		return
	}
	eqStart := line.contentStart + uint32(eq)
	keyEnd := eqStart
	for keyEnd > line.contentStart && (source[keyEnd-1] == ' ' || source[keyEnd-1] == '\t') {
		keyEnd--
	}
	valueStart := eqStart + 1
	for valueStart < line.contentEnd && (source[valueStart] == ' ' || source[valueStart] == '\t') {
		valueStart++
	}
	if keyEnd > line.contentStart {
		*children = append(*children, iniNewErrorLeaf(arena, source, line.contentStart, keyEnd))
	}
	*children = append(*children, iniNewLeaf(arena, source, eqSym, false, eqStart, eqStart+1))
	if valueStart < line.contentEnd {
		*children = append(*children, iniNewErrorLeaf(arena, source, valueStart, line.contentEnd))
	}
}

func iniExtendNodeThroughFollowingLineBreak(n *Node, source []byte) {
	if n == nil || n.endByte >= uint32(len(source)) {
		return
	}
	end := n.endByte
	switch source[end] {
	case '\n':
		end++
	case '\r':
		end++
		if end < uint32(len(source)) && source[end] == '\n' {
			end++
		}
	}
	if end > n.endByte {
		extendNodeEndTo(n, end, source)
	}
}

func iniNewErrorLeaf(arena *nodeArena, source []byte, start, end uint32) *Node {
	leaf := iniNewLeaf(arena, source, errorSymbol, true, start, end)
	leaf.setHasError(true)
	return leaf
}

func iniNewLeaf(arena *nodeArena, source []byte, sym Symbol, named bool, start, end uint32) *Node {
	return newLeafNodeInArena(arena, sym, named, start, end, advancePointByBytes(Point{}, source[:start]), advancePointByBytes(Point{}, source[:end]))
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
