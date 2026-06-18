package gotreesitter

import "testing"

func buildTwoWindowFullGSSReduceCase(t *testing.T, scratch *gssScratch, arena *nodeArena) (*Parser, glrStack, ParseAction) {
	t.Helper()

	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	altRight := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  leftNode,
		entry: newStackEntryNode(3, altRight),
	})

	parser := &Parser{language: &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "eof", Visible: true, Named: true},
			{Name: "left", Visible: true, Named: true},
			{Name: "right", Visible: true, Named: true},
			{Name: "parent", Visible: true, Named: true},
		},
	}}
	stack := glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}
	act := ParseAction{Type: ParseActionReduce, Symbol: 3, ChildCount: 2, DynamicPrecedence: 4}
	return parser, stack, act
}

func setSyntheticEOFAction(t *testing.T, parser *Parser, state StateID, actions []ParseAction) {
	t.Helper()
	if parser == nil || parser.language == nil {
		t.Fatal("parser/language must be initialized")
	}
	if parser.language.SymbolCount == 0 {
		parser.language.SymbolCount = 4
	}
	if parser.language.TokenCount == 0 {
		parser.language.TokenCount = 3
	}
	if parser.language.StateCount <= uint32(state) {
		parser.language.StateCount = uint32(state) + 1
	}
	for len(parser.language.ParseTable) <= int(state) {
		parser.language.ParseTable = append(parser.language.ParseTable, make([]uint16, parser.language.SymbolCount))
	}
	for i := range parser.language.ParseTable {
		if len(parser.language.ParseTable[i]) < int(parser.language.SymbolCount) {
			row := make([]uint16, parser.language.SymbolCount)
			copy(row, parser.language.ParseTable[i])
			parser.language.ParseTable[i] = row
		}
	}
	if len(parser.language.ParseActions) == 0 {
		parser.language.ParseActions = append(parser.language.ParseActions, ParseActionEntry{})
	}
	parser.language.ParseActions = append(parser.language.ParseActions, ParseActionEntry{Actions: actions})
	parser.language.ParseTable[state][0] = uint16(len(parser.language.ParseActions) - 1)
	parser.denseLimit = len(parser.language.ParseTable)
}

func setSyntheticPostReducePackingSafeEOF(t *testing.T, parser *Parser, states ...StateID) {
	t.Helper()
	for _, state := range states {
		setSyntheticEOFAction(t, parser, state, []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 1}})
	}
}

func TestFastVisibleReduceFromGSSDeclinesMultiLinkSpan(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := NewLeafNode(1, true, 0, 1, Point{}, Point{Column: 1})
	right := NewLeafNode(2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	altRight := NewLeafNode(2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  leftNode,
		entry: newStackEntryNode(3, altRight),
	})

	parser := &Parser{language: &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "eof", Visible: true, Named: true},
			{Name: "left", Visible: true, Named: true},
			{Name: "right", Visible: true, Named: true},
			{Name: "parent", Visible: true, Named: true},
		},
	}}
	stack := &glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}
	act := ParseAction{Type: ParseActionReduce, Symbol: 3, ChildCount: 2}
	var anyReduced bool
	nodeCount := 0
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	if parser.tryFastVisibleReduceActionFromGSS(stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, false, false) {
		t.Fatal("tryFastVisibleReduceActionFromGSS = true, want false for multi-link span")
	}
	if anyReduced {
		t.Fatal("anyReduced = true, want false")
	}
	if nodeCount != 0 {
		t.Fatalf("nodeCount = %d, want 0", nodeCount)
	}
	if stack.gss.head != rightNode {
		t.Fatal("stack mutated by declined fast reduce")
	}
}

