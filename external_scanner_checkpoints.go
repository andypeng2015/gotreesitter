package gotreesitter

import (
	"bytes"
	"sort"
	"unsafe"
)

type externalScannerCheckpoint struct {
	start []byte
	end   []byte
}

type externalScannerSnapshotRef struct {
	off  uint32
	slab uint16
	len  uint16
}

type externalScannerCheckpointRef struct {
	start externalScannerSnapshotRef
	end   externalScannerSnapshotRef
}

type externalScannerCheckpointSet struct {
	indexes []uint32
	refs    []externalScannerCheckpointRef
}

func languageUsesExternalScannerCheckpoints(lang *Language) bool {
	if lang == nil || lang.ExternalScanner == nil {
		return false
	}
	switch lang.Name {
	case "python", "mojo", "starlark":
		return true
	default:
		return false
	}
}

func underlyingDFATokenSource(ts TokenSource) *dfaTokenSource {
	switch src := ts.(type) {
	case *dfaTokenSource:
		return src
	case *includedRangeTokenSource:
		return underlyingDFATokenSource(src.base)
	default:
		return nil
	}
}

func (a *nodeArena) recordExternalScannerLeafCheckpoint(node *Node, start, end []byte) bool {
	if a == nil || node == nil {
		return false
	}
	startRef := a.copyExternalScannerSnapshotRef(start)
	endRef := startRef
	if !bytes.Equal(start, end) {
		endRef = a.copyExternalScannerSnapshotRef(end)
	}
	ok := a.setExternalScannerCheckpoint(node, externalScannerCheckpointRef{
		start: startRef,
		end:   endRef,
	})
	if ok {
		a.externalScannerCheckpointRecords++
	}
	return ok
}

func (a *nodeArena) recordExternalScannerCompactCheckpoint(start, end []byte) externalScannerCheckpointRef {
	if a == nil {
		return externalScannerCheckpointRef{}
	}
	startRef := a.copyExternalScannerSnapshotRef(start)
	endRef := startRef
	if !bytes.Equal(start, end) {
		endRef = a.copyExternalScannerSnapshotRef(end)
	}
	a.externalScannerCheckpointRecords++
	return externalScannerCheckpointRef{
		start: startRef,
		end:   endRef,
	}
}

func (a *nodeArena) copyExternalScannerSnapshotRef(src []byte) externalScannerSnapshotRef {
	if a == nil || len(src) == 0 {
		return externalScannerSnapshotRef{}
	}
	if bytes.Equal(src, a.externalScannerSnapshotBytes(a.externalScannerLastSnapshotRef)) {
		return a.externalScannerLastSnapshotRef
	}
	ref := a.allocExternalScannerSnapshotRef(src)
	a.externalScannerLastSnapshotRef = ref
	return ref
}

func (a *nodeArena) setExternalScannerCheckpoint(node *Node, cp externalScannerCheckpointRef) bool {
	if a == nil || node == nil {
		return false
	}
	set, idx, ok := a.externalScannerCheckpointSetForNode(node, true)
	if !ok {
		return false
	}
	a.allocatedBytes += set.upsert(idx, cp)
	return true
}

func externalScannerCheckpointForNode(node *Node) (externalScannerCheckpoint, bool) {
	cp, ok := externalScannerCheckpointRefForNode(node)
	if !ok || node == nil || node.ownerArena == nil {
		return externalScannerCheckpoint{}, false
	}
	return externalScannerCheckpoint{
		start: node.ownerArena.externalScannerSnapshotBytes(cp.start),
		end:   node.ownerArena.externalScannerSnapshotBytes(cp.end),
	}, true
}

func externalScannerCheckpointRefForNode(node *Node) (externalScannerCheckpointRef, bool) {
	if node == nil || node.ownerArena == nil {
		return externalScannerCheckpointRef{}, false
	}
	set, idx, ok := node.ownerArena.externalScannerCheckpointSetForNode(node, false)
	if !ok {
		return externalScannerCheckpointRef{}, false
	}
	cp, ok := set.lookup(idx)
	if !ok || (cp.start.len == 0 && cp.end.len == 0) {
		return externalScannerCheckpointRef{}, false
	}
	return cp, true
}

