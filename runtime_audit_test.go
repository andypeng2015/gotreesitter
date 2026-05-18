package gotreesitter

import "testing"

func TestRuntimeAuditObserveEntriesSkipsNoTreePayload(t *testing.T) {
	leaf := &Node{symbol: 1}
	audit := &runtimeAudit{
		currentTokenGen: 1,
		nodeInfo: map[*Node]runtimeAuditNodeInfo{
			leaf: {gen: 1, kind: runtimeAuditNodeKindLeaf},
		},
		seenNode: make(map[*Node]struct{}),
	}
	entries := []stackEntry{
		newStackEntryNode(1, leaf),
		newStackEntryNoTreeNode(2, &noTreeNode{}),
	}

	var parents, leaves uint64
	audit.observeEntries(entries, &parents, &leaves)

	if leaves != 1 {
		t.Fatalf("leaf retained = %d, want 1", leaves)
	}
	if parents != 0 {
		t.Fatalf("parent retained = %d, want 0", parents)
	}
}
