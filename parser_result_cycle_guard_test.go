package gotreesitter

import "testing"

func TestStripResultTreeSelfCycles(t *testing.T) {
	x := &Node{}
	x.children = []*Node{x}
	stripResultTreeSelfCycles(x)
	for _, c := range x.children {
		if c == x {
			t.Fatal("self-reference not removed")
		}
	}

	a := &Node{}
	b := &Node{}
	a.children = []*Node{b}
	b.children = []*Node{a}
	stripResultTreeSelfCycles(a)
	for _, c := range b.children {
		if c == a {
			t.Fatal("ancestor back-edge not removed")
		}
	}

	leaf := &Node{}
	mid := &Node{children: []*Node{leaf}}
	root := &Node{children: []*Node{mid}}
	stripResultTreeSelfCycles(root)
	if len(root.children) != 1 || root.children[0] != mid || len(mid.children) != 1 || mid.children[0] != leaf {
		t.Fatal("clean tree was mutated")
	}

	y := &Node{}
	kept := &Node{}
	y.children = []*Node{y, kept}
	y.fieldIDs = []FieldID{1, 2}
	y.fieldSources = []uint8{fieldSourceDirect, fieldSourceInherited}
	stripResultTreeSelfCycles(y)
	if len(y.children) != 1 || y.children[0] != kept {
		t.Fatalf("children after cycle strip = %v, want only kept child", y.children)
	}
	if len(y.fieldIDs) != 1 || y.fieldIDs[0] != 2 {
		t.Fatalf("fieldIDs after cycle strip = %v, want [2]", y.fieldIDs)
	}
	if len(y.fieldSources) != 1 || y.fieldSources[0] != fieldSourceInherited {
		t.Fatalf("fieldSources after cycle strip = %v, want inherited source", y.fieldSources)
	}
}

func TestFlattenInvisibleRootChildrenHandlesCycles(t *testing.T) {
	meta := []SymbolMetadata{
		{Name: "root", Visible: true},
		{Name: "_hidden", Visible: false},
		{Name: "visible", Visible: true},
	}

	self := &Node{symbol: 1}
	self.children = []*Node{self}
	root := &Node{symbol: 0, children: []*Node{self}}
	flattenInvisibleRootChildren(root, nil, &Language{SymbolMetadata: meta})
	if len(root.children) != 0 {
		t.Fatalf("self-cycle flatten children = %d, want 0", len(root.children))
	}

	visible := &Node{symbol: 2}
	outer := &Node{symbol: 1}
	inner := &Node{symbol: 1}
	outer.children = []*Node{inner}
	inner.children = []*Node{outer, visible}
	root = &Node{symbol: 0, children: []*Node{outer}}
	flattenInvisibleRootChildren(root, nil, &Language{SymbolMetadata: meta})
	if len(root.children) != 1 || root.children[0] != visible {
		t.Fatalf("ancestor-cycle flatten children = %v, want visible leaf", root.children)
	}
}

func TestFlattenInvisibleRootChildrenStillFlattensCleanHiddenWrappers(t *testing.T) {
	meta := []SymbolMetadata{
		{Name: "root", Visible: true},
		{Name: "_hidden", Visible: false},
		{Name: "visible", Visible: true},
	}
	visible := &Node{symbol: 2}
	hidden := &Node{symbol: 1, children: []*Node{visible}}
	root := &Node{symbol: 0, children: []*Node{hidden}}
	flattenInvisibleRootChildren(root, nil, &Language{SymbolMetadata: meta})
	if len(root.children) != 1 || root.children[0] != visible {
		t.Fatalf("clean flatten children = %v, want visible leaf", root.children)
	}
}