func TestFaithfulForkReduceFromGSSLinkedWindowsCoalescesPostReduceHead(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })
	if perfCountersEnabled {
		ResetPerfCounters()
	}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	parser, stack, act := buildTwoWindowFullGSSReduceCase(t, &scratch, arena)
	setSyntheticPostReducePackingSafeEOF(t, parser, 1)
	var anyReduced bool
	nodeCount := 0

	parser.applyReduceActionFromGSS(&stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false, false)
	if stack.dead {
		t.Fatal("stack.dead = true, want false")
	}
	if !anyReduced {
		t.Fatal("anyReduced = false, want true")
	}
	if nodeCount != 2 {
		t.Fatalf("nodeCount = %d, want 2", nodeCount)
	}
	if len(parser.pendingForkStacks) != 0 {
		t.Fatalf("pending forks = %d, want 0", len(parser.pendingForkStacks))
	}
	if stack.gss.head == nil {
		t.Fatal("post-reduce GSS head is nil")
	}
	if got := stack.gss.head.linkCount(); got != 2 {
		t.Fatalf("post-reduce head linkCount = %d, want 2", got)
	}
	if perfCountersEnabled {
		perf := PerfCountersSnapshot()
		if perf.ReduceForkCalls != 1 || perf.ReduceForkWindows != 2 || perf.ReduceForkMaxWindows != 2 {
			t.Fatalf("reduce fork counters = calls:%d windows:%d max:%d, want 1/2/2", perf.ReduceForkCalls, perf.ReduceForkWindows, perf.ReduceForkMaxWindows)
		}
		if perf.PostReduceMergeAttempts != 1 || perf.PostReduceMergePrimarySuccesses != 1 || perf.PendingForkStackAppends != 0 {
			t.Fatalf("post-reduce merge counters = attempts:%d primary:%d appends:%d, want 1/1/0", perf.PostReduceMergeAttempts, perf.PostReduceMergePrimarySuccesses, perf.PendingForkStackAppends)
		}
	}
}

func TestFaithfulForkReduceImmediateAcceptKeepsPendingFork(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })
	if perfCountersEnabled {
		ResetPerfCounters()
	}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	parser, stack, act := buildTwoWindowFullGSSReduceCase(t, &scratch, arena)
	setSyntheticEOFAction(t, parser, 1, []ParseAction{{Type: ParseActionAccept}})
	var anyReduced bool
	nodeCount := 0

	parser.applyReduceActionFromGSS(&stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false, false)
	if stack.dead {
		t.Fatal("stack.dead = true, want false")
	}
	if !anyReduced {
		t.Fatal("anyReduced = false, want true")
	}
	if nodeCount != 2 {
		t.Fatalf("nodeCount = %d, want 2", nodeCount)
	}
	if got := stack.gss.head.linkCount(); got != 1 {
		t.Fatalf("primary post-reduce head linkCount = %d, want 1", got)
	}
	if len(parser.pendingForkStacks) != 1 {
		t.Fatalf("pending forks = %d, want 1", len(parser.pendingForkStacks))
	}
	if got := parser.pendingForkStacks[0].gss.head.linkCount(); got != 1 {
		t.Fatalf("pending post-reduce head linkCount = %d, want 1", got)
	}
	if stack.top().state != parser.pendingForkStacks[0].top().state {
		t.Fatalf("primary top state = %d, pending top state = %d, want same accept-capable state", stack.top().state, parser.pendingForkStacks[0].top().state)
	}
	if perfCountersEnabled {
		perf := PerfCountersSnapshot()
		if perf.ReduceForkCalls != 1 || perf.ReduceForkWindows != 2 || perf.ReduceForkMaxWindows != 2 {
			t.Fatalf("reduce fork counters = calls:%d windows:%d max:%d, want 1/2/2", perf.ReduceForkCalls, perf.ReduceForkWindows, perf.ReduceForkMaxWindows)
		}
		if perf.PostReduceMergeFinalizationRiskSkips != 1 || perf.PostReduceMergeAttempts != 0 || perf.PendingForkStackAppends != 1 || perf.PendingForkStacksMaxLen != 1 {
			t.Fatalf("finalization-risk counters = skips:%d attempts:%d appends:%d max_pending:%d, want 1/0/1/1", perf.PostReduceMergeFinalizationRiskSkips, perf.PostReduceMergeAttempts, perf.PendingForkStackAppends, perf.PendingForkStacksMaxLen)
		}
	}
}

func TestFaithfulForkReduceFromPackedGSSHeadEnumeratesReducedParents(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	parser, stack, act := buildTwoWindowFullGSSReduceCase(t, &scratch, arena)
	setSyntheticPostReducePackingSafeEOF(t, parser, 1)
	var anyReduced bool
	nodeCount := 0

	parser.applyReduceActionFromGSS(&stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false, false)
	forks := reduceWindowsFromGSS(&stack, 1, maxStacksPerMergeKey)
	if len(forks) != 2 {
		t.Fatalf("reduced-parent windows = %d, want 2", len(forks))
	}
	for i, fork := range forks {
		if len(fork.window) != 1 {
			t.Fatalf("fork %d window length = %d, want 1", i, len(fork.window))
		}
		parent := stackEntryNode(fork.window[0])
		if parent == nil || parent.symbol != act.Symbol {
			t.Fatalf("fork %d parent symbol = %v, want %d", i, parent, act.Symbol)
		}
	}
}

