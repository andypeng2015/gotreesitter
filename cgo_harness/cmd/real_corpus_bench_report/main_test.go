package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestParseBenchmarkLine(t *testing.T) {
	line := "BenchmarkParityRealCorpusParseIncrementalSingleByteEdit/rust/tree-sitter-c-16 123 225000 ns/op 2048 B/op 3 allocs/op 2 files/op 100000 parse_wall_ns/op"
	s, ok, err := parseBenchmarkLine(line)
	if err != nil {
		t.Fatalf("parseBenchmarkLine error: %v", err)
	}
	if !ok {
		t.Fatal("parseBenchmarkLine did not match")
	}
	if s.Suite != "IncrementalSingleByteEdit" || s.Language != "rust" || s.Backend != "tree-sitter-c" {
		t.Fatalf("unexpected identity: %#v", s)
	}
	if s.Iterations != 123 {
		t.Fatalf("iterations=%d", s.Iterations)
	}
	if got := s.Metrics["ns/op"]; got != 225000 {
		t.Fatalf("ns/op=%v", got)
	}
	if got := s.Metrics["parse_wall_ns/op"]; got != 100000 {
		t.Fatalf("parse_wall_ns/op=%v", got)
	}
}

func TestBuildReportRanksWorstRatio(t *testing.T) {
	samples := []sample{
		{Suite: "Full", Language: "go", Backend: "gotreesitter", Metrics: map[string]float64{"ns/op": 200}},
		{Suite: "Full", Language: "go", Backend: "tree-sitter-c", Metrics: map[string]float64{"ns/op": 100}},
		{Suite: "IncrementalSingleByteEdit", Language: "go", Backend: "gotreesitter", Metrics: map[string]float64{"ns/op": 300, "parse_wall_ns/op": 250}},
		{Suite: "IncrementalSingleByteEdit", Language: "go", Backend: "tree-sitter-c", Metrics: map[string]float64{"ns/op": 100}},
		{Suite: "Full", Language: "rust", Backend: "gotreesitter", Metrics: map[string]float64{"ns/op": 600, "result_tree_build_ns/op": 400}},
		{Suite: "Full", Language: "rust", Backend: "tree-sitter-c", Metrics: map[string]float64{"ns/op": 100}},
	}
	r := buildReport([]string{"bench.txt"}, samples, nil)
	if len(r.Languages) != 2 {
		t.Fatalf("languages=%d", len(r.Languages))
	}
	if r.Languages[0].Language != "rust" {
		t.Fatalf("first language=%s", r.Languages[0].Language)
	}
	if got := r.Languages[0].WorstRatio; got != 6 {
		t.Fatalf("rust worst ratio=%v", got)
	}
	if len(r.Languages[0].TopAttribution) == 0 || r.Languages[0].TopAttribution[0].Name != "result_tree_build_ns/op" {
		t.Fatalf("unexpected attribution: %#v", r.Languages[0].TopAttribution)
	}
}

func TestBuildReportMarksAllowMismatchQuality(t *testing.T) {
	allowMismatch := true
	samples := []sample{
		{Suite: "Full", Language: "go", Backend: "gotreesitter", Metrics: map[string]float64{"ns/op": 200}},
		{Suite: "Full", Language: "go", Backend: "tree-sitter-c", Metrics: map[string]float64{"ns/op": 100}},
	}
	r := buildReport([]string{"bench.txt"}, samples, []runResource{
		{Language: "go", AllowMismatch: &allowMismatch},
	})
	if len(r.Languages) != 1 {
		t.Fatalf("languages=%d", len(r.Languages))
	}
	if !hasString(r.Languages[0].QualityNotes, "parity precheck disabled") {
		t.Fatalf("quality notes=%v", r.Languages[0].QualityNotes)
	}
}

func TestSummarizeValuesIncludesSpread(t *testing.T) {
	s := summarizeValues([]float64{90, 100, 110})
	if s.Samples != 3 {
		t.Fatalf("samples=%d", s.Samples)
	}
	if s.Median != 100 || s.Mean != 100 || s.Min != 90 || s.Max != 110 {
		t.Fatalf("unexpected summary: %#v", s)
	}
	if !near(s.StdDev, 10) {
		t.Fatalf("stddev=%v", s.StdDev)
	}
	if !near(s.RelStdDev, 0.1) {
		t.Fatalf("rel stddev=%v", s.RelStdDev)
	}
	if !near(s.MedianAbsDev, 10) {
		t.Fatalf("median abs dev=%v", s.MedianAbsDev)
	}
	if !near(s.RelMedianAbsDev, 0.1) {
		t.Fatalf("rel median abs dev=%v", s.RelMedianAbsDev)
	}
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func near(got, want float64) bool {
	return math.Abs(got-want) < 1e-9
}

func TestParseRunResources(t *testing.T) {
	root := t.TempDir()
	runDir := filepath.Join(root, "20260605T030054Z-real-corpus-bench-toml")
	if err := os.Mkdir(runDir, 0o755); err != nil {
		t.Fatal(err)
	}
	containerLog := filepath.Join(runDir, "container.log")
	if err := os.WriteFile(containerLog, []byte(`
Maximum resident set size (kbytes): 1156904
`), 0o644); err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(runDir, "metadata.txt")
	if err := os.WriteFile(metadata, []byte(`
	memory = 4g
	gomemlimit = 3GiB
	exit_code = 0
	oom_killed = false
	command=env GOMAXPROCS=1 GTS_REAL_CORPUS_BENCH_ALLOW_MISMATCH=1 GTS_REAL_CORPUS_BENCH_SKIP_MISMATCH=0 go test .
	`), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := expandInputs([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	resources, err := parseRunResources(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Fatalf("resources=%d", len(resources))
	}
	r := resources[0]
	if r.Language != "toml" {
		t.Fatalf("language=%q", r.Language)
	}
	if r.MaxRSSKB != 1156904 {
		t.Fatalf("max rss=%d", r.MaxRSSKB)
	}
	if r.OOMKilled == nil || *r.OOMKilled {
		t.Fatalf("oom killed=%v", r.OOMKilled)
	}
	if r.ExitCode == nil || *r.ExitCode != 0 {
		t.Fatalf("exit code=%v", r.ExitCode)
	}
	if r.Memory != "4g" || r.GOMEMLimit != "3GiB" {
		t.Fatalf("limits=%q/%q", r.Memory, r.GOMEMLimit)
	}
	if r.AllowMismatch == nil || !*r.AllowMismatch {
		t.Fatalf("allow mismatch=%v", r.AllowMismatch)
	}
	if r.SkipMismatch == nil || *r.SkipMismatch {
		t.Fatalf("skip mismatch=%v", r.SkipMismatch)
	}
}

func TestParseBenchmarkFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bench.log")
	err := os.WriteFile(path, []byte(`
BenchmarkParityRealCorpusParseFull/python/gotreesitter-16 10 1000 ns/op 1 files/op
BenchmarkParityRealCorpusParseFull/python/tree-sitter-c-16 10 500 ns/op 1 files/op
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	files, err := expandInputs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	samples, err := parseBenchmarkFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("samples=%d", len(samples))
	}
}
