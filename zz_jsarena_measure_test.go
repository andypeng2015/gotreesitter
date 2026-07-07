package gotreesitter_test

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// genAsmJS produces an asm.js-style module: one giant flat function whose body
// is a long sequence of tiny shallow expression statements. This reproduces the
// "enormous FLAT tree" class (box2d.js / lua_binarytrees.js) without the actual
// corpus files: a statement_block with numStmts expression_statement children,
// each a shallow arithmetic/member/call expression.
func genAsmJS(numStmts int) []byte {
	var b strings.Builder
	b.Grow(numStmts * 24)
	b.WriteString("function asm(global, env, buffer) {\n\"use asm\";\n")
	b.WriteString("var HEAP32 = new global.Int32Array(buffer);\n")
	b.WriteString("var HEAPF64 = new global.Float64Array(buffer);\n")
	b.WriteString("var imul = global.Math.imul;\n")
	b.WriteString("function f(i1, i2) {\ni1 = i1 | 0;\ni2 = i2 | 0;\n")
	b.WriteString("var i3 = 0, i4 = 0, i5 = 0, d6 = 0.0, d7 = 0.0;\n")
	for i := 0; i < numStmts; i++ {
		switch i % 6 {
		case 0:
			b.WriteString("i3 = (i1 + i2) | 0;\n")
		case 1:
			b.WriteString("i4 = HEAP32[i3 >> 2] | 0;\n")
		case 2:
			b.WriteString("d6 = +HEAPF64[i4 >> 3];\n")
		case 3:
			b.WriteString("i5 = imul(i3, i4) | 0;\n")
		case 4:
			b.WriteString("d7 = d6 * 2.0 + 1.0;\n")
		case 5:
			b.WriteString("HEAP32[i5 >> 2] = i4 + i3 | 0;\n")
		}
	}
	b.WriteString("return i4 | 0;\n}\nreturn { f: f };\n}\n")
	return []byte(b.String())
}

func TestZZJsArenaMeasure(t *testing.T) {
	if os.Getenv("GTS_JS_ARENA_MEASURE") == "" {
		t.Skip("set GTS_JS_ARENA_MEASURE=1 to run arena measurement")
	}
	stmts := 200000
	if v := os.Getenv("GTS_JS_ARENA_STMTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			stmts = n
		}
	}
	src := genAsmJS(stmts)
	t.Logf("asm.js source: %d statements, %d bytes (%.2f MB)", stmts, len(src), float64(len(src))/(1<<20))

	gotreesitter.EnableArenaBreakdown(true)
	defer gotreesitter.EnableArenaBreakdown(false)

	parser := gotreesitter.NewParser(grammars.JavascriptLanguage())
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer tree.Release()

	rt := tree.ParseRuntime()
	t.Logf("StopReason=%q stoppedEarly=%v", tree.ParseStopReason(), tree.ParseStoppedEarly())
	t.Logf("LastTokenEndByte=%d of %d (%.1f%% through source)",
		rt.LastTokenEndByte, len(src), 100*float64(rt.LastTokenEndByte)/float64(len(src)))
	t.Logf("MemoryBudgetBytes=%d (%.0f MB)", rt.MemoryBudgetBytes, float64(rt.MemoryBudgetBytes)/(1<<20))
	t.Logf("ArenaBytesAllocated=%d (%.1f MB)", rt.ArenaBytesAllocated, float64(rt.ArenaBytesAllocated)/(1<<20))
	t.Logf("ScratchBytesAllocated=%d (%.1f MB)", rt.ScratchBytesAllocated, float64(rt.ScratchBytesAllocated)/(1<<20))
	t.Logf("GSSBytesAllocated=%d (%.1f MB)", rt.GSSBytesAllocated, float64(rt.GSSBytesAllocated)/(1<<20))
	t.Logf("FinalNodes=%d (leaf=%d parent=%d)", rt.FinalNodes, rt.FinalLeafNodes, rt.FinalParentNodes)
	t.Logf("LeafNodesConstructed=%d ParentNodesConstructed=%d", rt.LeafNodesConstructed, rt.ParentNodesConstructed)
	t.Logf("MaxStacksSeen=%d PeakStackDepth=%d", rt.MaxStacksSeen, rt.PeakStackDepth)
	if rt.FinalNodes > 0 {
		t.Logf("PER-FINAL-NODE arena bytes = %.1f", float64(rt.ArenaBytesAllocated)/float64(rt.FinalNodes))
	}
	if rt.LeafNodesConstructed+rt.ParentNodesConstructed > 0 {
		t.Logf("PER-CONSTRUCTED-NODE arena bytes = %.1f",
			float64(rt.ArenaBytesAllocated)/float64(rt.LeafNodesConstructed+rt.ParentNodesConstructed))
	}

	bd, ok := tree.ArenaBreakdown()
	if !ok {
		t.Fatal("ArenaBreakdown unavailable")
	}
	type region struct {
		name  string
		bytes int64
	}
	regions := []region{
		{"NodeStruct", bd.NodeStructBytesAllocated},
		{"NoTreeNode", bd.NoTreeNodeBytesAllocated},
		{"CompactFullLeaf", bd.CompactFullLeafBytesAllocated},
		{"CompactCheckpointLeaf", bd.CompactCheckpointLeafBytesAllocated},
		{"PendingParent", bd.PendingParentBytesAllocated},
		{"PendingChildEntry", bd.PendingChildEntryBytesAllocated},
		{"RawShape", bd.RawShapeBytesAllocated},
		{"RawShapeChild", bd.RawShapeChildBytesAllocated},
		{"FinalChildSidecar", bd.FinalChildSidecarBytesAllocated},
		{"ChildSlice", bd.ChildSliceBytesAllocated},
		{"FieldID", bd.FieldIDBytesAllocated},
		{"FieldSource", bd.FieldSourceBytesAllocated},
	}
	sort.Slice(regions, func(i, j int) bool { return regions[i].bytes > regions[j].bytes })
	t.Logf("ARENA BREAKDOWN (top consumers):")
	for _, r := range regions {
		if r.bytes == 0 {
			continue
		}
		pct := 100 * float64(r.bytes) / float64(rt.ArenaBytesAllocated)
		t.Logf("  %-22s %12d  (%.1f MB, %5.1f%%)", r.name, r.bytes, float64(r.bytes)/(1<<20), pct)
	}
	t.Logf("NodeLiveCount=%d NodeCapacityCount=%d NodeCapacityWaste=%d",
		bd.NodeLiveCount, bd.NodeCapacityCount, bd.NodeCapacityWaste)
	if bd.NodeLiveCount > 0 {
		t.Logf("NodeStruct bytes / live node = %.1f", float64(bd.NodeStructBytesAllocated)/float64(bd.NodeLiveCount))
	}
	fmt.Fprintf(os.Stderr, "MEASURE_DONE\n")
}

func BenchmarkZZJsArenaParse(b *testing.B) {
	if os.Getenv("GTS_JS_ARENA_MEASURE") == "" {
		b.Skip("set GTS_JS_ARENA_MEASURE=1")
	}
	src := genAsmJS(20000)
	parser := gotreesitter.NewParser(grammars.JavascriptLanguage())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree, err := parser.Parse(src)
		if err != nil {
			b.Fatal(err)
		}
		tree.Release()
	}
}
