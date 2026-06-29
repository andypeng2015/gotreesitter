package gotreesitter

import "testing"

func TestCVersionStatusAddsPausedSkippedTreeCost(t *testing.T) {
	parser := &Parser{}

	paused := glrStack{cPaused: true}
	stackCost := parser.cStackErrorCost(&paused)
	if stackCost != cErrCostPerRecovery {
		t.Fatalf("paused stack cost = %d, want %d", stackCost, cErrCostPerRecovery)
	}
	status := parser.cVersionStatus(&paused)
	wantStatusCost := stackCost + cErrCostPerSkippedTree
	if status.cost != wantStatusCost {
		t.Fatalf("paused version status cost = %d, want %d", status.cost, wantStatusCost)
	}
	if !status.isInError {
		t.Fatal("paused version status isInError = false, want true")
	}

	openError := glrStack{cRec: &cRecoverState{}}
	stackCost = parser.cStackErrorCost(&openError)
	if stackCost != cErrCostPerRecovery {
		t.Fatalf("open ERROR_STATE stack cost = %d, want %d", stackCost, cErrCostPerRecovery)
	}
	status = parser.cVersionStatus(&openError)
	if status.cost != stackCost {
		t.Fatalf("open ERROR_STATE version status cost = %d, want stack cost %d", status.cost, stackCost)
	}
	if !status.isInError {
		t.Fatal("open ERROR_STATE version status isInError = false, want true")
	}
}

func TestCRecoveryCostCompetitionDisabledInNoTreeModes(t *testing.T) {
	parser := &Parser{errorCostCompetition: true}
	if !parser.errorCostCompetitionEnabled() {
		t.Fatal("errorCostCompetitionEnabled = false, want true for ordinary tree mode")
	}

	parser.noTreeBenchmarkOnly = true
	if parser.errorCostCompetitionEnabled() {
		t.Fatal("errorCostCompetitionEnabled = true, want false for no-tree benchmark mode")
	}

	parser.noTreeBenchmarkOnly = false
	parser.noTreeCheckpointBenchmarkOnly = true
	if parser.errorCostCompetitionEnabled() {
		t.Fatal("errorCostCompetitionEnabled = true, want false for no-tree checkpoint mode")
	}
}

func TestCApplyMergedErrorGroupBaselineUsesMaxHeadNodeCount(t *testing.T) {
	parser := cRecoveryElectionTestParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	short := cRecoveryBaselineStack(arena, 2, 1)
	long := cRecoveryBaselineStack(arena, 2, 3)
	versions := []glrStack{short, long}
	for vi := range versions {
		versions[vi].pushEntry(stackEntry{state: cErrorState}, nil, nil)
	}

	baseline := parser.cApplyMergedErrorGroupBaseline(versions)
	if baseline != 3 {
		t.Fatalf("merged baseline = %d, want max cumulative node count 3", baseline)
	}
	for i := range versions {
		if got := versions[i].cNodeBaseline; got != baseline {
			t.Fatalf("version %d baseline = %d, want shared %d", i, got, baseline)
		}
	}
	if got := parser.cNodeCountSinceError(&versions[0]); got != 0 {
		t.Fatalf("short version node count since error = %d, want clamp to 0", got)
	}
	if got := versions[0].cNodeBaseline; got != 1 {
		t.Fatalf("short version clamped baseline = %d, want current cumulative count 1", got)
	}
	if got := parser.cNodeCountSinceError(&versions[1]); got != 0 {
		t.Fatalf("long version node count since error = %d, want 0", got)
	}
	if got := versions[1].cNodeBaseline; got != baseline {
		t.Fatalf("long version baseline after node count = %d, want shared %d", got, baseline)
	}
}

