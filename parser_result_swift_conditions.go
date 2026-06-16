package gotreesitter

// Swift control-flow conditions: trailing-closure ambiguity recovery (issue #118).
//
// The Swift grammar misparses a binary/comparison operator in an if/while
// condition, e.g. `if x > 0 { foo() }`. The body `{ foo() }` is greedily
// consumed as a *trailing closure* of the last operand (`0 { foo() }` →
// call_expression(0, lambda_literal{...})), so the control-flow statement loses
// its body and the surrounding function/type collapses into ERROR nodes — no
// function_declaration survives and symbol extraction yields nothing.
//
// Swift's real grammar forbids trailing closures in a control-flow condition;
// wrapping the condition in parentheses removes the ambiguity (`if (x > 0) {…}`
// parses cleanly). This pass detects each affected condition, reparses the
// source with synthetic parentheses around those conditions, then maps the
// recovered tree back to the original byte coordinates — dropping the synthetic
// parens and unwrapping the synthetic parenthesised expression so the result is
// byte-faithful to the original source.

type swiftParenInsert struct {
	pos uint32 // original-source byte offset to insert before
	ch  byte   // '(' or ')'
}

func normalizeSwiftRecoveredTrailingClosureConditions(root *Node, source []byte, p *Parser, lang *Language) {
	if root == nil || p == nil || p.skipRecoveryReparse || lang == nil || root.ownerArena == nil || len(source) == 0 {
		return
	}
	if root.Type(lang) != "source_file" || !root.HasError() {
		return
	}
	inserts := swiftCollectConditionParenInserts(root, source, lang)
	if len(inserts) == 0 {
		return
	}
	transformed, insTPos := swiftApplyParenInserts(source, inserts)
	tree, err := p.parseForRecovery(transformed)
	if err != nil || tree == nil || tree.RootNode() == nil {
		if tree != nil {
			tree.Release()
		}
		return
	}
	defer tree.Release()
	tRoot := tree.RootNode()
	// Apply only when removing the trailing-closure ambiguity makes the reparse
	// fully clean. This keeps the rewrite trivially safe — a file with other,
	// unrelated parse errors is left untouched rather than partially rewritten.
	if tRoot.HasError() || tRoot.Type(lang) != "source_file" {
		return
	}
	rm := &swiftRemap{
		tsrc:   transformed,
		insSet: make(map[uint32]bool, len(insTPos)),
		insPos: insTPos,
		arena:  root.ownerArena,
	}
	for _, t := range insTPos {
		rm.insSet[t] = true
	}
	newRoot, ok := rm.remap(tRoot)
	if !ok || newRoot == nil {
		return
	}
	root.symbol = newRoot.symbol
	root.productionID = newRoot.productionID
	root.fieldIDs = newRoot.fieldIDs
	root.children = cloneNodeSliceIfArena(root.ownerArena, newRoot.children)
	root.startByte = 0
	root.endByte = uint32(len(source))
	populateParentNode(root, root.children)
	recomputeNodePointsFromBytes(root, source)
	root.endByte = uint32(len(source))
	root.endPoint = advancePointByBytes(Point{}, source)
	if !swiftAnyChildHasError(root) {
		root.setHasError(false)
	}
}

// swiftCollectConditionParenInserts walks the (broken) tree with explicit parent
// tracking and, for every `if`/`while` keyword token that failed to form its
// statement, computes a `(`/`)` insertion pair around its condition. The body
// brace is located by scanning the source forward to the first top-level `{`,
// skipping comments, strings and bracketed/parenthesised groups.
func swiftCollectConditionParenInserts(root *Node, source []byte, lang *Language) []swiftParenInsert {
	var inserts []swiftParenInsert
	seen := make(map[uint32]bool)
	var walk func(n, parent *Node)
	walk = func(n, parent *Node) {
		if n == nil {
			return
		}
		typ := n.Type(lang)
		if (typ == "if" || typ == "while") && resultChildCount(n) == 0 {
			parentType := ""
			if parent != nil {
				parentType = parent.Type(lang)
			}
			if parentType != "if_statement" && parentType != "while_statement" {
				if lp, rp, ok := swiftConditionParenPositions(source, n.endByte); ok && !seen[lp] {
					seen[lp] = true
					inserts = append(inserts, swiftParenInsert{pos: lp, ch: '('})
					inserts = append(inserts, swiftParenInsert{pos: rp, ch: ')'})
				}
			}
		}
		for i := 0; i < resultChildCount(n); i++ {
			walk(resultChildAt(n, i), n)
		}
	}
	walk(root, nil)
	return inserts
}

// swiftConditionParenPositions returns the byte offsets at which to insert the
// opening and closing parenthesis for a condition that begins right after a
// control-flow keyword ending at keywordEnd. It returns ok=false when no plausible
// top-level body brace is found.
func swiftConditionParenPositions(source []byte, keywordEnd uint32) (lParen, rParen uint32, ok bool) {
	condStart := swiftSkipHorizontalAndNewlineSpace(source, keywordEnd)
	bodyOpen, found := swiftFindConditionBodyBrace(source, condStart)
	if !found {
		return 0, 0, false
	}
	rp := bodyOpen
	for rp > condStart {
		switch source[rp-1] {
		case ' ', '\t', '\n', '\r':
			rp--
			continue
		}
		break
	}
	if condStart >= rp {
		return 0, 0, false
	}
	// Already fully parenthesised (`if (cond) {`): no need to inject.
	if source[condStart] == '(' && source[rp-1] == ')' {
		return 0, 0, false
	}
	return condStart, rp, true
}

