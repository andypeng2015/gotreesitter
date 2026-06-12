package gotreesitter

import "bytes"

func normalizeTemplCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "templ" || len(source) == 0 || root.HasError() {
		return
	}
	normalizeTemplComponentImportArguments(root, source, lang)
}

func normalizeTemplComponentImportArguments(root *Node, source []byte, lang *Language) {
	argListSym, argListNamed, hasArgList := symbolMeta(lang, "argument_list")
	walkResultTree(root, func(n *Node) {
		if n == nil || n.HasError() {
			return
		}
		children := resultChildSliceForMutation(n)
		if len(children) < 2 {
			return
		}
		changed := false
		out := make([]*Node, 0, len(children))
		for i := 0; i < len(children); i++ {
			child := children[i]
			if i+1 < len(children) && templCanMergeComponentImportArgs(child, children[i+1], source, lang) {
				args := children[i+1]
				if !hasArgList {
					out = append(out, child)
					continue
				}
				argList := templBuildSimpleArgumentList(args, source, lang, argListSym, argListNamed)
				if argList == nil {
					out = append(out, child)
					continue
				}
				replaceNodeChildrenUnfielded(child, cloneNodeSliceInArena(child.ownerArena, append(resultChildSliceForMutation(child), argList)))
				child.endByte = args.endByte
				child.endPoint = args.endPoint
				out = append(out, child)
				i++
				changed = true
				continue
			}
			out = append(out, child)
		}
		if changed {
			replaceNodeChildrenUnfielded(n, cloneNodeSliceInArena(n.ownerArena, out))
		}
	})
}

func templCanMergeComponentImportArgs(importNode, argsNode *Node, source []byte, lang *Language) bool {
	if importNode == nil || argsNode == nil {
		return false
	}
	if importNode.Type(lang) != "component_import" || argsNode.Type(lang) != "element_text" {
		return false
	}
	if importNode.endByte != argsNode.startByte || argsNode.startByte >= argsNode.endByte {
		return false
	}
	if int(argsNode.endByte) > len(source) {
		return false
	}
	args := source[argsNode.startByte:argsNode.endByte]
	return len(args) >= 2 && args[0] == '(' && args[len(args)-1] == ')'
}

