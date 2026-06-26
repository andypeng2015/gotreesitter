package gotreesitter

import "unsafe"

type rawShapeRef uint32

const rawShapeRefIndexBits = 20

type rawShape struct {
	symbol       Symbol
	productionID uint16
	childRange   rawShapeChildRange
	childCount   uint16
	flags        nodeFlags
}

type rawShapeChild struct {
	entry    stackEntry
	shapeRef rawShapeRef
}

type rawShapeSlab struct {
	data []rawShape
	used int
}

type rawShapeChildSlab struct {
	data []rawShapeChild
	used int
}

type rawShapeChildRange uint64

func rawShapeBytesForCap(n int) int64 {
	if n <= 0 {
		return 0
	}
	return int64(n) * int64(unsafe.Sizeof(rawShape{}))
}

func rawShapeChildBytesForCap(n int) int64 {
	if n <= 0 {
		return 0
	}
	return int64(n) * int64(unsafe.Sizeof(rawShapeChild{}))
}

func defaultRawShapeSlabCap(class arenaClass) int {
	slabBytes := incrementalArenaSlab
	if class == arenaClassFull {
		slabBytes = fullParseArenaSlab
	}
	size := int(unsafe.Sizeof(rawShape{}))
	if size <= 0 {
		return minArenaNodeCap
	}
	capacity := slabBytes / size
	if capacity < minArenaNodeCap {
		return minArenaNodeCap
	}
	return capacity
}

func defaultRawShapeChildSlabCap(class arenaClass) int {
	slabBytes := incrementalArenaSlab
	if class == arenaClassFull {
		slabBytes = fullParseArenaSlab
	}
	size := int(unsafe.Sizeof(rawShapeChild{}))
	if size <= 0 {
		return minArenaNodeCap
	}
	capacity := slabBytes / size
	if capacity < minArenaNodeCap {
		return minArenaNodeCap
	}
	return capacity
}

func makeRawShapeChildRange(slab, start, count int) rawShapeChildRange {
	return rawShapeChildRange((uint64(slab+1) << 48) | (uint64(start) << 16) | uint64(count))
}

func (r rawShapeChildRange) count() int {
	return int(uint64(r) & 0xffff)
}

func (r rawShapeChildRange) slabIndex() int {
	return int((uint64(r)>>48)&0xffff) - 1
}

func (r rawShapeChildRange) start() int {
	return int((uint64(r) >> 16) & 0xffffffff)
}

func (a *nodeArena) rawShapeForRef(ref rawShapeRef) (*rawShape, bool) {
	if a == nil || ref == 0 {
		return nil, false
	}
	slabIdx := int(uint32(ref)>>rawShapeRefIndexBits) - 1
	entryIdx := int(uint32(ref) & ((uint32(1) << rawShapeRefIndexBits) - 1))
	if slabIdx < 0 || slabIdx >= len(a.rawShapeSlabs) {
		return nil, false
	}
	slab := &a.rawShapeSlabs[slabIdx]
	if entryIdx < 0 || entryIdx >= slab.used || entryIdx >= len(slab.data) {
		return nil, false
	}
	return &slab.data[entryIdx], true
}

func (a *nodeArena) rawShapeChildren(shape *rawShape) []rawShapeChild {
	if a == nil || shape == nil || shape.childCount == 0 || shape.childRange == 0 {
		return nil
	}
	slabIdx := shape.childRange.slabIndex()
	start := shape.childRange.start()
	count := int(shape.childCount)
	if slabIdx < 0 || slabIdx >= len(a.rawShapeChildSlabs) {
		return nil
	}
	slab := &a.rawShapeChildSlabs[slabIdx]
	if start < 0 || count < 0 || start+count > slab.used || start+count > len(slab.data) {
		return nil
	}
	return slab.data[start : start+count]
}