func TestFaithfulForkReduceTargetMismatchFallsBackToPendingFork(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })
	if perfCountersEnabled {
		ResetPerfCounters()
	}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)

	altBase := scratch.allocNode(stackEntry{state: 9}, nil, 1)
	altLeft := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	altLeftNode := scratch.allocNode(newStackEntryNode(8, altLeft), altBase, 2)
	altRight := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  altLeftNode,
		entry: newStackEntryNode(7, altRight),
	})

	parser := &Parser{language: &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "eof", Visible: true, Named: true},
			{Name: "left", Visible: true, Named: true},
			{Name: "right", Visible: true, Named: true},
			{Name: "parent", Visible: true, Named: true},
		},
	}}
	setSyntheticPostReducePackingSafeEOF(t, parser, 9)
	stack := glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}
	act := ParseAction{Type: ParseActionReduce, Symbol: 3, ChildCount: 2}
	var anyReduced bool
	nodeCount := 0

	parser.applyReduceActionFromGSS(&stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false, false)
	if len(parser.pendingForkStacks) != 1 {
		t.Fatalf("pending forks = %d, want 1", len(parser.pendingForkStacks))
	}
	if stack.top().state == parser.pendingForkStacks[0].top().state {
		t.Fatalf("primary and pending top states both = %d, want mismatch fallback", stack.top().state)
	}
	if got := stack.gss.head.linkCount(); got != 1 {
		t.Fatalf("primary post-reduce head linkCount = %d, want 1", got)
	}
	if perfCountersEnabled {
		perf := PerfCountersSnapshot()
		if perf.ReduceForkCalls != 1 || perf.ReduceForkWindows != 2 || perf.ReduceForkMaxWindows != 2 {
			t.Fatalf("reduce fork counters = calls:%d windows:%d max:%d, want 1/2/2", perf.ReduceForkCalls, perf.ReduceForkWindows, perf.ReduceForkMaxWindows)
		}
		if perf.PostReduceMergeAttempts != 1 || perf.PostReduceMergePrimarySuccesses != 0 || perf.PendingForkStackAppends != 1 || perf.PendingForkStacksMaxLen != 1 {
			t.Fatalf("target-mismatch counters = attempts:%d primary:%d appends:%d max_pending:%d, want 1/0/1/1", perf.PostReduceMergeAttempts, perf.PostReduceMergePrimarySuccesses, perf.PendingForkStackAppends, perf.PendingForkStacksMaxLen)
		}
	}
}

func TestPostReduceForkMergeFinalizationRiskPredicate(t *testing.T) {
	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	stack := &glrStack{gss: gssStack{head: scratch.allocNode(stackEntry{state: 2}, base, 2)}, byteOffset: 1}
	eof := Token{Symbol: 0}

	t.Run("nil parser", func(t *testing.T) {
		var parser *Parser
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, eof) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("dead stack", func(t *testing.T) {
		parser := &Parser{language: &Language{}}
		dead := *stack
		dead.dead = true
		if !parser.postReduceForkMergeHasFinalizationRisk(&dead, eof) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("non EOF lookahead", func(t *testing.T) {
		parser := &Parser{language: &Language{}}
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, Token{Symbol: 1}) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("missing EOF action", func(t *testing.T) {
		parser := &Parser{language: &Language{
			StateCount:     3,
			SymbolCount:    4,
			TokenCount:     3,
			ParseActions:   []ParseActionEntry{{}},
			ParseTable:     [][]uint16{{0, 0, 0, 0}, {0, 0, 0, 0}, {0, 0, 0, 0}},
			SymbolMetadata: []SymbolMetadata{{Name: "eof"}},
		}}
		parser.denseLimit = len(parser.language.ParseTable)
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, eof) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("out of range EOF action", func(t *testing.T) {
		parser := &Parser{language: &Language{
			StateCount:     3,
			SymbolCount:    4,
			TokenCount:     3,
			ParseActions:   []ParseActionEntry{{}},
			ParseTable:     [][]uint16{{0, 0, 0, 0}, {0, 0, 0, 0}, {5, 0, 0, 0}},
			SymbolMetadata: []SymbolMetadata{{Name: "eof"}},
		}}
		parser.denseLimit = len(parser.language.ParseTable)
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, eof) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("single childful EOF reduce", func(t *testing.T) {
		parser := &Parser{language: &Language{SymbolMetadata: []SymbolMetadata{{Name: "eof"}}}}
		setSyntheticEOFAction(t, parser, 2, []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 1}})
		if parser.postReduceForkMergeHasFinalizationRisk(stack, eof) {
			t.Fatal("risk = true, want false")
		}
	})
	t.Run("single childful EOF reduce with no-lookahead is risky", func(t *testing.T) {
		parser := &Parser{language: &Language{SymbolMetadata: []SymbolMetadata{{Name: "eof"}}}}
		setSyntheticEOFAction(t, parser, 2, []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 1}})
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, Token{Symbol: 1, NoLookahead: true}) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("zero child EOF reduce", func(t *testing.T) {
		parser := &Parser{language: &Language{SymbolMetadata: []SymbolMetadata{{Name: "eof"}}}}
		setSyntheticEOFAction(t, parser, 2, []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 0}})
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, eof) {
			t.Fatal("risk = false, want true")
		}
	})
	t.Run("mixed EOF actions", func(t *testing.T) {
		parser := &Parser{language: &Language{SymbolMetadata: []SymbolMetadata{{Name: "eof"}}}}
		setSyntheticEOFAction(t, parser, 2, []ParseAction{
			{Type: ParseActionReduce, Symbol: 3, ChildCount: 1},
			{Type: ParseActionAccept},
		})
		if !parser.postReduceForkMergeHasFinalizationRisk(stack, eof) {
			t.Fatal("risk = false, want true")
		}
	})
}