func templBuildSimpleArgumentList(argsNode *Node, source []byte, lang *Language, argListSym Symbol, argListNamed bool) *Node {
	if argsNode == nil || int(argsNode.endByte) > len(source) {
		return nil
	}
	openSym, openNamed, ok := symbolMeta(lang, "(")
	if !ok {
		return nil
	}
	closeSym, closeNamed, ok := symbolMeta(lang, ")")
	if !ok {
		return nil
	}
	commaSym, commaNamed, ok := symbolMeta(lang, ",")
	if !ok {
		return nil
	}
	identSym, identNamed, ok := symbolMeta(lang, "identifier")
	if !ok {
		return nil
	}
	stringSym, stringNamed, hasString := symbolMeta(lang, "interpreted_string_literal")
	stringContentSym, stringContentNamed, hasStringContent := symbolMeta(lang, "interpreted_string_literal_content")
	quoteSym, quoteNamed, hasQuote := symbolMeta(lang, "\"")
	raw := source[argsNode.startByte:argsNode.endByte]
	if bytes.IndexByte(raw, '\n') >= 0 || bytes.IndexByte(raw, '\r') >= 0 {
		return nil
	}
	if len(raw) < 2 || raw[0] != '(' || raw[len(raw)-1] != ')' {
		return nil
	}

	arena := argsNode.ownerArena
	children := make([]*Node, 0, 5)
	start := argsNode.startByte
	children = append(children, newLeafNodeInArena(arena, openSym, openNamed, start, start+1, argsNode.startPoint, Point{Row: argsNode.startPoint.Row, Column: argsNode.startPoint.Column + 1}))
	i := 1
	for i < len(raw)-1 {
		for i < len(raw)-1 && (raw[i] == ' ' || raw[i] == '\t') {
			i++
		}
		if i >= len(raw)-1 {
			break
		}
		if raw[i] == ',' {
			pos := start + uint32(i)
			children = append(children, newLeafNodeInArena(arena, commaSym, commaNamed, pos, pos+1, Point{Row: argsNode.startPoint.Row, Column: argsNode.startPoint.Column + uint32(i)}, Point{Row: argsNode.startPoint.Row, Column: argsNode.startPoint.Column + uint32(i) + 1}))
			i++
			continue
		}
		if raw[i] == '"' {
			if !hasString || !hasStringContent || !hasQuote {
				return nil
			}
			str, next, ok := templBuildSimpleStringLiteral(argsNode, source, lang, i, stringSym, stringNamed, stringContentSym, stringContentNamed, quoteSym, quoteNamed)
			if !ok {
				return nil
			}
			children = append(children, str)
			i = next
			continue
		}
		if !templIsIdentifierStart(raw[i]) {
			return nil
		}
		identStart := i
		i++
		for i < len(raw)-1 && templIsIdentifierContinue(raw[i]) {
			i++
		}
		posStart := start + uint32(identStart)
		posEnd := start + uint32(i)
		children = append(children, newLeafNodeInArena(arena, identSym, identNamed, posStart, posEnd, Point{Row: argsNode.startPoint.Row, Column: argsNode.startPoint.Column + uint32(identStart)}, Point{Row: argsNode.startPoint.Row, Column: argsNode.startPoint.Column + uint32(i)}))
	}
	closeStart := argsNode.endByte - 1
	children = append(children, newLeafNodeInArena(arena, closeSym, closeNamed, closeStart, argsNode.endByte, Point{Row: argsNode.endPoint.Row, Column: argsNode.endPoint.Column - 1}, argsNode.endPoint))
	return newParentNodeInArena(arena, argListSym, argListNamed, cloneNodeSliceInArena(arena, children), nil, 0)
}

func templBuildSimpleStringLiteral(argsNode *Node, source []byte, lang *Language, relStart int, stringSym Symbol, stringNamed bool, contentSym Symbol, contentNamed bool, quoteSym Symbol, quoteNamed bool) (*Node, int, bool) {
	raw := source[argsNode.startByte:argsNode.endByte]
	if relStart >= len(raw)-1 || raw[relStart] != '"' {
		return nil, relStart, false
	}
	relEnd := relStart + 1
	for relEnd < len(raw)-1 && raw[relEnd] != '"' {
		if raw[relEnd] == '\\' {
			return nil, relStart, false
		}
		relEnd++
	}
	if relEnd >= len(raw)-1 || raw[relEnd] != '"' {
		return nil, relStart, false
	}
	arena := argsNode.ownerArena
	row := argsNode.startPoint.Row
	baseCol := argsNode.startPoint.Column
	start := argsNode.startByte + uint32(relStart)
	end := argsNode.startByte + uint32(relEnd) + 1
	open := newLeafNodeInArena(arena, quoteSym, quoteNamed, start, start+1, Point{Row: row, Column: baseCol + uint32(relStart)}, Point{Row: row, Column: baseCol + uint32(relStart) + 1})
	content := newLeafNodeInArena(arena, contentSym, contentNamed, start+1, end-1, Point{Row: row, Column: baseCol + uint32(relStart) + 1}, Point{Row: row, Column: baseCol + uint32(relEnd)})
	close := newLeafNodeInArena(arena, quoteSym, quoteNamed, end-1, end, Point{Row: row, Column: baseCol + uint32(relEnd)}, Point{Row: row, Column: baseCol + uint32(relEnd) + 1})
	str := newParentNodeInArena(arena, stringSym, stringNamed, []*Node{open, content, close}, nil, 0)
	return str, relEnd + 1, true
}

func templIsIdentifierStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}

func templIsIdentifierContinue(b byte) bool {
	return templIsIdentifierStart(b) || (b >= '0' && b <= '9')
}