func (p *Parser) captureRawShape(arena *nodeArena, symbol Symbol, productionID uint16, entries []stackEntry, start, end int) rawShapeRef {
	if arena == nil || start < 0 || end < start || end > len(entries) {
		return 0
	}
	count := 0
	for i := start; i < end; i++ {
		if stackEntryHasNode(entries[i]) {
			count++
		}
	}
	if count == 0 {
		return 0
	}
	ref, shape := arena.allocRawShape()
	if shape == nil {
		return 0
	}
	shape.symbol = symbol
	shape.productionID = productionID
	if count > 0xffff {
		count = 0xffff
	}
	shape.childCount = uint16(count)
	if count == 0 {
		return ref
	}
	childRange := arena.allocRawShapeChildren(count)
	children := arena.rawShapeChildren(&rawShape{childRange: childRange, childCount: uint16(count)})
	out := 0
	for i := start; i < end && out < count; i++ {
		entry := entries[i]
		if !stackEntryHasNode(entry) {
			continue
		}
		children[out] = rawShapeChild{
			entry:    entry,
			shapeRef: stackEntryRawShapeRef(entry),
		}
		out++
	}
	shape.childRange = childRange
	return ref
}

func stackEntryRawShapeRef(entry stackEntry) rawShapeRef {
	if n := stackEntryNode(entry); n != nil {
		return n.rawShape
	}
	if n := stackEntryNoTreeNode(entry); n != nil {
		return n.rawShape
	}
	if n := stackEntryCompactFullLeaf(entry); n != nil {
		return n.rawShape
	}
	if n := stackEntryPendingParent(entry); n != nil {
		return n.rawShape
	}
	return 0
}

func setStackEntryRawShapeRef(entry *stackEntry, ref rawShapeRef) {
	if entry == nil {
		return
	}
	if n := stackEntryNode(*entry); n != nil {
		n.rawShape = ref
		nodeBumpEquivVersion(n)
		return
	}
	if n := stackEntryNoTreeNode(*entry); n != nil {
		n.rawShape = ref
		return
	}
	if n := stackEntryCompactFullLeaf(*entry); n != nil {
		n.rawShape = ref
		return
	}
	if n := stackEntryPendingParent(*entry); n != nil {
		n.rawShape = ref
	}
}

func rawShapeRefForNode(n *Node) rawShapeRef {
	if n == nil {
		return 0
	}
	return n.rawShape
}

func compareAcceptedStackRawShapePreference(p *Parser, arena *nodeArena, a, b glrStack) int {
	if !a.accepted || !b.accepted || arena == nil {
		return 0
	}
	aCount := stackMaterializingResultEntryCount(a)
	if aCount == 0 || aCount != stackMaterializingResultEntryCount(b) {
		return 0
	}
	const maxBufferedRawShapeEntries = 8
	if aCount > maxBufferedRawShapeEntries {
		return 0
	}
	var aBuf, bBuf [maxBufferedRawShapeEntries]stackEntry
	aEntries, aOK := stackMaterializingResultEntries(a, aBuf[:0], aCount)
	bEntries, bOK := stackMaterializingResultEntries(b, bBuf[:0], aCount)
	if !aOK || !bOK {
		return 0
	}
	if !rawStackEntriesContainShape(arena, aEntries) && !rawStackEntriesContainShape(arena, bEntries) {
		return 0
	}
	for i := 0; i < aCount; i++ {
		cmp := p.compareRawStackEntries(arena, aEntries[i], bEntries[i])
		if cmp != 0 {
			if cmp < 0 {
				return 1
			}
			return -1
		}
	}
	return 0
}

func rawStackEntriesContainShape(arena *nodeArena, entries []stackEntry) bool {
	for i := range entries {
		if rawStackEntryContainsShape(arena, entries[i], 0) {
			return true
		}
	}
	return false
}

func rawStackEntryContainsShape(arena *nodeArena, entry stackEntry, depth int) bool {
	if depth > maxTreeWalkDepth {
		return false
	}
	if shape, ok := rawShapeForStackEntry(arena, entry); ok {
		if shape.childCount > 0 {
			return true
		}
	}
	childCount := stackEntryNodeChildCount(entry)
	for i := 0; i < childCount; i++ {
		child, ok := rawStackEntryChildAt(arena, entry, i)
		if !ok {
			continue
		}
		if rawStackEntryContainsShape(arena, child, depth+1) {
			return true
		}
	}
	return false
}
