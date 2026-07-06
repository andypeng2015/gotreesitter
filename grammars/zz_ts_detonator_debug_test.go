package grammars

// Intra-token population explosion debug harness (gated; not CI).
// GTS_TS_DET=1 go test ./grammars -run TestTSDetonatorSeries -v
// Parses the union-type detonator files at several sizes and reports
// stop reason, max stacks, node counts.

import (
	"fmt"
	"os"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
)

func TestTSDetonatorSeries(t *testing.T) {
	if os.Getenv("GTS_TS_DET") != "1" {
		t.Skip("set GTS_TS_DET=1")
	}
	type target struct {
		lang string
		path string
	}
	targets := []target{
		{"typescript", "/tmp/det800.d.ts"},
		{"typescript", "/tmp/det1000.d.ts"},
		{"typescript", "/tmp/det1300.d.ts"},
		{"typescript", "/tmp/det2000.d.ts"},
	}
	if only := os.Getenv("GTS_TS_DET_ONLY"); only != "" {
		lang := os.Getenv("GTS_TS_DET_LANG")
		if lang == "" {
			lang = "typescript"
		}
		targets = []target{{lang, only}}
	}
	for _, tg := range targets {
		path := tg.path
		src, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("DET %s READ-ERR %v\n", path, err)
			continue
		}
		var lang *gotreesitter.Language
		switch tg.lang {
		case "typescript":
			lang = TypescriptLanguage()
		case "rust":
			lang = RustLanguage()
		case "python":
			lang = PythonLanguage()
		}
		p := gotreesitter.NewParser(lang)
		p.SetTimeoutMicros(120_000_000)
		start := time.Now()
		tree, err := p.Parse(src)
		elapsed := time.Since(start)
		if err != nil || tree == nil {
			fmt.Printf("DET %s bytes=%d PARSE-ERR %v took=%v\n", path, len(src), err, elapsed)
			continue
		}
		root := tree.RootNode()
		rt := tree.ParseRuntime()
		fmt.Printf("DET %s bytes=%-7d took=%-12v hasErr=%-5v stop=%-14v maxStacks=%-6d nodes=%-9d nodeLimit=%-9d iter=%-9d lastTokEnd=%-7d rootEnd=%-7d truncated=%v\n",
			path, len(src), elapsed.Round(time.Millisecond), root.HasError(), rt.StopReason, rt.MaxStacksSeen,
			rt.NodesAllocated, rt.NodeLimit, rt.Iterations, rt.LastTokenEndByte, rt.RootEndByte, rt.Truncated)
	}
}