func rebuildExternalScannerCheckpoints(root *Node, lang *Language) {
	if root == nil || !languageUsesExternalScannerCheckpoints(lang) {
		return
	}

	type frame struct {
		node    *Node
		visited bool
	}

	stack := []frame{{node: root}}
	for len(stack) > 0 {
		last := len(stack) - 1
		f := stack[last]
		stack = stack[:last]
		n := f.node
		if n == nil {
			continue
		}
		if !f.visited {
			stack = append(stack, frame{node: n, visited: true})
			for i := len(n.children) - 1; i >= 0; i-- {
				stack = append(stack, frame{node: n.children[i]})
			}
			continue
		}
		if len(n.children) == 0 {
			continue
		}

		var start []byte
		var end []byte
		var startRef externalScannerSnapshotRef
		var endRef externalScannerSnapshotRef
		for _, child := range n.children {
			cp, ok := externalScannerCheckpointRefForNode(child)
			if !ok {
				continue
			}
			startRef = cp.start
			start = n.ownerArena.externalScannerSnapshotBytes(cp.start)
			break
		}
		for i := len(n.children) - 1; i >= 0; i-- {
			cp, ok := externalScannerCheckpointRefForNode(n.children[i])
			if !ok {
				continue
			}
			endRef = cp.end
			end = n.ownerArena.externalScannerSnapshotBytes(cp.end)
			break
		}
		if start == nil && end == nil {
			continue
		}
		n.ownerArena.setExternalScannerCheckpoint(n, externalScannerCheckpointRef{start: startRef, end: endRef})
	}
}

func currentExternalScannerCheckpoint(ts TokenSource) (externalScannerCheckpoint, uint32, uint32, bool) {
	dts := underlyingDFATokenSource(ts)
	if dts == nil || !languageUsesExternalScannerCheckpoints(dts.language) {
		return externalScannerCheckpoint{}, 0, 0, false
	}
	return dts.lastExternalScannerCheckpoint()
}

func canReuseNodeWithExternalScannerCheckpoint(ts TokenSource, startState StateID, node *Node) (externalScannerCheckpointRef, bool) {
	dts := underlyingDFATokenSource(ts)
	if dts == nil || !languageUsesExternalScannerCheckpoints(dts.language) {
		return externalScannerCheckpointRef{}, true
	}
	if node == nil || startState != node.PreGotoState() {
		return externalScannerCheckpointRef{}, false
	}
	cp, ok := externalScannerCheckpointRefForNode(node)
	if !ok {
		return externalScannerCheckpointRef{}, false
	}
	if !dts.externalScannerStateMatches(node.ownerArena.externalScannerSnapshotBytes(cp.start)) {
		return externalScannerCheckpointRef{}, false
	}
	return cp, true
}

func fastForwardWithExternalScannerCheckpoint(ts TokenSource, node *Node, cp externalScannerCheckpointRef) (Token, bool) {
	dts := underlyingDFATokenSource(ts)
	if dts == nil || !languageUsesExternalScannerCheckpoints(dts.language) {
		return Token{}, false
	}
	if node == nil {
		return Token{}, false
	}
	dts.restoreExternalScannerState(node.ownerArena.externalScannerSnapshotBytes(cp.end))
	if skipper, ok := ts.(PointSkippableTokenSource); ok {
		return skipper.SkipToByteWithPoint(node.EndByte(), node.EndPoint()), true
	}
	if skipper, ok := ts.(ByteSkippableTokenSource); ok {
		return skipper.SkipToByte(node.EndByte()), true
	}
	return advanceTokenSourceTo(ts, Token{
		StartByte:  node.StartByte(),
		EndByte:    node.StartByte(),
		StartPoint: node.StartPoint(),
		EndPoint:   node.StartPoint(),
	}, node.EndByte()), true
}