func TestCRecordSummaryUsesCompactEntryPosition(t *testing.T) {
	parser := &Parser{}
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	leaf := newCompactFullLeafInArena(arena, 1, true, 13, 21, Point{Row: 2, Column: 3}, Point{Row: 2, Column: 11})
	entry := newStackEntryCompactFullLeaf(3, leaf)
	summary := parser.cRecordSummary([]stackEntry{entry, {state: 2}})
	if len(summary) == 0 {
		t.Fatal("summary is empty")
	}
	if got := summary[0].posBytes; got != 21 {
		t.Fatalf("compact summary posBytes = %d, want 21", got)
	}
	if got := summary[0].posRow; got != 2 {
		t.Fatalf("compact summary posRow = %d, want 2", got)
	}
}

func TestCStackPosRowUsesCompactEntryPoint(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	leaf := newCompactFullLeafInArena(arena, 1, true, 13, 21, Point{Row: 4, Column: 3}, Point{Row: 4, Column: 11})
	entry := newStackEntryCompactFullLeaf(3, leaf)

	sliceStack := glrStack{entries: []stackEntry{{state: 1}, entry}}
	if got := cStackPosRow(&sliceStack); got != 4 {
		t.Fatalf("slice-backed compact stack row = %d, want 4", got)
	}

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	head := scratch.allocNode(entry, base, 2)
	gssStack := glrStack{gss: gssStack{head: head}}
	if got := cStackPosRow(&gssStack); got != 4 {
		t.Fatalf("GSS-backed compact stack row = %d, want 4", got)
	}
}

func TestCRecoverStrategy1ElectionRetriesDuplicateEntryForNextMember(t *testing.T) {
	parser := cRecoveryElectionTestParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	group := &cRecGroup{}
	entry := cStackSummaryEntry{depth: 1, state: 2, posBytes: 5}
	stacks := []glrStack{
		cRecoveryElectionStack(arena, 4, 3, 5, group, 0, []cStackSummaryEntry{entry}),
		cRecoveryElectionStack(arena, 2, 3, 10, group, 1, []cStackSummaryEntry{entry}),
	}
	nodeCount := 0
	didRecover, forked := parser.cRecoverStrategy1Election(&stacks, group, Token{Symbol: 1}, &nodeCount, arena, nil, nil, nil)
	if !didRecover || !forked {
		t.Fatalf("strategy 1 election = didRecover %v forked %v, want true/true", didRecover, forked)
	}
	if len(stacks) != 3 {
		t.Fatalf("stack count = %d, want recovered fork appended", len(stacks))
	}
	fork := stacks[2]
	if fork.cRec != nil {
		t.Fatal("recovered fork still has cRec state")
	}
	if got := fork.top().state; got != StateID(2) {
		t.Fatalf("recovered fork top state = %d, want 2", got)
	}
	if n := stackEntryNode(fork.top()); n == nil || n.symbol != errorSymbol {
		t.Fatalf("recovered fork top node = %v, want ERROR node", n)
	}
}

func TestCRecoverStrategy1ElectionIgnoresBetterVersionForNoActionEntry(t *testing.T) {
	parser := cRecoveryElectionTestParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	group := &cRecGroup{}
	stacks := []glrStack{
		cRecoveryElectionStack(arena, 2, 3, 2500, group, 0, []cStackSummaryEntry{
			{depth: 0, state: 4, posBytes: 0},
			{depth: 1, state: 2, posBytes: 2490},
		}),
		cRecoveryElectionPlainStack(arena, 4, 4, 2500),
	}
	nodeCount := 0
	didRecover, forked := parser.cRecoverStrategy1Election(&stacks, group, Token{Symbol: 1}, &nodeCount, arena, nil, nil, nil)
	if !didRecover || !forked {
		t.Fatalf("strategy 1 election = didRecover %v forked %v, want true/true", didRecover, forked)
	}
	if len(stacks) != 3 {
		t.Fatalf("stack count = %d, want recovered fork appended", len(stacks))
	}
	if got := stacks[2].top().state; got != StateID(2) {
		t.Fatalf("recovered fork top state = %d, want 2", got)
	}
}

