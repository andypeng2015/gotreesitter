package gotreesitter

import (
	"sync/atomic"
	"testing"
	"time"
)

type cancelAtEOFArithmeticTokenSource struct {
	cancel *uint32
	idx    int
}

func (s *cancelAtEOFArithmeticTokenSource) Next() Token {
	tokens := []Token{
		{Symbol: 1, StartByte: 0, EndByte: 1, StartPoint: Point{}, EndPoint: Point{Column: 1}},
		{Symbol: 2, StartByte: 1, EndByte: 2, StartPoint: Point{Column: 1}, EndPoint: Point{Column: 2}},
		{Symbol: 1, StartByte: 2, EndByte: 3, StartPoint: Point{Column: 2}, EndPoint: Point{Column: 3}},
	}
	if s.idx < len(tokens) {
		tok := tokens[s.idx]
		s.idx++
		return tok
	}
	if s.cancel != nil {
		atomic.StoreUint32(s.cancel, 1)
	}
	return Token{Symbol: 0, StartByte: 3, EndByte: 3, StartPoint: Point{Column: 3}, EndPoint: Point{Column: 3}}
}

func TestTransientChildScratchMaterializesReachableNode(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch transientChildScratch
	first := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	second := newLeafNodeInArena(arena, Symbol(2), true, 1, 2, Point{Column: 1}, Point{Column: 2})
	parent := newParentNodeInArenaNoLinksWithFieldSources(arena, Symbol(3), true, []*Node{}, nil, nil, 0, true)

	children := scratch.alloc(2)
	children[0] = first
	children[1] = second
	parent.children = children

	if !scratch.owns(parent.children) {
		t.Fatal("expected parent children to use transient storage before materialization")
	}

	var stack []*Node
	scratch.materializeNode(parent, arena, &stack)

	if scratch.owns(parent.children) {
		t.Fatal("expected parent children to be copied into arena storage")
	}
	if len(parent.children) != 2 || parent.children[0] != first || parent.children[1] != second {
		t.Fatalf("materialized children = %#v, want [%p %p]", parent.children, first, second)
	}

	scratch.reset()
	if len(parent.children) != 2 || parent.children[0] != first || parent.children[1] != second {
		t.Fatal("materialized children were invalidated by transient reset")
	}
}

func TestTransientParentScratchMaterializesReachableParent(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var childScratch transientChildScratch
	var parentScratch transientParentScratch
	first := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	second := newLeafNodeInArena(arena, Symbol(2), true, 1, 2, Point{Column: 1}, Point{Column: 2})
	children := childScratch.alloc(2)
	children[0] = first
	children[1] = second
	parent := parentScratch.allocParent(arena, Symbol(3), true, children, 11, true)
	parent.parseState = 7
	parent.preGotoState = 5

	if !parentScratch.owns(parent) {
		t.Fatal("expected parent to use transient parent storage before materialization")
	}
	if !childScratch.owns(parent.children) {
		t.Fatal("expected parent children to use transient child storage before materialization")
	}

	entries := []stackEntry{newStackEntryNode(parent.parseState, parent)}
	parentScratch.materializeEntries(entries, arena, &childScratch)

	got := stackEntryNode(entries[0])
	if got == nil {
		t.Fatal("materialized entry node = nil")
	}
	if parentScratch.owns(got) {
		t.Fatal("entry still points at transient parent after materialization")
	}
	if childScratch.owns(got.children) {
		t.Fatal("materialized parent still owns transient children")
	}
	if len(got.children) != 2 || got.children[0] != first || got.children[1] != second {
		t.Fatalf("materialized children = %#v, want [%p %p]", got.children, first, second)
	}
	if got.parseState != 7 || got.preGotoState != 5 || got.productionID != 11 {
		t.Fatalf("materialized states = (%d,%d,%d), want (7,5,11)", got.parseState, got.preGotoState, got.productionID)
	}
	if got.StartByte() != 0 || got.EndByte() != 2 {
		t.Fatalf("materialized span = [%d,%d], want [0,2]", got.StartByte(), got.EndByte())
	}
	if got := parentScratch.nodesMaterialized; got != 1 {
		t.Fatalf("nodesMaterialized = %d, want 1", got)
	}

	parentScratch.reset()
	childScratch.reset()
	if len(got.children) != 2 || got.children[0] != first || got.children[1] != second {
		t.Fatal("materialized parent was invalidated by scratch reset")
	}
}