func TestFaithfulForkReduceEOFNoActionKeepsPendingFork(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	parser, stack, act := buildTwoWindowFullGSSReduceCase(t, &scratch, arena)
	parser.language.StateCount = 4
	parser.language.SymbolCount = 4
	parser.language.TokenCount = 3
	parser.language.ParseActions = []ParseActionEntry{{}}
	parser.language.ParseTable = [][]uint16{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
	parser.denseLimit = len(parser.language.ParseTable)
	var anyReduced bool
	nodeCount := 0

	parser.applyReduceActionFromGSS(&stack, act, Token{Symbol: 0}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false, false)
	if stack.dead {
		t.Fatal("stack.dead = true, want false")
	}
	if got := stack.gss.head.linkCount(); got != 1 {
		t.Fatalf("primary post-reduce head linkCount = %d, want 1", got)
	}
	if len(parser.pendingForkStacks) != 1 {
		t.Fatalf("pending forks = %d, want 1", len(parser.pendingForkStacks))
	}
}

func TestFaithfulForkReduceEOFZeroChildReduceKeepsPendingFork(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	parser, stack, act := buildTwoWindowFullGSSReduceCase(t, &scratch, arena)
	setSyntheticEOFAction(t, parser, 1, []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 0}})
	var anyReduced bool
	nodeCount := 0

	parser.applyReduceActionFromGSS(&stack, act, Token{Symbol: 0}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false, false)
	if stack.dead {
		t.Fatal("stack.dead = true, want false")
	}
	if got := stack.gss.head.linkCount(); got != 1 {
		t.Fatalf("primary post-reduce head linkCount = %d, want 1", got)
	}
	if len(parser.pendingForkStacks) != 1 {
		t.Fatalf("pending forks = %d, want 1", len(parser.pendingForkStacks))
	}
}

func TestTryMergePostReduceForkDeclinesAcceptedStacks(t *testing.T) {
	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := glrStack{
		gss:        gssStack{head: scratch.allocNode(stackEntry{state: 2}, base, 2)},
		byteOffset: 3,
		accepted:   true,
	}
	right := glrStack{
		gss:        gssStack{head: scratch.allocNode(stackEntry{state: 2}, base, 2)},
		byteOffset: 3,
		accepted:   true,
	}

	if tryMergePostReduceFork(&left, &right) {
		t.Fatal("tryMergePostReduceFork = true, want false for accepted stacks")
	}
}

func TestNoTreeReduceFromGSSRejectsMultiLinkSpan(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newNoTreeLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newNoTreeLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNoTreeNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNoTreeNode(3, right), leftNode, 3)
	altRight := newNoTreeLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  leftNode,
		entry: newStackEntryNoTreeNode(3, altRight),
	})

	parser := &Parser{language: &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "eof", Visible: true, Named: true},
			{Name: "left", Visible: true, Named: true},
			{Name: "right", Visible: true, Named: true},
			{Name: "parent", Visible: true, Named: true},
		},
	}}
	stack := &glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}
	act := ParseAction{Type: ParseActionReduce, Symbol: 3, ChildCount: 2}
	var anyReduced bool
	nodeCount := 0

	parser.applyNoTreeReduceActionFromGSS(stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, &scratch, nil, nil, false)
	if !stack.dead {
		t.Fatal("stack.dead = false, want true for no-tree multi-link GSS reduce")
	}
	if anyReduced {
		t.Fatal("anyReduced = true, want false")
	}
	if nodeCount != 0 {
		t.Fatalf("nodeCount = %d, want 0", nodeCount)
	}
	if len(parser.pendingForkStacks) != 0 {
		t.Fatalf("pending forks = %d, want 0", len(parser.pendingForkStacks))
	}
}