func TestCRecoverStrategy1ElectionUsesOwnerMemberForBetterVersionCost(t *testing.T) {
	parser := cRecoveryElectionTestParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	group := &cRecGroup{}
	entry := cStackSummaryEntry{depth: 1, state: 2, posBytes: 0}
	stacks := []glrStack{
		cRecoveryElectionStack(arena, 4, 3, 2500, group, 0, nil),
		cRecoveryElectionStack(arena, 2, 3, 10, group, 1, []cStackSummaryEntry{entry}),
		cRecoveryElectionPlainStack(arena, 4, 4, 2500),
	}
	nodeCount := 0
	didRecover, forked := parser.cRecoverStrategy1Election(&stacks, group, Token{Symbol: 1}, &nodeCount, arena, nil, nil, nil)
	if !didRecover || !forked {
		t.Fatalf("strategy 1 election = didRecover %v forked %v, want true/true", didRecover, forked)
	}
	if len(stacks) != 4 {
		t.Fatalf("stack count = %d, want recovered fork appended", len(stacks))
	}
	if got := stacks[3].top().state; got != StateID(2) {
		t.Fatalf("recovered fork top state = %d, want 2", got)
	}
}

func cRecoveryElectionTestParser() *Parser {
	lang := &Language{
		TokenCount:  2,
		StateCount:  5,
		SymbolCount: 3,
		ParseTable: [][]uint16{
			nil,
			nil,
			{0, 1},
			nil,
			nil,
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 3}}},
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: false, Named: false},
			{Name: "tok", Visible: true, Named: true},
			{Name: "node", Visible: true, Named: true},
		},
	}
	return &Parser{language: lang, denseLimit: len(lang.ParseTable)}
}

func cRecoveryElectionStack(arena *nodeArena, base, top StateID, endByte uint32, group *cRecGroup, groupOrder int, summary []cStackSummaryEntry) glrStack {
	s := cRecoveryElectionPlainStack(arena, base, top, endByte)
	s.cRec = &cRecoverState{
		group:      group,
		groupOrder: groupOrder,
		summary:    summary,
	}
	s.cNodeBaseline = 1
	return s
}

func cRecoveryElectionPlainStack(arena *nodeArena, base, top StateID, endByte uint32) glrStack {
	s := newGLRStack(base)
	n := newLeafNodeInArena(arena, 1, true, 0, endByte, Point{}, Point{Column: endByte})
	s.pushEntry(newStackEntryNode(top, n), nil, nil)
	return s
}

func cRecoveryBaselineStack(arena *nodeArena, start StateID, visibleNodes int) glrStack {
	s := newGLRStack(start)
	for i := 0; i < visibleNodes; i++ {
		startByte := uint32(i)
		endByte := startByte + 1
		n := newLeafNodeInArena(arena, 1, true, startByte, endByte, Point{Column: startByte}, Point{Column: endByte})
		s.pushEntry(newStackEntryNode(start+StateID(i)+1, n), nil, nil)
	}
	return s
}

func TestCDoAllPotentialReductionsCollapsesSamePopTargetSlicesByCParentSelection(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	altRight := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	altRight.dynamicPrecedence = 9
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  leftNode,
		entry: newStackEntryNode(3, altRight),
	})
	start := glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(nil, start, 0, true, Token{}, &nodeCount, arena, nil, &scratch, nil)
	if canShift {
		t.Fatal("canShift = true, want false")
	}
	if len(parser.pendingForkStacks) != 0 {
		t.Fatalf("pending forks = %d, want 0", len(parser.pendingForkStacks))
	}
	if len(versions) != 1 {
		t.Fatalf("version count = %d, want one selected version", len(versions))
	}
	if versions[0].dead {
		t.Fatal("selected reduction version is dead")
	}
	// The forked GSS reducer now applies the selected primary fork directly;
	// C recovery's invariant is that no internal pending fork survives this
	// potential-reduction probe.
	top := versions[0].top()
	if !stackEntryHasNode(top) {
		t.Fatal("selected reduction version top has no node")
	}
	if got, want := stackEntryNode(top).symbol, Symbol(4); got != want {
		t.Fatalf("selected reduction symbol = %d, want %d", got, want)
	}
	parent := stackEntryNode(top)
	if len(parent.children) != 2 || parent.children[1] != altRight {
		t.Fatal("selected parent children did not preserve the C-selected same-pop child array")
	}
	if got := versions[0].gss.head.linkCount(); got != 1 {
		t.Fatalf("reduced top link count = %d, want one selected same-pop alternative", got)
	}
}