func TestTransientChildScratchMaterializesFieldedArenaParent(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch transientChildScratch
	first := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	second := newLeafNodeInArena(arena, Symbol(2), true, 1, 2, Point{Column: 1}, Point{Column: 2})
	children := scratch.alloc(2)
	children[0] = first
	children[1] = second
	fieldIDs := arena.allocFieldIDSlice(2)
	fieldSources := arena.allocFieldSourceSlice(2)
	fieldIDs[0] = 7
	fieldSources[0] = fieldSourceDirect
	parent := newParentNodeInArenaNoLinksWithFieldSources(arena, Symbol(3), true, children, fieldIDs, fieldSources, 0, true)

	var stack []*Node
	scratch.materializeNode(parent, arena, &stack)

	if scratch.owns(parent.children) {
		t.Fatal("fielded parent still owns transient children")
	}
	if len(parent.children) != 2 || parent.children[0] != first || parent.children[1] != second {
		t.Fatalf("materialized children = %#v, want [%p %p]", parent.children, first, second)
	}
	if len(parent.fieldIDs) != 2 || parent.fieldIDs[0] != 7 {
		t.Fatalf("field IDs = %#v, want first field 7", parent.fieldIDs)
	}
	if len(parent.fieldSources) != 2 || parent.fieldSources[0] != fieldSourceDirect {
		t.Fatalf("field sources = %#v, want first direct", parent.fieldSources)
	}

	scratch.reset()
	if len(parent.children) != 2 || parent.children[0] != first || parent.children[1] != second {
		t.Fatal("fielded parent children were invalidated by transient reset")
	}
}

func TestTransientParentScratchMaterializesRecoveredNodeSlice(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var childScratch transientChildScratch
	var parentScratch transientParentScratch
	first := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	second := newLeafNodeInArena(arena, Symbol(2), true, 1, 2, Point{Column: 1}, Point{Column: 2})
	children := childScratch.alloc(2)
	children[0] = first
	children[1] = second
	parent := parentScratch.allocParent(arena, Symbol(3), true, children, 17, true)
	nodes := []*Node{parent}

	materializeTransientParentNodes(nodes, arena, &parentScratch, &childScratch, nil)

	got := nodes[0]
	if got == nil {
		t.Fatal("materialized node = nil")
	}
	if parentScratch.owns(got) {
		t.Fatal("node slice still points at transient parent")
	}
	if childScratch.owns(got.children) {
		t.Fatal("materialized recovered parent still owns transient children")
	}
	if len(got.children) != 2 || got.children[0] != first || got.children[1] != second {
		t.Fatalf("materialized children = %#v, want [%p %p]", got.children, first, second)
	}
	if got.productionID != 17 {
		t.Fatalf("productionID = %d, want 17", got.productionID)
	}

	parentScratch.reset()
	childScratch.reset()
	if len(got.children) != 2 || got.children[0] != first || got.children[1] != second {
		t.Fatal("recovered materialized parent was invalidated by scratch reset")
	}
}

func TestTransientChildScratchMaterializeNodeUntilStopsWhenCancelled(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch transientChildScratch
	var cancelled uint32 = 1
	parser := &Parser{cancellationFlag: &cancelled}
	leaf := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	parent := newParentNodeInArenaNoLinksWithFieldSources(arena, Symbol(2), true, nil, nil, nil, 0, true)
	children := scratch.alloc(1)
	children[0] = leaf
	parent.children = children

	var stack []*Node
	if got, want := scratch.materializeNodeUntil(parent, arena, &stack, parser), ParseStopCancelled; got != want {
		t.Fatalf("materializeNodeUntil stop reason = %q, want %q", got, want)
	}
	if !scratch.owns(parent.children) {
		t.Fatal("parent children were materialized despite cancellation")
	}
	if len(stack) != 0 {
		t.Fatalf("scratch stack length = %d, want 0 after abort cleanup", len(stack))
	}
}

func TestTransientChildFinalizationAbortReturnsErrorTree(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_LANGS", "all")
	lang := buildArithmeticLanguage()
	parser := NewParser(lang)
	var cancelled uint32
	parser.SetCancellationFlag(&cancelled)

	tree, err := parser.ParseWithTokenSource([]byte("1+2"), &cancelAtEOFArithmeticTokenSource{cancel: &cancelled})
	if err != nil {
		t.Fatalf("ParseWithTokenSource() error = %v", err)
	}
	defer tree.Release()
	if got, want := tree.ParseStopReason(), ParseStopCancelled; got != want {
		t.Fatalf("ParseStopReason() = %q, want %q", got, want)
	}
	root := tree.RootNode()
	if root == nil {
		t.Fatal("RootNode() = nil")
	}
	if got := root.Type(lang); got != "ERROR" {
		t.Fatalf("root type = %q, want ERROR after transient child finalization abort", got)
	}
	if got := root.ChildCount(); got != 0 {
		t.Fatalf("root child count = %d, want 0 for parse error tree", got)
	}
}