func (a *nodeArena) externalScannerCheckpointSetForNode(node *Node, create bool) (*externalScannerCheckpointSet, int, bool) {
	if a == nil || node == nil {
		return nil, 0, false
	}
	if idx, ok := nodeIndexInStorage(node, a.nodes); ok {
		return &a.externalScannerNodeCheckpoints, idx, true
	}
	for i := range a.nodeSlabs {
		idx, ok := nodeIndexInStorage(node, a.nodeSlabs[i].data)
		if !ok {
			continue
		}
		if create {
			for len(a.externalScannerNodeCheckpointSlabs) <= i {
				a.externalScannerNodeCheckpointSlabs = append(a.externalScannerNodeCheckpointSlabs, externalScannerCheckpointSlab{})
			}
		} else if i >= len(a.externalScannerNodeCheckpointSlabs) {
			return nil, 0, false
		}
		return &a.externalScannerNodeCheckpointSlabs[i].checkpoints, idx, true
	}
	return nil, 0, false
}

func (s *externalScannerCheckpointSet) lookup(idx int) (externalScannerCheckpointRef, bool) {
	if s == nil || len(s.indexes) == 0 || idx < 0 {
		return externalScannerCheckpointRef{}, false
	}
	key := uint32(idx)
	pos := sort.Search(len(s.indexes), func(i int) bool {
		return s.indexes[i] >= key
	})
	if pos >= len(s.indexes) || s.indexes[pos] != key {
		return externalScannerCheckpointRef{}, false
	}
	return s.refs[pos], true
}

func (s *externalScannerCheckpointSet) upsert(idx int, cp externalScannerCheckpointRef) int64 {
	if s == nil || idx < 0 {
		return 0
	}
	key := uint32(idx)
	n := len(s.indexes)
	if n == 0 || s.indexes[n-1] < key {
		beforeIndexCap := cap(s.indexes)
		beforeRefCap := cap(s.refs)
		s.indexes = append(s.indexes, key)
		s.refs = append(s.refs, cp)
		return externalScannerCheckpointIndexBytesForCap(cap(s.indexes)-beforeIndexCap) +
			externalScannerCheckpointBytesForCap(cap(s.refs)-beforeRefCap)
	}
	before := s.bytesAllocated()
	pos := sort.Search(n, func(i int) bool {
		return s.indexes[i] >= key
	})
	if pos < n && s.indexes[pos] == key {
		s.refs[pos] = cp
		return 0
	}
	s.indexes = append(s.indexes, 0)
	copy(s.indexes[pos+1:], s.indexes[pos:])
	s.indexes[pos] = key
	s.refs = append(s.refs, externalScannerCheckpointRef{})
	copy(s.refs[pos+1:], s.refs[pos:])
	s.refs[pos] = cp
	return s.bytesAllocated() - before
}

func (s *externalScannerCheckpointSet) ensureCapacity(min int) int64 {
	if s == nil || min <= 0 || (cap(s.indexes) >= min && cap(s.refs) >= min) {
		return 0
	}
	before := s.bytesAllocated()
	if cap(s.indexes) < min {
		indexes := make([]uint32, len(s.indexes), min)
		copy(indexes, s.indexes)
		s.indexes = indexes
	}
	if cap(s.refs) < min {
		refs := make([]externalScannerCheckpointRef, len(s.refs), min)
		copy(refs, s.refs)
		s.refs = refs
	}
	return s.bytesAllocated() - before
}

func (s *externalScannerCheckpointSet) reset() {
	if s == nil {
		return
	}
	clear(s.refs)
	s.indexes = s.indexes[:0]
	s.refs = s.refs[:0]
}

func (s externalScannerCheckpointSet) bytesAllocated() int64 {
	return externalScannerCheckpointIndexBytesForCap(cap(s.indexes)) + externalScannerCheckpointBytesForCap(cap(s.refs))
}

func (s externalScannerCheckpointSet) slotsAllocated() uint64 {
	return uint64(cap(s.refs))
}

func nodeIndexInStorage(node *Node, storage []Node) (int, bool) {
	if node == nil || len(storage) == 0 {
		return 0, false
	}
	start := uintptr(unsafe.Pointer(&storage[0]))
	ptr := uintptr(unsafe.Pointer(node))
	size := unsafe.Sizeof(Node{})
	end := start + uintptr(len(storage))*size
	if ptr < start || ptr >= end {
		return 0, false
	}
	offset := ptr - start
	if offset%size != 0 {
		return 0, false
	}
	return int(offset / size), true
}