func TestRejectUndrainedPendingForkStacksClearsAndKills(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := &Parser{pendingForkStacks: []glrStack{newGLRStack(9)}}
	stack := newGLRStack(1)

	if !parser.rejectUndrainedPendingForkStacks(&stack) {
		t.Fatal("rejectUndrainedPendingForkStacks = false, want true")
	}
	if len(parser.pendingForkStacks) != 0 {
		t.Fatalf("pending forks = %d, want 0", len(parser.pendingForkStacks))
	}
	if !stack.dead {
		t.Fatal("stack.dead = false, want true")
	}
}

func TestBuildReduceChainHintsUsesLanguageMetadata(t *testing.T) {
	t.Setenv("GOT_GLR_REDUCE_CHAIN_HINTS", "1")
	ResetParseEnvConfigCacheForTests()
	t.Cleanup(ResetParseEnvConfigCacheForTests)

	lang := &Language{
		Name:        "python",
		StateCount:  10,
		SymbolCount: 10,
		SymbolNames: []string{
			"", "", "", "", "", "", "", "", "", "",
		},
		ReduceChainHints: []ReduceChainHint{{
			StartState:     StateID(3),
			Lookahead:      Symbol(2),
			TerminalStates: []StateID{StateID(4), StateID(5)},
			TerminalAction: ReduceChainTerminalSingleShift,
			MaxSteps:       7,
		}},
	}

	got := buildReduceChainHints(lang)
	if len(got) != 1 {
		t.Fatalf("hint count = %d, want 1", len(got))
	}
	hint := got[0]
	if hint.startState != StateID(3) || hint.lookahead != Symbol(2) || hint.maxSteps != 7 {
		t.Fatalf("hint = %+v, want state=3 lookahead=2 maxSteps=7", hint)
	}
	if hint.terminalAction != classifiedParseActionSingleShift {
		t.Fatalf("terminal action = %d, want single shift", hint.terminalAction)
	}
	if len(hint.terminalStates) != 2 || hint.terminalStates[0] != StateID(4) || hint.terminalStates[1] != StateID(5) {
		t.Fatalf("terminal states = %v, want [4 5]", hint.terminalStates)
	}

	lang.ReduceChainHints[0].TerminalStates[0] = StateID(9)
	if hint.terminalStates[0] != StateID(4) {
		t.Fatalf("internal hint terminal states alias language metadata: got %v", hint.terminalStates)
	}
}

func TestReduceChainHintForUsesStateIndex(t *testing.T) {
	p := &Parser{
		reduceChainHints: []reduceChainHint{
			{startState: StateID(8), lookahead: Symbol(3), maxSteps: 4},
			{startState: StateID(10), lookahead: Symbol(4), maxSteps: 5},
			{startState: StateID(10), lookahead: Symbol(5), maxSteps: 6},
		},
	}
	p.reduceChainHintByState = buildReduceChainHintIndex(p.reduceChainHints)

	hint, ok := p.reduceChainHintFor(StateID(8), Symbol(3))
	if !ok || hint.maxSteps != 4 {
		t.Fatalf("hint for state=8 lookahead=3 = %+v, %v; want maxSteps=4, true", hint, ok)
	}
	hint, ok = p.reduceChainHintFor(StateID(10), Symbol(5))
	if !ok || hint.maxSteps != 6 {
		t.Fatalf("hint for duplicate state=10 lookahead=5 = %+v, %v; want maxSteps=6, true", hint, ok)
	}
	if _, ok := p.reduceChainHintFor(StateID(9), Symbol(3)); ok {
		t.Fatal("unexpected hint for state without entry")
	}
	if _, ok := p.reduceChainHintFor(StateID(10), Symbol(6)); ok {
		t.Fatal("unexpected hint for duplicate state with unmatched lookahead")
	}
}