func TestTransientParentScratchMaterializeEntriesUntilStopsWhenCancelled(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var childScratch transientChildScratch
	var parentScratch transientParentScratch
	var cancelled uint32 = 1
	parser := &Parser{cancellationFlag: &cancelled}
	leaf := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	children := childScratch.alloc(1)
	children[0] = leaf
	parent := parentScratch.allocParent(arena, Symbol(2), true, children, 13, true)
	entries := []stackEntry{newStackEntryNode(parent.parseState, parent)}

	if got, want := parentScratch.materializeEntriesUntil(entries, arena, &childScratch, parser), ParseStopCancelled; got != want {
		t.Fatalf("materializeEntriesUntil stop reason = %q, want %q", got, want)
	}
	if stackEntryNode(entries[0]) != parent {
		t.Fatal("entry node changed despite cancellation before traversal")
	}
	if !parentScratch.owns(parent) {
		t.Fatal("parent was materialized despite cancellation")
	}
	if len(parentScratch.seen) != 0 {
		t.Fatal("materialization scratch was not cleared after abort")
	}
}

func TestTransientParentScratchMaterializeNodeSliceUntilStopsOnExpiredTimeout(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var childScratch transientChildScratch
	var parentScratch transientParentScratch
	parser := &Parser{}
	parser.SetTimeoutMicros(100)
	endBudget := parser.beginParseOperationBudget()
	defer endBudget()
	time.Sleep(2 * time.Millisecond)

	leaf := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	children := childScratch.alloc(1)
	children[0] = leaf
	parent := parentScratch.allocParent(arena, Symbol(2), true, children, 17, true)
	nodes := []*Node{parent}

	if got, want := parentScratch.materializeNodeSliceUntil(nodes, arena, &childScratch, parser), ParseStopTimeout; got != want {
		t.Fatalf("materializeNodeSliceUntil stop reason = %q, want %q", got, want)
	}
	if nodes[0] != parent {
		t.Fatal("node slice changed despite timeout before traversal")
	}
	if !parentScratch.owns(parent) {
		t.Fatal("parent was materialized despite timeout")
	}
	if len(parentScratch.seen) != 0 {
		t.Fatal("materialization scratch was not cleared after timeout")
	}
}

func TestTransientParentScratchMaterializesSharedTransientParentOnce(t *testing.T) {
	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var childScratch transientChildScratch
	var parentScratch transientParentScratch
	leaf := newLeafNodeInArena(arena, Symbol(1), true, 0, 1, Point{}, Point{Column: 1})
	sharedChildren := childScratch.alloc(1)
	sharedChildren[0] = leaf
	shared := parentScratch.allocParent(arena, Symbol(2), true, sharedChildren, 19, true)
	leftChildren := childScratch.alloc(1)
	leftChildren[0] = shared
	rightChildren := childScratch.alloc(1)
	rightChildren[0] = shared
	left := parentScratch.allocParent(arena, Symbol(3), true, leftChildren, 23, true)
	right := parentScratch.allocParent(arena, Symbol(4), true, rightChildren, 29, true)
	rootChildren := childScratch.alloc(2)
	rootChildren[0] = left
	rootChildren[1] = right
	root := parentScratch.allocParent(arena, Symbol(5), true, rootChildren, 31, true)

	entries := []stackEntry{newStackEntryNode(root.parseState, root)}
	parentScratch.materializeEntries(entries, arena, &childScratch)

	gotRoot := stackEntryNode(entries[0])
	if gotRoot == nil || parentScratch.owns(gotRoot) {
		t.Fatal("root was not materialized out of transient storage")
	}
	gotLeft := gotRoot.children[0]
	gotRight := gotRoot.children[1]
	if gotLeft == nil || gotRight == nil {
		t.Fatal("materialized root children are nil")
	}
	gotSharedLeft := gotLeft.children[0]
	gotSharedRight := gotRight.children[0]
	if gotSharedLeft == nil || gotSharedRight == nil {
		t.Fatal("materialized shared children are nil")
	}
	if gotSharedLeft != gotSharedRight {
		t.Fatal("shared transient parent was materialized more than once")
	}
	if parentScratch.owns(gotSharedLeft) {
		t.Fatal("shared parent still points at transient storage")
	}
	if got := parentScratch.nodesMaterialized; got != 4 {
		t.Fatalf("nodesMaterialized = %d, want 4", got)
	}
}
