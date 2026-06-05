package gotreesitter

import "testing"

func TestNormalizeGraphQLCompatibilityRestoresCollapsedNamedLeafChildren(t *testing.T) {
	lang := &Language{
		Name: "graphql",
		SymbolNames: []string{
			"EOF", "source_file", "document", "operation_type", "boolean_value",
			"query", "mutation", "subscription", "true", "false",
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "EOF", Visible: false, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "document", Visible: true, Named: true},
			{Name: "operation_type", Visible: true, Named: true},
			{Name: "boolean_value", Visible: true, Named: true},
			{Name: "query", Visible: true, Named: false},
			{Name: "mutation", Visible: true, Named: false},
			{Name: "subscription", Visible: true, Named: false},
			{Name: "true", Visible: true, Named: false},
			{Name: "false", Visible: true, Named: false},
		},
	}

	source := []byte("querymutationtruefalse")
	arena := newNodeArena(arenaClassFull)
	cases := []struct {
		name       string
		sym        Symbol
		start, end uint32
		want       string
	}{
		{name: "query", sym: 3, start: 0, end: 5, want: "query"},
		{name: "mutation", sym: 3, start: 5, end: 13, want: "mutation"},
		{name: "true", sym: 4, start: 13, end: 17, want: "true"},
		{name: "false", sym: 4, start: 17, end: 22, want: "false"},
	}
	nodes := make([]*Node, 0, len(cases))
	for _, tt := range cases {
		nodes = append(nodes, newLeafNodeInArena(arena, tt.sym, true, tt.start, tt.end, Point{Column: tt.start}, Point{Column: tt.end}))
	}
	document := newParentNodeInArena(arena, 2, true, nodes, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{document}, nil, 0)

	normalizeGraphQLCompatibility(root, source, lang)

	for i, tt := range cases {
		node := nodes[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := node.ChildCount(); got != 1 {
				t.Fatalf("child count = %d, want 1", got)
			}
			child := node.Child(0)
			if child == nil {
				t.Fatal("child = nil")
			}
			if got := child.Type(lang); got != tt.want {
				t.Fatalf("child type = %q, want %q", got, tt.want)
			}
			if child.IsNamed() {
				t.Fatal("child is named, want anonymous")
			}
			if got, want := child.StartByte(), tt.start; got != want {
				t.Fatalf("child start byte = %d, want %d", got, want)
			}
			if got, want := child.EndByte(), tt.end; got != want {
				t.Fatalf("child end byte = %d, want %d", got, want)
			}
			if child.parent != node {
				t.Fatal("child parent was not restored")
			}
		})
	}
}