func TestBuildSingleTokenWrapperSymbols(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "single_wrapper", Visible: true, Named: true},
			{Name: "multi_wrapper", Visible: true, Named: true},
			{Name: "statement", Visible: true, Named: true},
		},
		ParseActions: []ParseActionEntry{
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 1, ChildCount: 1, ProductionID: 10}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 2, ChildCount: 1, ProductionID: 11}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 2, ChildCount: 1, ProductionID: 12}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 2, ProductionID: 13}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 1}}},
		},
	}

	got := buildSingleTokenWrapperSymbols(lang)
	if !got[1] {
		t.Fatal("expected single_wrapper to be marked as a single-token wrapper")
	}
	if got[2] {
		t.Fatal("did not expect multi_wrapper to be marked as a single-token wrapper")
	}
	if got[3] {
		t.Fatal("did not expect statement to be marked as a single-token wrapper")
	}
}

func TestCanCollapseNamedLeafWrapperSingleAnonymousToken(t *testing.T) {
	p := &Parser{
		language: &Language{
			SymbolMetadata: []SymbolMetadata{
				{Name: "EOF"},
				{Name: "optional_chain", Visible: true, Named: true},
				{Name: "?.", Visible: true, Named: false},
				{Name: "identifier", Visible: true, Named: true},
				{Name: "_hidden", Visible: false, Named: false},
			},
		},
	}

	if !p.canCollapseNamedLeafWrapper(1, 2) {
		t.Fatal("expected visible named wrapper over visible anonymous token to collapse")
	}
	if p.canCollapseNamedLeafWrapper(1, 3) {
		t.Fatal("did not expect visible named wrapper over visible named child to collapse")
	}
	if p.canCollapseNamedLeafWrapper(1, 4) {
		t.Fatal("did not expect visible named wrapper over invisible child to collapse")
	}
}

// A named rule wrapping a DIFFERENT-named visible anonymous token (e.g.
// optional_chain over "?.") must NOT collapse to a childless leaf: C tree-sitter
// keeps the anonymous token as a visible child (childCount==1). The unary
// self-reduction collapse must therefore decline (return nil) so the normal
// reduce keeps the child.
func TestCollapsibleUnarySelfReductionKeepsDifferentNamedAnonChild(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "optional_chain", Visible: true, Named: true},
			{Name: "?.", Visible: true, Named: false},
		},
	}
	p := &Parser{language: lang}
	arena := newNodeArena(arenaClassFull)
	child := newLeafNodeInArena(arena, 2, false, 1, 3, Point{Column: 1}, Point{Column: 3})
	entries := []stackEntry{newStackEntryNode(0, child)}
	act := ParseAction{Symbol: 1, ChildCount: 1}

	if got := p.collapsibleUnarySelfReduction(act, Token{}, arena, entries, 0, 1, []*Node{child}, nil); got != nil {
		t.Fatalf("expected different-named visible anon child to be kept (no collapse), got node with cc=%d", got.ChildCount())
	}
}

func TestCollapsibleRawUnarySelfReductionKeepsDifferentNamedAnonChild(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "optional_chain", Visible: true, Named: true},
			{Name: "?.", Visible: true, Named: false},
		},
	}
	p := &Parser{language: lang}
	arena := newNodeArena(arenaClassFull)
	child := newLeafNodeInArena(arena, 2, false, 1, 3, Point{Column: 1}, Point{Column: 3})
	entries := []stackEntry{newStackEntryNode(0, child)}
	act := ParseAction{Symbol: 1, ChildCount: 1}

	if got := p.collapsibleRawUnarySelfReduction(act, Token{}, arena, entries, 0, 1); got != nil {
		t.Fatalf("expected different-named visible anon child to be kept (no collapse), got node with cc=%d", got.ChildCount())
	}
}

func TestCollapsibleRawUnarySelfReductionRejectsInvisibleChild(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "optional_chain", Visible: true, Named: true},
			{Name: "_hidden", Visible: false, Named: false},
		},
	}
	p := &Parser{
		language:                 lang,
		singleTokenWrapperSymbol: []bool{false, true, false},
	}
	arena := newNodeArena(arenaClassFull)
	child := newLeafNodeInArena(arena, 2, false, 1, 3, Point{Column: 1}, Point{Column: 3})
	entries := []stackEntry{newStackEntryNode(0, child)}
	act := ParseAction{Symbol: 1, ChildCount: 1}

	if got := p.collapsibleRawUnarySelfReduction(act, Token{}, arena, entries, 0, 1); got != nil {
		t.Fatalf("raw unary collapse returned %v for invisible child", got)
	}
}