func TestCDoAllPotentialReductionsCollapsesSamePopWithTrailingExtra(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	rightLow := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightHigh := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightHigh.dynamicPrecedence = 9
	extraLow := newLeafNodeInArena(arena, 5, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	extraLow.setExtra(true)
	extraHigh := newLeafNodeInArena(arena, 6, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	extraHigh.setExtra(true)

	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightLowNode := scratch.allocNode(newStackEntryNode(3, rightLow), leftNode, 3)
	rightHighNode := scratch.allocNode(newStackEntryNode(3, rightHigh), leftNode, 3)
	head := scratch.allocNode(newStackEntryNode(3, extraLow), rightLowNode, 4)
	head.extraLinks = append(head.extraLinks, gssMainLink{
		prev:  rightHighNode,
		entry: newStackEntryNode(3, extraHigh),
	})
	start := glrStack{gss: gssStack{head: head}, byteOffset: 3}

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(nil, start, 0, true, Token{}, &nodeCount, arena, nil, &scratch, nil)
	if canShift {
		t.Fatal("canShift = true, want false")
	}
	if len(versions) != 1 {
		t.Fatalf("version count = %d, want one selected version", len(versions))
	}
	if got := versions[0].gss.head.linkCount(); got != 1 {
		t.Fatalf("top link count = %d, want one selected extra replay path", got)
	}
	if stackEntryNode(versions[0].top()) != extraHigh {
		t.Fatal("selected trailing extra did not come from the C-selected same-pop child path")
	}
	parent := stackEntryNode(versions[0].gss.head.prev.entry)
	if parent == nil || parent.symbol != 4 {
		t.Fatalf("replayed extra predecessor = %+v, want reduced parent", parent)
	}
	if len(parent.children) != 2 || parent.children[1] != rightHigh {
		t.Fatal("trailing-extra same-pop collapse did not preserve selected parent children")
	}
}

func TestCAppendActionReductionVersionsCollapsesSamePopBeforeOlderMerge(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	olderPop := scratch.allocNode(stackEntry{state: 8}, nil, 1)
	popTo := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	original := glrStack{gss: gssStack{head: scratch.allocNode(stackEntry{state: 3}, popTo, 2)}, byteOffset: 2}

	olderChild := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	lowChild := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	highChild := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	highChild.dynamicPrecedence = 7
	olderParent := newParentNodeInArena(arena, 4, true, []*Node{olderChild}, nil, 0)
	lowParent := newParentNodeInArena(arena, 4, true, []*Node{lowChild}, nil, 0)
	highParent := newParentNodeInArena(arena, 4, true, []*Node{highChild}, nil, 0)
	highParent.dynamicPrecedence = highChild.dynamicPrecedence

	versions := []glrStack{
		{gss: gssStack{head: scratch.allocNode(newStackEntryNode(9, olderParent), olderPop, 2)}, byteOffset: 2},
		original,
	}
	candidates := []glrStack{
		{gss: gssStack{head: scratch.allocNode(newStackEntryNode(9, lowParent), popTo, 2)}, byteOffset: 2},
		{gss: gssStack{head: scratch.allocNode(newStackEntryNode(9, highParent), popTo, 2)}, byteOffset: 2},
	}

	versions, reductionVersion := parser.cAppendActionReductionVersions(versions, candidates, 1, arena)
	if reductionVersion != -1 {
		t.Fatalf("reduction version = %d, want merge into older existing version", reductionVersion)
	}
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want older plus original", len(versions))
	}
	if got := versions[0].gss.head.linkCount(); got != 2 {
		t.Fatalf("older merged head link count = %d, want one selected reduction link added", got)
	}
	_, mergedEntry := versions[0].gss.head.link(1)
	if stackEntryNode(mergedEntry) != highParent {
		t.Fatal("older merge received uncollapsed lower-precedence same-pop parent")
	}
}

func newCRecoverySyntheticReduceParser() *Parser {
	lang := &Language{
		TokenCount:  3,
		StateCount:  10,
		SymbolCount: 7,
		ParseTable: [][]uint16{
			nil,
			{0, 0, 0, 0, 2}, // goto parent(4) -> state 9.
			nil,
			{0, 1, 1},
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 4, ChildCount: 2}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 9}}},
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "eof", Visible: true, Named: true},
			{Name: "a", Visible: true, Named: true},
			{Name: "b", Visible: true, Named: true},
			{Name: "unused", Visible: true, Named: true},
			{Name: "parent", Visible: true, Named: true},
			{Name: "extra_low", Visible: false, Named: false},
			{Name: "extra_high", Visible: false, Named: false},
		},
	}
	return &Parser{language: lang, denseLimit: len(lang.ParseTable)}
}