func swiftSkipHorizontalAndNewlineSpace(source []byte, start uint32) uint32 {
	i := start
	for i < uint32(len(source)) {
		switch source[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

// swiftFindConditionBodyBrace scans forward from start to the first `{` at
// bracket depth zero, skipping line/block comments and string literals (single
// and multiline) and counting (), [] nesting. Returns false if a `}`/`;` is
// reached at depth zero first (no body brace).
func swiftFindConditionBodyBrace(source []byte, start uint32) (uint32, bool) {
	depth := 0
	i := start
	n := uint32(len(source))
	for i < n {
		b := source[i]
		// Comments.
		if b == '/' && i+1 < n {
			if source[i+1] == '/' {
				i += 2
				for i < n && source[i] != '\n' {
					i++
				}
				continue
			}
			if source[i+1] == '*' {
				i += 2
				for i+1 < n && !(source[i] == '*' && source[i+1] == '/') {
					i++
				}
				i += 2
				continue
			}
		}
		// Strings.
		if b == '"' {
			if i+2 < n && source[i+1] == '"' && source[i+2] == '"' {
				i += 3
				for i+2 < n && !(source[i] == '"' && source[i+1] == '"' && source[i+2] == '"') {
					if source[i] == '\\' {
						i++
					}
					i++
				}
				i += 3
				continue
			}
			i++
			for i < n && source[i] != '"' {
				if source[i] == '\\' {
					i++
				}
				i++
			}
			i++
			continue
		}
		switch b {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case '{':
			if depth == 0 {
				return i, true
			}
		case '}', ';':
			if depth == 0 {
				return 0, false
			}
		}
		i++
	}
	return 0, false
}

func swiftApplyParenInserts(source []byte, inserts []swiftParenInsert) ([]byte, []uint32) {
	// Insertion sort by position (stable); the list is tiny.
	sorted := make([]swiftParenInsert, len(inserts))
	copy(sorted, inserts)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1].pos > sorted[j].pos; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	out := make([]byte, 0, len(source)+len(sorted))
	insTPos := make([]uint32, 0, len(sorted))
	srcIdx := uint32(0)
	for _, ins := range sorted {
		if ins.pos > srcIdx {
			out = append(out, source[srcIdx:ins.pos]...)
			srcIdx = ins.pos
		} else if ins.pos < srcIdx {
			// Overlapping/unsorted guard — should not happen.
			continue
		}
		insTPos = append(insTPos, uint32(len(out)))
		out = append(out, ins.ch)
	}
	out = append(out, source[srcIdx:]...)
	return out, insTPos
}

type swiftRemap struct {
	tsrc   []byte
	insSet map[uint32]bool
	insPos []uint32
	arena  *nodeArena
}

func (r *swiftRemap) isSyntheticParen(n *Node) bool {
	if n == nil || resultChildCount(n) != 0 {
		return false
	}
	if n.endByte != n.startByte+1 || !r.insSet[n.startByte] {
		return false
	}
	c := r.tsrc[n.startByte]
	return c == '(' || c == ')'
}

func (r *swiftRemap) mapByte(t uint32) uint32 {
	c := uint32(0)
	for _, p := range r.insPos {
		if p < t {
			c++
		}
	}
	return t - c
}

// remap clones a node from transformed coordinates into the arena in original
// coordinates, dropping synthetic parens and unwrapping the synthetic
// parenthesised expression. Returns ok=false if a synthetic paren is found
// outside an unwrappable wrapper (so the caller can abort safely).
func (r *swiftRemap) remap(n *Node) (*Node, bool) {
	if n == nil {
		return nil, true
	}
	// Unwrap a wrapper whose first and last children are our synthetic parens
	// (e.g. the tuple_expression the grammar builds around `(cond)`).
	if cc := resultChildCount(n); cc >= 2 {
		first := resultChildAt(n, 0)
		last := resultChildAt(n, cc-1)
		if r.isSyntheticParen(first) && r.isSyntheticParen(last) {
			var inner *Node
			for i := 0; i < cc; i++ {
				c := resultChildAt(n, i)
				if r.isSyntheticParen(c) {
					continue
				}
				if inner != nil {
					return nil, false
				}
				inner = c
			}
			if inner == nil {
				return nil, false
			}
			return r.remap(inner)
		}
	}
	if r.isSyntheticParen(n) {
		return nil, false
	}
	newStart := r.mapByte(n.startByte)
	newEnd := r.mapByte(n.endByte)
	if resultChildCount(n) == 0 {
		leaf := newLeafNodeInArena(r.arena, n.symbol, n.isNamed(), newStart, newEnd, Point{}, Point{})
		leaf.setExtra(n.IsExtra())
		return leaf, true
	}
	kids := make([]*Node, 0, resultChildCount(n))
	for i := 0; i < resultChildCount(n); i++ {
		rc, ok := r.remap(resultChildAt(n, i))
		if !ok {
			return nil, false
		}
		if rc != nil {
			kids = append(kids, rc)
		}
	}
	kids = cloneNodeSliceInArena(r.arena, kids)
	fieldIDs := cloneFieldIDSliceInArena(r.arena, n.fieldIDs)
	node := newParentNodeInArena(r.arena, n.symbol, n.isNamed(), kids, fieldIDs, n.productionID)
	node.setExtra(n.IsExtra())
	node.startByte = newStart
	node.endByte = newEnd
	return node, true
}