func TestReduceProductionHasEffectiveFieldsIgnoresConflictedZeroFields(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "expr", Visible: true, Named: true},
			{Name: "identifier", Visible: true, Named: true},
		},
		FieldNames: []string{"", "left", "right"},
		FieldMapSlices: [][2]uint16{
			{0, 2},
		},
		FieldMapEntries: []FieldMapEntry{
			{FieldID: 1, ChildIndex: 0, Inherited: true},
			{FieldID: 2, ChildIndex: 0, Inherited: true},
		},
		ParseActions: []ParseActionEntry{
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 1, ChildCount: 1, ProductionID: 0}}},
		},
	}
	p := NewParser(lang)
	arena := newNodeArena(arenaClassFull)

	if p.reduceProductionHasFields(0) {
		t.Fatal("reduceProductionHasFields = true, want false for conflicted zero field IDs")
	}
	if p.reduceProductionHasEffectiveFields(1, 0, arena) {
		t.Fatal("reduceProductionHasEffectiveFields = true, want false for conflicted zero field IDs")
	}
	fieldIDs, _ := p.buildFieldIDs(1, 0, arena)
	if got := len(fieldIDs); got != 1 {
		t.Fatalf("buildFieldIDs len = %d, want 1", got)
	}
	if got := fieldIDs[0]; got != 0 {
		t.Fatalf("buildFieldIDs[0] = %d, want 0", got)
	}
}

func TestTryPushPendingNoFieldParentAllowsEffectiveNoFieldProduction(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "expr", Visible: true, Named: true},
			{Name: "identifier", Visible: true, Named: true},
		},
		FieldNames: []string{"", "left", "right"},
		FieldMapSlices: [][2]uint16{
			{0, 2},
		},
		FieldMapEntries: []FieldMapEntry{
			{FieldID: 1, ChildIndex: 0, Inherited: true},
			{FieldID: 2, ChildIndex: 0, Inherited: true},
		},
		ParseActions: []ParseActionEntry{
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 1, ChildCount: 1, ProductionID: 0}}},
		},
	}
	p := NewParser(lang)
	p.pendingFullParents = true
	arena := newNodeArena(arenaClassFull)
	leaf := newCompactFullLeafInArena(arena, 2, true, 1, 3, Point{Column: 1}, Point{Column: 3})
	entry := newStackEntryCompactFullLeaf(4, leaf)
	stack := &glrStack{entries: []stackEntry{entry}}
	act := ParseAction{Symbol: 1, ChildCount: 1, ProductionID: 0}
	anyReduced := false
	nodeCount := 0

	if !p.tryPushPendingNoFieldParent(stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, nil, []stackEntry{entry}, 0, 1, 1, 0, 0) {
		t.Fatal("tryPushPendingNoFieldParent = false, want true for effective no-field production")
	}
	if !anyReduced {
		t.Fatal("anyReduced = false, want true")
	}
	if nodeCount != 1 {
		t.Fatalf("nodeCount = %d, want 1", nodeCount)
	}
	if got := arena.pendingParentRejectedFields; got != 0 {
		t.Fatalf("pendingParentRejectedFields = %d, want 0", got)
	}
	if got := arena.pendingParentCreated; got != 1 {
		t.Fatalf("pendingParentCreated = %d, want 1", got)
	}
	if got := len(stack.entries); got != 1 {
		t.Fatalf("stack entries = %d, want 1", got)
	}
	parent := stackEntryPendingParent(stack.entries[0])
	if parent == nil {
		t.Fatal("stack entry is not a pending parent")
	}
	if got := parent.childEntryCount(); got != 1 {
		t.Fatalf("pending parent child count = %d, want 1", got)
	}
}

func TestTryPushPendingNoFieldParentCountsOrdinaryHiddenNodeRefs(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "expr", Visible: true, Named: true},
			{Name: "_hidden", Visible: false, Named: false},
			{Name: "identifier", Visible: true, Named: true},
		},
	}
	p := NewParser(lang)
	p.pendingFullParents = true
	arena := newNodeArena(arenaClassFull)
	first := newLeafNodeInArena(arena, 3, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	second := newLeafNodeInArena(arena, 3, true, 3, 4, Point{Column: 3}, Point{Column: 4})
	hidden := newParentNodeInArena(arena, 2, false, []*Node{first, second}, nil, 0)
	entry := newStackEntryNode(4, hidden)
	stack := &glrStack{entries: []stackEntry{entry}}
	act := ParseAction{Symbol: 1, ChildCount: 1, ProductionID: 0}
	anyReduced := false
	nodeCount := 0

	if !p.tryPushPendingNoFieldParent(stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, nil, []stackEntry{entry}, 0, 1, 1, 0, 0) {
		t.Fatal("tryPushPendingNoFieldParent = false, want true")
	}
	if got := arena.pendingParentsFlattened; got != 0 {
		t.Fatalf("pendingParentsFlattened = %d, want 0 for ordinary hidden node", got)
	}
	if got := arena.pendingChildRefsFlattened; got != 2 {
		t.Fatalf("pendingChildRefsFlattened = %d, want 2", got)
	}
	parent := stackEntryPendingParent(stack.entries[0])
	if parent == nil {
		t.Fatal("stack entry is not a pending parent")
	}
	if got := parent.childEntryCount(); got != 2 {
		t.Fatalf("pending parent child count = %d, want 2", got)
	}
}