func TestCDoAllPotentialReductionsDistinguishesEOFFromAnyLookahead(t *testing.T) {
	lang := &Language{
		TokenCount:  2,
		StateCount:  4,
		SymbolCount: 3,
		ParseTable: [][]uint16{
			nil,
			{1: 1}, // state 1 can shift a real token, but has no EOF action.
			{0: 2}, // state 2 can accept EOF.
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 3}}},
			{Actions: []ParseAction{{Type: ParseActionAccept}}},
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: true, Named: true},
			{Name: "tok", Visible: true, Named: true},
			{Name: "node", Visible: true, Named: true},
		},
	}
	parser := &Parser{language: lang, denseLimit: len(lang.ParseTable)}
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(nil, newGLRStack(1), 0, true, Token{}, &nodeCount, arena, nil, nil, nil)
	if !canShift || len(versions) != 1 {
		t.Fatalf("any-lookahead reductions: canShift=%v versions=%d, want true/1", canShift, len(versions))
	}

	versions, canShift = parser.cDoAllPotentialReductions(nil, newGLRStack(1), 0, false, Token{}, &nodeCount, arena, nil, nil, nil)
	if canShift || len(versions) != 0 {
		t.Fatalf("exact EOF reductions on non-EOF state: canShift=%v versions=%d, want false/0", canShift, len(versions))
	}

	versions, canShift = parser.cDoAllPotentialReductions(nil, newGLRStack(2), 0, false, Token{}, &nodeCount, arena, nil, nil, nil)
	if !canShift || len(versions) != 1 {
		t.Fatalf("exact EOF accept reductions: canShift=%v versions=%d, want true/1", canShift, len(versions))
	}
}

func TestCResultSelectionPrefersLaterEqualErrorCostTree(t *testing.T) {
	lang := &Language{
		TokenCount:  2,
		StateCount:  2,
		SymbolCount: 3,
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: true, Named: true},
			{Name: "identifier", Visible: true, Named: true},
			{Name: "translation_unit", Visible: true, Named: true},
		},
	}
	parser := &Parser{language: lang, errorCostCompetition: true}
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	leftMissing := newLeafNodeInArena(arena, 1, true, 0, 0, Point{}, Point{})
	leftMissing.setMissing(true)
	leftMissing.setHasError(true)
	rightMissing := newLeafNodeInArena(arena, 1, true, 0, 0, Point{}, Point{})
	rightMissing.setMissing(true)
	rightMissing.setHasError(true)

	left := glrStack{accepted: true}
	left.pushEntry(newStackEntryNode(1, leftMissing), nil, nil)
	right := glrStack{accepted: true}
	right.pushEntry(newStackEntryNode(1, rightMissing), nil, nil)

	if lc, rc := parser.cStackResultErrorCost(&left), parser.cStackResultErrorCost(&right); lc == 0 || lc != rc {
		t.Fatalf("test setup costs = %d/%d, want equal nonzero", lc, rc)
	}
	if got := stackCompareForResultSelection(parser, arena, &right, &left, false); got <= 0 {
		t.Fatalf("later equal-error candidate compare = %d, want preferred", got)
	}
}