func TestTryPushPendingNoFieldParentKeepsHiddenCompactLeafCompact(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "expr", Visible: true, Named: true},
			{Name: "_hidden", Visible: false, Named: false},
		},
	}
	p := NewParser(lang)
	p.pendingFullParents = true
	arena := newNodeArena(arenaClassFull)
	leaf := newCompactFullLeafInArena(arena, 2, false, 5, 8, Point{Column: 5}, Point{Column: 8})
	entry := newStackEntryCompactFullLeaf(4, leaf)
	stack := &glrStack{entries: []stackEntry{entry}}
	act := ParseAction{Symbol: 1, ChildCount: 1, ProductionID: 0}
	anyReduced := false
	nodeCount := 0

	if !p.tryPushPendingNoFieldParent(stack, act, Token{}, &anyReduced, &nodeCount, arena, nil, nil, []stackEntry{entry}, 0, 1, 1, 0, 0) {
		t.Fatal("tryPushPendingNoFieldParent = false, want true for hidden compact leaf")
	}
	if got := arena.compactFullLeafMaterialized; got != 0 {
		t.Fatalf("compactFullLeafMaterialized = %d, want 0", got)
	}
	parent := stackEntryPendingParent(stack.entries[0])
	if parent == nil {
		t.Fatal("stack entry is not a pending parent")
	}
	if got := parent.childEntryCount(); got != 0 {
		t.Fatalf("pending parent child count = %d, want 0", got)
	}
	if parent.startByte != 5 || parent.endByte != 8 {
		t.Fatalf("pending parent span = [%d,%d], want [5,8]", parent.startByte, parent.endByte)
	}
}

func TestCollapsibleRawUnarySelfReductionEntryCollapsesPendingParentSameSymbol(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "expr", Visible: true, Named: true},
		},
	}
	p := &Parser{language: lang}
	arena := newNodeArena(arenaClassFull)
	parent := newPendingParentInArena(arena, 1, true, 3, nil, 1, 3, Point{Column: 1}, Point{Column: 3}, false)
	entry := newStackEntryPendingParent(4, parent)
	act := ParseAction{Symbol: 1, ChildCount: 1, ProductionID: 9}

	got, ok := p.collapsibleRawUnarySelfReductionEntry(act, Token{}, arena, []stackEntry{entry}, 0, 1)
	if !ok {
		t.Fatal("expected pending parent raw unary reduction to collapse")
	}
	if stackEntryPendingParent(got) != parent {
		t.Fatal("collapsed entry did not preserve pending parent payload")
	}
	setCollapsedUnaryEntryMetadata(&got, act, false, 2, 5)
	if parent.productionID != 9 || parent.preGotoState != 2 || parent.parseState != 5 || got.state != 5 {
		t.Fatalf("pending parent metadata = prod %d pre %d state %d entry %d", parent.productionID, parent.preGotoState, parent.parseState, got.state)
	}
}

func TestCollapsibleRawUnarySelfReductionEntryCollapsesPendingParentInvisibleWrapper(t *testing.T) {
	lang := &Language{
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF"},
			{Name: "_wrapper", Visible: false, Named: false},
			{Name: "expr", Visible: true, Named: true},
		},
	}
	p := &Parser{language: lang}
	arena := newNodeArena(arenaClassFull)
	parent := newPendingParentInArena(arena, 2, true, 3, nil, 1, 3, Point{Column: 1}, Point{Column: 3}, false)
	entry := newStackEntryPendingParent(4, parent)
	act := ParseAction{Symbol: 1, ChildCount: 1, ProductionID: 9}

	got, ok := p.collapsibleRawUnarySelfReductionEntry(act, Token{}, arena, []stackEntry{entry}, 0, 1)
	if !ok {
		t.Fatal("expected invisible wrapper over pending parent to collapse")
	}
	if stackEntryPendingParent(got) != parent {
		t.Fatal("collapsed wrapper did not preserve pending parent payload")
	}
}
